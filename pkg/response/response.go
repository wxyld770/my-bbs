package response

import (
	"net/http"

	"my-bbs/pkg/bizerr"

	"github.com/gin-gonic/gin"
)

// CodeSuccess 成功业务码
const CodeSuccess = 0

// Body 统一响应体
type Body struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// OK 成功响应（可带 data）
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Body{
		Code:    CodeSuccess,
		Message: "ok",
		Data:    data,
	})
}

// OKMsg 成功响应（仅消息）
func OKMsg(c *gin.Context, message string) {
	c.JSON(http.StatusOK, Body{
		Code:    CodeSuccess,
		Message: message,
	})
}

// Fail 失败响应：自动识别 bizerr.Error，否则当作内部错误
func Fail(c *gin.Context, err error) {
	if err == nil {
		return
	}
	if e, ok := bizerr.As(err); ok {
		c.JSON(e.HTTPStatus, Body{
			Code:    e.Code,
			Message: e.Message,
		})
		return
	}
	c.JSON(http.StatusInternalServerError, Body{
		Code:    bizerr.ErrInternal.Code,
		Message: bizerr.ErrInternal.Message,
	})
}

// ReportError reports an error to ErrorHandler and stops the remaining
// handlers. It deliberately does not write a response: the middleware owns
// error logging and conversion to the public response format.
func ReportError(c *gin.Context, err error) {
	if err == nil {
		return
	}
	_ = c.Error(err)
	c.Abort()
}

// AbortFail 失败并中断后续处理（中间件用）
func AbortFail(c *gin.Context, err error) {
	Fail(c, err)
	c.Abort()
}
