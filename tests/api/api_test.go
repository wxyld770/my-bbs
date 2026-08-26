package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"my-bbs/internal/authorization"
	"my-bbs/internal/model"
	"my-bbs/internal/modules/comment"
	"my-bbs/internal/modules/like"
	"my-bbs/internal/modules/post"
	"my-bbs/internal/modules/user"
	"my-bbs/internal/repository/gormrepo"
	"my-bbs/internal/router"
	"my-bbs/internal/service"
	"my-bbs/pkg/bizerr"
	"my-bbs/tests/testutil"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type readinessCheckerFunc func(context.Context) error

func (f readinessCheckerFunc) PingContext(ctx context.Context) error {
	return f(ctx)
}

func setupTestRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	return setupTestRouterWithAdminUsers(t, "admin")
}

func setupTestRouterWithAdminUsers(t *testing.T, usernames ...string) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	testutil.InitJWT(t)
	db := testutil.NewTestDB(t)
	redisClient := testutil.NewTestRedis(t)
	adminUsers := authorization.NewAdminUsers(usernames...)
	userRepo := gormrepo.NewUserRepository(db)
	userService := service.NewUserService(userRepo)
	for _, username := range adminUsers.Usernames() {
		if err := userService.Register(context.Background(), username, "password1", username); err != nil {
			t.Fatalf("seed configured admin %q before router startup: %v", username, err)
		}
	}
	if err := authorization.ValidateExistingAdminUsers(context.Background(), userRepo, adminUsers); err != nil {
		t.Fatalf("validate configured admins before router startup: %v", err)
	}

	deps := router.RouterDeps{
		Modules: []router.RouteRegister{
			user.Initialize(db, redisClient, adminUsers),
			post.Initialize(db, redisClient, adminUsers),
			comment.Initialize(db, redisClient),
			like.Initialize(db, redisClient),
		},
	}
	return router.SetupRouter(deps), db
}

