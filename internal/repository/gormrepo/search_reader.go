package gormrepo

import (
	"context"
	"strings"

	"my-bbs/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SearchReader 是全局搜索只读端口的 GORM 实现。
type SearchReader struct {
	db *gorm.DB
}

func NewSearchReader(db *gorm.DB) *SearchReader {
	return &SearchReader{db: db}
}

// SearchUsers 按用户名、昵称搜索用户。LIKE 中的通配符会被转义，确保
// 调用方输入的 !、% 和 _ 都作为普通字符参与匹配。
func (r *SearchReader) SearchUsers(ctx context.Context, keyword string, offset, limit int) ([]model.User, error) {
	escaped := escapeLikeLiteral(keyword)
	prefixPattern := escaped + "%"
	containsPattern := "%" + escaped + "%"

	const rankExpression = `CASE
		WHEN username = ? THEN 0
		WHEN nickname = ? THEN 1
		WHEN username LIKE ? ESCAPE '!' THEN 2
		WHEN nickname LIKE ? ESCAPE '!' THEN 3
		WHEN username LIKE ? ESCAPE '!' THEN 4
		WHEN nickname LIKE ? ESCAPE '!' THEN 5
		ELSE 6
	END, update_time DESC, id DESC`

	var users []model.User
	err := r.db.WithContext(ctx).
		Select("id", "create_time", "update_time", "username", "nickname", "introduction", "avatar_url").
		Where("username LIKE ? ESCAPE '!' OR nickname LIKE ? ESCAPE '!'", containsPattern, containsPattern).
		Clauses(clause.OrderBy{Expression: clause.Expr{
			SQL:                rankExpression,
			Vars:               []any{keyword, keyword, prefixPattern, prefixPattern, containsPattern, containsPattern},
			WithoutParentheses: true,
		}}).
		Offset(offset).
		Limit(limit).
		Find(&users).Error
	if err != nil {
		return nil, translateError(err)
	}
	return users, nil
}

// SearchPublicPosts 按标题、正文搜索公开帖子。私密帖在 SQL 层即被排除，
// GORM 的默认 scope 同时负责过滤软删除记录。
func (r *SearchReader) SearchPublicPosts(ctx context.Context, keyword string, offset, limit int) ([]model.Post, error) {
	escaped := escapeLikeLiteral(keyword)
	prefixPattern := escaped + "%"
	containsPattern := "%" + escaped + "%"

	const rankExpression = `CASE
		WHEN title = ? THEN 0
		WHEN title LIKE ? ESCAPE '!' THEN 1
		WHEN title LIKE ? ESCAPE '!' THEN 2
		ELSE 3
	END, create_time DESC, id DESC`

	var posts []model.Post
	err := r.db.WithContext(ctx).
		Select("id", "create_time", "update_time", "user_id", "title", "content", "visible", "pinned_until").
		Where("visible = ?", model.VisiblePublic).
		Where("title LIKE ? ESCAPE '!' OR content LIKE ? ESCAPE '!'", containsPattern, containsPattern).
		Clauses(clause.OrderBy{Expression: clause.Expr{
			SQL:                rankExpression,
			Vars:               []any{keyword, prefixPattern, containsPattern},
			WithoutParentheses: true,
		}}).
		Offset(offset).
		Limit(limit).
		Find(&posts).Error
	if err != nil {
		return nil, translateError(err)
	}
	return posts, nil
}

func escapeLikeLiteral(keyword string) string {
	return strings.NewReplacer(
		"!", "!!",
		"%", "!%",
		"_", "!_",
	).Replace(keyword)
}
