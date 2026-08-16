package service

import (
	"context"
	"strings"

	"my-bbs/internal/model"
	"my-bbs/internal/repository"
	"my-bbs/pkg/bizerr"
	"my-bbs/pkg/pagination"
	"my-bbs/pkg/set"
)

type PostService struct {
	postRepo    *repository.PostRepository
	userRepo    *repository.UserRepository
	commentRepo *repository.CommentRepository
	likeRepo    *repository.LikeRepository
}

func NewPostService(
	postRepo *repository.PostRepository,
	userRepo *repository.UserRepository,
	commentRepo *repository.CommentRepository,
	likeRepo *repository.LikeRepository,
) *PostService {
	return &PostService{
		postRepo:    postRepo,
		userRepo:    userRepo,
		commentRepo: commentRepo,
		likeRepo:    likeRepo,
	}
}

// PostDetail 帖子详情（含互动字段）
type PostDetail struct {
	Post         model.Post
	LikeCount    int64
	CommentCount int64
	IsLiked      bool
}

// CreatePost 创建帖子（默认公开）
func (s *PostService) CreatePost(ctx context.Context, userID uint, title, content string) error {
	title = strings.TrimSpace(title)
	content = strings.TrimSpace(content)
	if title == "" {
		return bizerr.ErrBadRequest.WithMessage("帖子标题不能为空")
	}
	if content == "" {
		return bizerr.ErrBadRequest.WithMessage("帖子内容不能为空")
	}

	post := &model.Post{
		UserID:  userID,
		Title:   title,
		Content: content,
		Visible: model.VisiblePublic,
	}
	return s.postRepo.CreatePost(ctx, post)
}

// GetPostByID 根据 ID 获取帖子详情。
// 公开帖任何人可读；私密帖仅作者可读（viewerID 为作者）。viewerID>0 时填充 is_liked。
func (s *PostService) GetPostByID(ctx context.Context, id uint, viewerID uint) (*PostDetail, error) {
	post, err := s.postRepo.FindPostByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, bizerr.ErrPostNotFound
	}
	if post.IsPrivate() && post.UserID != viewerID {
		return nil, bizerr.ErrPostNotFound
	}
	if err := s.fillUsers(ctx, []*model.Post{post}); err != nil {
		return nil, err
	}

	likeCount, err := s.likeRepo.CountByPostID(ctx, id)
	if err != nil {
		return nil, err
	}
	commentCount, err := s.commentRepo.CountByPostID(ctx, id)
	if err != nil {
		return nil, err
	}

	isLiked := false
	if viewerID > 0 {
		isLiked, err = s.likeRepo.ExistsByUserAndPost(ctx, viewerID, id)
		if err != nil {
			return nil, err
		}
	}

	return &PostDetail{
		Post:         *post,
		LikeCount:    likeCount,
		CommentCount: commentCount,
		IsLiked:      isLiked,
	}, nil
}

// GetPostsByUser 分页获取某用户的帖子（个人主页，含私密，无限下拉）
func (s *PostService) GetPostsByUser(ctx context.Context, userID uint, q pagination.Query) (pagination.Result[model.Post], error) {
	q.Normalize()
	posts, err := s.postRepo.FindPostsByUserID(ctx, userID, q.Offset(), q.PageSize)
	if err != nil {
		return pagination.Result[model.Post]{}, err
	}
	if err := s.fillUsers(ctx, toPostPtrs(posts)); err != nil {
		return pagination.Result[model.Post]{}, err
	}
	return pagination.NewResult(posts, q), nil
}

// GetPublicPostsByUser 分页获取某用户的公开帖（他人主页）
func (s *PostService) GetPublicPostsByUser(ctx context.Context, targetUserID uint, q pagination.Query) (pagination.Result[model.Post], error) {
	user, err := s.userRepo.FindUserByID(ctx, targetUserID)
	if err != nil {
		return pagination.Result[model.Post]{}, err
	}
	if user == nil {
		return pagination.Result[model.Post]{}, bizerr.ErrUserNotFound
	}

	q.Normalize()
	posts, err := s.postRepo.FindPublicPostsByUserID(ctx, targetUserID, q.Offset(), q.PageSize)
	if err != nil {
		return pagination.Result[model.Post]{}, err
	}
	if err := s.fillUsers(ctx, toPostPtrs(posts)); err != nil {
		return pagination.Result[model.Post]{}, err
	}
	return pagination.NewResult(posts, q), nil
}

