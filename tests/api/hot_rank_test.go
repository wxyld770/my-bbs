package api_test

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"my-bbs/internal/model"

	"gorm.io/gorm"
)

const (
	hotRankPasswordSecret = "hot-rank-password-must-not-leak"
	hotRankContentSecret  = "hot-rank-content-must-not-leak"
)

type hotRankAPIResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		List []hotRankPostItem `json:"list"`
	} `json:"data"`
}

type hotRankPostItem struct {
	ID           uint            `json:"id"`
	Title        string          `json:"title"`
	Visible      uint8           `json:"visible"`
	LikeCount    int64           `json:"like_count"`
	CommentCount int64           `json:"comment_count"`
	Score        float64         `json:"score"`
	User         json.RawMessage `json:"user"`
}

func TestAPI_HotPostsUsesYesterdayAndTodayWindowAndHidesPrivateData(t *testing.T) {
	r, db := setupTestRouter(t)
	author := createHotRankAuthor(t, db)
	startYesterday, startTomorrow := hotRankWindow(time.Now())

	lowerBoundary := createHotRankPost(t, db, author.ID, "included-yesterday-boundary", startYesterday, model.VisiblePublic)
	upperBoundary := createHotRankPost(t, db, author.ID, "included-before-tomorrow", startTomorrow.Add(-time.Millisecond), model.VisiblePublic)
	beforeWindow := createHotRankPost(t, db, author.ID, "excluded-before-yesterday", startYesterday.Add(-time.Millisecond), model.VisiblePublic)
	tomorrowBoundary := createHotRankPost(t, db, author.ID, "excluded-tomorrow-boundary", startTomorrow, model.VisiblePublic)
	privatePost := createHotRankPost(t, db, author.ID, "excluded-private", startYesterday.Add(12*time.Hour), model.VisiblePrivate)
	deletedPost := createHotRankPost(t, db, author.ID, "excluded-soft-deleted", startYesterday.Add(13*time.Hour), model.VisiblePublic)
	if err := db.Delete(&deletedPost).Error; err != nil {
		t.Fatalf("soft-delete hot-rank fixture: %v", err)
	}

	w := doJSON(t, r, http.MethodGet, "/api/posts/hot", "", nil)
	response := decodeHotRankResponse(t, w)
	if len(response.Data.List) != 2 {
		t.Fatalf("hot list length=%d, want=2 body=%s", len(response.Data.List), w.Body.String())
	}
	if got, want := []uint{response.Data.List[0].ID, response.Data.List[1].ID}, []uint{upperBoundary.ID, lowerBoundary.ID}; !equalHotRankIDs(got, want) {
		t.Fatalf("hot list ids=%v, want=%v", got, want)
	}

	excluded := []model.Post{beforeWindow, tomorrowBoundary, privatePost, deletedPost}
	for _, post := range excluded {
		if strings.Contains(w.Body.String(), post.Title) {
			t.Fatalf("excluded post %q leaked into hot list: %s", post.Title, w.Body.String())
		}
	}
	if strings.Contains(w.Body.String(), hotRankContentSecret) {
		t.Fatalf("post content leaked into hot list: %s", w.Body.String())
	}
	if strings.Contains(w.Body.String(), hotRankPasswordSecret) {
		t.Fatalf("user password leaked into hot list: %s", w.Body.String())
	}

	assertHotRankResponseHasNoPrivateFields(t, w.Body.Bytes())
}

