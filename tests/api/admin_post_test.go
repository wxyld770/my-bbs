package api_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"my-bbs/internal/model"
	"my-bbs/internal/repository/gormrepo"
	"my-bbs/pkg/bizerr"
)

const (
	adminTestUsername = "admin"
	adminPinDuration  = 24 * time.Hour
)

func TestAPI_ConfiguredAdminUsesExistingAccountAndUsernameRemainsUnique(t *testing.T) {
	r, _ := setupTestRouter(t)
	adminToken := loginAdminTestUser(t, r, adminTestUsername, "password1")
	assertAdminIdentityResponse(t, r, adminToken, true)

	w := doJSON(t, r, http.MethodPost, "/api/register", "", map[string]string{
		"username":    adminTestUsername,
		"password":    "password1",
		"nickname":    "duplicate admin",
		"invite_code": generateAPIInvitation(t, r, adminToken),
	})
	assertAdminBizError(t, w, bizerr.ErrUsernameExists)
}

func TestAPI_AdminPrivilegesFollowConfiguredUsernames(t *testing.T) {
	r, _ := setupTestRouterWithAdminUsers(t, "moderator")
	moderatorToken := loginAdminTestUser(t, r, "moderator", "password1")
	authorToken := registerAndLoginAdminTestUserWithInviter(t, r, moderatorToken, "configauthor")
	unconfiguredAdminToken := registerAndLoginAdminTestUserWithInviter(t, r, moderatorToken, "admin")
	postID := createAdminTestPost(t, r, authorToken, "configured-admin-target")

	w := doJSON(t, r, http.MethodPost, fmt.Sprintf("/api/posts/pin/%d", postID), unconfiguredAdminToken, nil)
	assertAdminBizError(t, w, bizerr.ErrForbidden)
	w = doJSON(t, r, http.MethodPost, fmt.Sprintf("/api/posts/pin/%d", postID), moderatorToken, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("configured moderator pin status=%d body=%s", w.Code, w.Body.String())
	}
	assertAdminIdentityResponse(t, r, unconfiguredAdminToken, false)
	assertAdminIdentityResponse(t, r, moderatorToken, true)
}

