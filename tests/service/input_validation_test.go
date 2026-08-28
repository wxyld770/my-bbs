package service_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"my-bbs/internal/service"
	"my-bbs/pkg/bizerr"
)

func TestServices_EnforceInputInvariantsOutsideHTTP(t *testing.T) {
	tests := []struct {
		name    string
		call    func() error
		message string
	}{
		{
			name: "bcrypt byte limit",
			call: func() error {
				// 25 个汉字是 75 字节，字符数未超过 HTTP 的 64 字符限制。
				return service.NewUserService(nil).Register(context.Background(), "alice", strings.Repeat("密", 25), "", "TEST01")
			},
			message: "密码不能超过72字节",
		},
		{
			name: "registration nickname length",
			call: func() error {
				return service.NewUserService(nil).Register(context.Background(), "alice", "password1", strings.Repeat("a", 65), "TEST01")
			},
			message: "昵称长度不能超过64个字符",
		},
		{
			name: "post text database limit",
			call: func() error {
				return service.NewPostService(nil, nil, nil, nil).
					CreatePost(context.Background(), 1, "title", strings.Repeat("a", 65_536))
			},
			message: "帖子内容不能超过65535字节",
		},
		{
			name: "blank comment",
			call: func() error {
				return service.NewCommentService(nil, nil, nil).
					CreateComment(context.Background(), 1, 1, " \n\t ")
			},
			message: "评论内容不能为空",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()
			businessError, ok := bizerr.As(err)
			if !ok {
				t.Fatalf("error = %v (%T), want business error", err, err)
			}
			if businessError.HTTPStatus != http.StatusBadRequest || businessError.Message != tt.message {
				t.Fatalf("business error = %#v, want status=400 message=%q", businessError, tt.message)
			}
		})
	}
}
