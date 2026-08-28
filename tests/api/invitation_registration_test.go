package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"

	"my-bbs/internal/model"
	"my-bbs/internal/repository/gormrepo"
	"my-bbs/pkg/bizerr"
)

var invitationCodePattern = regexp.MustCompile(`^[A-Z0-9]{6}$`)

func TestAPI_InvitationRegistrationLifecycle(t *testing.T) {
	r, db := setupTestRouter(t)

	t.Run("generation requires authentication", func(t *testing.T) {
		w := doJSON(t, r, http.MethodPost, "/api/invitations", "", nil)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status=%d, want=401 body=%s", w.Code, w.Body.String())
		}
		body := decodeResp(t, w)
		if got := int(body["code"].(float64)); got != bizerr.ErrTokenMissing.Code {
			t.Fatalf("code=%d, want=%d body=%v", got, bizerr.ErrTokenMissing.Code, body)
		}
	})

	adminToken := loginAPIUser(t, r, "admin", "password1")
	adminRepo := gormrepo.NewUserRepository(db)
	admin, err := adminRepo.FindUserByUsername(t.Context(), "admin")
	if err != nil || admin == nil {
		t.Fatalf("find seeded inviter: user=%v err=%v", admin, err)
	}
	if admin.InviteCode != nil {
		t.Fatalf("historical user invite_code=%v, want NULL", admin.InviteCode)
	}

	t.Run("invalid invitation does not reveal an existing username", func(t *testing.T) {
		w := doJSON(t, r, http.MethodPost, "/api/register", "", map[string]string{
			"username":    "admin",
			"password":    "password1",
			"nickname":    "Duplicate",
			"invite_code": "NOPE00",
		})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status=%d, want=400 body=%s", w.Code, w.Body.String())
		}
		body := decodeResp(t, w)
		if got := int(body["code"].(float64)); got != bizerr.ErrInvitationUnavailable.Code {
			t.Fatalf("code=%d, want=%d body=%v", got, bizerr.ErrInvitationUnavailable.Code, body)
		}
	})

	generated := doJSON(t, r, http.MethodPost, "/api/invitations", adminToken, nil)
	if generated.Code != http.StatusOK {
		t.Fatalf("generate status=%d body=%s", generated.Code, generated.Body.String())
	}
	generatedData, _ := decodeResp(t, generated)["data"].(map[string]any)
	code, _ := generatedData["code"].(string)
	if !invitationCodePattern.MatchString(code) {
		t.Fatalf("generated code=%q, want six uppercase letters or digits", code)
	}
	if len(generatedData) != 1 {
		t.Fatalf("generation data must expose only this newly generated code: %v", generatedData)
	}

	t.Run("registration requires an invitation", func(t *testing.T) {
		w := doJSON(t, r, http.MethodPost, "/api/register", "", map[string]string{
			"username": "missinginvite",
			"password": "password1",
			"nickname": "Missing",
		})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status=%d, want=400 body=%s", w.Code, w.Body.String())
		}
		body := decodeResp(t, w)
		if got := int(body["code"].(float64)); got != bizerr.ErrBadRequest.Code {
			t.Fatalf("code=%d, want=%d body=%v", got, bizerr.ErrBadRequest.Code, body)
		}
		if got, _ := body["message"].(string); got != "字段 invite_code 不能为空" {
			t.Fatalf("message=%q, want invite_code required error", got)
		}
	})

	registerResponse := doJSON(t, r, http.MethodPost, "/api/register", "", map[string]string{
		"username":    "inviteduser",
		"password":    "password1",
		"nickname":    "Invited",
		"invite_code": code,
	})
	if registerResponse.Code != http.StatusOK {
		t.Fatalf("register status=%d body=%s", registerResponse.Code, registerResponse.Body.String())
	}
	if bytes.Contains(registerResponse.Body.Bytes(), []byte(code)) {
		t.Fatalf("registration response must not repeat invitation code: %s", registerResponse.Body.String())
	}

	invited, err := adminRepo.FindUserByUsername(t.Context(), "inviteduser")
	if err != nil || invited == nil {
		t.Fatalf("find invited user: user=%v err=%v", invited, err)
	}
	if invited.InviteCode == nil || *invited.InviteCode != code {
		t.Fatalf("user invite_code=%v, want=%q", invited.InviteCode, code)
	}

	var invitation model.Invitation
	if err := db.WithContext(t.Context()).Where("code = ?", code).First(&invitation).Error; err != nil {
		t.Fatalf("find invitation row: %v", err)
	}
	if invitation.Code != code {
		t.Fatalf("stored code=%q, want plaintext %q", invitation.Code, code)
	}
	if invitation.CreatorID != admin.ID {
		t.Fatalf("creator_id=%d, want=%d", invitation.CreatorID, admin.ID)
	}
	if invitation.UsedBy == nil || *invitation.UsedBy != invited.ID {
		t.Fatalf("used_by=%v, want=%d", invitation.UsedBy, invited.ID)
	}
	if invitation.UsedAt == nil {
		t.Fatal("used_at must be set after successful registration")
	}

	t.Run("a consumed code cannot register another user", func(t *testing.T) {
		w := doJSON(t, r, http.MethodPost, "/api/register", "", map[string]string{
			"username":    "reuseattempt",
			"password":    "password1",
			"nickname":    "Reuse",
			"invite_code": code,
		})
		if w.Code != bizerr.ErrInvitationUnavailable.HTTPStatus {
			t.Fatalf("status=%d, want=%d body=%s", w.Code, bizerr.ErrInvitationUnavailable.HTTPStatus, w.Body.String())
		}
		body := decodeResp(t, w)
		if got := int(body["code"].(float64)); got != bizerr.ErrInvitationUnavailable.Code {
			t.Fatalf("code=%d, want=%d body=%v", got, bizerr.ErrInvitationUnavailable.Code, body)
		}
	})

	invitedToken := loginAPIUser(t, r, "inviteduser", "password1")
	var invitationsBefore int64
	if err := db.Model(&model.Invitation{}).Count(&invitationsBefore).Error; err != nil {
		t.Fatalf("count invitations before eligibility check: %v", err)
	}
	restrictedGeneration := doJSON(t, r, http.MethodPost, "/api/invitations", invitedToken, nil)
	if restrictedGeneration.Code != bizerr.ErrInvitationGenerationRestricted.HTTPStatus {
		t.Fatalf("new user generation status=%d, want=%d body=%s", restrictedGeneration.Code, bizerr.ErrInvitationGenerationRestricted.HTTPStatus, restrictedGeneration.Body.String())
	}
	restrictedBody := decodeResp(t, restrictedGeneration)
	if got := int(restrictedBody["code"].(float64)); got != bizerr.ErrInvitationGenerationRestricted.Code {
		t.Fatalf("new user generation code=%d, want=%d body=%v", got, bizerr.ErrInvitationGenerationRestricted.Code, restrictedBody)
	}
	var invitationsAfter int64
	if err := db.Model(&model.Invitation{}).Count(&invitationsAfter).Error; err != nil {
		t.Fatalf("count invitations after eligibility check: %v", err)
	}
	if invitationsAfter != invitationsBefore {
		t.Fatalf("restricted generation created invitation: before=%d after=%d", invitationsBefore, invitationsAfter)
	}

	invalidPost := doJSON(t, r, http.MethodPost, "/api/posts/create", invitedToken, map[string]string{
		"title":   "",
		"content": "not published",
	})
	if invalidPost.Code != http.StatusBadRequest {
		t.Fatalf("invalid post status=%d, want=400 body=%s", invalidPost.Code, invalidPost.Body.String())
	}
	stillRestricted := doJSON(t, r, http.MethodPost, "/api/invitations", invitedToken, nil)
	if stillRestricted.Code != bizerr.ErrInvitationGenerationRestricted.HTTPStatus {
		t.Fatalf("failed post unlocked invitation: status=%d body=%s", stillRestricted.Code, stillRestricted.Body.String())
	}

	publishedPost := doJSON(t, r, http.MethodPost, "/api/posts/create", invitedToken, map[string]string{
		"title":   "My first post",
		"content": "Publishing unlocks invitation generation.",
	})
	if publishedPost.Code != http.StatusOK {
		t.Fatalf("publish first post status=%d body=%s", publishedPost.Code, publishedPost.Body.String())
	}
	secondGeneration := doJSON(t, r, http.MethodPost, "/api/invitations", invitedToken, nil)
	if secondGeneration.Code != http.StatusOK {
		t.Fatalf("published new user generate status=%d body=%s", secondGeneration.Code, secondGeneration.Body.String())
	}
	secondCode, _ := decodeResp(t, secondGeneration)["data"].(map[string]any)["code"].(string)
	if !invitationCodePattern.MatchString(secondCode) || secondCode == code {
		t.Fatalf("second generated code=%q, first=%q", secondCode, code)
	}

	t.Run("there is no invitation history endpoint or profile disclosure", func(t *testing.T) {
		history := doJSON(t, r, http.MethodGet, "/api/invitations", invitedToken, nil)
		if history.Code != http.StatusMethodNotAllowed {
			t.Fatalf("GET invitations status=%d, want=405 body=%s", history.Code, history.Body.String())
		}
		if bytes.Contains(history.Body.Bytes(), []byte(secondCode)) {
			t.Fatalf("history response disclosed generated code: %s", history.Body.String())
		}

		me := doJSON(t, r, http.MethodGet, "/api/user/me", invitedToken, nil)
		if me.Code != http.StatusOK {
			t.Fatalf("get me status=%d body=%s", me.Code, me.Body.String())
		}
		meUser := decodeResp(t, me)["data"].(map[string]any)["user"].(map[string]any)
		if _, exists := meUser["invite_code"]; exists {
			t.Fatalf("current-user response disclosed invite_code: %v", meUser)
		}

		profile := doJSON(t, r, http.MethodGet, fmt.Sprintf("/api/users/%d", invited.ID), "", nil)
		if profile.Code != http.StatusOK {
			t.Fatalf("public profile status=%d body=%s", profile.Code, profile.Body.String())
		}
		profileUser := decodeResp(t, profile)["data"].(map[string]any)["user"].(map[string]any)
		if _, exists := profileUser["invite_code"]; exists {
			t.Fatalf("public profile disclosed invite_code: %v", profileUser)
		}
	})
}

