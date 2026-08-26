package httpresponse

import (
	"time"

	"my-bbs/internal/model"
	"my-bbs/internal/service"
	"my-bbs/pkg/pagination"
)

type PostResponse struct {
	ID          uint          `json:"id"`
	CreateTime  time.Time     `json:"create_time"`
	UpdateTime  time.Time     `json:"update_time"`
	UserID      uint          `json:"user_id"`
	Title       string        `json:"title"`
	Content     string        `json:"content"`
	Visible     uint8         `json:"visible"`
	PinnedUntil *time.Time    `json:"pinned_until"`
	IsPinned    bool          `json:"is_pinned"`
	User        *UserResponse `json:"user"`
}

type PostDetailResponse struct {
	Post         PostResponse `json:"post"`
	LikeCount    int64        `json:"like_count"`
	CommentCount int64        `json:"comment_count"`
	IsLiked      bool         `json:"is_liked"`
}

type PostListItemResponse struct {
	ID           uint          `json:"id"`
	CreateTime   time.Time     `json:"create_time"`
	UpdateTime   time.Time     `json:"update_time"`
	UserID       uint          `json:"user_id"`
	Title        string        `json:"title"`
	Visible      uint8         `json:"visible"`
	PinnedUntil  *time.Time    `json:"pinned_until"`
	IsPinned     bool          `json:"is_pinned"`
	User         *UserResponse `json:"user"`
	LikeCount    int64         `json:"like_count"`
	CommentCount int64         `json:"comment_count"`
}

type PinPostResponse struct {
	PinnedUntil time.Time `json:"pinned_until"`
	IsPinned    bool      `json:"is_pinned"`
}

func NewPinPostResponse(pinnedUntil time.Time) PinPostResponse {
	return PinPostResponse{PinnedUntil: pinnedUntil, IsPinned: true}
}

func NewPostDetailResponse(detail *service.PostDetail) PostDetailResponse {
	if detail == nil {
		return PostDetailResponse{}
	}
	return PostDetailResponse{
		Post:         newPostResponse(detail.Post),
		LikeCount:    detail.LikeCount,
		CommentCount: detail.CommentCount,
		IsLiked:      detail.IsLiked,
	}
}

func NewPostPageResponse(result pagination.Result[service.PostSummary]) PageResponse[PostListItemResponse] {
	now := time.Now()
	return newPageResponse(result, func(summary service.PostSummary) PostListItemResponse {
		return newPostListItemResponse(summary, now)
	})
}

func newPostListItemResponse(summary service.PostSummary, now time.Time) PostListItemResponse {
	pinnedUntil, isPinned := activePin(summary.Post, now)
	return PostListItemResponse{
		ID:           summary.Post.ID,
		CreateTime:   summary.Post.CreateTime,
		UpdateTime:   summary.Post.UpdateTime,
		UserID:       summary.Post.UserID,
		Title:        summary.Post.Title,
		Visible:      summary.Post.Visible,
		PinnedUntil:  pinnedUntil,
		IsPinned:     isPinned,
		User:         newUserResponse(summary.Post.User),
		LikeCount:    summary.LikeCount,
		CommentCount: summary.CommentCount,
	}
}

func newPostResponse(post model.Post) PostResponse {
	pinnedUntil, isPinned := activePin(post, time.Now())
	return PostResponse{
		ID:          post.ID,
		CreateTime:  post.CreateTime,
		UpdateTime:  post.UpdateTime,
		UserID:      post.UserID,
		Title:       post.Title,
		Content:     post.Content,
		Visible:     post.Visible,
		PinnedUntil: pinnedUntil,
		IsPinned:    isPinned,
		User:        newUserResponse(post.User),
	}
}

// activePin 将已过期的数据库时间归一化为对外的未置顶状态。
func activePin(post model.Post, now time.Time) (*time.Time, bool) {
	if !post.IsPinnedAt(now) {
		return nil, false
	}
	return post.PinnedUntil, true
}
