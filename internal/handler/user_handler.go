package handler

import (
	"strconv"

	httpreq "my-bbs/internal/handler/httprequest"
	httpresp "my-bbs/internal/handler/httpresponse"
	"my-bbs/internal/middleware"
	"my-bbs/internal/model"
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

	if err := h.userService.Register(
		c.Request.Context(),
		req.Username,
		req.Password,
		req.Nickname,
		req.InviteCode,
	); err != nil {
		response.ReportError(c, err)
		return
	}

	response.OKMsg(c, "注册成功")
}

// GenerateInvitation 创建一个新的单次使用邀请码。
func (h *UserHandler) GenerateInvitation(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.ReportError(c, bizerr.ErrUnauthorized)
		return
	}

	code, err := h.userService.GenerateInvitation(c.Request.Context(), userID)
	if err != nil {
		response.ReportError(c, err)
		return
	}

	response.OK(c, httpresp.NewInvitationResponse(code))
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

// ChangePassword 校验原密码并修改当前账号密码。成功后不签发新 Token；
// session_version 已递增，调用方必须清除旧 Token 并重新登录。
func (h *UserHandler) ChangePassword(c *gin.Context) {
	userID, userIDOK := middleware.GetUserID(c)
	sessionVersion, sessionVersionOK := middleware.GetSessionVersion(c)
	if !userIDOK || !sessionVersionOK {
		response.ReportError(c, bizerr.ErrInvalidToken)
		return
	}

	var req httpreq.ChangePasswordRequest
	if err := httpreq.BindJSON(c, &req); err != nil {
		response.ReportError(c, err)
		return
	}
	if err := h.userService.ChangePassword(
		c.Request.Context(),
		userID,
		sessionVersion,
		req.OldPassword,
		req.NewPassword,
	); err != nil {
		response.ReportError(c, err)
		return
	}

	response.OKMsg(c, "密码修改成功，请重新登录")
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

	response.OK(c, httpresp.NewCurrentUserProfileResponse(user, h.userService.IsAdminUsername(user.Username)))
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

	response.OK(c, httpresp.NewUserProfileResponse(user, h.userService.IsAdminUsername(user.Username)))
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

func (h *UserHandler) UpdateAvatar(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.ReportError(c, bizerr.ErrUnauthorized)
		return
	}

	var req httpreq.UpdateAvatarRequest
	if err := httpreq.BindJSON(c, &req); err != nil {
		response.ReportError(c, err)
		return
	}

	if err := h.userService.UpdateAvatar(c.Request.Context(), userID, *req.AvatarURL); err != nil {
		response.ReportError(c, err)
		return
	}

	response.OKMsg(c, "头像更新成功")
}

func (h *UserHandler) MuteUser(c *gin.Context) {
	h.setUserStatus(c, model.UserStatusMuted, "用户已禁言")
}

func (h *UserHandler) UnmuteUser(c *gin.Context) {
	h.setUserStatus(c, model.UserStatusNormal, "用户已解除禁言")
}

func (h *UserHandler) ResetUserPassword(c *gin.Context) {
	targetID, err := strconv.Atoi(c.Param("id"))
	if err != nil || targetID <= 0 {
		response.ReportError(c, bizerr.ErrInvalidUserID)
		return
	}
	actorID, ok := middleware.GetUserID(c)
	if !ok {
		response.ReportError(c, bizerr.ErrUnauthorized)
		return
	}
	if err := h.userService.ResetUserPassword(c.Request.Context(), actorID, uint(targetID)); err != nil {
		response.ReportError(c, err)
		return
	}
	response.OKMsg(c, "密码已重置为用户名")
}

func (h *UserHandler) setUserStatus(c *gin.Context, status uint, successMessage string) {
	targetID, err := strconv.Atoi(c.Param("id"))
	if err != nil || targetID <= 0 {
		response.ReportError(c, bizerr.ErrInvalidUserID)
		return
	}
	actorID, ok := middleware.GetUserID(c)
	if !ok {
		response.ReportError(c, bizerr.ErrUnauthorized)
		return
	}
	if err := h.userService.SetUserStatus(c.Request.Context(), actorID, uint(targetID), status); err != nil {
		response.ReportError(c, err)
		return
	}
	response.OKMsg(c, successMessage)
}