func TestAPI_HotPostsOrdersByScoreCommentsAndStableTies(t *testing.T) {
	r, db := setupTestRouter(t)
	author := createHotRankAuthor(t, db)
	startYesterday, _ := hotRankWindow(time.Now())
	baseTime := startYesterday.Add(12 * time.Hour)

	leader := createHotRankPost(t, db, author.ID, "leader", baseTime, model.VisiblePublic)
	commentTieWinner := createHotRankPost(t, db, author.ID, "same-score-more-comments", baseTime.Add(time.Minute), model.VisiblePublic)
	commentTieLoser := createHotRankPost(t, db, author.ID, "same-score-fewer-comments", baseTime.Add(2*time.Minute), model.VisiblePublic)
	olderTie := createHotRankPost(t, db, author.ID, "same-metrics-older", baseTime.Add(3*time.Minute), model.VisiblePublic)
	newerTie := createHotRankPost(t, db, author.ID, "same-metrics-newer", baseTime.Add(4*time.Minute), model.VisiblePublic)
	sameTimeLowerID := createHotRankPost(t, db, author.ID, "same-time-lower-id", baseTime.Add(5*time.Minute), model.VisiblePublic)
	sameTimeHigherID := createHotRankPost(t, db, author.ID, "same-time-higher-id", baseTime.Add(5*time.Minute), model.VisiblePublic)

	metrics := map[uint]struct {
		comments int
		likes    int
	}{
		leader.ID:           {comments: 4, likes: 4}, // score 4.000
		commentTieWinner.ID: {comments: 5, likes: 0}, // score 3.000
		commentTieLoser.ID:  {comments: 3, likes: 3}, // score 3.000
		olderTie.ID:         {comments: 1, likes: 1}, // score 1.000
		newerTie.ID:         {comments: 1, likes: 1}, // score 1.000
		sameTimeLowerID.ID:  {comments: 1, likes: 0}, // score 0.600
		sameTimeHigherID.ID: {comments: 1, likes: 0}, // score 0.600
	}
	for postID, metric := range metrics {
		addHotRankInteractions(t, db, postID, author.ID, metric.comments, metric.likes)
	}
	deletedComment := model.Comment{PostID: leader.ID, UserID: author.ID, Content: "soft-deleted-comment-must-not-count"}
	if err := db.Create(&deletedComment).Error; err != nil {
		t.Fatalf("create soft-deleted comment fixture: %v", err)
	}
	if err := db.Delete(&deletedComment).Error; err != nil {
		t.Fatalf("soft-delete comment fixture: %v", err)
	}

	w := doJSON(t, r, http.MethodGet, "/api/posts/hot", "", nil)
	response := decodeHotRankResponse(t, w)
	wantIDs := []uint{
		leader.ID,
		commentTieWinner.ID,
		commentTieLoser.ID,
		newerTie.ID,
		olderTie.ID,
		sameTimeHigherID.ID,
		sameTimeLowerID.ID,
	}
	gotIDs := make([]uint, len(response.Data.List))
	for i, item := range response.Data.List {
		gotIDs[i] = item.ID
		metric, ok := metrics[item.ID]
		if !ok {
			t.Fatalf("unexpected post in hot list: %+v", item)
		}
		if item.CommentCount != int64(metric.comments) || item.LikeCount != int64(metric.likes) {
			t.Fatalf("post %d counts=(%d,%d), want=(%d,%d)", item.ID, item.CommentCount, item.LikeCount, metric.comments, metric.likes)
		}

		wantScore := math.Trunc(float64(metric.comments*600+metric.likes*400)) / 1000
		if math.Abs(item.Score-wantScore) > 1e-9 {
			t.Fatalf("post %d score=%v, want=%v", item.ID, item.Score, wantScore)
		}
		// The public score must have no precision beyond three decimal places.
		if math.Abs(item.Score*1000-math.Trunc(item.Score*1000)) > 1e-9 {
			t.Fatalf("post %d score=%v has more than three decimal places", item.ID, item.Score)
		}
	}
	if !equalHotRankIDs(gotIDs, wantIDs) {
		t.Fatalf("hot list ids=%v, want=%v", gotIDs, wantIDs)
	}
	assertHotRankScoresAreThreeDecimalJSONNumbers(t, w.Body.Bytes(), metrics)
}

func TestAPI_HotPostsReturnsEmptyPartialAndTopTenLists(t *testing.T) {
	t.Run("empty list is an array", func(t *testing.T) {
		r, _ := setupTestRouter(t)
		w := doJSON(t, r, http.MethodGet, "/api/posts/hot", "", nil)
		response := decodeHotRankResponse(t, w)
		if response.Data.List == nil {
			t.Fatalf("empty hot list must be [] instead of null: %s", w.Body.String())
		}
		if len(response.Data.List) != 0 {
			t.Fatalf("empty hot list length=%d, want=0", len(response.Data.List))
		}
	})

	t.Run("fewer than ten posts are all returned", func(t *testing.T) {
		r, db := setupTestRouter(t)
		author := createHotRankAuthor(t, db)
		startYesterday, _ := hotRankWindow(time.Now())
		for i := 0; i < 3; i++ {
			createHotRankPost(t, db, author.ID, fmt.Sprintf("partial-%d", i), startYesterday.Add(time.Duration(i)*time.Minute), model.VisiblePublic)
		}

		response := decodeHotRankResponse(t, doJSON(t, r, http.MethodGet, "/api/posts/hot", "", nil))
		if len(response.Data.List) != 3 {
			t.Fatalf("partial hot list length=%d, want=3", len(response.Data.List))
		}
	})

	t.Run("more than ten posts are capped with stable id order", func(t *testing.T) {
		r, db := setupTestRouter(t)
		author := createHotRankAuthor(t, db)
		startYesterday, _ := hotRankWindow(time.Now())
		created := make([]model.Post, 0, 12)
		for i := 0; i < 12; i++ {
			created = append(created, createHotRankPost(t, db, author.ID, fmt.Sprintf("top-ten-%02d", i), startYesterday.Add(time.Hour), model.VisiblePublic))
		}

		response := decodeHotRankResponse(t, doJSON(t, r, http.MethodGet, "/api/posts/hot", "", nil))
		if len(response.Data.List) != 10 {
			t.Fatalf("hot list length=%d, want=10", len(response.Data.List))
		}
		for i, item := range response.Data.List {
			wantID := created[len(created)-1-i].ID
			if item.ID != wantID {
				t.Fatalf("hot list index=%d id=%d, want=%d", i, item.ID, wantID)
			}
		}
	})
}

