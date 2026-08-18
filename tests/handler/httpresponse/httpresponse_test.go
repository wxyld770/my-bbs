package httpresponse_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	httpresp "my-bbs/internal/handler/httpresponse"
	"my-bbs/internal/model"
	"my-bbs/internal/service"
	"my-bbs/pkg/pagination"
)

func TestUserProfileResponse_DoesNotExposePersistenceFields(t *testing.T) {
	now := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	user := &model.User{
		BaseModel: model.BaseModel{ID: 7, CreateTime: now, UpdateTime: now},
		Username:  "alice",
		Password:  "super-secret-hash",
		Nickname:  "Alice",
		Status:    model.UserStatusNormal,
	}

	body, err := json.Marshal(httpresp.NewUserProfileResponse(user))
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	jsonText := string(body)
	if strings.Contains(jsonText, "super-secret-hash") || strings.Contains(jsonText, "deleted") {
		t.Fatalf("response leaked persistence fields: %s", jsonText)
	}
	for _, field := range []string{`"id":7`, `"username":"alice"`, `"create_time"`, `"update_time"`} {
		if !strings.Contains(jsonText, field) {
			t.Fatalf("response missing %s: %s", field, jsonText)
		}
	}
}

func TestPostResponses_PreserveHTTPContract(t *testing.T) {
	user := &model.User{BaseModel: model.BaseModel{ID: 2}, Username: "author"}
	post := model.Post{
		BaseModel: model.BaseModel{ID: 9},
		UserID:    user.ID,
		Title:     "hello",
		Content:   "world",
		Visible:   model.VisiblePublic,
		User:      user,
	}

	detail := httpresp.NewPostDetailResponse(&service.PostDetail{
		Post:         post,
		LikeCount:    3,
		CommentCount: 4,
		IsLiked:      true,
	})
	if detail.Post.ID != post.ID || detail.Post.User == nil || detail.Post.User.Username != "author" {
		t.Fatalf("unexpected post detail response: %+v", detail)
	}
	if detail.LikeCount != 3 || detail.CommentCount != 4 || !detail.IsLiked {
		t.Fatalf("unexpected interaction fields: %+v", detail)
	}

	page := httpresp.NewPostPageResponse(pagination.Result[model.Post]{
		List:     []model.Post{post},
		PageNo:   2,
		PageSize: 10,
		HasMore:  true,
	})
	if len(page.List) != 1 || page.List[0].ID != post.ID {
		t.Fatalf("unexpected page list: %+v", page)
	}
	if page.PageNo != 2 || page.PageSize != 10 || !page.HasMore {
		t.Fatalf("unexpected page metadata: %+v", page)
	}
}

func TestPageResponse_UsesEmptyArrayForNilList(t *testing.T) {
	page := httpresp.NewCommentPageResponse(pagination.Result[model.Comment]{
		PageNo:   1,
		PageSize: 10,
	})
	body, err := json.Marshal(page)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	if !strings.Contains(string(body), `"list":[]`) {
		t.Fatalf("list should be an empty JSON array: %s", body)
	}
}