func TestAPI_AdminCanDeleteOtherUsersPost(t *testing.T) {
	r, _ := setupTestRouter(t)
	authorToken := registerAndLoginAdminTestUser(t, r, "deleteauthor")
	ordinaryToken := registerAndLoginAdminTestUser(t, r, "deleteordinary")
	adminToken := loginAdminTestUser(t, r, adminTestUsername, "password1")

	assertAdminIdentityResponse(t, r, ordinaryToken, false)
	assertAdminIdentityResponse(t, r, adminToken, true)

	postID := createAdminTestPost(t, r, authorToken, "admin-delete-target")

	w := doJSON(t, r, http.MethodPost, fmt.Sprintf("/api/posts/del/%d", postID), ordinaryToken, nil)
	assertAdminBizError(t, w, bizerr.ErrPostNoPermission)

	w = doJSON(t, r, http.MethodPost, fmt.Sprintf("/api/posts/pin/%d", postID), ordinaryToken, nil)
	assertAdminBizError(t, w, bizerr.ErrForbidden)
	w = doJSON(t, r, http.MethodPost, fmt.Sprintf("/api/posts/unpin/%d", postID), ordinaryToken, nil)
	assertAdminBizError(t, w, bizerr.ErrForbidden)

	// 管理员删除权限同样覆盖他人的私密帖子。
	w = doJSON(t, r, http.MethodPost, fmt.Sprintf("/api/posts/visible/%d", postID), authorToken, map[string]any{
		"visible": model.VisiblePrivate,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("set target private status=%d body=%s", w.Code, w.Body.String())
	}

	w = doJSON(t, r, http.MethodPost, fmt.Sprintf("/api/posts/del/%d", postID), adminToken, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("admin delete status=%d body=%s", w.Code, w.Body.String())
	}
	w = doJSON(t, r, http.MethodGet, fmt.Sprintf("/api/posts/%d", postID), authorToken, nil)
	assertAdminBizError(t, w, bizerr.ErrPostNotFound)
}

func TestAPI_AdminCanMuteAndUnmuteOrdinaryUsers(t *testing.T) {
	r, _ := setupTestRouter(t)
	targetToken := registerAndLoginAdminTestUser(t, r, "mutetarget")
	ordinaryToken := registerAndLoginAdminTestUser(t, r, "muteordinary")
	adminToken := loginAdminTestUser(t, r, adminTestUsername, "password1")
	targetID := adminTestCurrentUserID(t, r, targetToken)
	adminID := adminTestCurrentUserID(t, r, adminToken)

	w := doJSON(t, r, http.MethodPost, fmt.Sprintf("/api/users/%d/mute", targetID), ordinaryToken, nil)
	assertAdminBizError(t, w, bizerr.ErrForbidden)

	w = doJSON(t, r, http.MethodPost, fmt.Sprintf("/api/users/%d/mute", targetID), adminToken, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("admin mute status=%d body=%s", w.Code, w.Body.String())
	}
	w = doJSON(t, r, http.MethodGet, "/api/user/me", targetToken, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("muted user should retain read access: status=%d body=%s", w.Code, w.Body.String())
	}
	mutedProfile := decodeResp(t, w)["data"].(map[string]any)["user"].(map[string]any)
	if got := uint(mutedProfile["status"].(float64)); got != model.UserStatusMuted {
		t.Fatalf("muted profile status=%d, want=%d", got, model.UserStatusMuted)
	}
	w = doJSON(t, r, http.MethodPost, "/api/login", "", map[string]string{
		"username": "mutetarget",
		"password": "password1",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("muted user should be able to login: status=%d body=%s", w.Code, w.Body.String())
	}
	w = doJSON(t, r, http.MethodPost, "/api/posts/create", targetToken, map[string]string{"title": "blocked", "content": "blocked"})
	assertAdminBizError(t, w, bizerr.ErrUserMuted)

	w = doJSON(t, r, http.MethodPost, fmt.Sprintf("/api/users/%d/unmute", targetID), adminToken, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("admin unmute status=%d body=%s", w.Code, w.Body.String())
	}
	if token := loginAdminTestUser(t, r, "mutetarget", "password1"); token == "" {
		t.Fatal("unmuted user must be able to log in")
	}

	w = doJSON(t, r, http.MethodPost, fmt.Sprintf("/api/users/%d/mute", adminID), adminToken, nil)
	assertAdminBizError(t, w, bizerr.ErrAdminCannotManageAdmin)
}

func TestAPI_AdminPostPinLifecycle(t *testing.T) {
	r, db := setupTestRouter(t)
	authorToken := registerAndLoginAdminTestUser(t, r, "pinauthor")
	adminToken := loginAdminTestUser(t, r, adminTestUsername, "password1")

	olderID := createAdminTestPost(t, r, authorToken, "pin-older")
	newerID := createAdminTestPost(t, r, authorToken, "pin-newer")

	beforePin := time.Now()
	w := doJSON(t, r, http.MethodPost, fmt.Sprintf("/api/posts/pin/%d", olderID), adminToken, nil)
	afterPin := time.Now()
	if w.Code != http.StatusOK {
		t.Fatalf("pin status=%d body=%s", w.Code, w.Body.String())
	}
	pinData := decodeResp(t, w)["data"].(map[string]any)
	if isPinned, _ := pinData["is_pinned"].(bool); !isPinned {
		t.Fatalf("pin response must contain is_pinned=true: %v", pinData)
	}
	pinnedUntil := parseAdminPinnedUntil(t, pinData["pinned_until"])
	if pinnedUntil.Before(beforePin.Add(adminPinDuration-5*time.Second)) ||
		pinnedUntil.After(afterPin.Add(adminPinDuration+5*time.Second)) {
		t.Fatalf("pinned_until=%s, want about 24h after request", pinnedUntil)
	}
	if permanent, _ := pinData["is_permanent"].(bool); permanent {
		t.Fatalf("legacy one-day pin must not be permanent: %v", pinData)
	}
	if duration, _ := pinData["duration"].(string); duration != string(model.PostPinDurationDay) {
		t.Fatalf("legacy duration=%q, want day", duration)
	}

	posts := listAdminTestPosts(t, r)
	if len(posts) < 2 || adminTestPostID(t, posts[0]) != olderID {
		t.Fatalf("active pinned post must sort first: %v", posts)
	}
	assertAdminPinState(t, findAdminTestPost(t, posts, olderID), true)
	assertAdminPinState(t, findAdminTestPost(t, posts, newerID), false)
	detailResponse := doJSON(t, r, http.MethodGet, fmt.Sprintf("/api/posts/%d", olderID), "", nil)
	if detailResponse.Code != http.StatusOK {
		t.Fatalf("pinned detail status=%d body=%s", detailResponse.Code, detailResponse.Body.String())
	}
	detailPost := decodeResp(t, detailResponse)["data"].(map[string]any)["post"].(map[string]any)
	assertAdminPinState(t, detailPost, true)

	t.Run("selectable pin durations", func(t *testing.T) {
		tests := []struct {
			name      string
			duration  model.PostPinDuration
			wantAfter time.Duration
			permanent bool
		}{
			{name: "day", duration: model.PostPinDurationDay, wantAfter: 24 * time.Hour},
			{name: "week", duration: model.PostPinDurationWeek, wantAfter: 7 * 24 * time.Hour},
			{name: "month", duration: model.PostPinDurationMonth, wantAfter: 30 * 24 * time.Hour},
			{name: "permanent", duration: model.PostPinDurationPermanent, permanent: true},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				postID := createAdminTestPost(t, r, authorToken, "pin-duration-"+tt.name)
				before := time.Now()
				response := doJSON(t, r, http.MethodPost, fmt.Sprintf("/api/posts/pin/%d/duration", postID), adminToken, map[string]string{"duration": string(tt.duration)})
				after := time.Now()
				if response.Code != http.StatusOK {
					t.Fatalf("pin duration status=%d body=%s", response.Code, response.Body.String())
				}
				data := decodeResp(t, response)["data"].(map[string]any)
				if data["duration"] != string(tt.duration) || data["is_permanent"] != tt.permanent {
					t.Fatalf("pin duration response=%v", data)
				}
				until := parseAdminPinnedUntil(t, data["pinned_until"])
				if tt.permanent {
					if got := until.Format("2006-01-02 15:04:05.000"); got != "9999-12-31 23:59:59.999" {
						t.Fatalf("permanent pinned_until=%s, want MariaDB DATETIME(3) maximum", got)
					}
				} else if until.Before(before.Add(tt.wantAfter-5*time.Second)) || until.After(after.Add(tt.wantAfter+5*time.Second)) {
					t.Fatalf("pinned_until=%s, want about %s after request", until, tt.wantAfter)
				}
				detail := doJSON(t, r, http.MethodGet, fmt.Sprintf("/api/posts/%d", postID), "", nil)
				post := decodeResp(t, detail)["data"].(map[string]any)["post"].(map[string]any)
				if post["is_permanent"] != tt.permanent {
					t.Fatalf("detail is_permanent=%v, want=%v", post["is_permanent"], tt.permanent)
				}
				unpin := doJSON(t, r, http.MethodPost, fmt.Sprintf("/api/posts/unpin/%d", postID), adminToken, nil)
				if unpin.Code != http.StatusOK {
					t.Fatalf("cleanup unpin status=%d body=%s", unpin.Code, unpin.Body.String())
				}
			})
		}

		invalid := doJSON(t, r, http.MethodPost, fmt.Sprintf("/api/posts/pin/%d/duration", newerID), adminToken, map[string]string{"duration": "forever-ish"})
		if invalid.Code != http.StatusBadRequest {
			t.Fatalf("invalid duration status=%d body=%s", invalid.Code, invalid.Body.String())
		}
	})

	w = doJSON(t, r, http.MethodPost, fmt.Sprintf("/api/posts/unpin/%d", olderID), adminToken, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("unpin status=%d body=%s", w.Code, w.Body.String())
	}
	posts = listAdminTestPosts(t, r)
	newerIndex := indexAdminTestPost(t, posts, newerID)
	olderIndex := indexAdminTestPost(t, posts, olderID)
	if newerIndex >= olderIndex {
		t.Fatalf("unpin must restore chronological order: %v", posts)
	}
	assertAdminPinState(t, findAdminTestPost(t, posts, olderID), false)

	t.Run("private post cannot be pinned", func(t *testing.T) {
		privateID := createAdminTestPost(t, r, authorToken, "pin-private")
		response := doJSON(t, r, http.MethodPost, fmt.Sprintf("/api/posts/visible/%d", privateID), authorToken, map[string]any{
			"visible": model.VisiblePrivate,
		})
		if response.Code != http.StatusOK {
			t.Fatalf("set private status=%d body=%s", response.Code, response.Body.String())
		}

		response = doJSON(t, r, http.MethodPost, fmt.Sprintf("/api/posts/pin/%d", privateID), adminToken, nil)
		assertAdminBizError(t, response, bizerr.ErrPrivatePostCannotPin)
	})

	t.Run("making a post private clears its pin", func(t *testing.T) {
		postID := createAdminTestPost(t, r, authorToken, "pin-then-private")
		response := doJSON(t, r, http.MethodPost, fmt.Sprintf("/api/posts/pin/%d", postID), adminToken, nil)
		if response.Code != http.StatusOK {
			t.Fatalf("pin status=%d body=%s", response.Code, response.Body.String())
		}
		response = doJSON(t, r, http.MethodPost, fmt.Sprintf("/api/posts/visible/%d", postID), authorToken, map[string]any{
			"visible": model.VisiblePrivate,
		})
		if response.Code != http.StatusOK {
			t.Fatalf("set private status=%d body=%s", response.Code, response.Body.String())
		}

		response = doJSON(t, r, http.MethodGet, fmt.Sprintf("/api/posts/%d", postID), authorToken, nil)
		if response.Code != http.StatusOK {
			t.Fatalf("author detail status=%d body=%s", response.Code, response.Body.String())
		}
		post := decodeResp(t, response)["data"].(map[string]any)["post"].(map[string]any)
		assertAdminPinState(t, post, false)
	})

	t.Run("expired pin is normalized and does not affect order", func(t *testing.T) {
		expiredID := createAdminTestPost(t, r, authorToken, "pin-expired")
		latestID := createAdminTestPost(t, r, authorToken, "pin-after-expired")
		past := time.Now().Add(-time.Hour)
		postRepo := gormrepo.NewPostRepository(db)
		if err := postRepo.SetPostPinnedUntil(context.Background(), expiredID, &past); err != nil {
			t.Fatalf("store expired pin through repository: %v", err)
		}

		listed := listAdminTestPosts(t, r)
		expiredIndex := indexAdminTestPost(t, listed, expiredID)
		latestIndex := indexAdminTestPost(t, listed, latestID)
		if latestIndex >= expiredIndex {
			t.Fatalf("expired post must use chronological order: latest index=%d expired index=%d", latestIndex, expiredIndex)
		}
		assertAdminPinState(t, findAdminTestPost(t, listed, expiredID), false)
	})
}

