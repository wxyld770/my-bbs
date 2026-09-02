package gormrepo

import (
	"context"

	"my-bbs/internal/model"

	"gorm.io/gorm"
)

type MessageRepository struct {
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

func (r *MessageRepository) CreateMessage(ctx context.Context, message *model.Message) error {
	return translateError(r.db.WithContext(ctx).Create(message).Error)
}

func (r *MessageRepository) FindMessagesByUserID(
	ctx context.Context,
	userID uint,
	offset, limit int,
) ([]model.Message, error) {
	var messages []model.Message
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("create_time DESC").
		Order("id DESC").
		Offset(offset).
		Limit(limit).
		Find(&messages).Error
	return messages, translateError(err)
}

func (r *MessageRepository) FindAllMessages(ctx context.Context, offset, limit int) ([]model.Message, error) {
	var messages []model.Message
	err := r.db.WithContext(ctx).
		Order("create_time DESC").
		Order("id DESC").
		Offset(offset).
		Limit(limit).
		Find(&messages).Error
	return messages, translateError(err)
}