// GetAllPosts 分页获取公开帖子（广场，无限下拉）
func (s *PostService) GetAllPosts(ctx context.Context, q pagination.Query) (pagination.Result[model.Post], error) {
	q.Normalize()
	posts, err := s.postRepo.FindPublicPosts(ctx, q.Offset(), q.PageSize)
	if err != nil {
		return pagination.Result[model.Post]{}, err
	}
	if err := s.fillUsers(ctx, toPostPtrs(posts)); err != nil {
		return pagination.Result[model.Post]{}, err
	}
	return pagination.NewResult(posts, q), nil
}

// UpdatePost 部分更新帖子：title 和 content 至少提供一个，未提供的字段保持原值。
// 需验证当前用户是否为作者。
func (s *PostService) UpdatePost(ctx context.Context, postID uint, userID uint, title, content *string) error {
	if title == nil && content == nil {
		return bizerr.ErrBadRequest.WithMessage("请至少提供 title 或 content 之一")
	}

	post, err := s.requireAuthor(ctx, postID, userID)
	if err != nil {
		return err
	}
	if title != nil {
		trimmedTitle := strings.TrimSpace(*title)
		if trimmedTitle == "" {
			return bizerr.ErrBadRequest.WithMessage("帖子标题不能为空")
		}
		post.Title = trimmedTitle
	}
	if content != nil {
		trimmedContent := strings.TrimSpace(*content)
		if trimmedContent == "" {
			return bizerr.ErrBadRequest.WithMessage("帖子内容不能为空")
		}
		post.Content = trimmedContent
	}
	return s.postRepo.UpdatePost(ctx, post)
}

// DeletePost 删除帖子，需验证当前用户是否为作者
func (s *PostService) DeletePost(ctx context.Context, postID uint, userID uint) error {
	if _, err := s.requireAuthor(ctx, postID, userID); err != nil {
		return err
	}
	return s.postRepo.DeletePost(ctx, postID)
}

// SetPostVisible 设置帖子可见性，需验证当前用户是否为作者
func (s *PostService) SetPostVisible(ctx context.Context, postID uint, userID uint, visible uint8) error {
	if !model.IsValidVisible(visible) {
		return bizerr.ErrInvalidVisible
	}
	if _, err := s.requireAuthor(ctx, postID, userID); err != nil {
		return err
	}
	return s.postRepo.UpdatePostVisible(ctx, postID, visible)
}

// requireAuthor 加载帖子并校验当前用户是否为作者
func (s *PostService) requireAuthor(ctx context.Context, postID, userID uint) (*model.Post, error) {
	post, err := s.postRepo.FindPostByID(ctx, postID)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, bizerr.ErrPostNotFound
	}
	if post.UserID != userID {
		return nil, bizerr.ErrPostNoPermission
	}
	return post, nil
}

// fillUsers 批量填充帖子作者信息
func (s *PostService) fillUsers(ctx context.Context, posts []*model.Post) error {
	if len(posts) == 0 {
		return nil
	}

	ids := make([]uint, 0, len(posts))
	for _, post := range posts {
		if post != nil {
			ids = append(ids, post.UserID)
		}
	}

	users, err := s.userRepo.FindUsersByIDs(ctx, set.FromSlice(ids).ToSlice())
	if err != nil {
		return err
	}

	userMap := make(map[uint]*model.User, len(users))
	for i := range users {
		userMap[users[i].ID] = &users[i]
	}
	for _, post := range posts {
		if post != nil {
			post.User = userMap[post.UserID]
		}
	}
	return nil
}

func toPostPtrs(posts []model.Post) []*model.Post {
	ptrs := make([]*model.Post, len(posts))
	for i := range posts {
		ptrs[i] = &posts[i]
	}
	return ptrs
}
