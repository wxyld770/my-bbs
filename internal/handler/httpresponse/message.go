package httpresponse

import (
	"time"

	"my-bbs/internal/model"
	"my-bbs/pkg/pagination"
)

type MessageUserResponse struct {
	ID        uint   `json:"id"`
	Username  string `json:"username"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
}

type MessageResponse struct {
	ID         uint                 `json:"id"`
	UserID     uint                 `json:"user_id"`
	Content    string               `json:"content"`
	CreateTime time.Time            `json:"create_time"`
	User       *MessageUserResponse `json:"user"`
}

func NewMessagePageResponse(result pagination.Result[model.Message]) PageResponse[MessageResponse] {
	return newPageResponse(result, newMessageResponse)
}

func newMessageResponse(message model.Message) MessageResponse {
	return MessageResponse{
		ID:         message.ID,
		UserID:     message.UserID,
		Content:    message.Content,
		CreateTime: message.CreateTime,
		User:       newMessageUserResponse(message.User),
	}
}

func newMessageUserResponse(user *model.User) *MessageUserResponse {
	if user == nil {
		return nil
	}
	return &MessageUserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Nickname:  user.Nickname,
		AvatarURL: user.AvatarURL,
	}
}
