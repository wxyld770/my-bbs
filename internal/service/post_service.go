package service

import (
	"context"
	"errors"
	"time"

	"my-bbs/internal/authorization"
	"my-bbs/internal/model"
	"my-bbs/internal/repository"
	"my-bbs/pkg/bizerr"
	"my-bbs/pkg/pagination"
	"my-bbs/pkg/set"
)

// DefaultPostPinDuration 是管理员置顶帖子的默认有效期。
const DefaultPostPinDuration = 24 * time.Hour

type PostService struct {
	postRepo    repository.PostRepository
	userRepo    repository.UserReader
	commentRepo repository.CommentCounter
	likeRepo    repository.LikeReader
	admins      authorization.AdminChecker
}

func NewPostService(
	postRepo repository.PostRepository,
	userRepo repository.UserReader,
	commentRepo repository.CommentCounter,
	likeRepo repository.LikeReader,
	adminCheckers ...authorization.AdminChecker,
) *PostService {
	return &PostService{
		postRepo:    postRepo,
		userRepo:    userRepo,
		commentRepo: commentRepo,
		likeRepo:    likeRepo,
		admins:      firstAdminChecker(adminCheckers),
	}
}

// PostDetail 帖子详情（含互动字段）
type PostDetail struct {
	Post         model.Post
	LikeCount    int64
	CommentCount int64
	IsLiked      bool
}

// PostSummary 帖子列表项（含批量聚合的互动计数）。
type PostSummary struct {
	Post         model.Post
	LikeCount    int64
	CommentCount int64
}

// CreatePost 创建帖子（默认公开）
func (s *PostService) CreatePost(ctx context.Context, userID uint, title, content string) error {
	var err error
	title, err = requiredTrimmed(title, "帖子标题不能为空")
	if err != nil {
		return err
	}
	content, err = requiredTrimmed(content, "帖子内容不能为空")
	if err != nil {
		return err
	}
	if err := validateRuneLength(title, "帖子标题", 0, 255); err != nil {
		return err
	}
	if err := validateByteLength(content, "帖子内容", maxTextFieldBytes); err != nil {
		return err
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
func (s *PostService) GetPostsByUser(ctx context.Context, userID uint, q pagination.Query) (pagination.Result[PostSummary], error) {
	q.Normalize()
	posts, err := s.postRepo.FindPostsByUserID(ctx, userID, q.Offset(), q.PageSize)
	if err != nil {
		return pagination.Result[PostSummary]{}, err
	}
	summaries, err := s.summarizePosts(ctx, posts)
	if err != nil {
		return pagination.Result[PostSummary]{}, err
	}
	return pagination.NewResult(summaries, q), nil
}

// GetPublicPostsByUser 分页获取某用户的公开帖（他人主页）
func (s *PostService) GetPublicPostsByUser(ctx context.Context, targetUserID uint, q pagination.Query) (pagination.Result[PostSummary], error) {
	user, err := s.userRepo.FindUserByID(ctx, targetUserID)
	if err != nil {
		return pagination.Result[PostSummary]{}, err
	}
	if user == nil {
		return pagination.Result[PostSummary]{}, bizerr.ErrUserNotFound
	}

	q.Normalize()
	posts, err := s.postRepo.FindPublicPostsByUserID(ctx, targetUserID, q.Offset(), q.PageSize)
	if err != nil {
		return pagination.Result[PostSummary]{}, err
	}
	summaries, err := s.summarizePosts(ctx, posts)
	if err != nil {
		return pagination.Result[PostSummary]{}, err
	}
	return pagination.NewResult(summaries, q), nil
}

// GetAllPosts 分页获取公开帖子（广场，无限下拉）
func (s *PostService) GetAllPosts(ctx context.Context, q pagination.Query) (pagination.Result[PostSummary], error) {
	q.Normalize()
	posts, err := s.postRepo.FindPublicPosts(ctx, q.Offset(), q.PageSize)
	if err != nil {
		return pagination.Result[PostSummary]{}, err
	}
	summaries, err := s.summarizePosts(ctx, posts)
	if err != nil {
		return pagination.Result[PostSummary]{}, err
	}
	return pagination.NewResult(summaries, q), nil
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
		trimmedTitle, err := requiredTrimmed(*title, "帖子标题不能为空")
		if err != nil {
			return err
		}
		if err := validateRuneLength(trimmedTitle, "帖子标题", 0, 255); err != nil {
			return err
		}
		post.Title = trimmedTitle
	}
	if content != nil {
		trimmedContent, err := requiredTrimmed(*content, "帖子内容不能为空")
		if err != nil {
			return err
		}
		if err := validateByteLength(trimmedContent, "帖子内容", maxTextFieldBytes); err != nil {
			return err
		}
		post.Content = trimmedContent
	}
	return mapPostMutationError(s.postRepo.UpdatePost(ctx, post))
}

// DeletePost 删除帖子，作者和管理员均可操作。
func (s *PostService) DeletePost(ctx context.Context, postID uint, userID uint) error {
	if _, err := s.requireAuthorOrAdmin(ctx, postID, userID); err != nil {
		return err
	}
	return mapPostMutationError(s.postRepo.DeletePost(ctx, postID))
}

// PinPost 将公开帖子置顶默认时长，仅管理员可操作。
func (s *PostService) PinPost(ctx context.Context, postID uint, userID uint) (time.Time, error) {
	if err := s.requireAdmin(ctx, userID); err != nil {
		return time.Time{}, err
	}

	post, err := s.postRepo.FindPostByID(ctx, postID)
	if err != nil {
		return time.Time{}, err
	}
	if post == nil {
		return time.Time{}, bizerr.ErrPostNotFound
	}
	if post.Visible != model.VisiblePublic {
		return time.Time{}, bizerr.ErrPrivatePostCannotPin
	}

	// 统一到数据库常用的毫秒精度，避免幂等更新因纳秒被截断而误判。
	pinnedUntil := time.Now().Add(DefaultPostPinDuration).Truncate(time.Millisecond)
	if err := mapPostMutationError(s.postRepo.SetPostPinnedUntil(ctx, postID, &pinnedUntil)); err != nil {
		return time.Time{}, err
	}
	return pinnedUntil, nil
}

// UnpinPost 取消帖子置顶，仅管理员可操作。
func (s *PostService) UnpinPost(ctx context.Context, postID uint, userID uint) error {
	if err := s.requireAdmin(ctx, userID); err != nil {
		return err
	}
	post, err := s.postRepo.FindPostByID(ctx, postID)
	if err != nil {
		return err
	}
	if post == nil {
		return bizerr.ErrPostNotFound
	}
	return mapPostMutationError(s.postRepo.SetPostPinnedUntil(ctx, postID, nil))
}

// SetPostVisible 设置帖子可见性，需验证当前用户是否为作者
func (s *PostService) SetPostVisible(ctx context.Context, postID uint, userID uint, visible uint8) error {
	if !model.IsValidVisible(visible) {
		return bizerr.ErrInvalidVisible
	}
	if _, err := s.requireAuthor(ctx, postID, userID); err != nil {
		return err
	}
	return mapPostMutationError(s.postRepo.UpdatePostVisible(ctx, postID, visible))
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

// requireAuthorOrAdmin 优先按作者授权，非作者再校验管理员身份。
func (s *PostService) requireAuthorOrAdmin(ctx context.Context, postID, userID uint) (*model.Post, error) {
	post, err := s.postRepo.FindPostByID(ctx, postID)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, bizerr.ErrPostNotFound
	}
	if post.UserID == userID {
		return post, nil
	}

	user, err := s.userRepo.FindUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, bizerr.ErrUserNotFound
	}
	if !authorization.IsAdmin(s.admins, user.Username) {
		return nil, bizerr.ErrPostNoPermission
	}
	return post, nil
}

