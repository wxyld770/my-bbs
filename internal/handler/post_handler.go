package handler

import (
	"strconv"

	httpresp "my-bbs/internal/handler/httpresponse"
	"my-bbs/internal/middleware"
	"my-bbs/internal/service"
	"my-bbs/pkg/bizerr"
	"my-bbs/pkg/pagination"
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

	if err := h.postService.CreatePost(c.Request.Context(), userID, req.Title, req.Content); err != nil {
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

	var viewerID uint
	if uid, ok := middleware.GetUserID(c); ok {
		viewerID = uid
	}

	detail, err := h.postService.GetPostByID(c.Request.Context(), uint(id), viewerID)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.OK(c, httpresp.NewPostDetailResponse(detail))
}

func (h *PostHandler) GetAllPosts(c *gin.Context) {
	q := parsePageQuery(c)
	result, err := h.postService.GetAllPosts(c.Request.Context(), q)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, httpresp.NewPostPageResponse(result))
}

func (h *PostHandler) GetMyPosts(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Fail(c, bizerr.ErrUnauthorized)
		return
	}

	q := parsePageQuery(c)
	result, err := h.postService.GetPostsByUser(c.Request.Context(), userID, q)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, httpresp.NewPostPageResponse(result))
}

func (h *PostHandler) GetUserPublicPosts(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.Fail(c, bizerr.ErrInvalidUserID)
		return
	}

	q := parsePageQuery(c)
	result, err := h.postService.GetPublicPostsByUser(c.Request.Context(), uint(id), q)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, httpresp.NewPostPageResponse(result))
}

// parsePageQuery 从 query 解析分页参数
func parsePageQuery(c *gin.Context) pagination.Query {
	pageNo, _ := strconv.Atoi(c.DefaultQuery("pageNo", strconv.Itoa(pagination.DefaultPageNo)))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", strconv.Itoa(pagination.DefaultPageSize)))
	q := pagination.Query{PageNo: pageNo, PageSize: pageSize}
	q.Normalize()
	return q
}

type UpdatePostRequest struct {
	// 指针用于区分“字段未传”和“传了空字符串”。这是部分更新接口的关键。
	Title   *string `json:"title" binding:"omitempty,max=255"`
	Content *string `json:"content"`
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

	if err := h.postService.UpdatePost(c.Request.Context(), uint(id), userID, req.Title, req.Content); err != nil {
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

	if err := h.postService.DeletePost(c.Request.Context(), uint(id), userID); err != nil {
		response.Fail(c, err)
		return
	}

	response.OKMsg(c, "删除成功")
}

type SetVisibleRequest struct {
	Visible *uint8 `json:"visible" binding:"required,oneof=0 1"`
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

	if err := h.postService.SetPostVisible(c.Request.Context(), uint(id), userID, *req.Visible); err != nil {
		response.Fail(c, err)
		return
	}

	response.OKMsg(c, "可见性设置成功")
}
