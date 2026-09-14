package handler

import (
	"strconv"

	httpresp "my-bbs/internal/handler/httpresponse"
	"my-bbs/internal/middleware"
	"my-bbs/internal/service"
	"my-bbs/pkg/bizerr"
	"my-bbs/pkg/response"

	"github.com/gin-gonic/gin"
)

type LikeHandler struct {
	likeService *service.LikeService
}

func NewLikeHandler(likeService *service.LikeService) *LikeHandler {
	return &LikeHandler{likeService: likeService}
}

func (h *LikeHandler) Like(c *gin.Context) {
	h.setLikeState(c, true)
}

func (h *LikeHandler) Unlike(c *gin.Context) {
	h.setLikeState(c, false)
}

func (h *LikeHandler) setLikeState(c *gin.Context, liked bool) {
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

	var result *service.LikeResult
	if liked {
		result, err = h.likeService.Like(c.Request.Context(), uint(postID), userID)
	} else {
		result, err = h.likeService.Unlike(c.Request.Context(), uint(postID), userID)
	}
	if err != nil {
		response.ReportError(c, err)
		return
	}

	response.OK(c, httpresp.NewLikeResponse(result))
}