func doJSON(t *testing.T, r http.Handler, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func doRaw(t *testing.T, r http.Handler, method, path, contentType, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func doLocalHealth(r http.Handler, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.RemoteAddr = "127.0.0.1:1234"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func decodeResp(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v body=%s", err, w.Body.String())
	}
	return resp
}

func TestAPI_HealthChecks(t *testing.T) {
	gin.SetMode(gin.TestMode)

	readyRouter := router.SetupRouter(router.RouterDeps{
		ReadinessChecker: readinessCheckerFunc(func(ctx context.Context) error {
			if _, ok := ctx.Deadline(); !ok {
				t.Fatal("readiness context should have a deadline")
			}
			return nil
		}),
		HealthTimeout: 100 * time.Millisecond,
	})

	w := doJSON(t, readyRouter, http.MethodGet, "/livez", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("livez status=%d body=%s", w.Code, w.Body.String())
	}
	w = doLocalHealth(readyRouter, "/readyz")
	if w.Code != http.StatusOK {
		t.Fatalf("readyz status=%d body=%s", w.Code, w.Body.String())
	}

	publicReq := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	publicReq.RemoteAddr = "203.0.113.10:1234"
	publicResp := httptest.NewRecorder()
	readyRouter.ServeHTTP(publicResp, publicReq)
	if publicResp.Code != http.StatusNotFound {
		t.Fatalf("public readyz status=%d, want 404 body=%s", publicResp.Code, publicResp.Body.String())
	}

	proxiedReq := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	proxiedReq.RemoteAddr = "127.0.0.1:1234"
	proxiedReq.Header.Set("X-Real-IP", "203.0.113.11")
	proxiedResp := httptest.NewRecorder()
	readyRouter.ServeHTTP(proxiedResp, proxiedReq)
	if proxiedResp.Code != http.StatusNotFound {
		t.Fatalf("proxied public readyz status=%d, want 404 body=%s", proxiedResp.Code, proxiedResp.Body.String())
	}

	unavailableRouter := router.SetupRouter(router.RouterDeps{})
	w = doLocalHealth(unavailableRouter, "/readyz")
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("readyz without checker status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestAPI_UsesUnifiedErrorsForUnknownRouteAndMethod(t *testing.T) {
	r := router.SetupRouter(router.RouterDeps{})
	tests := []struct {
		name   string
		method string
		path   string
		want   *bizerr.Error
	}{
		{name: "unknown route", method: http.MethodGet, path: "/missing", want: bizerr.ErrNotFound},
		{name: "method not allowed", method: http.MethodPost, path: "/livez", want: bizerr.ErrMethodNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := doRaw(t, r, tt.method, tt.path, "", "")
			if w.Code != tt.want.HTTPStatus {
				t.Fatalf("status=%d, want=%d body=%s", w.Code, tt.want.HTTPStatus, w.Body.String())
			}
			body := decodeResp(t, w)
			if got := int(body["code"].(float64)); got != tt.want.Code {
				t.Fatalf("code=%d, want=%d body=%v", got, tt.want.Code, body)
			}
			if requestID := w.Header().Get("X-Request-ID"); requestID == "" {
				t.Fatal("response is missing X-Request-ID")
			}
		})
	}
}

func TestAPI_StrictJSONRequestBoundary(t *testing.T) {
	r, _ := setupTestRouter(t)
	tests := []struct {
		name        string
		contentType string
		body        string
		status      int
		code        int
		message     string
	}{
		{
			name:        "unknown field",
			contentType: "application/json",
			body:        `{"username":"alice","password":"password1","role":"admin"}`,
			status:      http.StatusBadRequest,
			code:        bizerr.ErrBadRequest.Code,
			message:     "请求体包含未知字段: role",
		},
		{
			name:        "multiple json objects",
			contentType: "application/json",
			body:        `{"username":"alice","password":"password1"} {"username":"bob"}`,
			status:      http.StatusBadRequest,
			code:        bizerr.ErrBadRequest.Code,
			message:     "请求体只能包含一个 JSON 对象",
		},
		{
			name:        "unsupported media type",
			contentType: "text/plain",
			body:        `{"username":"alice","password":"password1"}`,
			status:      http.StatusUnsupportedMediaType,
			code:        bizerr.ErrUnsupportedMediaType.Code,
			message:     bizerr.ErrUnsupportedMediaType.Message,
		},
		{
			name:        "validation message",
			contentType: "application/json",
			body:        `{"username":"alice","password":"password1","nickname":"` + strings.Repeat("a", 65) + `"}`,
			status:      http.StatusBadRequest,
			code:        bizerr.ErrBadRequest.Code,
			message:     "字段 nickname 长度不能超过 64 个字符",
		},
		{
			name:        "payload too large",
			contentType: "application/json",
			body:        `{"username":"` + strings.Repeat("a", (1<<20)+1) + `"}`,
			status:      http.StatusRequestEntityTooLarge,
			code:        bizerr.ErrPayloadTooLarge.Code,
			message:     bizerr.ErrPayloadTooLarge.Message,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := doRaw(t, r, http.MethodPost, "/api/register", tt.contentType, tt.body)
			if w.Code != tt.status {
				t.Fatalf("status=%d, want=%d body=%s", w.Code, tt.status, w.Body.String())
			}
			body := decodeResp(t, w)
			if got := int(body["code"].(float64)); got != tt.code {
				t.Fatalf("code=%d, want=%d body=%v", got, tt.code, body)
			}
			if got, _ := body["message"].(string); got != tt.message {
				t.Fatalf("message=%q, want=%q", got, tt.message)
			}
			if requestID := w.Header().Get("X-Request-ID"); requestID == "" {
				t.Fatal("response is missing X-Request-ID")
			}
		})
	}
}

func TestAPI_RejectsMalformedPaginationQuery(t *testing.T) {
	r, _ := setupTestRouter(t)
	w := doRaw(t, r, http.MethodGet, "/api/posts?pageNo=not-a-number", "", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want=400 body=%s", w.Code, w.Body.String())
	}
	body := decodeResp(t, w)
	if got, _ := body["message"].(string); got != "字段 pageNo 必须是正整数" {
		t.Fatalf("message=%q", got)
	}

	w = doRaw(t, r, http.MethodGet, "/api/posts?pageNo=502&pageSize=10", "", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("deep pagination status=%d, want=400 body=%s", w.Code, w.Body.String())
	}
	body = decodeResp(t, w)
	if got, _ := body["message"].(string); got != "分页位置不能超过 5000 条" {
		t.Fatalf("deep pagination message=%q", got)
	}
}

func TestAPI_CorePath_RegisterLoginPostLikeComment(t *testing.T) {
	r, _ := setupTestRouter(t)

	w := doJSON(t, r, http.MethodPost, "/api/register", "", map[string]string{
		"username": "coreuser",
		"password": "password1",
		"nickname": "Core",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("register status=%d body=%s", w.Code, w.Body.String())
	}

	w = doJSON(t, r, http.MethodPost, "/api/login", "", map[string]string{
		"username": "coreuser",
		"password": "password1",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("login status=%d body=%s", w.Code, w.Body.String())
	}
	loginResp := decodeResp(t, w)
	data, _ := loginResp["data"].(map[string]any)
	token, _ := data["token"].(string)
	if token == "" {
		t.Fatalf("empty token: %v", loginResp)
	}

	w = doJSON(t, r, http.MethodPost, "/api/posts/create", token, map[string]string{
		"title":   "hello",
		"content": "world",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("create post status=%d body=%s", w.Code, w.Body.String())
	}

	w = doJSON(t, r, http.MethodGet, "/api/posts", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list posts status=%d body=%s", w.Code, w.Body.String())
	}
	listResp := decodeResp(t, w)
	listData, _ := listResp["data"].(map[string]any)
	list, _ := listData["list"].([]any)
	if len(list) != 1 {
		t.Fatalf("want 1 post, got %v", list)
	}
	postObj, _ := list[0].(map[string]any)
	if _, exists := postObj["content"]; exists {
		t.Fatalf("post list must not include content: %v", postObj)
	}
	postID := uint(postObj["id"].(float64))

	w = doJSON(t, r, http.MethodPost, fmt.Sprintf("/api/posts/%d/like", postID), token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("like status=%d body=%s", w.Code, w.Body.String())
	}
	likeResp := decodeResp(t, w)
	likeData, _ := likeResp["data"].(map[string]any)
	if liked, _ := likeData["liked"].(bool); !liked {
		t.Fatalf("expected liked=true, got %v", likeData)
	}

	w = doJSON(t, r, http.MethodPost, fmt.Sprintf("/api/posts/%d/comments/create", postID), token, map[string]string{
		"content": "nice post",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("comment status=%d body=%s", w.Code, w.Body.String())
	}

	w = doJSON(t, r, http.MethodGet, "/api/posts", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list posts with counts status=%d body=%s", w.Code, w.Body.String())
	}
	countList := decodeResp(t, w)["data"].(map[string]any)["list"].([]any)
	countedPost := countList[0].(map[string]any)
	if countedPost["like_count"].(float64) != 1 || countedPost["comment_count"].(float64) != 1 {
		t.Fatalf("unexpected list counts: %v", countedPost)
	}

	w = doJSON(t, r, http.MethodGet, fmt.Sprintf("/api/posts/%d", postID), token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("detail status=%d body=%s", w.Code, w.Body.String())
	}
	detail := decodeResp(t, w)
	detailData, _ := detail["data"].(map[string]any)
	if detailData["like_count"].(float64) != 1 || detailData["comment_count"].(float64) != 1 {
		t.Fatalf("unexpected counts: %v", detailData)
	}
	if !detailData["is_liked"].(bool) {
		t.Fatal("expected is_liked=true")
	}
	detailPost := detailData["post"].(map[string]any)
	if detailPost["content"] != "world" {
		t.Fatalf("post detail must include content: %v", detailPost)
	}
}

func TestAPI_LogoutRevokesOnlyCurrentToken(t *testing.T) {
	r, _ := setupTestRouter(t)

	w := doJSON(t, r, http.MethodPost, "/api/register", "", map[string]string{
		"username": "logoutuser",
		"password": "password1",
		"nickname": "Logout",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("register status=%d body=%s", w.Code, w.Body.String())
	}

	login := func() string {
		t.Helper()
		response := doJSON(t, r, http.MethodPost, "/api/login", "", map[string]string{
			"username": "logoutuser",
			"password": "password1",
		})
		if response.Code != http.StatusOK {
			t.Fatalf("login status=%d body=%s", response.Code, response.Body.String())
		}
		body := decodeResp(t, response)
		data, _ := body["data"].(map[string]any)
		token, _ := data["token"].(string)
		if token == "" {
			t.Fatalf("login returned empty token: %v", body)
		}
		return token
	}

	revokedToken := login()
	activeToken := login()
	w = doJSON(t, r, http.MethodPost, "/api/logout", revokedToken, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("logout status=%d body=%s", w.Code, w.Body.String())
	}

	w = doJSON(t, r, http.MethodGet, "/api/user/me", revokedToken, nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("revoked token status=%d, want 401 body=%s", w.Code, w.Body.String())
	}
	body := decodeResp(t, w)
	if got := int(body["code"].(float64)); got != bizerr.ErrInvalidToken.Code {
		t.Fatalf("revoked token code=%d, want=%d", got, bizerr.ErrInvalidToken.Code)
	}

	w = doJSON(t, r, http.MethodGet, "/api/user/me", activeToken, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("other session status=%d, want 200 body=%s", w.Code, w.Body.String())
	}
}

func TestAPI_MutedUser_LoginAndAuthBlocked(t *testing.T) {
	r, db := setupTestRouter(t)
	userRepo := gormrepo.NewUserRepository(db)

	w := doJSON(t, r, http.MethodPost, "/api/register", "", map[string]string{
		"username": "mutedapi",
		"password": "password1",
		"nickname": "M",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("register: %s", w.Body.String())
	}

	w = doJSON(t, r, http.MethodPost, "/api/login", "", map[string]string{
		"username": "mutedapi",
		"password": "password1",
	})
	loginResp := decodeResp(t, w)
	data, _ := loginResp["data"].(map[string]any)
	token, _ := data["token"].(string)

	user, err := userRepo.FindUserByUsername(context.Background(), "mutedapi")
	if err != nil || user == nil {
		t.Fatalf("find user: %v", err)
	}
	if err := db.Model(&model.User{}).Where("id = ?", user.ID).Update("status", model.UserStatusMuted).Error; err != nil {
		t.Fatalf("mute: %v", err)
	}

	w = doJSON(t, r, http.MethodPost, "/api/login", "", map[string]string{
		"username": "mutedapi",
		"password": "password1",
	})
	if w.Code != http.StatusForbidden {
		t.Fatalf("muted login want 403, got %d body=%s", w.Code, w.Body.String())
	}
	resp := decodeResp(t, w)
	if int(resp["code"].(float64)) != bizerr.ErrUserMuted.Code {
		t.Fatalf("want muted code, got %v", resp)
	}

	w = doJSON(t, r, http.MethodGet, "/api/user/me", token, nil)
	if w.Code != http.StatusForbidden {
		t.Fatalf("muted auth want 403, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestAPI_PrivatePost_AuthorOnly(t *testing.T) {
	r, _ := setupTestRouter(t)

	registerLogin := func(username string) string {
		t.Helper()
		w := doJSON(t, r, http.MethodPost, "/api/register", "", map[string]string{
			"username": username,
			"password": "password1",
			"nickname": username,
		})
		if w.Code != http.StatusOK {
			t.Fatalf("register %s: %s", username, w.Body.String())
		}
		w = doJSON(t, r, http.MethodPost, "/api/login", "", map[string]string{
			"username": username,
			"password": "password1",
		})
		resp := decodeResp(t, w)
		data, _ := resp["data"].(map[string]any)
		return data["token"].(string)
	}

	authorToken := registerLogin("privauthor")
	otherToken := registerLogin("privother")

	w := doJSON(t, r, http.MethodPost, "/api/posts/create", authorToken, map[string]string{
		"title":   "will-be-private",
		"content": "secret",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("create: %s", w.Body.String())
	}

	w = doJSON(t, r, http.MethodGet, "/api/posts", "", nil)
	list := decodeResp(t, w)
	listData := list["data"].(map[string]any)
	posts := listData["list"].([]any)
	postID := uint(posts[0].(map[string]any)["id"].(float64))

	w = doJSON(t, r, http.MethodPost, fmt.Sprintf("/api/posts/visible/%d", postID), authorToken, map[string]any{
		"visible": 0,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("set visible: %s", w.Body.String())
	}

	w = doJSON(t, r, http.MethodGet, fmt.Sprintf("/api/posts/%d", postID), authorToken, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("author detail want 200, got %d body=%s", w.Code, w.Body.String())
	}

	w = doJSON(t, r, http.MethodGet, fmt.Sprintf("/api/posts/%d", postID), otherToken, nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("other detail want 404, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestAPI_PublicProfileAndPosts(t *testing.T) {
	r, _ := setupTestRouter(t)

	w := doJSON(t, r, http.MethodPost, "/api/register", "", map[string]string{
		"username": "profileuser",
		"password": "password1",
		"nickname": "Profile",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("register: %s", w.Body.String())
	}
	w = doJSON(t, r, http.MethodPost, "/api/login", "", map[string]string{
		"username": "profileuser",
		"password": "password1",
	})
	token := decodeResp(t, w)["data"].(map[string]any)["token"].(string)

	w = doJSON(t, r, http.MethodGet, "/api/user/me", token, nil)
	me := decodeResp(t, w)
	userID := uint(me["data"].(map[string]any)["user"].(map[string]any)["id"].(float64))

	w = doJSON(t, r, http.MethodPost, "/api/posts/create", token, map[string]string{
		"title":   "public-one",
		"content": "c1",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("create public: %s", w.Body.String())
	}
	w = doJSON(t, r, http.MethodPost, "/api/posts/create", token, map[string]string{
		"title":   "to-private",
		"content": "c2",
	})
	list := decodeResp(t, doJSON(t, r, http.MethodGet, "/api/posts", "", nil))
	posts := list["data"].(map[string]any)["list"].([]any)
	var privateID uint
	for _, item := range posts {
		p := item.(map[string]any)
		if p["title"].(string) == "to-private" {
			privateID = uint(p["id"].(float64))
		}
	}
	w = doJSON(t, r, http.MethodPost, fmt.Sprintf("/api/posts/visible/%d", privateID), token, map[string]any{
		"visible": 0,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("set private: %s", w.Body.String())
	}

	w = doJSON(t, r, http.MethodGet, fmt.Sprintf("/api/users/%d", userID), "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("public profile: %s", w.Body.String())
	}
	profile := decodeResp(t, w)
	if profile["data"].(map[string]any)["user"].(map[string]any)["username"].(string) != "profileuser" {
		t.Fatalf("unexpected profile: %v", profile)
	}

	w = doJSON(t, r, http.MethodGet, fmt.Sprintf("/api/users/%d/posts", userID), "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("public posts: %s", w.Body.String())
	}
	pubPosts := decodeResp(t, w)["data"].(map[string]any)["list"].([]any)
	if len(pubPosts) != 1 || pubPosts[0].(map[string]any)["title"].(string) != "public-one" {
		t.Fatalf("expected only public-one, got %v", pubPosts)
	}
}
