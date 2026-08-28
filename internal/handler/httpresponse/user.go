package httpresponse

import (
	"time"

	"my-bbs/internal/model"
)

type LoginResponse struct {
	Token string `json:"token"`
}

func NewLoginResponse(token string) LoginResponse {
	return LoginResponse{Token: token}
}

// InvitationResponse 只用于创建邀请码的单次响应；没有历史查询接口。
type InvitationResponse struct {
	Code string `json:"code"`
}

func NewInvitationResponse(code string) InvitationResponse {
	return InvitationResponse{Code: code}
}

// UserResponse 是对外用户模型，不包含密码、软删除状态等数据库字段。
type UserResponse struct {
	ID           uint      `json:"id"`
	CreateTime   time.Time `json:"create_time"`
	UpdateTime   time.Time `json:"update_time"`
	Username     string    `json:"username"`
	Nickname     string    `json:"nickname"`
	Status       uint      `json:"status"`
	Introduction string    `json:"introduction"`
	IsAdmin      bool      `json:"is_admin"`
}

type UserProfileResponse struct {
	User *UserResponse `json:"user"`
}

func NewUserProfileResponse(user *model.User, isAdmin ...bool) UserProfileResponse {
	return UserProfileResponse{User: newUserResponse(user, isAdmin...)}
}

func newUserResponse(user *model.User, isAdmin ...bool) *UserResponse {
	if user == nil {
		return nil
	}
	admin := len(isAdmin) > 0 && isAdmin[0]
	return &UserResponse{
		ID:           user.ID,
		CreateTime:   user.CreateTime,
		UpdateTime:   user.UpdateTime,
		Username:     user.Username,
		Nickname:     user.Nickname,
		Status:       user.Status,
		Introduction: user.Introduction,
		IsAdmin:      admin,
	}
}
