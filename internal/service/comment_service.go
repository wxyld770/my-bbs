package service

import (
	"my-bbs/internal/model"
	"my-bbs/internal/repository"
	"my-bbs/pkg/bizerr"
	"my-bbs/pkg/pagination"
	"my-bbs/pkg/set"
)

type CommentService struct {
	commentRepo *repository.CommentRepository
	postRepo    *repository.PostRepository
	userRepo    *repository.UserRepository
}

func NewCommentService(
	commentRepo *repository.CommentRepository,
	postRepo *repository.PostRepository,
	userRepo *repository.UserRepository,
) *CommentService {
	return &CommentService{
		commentRepo: commentRepo,
		postRepo:    postRepo,
		userRepo:    userRepo,
	}
}

// CreateComment 在公开帖下发表评论
func (s *CommentService) CreateComment(postID, userID uint, content string) error {
	if err := s.requirePublicPost(postID); err != nil {
		return err
	}
	comment := &model.Comment{
		PostID:  postID,
		UserID:  userID,
		Content: content,
	}
	return s.commentRepo.Create(comment)
}

// ListComments 分页获取公开帖的评论
func (s *CommentService) ListComments(postID uint, q pagination.Query) (pagination.Result[model.Comment], error) {
	if err := s.requirePublicPost(postID); err != nil {
		return pagination.Result[model.Comment]{}, err
	}
	q.Normalize()
	comments, err := s.commentRepo.FindByPostID(postID, q.Offset(), q.PageSize)
	if err != nil {
		return pagination.Result[model.Comment]{}, err
	}
	if err := s.fillUsers(comments); err != nil {
		return pagination.Result[model.Comment]{}, err
	}
	return pagination.NewResult(comments, q), nil
}

// DeleteComment 删除评论（仅评论作者）
func (s *CommentService) DeleteComment(commentID, userID uint) error {
	comment, err := s.commentRepo.FindByID(commentID)
	if err != nil {
		return err
	}
	if comment == nil {
		return bizerr.ErrCommentNotFound
	}
	if comment.UserID != userID {
		return bizerr.ErrCommentNoPermission
	}
	return s.commentRepo.SoftDelete(commentID)
}

// requirePublicPost 校验帖子存在且公开
func (s *CommentService) requirePublicPost(postID uint) error {
	post, err := s.postRepo.FindPostByID(postID)
	if err != nil {
		return err
	}
	if post == nil || post.IsPrivate() {
		return bizerr.ErrPostNotFound
	}
	return nil
}

func (s *CommentService) fillUsers(comments []model.Comment) error {
	if len(comments) == 0 {
		return nil
	}
	ids := make([]uint, 0, len(comments))
	for i := range comments {
		ids = append(ids, comments[i].UserID)
	}
	users, err := s.userRepo.FindUsersByIDs(set.FromSlice(ids).ToSlice())
	if err != nil {
		return err
	}
	userMap := make(map[uint]*model.User, len(users))
	for i := range users {
		userMap[users[i].ID] = &users[i]
	}
	for i := range comments {
		comments[i].User = userMap[comments[i].UserID]
	}
	return nil
}
