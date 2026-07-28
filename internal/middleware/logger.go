package middleware

import (
	"net"
	"net/http/httputil"
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
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()

		if query != "" {
			path = path + "?" + query
		}

		logger.Info("HTTP status=%d | latency=%v | ip=%s | method=%-7s | path=%s %s",
			status,
			latency,
			clientIP,
			method,
			path,
			errorMessage,
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

				httpRequest, _ := httputil.DumpRequest(c.Request, false)
				if brokenPipe {
					logger.Warn("Recovery broken pipe | error=%v | request=%s", err, string(httpRequest))
					if e, ok := err.(error); ok {
						_ = c.Error(e)
					}
					c.Abort()
					return
				}

				logger.Error("Recovery panic recovered | error=%v\nrequest=%s\nstack=%s",
					err,
					string(httpRequest),
					debug.Stack(),
				)
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
		// 已经写过响应（如 Handler 自己调用了 response.Fail）则不再重复写
		if c.Writer.Written() {
			return
		}

		err := c.Errors.Last().Err
		response.Fail(c, err)
	}
}
