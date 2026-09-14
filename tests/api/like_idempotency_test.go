package api_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"my-bbs/pkg/bizerr"
)

func TestLikeAPI_SetsTargetStateIdempotently(t *testing.T) {
	r, _ := setupTestRouter(t)
	token := loginAPIUser(t, r, "admin", "password1")

	created := doJSON(t, r, http.MethodPost, "/api/posts/create", token, map[string]string{
		"title":   "idempotent like",
		"content": "body",
	})
	if created.Code != http.StatusOK {
		t.Fatalf("create post status=%d body=%s", created.Code, created.Body.String())
	}

	listed := doJSON(t, r, http.MethodGet, "/api/posts", "", nil)
	if listed.Code != http.StatusOK {
		t.Fatalf("list posts status=%d body=%s", listed.Code, listed.Body.String())
	}
	posts := decodeResp(t, listed)["data"].(map[string]any)["list"].([]any)
	if len(posts) != 1 {
		t.Fatalf("post count=%d, want 1", len(posts))
	}
	postID := uint(posts[0].(map[string]any)["id"].(float64))
	path := fmt.Sprintf("/api/posts/%d/like", postID)

	for attempt := 1; attempt <= 2; attempt++ {
		response := doJSON(t, r, http.MethodPut, path, token, nil)
		assertLikeState(t, response, true, 1, fmt.Sprintf("PUT attempt %d", attempt))
	}

	for attempt := 1; attempt <= 2; attempt++ {
		response := doJSON(t, r, http.MethodDelete, path, token, nil)
		assertLikeState(t, response, false, 0, fmt.Sprintf("DELETE attempt %d", attempt))
	}

	legacyToggle := doJSON(t, r, http.MethodPost, path, token, nil)
	if legacyToggle.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST status=%d body=%s, want 405", legacyToggle.Code, legacyToggle.Body.String())
	}
	if got := int(decodeResp(t, legacyToggle)["code"].(float64)); got != bizerr.ErrMethodNotAllowed.Code {
		t.Fatalf("POST error code=%d, want %d", got, bizerr.ErrMethodNotAllowed.Code)
	}
}

func assertLikeState(t *testing.T, response *httptest.ResponseRecorder, liked bool, likeCount int, operation string) {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("%s status=%d body=%s, want 200", operation, response.Code, response.Body.String())
	}
	data := decodeResp(t, response)["data"].(map[string]any)
	if got, _ := data["liked"].(bool); got != liked {
		t.Fatalf("%s liked=%v, want %v", operation, got, liked)
	}
	if got := int(data["like_count"].(float64)); got != likeCount {
		t.Fatalf("%s like_count=%d, want %d", operation, got, likeCount)
	}
}
