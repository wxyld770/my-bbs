package service

import (
	"context"
	"errors"
	"strings"

	"my-bbs/internal/model"
	"my-bbs/internal/repository"
	"my-bbs/pkg/bizerr"

	"gorm.io/gorm"
)

// LikeToggleResult 点赞切换结果
type LikeToggleResult struct {
	Liked     bool
	LikeCount int64
}

type LikeService struct {
	likeRepo *repository.LikeRepository
	postRepo *repository.PostRepository
}

func NewLikeService(likeRepo *repository.LikeRepository, postRepo *repository.PostRepository) *LikeService {
	return &LikeService{
		likeRepo: likeRepo,
		postRepo: postRepo,
	}
}

// Toggle 切换点赞：已赞则取消，未赞则点赞
func (s *LikeService) Toggle(ctx context.Context, postID, userID uint) (*LikeToggleResult, error) {
	if err := s.requirePublicPost(ctx, postID); err != nil {
		return nil, err
	}

	existing, err := s.likeRepo.FindByUserAndPost(ctx, userID, postID)
	if err != nil {
		return nil, err
	}

	liked := false
	if existing != nil {
		if err := s.likeRepo.DeleteByUserAndPost(ctx, userID, postID); err != nil {
			return nil, err
		}
		liked = false
	} else {
		like := &model.PostLike{PostID: postID, UserID: userID}
		if err := s.likeRepo.Create(ctx, like); err != nil {
			// 并发下唯一索引冲突：视为已点赞
			if errors.Is(err, gorm.ErrDuplicatedKey) || isDuplicateKey(err) {
				liked = true
			} else {
				return nil, err
			}
		} else {
			liked = true
		}
	}

	count, err := s.likeRepo.CountByPostID(ctx, postID)
	if err != nil {
		return nil, err
	}
	return &LikeToggleResult{Liked: liked, LikeCount: count}, nil
}

func (s *LikeService) requirePublicPost(ctx context.Context, postID uint) error {
	post, err := s.postRepo.FindPostByID(ctx, postID)
	if err != nil {
		return err
	}
	if post == nil || post.IsPrivate() {
		return bizerr.ErrPostNotFound
	}
	return nil
}

// isDuplicateKey 判断 MySQL 唯一键冲突（1062）
func isDuplicateKey(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "Duplicate entry") || strings.Contains(msg, "1062")
}
