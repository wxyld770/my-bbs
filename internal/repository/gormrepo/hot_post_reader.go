package gormrepo

import (
	"context"
	"time"

	"my-bbs/internal/model"
	"my-bbs/internal/repository"

	"gorm.io/gorm"
)

// HotPostReader 是热榜只读端口的 GORM 实现。
type HotPostReader struct {
	db *gorm.DB
}

func NewHotPostReader(db *gorm.DB) *HotPostReader {
	return &HotPostReader{db: db}
}

type hotPostRow struct {
	model.Post   `gorm:"embedded"`
	LikeCount    int64 `gorm:"column:like_count"`
	CommentCount int64 `gorm:"column:comment_count"`
}

// FindPublicPostEngagements 一次查询完成候选帖子及互动计数聚合。
// 帖子和评论的软删除过滤分别由各自的 GORM 默认作用域完成；点赞为硬删除。
func (r *HotPostReader) FindPublicPostEngagements(
	ctx context.Context,
	publishedFrom time.Time,
	publishedBefore time.Time,
) ([]repository.PostEngagement, error) {
	candidatePostIDs := r.db.Model(&model.Post{}).
		Select("id").
		Where("visible = ?", model.VisiblePublic).
		Where("create_time >= ? AND create_time < ?", publishedFrom, publishedBefore)
	commentCounts := r.db.Model(&model.Comment{}).
		Select("post_id, COUNT(*) AS comment_count").
		Where("post_id IN (?)", candidatePostIDs).
		Group("post_id")
	likeCounts := r.db.Model(&model.PostLike{}).
		Select("post_id, COUNT(*) AS like_count").
		Where("post_id IN (?)", candidatePostIDs).
		Group("post_id")

	var rows []hotPostRow
	err := r.db.WithContext(ctx).
		Model(&model.Post{}).
		Select(
			"posts.id, posts.create_time, posts.update_time, posts.user_id, posts.title, "+
				"posts.visible, posts.pinned_until, "+
				"COALESCE(comment_totals.comment_count, 0) AS comment_count, "+
				"COALESCE(like_totals.like_count, 0) AS like_count",
		).
		Joins("LEFT JOIN (?) AS comment_totals ON comment_totals.post_id = posts.id", commentCounts).
		Joins("LEFT JOIN (?) AS like_totals ON like_totals.post_id = posts.id", likeCounts).
		Where("posts.visible = ?", model.VisiblePublic).
		Where("posts.create_time >= ? AND posts.create_time < ?", publishedFrom, publishedBefore).
		Find(&rows).Error
	if err != nil {
		return nil, translateError(err)
	}

	results := make([]repository.PostEngagement, len(rows))
	for i := range rows {
		results[i] = repository.PostEngagement{
			Post:         rows[i].Post,
			LikeCount:    rows[i].LikeCount,
			CommentCount: rows[i].CommentCount,
		}
	}
	return results, nil
}
