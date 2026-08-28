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
	ErrBadRequest           = New(http.StatusBadRequest, 40000, "参数错误")
	ErrUnauthorized         = New(http.StatusUnauthorized, 40100, "未登录")
	ErrForbidden            = New(http.StatusForbidden, 40300, "无权限")
	ErrNotFound             = New(http.StatusNotFound, 40400, "资源不存在")
	ErrMethodNotAllowed     = New(http.StatusMethodNotAllowed, 40500, "请求方法不支持")
	ErrConflict             = New(http.StatusConflict, 40900, "资源冲突")
	ErrPayloadTooLarge      = New(http.StatusRequestEntityTooLarge, 41300, "请求体过大")
	ErrUnsupportedMediaType = New(http.StatusUnsupportedMediaType, 41500, "仅支持 application/json 请求")
	ErrTooManyRequests      = New(http.StatusTooManyRequests, 42900, "请求过于频繁，请稍后重试")
	ErrInternal             = New(http.StatusInternalServerError, 50000, "服务器内部错误")
	ErrServiceUnavailable   = New(http.StatusServiceUnavailable, 50300, "服务暂时不可用，请稍后重试")

	ErrInvalidToken                   = New(http.StatusUnauthorized, 40101, "无效的认证令牌")
	ErrTokenExpired                   = New(http.StatusUnauthorized, 40102, "认证令牌已过期，请重新登录")
	ErrTokenMissing                   = New(http.StatusUnauthorized, 40103, "缺少认证令牌")
	ErrTokenFormat                    = New(http.StatusUnauthorized, 40104, "认证令牌格式错误，请使用 Bearer <token>")
	ErrLoginFailed                    = New(http.StatusUnauthorized, 40105, "用户名或密码错误")
	ErrUsernameExists                 = New(http.StatusConflict, 40901, "用户名已存在")
	ErrPrivatePostCannotPin           = New(http.StatusConflict, 40903, "私密帖子不能置顶")
	ErrUserNotFound                   = New(http.StatusNotFound, 40401, "用户不存在")
	ErrUserMuted                      = New(http.StatusForbidden, 40303, "账号已被禁言")
	ErrAdminCannotManageAdmin         = New(http.StatusForbidden, 40304, "不能管理其他管理员账号")
	ErrInvitationGenerationRestricted = New(http.StatusForbidden, 40305, "注册未满 7 天；发布一篇帖子后即可立即生成邀请码")
	ErrPostNotFound                   = New(http.StatusNotFound, 40402, "帖子不存在")
	ErrPostNoPermission               = New(http.StatusForbidden, 40301, "无权限操作此帖子")
	ErrInvalidUserID                  = New(http.StatusBadRequest, 40004, "无效的用户ID")
	ErrInvalidVisible                 = New(http.StatusBadRequest, 40001, "可见性参数无效，仅支持 0（仅自己）或 1（所有人）")
	ErrInvalidPostID                  = New(http.StatusBadRequest, 40002, "无效的帖子ID")
	ErrCommentNotFound                = New(http.StatusNotFound, 40403, "评论不存在")
	ErrCommentNoPermission            = New(http.StatusForbidden, 40302, "无权限操作此评论")
	ErrInvalidCommentID               = New(http.StatusBadRequest, 40003, "无效的评论ID")
	ErrInvitationRequired             = New(http.StatusBadRequest, 40005, "请输入邀请码")
	ErrInvitationUnavailable          = New(http.StatusBadRequest, 40006, "邀请码无效或已使用")
	ErrInvalidAvatarURL               = New(http.StatusBadRequest, 40007, "头像链接必须是有效的 HTTPS 地址")
	ErrAvatarUpdateTooFrequent        = New(http.StatusTooManyRequests, 42901, "头像每 24 小时只能修改一次")
)
