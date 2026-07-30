package handler

import (
	"strconv"

	"my-bbs/internal/middleware"
	"my-bbs/internal/service"
	"my-bbs/pkg/bizerr"
	"my-bbs/pkg/response"

	"github.com/gin-gonic/gin"
)

type CommentHandler struct {
	commentService *service.CommentService
}

func NewCommentHandler(commentService *service.CommentService) *CommentHandler {
	return &CommentHandler{commentService: commentService}
}

type CreateCommentRequest struct {
	Content string `json:"content" binding:"required"`
}

func (h *CommentHandler) CreateComment(c *gin.Context) {
	postID, err := strconv.Atoi(c.Param("id"))
	if err != nil || postID <= 0 {
		response.Fail(c, bizerr.ErrInvalidPostID)
		return
	}

	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Fail(c, bizerr.ErrUnauthorized)
		return
	}

	var req CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bizerr.ErrBadRequest.WithMessage("参数错误: "+err.Error()))
		return
	}

	if err := h.commentService.CreateComment(uint(postID), userID, req.Content); err != nil {
		response.Fail(c, err)
		return
	}

	response.OKMsg(c, "评论成功")
}

func (h *CommentHandler) ListComments(c *gin.Context) {
	postID, err := strconv.Atoi(c.Param("id"))
	if err != nil || postID <= 0 {
		response.Fail(c, bizerr.ErrInvalidPostID)
		return
	}

	q := parsePageQuery(c)
	result, err := h.commentService.ListComments(uint(postID), q)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, result)
}

func (h *CommentHandler) DeleteComment(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.Fail(c, bizerr.ErrInvalidCommentID)
		return
	}

	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Fail(c, bizerr.ErrUnauthorized)
		return
	}

	if err := h.commentService.DeleteComment(uint(id), userID); err != nil {
		response.Fail(c, err)
		return
	}

	response.OKMsg(c, "删除成功")
}
