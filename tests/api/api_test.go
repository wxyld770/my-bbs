package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"my-bbs/internal/model"
	"my-bbs/internal/modules/comment"
	"my-bbs/internal/modules/like"
	"my-bbs/internal/modules/post"
	"my-bbs/internal/modules/user"
	"my-bbs/internal/repository"
	"my-bbs/internal/router"
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
	t.Helper()
	gin.SetMode(gin.TestMode)
	testutil.InitJWT(t)
	db := testutil.NewTestDB(t)

	deps := router.RouterDeps{
		Modules: []router.RouteRegister{
			user.Initialize(db),
			post.Initialize(db),
			comment.Initialize(db),
			like.Initialize(db),
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
	w = doJSON(t, readyRouter, http.MethodGet, "/readyz", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("readyz status=%d body=%s", w.Code, w.Body.String())
	}

	unavailableRouter := router.SetupRouter(router.RouterDeps{})
	w = doJSON(t, unavailableRouter, http.MethodGet, "/readyz", "", nil)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("readyz without checker status=%d body=%s", w.Code, w.Body.String())
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
}

func TestAPI_MutedUser_LoginAndAuthBlocked(t *testing.T) {
	r, db := setupTestRouter(t)
	userRepo := repository.NewUserRepository(db)

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
