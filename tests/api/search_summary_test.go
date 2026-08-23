package api_test

import (
	"net/url"
	"strings"
	"testing"

	"my-bbs/internal/model"
)

func TestSearchAPI_ReturnsAuthorAndInteractionSummary(t *testing.T) {
	r, db := setupSearchTestRouter(t)
	author := createSearchFixtureUser(t, db, "summary-author", "摘要作者")
	reader := createSearchFixtureUser(t, db, "summary-reader", "摘要读者")
	post := createSearchFixturePost(t, db, author.ID, "summary-key post", "body", model.VisiblePublic)

	if err := db.Create(&model.PostLike{PostID: post.ID, UserID: reader.ID}).Error; err != nil {
		t.Fatalf("create search fixture like: %v", err)
	}
	if err := db.Create(&model.Comment{PostID: post.ID, UserID: reader.ID, Content: "comment"}).Error; err != nil {
		t.Fatalf("create search fixture comment: %v", err)
	}

	w := performSearchRequest(t, r, url.Values{"q": {"summary-key"}, "scope": {"posts"}})
	response := requireSearchSuccess(t, w)
	if len(response.Data.Posts.List) != 1 {
		t.Fatalf("post count=%d, want=1", len(response.Data.Posts.List))
	}
	result := response.Data.Posts.List[0]
	if result.User == nil || result.User.ID != author.ID {
		t.Fatalf("author=%+v, want user %d", result.User, author.ID)
	}
	if result.LikeCount != 1 || result.CommentCount != 1 {
		t.Fatalf("like/comment counts=%d/%d, want=1/1", result.LikeCount, result.CommentCount)
	}
}

func TestSearchAPI_ContentMatchExcerptContainsMatchContext(t *testing.T) {
	r, db := setupSearchTestRouter(t)
	author := createSearchFixtureUser(t, db, "excerpt-author", "摘要作者")
	createSearchFixturePost(
		t,
		db,
		author.ID,
		"unrelated title",
		strings.Repeat("before ", 40)+"needle-key appears here",
		model.VisiblePublic,
	)

	w := performSearchRequest(t, r, url.Values{"q": {"needle-key"}, "scope": {"posts"}})
	response := requireSearchSuccess(t, w)
	if len(response.Data.Posts.List) != 1 {
		t.Fatalf("post count=%d, want=1", len(response.Data.Posts.List))
	}
	if excerpt := response.Data.Posts.List[0].Excerpt; !strings.Contains(excerpt, "needle-key") {
		t.Fatalf("excerpt=%q does not contain the content match", excerpt)
	}
}

func TestSearchAPI_PublicPostWithDeletedAuthorReturnsNullAuthor(t *testing.T) {
	r, db := setupSearchTestRouter(t)
	author := createSearchFixtureUser(t, db, "orphan-author", "待删除作者")
	post := createSearchFixturePost(t, db, author.ID, "orphan-key post", "body", model.VisiblePublic)
	if err := db.Delete(author).Error; err != nil {
		t.Fatalf("soft delete author: %v", err)
	}

	w := performSearchRequest(t, r, url.Values{"q": {"orphan-key"}, "scope": {"posts"}})
	response := requireSearchSuccess(t, w)
	if len(response.Data.Posts.List) != 1 || response.Data.Posts.List[0].ID != post.ID {
		t.Fatalf("unexpected post results: %+v", response.Data.Posts.List)
	}
	if response.Data.Posts.List[0].User != nil {
		t.Fatalf("deleted author must be encoded as null: %+v", response.Data.Posts.List[0].User)
	}
}
