package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"my-bbs/internal/model"
	searchmodule "my-bbs/internal/modules/search"
	"my-bbs/internal/router"
	"my-bbs/pkg/bizerr"
	"my-bbs/tests/testutil"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type searchUserContract struct {
	ID           uint   `json:"id"`
	Username     string `json:"username"`
	Nickname     string `json:"nickname"`
	Introduction string `json:"introduction"`
}

type searchPostAuthorContract struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
}

type searchPostContract struct {
	ID           uint                      `json:"id"`
	UserID       uint                      `json:"user_id"`
	Title        string                    `json:"title"`
	Excerpt      string                    `json:"excerpt"`
	User         *searchPostAuthorContract `json:"user"`
	LikeCount    int64                     `json:"like_count"`
	CommentCount int64                     `json:"comment_count"`
}

type searchPageContract[T any] struct {
	List     []T  `json:"list"`
	PageNo   int  `json:"pageNo"`
	PageSize int  `json:"pageSize"`
	HasMore  bool `json:"hasMore"`
}

type searchDataContract struct {
	Query string                                 `json:"query"`
	Scope string                                 `json:"scope"`
	Users searchPageContract[searchUserContract] `json:"users"`
	Posts searchPageContract[searchPostContract] `json:"posts"`
}

type searchResponseContract struct {
	Code    int                `json:"code"`
	Message string             `json:"message"`
	Data    searchDataContract `json:"data"`
}

func setupSearchTestRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	r := router.SetupRouter(router.RouterDeps{
		Modules: []router.RouteRegister{searchmodule.Initialize(db, time.Second)},
	})
	return r, db
}

func createSearchFixtureUser(t *testing.T, db *gorm.DB, username, nickname string) *model.User {
	t.Helper()
	user := &model.User{
		Username:     username,
		Password:     "fixture-password-not-used",
		Nickname:     nickname,
		Status:       model.UserStatusNormal,
		Introduction: "public introduction for " + nickname,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create search fixture user %q: %v", username, err)
	}
	return user
}

func createSearchFixturePost(t *testing.T, db *gorm.DB, userID uint, title, content string, visible uint8) *model.Post {
	t.Helper()
	post := &model.Post{
		UserID:  userID,
		Title:   title,
		Content: content,
		Visible: visible,
	}
	if err := db.Create(post).Error; err != nil {
		t.Fatalf("create search fixture post %q: %v", title, err)
	}
	return post
}

func performSearchRequest(t *testing.T, handler http.Handler, values url.Values) *httptest.ResponseRecorder {
	t.Helper()
	path := "/api/search"
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}
	req := httptest.NewRequest(http.MethodGet, path, nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w
}

func decodeSearchResponse(t *testing.T, w *httptest.ResponseRecorder) searchResponseContract {
	t.Helper()
	var response searchResponseContract
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode search response: %v body=%s", err, w.Body.String())
	}
	return response
}

func requireSearchSuccess(t *testing.T, w *httptest.ResponseRecorder) searchResponseContract {
	t.Helper()
	if w.Code != http.StatusOK {
		t.Fatalf("search status=%d, want=200 body=%s", w.Code, w.Body.String())
	}
	if requestID := w.Header().Get("X-Request-ID"); requestID == "" {
		t.Fatal("successful search response is missing X-Request-ID")
	}
	response := decodeSearchResponse(t, w)
	if response.Code != 0 || response.Message != "ok" {
		t.Fatalf("unexpected success envelope: %+v", response)
	}
	return response
}

func searchUserIDs(users []searchUserContract) []uint {
	ids := make([]uint, len(users))
	for i := range users {
		ids[i] = users[i].ID
	}
	return ids
}

func searchPostIDs(posts []searchPostContract) []uint {
	ids := make([]uint, len(posts))
	for i := range posts {
		ids[i] = posts[i].ID
	}
	return ids
}

func requireSearchIDs(t *testing.T, got, want []uint) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("ids=%v, want=%v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ids=%v, want=%v", got, want)
		}
	}
}

