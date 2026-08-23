package middleware

import (
	"container/list"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"my-bbs/pkg/bizerr"
	"my-bbs/pkg/jwt"
	"my-bbs/pkg/response"

	"github.com/gin-gonic/gin"
)

type rateTier string

const (
	rateTierLogin    rateTier = "login"
	rateTierRegister rateTier = "register"
	rateTierSearch   rateTier = "search"
	rateTierWrite    rateTier = "write"
	rateTierRead     rateTier = "read"
)

// RateLimitPolicy describes a token bucket. Requests tokens are replenished
// over Window, while Burst is the maximum number of immediately available
// tokens.
type RateLimitPolicy struct {
	Requests int
	Window   time.Duration
	Burst    int
}

// TieredRateLimitConfig configures all API rate-limit tiers. MaxEntries is a
// process-wide upper bound across all tiers; when it is full, previously
// unseen clients share a small overflow bucket instead of growing memory.
type TieredRateLimitConfig struct {
	Login      RateLimitPolicy
	Register   RateLimitPolicy
	Search     RateLimitPolicy
	Write      RateLimitPolicy
	Read       RateLimitPolicy
	MaxEntries int
	IdleTTL    time.Duration
	Now        func() time.Time
}

type rateLimitEntry struct {
	tokens    float64
	updatedAt time.Time
	lastSeen  time.Time
	element   *list.Element
}

// TieredRateLimiter is an in-memory, concurrency-safe token bucket limiter.
// Its state is intentionally local to one application process; a reverse
// proxy or shared store is still needed to enforce a cluster-wide limit.
type TieredRateLimiter struct {
	mu         sync.Mutex
	policies   map[rateTier]RateLimitPolicy
	entries    map[string]*rateLimitEntry
	overflow   map[rateTier]*rateLimitEntry
	lru        *list.List
	maxEntries int
	idleTTL    time.Duration
	now        func() time.Time
}

// NewTieredRateLimiter builds a bounded limiter and validates every policy.
func NewTieredRateLimiter(cfg TieredRateLimitConfig) (*TieredRateLimiter, error) {
	policies := map[rateTier]RateLimitPolicy{
		rateTierLogin:    cfg.Login,
		rateTierRegister: cfg.Register,
		rateTierSearch:   cfg.Search,
		rateTierWrite:    cfg.Write,
		rateTierRead:     cfg.Read,
	}
	for tier, policy := range policies {
		if policy.Requests <= 0 {
			return nil, fmt.Errorf("%s rate limit requests must be greater than zero", tier)
		}
		if policy.Window <= 0 {
			return nil, fmt.Errorf("%s rate limit window must be greater than zero", tier)
		}
		if policy.Burst <= 0 {
			return nil, fmt.Errorf("%s rate limit burst must be greater than zero", tier)
		}
	}
	if cfg.MaxEntries <= 0 {
		return nil, fmt.Errorf("rate limit max entries must be greater than zero")
	}
	if cfg.IdleTTL <= 0 {
		return nil, fmt.Errorf("rate limit idle TTL must be greater than zero")
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	return &TieredRateLimiter{
		policies:   policies,
		entries:    make(map[string]*rateLimitEntry),
		overflow:   make(map[rateTier]*rateLimitEntry, len(policies)),
		lru:        list.New(),
		maxEntries: cfg.MaxEntries,
		idleTTL:    cfg.IdleTTL,
		now:        now,
	}, nil
}

// Handler applies the appropriate tier to each /api request. Login and
// registration are always keyed by client IP. Mutation requests use the user
// ID from a valid signed JWT when available, falling back to the client IP.
func (l *TieredRateLimiter) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		tier := classifyRateTier(c.Request.Method, c.Request.URL.Path)
		identity := clientIdentity(c)
		if tier == rateTierWrite {
			if userID, ok := signedTokenUserID(c.GetHeader("Authorization")); ok {
				identity = "user:" + strconv.FormatUint(uint64(userID), 10)
			}
		}

		allowed, retryAfter := l.allow(tier, identity)
		if !allowed {
			seconds := int64(math.Ceil(retryAfter.Seconds()))
			if seconds < 1 {
				seconds = 1
			}
			c.Header("Retry-After", strconv.FormatInt(seconds, 10))
			response.ReportError(c, bizerr.ErrTooManyRequests)
			return
		}
		c.Next()
	}
}

