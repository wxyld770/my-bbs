package service

import (
    "errors"
    "my-bbs/internal/model"
    "my-bbs/internal/repository"
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

// GetPostByID 根据ID获取帖子（含作者信息）
func (s *PostService) GetPostByID(id uint) (*model.Post, error) {
    return s.postRepo.FindPostByID(id)
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
    // 先查询帖子
    post, err := s.postRepo.FindPostByID(postID)
    if err != nil {
        return err
    }
    if post == nil {
        return errors.New("帖子不存在")
    }
    // 权限检查：只有作者可以修改
    if post.UserID != userID {
        return errors.New("无权限修改此帖子")
    }

    // 更新内容
    post.Title = title
    post.Content = content
    return s.postRepo.UpdatePost(post)
}

// DeletePost 删除帖子，需验证当前用户是否为作者
func (s *PostService) DeletePost(postID uint, userID uint) error {
    // 先查询帖子
    post, err := s.postRepo.FindPostByID(postID)
    if err != nil {
        return err
    }
    if post == nil {
        return errors.New("帖子不存在")
    }
    // 权限检查
    if post.UserID != userID {
        return errors.New("无权限删除此帖子")
    }

    return s.postRepo.DeletePost(postID)
}