func TestSearchAPI_SearchesUsersAndPostsWithStableRanking(t *testing.T) {
	r, db := setupSearchTestRouter(t)

	exactUsername := createSearchFixtureUser(t, db, "go", "Exact username")
	exactNickname := createSearchFixtureUser(t, db, "nickname-exact", "go")
	prefixUsername := createSearchFixtureUser(t, db, "gopher", "Prefix username")
	prefixNickname := createSearchFixtureUser(t, db, "nickname-prefix", "go reader")

	exactTitle := createSearchFixturePost(t, db, exactUsername.ID, "go", "exact-title body", model.VisiblePublic)
	prefixTitle := createSearchFixturePost(t, db, exactUsername.ID, "go guide", "prefix-title body", model.VisiblePublic)
	containsTitle := createSearchFixturePost(t, db, exactUsername.ID, "learning go today", "contains-title body", model.VisiblePublic)
	contentOnly := createSearchFixturePost(t, db, exactUsername.ID, "unrelated title", "this body discusses go contexts", model.VisiblePublic)

	w := performSearchRequest(t, r, url.Values{
		"q":        {"  go  "},
		"scope":    {"all"},
		"pageNo":   {"1"},
		"pageSize": {"10"},
	})
	response := requireSearchSuccess(t, w)

	if response.Data.Query != "go" || response.Data.Scope != "all" {
		t.Fatalf("normalized query/scope=%q/%q, want go/all", response.Data.Query, response.Data.Scope)
	}
	requireSearchIDs(t, searchUserIDs(response.Data.Users.List), []uint{
		exactUsername.ID,
		exactNickname.ID,
		prefixUsername.ID,
		prefixNickname.ID,
	})
	requireSearchIDs(t, searchPostIDs(response.Data.Posts.List), []uint{
		exactTitle.ID,
		prefixTitle.ID,
		containsTitle.ID,
		contentOnly.ID,
	})

	if response.Data.Posts.List[3].Excerpt == "" {
		t.Fatal("content match must include a non-empty plain-text excerpt")
	}
	for _, post := range response.Data.Posts.List {
		if post.User == nil || post.User.ID != exactUsername.ID {
			t.Fatalf("post %d has unexpected author: %+v", post.ID, post.User)
		}
	}
}

func TestSearchAPI_ScopeAndEmptyResultsKeepStableResponseShape(t *testing.T) {
	r, db := setupSearchTestRouter(t)
	user := createSearchFixtureUser(t, db, "scope-user", "scope-key member")
	post := createSearchFixturePost(t, db, user.ID, "scope-key post", "body", model.VisiblePublic)

	t.Run("default all", func(t *testing.T) {
		w := performSearchRequest(t, r, url.Values{"q": {"scope-key"}})
		response := requireSearchSuccess(t, w)
		if response.Data.Scope != "all" {
			t.Fatalf("scope=%q, want all", response.Data.Scope)
		}
		requireSearchIDs(t, searchUserIDs(response.Data.Users.List), []uint{user.ID})
		requireSearchIDs(t, searchPostIDs(response.Data.Posts.List), []uint{post.ID})
	})

	t.Run("users only", func(t *testing.T) {
		w := performSearchRequest(t, r, url.Values{"q": {"scope-key"}, "scope": {"users"}})
		response := requireSearchSuccess(t, w)
		requireSearchIDs(t, searchUserIDs(response.Data.Users.List), []uint{user.ID})
		if response.Data.Posts.List == nil || len(response.Data.Posts.List) != 0 || response.Data.Posts.HasMore {
			t.Fatalf("unrequested posts page must be a non-nil empty page: %+v", response.Data.Posts)
		}
	})

	t.Run("posts only", func(t *testing.T) {
		w := performSearchRequest(t, r, url.Values{"q": {"scope-key"}, "scope": {"posts"}})
		response := requireSearchSuccess(t, w)
		requireSearchIDs(t, searchPostIDs(response.Data.Posts.List), []uint{post.ID})
		if response.Data.Users.List == nil || len(response.Data.Users.List) != 0 || response.Data.Users.HasMore {
			t.Fatalf("unrequested users page must be a non-nil empty page: %+v", response.Data.Users)
		}
	})

	t.Run("no matches", func(t *testing.T) {
		w := performSearchRequest(t, r, url.Values{"q": {"no-such-result"}})
		response := requireSearchSuccess(t, w)
		if response.Data.Users.List == nil || response.Data.Posts.List == nil {
			t.Fatalf("empty result lists must encode as arrays: %+v", response.Data)
		}
		if len(response.Data.Users.List) != 0 || len(response.Data.Posts.List) != 0 {
			t.Fatalf("unexpected empty-search results: %+v", response.Data)
		}
	})
}

