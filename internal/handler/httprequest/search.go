package httprequest

import (
	"strconv"
	"strings"

	"my-bbs/internal/service"
	"my-bbs/pkg/bizerr"

	"github.com/gin-gonic/gin"
)

// SearchQuery is the HTTP representation of a global-search query.
// Keyword validation belongs to the application service so non-HTTP callers
// observe the same business rule.
type SearchQuery struct {
	Keyword  string
	Scope    string
	PageNo   int
	PageSize int
}

// BindSearchQuery strictly parses the search query string. Missing pagination
// values use documented defaults; malformed or excessively deep pagination is
// rejected instead of being silently normalized.
func BindSearchQuery(c *gin.Context) (SearchQuery, error) {
	pageNo, err := positiveQueryInt(c, "pageNo", service.DefaultSearchPageNo)
	if err != nil {
		return SearchQuery{}, err
	}
	pageSize, err := positiveQueryInt(c, "pageSize", service.DefaultSearchPageSize)
	if err != nil {
		return SearchQuery{}, err
	}
	if pageSize > service.MaxSearchPageSize {
		return SearchQuery{}, bizerr.ErrBadRequest.WithMessage(
			"字段 pageSize 不能超过 " + strconv.Itoa(service.MaxSearchPageSize),
		)
	}
	if pageNo-1 > service.MaxSearchOffset/pageSize {
		return SearchQuery{}, bizerr.ErrBadRequest.WithMessage(
			"分页位置不能超过 " + strconv.Itoa(service.MaxSearchOffset) + " 条",
		)
	}

	scope := string(service.SearchScopeAll)
	if raw, exists := c.GetQuery("scope"); exists {
		scope = strings.ToLower(strings.TrimSpace(raw))
	}

	return SearchQuery{
		Keyword:  c.Query("q"),
		Scope:    scope,
		PageNo:   pageNo,
		PageSize: pageSize,
	}, nil
}
