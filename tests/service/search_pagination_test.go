package service_test

import (
	"context"
	"testing"

	"my-bbs/internal/model"
	"my-bbs/internal/service"
)

type boundarySearchReader struct {
	offset int
	limit  int
}

func (r *boundarySearchReader) SearchUsers(_ context.Context, _ string, offset, limit int) ([]model.User, error) {
	r.offset = offset
	r.limit = limit
	users := make([]model.User, limit)
	for i := range users {
		users[i].ID = uint(i + 1)
	}
	return users, nil
}

func (*boundarySearchReader) SearchPublicPosts(context.Context, string, int, int) ([]model.Post, error) {
	return []model.Post{}, nil
}

func TestSearchService_DoesNotAdvertisePageBeyondMaximumOffset(t *testing.T) {
	reader := &boundarySearchReader{}
	svc := service.NewSearchService(reader, nil, nil, nil)

	result, err := svc.Search(context.Background(), service.SearchQuery{
		Keyword:  "go",
		Scope:    service.SearchScopeUsers,
		PageNo:   101,
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("Search() error=%v", err)
	}
	if reader.offset != 1000 || reader.limit != 11 {
		t.Fatalf("repository offset/limit=%d/%d, want=1000/11", reader.offset, reader.limit)
	}
	if len(result.Users.List) != 10 {
		t.Fatalf("result length=%d, want=10", len(result.Users.List))
	}
	if result.Users.HasMore {
		t.Fatal("HasMore=true would advertise a page beyond the allowed offset")
	}
}
