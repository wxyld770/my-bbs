package httpresponse

import (
	"time"

	"my-bbs/internal/model"
	"my-bbs/pkg/pagination"
)

type CommentResponse struct {
	ID         uint          `json:"id"`
	CreateTime time.Time     `json:"create_time"`
	UpdateTime time.Time     `json:"update_time"`
	PostID     uint          `json:"post_id"`
	UserID     uint          `json:"user_id"`
	Content    string        `json:"content"`
	User       *UserResponse `json:"user"`
}

func NewCommentPageResponse(result pagination.Result[model.Comment]) PageResponse[CommentResponse] {
	return newPageResponse(result, newCommentResponse)
}

func newCommentResponse(comment model.Comment) CommentResponse {
	return CommentResponse{
		ID:         comment.ID,
		CreateTime: comment.CreateTime,
		UpdateTime: comment.UpdateTime,
		PostID:     comment.PostID,
		UserID:     comment.UserID,
		Content:    comment.Content,
		User:       newUserResponse(comment.User),
	}
}
