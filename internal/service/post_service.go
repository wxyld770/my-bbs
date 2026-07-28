package service

import (
	"my-bbs/internal/model"
	"my-bbs/internal/repository"
	"my-bbs/pkg/bizerr"
)

type PostService struct {
	postRepo *repository.PostRepository
}

func NewPostService(postRepo *repository.PostRepository) *PostService {
	return &PostService{postRepo: postRepo}
}

// CreatePost 创建帖子
func (s *PostService) CreatePost(userID uint, title, content string) error {
	post := &model.Post{
		UserID:  userID,
		Title:   title,
		Content: content,
	}
	return s.postRepo.CreatePost(post)
}

// GetPostByID 根据 ID 获取帖子（含作者信息）
func (s *PostService) GetPostByID(id uint) (*model.Post, error) {
	post, err := s.postRepo.FindPostByID(id)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, bizerr.ErrPostNotFound
	}
	return post, nil
}

// GetPostsByUser 获取某用户的所有帖子（个人主页）
func (s *PostService) GetPostsByUser(userID uint) ([]model.Post, error) {
	return s.postRepo.FindPostsByUserID(userID)
}

// GetAllPosts 获取所有帖子（广场）
func (s *PostService) GetAllPosts() ([]model.Post, error) {
	return s.postRepo.FindAllPosts()
}

// UpdatePost 更新帖子，需验证当前用户是否为作者
func (s *PostService) UpdatePost(postID uint, userID uint, title, content string) error {
	post, err := s.postRepo.FindPostByID(postID)
	if err != nil {
		return err
	}
	if post == nil {
		return bizerr.ErrPostNotFound
	}
	if post.UserID != userID {
		return bizerr.ErrPostNoPermission
	}

	post.Title = title
	post.Content = content
	return s.postRepo.UpdatePost(post)
}

// DeletePost 删除帖子，需验证当前用户是否为作者
func (s *PostService) DeletePost(postID uint, userID uint) error {
	post, err := s.postRepo.FindPostByID(postID)
	if err != nil {
		return err
	}
	if post == nil {
		return bizerr.ErrPostNotFound
	}
	if post.UserID != userID {
		return bizerr.ErrPostNoPermission
	}

	return s.postRepo.DeletePost(postID)
}

// SetPostVisible 设置帖子可见性，需验证当前用户是否为作者
func (s *PostService) SetPostVisible(postID uint, userID uint, visible string) error {
	if visible != "0" && visible != "1" {
		return bizerr.ErrInvalidVisible
	}
	post, err := s.postRepo.FindPostByID(postID)
	if err != nil {
		return err
	}
	if post == nil {
		return bizerr.ErrPostNotFound
	}
	if post.UserID != userID {
		return bizerr.ErrPostNoPermission
	}
	return s.postRepo.UpdatePostVisible(postID, visible)
}
