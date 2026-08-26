package gormrepo

import (
	"context"
	"errors"
	"time"

	"my-bbs/internal/model"
	"my-bbs/internal/repository"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PostRepository struct {
	db *gorm.DB
}

// postListColumns deliberately excludes content. List views only need the
// metadata below; the full body is loaded exclusively by FindPostByID.
var postListColumns = []string{"id", "create_time", "update_time", "user_id", "title", "visible", "pinned_until"}

func NewPostRepository(db *gorm.DB) *PostRepository {
	return &PostRepository{db: db}
}

func (r *PostRepository) CreatePost(ctx context.Context, post *model.Post) error {
	return translateError(r.db.WithContext(ctx).Create(post).Error)
}

func (r *PostRepository) FindPostByID(ctx context.Context, id uint) (*model.Post, error) {
	var post model.Post
	result := r.db.WithContext(ctx).First(&post, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, translateError(result.Error)
	}
	return &post, nil
}

func (r *PostRepository) FindPublicPosts(ctx context.Context, offset, limit int) ([]model.Post, error) {
	var posts []model.Post
	err := r.db.WithContext(ctx).
		Select(postListColumns).
		Where("visible = ?", model.VisiblePublic).
		Clauses(clause.OrderBy{Expression: clause.Expr{
			SQL:                "CASE WHEN pinned_until > ? THEN 0 ELSE 1 END ASC, create_time DESC, id DESC",
			Vars:               []any{time.Now()},
			WithoutParentheses: true,
		}}).
		Offset(offset).
		Limit(limit).
		Find(&posts).Error
	return posts, translateError(err)
}

func (r *PostRepository) FindPostsByUserID(ctx context.Context, userID uint, offset, limit int) ([]model.Post, error) {
	var posts []model.Post
	err := r.db.WithContext(ctx).
		Select(postListColumns).
		Where("user_id = ?", userID).
		Order("create_time DESC").
		Order("id DESC").
		Offset(offset).
		Limit(limit).
		Find(&posts).Error
	return posts, translateError(err)
}

func (r *PostRepository) FindPublicPostsByUserID(ctx context.Context, userID uint, offset, limit int) ([]model.Post, error) {
	var posts []model.Post
	err := r.db.WithContext(ctx).
		Select(postListColumns).
		Where("user_id = ? AND visible = ?", userID, model.VisiblePublic).
		Order("create_time DESC").
		Order("id DESC").
		Offset(offset).
		Limit(limit).
		Find(&posts).Error
	return posts, translateError(err)
}

func (r *PostRepository) UpdatePost(ctx context.Context, post *model.Post) error {
	// 普通编辑只能写标题和正文。若与可见性或置顶操作并发，不能把
	// 此前读到的旧 visible/pinned_until 值写回数据库。
	result := r.db.WithContext(ctx).Model(&model.Post{}).
		Where("id = ?", post.ID).
		Select("title", "content").
		Updates(post)
	return r.resolvePostUpdateResult(ctx, post.ID, result)
}

func (r *PostRepository) UpdatePostVisible(ctx context.Context, id uint, visible uint8) error {
	updates := map[string]any{"visible": visible}
	if visible == model.VisiblePrivate {
		updates["pinned_until"] = nil
	}
	result := r.db.WithContext(ctx).Model(&model.Post{}).Where("id = ?", id).
		Updates(updates)
	return r.resolvePostUpdateResult(ctx, id, result)
}

func (r *PostRepository) SetPostPinnedUntil(ctx context.Context, id uint, pinnedUntil *time.Time) error {
	if pinnedUntil == nil {
		result := r.db.WithContext(ctx).Model(&model.Post{}).Where("id = ?", id).
			Update("pinned_until", nil)
		return r.resolvePostUpdateResult(ctx, id, result)
	}
	normalizedPinnedUntil := pinnedUntil.Truncate(time.Millisecond)
	pinnedUntil = &normalizedPinnedUntil

	// 置顶与可见性变更可能并发，在 SQL 层继续保证私密帖不会被置顶。
	// MySQL 在新旧值相同时也可能返回 RowsAffected=0，因此回读状态
	// 区分“值已经相同”与“并发改为私密”，必要时再尝试一次。
	for attempt := 0; attempt < 2; attempt++ {
		result := r.db.WithContext(ctx).Model(&model.Post{}).
			Where("id = ? AND visible = ?", id, model.VisiblePublic).
			Update("pinned_until", pinnedUntil)
		if result.Error != nil {
			return translateError(result.Error)
		}
		if result.RowsAffected > 0 {
			return nil
		}

		var post model.Post
		if err := r.db.WithContext(ctx).Select("visible", "pinned_until").First(&post, id).Error; err != nil {
			return translateError(err)
		}
		if post.Visible != model.VisiblePublic {
			return repository.ErrNotFound
		}
		if post.PinnedUntil != nil && post.PinnedUntil.Truncate(time.Millisecond).Equal(*pinnedUntil) {
			return nil
		}
	}
	return repository.ErrNotFound
}

// resolvePostUpdateResult 区分“更新值未变化”和“记录不存在”。GORM 的
// 默认软删除作用域同时确保已删除帖子在这里按不存在处理。
func (r *PostRepository) resolvePostUpdateResult(ctx context.Context, id uint, result *gorm.DB) error {
	if result.Error != nil {
		return translateError(result.Error)
	}
	if result.RowsAffected > 0 {
		return nil
	}

	var post model.Post
	return translateError(r.db.WithContext(ctx).Select("id").First(&post, id).Error)
}

func (r *PostRepository) DeletePost(ctx context.Context, id uint) error {
	return translateDeleteResult(r.db.WithContext(ctx).Delete(&model.Post{}, id))
}
