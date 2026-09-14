package service

import (
	"context"
	"errors"

	postcache "my-bbs/internal/cache"
	"my-bbs/internal/model"
	"my-bbs/internal/repository"
	"my-bbs/pkg/bizerr"
)

// LikeResult 表示将帖子点赞设置为目标状态后的结果。
type LikeResult struct {
	Liked     bool
	LikeCount int64
}

type LikeService struct {
	likeRepo   repository.LikeRepository
	postRepo   repository.PostReader
	userRepo   repository.UserReader
	countCache *postcache.PostCountCache
}

func NewLikeService(likeRepo repository.LikeRepository, postRepo repository.PostReader, userRepo repository.UserReader) *LikeService {
	return NewLikeServiceWithCountCache(likeRepo, postRepo, userRepo, nil)
}

func NewLikeServiceWithCountCache(
	likeRepo repository.LikeRepository,
	postRepo repository.PostReader,
	userRepo repository.UserReader,
	countCache *postcache.PostCountCache,
) *LikeService {
	return &LikeService{
		likeRepo:   likeRepo,
		postRepo:   postRepo,
		userRepo:   userRepo,
		countCache: countCache,
	}
}

// Like 确保用户已点赞帖子。重复调用与并发唯一键冲突均视为成功。
func (s *LikeService) Like(ctx context.Context, postID, userID uint) (*LikeResult, error) {
	if _, err := requireActiveActor(ctx, s.userRepo, userID); err != nil {
		return nil, err
	}
	if err := s.requirePublicPost(ctx, postID); err != nil {
		return nil, err
	}

	like := &model.PostLike{PostID: postID, UserID: userID}
	if err := s.likeRepo.Create(ctx, like); err != nil && !errors.Is(err, repository.ErrAlreadyExists) {
		return nil, err
	}
	return s.result(ctx, postID, true)
}

// Unlike 确保用户未点赞帖子。重复调用与并发删除均视为成功。
func (s *LikeService) Unlike(ctx context.Context, postID, userID uint) (*LikeResult, error) {
	if _, err := requireActiveActor(ctx, s.userRepo, userID); err != nil {
		return nil, err
	}
	if err := s.requirePublicPost(ctx, postID); err != nil {
		return nil, err
	}

	if err := s.likeRepo.DeleteByUserAndPost(ctx, userID, postID); err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}
	return s.result(ctx, postID, false)
}

func (s *LikeService) result(ctx context.Context, postID uint, liked bool) (*LikeResult, error) {
	if s.countCache != nil {
		// DB 写入已完成后先失效缓存；即使随后的统计失败，也不能继续暴露旧值。
		// 写请求之间可能交错，因此不在这里回写可能已经过时的精确计数。
		s.countCache.DeleteLikeCounts(ctx, postID)
	}
	count, err := s.likeRepo.CountByPostID(ctx, postID)
	if err != nil {
		return nil, err
	}
	return &LikeResult{Liked: liked, LikeCount: count}, nil
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