func (s *PostService) requireAdmin(ctx context.Context, userID uint) error {
	user, err := s.userRepo.FindUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return bizerr.ErrUserNotFound
	}
	if !authorization.IsAdmin(s.admins, user.Username) {
		return bizerr.ErrForbidden
	}
	return nil
}

func mapPostMutationError(err error) error {
	if errors.Is(err, repository.ErrNotFound) {
		return bizerr.ErrPostNotFound
	}
	return err
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

// summarizePosts 批量填充作者和互动计数，查询次数不随帖子数量增长。
func (s *PostService) summarizePosts(ctx context.Context, posts []model.Post) ([]PostSummary, error) {
	if err := s.fillUsers(ctx, toPostPtrs(posts)); err != nil {
		return nil, err
	}
	if len(posts) == 0 {
		return []PostSummary{}, nil
	}

	postIDs := make([]uint, len(posts))
	for i := range posts {
		postIDs[i] = posts[i].ID
	}

	likeCounts, err := s.likeRepo.CountByPostIDs(ctx, postIDs)
	if err != nil {
		return nil, err
	}
	commentCounts, err := s.commentRepo.CountByPostIDs(ctx, postIDs)
	if err != nil {
		return nil, err
	}

	summaries := make([]PostSummary, len(posts))
	for i := range posts {
		summaries[i] = PostSummary{
			Post:         posts[i],
			LikeCount:    likeCounts[posts[i].ID],
			CommentCount: commentCounts[posts[i].ID],
		}
	}
	return summaries, nil
}

func toPostPtrs(posts []model.Post) []*model.Post {
	ptrs := make([]*model.Post, len(posts))
	for i := range posts {
		ptrs[i] = &posts[i]
	}
	return ptrs
}
