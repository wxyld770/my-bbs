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

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

func (h *UserHandler) Register(c *gin.Context) {
	var req httpreq.RegisterRequest
	if err := httpreq.BindJSON(c, &req); err != nil {
		response.ReportError(c, err)
		return
	}

	if err := h.userService.Register(c.Request.Context(), req.Username, req.Password, req.Nickname); err != nil {
		response.ReportError(c, err)
		return
	}

	response.OKMsg(c, "注册成功")
}

func (h *UserHandler) Login(c *gin.Context) {
	var req httpreq.LoginRequest
	if err := httpreq.BindJSON(c, &req); err != nil {
		response.ReportError(c, err)
		return
	}

	token, err := h.userService.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		response.ReportError(c, err)
		return
	}

	response.OK(c, httpresp.NewLoginResponse(token))
}

// Logout 使当前请求使用的 Token 立即失效。
func (h *UserHandler) Logout(c *gin.Context) {
	tokenID, tokenIDOK := middleware.GetTokenID(c)
	expiresAt, expiresAtOK := middleware.GetTokenExpiresAt(c)
	if !tokenIDOK || !expiresAtOK {
		response.ReportError(c, bizerr.ErrInvalidToken)
		return
	}

	if err := h.userService.Logout(c.Request.Context(), tokenID, expiresAt); err != nil {
		response.ReportError(c, err)
		return
	}

	response.OKMsg(c, "退出登录成功")
}

func (h *UserHandler) GetMe(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.ReportError(c, bizerr.ErrUnauthorized)
		return
	}

	user, err := h.userService.GetMe(c.Request.Context(), userID)
	if err != nil {
		response.ReportError(c, err)
		return
	}

	response.OK(c, httpresp.NewUserProfileResponse(user))
}

func (h *UserHandler) GetPublicProfile(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.ReportError(c, bizerr.ErrInvalidUserID)
		return
	}

	user, err := h.userService.GetPublicProfile(c.Request.Context(), uint(id))
	if err != nil {
		response.ReportError(c, err)
		return
	}

	response.OK(c, httpresp.NewUserProfileResponse(user))
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.ReportError(c, bizerr.ErrUnauthorized)
		return
	}

	var req httpreq.UpdateProfileRequest
	if err := httpreq.BindJSON(c, &req); err != nil {
		response.ReportError(c, err)
		return
	}

	if err := h.userService.UpdateProfile(c.Request.Context(), userID, req.Nickname, req.Introduction); err != nil {
		response.ReportError(c, err)
		return
	}

	response.OKMsg(c, "资料更新成功")
}
