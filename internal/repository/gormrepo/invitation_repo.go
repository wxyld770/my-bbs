package gormrepo

import (
	"context"
	"time"

	"my-bbs/internal/model"
	"my-bbs/internal/repository"

	"gorm.io/gorm"
)

type InvitationRepository struct {
	db *gorm.DB
}

func NewInvitationRepository(db *gorm.DB) *InvitationRepository {
	return &InvitationRepository{db: db}
}

func (r *InvitationRepository) CreateInvitation(ctx context.Context, invitation *model.Invitation) error {
	return translateError(r.db.WithContext(ctx).Create(invitation).Error)
}

// RegisterUserWithInvitation 原子地消费邀请码并创建用户。
//
// 第一条条件 UPDATE 既验证邀请码可用，也在数据库层取得该行的写锁。
// 并发请求会等待持锁事务结束，随后因 used_at 已被填写而得到零影响行数。
// 用户创建失败时事务回滚，邀请码也会恢复为未使用状态。
func (r *InvitationRepository) RegisterUserWithInvitation(
	ctx context.Context,
	user *model.User,
	code string,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		usedAt := time.Now()
		claim := tx.Model(&model.Invitation{}).
			Where("code = ? AND used_by IS NULL AND used_at IS NULL", code).
			Update("used_at", usedAt)
		if claim.Error != nil {
			return translateError(claim.Error)
		}
		if claim.RowsAffected != 1 {
			return repository.ErrInvitationUnavailable
		}

		user.InviteCode = &code
		if err := tx.Create(user).Error; err != nil {
			return translateError(err)
		}

		consume := tx.Model(&model.Invitation{}).
			Where("code = ? AND used_by IS NULL", code).
			Update("used_by", user.ID)
		if consume.Error != nil {
			return translateError(consume.Error)
		}
		if consume.RowsAffected != 1 {
			return repository.ErrInvitationUnavailable
		}
		return nil
	})
}
