package service

import (
	"context"

	"my-bbs/internal/model"
	"my-bbs/pkg/bizerr"
)

type actorReader interface {
	FindUserByID(ctx context.Context, id uint) (*model.User, error)
}

// requireActiveActor 为业务写操作提供纵深校验。HTTP 路由也会统一拦截禁言用户，
// 这里用于防止未来新增入口或直接调用 Service 时绕过只读限制。
func requireActiveActor(ctx context.Context, users actorReader, userID uint) (*model.User, error) {
	if users == nil {
		return nil, bizerr.ErrUserNotFound
	}
	user, err := users.FindUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, bizerr.ErrUserNotFound
	}
	if !user.IsActive() {
		return nil, bizerr.ErrUserMuted
	}
	return user, nil
}
