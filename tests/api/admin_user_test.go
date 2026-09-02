package api_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"my-bbs/internal/model"
	"my-bbs/internal/repository/gormrepo"
	"my-bbs/pkg/bcrypt"
	"my-bbs/pkg/bizerr"
)

func TestAPI_AdminCanResetOrdinaryUserPasswordToUsername(t *testing.T) {
	r, db := setupTestRouter(t)
	targetToken := registerAndLoginAdminTestUser(t, r, "reset-target")
	ordinaryToken := registerAndLoginAdminTestUser(t, r, "reset-ordinary")
	adminToken := loginAdminTestUser(t, r, adminTestUsername, "password1")
	targetID := adminTestCurrentUserID(t, r, targetToken)
	path := fmt.Sprintf("/api/users/%d/reset-password", targetID)

	w := doJSON(t, r, http.MethodPost, path, ordinaryToken, nil)
	assertAdminBizError(t, w, bizerr.ErrForbidden)

	w = doJSON(t, r, http.MethodPost, path, adminToken, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("admin reset password status=%d body=%s", w.Code, w.Body.String())
	}
	body := decodeResp(t, w)
	if body["code"] != float64(0) || body["message"] != "密码已重置为用户名" {
		t.Fatalf("unexpected reset response: %v", body)
	}
	if _, exists := body["data"]; exists {
		t.Fatalf("reset response must not expose user or password data: %v", body)
	}
	if strings.Contains(w.Body.String(), "reset-target") || strings.Contains(w.Body.String(), "$2") {
		t.Fatalf("reset response exposed username or password hash: %s", w.Body.String())
	}

	stored, err := gormrepo.NewUserRepository(db).FindUserByID(context.Background(), targetID)
	if err != nil || stored == nil {
		t.Fatalf("load reset user: user=%+v err=%v", stored, err)
	}
	if stored.Password == stored.Username || !bcrypt.CheckPassword(stored.Username, stored.Password) {
		t.Fatal("repository must store a bcrypt hash of the username")
	}
	if stored.SessionVersion != 1 {
		t.Fatalf("session version = %d, want 1 after password reset", stored.SessionVersion)
	}

	w = doJSON(t, r, http.MethodGet, "/api/user/me", targetToken, nil)
	assertAdminBizError(t, w, bizerr.ErrInvalidToken)

	w = doJSON(t, r, http.MethodPost, "/api/login", "", map[string]string{
		"username": "reset-target",
		"password": "password1",
	})
	assertAdminBizError(t, w, bizerr.ErrLoginFailed)
	if token := loginAdminTestUser(t, r, "reset-target", "reset-target"); token == "" {
		t.Fatal("user must be able to log in with the reset password")
	}
}

func TestAPI_AdminPasswordResetPermissionAndTargetErrors(t *testing.T) {
	t.Run("configured admin accounts are protected including self", func(t *testing.T) {
		r, _ := setupTestRouterWithAdminUsers(t, "admin", "moderator")
		adminToken := loginAdminTestUser(t, r, "admin", "password1")
		moderatorToken := loginAdminTestUser(t, r, "moderator", "password1")
		moderatorID := adminTestCurrentUserID(t, r, moderatorToken)
		adminID := adminTestCurrentUserID(t, r, adminToken)

		w := doJSON(t, r, http.MethodPost, fmt.Sprintf("/api/users/%d/reset-password", moderatorID), adminToken, nil)
		assertAdminBizError(t, w, bizerr.ErrAdminCannotManageAdmin)

		w = doJSON(t, r, http.MethodPost, fmt.Sprintf("/api/users/%d/reset-password", adminID), adminToken, nil)
		assertAdminBizError(t, w, bizerr.ErrAdminCannotManageAdmin)

		if token := loginAdminTestUser(t, r, "admin", "password1"); token == "" {
			t.Fatal("rejected self reset must preserve the administrator password")
		}
	})

	t.Run("missing and invalid target", func(t *testing.T) {
		r, _ := setupTestRouter(t)
		adminToken := loginAdminTestUser(t, r, adminTestUsername, "password1")

		w := doJSON(t, r, http.MethodPost, "/api/users/999999/reset-password", adminToken, nil)
		assertAdminBizError(t, w, bizerr.ErrUserNotFound)
		w = doJSON(t, r, http.MethodPost, "/api/users/not-a-number/reset-password", adminToken, nil)
		assertAdminBizError(t, w, bizerr.ErrInvalidUserID)
	})

	t.Run("muted ordinary target remains resettable", func(t *testing.T) {
		r, db := setupTestRouter(t)
		adminToken := loginAdminTestUser(t, r, adminTestUsername, "password1")
		targetToken := registerAndLoginAdminTestUser(t, r, "muted-reset")
		targetID := adminTestCurrentUserID(t, r, targetToken)
		if err := db.Model(&model.User{}).Where("id = ?", targetID).Update("status", model.UserStatusMuted).Error; err != nil {
			t.Fatalf("mute reset target: %v", err)
		}

		w := doJSON(t, r, http.MethodPost, fmt.Sprintf("/api/users/%d/reset-password", targetID), adminToken, nil)
		if w.Code != http.StatusOK {
			t.Fatalf("reset muted target status=%d body=%s", w.Code, w.Body.String())
		}
		if token := loginAdminTestUser(t, r, "muted-reset", "muted-reset"); token == "" {
			t.Fatal("muted target must be able to authenticate with reset password")
		}
	})
}

func TestAPI_AdminPasswordResetRejectsUsernameBeyondBcryptLimit(t *testing.T) {
	r, db := setupTestRouter(t)
	adminToken := loginAdminTestUser(t, r, adminTestUsername, "password1")
	longUsername := strings.Repeat("界", 25) // 25 字符合法，但 UTF-8 编码为 75 字节。
	originalHash, err := bcrypt.HashPassword("password1")
	if err != nil {
		t.Fatalf("hash original password: %v", err)
	}
	target := &model.User{
		Username: longUsername,
		Password: originalHash,
		Nickname: "long username",
		Status:   model.UserStatusNormal,
	}
	if err := gormrepo.NewUserRepository(db).CreateUser(context.Background(), target); err != nil {
		t.Fatalf("seed long username: %v", err)
	}

	w := doJSON(t, r, http.MethodPost, fmt.Sprintf("/api/users/%d/reset-password", target.ID), adminToken, nil)
	assertAdminBizError(t, w, bizerr.ErrConflict)
	if got := decodeResp(t, w)["message"]; got != "用户名超过密码重置支持的72字节限制" {
		t.Fatalf("message=%v", got)
	}
	stored, err := gormrepo.NewUserRepository(db).FindUserByID(context.Background(), target.ID)
	if err != nil || stored == nil || stored.Password != originalHash {
		t.Fatalf("failed reset must preserve password: user=%+v err=%v", stored, err)
	}
}
