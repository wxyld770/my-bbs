package httpresponse

import (
	"time"

	"my-bbs/internal/model"
	"my-bbs/internal/service"
	"my-bbs/pkg/pagination"
)

type PostResponse struct {
	ID         uint          `json:"id"`
	CreateTime time.Time     `json:"create_time"`
	UpdateTime time.Time     `json:"update_time"`
	UserID     uint          `json:"user_id"`
	Title      string        `json:"title"`
	Content    string        `json:"content"`
	Visible    uint8         `json:"visible"`
	User       *UserResponse `json:"user"`
}

type PostDetailResponse struct {
	Post         PostResponse `json:"post"`
	LikeCount    int64        `json:"like_count"`
	CommentCount int64        `json:"comment_count"`
	IsLiked      bool         `json:"is_liked"`
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

func NewPostPageResponse(result pagination.Result[model.Post]) PageResponse[PostResponse] {
	return newPageResponse(result, newPostResponse)
}

func newPostResponse(post model.Post) PostResponse {
	return PostResponse{
		ID:         post.ID,
		CreateTime: post.CreateTime,
		UpdateTime: post.UpdateTime,
		UserID:     post.UserID,
		Title:      post.Title,
		Content:    post.Content,
		Visible:    post.Visible,
		User:       newUserResponse(post.User),
	}
}
