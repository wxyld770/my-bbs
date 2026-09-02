package api_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"my-bbs/internal/authorization"
	"my-bbs/internal/model"
	messageModule "my-bbs/internal/modules/message"
	"my-bbs/internal/repository/gormrepo"
	"my-bbs/internal/router"
	"my-bbs/pkg/bizerr"
	"my-bbs/pkg/jwt"
	"my-bbs/tests/testutil"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type messageTestUser struct {
	ID    uint
	Token string
}

func setupMessageTestRouter(t *testing.T) (*gin.Engine, *gorm.DB, map[string]messageTestUser) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	testutil.InitJWT(t)
	db := testutil.NewTestDB(t)
	redisClient := testutil.NewTestRedis(t)
	admins := authorization.NewAdminUsers("admin")
	users := gormrepo.NewUserRepository(db)

	result := make(map[string]messageTestUser, 3)
	for _, username := range []string{"admin", "alice", "bob"} {
		user := &model.User{Username: username, Password: "not-used", Nickname: username, Status: model.UserStatusNormal}
		if err := users.CreateUser(context.Background(), user); err != nil {
			t.Fatalf("create test user %s: %v", username, err)
		}
		token, err := jwt.GenerateToken(user.ID)
		if err != nil {
			t.Fatalf("generate token for %s: %v", username, err)
		}
		result[username] = messageTestUser{ID: user.ID, Token: token}
	}

	r := router.SetupRouter(router.RouterDeps{
		Modules: []router.RouteRegister{messageModule.Initialize(db, redisClient, admins)},
	})
	return r, db, result
}

func TestAPI_MessageLifecycleKeepsUserHistoryPrivateAndAllowsMutedAppeals(t *testing.T) {
	r, db, users := setupMessageTestRouter(t)

	first := doJSON(t, r, http.MethodPost, "/api/messages", users["alice"].Token, map[string]string{
		"content": "  第一条留言  ",
	})
	if first.Code != http.StatusOK {
		t.Fatalf("create first message status=%d body=%s", first.Code, first.Body.String())
	}

	if err := gormrepo.NewUserRepository(db).UpdateUserStatus(
		context.Background(),
		users["alice"].ID,
		model.UserStatusMuted,
	); err != nil {
		t.Fatalf("mute alice: %v", err)
	}
	mutedAppeal := doJSON(t, r, http.MethodPost, "/api/messages", users["alice"].Token, map[string]string{
		"content": "申请解除禁言",
	})
	if mutedAppeal.Code != http.StatusOK {
		t.Fatalf("muted appeal status=%d body=%s", mutedAppeal.Code, mutedAppeal.Body.String())
	}

	bobMessage := doJSON(t, r, http.MethodPost, "/api/messages", users["bob"].Token, map[string]string{
		"content": "Bob 的私有留言",
	})
	if bobMessage.Code != http.StatusOK {
		t.Fatalf("create bob message status=%d body=%s", bobMessage.Code, bobMessage.Body.String())
	}

	alicePage1 := doJSON(t, r, http.MethodGet, "/api/messages?pageNo=1&pageSize=1", users["alice"].Token, nil)
	if alicePage1.Code != http.StatusOK {
		t.Fatalf("list alice page 1 status=%d body=%s", alicePage1.Code, alicePage1.Body.String())
	}
	page1 := decodeResp(t, alicePage1)["data"].(map[string]any)
	list1 := page1["list"].([]any)
	if len(list1) != 1 || page1["hasMore"] != true {
		t.Fatalf("alice page 1=%v, want one item and hasMore=true", page1)
	}
	item1 := list1[0].(map[string]any)
	if item1["content"] != "申请解除禁言" || uint(item1["user_id"].(float64)) != users["alice"].ID {
		t.Fatalf("alice newest message=%v", item1)
	}
	if item1["user"].(map[string]any)["username"] != "alice" {
		t.Fatalf("alice public user summary=%v", item1["user"])
	}

	alicePage2 := doJSON(t, r, http.MethodGet, "/api/messages?pageNo=2&pageSize=1", users["alice"].Token, nil)
	page2 := decodeResp(t, alicePage2)["data"].(map[string]any)
	list2 := page2["list"].([]any)
	if len(list2) != 1 || page2["hasMore"] != false {
		t.Fatalf("alice page 2=%v, want exact final page", page2)
	}
	if content := list2[0].(map[string]any)["content"]; content != "第一条留言" {
		t.Fatalf("stored trimmed content=%v", content)
	}

	adminList := doJSON(t, r, http.MethodGet, "/api/admin/messages?pageNo=1&pageSize=10", users["admin"].Token, nil)
	if adminList.Code != http.StatusOK {
		t.Fatalf("admin list status=%d body=%s", adminList.Code, adminList.Body.String())
	}
	adminData := decodeResp(t, adminList)["data"].(map[string]any)
	if got := len(adminData["list"].([]any)); got != 3 {
		t.Fatalf("admin message count=%d, want=3 data=%v", got, adminData)
	}
	if strings.Contains(adminList.Body.String(), "password") || strings.Contains(adminList.Body.String(), "deleted") {
		t.Fatalf("admin message list leaked private persistence fields: %s", adminList.Body.String())
	}

	forbidden := doJSON(t, r, http.MethodGet, "/api/admin/messages", users["bob"].Token, nil)
	if forbidden.Code != bizerr.ErrForbidden.HTTPStatus {
		t.Fatalf("ordinary admin list status=%d body=%s", forbidden.Code, forbidden.Body.String())
	}
	unauthorized := doJSON(t, r, http.MethodGet, "/api/messages", "", nil)
	if unauthorized.Code != bizerr.ErrTokenMissing.HTTPStatus {
		t.Fatalf("anonymous list status=%d body=%s", unauthorized.Code, unauthorized.Body.String())
	}

	if err := gormrepo.NewUserRepository(db).UpdateUserStatus(
		context.Background(),
		users["admin"].ID,
		model.UserStatusMuted,
	); err != nil {
		t.Fatalf("mute admin: %v", err)
	}
	mutedAdmin := doJSON(t, r, http.MethodGet, "/api/admin/messages", users["admin"].Token, nil)
	if mutedAdmin.Code != bizerr.ErrUserMuted.HTTPStatus {
		t.Fatalf("muted admin list status=%d body=%s", mutedAdmin.Code, mutedAdmin.Body.String())
	}
}

func TestAPI_MessageContentContract(t *testing.T) {
	r, _, users := setupMessageTestRouter(t)

	tests := []struct {
		name    string
		body    map[string]string
		wantMsg string
	}{
		{name: "blank", body: map[string]string{"content": " \n\t "}, wantMsg: "留言内容不能为空"},
		{name: "too long", body: map[string]string{"content": strings.Repeat("留", 2001)}, wantMsg: "留言内容长度不能超过2000个字符"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := doJSON(t, r, http.MethodPost, "/api/messages", users["alice"].Token, tt.body)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
			body := decodeResp(t, w)
			if body["message"] != tt.wantMsg {
				t.Fatalf("message=%v, want=%q body=%v", body["message"], tt.wantMsg, body)
			}
		})
	}
}
