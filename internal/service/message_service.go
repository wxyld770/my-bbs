package service

import (
	"context"

	"my-bbs/internal/authorization"
	"my-bbs/internal/model"
	"my-bbs/internal/repository"
	"my-bbs/pkg/bizerr"
	"my-bbs/pkg/pagination"
	"my-bbs/pkg/set"
)

const MessageContentMaxRunes = 2000

type MessageService struct {
	messageRepo repository.MessageRepository
	userRepo    repository.UserReader
	admins      authorization.AdminChecker
}

func NewMessageService(
	messageRepo repository.MessageRepository,
	userRepo repository.UserReader,
	admins authorization.AdminChecker,
) *MessageService {
	return &MessageService{messageRepo: messageRepo, userRepo: userRepo, admins: admins}
}

// CreateMessage 创建留言。留言也是被禁言用户的申诉通道，因此这里只校验
// 用户存在，不要求用户处于正常状态。
func (s *MessageService) CreateMessage(ctx context.Context, userID uint, content string) error {
	if userID == 0 {
		return bizerr.ErrUnauthorized
	}
	content, err := requiredTrimmed(content, "留言内容不能为空")
	if err != nil {
		return err
	}
	if err := validateRuneLength(content, "留言内容", 1, MessageContentMaxRunes); err != nil {
		return err
	}

	user, err := s.userRepo.FindUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return bizerr.ErrUserNotFound
	}

	return s.messageRepo.CreateMessage(ctx, &model.Message{UserID: userID, Content: content})
}

// ListMyMessages 只根据认证上下文中的 userID 查询，HTTP 层不接受目标用户 ID，
// 从接口形态上避免普通用户枚举他人留言。
func (s *MessageService) ListMyMessages(
	ctx context.Context,
	userID uint,
	q pagination.Query,
) (pagination.Result[model.Message], error) {
	if userID == 0 {
		return pagination.Result[model.Message]{}, bizerr.ErrUnauthorized
	}
	q.Normalize()
	messages, err := s.messageRepo.FindMessagesByUserID(ctx, userID, q.Offset(), q.PageSize+1)
	if err != nil {
		return pagination.Result[model.Message]{}, err
	}
	return s.buildPage(ctx, messages, q)
}

// ListAllMessages 只允许配置中的管理员查询全部留言。
func (s *MessageService) ListAllMessages(
	ctx context.Context,
	actorID uint,
	q pagination.Query,
) (pagination.Result[model.Message], error) {
	actor, err := s.userRepo.FindUserByID(ctx, actorID)
	if err != nil {
		return pagination.Result[model.Message]{}, err
	}
	if actor == nil {
		return pagination.Result[model.Message]{}, bizerr.ErrUserNotFound
	}
	if !actor.IsActive() {
		return pagination.Result[model.Message]{}, bizerr.ErrUserMuted
	}
	if !authorization.IsAdmin(s.admins, actor.Username) {
		return pagination.Result[model.Message]{}, bizerr.ErrForbidden
	}

	q.Normalize()
	messages, err := s.messageRepo.FindAllMessages(ctx, q.Offset(), q.PageSize+1)
	if err != nil {
		return pagination.Result[model.Message]{}, err
	}
	return s.buildPage(ctx, messages, q)
}

func (s *MessageService) buildPage(
	ctx context.Context,
	messages []model.Message,
	q pagination.Query,
) (pagination.Result[model.Message], error) {
	hasMore := len(messages) > q.PageSize && q.PageNo <= pagination.MaxOffset/q.PageSize
	if len(messages) > q.PageSize {
		messages = messages[:q.PageSize]
	}
	if messages == nil {
		messages = []model.Message{}
	}
	if err := s.fillMessageUsers(ctx, messages); err != nil {
		return pagination.Result[model.Message]{}, err
	}
	return pagination.Result[model.Message]{
		List:     messages,
		PageNo:   q.PageNo,
		PageSize: q.PageSize,
		HasMore:  hasMore,
	}, nil
}

func (s *MessageService) fillMessageUsers(ctx context.Context, messages []model.Message) error {
	if len(messages) == 0 {
		return nil
	}
	userIDs := make([]uint, 0, len(messages))
	for i := range messages {
		userIDs = append(userIDs, messages[i].UserID)
	}
	users, err := s.userRepo.FindUsersByIDs(ctx, set.FromSlice(userIDs).ToSlice())
	if err != nil {
		return err
	}
	userByID := make(map[uint]*model.User, len(users))
	for i := range users {
		userByID[users[i].ID] = &users[i]
	}
	for i := range messages {
		messages[i].User = userByID[messages[i].UserID]
	}
	return nil
}