func registerAndLoginAdminTestUser(t *testing.T, r http.Handler, username string) string {
	t.Helper()
	inviterToken := loginAdminTestUser(t, r, adminTestUsername, "password1")
	return registerAndLoginAdminTestUserWithInviter(t, r, inviterToken, username)
}

func registerAndLoginAdminTestUserWithInviter(
	t *testing.T,
	r http.Handler,
	inviterToken, username string,
) string {
	t.Helper()
	w := registerAPIUser(t, r, inviterToken, username, "password1", username)
	if w.Code != http.StatusOK {
		t.Fatalf("register %s status=%d body=%s", username, w.Code, w.Body.String())
	}
	return loginAdminTestUser(t, r, username, "password1")
}

func loginAdminTestUser(t *testing.T, r http.Handler, username, password string) string {
	t.Helper()
	w := doJSON(t, r, http.MethodPost, "/api/login", "", map[string]string{
		"username": username,
		"password": password,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("login %s status=%d body=%s", username, w.Code, w.Body.String())
	}
	data, _ := decodeResp(t, w)["data"].(map[string]any)
	token, _ := data["token"].(string)
	if token == "" {
		t.Fatalf("login %s returned empty token: %v", username, data)
	}
	return token
}

func assertAdminIdentityResponse(t *testing.T, r http.Handler, token string, wantAdmin bool) {
	t.Helper()
	w := doJSON(t, r, http.MethodGet, "/api/user/me", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("get current user status=%d body=%s", w.Code, w.Body.String())
	}
	user := decodeResp(t, w)["data"].(map[string]any)["user"].(map[string]any)
	isAdmin, ok := user["is_admin"].(bool)
	if !ok || isAdmin != wantAdmin {
		t.Fatalf("is_admin=%v, want=%v user=%v", user["is_admin"], wantAdmin, user)
	}
}

func adminTestCurrentUserID(t *testing.T, r http.Handler, token string) uint {
	t.Helper()
	w := doJSON(t, r, http.MethodGet, "/api/user/me", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("get current user status=%d body=%s", w.Code, w.Body.String())
	}
	user := decodeResp(t, w)["data"].(map[string]any)["user"].(map[string]any)
	id, ok := user["id"].(float64)
	if !ok || id <= 0 {
		t.Fatalf("invalid current user id: %v", user)
	}
	return uint(id)
}

func createAdminTestPost(t *testing.T, r http.Handler, token, title string) uint {
	t.Helper()
	w := doJSON(t, r, http.MethodPost, "/api/posts/create", token, map[string]string{
		"title":   title,
		"content": "content for " + title,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("create %q status=%d body=%s", title, w.Code, w.Body.String())
	}
	posts := listAdminTestPosts(t, r)
	for _, post := range posts {
		if post["title"] == title {
			return adminTestPostID(t, post)
		}
	}
	t.Fatalf("created post %q not found in public list", title)
	return 0
}

func listAdminTestPosts(t *testing.T, r http.Handler) []map[string]any {
	t.Helper()
	w := doJSON(t, r, http.MethodGet, "/api/posts?pageNo=1&pageSize=50", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list posts status=%d body=%s", w.Code, w.Body.String())
	}
	rawList, _ := decodeResp(t, w)["data"].(map[string]any)["list"].([]any)
	posts := make([]map[string]any, 0, len(rawList))
	for _, raw := range rawList {
		post, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("unexpected post response: %T %v", raw, raw)
		}
		posts = append(posts, post)
	}
	return posts
}

func findAdminTestPost(t *testing.T, posts []map[string]any, id uint) map[string]any {
	t.Helper()
	for _, post := range posts {
		if adminTestPostID(t, post) == id {
			return post
		}
	}
	t.Fatalf("post %d not found in list", id)
	return nil
}

func indexAdminTestPost(t *testing.T, posts []map[string]any, id uint) int {
	t.Helper()
	for i, post := range posts {
		if adminTestPostID(t, post) == id {
			return i
		}
	}
	t.Fatalf("post %d not found in list", id)
	return -1
}

func adminTestPostID(t *testing.T, post map[string]any) uint {
	t.Helper()
	id, ok := post["id"].(float64)
	if !ok || id <= 0 {
		t.Fatalf("invalid post id: %v", post)
	}
	return uint(id)
}

func assertAdminPinState(t *testing.T, post map[string]any, wantPinned bool) {
	t.Helper()
	isPinned, ok := post["is_pinned"].(bool)
	if !ok || isPinned != wantPinned {
		t.Fatalf("is_pinned=%v, want=%v response=%v", post["is_pinned"], wantPinned, post)
	}
	pinnedUntil, exists := post["pinned_until"]
	if !exists {
		t.Fatalf("response is missing pinned_until: %v", post)
	}
	if wantPinned {
		parseAdminPinnedUntil(t, pinnedUntil)
		return
	}
	if pinnedUntil != nil {
		t.Fatalf("inactive pin must expose pinned_until=null, got %v", pinnedUntil)
	}
}

func parseAdminPinnedUntil(t *testing.T, raw any) time.Time {
	t.Helper()
	value, ok := raw.(string)
	if !ok || value == "" {
		t.Fatalf("invalid pinned_until: %T %v", raw, raw)
	}
	pinnedUntil, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		t.Fatalf("parse pinned_until %q: %v", value, err)
	}
	return pinnedUntil
}

func assertAdminBizError(t *testing.T, w *httptest.ResponseRecorder, want *bizerr.Error) {
	t.Helper()
	if w.Code != want.HTTPStatus {
		t.Fatalf("status=%d, want=%d body=%s", w.Code, want.HTTPStatus, w.Body.String())
	}
	body := decodeResp(t, w)
	if code, ok := body["code"].(float64); !ok || int(code) != want.Code {
		t.Fatalf("code=%v, want=%d body=%v", body["code"], want.Code, body)
	}
}