func TestSearchAPI_DoesNotLeakPrivateOrSoftDeletedRecords(t *testing.T) {
	r, db := setupSearchTestRouter(t)
	author := createSearchFixtureUser(t, db, "privacy-author", "ordinary author")
	deletedUser := createSearchFixtureUser(t, db, "deleted-user", "privacy-key deleted member")
	if err := db.Delete(deletedUser).Error; err != nil {
		t.Fatalf("soft delete user: %v", err)
	}

	publicPost := createSearchFixturePost(t, db, author.ID, "privacy-key public", "visible", model.VisiblePublic)
	createSearchFixturePost(t, db, author.ID, "privacy-key private", "must stay hidden", model.VisiblePrivate)
	deletedPost := createSearchFixturePost(t, db, author.ID, "privacy-key deleted", "must stay hidden", model.VisiblePublic)
	if err := db.Delete(deletedPost).Error; err != nil {
		t.Fatalf("soft delete post: %v", err)
	}

	w := performSearchRequest(t, r, url.Values{"q": {"privacy-key"}})
	response := requireSearchSuccess(t, w)
	if len(response.Data.Users.List) != 0 {
		t.Fatalf("soft-deleted user leaked through search: %+v", response.Data.Users.List)
	}
	requireSearchIDs(t, searchPostIDs(response.Data.Posts.List), []uint{publicPost.ID})
}

func TestSearchAPI_TreatsLikeWildcardsAsLiteralCharacters(t *testing.T) {
	r, db := setupSearchTestRouter(t)
	author := createSearchFixtureUser(t, db, "literal-author", "literal !%_ member")
	createSearchFixtureUser(t, db, "wildcard-decoy", "literal abc member")
	literalPost := createSearchFixturePost(t, db, author.ID, "literal !%_ post", "body", model.VisiblePublic)
	createSearchFixturePost(t, db, author.ID, "literal abc post", "body", model.VisiblePublic)

	w := performSearchRequest(t, r, url.Values{"q": {"!%_"}})
	response := requireSearchSuccess(t, w)
	requireSearchIDs(t, searchUserIDs(response.Data.Users.List), []uint{author.ID})
	requireSearchIDs(t, searchPostIDs(response.Data.Posts.List), []uint{literalPost.ID})

	w = performSearchRequest(t, r, url.Values{"q": {"' OR 1=1 --"}})
	response = requireSearchSuccess(t, w)
	if len(response.Data.Users.List) != 0 || len(response.Data.Posts.List) != 0 {
		t.Fatalf("SQL-looking input must remain ordinary search text: %+v", response.Data)
	}
}

func TestSearchAPI_UsesLookaheadForHasMore(t *testing.T) {
	tests := []struct {
		name        string
		matchCount  int
		wantHasMore bool
	}{
		{name: "exact page", matchCount: 2, wantHasMore: false},
		{name: "one extra result", matchCount: 3, wantHasMore: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, db := setupSearchTestRouter(t)
			author := createSearchFixtureUser(t, db, "page-author", "ordinary author")
			for i := 0; i < tt.matchCount; i++ {
				suffix := string(rune('a' + i))
				createSearchFixtureUser(t, db, "page-user-"+suffix, "page-key "+suffix)
				createSearchFixturePost(t, db, author.ID, "page-key "+suffix, "body", model.VisiblePublic)
			}

			w := performSearchRequest(t, r, url.Values{
				"q":        {"page-key"},
				"pageNo":   {"1"},
				"pageSize": {"2"},
			})
			response := requireSearchSuccess(t, w)
			if len(response.Data.Users.List) != 2 || len(response.Data.Posts.List) != 2 {
				t.Fatalf("page lengths users/posts=%d/%d, want 2/2", len(response.Data.Users.List), len(response.Data.Posts.List))
			}
			if response.Data.Users.HasMore != tt.wantHasMore || response.Data.Posts.HasMore != tt.wantHasMore {
				t.Fatalf("hasMore users/posts=%v/%v, want %v", response.Data.Users.HasMore, response.Data.Posts.HasMore, tt.wantHasMore)
			}
		})
	}
}

