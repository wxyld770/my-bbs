package handler

import (
	"strconv"

	httpreq "my-bbs/internal/handler/httprequest"
	httpresp "my-bbs/internal/handler/httpresponse"
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

func (h *CommentHandler) CreateComment(c *gin.Context) {
	postID, err := strconv.Atoi(c.Param("id"))
	if err != nil || postID <= 0 {
		response.ReportError(c, bizerr.ErrInvalidPostID)
		return
	}

	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.ReportError(c, bizerr.ErrUnauthorized)
		return
	}

	var req httpreq.CreateCommentRequest
	if err := httpreq.BindJSON(c, &req); err != nil {
		response.ReportError(c, err)
		return
	}

	if err := h.commentService.CreateComment(c.Request.Context(), uint(postID), userID, req.Content); err != nil {
		response.ReportError(c, err)
		return
	}

	response.OKMsg(c, "评论成功")
}

func (h *CommentHandler) ListComments(c *gin.Context) {
	postID, err := strconv.Atoi(c.Param("id"))
	if err != nil || postID <= 0 {
		response.ReportError(c, bizerr.ErrInvalidPostID)
		return
	}

	q, err := httpreq.BindPageQuery(c)
	if err != nil {
		response.ReportError(c, err)
		return
	}
	result, err := h.commentService.ListComments(c.Request.Context(), uint(postID), q)
	if err != nil {
		response.ReportError(c, err)
		return
	}
	response.OK(c, httpresp.NewCommentPageResponse(result))
}

func (h *CommentHandler) DeleteComment(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.ReportError(c, bizerr.ErrInvalidCommentID)
		return
	}

	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.ReportError(c, bizerr.ErrUnauthorized)
		return
	}

	if err := h.commentService.DeleteComment(c.Request.Context(), uint(id), userID); err != nil {
		response.ReportError(c, err)
		return
	}

	response.OKMsg(c, "删除成功")
}
