package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"my-bbs/internal/middleware"
	"my-bbs/internal/router"
	"my-bbs/pkg/bizerr"
	jwtpkg "my-bbs/pkg/jwt"

	"github.com/gin-gonic/gin"
)

type rateLimitRouteRegister func(*gin.RouterGroup)

func (f rateLimitRouteRegister) Register(group *gin.RouterGroup) {
	f(group)
}

func TestTieredRateLimiter_ReturnsUnified429AndSeparatesLoginRegister(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := testRateLimitConfig()
	cfg.Login = middleware.RateLimitPolicy{Requests: 1, Window: time.Hour, Burst: 1}
	cfg.Register = middleware.RateLimitPolicy{Requests: 1, Window: time.Hour, Burst: 1}
	limiter := mustRateLimiter(t, cfg)
	r := newRateLimitRouter(limiter)

	first := performRateLimitRequest(r, http.MethodPost, "/api/login", "203.0.113.10:1234", "", "")
	if first.Code != http.StatusOK {
		t.Fatalf("first login status = %d, want 200; body=%s", first.Code, first.Body.String())
	}
	limited := performRateLimitRequest(r, http.MethodPost, "/api/login", "203.0.113.10:1234", "", "")
	assertRateLimited(t, limited)

	// Registration has an independent, stricter-policy bucket for the same IP.
	registration := performRateLimitRequest(r, http.MethodPost, "/api/register", "203.0.113.10:1234", "", "")
	if registration.Code != http.StatusOK {
		t.Fatalf("registration status = %d, want 200; body=%s", registration.Code, registration.Body.String())
	}
}

func TestTieredRateLimiter_WriteRequestsUseSignedUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtpkg.Init("rate-limit-user-identity-test-secret")
	cfg := testRateLimitConfig()
	cfg.Write = middleware.RateLimitPolicy{Requests: 1, Window: time.Hour, Burst: 1}
	limiter := mustRateLimiter(t, cfg)
	r := newRateLimitRouter(limiter)

	userOneToken, err := jwtpkg.GenerateToken(1)
	if err != nil {
		t.Fatalf("generate user one token: %v", err)
	}
	userTwoToken, err := jwtpkg.GenerateToken(2)
	if err != nil {
		t.Fatalf("generate user two token: %v", err)
	}

	first := performRateLimitRequest(r, http.MethodPost, "/api/posts/create", "203.0.113.20:1000", "", userOneToken)
	if first.Code != http.StatusOK {
		t.Fatalf("first user write status = %d, want 200", first.Code)
	}
	// Changing IP does not reset a valid signed user's write quota.
	second := performRateLimitRequest(r, http.MethodPost, "/api/posts/create", "198.51.100.20:2000", "", userOneToken)
	assertRateLimited(t, second)

	// Another signed user behind the same IP has a separate write quota.
	otherUser := performRateLimitRequest(r, http.MethodPost, "/api/posts/create", "203.0.113.20:3000", "", userTwoToken)
	if otherUser.Code != http.StatusOK {
		t.Fatalf("other user write status = %d, want 200; body=%s", otherUser.Code, otherUser.Body.String())
	}
}

func TestTieredRateLimiter_SearchUsesDedicatedQuota(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := testRateLimitConfig()
	cfg.Search = middleware.RateLimitPolicy{Requests: 1, Window: time.Hour, Burst: 1}
	r := newRateLimitRouter(mustRateLimiter(t, cfg))

	first := performRateLimitRequest(r, http.MethodGet, "/api/search?q=go", "203.0.113.25:1000", "", "")
	if first.Code != http.StatusOK {
		t.Fatalf("first search status = %d, want 200; body=%s", first.Code, first.Body.String())
	}
	limited := performRateLimitRequest(r, http.MethodGet, "/api/search?q=go", "203.0.113.25:2000", "", "")
	assertRateLimited(t, limited)

	// Exhausting the heavier search tier must not consume the ordinary read bucket.
	read := performRateLimitRequest(r, http.MethodGet, "/api/posts", "203.0.113.25:3000", "", "")
	if read.Code != http.StatusOK {
		t.Fatalf("read status = %d, want 200; body=%s", read.Code, read.Body.String())
	}
}

func TestRouterTrustsXRealIPOnlyFromLoopbackProxy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	newRouter := func() *gin.Engine {
		cfg := testRateLimitConfig()
		cfg.Login = middleware.RateLimitPolicy{Requests: 1, Window: time.Hour, Burst: 1}
		return router.SetupRouter(router.RouterDeps{
			RateLimiter: mustRateLimiter(t, cfg),
			Modules: []router.RouteRegister{
				rateLimitRouteRegister(func(group *gin.RouterGroup) {
					group.POST("/login", func(c *gin.Context) { c.Status(http.StatusOK) })
				}),
			},
		})
	}

	t.Run("untrusted peer cannot spoof header", func(t *testing.T) {
		r := newRouter()
		first := performRateLimitRequest(r, http.MethodPost, "/api/login", "203.0.113.30:1000", "198.51.100.1", "")
		second := performRateLimitRequest(r, http.MethodPost, "/api/login", "203.0.113.30:2000", "198.51.100.2", "")
		if first.Code != http.StatusOK {
			t.Fatalf("first status = %d, want 200", first.Code)
		}
		assertRateLimited(t, second)
	})

	t.Run("loopback reverse proxy supplies client IP", func(t *testing.T) {
		r := newRouter()
		first := performRateLimitRequest(r, http.MethodPost, "/api/login", "127.0.0.1:1000", "198.51.100.1", "")
		second := performRateLimitRequest(r, http.MethodPost, "/api/login", "127.0.0.1:2000", "198.51.100.2", "")
		if first.Code != http.StatusOK || second.Code != http.StatusOK {
			t.Fatalf("loopback proxy statuses = %d, %d; want 200, 200", first.Code, second.Code)
		}
	})

	t.Run("x-forwarded-for is not a trusted input", func(t *testing.T) {
		r := newRouter()
		first := performRateLimitRequestWithForwardedFor(r, "198.51.100.1")
		second := performRateLimitRequestWithForwardedFor(r, "198.51.100.2")
		if first.Code != http.StatusOK {
			t.Fatalf("first status = %d, want 200", first.Code)
		}
		assertRateLimited(t, second)
	})
}