func TestSearchAPI_RejectsInvalidQueryAndPagination(t *testing.T) {
	r, _ := setupSearchTestRouter(t)
	tests := []struct {
		name   string
		values url.Values
	}{
		{name: "missing q", values: url.Values{}},
		{name: "blank q", values: url.Values{"q": {"  \t  "}}},
		{name: "one rune q", values: url.Values{"q": {"x"}}},
		{name: "q too long", values: url.Values{"q": {strings.Repeat("界", 51)}}},
		{name: "invalid scope", values: url.Values{"q": {"valid"}, "scope": {"everything"}}},
		{name: "zero page number", values: url.Values{"q": {"valid"}, "pageNo": {"0"}}},
		{name: "malformed page number", values: url.Values{"q": {"valid"}, "pageNo": {"abc"}}},
		{name: "zero page size", values: url.Values{"q": {"valid"}, "pageSize": {"0"}}},
		{name: "page size above search maximum", values: url.Values{"q": {"valid"}, "pageSize": {"21"}}},
		{name: "offset above search maximum", values: url.Values{"q": {"valid"}, "pageNo": {"102"}, "pageSize": {"10"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := performSearchRequest(t, r, tt.values)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status=%d, want=400 body=%s", w.Code, w.Body.String())
			}
			if requestID := w.Header().Get("X-Request-ID"); requestID == "" {
				t.Fatal("invalid search response is missing X-Request-ID")
			}
			var body struct {
				Code int `json:"code"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode invalid search response: %v body=%s", err, w.Body.String())
			}
			if body.Code != bizerr.ErrBadRequest.Code {
				t.Fatalf("code=%d, want=%d body=%s", body.Code, bizerr.ErrBadRequest.Code, w.Body.String())
			}
		})
	}
}

func TestSearchAPI_ResponseDoesNotExposePersistenceOrFullContentFields(t *testing.T) {
	r, db := setupSearchTestRouter(t)
	user := createSearchFixtureUser(t, db, "safe-user", "safe-key member")
	createSearchFixturePost(t, db, user.ID, "safe-key post", strings.Repeat("full-content-must-not-leak ", 20), model.VisiblePublic)

	w := performSearchRequest(t, r, url.Values{"q": {"safe-key"}})
	requireSearchSuccess(t, w)

	var envelope map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode raw search response: %v", err)
	}
	data := envelope["data"].(map[string]any)
	users := data["users"].(map[string]any)["list"].([]any)
	posts := data["posts"].(map[string]any)["list"].([]any)
	if len(users) != 1 || len(posts) != 1 {
		t.Fatalf("unexpected raw search lists: users=%v posts=%v", users, posts)
	}

	userObject := users[0].(map[string]any)
	for _, forbidden := range []string{"password", "deleted", "status", "create_time", "update_time"} {
		if _, exists := userObject[forbidden]; exists {
			t.Fatalf("search user exposes forbidden field %q: %v", forbidden, userObject)
		}
	}
	postObject := posts[0].(map[string]any)
	for _, forbidden := range []string{"content", "deleted", "visible", "update_time"} {
		if _, exists := postObject[forbidden]; exists {
			t.Fatalf("search post exposes forbidden field %q: %v", forbidden, postObject)
		}
	}
	if excerpt, _ := postObject["excerpt"].(string); utf8.RuneCountInString(excerpt) > 120 {
		t.Fatalf("search excerpt length=%d, want <=120", utf8.RuneCountInString(excerpt))
	}
	authorObject := postObject["user"].(map[string]any)
	for _, forbidden := range []string{"password", "deleted", "status", "introduction", "create_time", "update_time"} {
		if _, exists := authorObject[forbidden]; exists {
			t.Fatalf("search post author exposes forbidden field %q: %v", forbidden, authorObject)
		}
	}
}
