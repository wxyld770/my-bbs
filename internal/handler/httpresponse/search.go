package httpresponse

import (
	"time"

	"my-bbs/internal/model"
	"my-bbs/internal/service"
)

// SearchResponse keeps user and post results in separate groups. This makes
// relevance and pagination explicit for each resource type.
type SearchResponse struct {
	Query string                                  `json:"query"`
	Scope string                                  `json:"scope"`
	Users SearchGroupResponse[SearchUserResponse] `json:"users"`
	Posts SearchGroupResponse[SearchPostResponse] `json:"posts"`
}

type SearchGroupResponse[T any] struct {
	List     []T  `json:"list"`
	PageNo   int  `json:"pageNo"`
	PageSize int  `json:"pageSize"`
	HasMore  bool `json:"hasMore"`
}

type SearchUserResponse struct {
	ID           uint   `json:"id"`
	Username     string `json:"username"`
	Nickname     string `json:"nickname"`
	Introduction string `json:"introduction"`
}

type SearchUserCompactResponse struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
}

type SearchPostResponse struct {
	ID           uint                       `json:"id"`
	CreateTime   time.Time                  `json:"create_time"`
	UserID       uint                       `json:"user_id"`
	Title        string                     `json:"title"`
	Excerpt      string                     `json:"excerpt"`
	User         *SearchUserCompactResponse `json:"user"`
	LikeCount    int64                      `json:"like_count"`
	CommentCount int64                      `json:"comment_count"`
}

func NewSearchResponse(result service.SearchResult) SearchResponse {
	return SearchResponse{
		Query: result.Query,
		Scope: string(result.Scope),
		Users: newSearchGroupResponse(result.Users, newSearchUserResponse),
		Posts: newSearchGroupResponse(result.Posts, newSearchPostResponse),
	}
}

func newSearchGroupResponse[S, T any](source service.SearchPage[S], convert func(S) T) SearchGroupResponse[T] {
	list := make([]T, len(source.List))
	for i, item := range source.List {
		list[i] = convert(item)
	}
	return SearchGroupResponse[T]{
		List:     list,
		PageNo:   source.PageNo,
		PageSize: source.PageSize,
		HasMore:  source.HasMore,
	}
}

func newSearchUserResponse(user model.User) SearchUserResponse {
	return SearchUserResponse{
		ID:           user.ID,
		Username:     user.Username,
		Nickname:     user.Nickname,
		Introduction: user.Introduction,
	}
}

func newSearchPostResponse(result service.SearchPost) SearchPostResponse {
	return SearchPostResponse{
		ID:           result.Post.ID,
		CreateTime:   result.Post.CreateTime,
		UserID:       result.Post.UserID,
		Title:        result.Post.Title,
		Excerpt:      result.Excerpt,
		User:         newSearchUserCompactResponse(result.Post.User),
		LikeCount:    result.LikeCount,
		CommentCount: result.CommentCount,
	}
}

func newSearchUserCompactResponse(user *model.User) *SearchUserCompactResponse {
	if user == nil {
		return nil
	}
	return &SearchUserCompactResponse{
		ID:       user.ID,
		Username: user.Username,
		Nickname: user.Nickname,
	}
}
