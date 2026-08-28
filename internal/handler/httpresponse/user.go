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
	ID              uint       `json:"id"`
	CreateTime      time.Time  `json:"create_time"`
	UpdateTime      time.Time  `json:"update_time"`
	Username        string     `json:"username"`
	Nickname        string     `json:"nickname"`
	Status          uint       `json:"status"`
	Introduction    string     `json:"introduction"`
	AvatarURL       string     `json:"avatar_url"`
	AvatarUpdatedAt *time.Time `json:"avatar_updated_at,omitempty"`
	IsAdmin         bool       `json:"is_admin"`
}

type UserProfileResponse struct {
	User *UserResponse `json:"user"`
}

func NewUserProfileResponse(user *model.User, isAdmin ...bool) UserProfileResponse {
	return UserProfileResponse{User: newUserResponse(user, isAdmin...)}
}

// NewCurrentUserProfileResponse 只在“我的资料”返回头像冷却时间；公开用户资料、
// 帖子作者和评论作者无需暴露该内部限额状态。
func NewCurrentUserProfileResponse(user *model.User, isAdmin ...bool) UserProfileResponse {
	response := newUserResponse(user, isAdmin...)
	if response != nil {
		response.AvatarUpdatedAt = user.AvatarUpdatedAt
	}
	return UserProfileResponse{User: response}
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
		AvatarURL:    user.AvatarURL,
		IsAdmin:      admin,
	}
}
