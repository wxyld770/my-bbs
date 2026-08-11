package handler

import (
	"strconv"

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

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=64"`
	Password string `json:"password" binding:"required,min=6,max=64"`
	Nickname string `json:"nickname"`
}

func (h *UserHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bizerr.ErrBadRequest.WithMessage("参数错误: "+err.Error()))
		return
	}

	if err := h.userService.Register(req.Username, req.Password, req.Nickname); err != nil {
		response.Fail(c, err)
		return
	}

	response.OKMsg(c, "注册成功")
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *UserHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bizerr.ErrBadRequest.WithMessage("参数错误: "+err.Error()))
		return
	}

	token, err := h.userService.Login(req.Username, req.Password)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.OK(c, gin.H{
		"token": token,
	})
}

func (h *UserHandler) GetMe(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Fail(c, bizerr.ErrUnauthorized)
		return
	}

	user, err := h.userService.GetMe(userID)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.OK(c, gin.H{"user": user})
}

func (h *UserHandler) GetPublicProfile(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.Fail(c, bizerr.ErrInvalidUserID)
		return
	}

	user, err := h.userService.GetPublicProfile(uint(id))
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.OK(c, gin.H{"user": user})
}

type UpdateProfileRequest struct {
	Nickname     string `json:"nickname" binding:"max=64"`
	Introduction string `json:"introduction" binding:"max=1024"`
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Fail(c, bizerr.ErrUnauthorized)
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bizerr.ErrBadRequest.WithMessage("参数错误: "+err.Error()))
		return
	}

	if err := h.userService.UpdateProfile(userID, req.Nickname, req.Introduction); err != nil {
		response.Fail(c, err)
		return
	}

	response.OKMsg(c, "资料更新成功")
}