func TestAPI_InvitationCodeIsSingleUseUnderConcurrentRegistration(t *testing.T) {
	r, db := setupTestRouter(t)
	adminToken := loginAPIUser(t, r, "admin", "password1")
	code := generateAPIInvitation(t, r, adminToken)

	const attempts = 4
	start := make(chan struct{})
	results := make(chan int, attempts)
	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			payload, err := json.Marshal(map[string]string{
				"username":    fmt.Sprintf("concurrent%d", index),
				"password":    "password1",
				"nickname":    "Concurrent",
				"invite_code": code,
			})
			if err != nil {
				results <- 0
				return
			}
			<-start
			req := httptest.NewRequest(http.MethodPost, "/api/register", bytes.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			results <- w.Code
		}(i)
	}
	close(start)
	wg.Wait()
	close(results)

	successes := 0
	for status := range results {
		if status == http.StatusOK {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("successful registrations=%d, want exactly one", successes)
	}

	var registered int64
	if err := db.WithContext(t.Context()).Model(&model.User{}).
		Where("invite_code = ?", code).
		Count(&registered).Error; err != nil {
		t.Fatalf("count users registered by invitation: %v", err)
	}
	if registered != 1 {
		t.Fatalf("persisted users with invite_code=%q: %d, want=1", code, registered)
	}

	var invitation model.Invitation
	if err := db.WithContext(t.Context()).Where("code = ?", code).First(&invitation).Error; err != nil {
		t.Fatalf("find consumed invitation: %v", err)
	}
	if invitation.UsedBy == nil || invitation.UsedAt == nil {
		t.Fatalf("consumed invitation missing usage metadata: %+v", invitation)
	}
}
