package service

import (
	"context"
	"strings"
	"sync"
	"unicode/utf8"

	"my-bbs/internal/model"
	"my-bbs/internal/repository"
	"my-bbs/pkg/bizerr"
	"my-bbs/pkg/set"
)

const (
	SearchScopeAll   SearchScope = "all"
	SearchScopeUsers SearchScope = "users"
	SearchScopePosts SearchScope = "posts"

	// Search pagination uses a tighter boundary than ordinary feed pagination
	// because V1 performs literal contains queries against post content.
	DefaultSearchPageNo   = 1
	DefaultSearchPageSize = 10
	MaxSearchPageSize     = 20
	MaxSearchOffset       = 1000

	maxSearchExcerptRunes = 120
)

type SearchScope string

// SearchQuery 是搜索应用服务的输入，不依赖具体 HTTP 框架。
type SearchQuery struct {
	Keyword  string
	Scope    SearchScope
	PageNo   int
	PageSize int
}

// SearchPage 表示一类搜索结果的独立分页信息。
type SearchPage[T any] struct {
	List     []T
	PageNo   int
	PageSize int
	HasMore  bool
}

// SearchPost 是搜索场景的帖子摘要。
type SearchPost struct {
	Post         model.Post
	Excerpt      string
	LikeCount    int64
	CommentCount int64
}

// SearchResult 将不同资源分组返回，避免用户与帖子使用含义不清的混合分页。
type SearchResult struct {
	Query string
	Scope SearchScope
	Users SearchPage[model.User]
	Posts SearchPage[SearchPost]
}

type SearchService struct {
	searchRepo  repository.SearchReader
	userRepo    repository.UserReader
	commentRepo repository.CommentCounter
	likeRepo    repository.LikeReader
}

func NewSearchService(
	searchRepo repository.SearchReader,
	userRepo repository.UserReader,
	commentRepo repository.CommentCounter,
	likeRepo repository.LikeReader,
) *SearchService {
	return &SearchService{
		searchRepo:  searchRepo,
		userRepo:    userRepo,
		commentRepo: commentRepo,
		likeRepo:    likeRepo,
	}
}

// Search 搜索用户和公开帖子。all 场景下两类基础查询并行执行，且共享由
// 调用方传入的 Context；任一查询失败会取消另一查询。
func (s *SearchService) Search(ctx context.Context, query SearchQuery) (SearchResult, error) {
	normalized, err := normalizeSearchQuery(query)
	if err != nil {
		return SearchResult{}, err
	}

	result := newEmptySearchResult(normalized)
	offset := (normalized.PageNo - 1) * normalized.PageSize
	limit := normalized.PageSize + 1

	var users []model.User
	var posts []model.Post
	switch normalized.Scope {
	case SearchScopeUsers:
		users, err = s.searchRepo.SearchUsers(ctx, normalized.Keyword, offset, limit)
	case SearchScopePosts:
		posts, err = s.searchRepo.SearchPublicPosts(ctx, normalized.Keyword, offset, limit)
	case SearchScopeAll:
		users, posts, err = s.searchAll(ctx, normalized.Keyword, offset, limit)
	}
	if err != nil {
		return SearchResult{}, err
	}

	if normalized.Scope != SearchScopePosts {
		result.Users = searchPage(users, normalized.PageNo, normalized.PageSize)
	}
	if normalized.Scope != SearchScopeUsers {
		postPage := searchPage(posts, normalized.PageNo, normalized.PageSize)
		postResults, err := s.summarizeSearchPosts(ctx, postPage.List, normalized.Keyword)
		if err != nil {
			return SearchResult{}, err
		}
		result.Posts = SearchPage[SearchPost]{
			List:     postResults,
			PageNo:   postPage.PageNo,
			PageSize: postPage.PageSize,
			HasMore:  postPage.HasMore,
		}
	}

	return result, nil
}

func normalizeSearchQuery(query SearchQuery) (SearchQuery, error) {
	if !utf8.ValidString(query.Keyword) {
		return SearchQuery{}, bizerr.ErrBadRequest.WithMessage("搜索关键词不是有效的 UTF-8 文本")
	}
	query.Keyword = strings.Join(strings.Fields(query.Keyword), " ")
	keywordRunes := utf8.RuneCountInString(query.Keyword)
	if keywordRunes < 2 {
		return SearchQuery{}, bizerr.ErrBadRequest.WithMessage("搜索关键词长度不能少于2个字符")
	}
	if keywordRunes > 50 {
		return SearchQuery{}, bizerr.ErrBadRequest.WithMessage("搜索关键词长度不能超过50个字符")
	}
	if len(query.Keyword) > 200 {
		return SearchQuery{}, bizerr.ErrBadRequest.WithMessage("搜索关键词不能超过200字节")
	}

	query.Scope = SearchScope(strings.ToLower(strings.TrimSpace(string(query.Scope))))
	if query.Scope == "" {
		query.Scope = SearchScopeAll
	}
	switch query.Scope {
	case SearchScopeAll, SearchScopeUsers, SearchScopePosts:
	default:
		return SearchQuery{}, bizerr.ErrBadRequest.WithMessage("scope 仅支持 all、users 或 posts")
	}

	if query.PageNo <= 0 {
		query.PageNo = DefaultSearchPageNo
	}
	if query.PageSize <= 0 {
		query.PageSize = DefaultSearchPageSize
	}
	if query.PageSize > MaxSearchPageSize {
		return SearchQuery{}, bizerr.ErrBadRequest.WithMessagef("pageSize 不能超过%d", MaxSearchPageSize)
	}
	// 先除后乘，避免不可信 pageNo 导致整数溢出。
	if query.PageNo-1 > MaxSearchOffset/query.PageSize {
		return SearchQuery{}, bizerr.ErrBadRequest.WithMessagef("搜索分页偏移不能超过%d", MaxSearchOffset)
	}

	return query, nil
}

