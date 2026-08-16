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

func (h *LikeHandler) ToggleLike(c *gin.Context) {
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

	result, err := h.likeService.Toggle(c.Request.Context(), uint(postID), userID)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.OK(c, httpresp.NewLikeToggleResponse(result))
}
