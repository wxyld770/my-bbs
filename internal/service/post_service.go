package service

import (
	"my-bbs/internal/model"
	"my-bbs/internal/repository"
	"my-bbs/pkg/bizerr"
	"my-bbs/pkg/pagination"
	"my-bbs/pkg/set"
)

type PostService struct {
	postRepo *repository.PostRepository
	userRepo *repository.UserRepository
}

func NewPostService(postRepo *repository.PostRepository, userRepo *repository.UserRepository) *PostService {
	return &PostService{
		postRepo: postRepo,
		userRepo: userRepo,
	}
}

// CreatePost 创建帖子（默认公开）
func (s *PostService) CreatePost(userID uint, title, content string) error {
	post := &model.Post{
		UserID:  userID,
		Title:   title,
		Content: content,
		Visible: model.VisiblePublic,
	}
	return s.postRepo.CreatePost(post)
}

// GetPostByID 根据 ID 获取公开帖子；私密帖不可通过此接口访问
func (s *PostService) GetPostByID(id uint) (*model.Post, error) {
	post, err := s.postRepo.FindPostByID(id)
	if err != nil {
		return nil, err
	}
	if post == nil || post.IsPrivate() {
		return nil, bizerr.ErrPostNotFound
	}
	if err := s.fillUsers([]*model.Post{post}); err != nil {
		return nil, err
	}
	return post, nil
}

// GetPostsByUser 分页获取某用户的帖子（个人主页，无限下拉）
func (s *PostService) GetPostsByUser(userID uint, q pagination.Query) (pagination.Result[model.Post], error) {
	q.Normalize()
	posts, err := s.postRepo.FindPostsByUserID(userID, q.Offset(), q.PageSize)
	if err != nil {
		return pagination.Result[model.Post]{}, err
	}
	if err := s.fillUsers(toPostPtrs(posts)); err != nil {
		return pagination.Result[model.Post]{}, err
	}
	return pagination.NewResult(posts, q), nil
}

// GetAllPosts 分页获取公开帖子（广场，无限下拉）
func (s *PostService) GetAllPosts(q pagination.Query) (pagination.Result[model.Post], error) {
	q.Normalize()
	posts, err := s.postRepo.FindPublicPosts(q.Offset(), q.PageSize)
	if err != nil {
		return pagination.Result[model.Post]{}, err
	}
	if err := s.fillUsers(toPostPtrs(posts)); err != nil {
		return pagination.Result[model.Post]{}, err
	}
	return pagination.NewResult(posts, q), nil
}

// UpdatePost 更新帖子，需验证当前用户是否为作者
func (s *PostService) UpdatePost(postID uint, userID uint, title, content string) error {
	post, err := s.requireAuthor(postID, userID)
	if err != nil {
		return err
	}
	post.Title = title
	post.Content = content
	return s.postRepo.UpdatePost(post)
}

// DeletePost 删除帖子，需验证当前用户是否为作者
func (s *PostService) DeletePost(postID uint, userID uint) error {
	if _, err := s.requireAuthor(postID, userID); err != nil {
		return err
	}
	return s.postRepo.DeletePost(postID)
}

// SetPostVisible 设置帖子可见性，需验证当前用户是否为作者
func (s *PostService) SetPostVisible(postID uint, userID uint, visible uint8) error {
	if !model.IsValidVisible(visible) {
		return bizerr.ErrInvalidVisible
	}
	if _, err := s.requireAuthor(postID, userID); err != nil {
		return err
	}
	return s.postRepo.UpdatePostVisible(postID, visible)
}

// requireAuthor 加载帖子并校验当前用户是否为作者
func (s *PostService) requireAuthor(postID, userID uint) (*model.Post, error) {
	post, err := s.postRepo.FindPostByID(postID)
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
func (s *PostService) fillUsers(posts []*model.Post) error {
	if len(posts) == 0 {
		return nil
	}

	ids := make([]uint, 0, len(posts))
	for _, post := range posts {
		if post != nil {
			ids = append(ids, post.UserID)
		}
	}

	users, err := s.userRepo.FindUsersByIDs(set.FromSlice(ids).ToSlice())
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
