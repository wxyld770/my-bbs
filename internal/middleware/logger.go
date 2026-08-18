package middleware

import (
	"net"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"my-bbs/internal/logger"
	"my-bbs/pkg/bizerr"
	"my-bbs/pkg/response"

	"github.com/gin-gonic/gin"
)

// RequestLogger 自定义请求日志中间件
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		method := c.Request.Method
		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}

		logger.Info("HTTP request_id=%s | method=%s | route=%s | path=%s | status=%d | latency=%v",
			GetRequestID(c),
			method,
			route,
			path,
			status,
			latency,
		)
	}
}

// Recovery 自定义 panic 恢复中间件：统一错误响应格式
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				var brokenPipe bool
				if ne, ok := err.(*net.OpError); ok {
					if se, ok := ne.Err.(*os.SyscallError); ok {
						errMsg := strings.ToLower(se.Error())
						if strings.Contains(errMsg, "broken pipe") ||
							strings.Contains(errMsg, "connection reset by peer") {
							brokenPipe = true
						}
					}
				}

				if brokenPipe {
					logger.Warn("Recovery broken pipe | request_id=%s | method=%s | route=%s | path=%s | error=%v",
						GetRequestID(c),
						c.Request.Method,
						requestRoute(c),
						c.Request.URL.Path,
						err,
					)
					if e, ok := err.(error); ok {
						_ = c.Error(e)
					}
					c.Abort()
					return
				}

				logger.Error("Recovery panic recovered | request_id=%s | method=%s | route=%s | path=%s | error=%v\nstack=%s",
					GetRequestID(c),
					c.Request.Method,
					requestRoute(c),
					c.Request.URL.Path,
					err,
					debug.Stack(),
				)
				// A panic unwinds ErrorHandler before it reaches its post-c.Next
				// response logic, so Recovery must write the safe 500 itself.
				response.AbortFail(c, bizerr.ErrInternal)
			}
		}()
		c.Next()
	}
}

// ErrorHandler 统一异常处理中间件：
// Handler / 下层通过 c.Error(err) 上报后，若尚未写响应则在此统一输出
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		err := c.Errors.Last().Err
		businessError, isBusinessError := bizerr.As(err)
		if !isBusinessError || businessError.HTTPStatus >= 500 {
			logger.Error("HTTP unhandled error | request_id=%s | method=%s | route=%s | path=%s | error=%v",
				GetRequestID(c),
				c.Request.Method,
				requestRoute(c),
				c.Request.URL.Path,
				err,
			)
		}

		// 兼容已经写过响应的上游中间件，避免重复写响应体。
		if c.Writer.Written() {
			return
		}

		response.Fail(c, err)
	}
}

func requestRoute(c *gin.Context) string {
	route := c.FullPath()
	if route == "" {
		return "unmatched"
	}
	return route
}
