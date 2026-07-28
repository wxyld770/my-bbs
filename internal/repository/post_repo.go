package repository

import (
    "errors"
    "my-bbs/internal/model"
    "gorm.io/gorm"
)

type PostRepository struct {
    db *gorm.DB
}

func NewPostRepository(db *gorm.DB) *PostRepository {
    return &PostRepository{db: db}
}

// CreatePost 创建帖子
func (r *PostRepository) CreatePost(post *model.Post) error {
    result := r.db.Create(post)
    return result.Error
}

// FindPostByID 根据 ID 查找帖子
func (r *PostRepository) FindPostByID(id uint) (*model.Post, error) {
    var post model.Post
    result := r.db.Preload("User").First(&post, id) // Preload 加载关联用户信息
    if result.Error != nil {
        if errors.Is(result.Error, gorm.ErrRecordNotFound) {
            return nil, nil
        }
        return nil, result.Error
    }
    return &post, nil
}

// FindPostsByUserID 查询某用户的所有帖子（用于个人主页）
func (r *PostRepository) FindPostsByUserID(userID uint) ([]model.Post, error) {
    var posts []model.Post
    result := r.db.Preload("User").
        Where("user_id = ?", userID).
        Order("create_time DESC").
        Find(&posts)
    if result.Error != nil {
        return nil, result.Error
    }
    return posts, nil
}

// FindAllPosts 查询所有帖子（用于广场），只返回公开帖子，按创建时间倒序
func (r *PostRepository) FindAllPosts() ([]model.Post, error) {
    var posts []model.Post
    result := r.db.Preload("User").
        Where("visible = ?", 1).
        Order("create_time DESC").
        Find(&posts)
    if result.Error != nil {
        return nil, result.Error
    }
    return posts, nil
}

// UpdatePost 更新帖子
func (r *PostRepository) UpdatePost(post *model.Post) error {
    result := r.db.Updates(post)
    return result.Error
}

// UpdatePostVisible 更新帖子可见性
func (r *PostRepository) UpdatePostVisible(id uint, visible string) error {
    result := r.db.Model(&model.Post{}).Where("id = ?", id).
        Update("visible", visible)
    return result.Error
}

// DeletePost 根据 ID 删除帖子（软删除，写入 deleted 字段）
func (r *PostRepository) DeletePost(id uint) error {
    result := r.db.Delete(&model.Post{}, id)
    if result.Error != nil {
        return result.Error
    }
    if result.RowsAffected == 0 {
        return gorm.ErrRecordNotFound // 未找到记录
    }
    return nil
}