func decodeHotRankResponse(t *testing.T, w *httptest.ResponseRecorder) hotRankAPIResponse {
	t.Helper()
	if w.Code != http.StatusOK {
		t.Fatalf("hot posts status=%d, want=200 body=%s", w.Code, w.Body.String())
	}
	var body hotRankAPIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode hot posts response: %v body=%s", err, w.Body.String())
	}
	if body.Code != 0 {
		t.Fatalf("hot posts code=%d, want=0 body=%s", body.Code, w.Body.String())
	}
	return body
}

func hotRankWindow(now time.Time) (time.Time, time.Time) {
	now = now.In(time.Local)
	startToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	return startToday.AddDate(0, 0, -1), startToday.AddDate(0, 0, 1)
}

func createHotRankAuthor(t *testing.T, db *gorm.DB) model.User {
	t.Helper()
	user := model.User{
		Username:     "hot-rank-author",
		Password:     hotRankPasswordSecret,
		Nickname:     "Hot Rank Author",
		Status:       model.UserStatusNormal,
		Introduction: "hot rank fixture author",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create hot-rank author: %v", err)
	}
	return user
}

func createHotRankPost(t *testing.T, db *gorm.DB, userID uint, title string, createdAt time.Time, visible uint8) model.Post {
	t.Helper()
	post := model.Post{
		BaseModel: model.BaseModel{
			CreateTime: createdAt,
			UpdateTime: createdAt,
		},
		UserID:  userID,
		Title:   title,
		Content: hotRankContentSecret + ":" + title,
		Visible: visible,
	}
	if err := db.Create(&post).Error; err != nil {
		t.Fatalf("create hot-rank post %q: %v", title, err)
	}
	return post
}

func addHotRankInteractions(t *testing.T, db *gorm.DB, postID, authorID uint, comments, likes int) {
	t.Helper()
	for i := 0; i < comments; i++ {
		comment := model.Comment{PostID: postID, UserID: authorID, Content: fmt.Sprintf("comment-%d", i)}
		if err := db.Create(&comment).Error; err != nil {
			t.Fatalf("create comment %d for post %d: %v", i, postID, err)
		}
	}
	for i := 0; i < likes; i++ {
		like := model.PostLike{PostID: postID, UserID: authorID + uint(i) + 1000}
		if err := db.Create(&like).Error; err != nil {
			t.Fatalf("create like %d for post %d: %v", i, postID, err)
		}
	}
}

func assertHotRankResponseHasNoPrivateFields(t *testing.T, body []byte) {
	t.Helper()
	var envelope struct {
		Data struct {
			List []map[string]any `json:"list"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("decode hot list field contract: %v", err)
	}
	for _, post := range envelope.Data.List {
		if _, exists := post["content"]; exists {
			t.Fatalf("hot list post exposes content: %v", post)
		}
		if _, exists := post["deleted"]; exists {
			t.Fatalf("hot list post exposes deleted marker: %v", post)
		}
		user, ok := post["user"].(map[string]any)
		if !ok {
			t.Fatalf("hot list post has invalid user response: %v", post)
		}
		if _, exists := user["password"]; exists {
			t.Fatalf("hot list user exposes password: %v", user)
		}
		if _, exists := user["deleted"]; exists {
			t.Fatalf("hot list user exposes deleted marker: %v", user)
		}
	}
}

func assertHotRankScoresAreThreeDecimalJSONNumbers(
	t *testing.T,
	body []byte,
	metrics map[uint]struct {
		comments int
		likes    int
	},
) {
	t.Helper()
	var envelope struct {
		Data struct {
			List []struct {
				ID    uint            `json:"id"`
				Score json.RawMessage `json:"score"`
			} `json:"list"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("decode raw hot score contract: %v", err)
	}
	for _, post := range envelope.Data.List {
		metric, ok := metrics[post.ID]
		if !ok {
			t.Fatalf("unexpected post %d in raw hot score contract", post.ID)
		}
		scoreMillis := int64(metric.comments*600 + metric.likes*400)
		want := fmt.Sprintf("%d.%03d", scoreMillis/1000, scoreMillis%1000)
		if got := string(post.Score); got != want {
			t.Fatalf("post %d raw score=%s, want JSON number %s", post.ID, got, want)
		}
	}
}

func equalHotRankIDs(got, want []uint) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
