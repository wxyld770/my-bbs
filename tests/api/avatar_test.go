package api_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"my-bbs/internal/model"
	"my-bbs/internal/service"
	"my-bbs/pkg/bizerr"
)

func TestAPI_UpdateAvatarLifecycle(t *testing.T) {
	r, db := setupTestRouter(t)
	token := loginAPIUser(t, r, "admin", "password1")

	meResponse := doJSON(t, r, http.MethodGet, "/api/user/me", token, nil)
	if meResponse.Code != http.StatusOK {
		t.Fatalf("get me status=%d body=%s", meResponse.Code, meResponse.Body.String())
	}
	me := decodeResp(t, meResponse)["data"].(map[string]any)["user"].(map[string]any)
	userID := uint(me["id"].(float64))
	if got, _ := me["avatar_url"].(string); got != "" {
		t.Fatalf("initial avatar_url=%q", got)
	}
	if _, exists := me["avatar_updated_at"]; exists {
		t.Fatalf("initial response should omit null avatar_updated_at: %v", me)
	}

	missing := doJSON(t, r, http.MethodPost, "/api/user/avatar", token, map[string]any{})
	if missing.Code != http.StatusBadRequest {
		t.Fatalf("missing avatar_url status=%d body=%s", missing.Code, missing.Body.String())
	}
	invalid := doJSON(t, r, http.MethodPost, "/api/user/avatar", token, map[string]string{
		"avatar_url": "http://example.com/avatar.png",
	})
	if invalid.Code != bizerr.ErrInvalidAvatarURL.HTTPStatus {
		t.Fatalf("invalid avatar status=%d body=%s", invalid.Code, invalid.Body.String())
	}
	if got := int(decodeResp(t, invalid)["code"].(float64)); got != bizerr.ErrInvalidAvatarURL.Code {
		t.Fatalf("invalid avatar code=%d", got)
	}

	avatarURL := "https://cdn.example.com/avatar.png"
	updated := doJSON(t, r, http.MethodPost, "/api/user/avatar", token, map[string]string{
		"avatar_url": avatarURL,
	})
	if updated.Code != http.StatusOK {
		t.Fatalf("first update status=%d body=%s", updated.Code, updated.Body.String())
	}

	meResponse = doJSON(t, r, http.MethodGet, "/api/user/me", token, nil)
	me = decodeResp(t, meResponse)["data"].(map[string]any)["user"].(map[string]any)
	if me["avatar_url"] != avatarURL || me["avatar_updated_at"] == nil {
		t.Fatalf("private profile missing avatar state: %v", me)
	}
	publicResponse := doJSON(t, r, http.MethodGet, fmt.Sprintf("/api/users/%d", userID), "", nil)
	publicUser := decodeResp(t, publicResponse)["data"].(map[string]any)["user"].(map[string]any)
	if publicUser["avatar_url"] != avatarURL {
		t.Fatalf("public profile avatar_url=%v", publicUser["avatar_url"])
	}
	if _, exists := publicUser["avatar_updated_at"]; exists {
		t.Fatalf("public profile leaked avatar_updated_at: %v", publicUser)
	}

	// 网络重试同一值按幂等成功处理。
	same := doJSON(t, r, http.MethodPost, "/api/user/avatar", token, map[string]string{
		"avatar_url": avatarURL,
	})
	if same.Code != http.StatusOK {
		t.Fatalf("same URL retry status=%d body=%s", same.Code, same.Body.String())
	}

	tooSoon := doJSON(t, r, http.MethodPost, "/api/user/avatar", token, map[string]string{
		"avatar_url": "https://cdn.example.com/other.png",
	})
	if tooSoon.Code != bizerr.ErrAvatarUpdateTooFrequent.HTTPStatus {
		t.Fatalf("too-soon status=%d body=%s", tooSoon.Code, tooSoon.Body.String())
	}
	if got := int(decodeResp(t, tooSoon)["code"].(float64)); got != bizerr.ErrAvatarUpdateTooFrequent.Code {
		t.Fatalf("too-soon code=%d", got)
	}

	oldEnough := time.Now().UTC().Add(-service.AvatarUpdateInterval - time.Minute).Truncate(time.Millisecond)
	if err := db.Model(&model.User{}).Where("id = ?", userID).Update("avatar_updated_at", oldEnough).Error; err != nil {
		t.Fatalf("age avatar cooldown: %v", err)
	}
	cleared := doJSON(t, r, http.MethodPost, "/api/user/avatar", token, map[string]string{
		"avatar_url": "",
	})
	if cleared.Code != http.StatusOK {
		t.Fatalf("clear avatar status=%d body=%s", cleared.Code, cleared.Body.String())
	}
	meResponse = doJSON(t, r, http.MethodGet, "/api/user/me", token, nil)
	me = decodeResp(t, meResponse)["data"].(map[string]any)["user"].(map[string]any)
	if got, _ := me["avatar_url"].(string); got != "" {
		t.Fatalf("cleared avatar_url=%q", got)
	}
}
