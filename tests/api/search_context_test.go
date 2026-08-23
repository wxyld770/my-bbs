package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"my-bbs/internal/handler"
	"my-bbs/internal/model"
	searchmodule "my-bbs/internal/modules/search"
	"my-bbs/internal/router"
	"my-bbs/internal/service"
	"my-bbs/pkg/bizerr"

	"github.com/gin-gonic/gin"
)

type deadlineSearchReader struct {
	sawDeadline chan bool
}

func (r *deadlineSearchReader) SearchUsers(ctx context.Context, _ string, _, _ int) ([]model.User, error) {
	_, hasDeadline := ctx.Deadline()
	r.sawDeadline <- hasDeadline
	<-ctx.Done()
	return nil, ctx.Err()
}

func (*deadlineSearchReader) SearchPublicPosts(context.Context, string, int, int) ([]model.Post, error) {
	return []model.Post{}, nil
}

func TestSearchAPI_PropagatesDeadlineAndReturnsServiceUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	reader := &deadlineSearchReader{sawDeadline: make(chan bool, 1)}
	svc := service.NewSearchService(reader, nil, nil, nil)
	mod := &searchmodule.Module{Handler: handler.NewSearchHandler(svc, 10*time.Millisecond)}
	r := router.SetupRouter(router.RouterDeps{Modules: []router.RouteRegister{mod}})

	req := httptest.NewRequest(http.MethodGet, "/api/search?q=go&scope=users", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !<-reader.sawDeadline {
		t.Fatal("repository did not receive the search request deadline")
	}
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d, want=503 body=%s", w.Code, w.Body.String())
	}
	if requestID := w.Header().Get("X-Request-ID"); requestID == "" {
		t.Fatal("deadline response is missing X-Request-ID")
	}
	var body struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode deadline response: %v body=%s", err, w.Body.String())
	}
	if body.Code != bizerr.ErrServiceUnavailable.Code {
		t.Fatalf("code=%d, want=%d body=%s", body.Code, bizerr.ErrServiceUnavailable.Code, w.Body.String())
	}
}
