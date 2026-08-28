package repository

import (
	"context"
	"time"

	"my-bbs/internal/model"
)

// UserRepository 是用户用例所需的持久化端口。
type UserRepository interface {
	CreateUser(ctx context.Context, user *model.User) error
	FindUserByUsername(ctx context.Context, username string) (*model.User, error)
	FindUserByID(ctx context.Context, id uint) (*model.User, error)
	UpdateUserStatus(ctx context.Context, id uint, status uint) error
	UpdateProfile(ctx context.Context, id uint, nickname, introduction string) error
}

// InvitationRepository 是邀请码生成和注册消费所需的持久化端口。
// RegisterUserWithInvitation 必须在同一个事务中完成邀请码占用、用户创建
// 和使用记录更新，并保证同一邀请码并发消费时最多一个请求成功。
type InvitationRepository interface {
	CreateInvitation(ctx context.Context, invitation *model.Invitation) error
	RegisterUserWithInvitation(ctx context.Context, user *model.User, code string) error
}

// UserReader 是帖子和评论用例所需的用户只读端口。
type UserReader interface {
	FindUserByID(ctx context.Context, id uint) (*model.User, error)
	FindUsersByIDs(ctx context.Context, ids []uint) ([]model.User, error)
}

// SearchReader 是全局搜索用例所需的跨资源只读端口。
// keyword 由应用层完成规范化；实现负责按字面量匹配而不是把 LIKE
// 通配符暴露给调用方。
type SearchReader interface {
	SearchUsers(ctx context.Context, keyword string, offset, limit int) ([]model.User, error)
	SearchPublicPosts(ctx context.Context, keyword string, offset, limit int) ([]model.Post, error)
}

// PostReader 是互动用例所需的帖子只读端口。
type PostReader interface {
	FindPostByID(ctx context.Context, id uint) (*model.Post, error)
}

// PostRepository 是帖子用例所需的完整持久化端口。
type PostRepository interface {
	PostReader
	CreatePost(ctx context.Context, post *model.Post) error
	FindPublicPosts(ctx context.Context, offset, limit int) ([]model.Post, error)
	FindPostsByUserID(ctx context.Context, userID uint, offset, limit int) ([]model.Post, error)
	FindPublicPostsByUserID(ctx context.Context, userID uint, offset, limit int) ([]model.Post, error)
	UpdatePost(ctx context.Context, post *model.Post) error
	UpdatePostVisible(ctx context.Context, id uint, visible uint8) error
	SetPostPinnedUntil(ctx context.Context, id uint, pinnedUntil *time.Time) error
	DeletePost(ctx context.Context, id uint) error
}

// PostEngagement 是热榜查询返回的帖子互动只读模型。
// Repository 只负责在给定发布时间范围内聚合计数；热度计算、排序和截断
// 属于应用层规则，由 Service 完成。
type PostEngagement struct {
	Post         model.Post
	LikeCount    int64
	CommentCount int64
}

// HotPostReader 是热榜用例所需的只读端口。
type HotPostReader interface {
	FindPublicPostEngagements(
		ctx context.Context,
		publishedFrom time.Time,
		publishedBefore time.Time,
	) ([]PostEngagement, error)
}

// CommentCounter 是帖子详情所需的评论统计端口。
type CommentCounter interface {
	CountByPostID(ctx context.Context, postID uint) (int64, error)
	CountByPostIDs(ctx context.Context, postIDs []uint) (map[uint]int64, error)
}

// CommentRepository 是评论用例所需的持久化端口。
type CommentRepository interface {
	CommentCounter
	Create(ctx context.Context, comment *model.Comment) error
	FindByID(ctx context.Context, id uint) (*model.Comment, error)
	FindByPostID(ctx context.Context, postID uint, offset, limit int) ([]model.Comment, error)
	SoftDelete(ctx context.Context, id uint) error
}

// LikeReader 是帖子详情所需的点赞查询端口。
type LikeReader interface {
	CountByPostID(ctx context.Context, postID uint) (int64, error)
	CountByPostIDs(ctx context.Context, postIDs []uint) (map[uint]int64, error)
	ExistsByUserAndPost(ctx context.Context, userID, postID uint) (bool, error)
}

// LikeRepository 是点赞用例所需的持久化端口。
type LikeRepository interface {
	LikeReader
	Create(ctx context.Context, like *model.PostLike) error
	FindByUserAndPost(ctx context.Context, userID, postID uint) (*model.PostLike, error)
	DeleteByUserAndPost(ctx context.Context, userID, postID uint) error
}