func TestTieredRateLimiter_IsBoundedAndExpiresIdleEntries(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	cfg := testRateLimitConfig()
	cfg.Read = middleware.RateLimitPolicy{Requests: 1, Window: time.Hour, Burst: 1}
	cfg.MaxEntries = 1
	cfg.IdleTTL = time.Minute
	cfg.Now = func() time.Time { return now }
	r := newRateLimitRouter(mustRateLimiter(t, cfg))

	tracked := performRateLimitRequest(r, http.MethodGet, "/api/posts", "203.0.113.40:1000", "", "")
	overflow := performRateLimitRequest(r, http.MethodGet, "/api/posts", "203.0.113.41:1000", "", "")
	sharedOverflow := performRateLimitRequest(r, http.MethodGet, "/api/posts", "203.0.113.42:1000", "", "")
	if tracked.Code != http.StatusOK || overflow.Code != http.StatusOK {
		t.Fatalf("initial statuses = %d, %d; want 200, 200", tracked.Code, overflow.Code)
	}
	assertRateLimited(t, sharedOverflow)

	// After the tracked entry becomes idle, a new client gets the released slot
	// instead of remaining in the shared overflow bucket.
	now = now.Add(2 * time.Minute)
	afterExpiry := performRateLimitRequest(r, http.MethodGet, "/api/posts", "203.0.113.42:1000", "", "")
	if afterExpiry.Code != http.StatusOK {
		t.Fatalf("status after idle expiry = %d, want 200; body=%s", afterExpiry.Code, afterExpiry.Body.String())
	}
}

func TestTieredRateLimiter_IsConcurrencySafe(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := testRateLimitConfig()
	cfg.Read = middleware.RateLimitPolicy{Requests: 1, Window: time.Hour, Burst: 10}
	r := newRateLimitRouter(mustRateLimiter(t, cfg))

	const requestCount = 100
	var successes atomic.Int32
	var limited atomic.Int32
	var wg sync.WaitGroup
	wg.Add(requestCount)
	for i := 0; i < requestCount; i++ {
		go func() {
			defer wg.Done()
			response := performRateLimitRequest(r, http.MethodGet, "/api/posts", "203.0.113.50:1234", "", "")
			switch response.Code {
			case http.StatusOK:
				successes.Add(1)
			case http.StatusTooManyRequests:
				limited.Add(1)
			}
		}()
	}
	wg.Wait()

	if got := successes.Load(); got != 10 {
		t.Fatalf("successful requests = %d, want 10", got)
	}
	if got := limited.Load(); got != requestCount-10 {
		t.Fatalf("limited requests = %d, want %d", got, requestCount-10)
	}
}

func testRateLimitConfig() middleware.TieredRateLimitConfig {
	policy := middleware.RateLimitPolicy{Requests: 1000, Window: time.Minute, Burst: 1000}
	return middleware.TieredRateLimitConfig{
		Login:      policy,
		Register:   policy,
		Search:     policy,
		Write:      policy,
		Read:       policy,
		MaxEntries: 100,
		IdleTTL:    time.Minute,
	}
}

func mustRateLimiter(t *testing.T, cfg middleware.TieredRateLimitConfig) *middleware.TieredRateLimiter {
	t.Helper()
	limiter, err := middleware.NewTieredRateLimiter(cfg)
	if err != nil {
		t.Fatalf("NewTieredRateLimiter() error = %v", err)
	}
	return limiter
}

func newRateLimitRouter(limiter *middleware.TieredRateLimiter) *gin.Engine {
	r := gin.New()
	_ = r.SetTrustedProxies(nil)
	r.Use(middleware.RequestID(), middleware.ErrorHandler())
	api := r.Group("/api")
	api.Use(limiter.Handler())
	api.Any("/*path", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func performRateLimitRequest(handler http.Handler, method, path, remoteAddr, realIP, token string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, nil)
	request.RemoteAddr = remoteAddr
	if realIP != "" {
		request.Header.Set("X-Real-IP", realIP)
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func performRateLimitRequestWithForwardedFor(handler http.Handler, forwardedFor string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/api/login", nil)
	request.RemoteAddr = "127.0.0.1:1234"
	request.Header.Set("X-Forwarded-For", forwardedFor)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func assertRateLimited(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	if response.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429; body=%s", response.Code, response.Body.String())
	}
	if response.Header().Get("Retry-After") == "" {
		t.Fatal("rate-limited response is missing Retry-After")
	}
	var body struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, response.Body.String())
	}
	if body.Code != bizerr.ErrTooManyRequests.Code || body.Message != bizerr.ErrTooManyRequests.Message {
		t.Fatalf("body = %+v, want code=%d message=%q", body, bizerr.ErrTooManyRequests.Code, bizerr.ErrTooManyRequests.Message)
	}
}
