package httprequest_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	httprequest "my-bbs/internal/handler/httprequest"
	"my-bbs/pkg/bizerr"

	"github.com/gin-gonic/gin"
)

func TestBindJSONSuccess(t *testing.T) {
	var req httprequest.RegisterRequest
	err := bindBody(t, "application/json; charset=utf-8", `{
		"username":"alice",
		"password":"secret12",
		"nickname":"Alice",
		"invite_code":"A1B2C3"
	}`, &req)
	if err != nil {
		t.Fatalf("BindJSON() error = %v", err)
	}
	if req.Username != "alice" || req.Password != "secret12" || req.Nickname != "Alice" || req.InviteCode != "A1B2C3" {
		t.Fatalf("BindJSON() request = %#v", req)
	}
}

func TestBindJSONRejectsUnsupportedContentType(t *testing.T) {
	tests := []string{"", "text/plain", "application/json; charset"}
	for _, contentType := range tests {
		t.Run(contentType, func(t *testing.T) {
			var req httprequest.LoginRequest
			err := bindBody(t, contentType, `{"username":"alice","password":"secret"}`, &req)
			assertBizError(t, err, bizerr.ErrUnsupportedMediaType.HTTPStatus, bizerr.ErrUnsupportedMediaType.Code, bizerr.ErrUnsupportedMediaType.Message)
		})
	}
}

func TestBindJSONRejectsInvalidJSONBoundary(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		message string
	}{
		{name: "empty", body: "", message: "请求体不能为空"},
		{name: "whitespace", body: " \n\t", message: "请求体不能为空"},
		{name: "syntax", body: `{"username":`, message: "请求体 JSON 格式错误"},
		{name: "not object", body: `[]`, message: "请求体必须是 JSON 对象"},
		{name: "null", body: `null`, message: "请求体必须是 JSON 对象"},
		{
			name:    "multiple values",
			body:    `{"username":"alice","password":"secret"} {"username":"bob","password":"secret"}`,
			message: "请求体只能包含一个 JSON 对象",
		},
		{
			name:    "trailing invalid data",
			body:    `{"username":"alice","password":"secret"} x`,
			message: "请求体 JSON 格式错误",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req httprequest.LoginRequest
			err := bindBody(t, "application/json", tt.body, &req)
			assertBizError(t, err, http.StatusBadRequest, bizerr.ErrBadRequest.Code, tt.message)
		})
	}
}

func TestBindJSONRejectsUnknownAndWrongTypeFields(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		message string
	}{
		{
			name:    "unknown field",
			body:    `{"username":"alice","password":"secret","admin":true}`,
			message: "请求体包含未知字段: admin",
		},
		{
			name:    "wrong field type",
			body:    `{"username":123,"password":"secret"}`,
			message: "字段 username 类型错误",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req httprequest.LoginRequest
			err := bindBody(t, "application/json", tt.body, &req)
			assertBizError(t, err, http.StatusBadRequest, bizerr.ErrBadRequest.Code, tt.message)
		})
	}
}

func TestBindJSONConvertsValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		dst     any
		message string
	}{
		{
			name:    "required",
			body:    `{"password":"secret"}`,
			dst:     &httprequest.LoginRequest{},
			message: "字段 username 不能为空",
		},
		{
			name:    "minimum length",
			body:    `{"username":"ab","password":"secret"}`,
			dst:     &httprequest.RegisterRequest{},
			message: "字段 username 长度不能少于 3 个字符",
		},
		{
			name:    "invitation required",
			body:    `{"username":"alice","password":"secret12","nickname":"Alice"}`,
			dst:     &httprequest.RegisterRequest{},
			message: "字段 invite_code 不能为空",
		},
		{
			name:    "maximum length",
			body:    `{"nickname":"` + strings.Repeat("a", 65) + `"}`,
			dst:     &httprequest.UpdateProfileRequest{},
			message: "字段 nickname 长度不能超过 64 个字符",
		},
		{
			name:    "one of",
			body:    `{"visible":2}`,
			dst:     &httprequest.SetVisibleRequest{},
			message: "字段 visible 的值无效",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := bindBody(t, "application/json", tt.body, tt.dst)
			assertBizError(t, err, http.StatusBadRequest, bizerr.ErrBadRequest.Code, tt.message)
		})
	}
}

func TestBindJSONLimitsBodySize(t *testing.T) {
	body := `{"username":"` + strings.Repeat("a", int(httprequest.MaxJSONBodyBytes)) + `"}`
	var req httprequest.LoginRequest
	err := bindBody(t, "application/json", body, &req)
	assertBizError(t, err, bizerr.ErrPayloadTooLarge.HTTPStatus, bizerr.ErrPayloadTooLarge.Code, bizerr.ErrPayloadTooLarge.Message)
}

func TestBindJSONLimitsStreamingBodySize(t *testing.T) {
	body := `{"username":"` + strings.Repeat("a", int(httprequest.MaxJSONBodyBytes)) + `"}`
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.ContentLength = -1

	var req httprequest.LoginRequest
	err := httprequest.BindJSON(c, &req)
	assertBizError(t, err, bizerr.ErrPayloadTooLarge.HTTPStatus, bizerr.ErrPayloadTooLarge.Code, bizerr.ErrPayloadTooLarge.Message)
}

func bindBody(t *testing.T, contentType, body string, dst any) error {
	t.Helper()
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	if contentType != "" {
		c.Request.Header.Set("Content-Type", contentType)
	}
	return httprequest.BindJSON(c, dst)
}

func assertBizError(t *testing.T, err error, status, code int, message string) {
	t.Helper()
	if err == nil {
		t.Fatal("BindJSON() error = nil")
	}
	actual, ok := bizerr.As(err)
	if !ok {
		t.Fatalf("BindJSON() error type = %T, want *bizerr.Error", err)
	}
	if actual.HTTPStatus != status || actual.Code != code || actual.Message != message {
		t.Fatalf("BindJSON() error = %#v, want status=%d code=%d message=%q", actual, status, code, message)
	}
}
