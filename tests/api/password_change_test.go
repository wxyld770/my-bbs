package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"my-bbs/internal/model"
	"my-bbs/internal/repository/gormrepo"
	"my-bbs/pkg/bcrypt"
	"my-bbs/pkg/bizerr"
)

const changePasswordPath = "/api/user/password"

func TestAPI_ChangePasswordInvalidatesEveryOldSessionAndRequiresNewPassword(t *testing.T) {
	r, db := setupTestRouter(t)
	registerPasswordChangeUser(t, r, "password-owner")
	firstToken := loginAPIUser(t, r, "password-owner", "password1")
	secondToken := loginAPIUser(t, r, "password-owner", "password1")

	w := changeAPIPassword(t, r, firstToken, "password1", "new-password2")
	if w.Code != http.StatusOK {
		t.Fatalf("change password status=%d body=%s", w.Code, w.Body.String())
	}
	body := decodeResp(t, w)
	if body["code"] != float64(0) || body["message"] != "密码修改成功，请重新登录" {
		t.Fatalf("unexpected change-password response: %v", body)
	}
	if _, exists := body["data"]; exists {
		t.Fatalf("change-password response must not return a replacement token: %v", body)
	}
	if strings.Contains(w.Body.String(), "password1") || strings.Contains(w.Body.String(), "new-password2") || strings.Contains(w.Body.String(), "$2") {
		t.Fatalf("change-password response exposed credential material: %s", w.Body.String())
	}

	stored, err := gormrepo.NewUserRepository(db).FindUserByUsername(context.Background(), "password-owner")
	if err != nil || stored == nil {
		t.Fatalf("load changed user: user=%+v err=%v", stored, err)
	}
	if stored.SessionVersion != 1 {
		t.Fatalf("session_version=%d, want 1", stored.SessionVersion)
	}
	if !bcrypt.CheckPassword("new-password2", stored.Password) || bcrypt.CheckPassword("password1", stored.Password) {
		t.Fatal("repository did not atomically replace the password hash")
	}

	// The token used for the change and another independently issued token must
	// both become invalid after the single session-version increment.
	for name, token := range map[string]string{"request token": firstToken, "other session": secondToken} {
		t.Run(name+" is invalid", func(t *testing.T) {
			response := doJSON(t, r, http.MethodGet, "/api/user/me", token, nil)
			assertAdminBizError(t, response, bizerr.ErrInvalidToken)
		})
	}

	oldLogin := doJSON(t, r, http.MethodPost, "/api/login", "", map[string]string{
		"username": "password-owner",
		"password": "password1",
	})
	assertAdminBizError(t, oldLogin, bizerr.ErrLoginFailed)

	newToken := loginAPIUser(t, r, "password-owner", "new-password2")
	me := doJSON(t, r, http.MethodGet, "/api/user/me", newToken, nil)
	if me.Code != http.StatusOK {
		t.Fatalf("new session status=%d body=%s", me.Code, me.Body.String())
	}
}

