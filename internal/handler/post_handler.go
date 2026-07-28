package handler

import (
	"strconv"

	"my-bbs/internal/middleware"
	"my-bbs/internal/service"
	"my-bbs/pkg/bizerr"
	"my-bbs/pkg/response"

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
		response.Fail(c, bizerr.ErrUnauthorized)
		return
	}

	var req CreatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bizerr.ErrBadRequest.WithMessage("参数错误: "+err.Error()))
		return
	}

	if err := h.postService.CreatePost(userID, req.Title, req.Content); err != nil {
		response.Fail(c, err)
		return
	}

	response.OKMsg(c, "发布成功")
}

func (h *PostHandler) GetPost(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.Fail(c, bizerr.ErrInvalidPostID)
		return
	}

	post, err := h.postService.GetPostByID(uint(id))
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.OK(c, gin.H{"post": post})
}

func (h *PostHandler) GetAllPosts(c *gin.Context) {
	posts, err := h.postService.GetAllPosts()
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.OK(c, gin.H{"posts": posts})
}

func (h *PostHandler) GetMyPosts(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Fail(c, bizerr.ErrUnauthorized)
		return
	}

	posts, err := h.postService.GetPostsByUser(userID)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.OK(c, gin.H{"posts": posts})
}

type UpdatePostRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

func (h *PostHandler) UpdatePost(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.Fail(c, bizerr.ErrInvalidPostID)
		return
	}

	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Fail(c, bizerr.ErrUnauthorized)
		return
	}

	var req UpdatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bizerr.ErrBadRequest.WithMessage("参数错误: "+err.Error()))
		return
	}

	if err := h.postService.UpdatePost(uint(id), userID, req.Title, req.Content); err != nil {
		response.Fail(c, err)
		return
	}

	response.OKMsg(c, "更新成功")
}

func (h *PostHandler) DeletePost(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.Fail(c, bizerr.ErrInvalidPostID)
		return
	}

	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Fail(c, bizerr.ErrUnauthorized)
		return
	}

	if err := h.postService.DeletePost(uint(id), userID); err != nil {
		response.Fail(c, err)
		return
	}

	response.OKMsg(c, "删除成功")
}

type SetVisibleRequest struct {
	Visible string `json:"visible" binding:"required,oneof=0 1"`
}

func (h *PostHandler) SetVisible(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.Fail(c, bizerr.ErrInvalidPostID)
		return
	}

	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Fail(c, bizerr.ErrUnauthorized)
		return
	}

	var req SetVisibleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bizerr.ErrBadRequest.WithMessage("参数错误: "+err.Error()))
		return
	}

	if err := h.postService.SetPostVisible(uint(id), userID, req.Visible); err != nil {
		response.Fail(c, err)
		return
	}

	response.OKMsg(c, "可见性设置成功")
}