func classifyRateTier(method, path string) rateTier {
	switch {
	case method == http.MethodPost && path == "/api/login":
		return rateTierLogin
	case method == http.MethodPost && path == "/api/register":
		return rateTierRegister
	case method == http.MethodGet && path == "/api/search":
		return rateTierSearch
	case method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions:
		return rateTierRead
	case method == http.MethodPost && path == "/api/user/posts":
		// This legacy endpoint reads the current user's posts despite using POST.
		return rateTierRead
	default:
		return rateTierWrite
	}
}

func clientIdentity(c *gin.Context) string {
	if ip := strings.TrimSpace(c.ClientIP()); ip != "" {
		return "ip:" + ip
	}
	// Malformed or synthetic requests with no parseable remote address share
	// one bucket instead of bypassing the limiter.
	return "ip:unknown"
}

func signedTokenUserID(header string) (uint, bool) {
	// Do not hash an attacker-controlled, near-MaxHeaderBytes token before the
	// IP bucket can reject it. Real JWTs generated by this service are tiny.
	if len(header) == 0 || len(header) > 4096 {
		return 0, false
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return 0, false
	}
	claims, err := jwt.ParseClaims(parts[1])
	if err != nil || claims.UserID == 0 {
		return 0, false
	}
	return claims.UserID, true
}

func (l *TieredRateLimiter) allow(tier rateTier, identity string) (bool, time.Duration) {
	now := l.now()
	policy := l.policies[tier]
	key := string(tier) + "\x00" + identity

	l.mu.Lock()
	defer l.mu.Unlock()

	l.pruneIdle(now)
	entry, exists := l.entries[key]
	if !exists {
		if len(l.entries) >= l.maxEntries {
			entry = l.overflow[tier]
			if entry == nil {
				entry = newRateLimitEntry(policy, now)
				l.overflow[tier] = entry
			}
			return consumeToken(entry, policy, now)
		}

		entry = newRateLimitEntry(policy, now)
		entry.element = l.lru.PushBack(key)
		l.entries[key] = entry
	} else {
		entry.lastSeen = now
		l.lru.MoveToBack(entry.element)
	}
	return consumeToken(entry, policy, now)
}

func newRateLimitEntry(policy RateLimitPolicy, now time.Time) *rateLimitEntry {
	return &rateLimitEntry{
		tokens:    float64(policy.Burst),
		updatedAt: now,
		lastSeen:  now,
	}
}

func consumeToken(entry *rateLimitEntry, policy RateLimitPolicy, now time.Time) (bool, time.Duration) {
	elapsed := now.Sub(entry.updatedAt)
	if elapsed < 0 {
		elapsed = 0
	}
	refillPerSecond := float64(policy.Requests) / policy.Window.Seconds()
	entry.tokens = math.Min(float64(policy.Burst), entry.tokens+elapsed.Seconds()*refillPerSecond)
	entry.updatedAt = now
	entry.lastSeen = now

	if entry.tokens >= 1 {
		entry.tokens--
		return true, 0
	}
	waitSeconds := (1 - entry.tokens) / refillPerSecond
	return false, time.Duration(math.Ceil(waitSeconds * float64(time.Second)))
}

func (l *TieredRateLimiter) pruneIdle(now time.Time) {
	for element := l.lru.Front(); element != nil; element = l.lru.Front() {
		key, _ := element.Value.(string)
		entry := l.entries[key]
		if entry == nil {
			l.lru.Remove(element)
			continue
		}
		if now.Sub(entry.lastSeen) < l.idleTTL {
			return
		}
		delete(l.entries, key)
		l.lru.Remove(element)
	}
}
