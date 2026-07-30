package bizerr

import (
	"errors"
	"fmt"
	"net/http"
)

// Error 业务异常：带 HTTP 状态码和业务 code
type Error struct {
	HTTPStatus int    `json:"-"`
	Code       int    `json:"code"`
	Message    string `json:"message"`
}

func (e *Error) Error() string {
	return e.Message
}

// New 创建业务异常
func New(httpStatus, code int, message string) *Error {
	return &Error{
		HTTPStatus: httpStatus,
		Code:       code,
		Message:    message,
	}
}

// WithMessage 基于已有异常替换文案（保留 code/httpStatus）
func (e *Error) WithMessage(message string) *Error {
	return &Error{
		HTTPStatus: e.HTTPStatus,
		Code:       e.Code,
		Message:    message,
	}
}

// WithMessagef 格式化文案
func (e *Error) WithMessagef(format string, args ...any) *Error {
	return e.WithMessage(fmt.Sprintf(format, args...))
}

// As 判断是否为业务异常
func As(err error) (*Error, bool) {
	var e *Error
	if errors.As(err, &e) {
		return e, true
	}
	return nil, false
}

// 预定义业务异常（code 段：4xxxx 客户端，5xxxx 服务端）
var (
	ErrBadRequest   = New(http.StatusBadRequest, 40000, "参数错误")
	ErrUnauthorized = New(http.StatusUnauthorized, 40100, "未登录")
	ErrForbidden    = New(http.StatusForbidden, 40300, "无权限")
	ErrNotFound     = New(http.StatusNotFound, 40400, "资源不存在")
	ErrConflict     = New(http.StatusConflict, 40900, "资源冲突")
	ErrInternal     = New(http.StatusInternalServerError, 50000, "服务器内部错误")

	ErrInvalidToken        = New(http.StatusUnauthorized, 40101, "无效的认证令牌")
	ErrTokenExpired        = New(http.StatusUnauthorized, 40102, "认证令牌已过期，请重新登录")
	ErrTokenMissing        = New(http.StatusUnauthorized, 40103, "缺少认证令牌")
	ErrTokenFormat         = New(http.StatusUnauthorized, 40104, "认证令牌格式错误，请使用 Bearer <token>")
	ErrLoginFailed         = New(http.StatusUnauthorized, 40105, "用户名或密码错误")
	ErrUsernameExists      = New(http.StatusConflict, 40901, "用户名已存在")
	ErrUserNotFound        = New(http.StatusNotFound, 40401, "用户不存在")
	ErrPostNotFound        = New(http.StatusNotFound, 40402, "帖子不存在")
	ErrPostNoPermission    = New(http.StatusForbidden, 40301, "无权限操作此帖子")
	ErrInvalidVisible      = New(http.StatusBadRequest, 40001, "可见性参数无效，仅支持 0（仅自己）或 1（所有人）")
	ErrInvalidPostID       = New(http.StatusBadRequest, 40002, "无效的帖子ID")
	ErrCommentNotFound     = New(http.StatusNotFound, 40403, "评论不存在")
	ErrCommentNoPermission = New(http.StatusForbidden, 40302, "无权限操作此评论")
	ErrInvalidCommentID    = New(http.StatusBadRequest, 40003, "无效的评论ID")
)
