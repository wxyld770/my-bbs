package handler

import (
    "net/http"
    "strconv"
    "my-bbs/internal/service"
    "my-bbs/internal/middleware"
    "github.com/gin-gonic/gin"
)

type PostHandler struct {
    postService *service.PostService
}

func NewPostHandler(postService *service.PostService) *PostHandler {
    return &PostHandler{postService: postService}
}

type CreatePostRequest struct {
    Title   string `json:"title" binding:"required,max=255"`
    Content string `json:"content" binding:"required"`
}

func (h *PostHandler) CreatePost(c *gin.Context) {
    userID, ok := middleware.GetUserID(c)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
        return
    }

    var req CreatePostRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "参数错误: " + err.Error(),
        })
        return
    }

    err := h.postService.CreatePost(userID, req.Title, req.Content)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusCreated, gin.H{
        "message": "发布成功",
    })
}

func (h *PostHandler) GetPost(c *gin.Context) {
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil || id <= 0 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "无效的帖子ID"})
        return
    }

    post, err := h.postService.GetPostByID(uint(id))
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    if post == nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "帖子不存在"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "post": post,
    })
}

func (h *PostHandler) GetAllPosts(c *gin.Context) {
    posts, err := h.postService.GetAllPosts()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "posts": posts,
    })
}

func (h *PostHandler) GetMyPosts(c *gin.Context) {
    userID, ok := middleware.GetUserID(c)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
        return
    }

    posts, err := h.postService.GetPostsByUser(userID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "posts": posts,
    })
}

type UpdatePostRequest struct {
    Title   string `json:"title"`
    Content string `json:"content"`
}

func (h *PostHandler) UpdatePost(c *gin.Context) {
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil || id <= 0 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "无效的帖子ID"})
        return
    }

    userID, ok := middleware.GetUserID(c)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
        return
    }

    var req UpdatePostRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "参数错误: " + err.Error(),
        })
        return
    }

    err = h.postService.UpdatePost(uint(id), userID, req.Title, req.Content)
    if err != nil {
        if err.Error() == "帖子不存在" {
            c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
            return
        }
        if err.Error() == "无权限修改此帖子" {
            c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
            return
        }
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "message": "更新成功",
    })
}

func (h *PostHandler) DeletePost(c *gin.Context) {
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil || id <= 0 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "无效的帖子ID"})
        return
    }

    userID, ok := middleware.GetUserID(c)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
        return
    }

    err = h.postService.DeletePost(uint(id), userID)
    if err != nil {
        if err.Error() == "帖子不存在" {
            c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
            return
        }
        if err.Error() == "无权限删除此帖子" {
            c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
            return
        }
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "message": "删除成功",
    })
}

type SetVisibleRequest struct {
    Visible string `json:"visible" binding:"required,oneof=0 1"`
}

func (h *PostHandler) SetVisible(c *gin.Context) {
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil || id <= 0 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "无效的帖子ID"})
        return
    }

    userID, ok := middleware.GetUserID(c)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
        return
    }

    var req SetVisibleRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
        return
    }

    err = h.postService.SetPostVisible(uint(id), userID, req.Visible)
    if err != nil {
        if err.Error() == "帖子不存在" {
            c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
            return
        }
        if err.Error() == "无权限修改此帖子" {
            c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
            return
        }
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "可见性设置成功"})
}