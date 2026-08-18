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

type PostHandler struct {
	postService *service.PostService
}

func NewPostHandler(postService *service.PostService) *PostHandler {
	return &PostHandler{postService: postService}
}

func (h *PostHandler) CreatePost(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.ReportError(c, bizerr.ErrUnauthorized)
		return
	}

	var req httpreq.CreatePostRequest
	if err := httpreq.BindJSON(c, &req); err != nil {
		response.ReportError(c, err)
		return
	}

	if err := h.postService.CreatePost(c.Request.Context(), userID, req.Title, req.Content); err != nil {
		response.ReportError(c, err)
		return
	}

	response.OKMsg(c, "发布成功")
}

func (h *PostHandler) GetPost(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.ReportError(c, bizerr.ErrInvalidPostID)
		return
	}

	var viewerID uint
	if uid, ok := middleware.GetUserID(c); ok {
		viewerID = uid
	}

	detail, err := h.postService.GetPostByID(c.Request.Context(), uint(id), viewerID)
	if err != nil {
		response.ReportError(c, err)
		return
	}

	response.OK(c, httpresp.NewPostDetailResponse(detail))
}

func (h *PostHandler) GetAllPosts(c *gin.Context) {
	q, err := httpreq.BindPageQuery(c)
	if err != nil {
		response.ReportError(c, err)
		return
	}
	result, err := h.postService.GetAllPosts(c.Request.Context(), q)
	if err != nil {
		response.ReportError(c, err)
		return
	}
	response.OK(c, httpresp.NewPostPageResponse(result))
}

func (h *PostHandler) GetMyPosts(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.ReportError(c, bizerr.ErrUnauthorized)
		return
	}

	q, err := httpreq.BindPageQuery(c)
	if err != nil {
		response.ReportError(c, err)
		return
	}
	result, err := h.postService.GetPostsByUser(c.Request.Context(), userID, q)
	if err != nil {
		response.ReportError(c, err)
		return
	}
	response.OK(c, httpresp.NewPostPageResponse(result))
}

func (h *PostHandler) GetUserPublicPosts(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.ReportError(c, bizerr.ErrInvalidUserID)
		return
	}

	q, err := httpreq.BindPageQuery(c)
	if err != nil {
		response.ReportError(c, err)
		return
	}
	result, err := h.postService.GetPublicPostsByUser(c.Request.Context(), uint(id), q)
	if err != nil {
		response.ReportError(c, err)
		return
	}
	response.OK(c, httpresp.NewPostPageResponse(result))
}

func (h *PostHandler) UpdatePost(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.ReportError(c, bizerr.ErrInvalidPostID)
		return
	}

	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.ReportError(c, bizerr.ErrUnauthorized)
		return
	}

	var req httpreq.UpdatePostRequest
	if err := httpreq.BindJSON(c, &req); err != nil {
		response.ReportError(c, err)
		return
	}

	if err := h.postService.UpdatePost(c.Request.Context(), uint(id), userID, req.Title, req.Content); err != nil {
		response.ReportError(c, err)
		return
	}

	response.OKMsg(c, "更新成功")
}

func (h *PostHandler) DeletePost(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.ReportError(c, bizerr.ErrInvalidPostID)
		return
	}

	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.ReportError(c, bizerr.ErrUnauthorized)
		return
	}

	if err := h.postService.DeletePost(c.Request.Context(), uint(id), userID); err != nil {
		response.ReportError(c, err)
		return
	}

	response.OKMsg(c, "删除成功")
}

func (h *PostHandler) SetVisible(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.ReportError(c, bizerr.ErrInvalidPostID)
		return
	}

	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.ReportError(c, bizerr.ErrUnauthorized)
		return
	}

	var req httpreq.SetVisibleRequest
	if err := httpreq.BindJSON(c, &req); err != nil {
		response.ReportError(c, err)
		return
	}

	if err := h.postService.SetPostVisible(c.Request.Context(), uint(id), userID, *req.Visible); err != nil {
		response.ReportError(c, err)
		return
	}

	response.OKMsg(c, "可见性设置成功")
}