func newEmptySearchResult(query SearchQuery) SearchResult {
	return SearchResult{
		Query: query.Keyword,
		Scope: query.Scope,
		Users: SearchPage[model.User]{
			List:     []model.User{},
			PageNo:   query.PageNo,
			PageSize: query.PageSize,
		},
		Posts: SearchPage[SearchPost]{
			List:     []SearchPost{},
			PageNo:   query.PageNo,
			PageSize: query.PageSize,
		},
	}
}

func searchPage[T any](rows []T, pageNo, pageSize int) SearchPage[T] {
	// Do not advertise a page whose offset would exceed the search boundary.
	hasExtraRow := len(rows) > pageSize
	hasMore := hasExtraRow && pageNo <= MaxSearchOffset/pageSize
	if hasExtraRow {
		rows = rows[:pageSize]
	}
	if rows == nil {
		rows = []T{}
	}
	return SearchPage[T]{
		List:     rows,
		PageNo:   pageNo,
		PageSize: pageSize,
		HasMore:  hasMore,
	}
}

func (s *SearchService) searchAll(ctx context.Context, keyword string, offset, limit int) ([]model.User, []model.Post, error) {
	searchCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var users []model.User
	var posts []model.Post
	var firstErr error
	var errOnce sync.Once
	var wait sync.WaitGroup
	wait.Add(2)

	recordError := func(err error) {
		if err == nil {
			return
		}
		errOnce.Do(func() {
			firstErr = err
			cancel()
		})
	}

	go func() {
		defer wait.Done()
		var err error
		users, err = s.searchRepo.SearchUsers(searchCtx, keyword, offset, limit)
		recordError(err)
	}()
	go func() {
		defer wait.Done()
		var err error
		posts, err = s.searchRepo.SearchPublicPosts(searchCtx, keyword, offset, limit)
		recordError(err)
	}()

	wait.Wait()
	return users, posts, firstErr
}

// summarizeSearchPosts 固定使用三次批量查询填充作者、点赞数和评论数，
// 查询次数不会随结果条数增长。
func (s *SearchService) summarizeSearchPosts(ctx context.Context, posts []model.Post, keyword string) ([]SearchPost, error) {
	if len(posts) == 0 {
		return []SearchPost{}, nil
	}

	userIDs := make([]uint, len(posts))
	postIDs := make([]uint, len(posts))
	for i := range posts {
		userIDs[i] = posts[i].UserID
		postIDs[i] = posts[i].ID
	}

	users, err := s.userRepo.FindUsersByIDs(ctx, set.FromSlice(userIDs).ToSlice())
	if err != nil {
		return nil, err
	}
	likeCounts, err := s.likeRepo.CountByPostIDs(ctx, postIDs)
	if err != nil {
		return nil, err
	}
	commentCounts, err := s.commentRepo.CountByPostIDs(ctx, postIDs)
	if err != nil {
		return nil, err
	}

	usersByID := make(map[uint]*model.User, len(users))
	for i := range users {
		usersByID[users[i].ID] = &users[i]
	}

	results := make([]SearchPost, len(posts))
	for i := range posts {
		posts[i].User = usersByID[posts[i].UserID]
		results[i] = SearchPost{
			Post:         posts[i],
			Excerpt:      searchExcerpt(posts[i].Content, keyword),
			LikeCount:    likeCounts[posts[i].ID],
			CommentCount: commentCounts[posts[i].ID],
		}
	}
	return results, nil
}

func searchExcerpt(content, keyword string) string {
	// Post content is stored and rendered as plain text. Keep literal symbols
	// such as <T> or Markdown markers intact, collapse whitespace, and let the
	// HTTP/React layers escape the resulting JSON/text normally.
	plain := strings.Join(strings.Fields(content), " ")

	runes := []rune(plain)
	start := 0
	lowerPlain := strings.ToLower(plain)
	if matchAt := strings.Index(lowerPlain, strings.ToLower(keyword)); matchAt >= 0 {
		matchRune := utf8.RuneCountInString(lowerPlain[:matchAt])
		const contextBeforeMatch = 30
		if matchRune > contextBeforeMatch {
			start = matchRune - contextBeforeMatch
		}
	}
	end := start + maxSearchExcerptRunes
	if end > len(runes) {
		end = len(runes)
	}
	return string(runes[start:end])
}