func TestAPI_ChangePasswordRejectsWrongInvalidAndUnchangedPasswordsWithoutMutation(t *testing.T) {
	r, db := setupTestRouter(t)
	registerPasswordChangeUser(t, r, "password-rules")
	token := loginAPIUser(t, r, "password-rules", "password1")
	anonymous := changeAPIPassword(t, r, "", "password1", "valid-password2")
	assertAdminBizError(t, anonymous, bizerr.ErrTokenMissing)

	tests := []struct {
		name        string
		oldPassword string
		newPassword string
		wantStatus  int
		wantCode    int
	}{
		{name: "wrong old password", oldPassword: "wrong-password", newPassword: "valid-password2", wantStatus: http.StatusBadRequest, wantCode: 40009},
		{name: "same password", oldPassword: "password1", newPassword: "password1", wantStatus: http.StatusBadRequest, wantCode: 40010},
		{name: "new password too short", oldPassword: "password1", newPassword: "short", wantStatus: http.StatusBadRequest, wantCode: bizerr.ErrBadRequest.Code},
		{name: "new password too many characters", oldPassword: "password1", newPassword: strings.Repeat("a", 65), wantStatus: http.StatusBadRequest, wantCode: bizerr.ErrBadRequest.Code},
		{name: "new password exceeds bcrypt byte limit", oldPassword: "password1", newPassword: strings.Repeat("界", 25), wantStatus: http.StatusBadRequest, wantCode: bizerr.ErrBadRequest.Code},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := changeAPIPassword(t, r, token, tt.oldPassword, tt.newPassword)
			if w.Code != tt.wantStatus {
				t.Fatalf("status=%d, want=%d body=%s", w.Code, tt.wantStatus, w.Body.String())
			}
			body := decodeResp(t, w)
			if code, ok := body["code"].(float64); !ok || int(code) != tt.wantCode {
				t.Fatalf("code=%v, want=%d body=%v", body["code"], tt.wantCode, body)
			}

			stored, err := gormrepo.NewUserRepository(db).FindUserByUsername(context.Background(), "password-rules")
			if err != nil || stored == nil {
				t.Fatalf("load unchanged user: user=%+v err=%v", stored, err)
			}
			if stored.SessionVersion != 0 || !bcrypt.CheckPassword("password1", stored.Password) {
				t.Fatalf("rejected change mutated credentials: version=%d hash=%q", stored.SessionVersion, stored.Password)
			}
		})
	}

	// A rejected password change must not invalidate the otherwise valid session.
	me := doJSON(t, r, http.MethodGet, "/api/user/me", token, nil)
	if me.Code != http.StatusOK {
		t.Fatalf("valid token was invalidated by rejected change: status=%d body=%s", me.Code, me.Body.String())
	}
}

func TestAPI_MutedUserCanChangePasswordButRemainsReadOnly(t *testing.T) {
	r, db := setupTestRouter(t)
	registerPasswordChangeUser(t, r, "muted-password")
	token := loginAPIUser(t, r, "muted-password", "password1")
	userRepo := gormrepo.NewUserRepository(db)
	user, err := userRepo.FindUserByUsername(context.Background(), "muted-password")
	if err != nil || user == nil {
		t.Fatalf("load muted password user: user=%+v err=%v", user, err)
	}
	if err := userRepo.UpdateUserStatus(context.Background(), user.ID, model.UserStatusMuted); err != nil {
		t.Fatalf("mute password user: %v", err)
	}

	w := changeAPIPassword(t, r, token, "password1", "muted-password2")
	if w.Code != http.StatusOK {
		t.Fatalf("muted password change status=%d body=%s", w.Code, w.Body.String())
	}

	newToken := loginAPIUser(t, r, "muted-password", "muted-password2")
	me := doJSON(t, r, http.MethodGet, "/api/user/me", newToken, nil)
	if me.Code != http.StatusOK {
		t.Fatalf("muted user read after password change status=%d body=%s", me.Code, me.Body.String())
	}
	write := doJSON(t, r, http.MethodPost, "/api/user/profile", newToken, map[string]string{
		"nickname":     "must not change",
		"introduction": "must remain read-only",
	})
	assertAdminBizError(t, write, bizerr.ErrUserMuted)
}

func registerPasswordChangeUser(t *testing.T, r http.Handler, username string) {
	t.Helper()
	w := registerDefaultAPIUser(t, r, username, username)
	if w.Code != http.StatusOK {
		t.Fatalf("register %s status=%d body=%s", username, w.Code, w.Body.String())
	}
}

func changeAPIPassword(t *testing.T, r http.Handler, token, oldPassword, newPassword string) *httptest.ResponseRecorder {
	t.Helper()
	return doJSON(t, r, http.MethodPost, changePasswordPath, token, map[string]string{
		"old_password": oldPassword,
		"new_password": newPassword,
	})
}
