package httprequest

import (
	"strconv"
	"strings"

	"my-bbs/pkg/pagination"

	"github.com/gin-gonic/gin"
)

// BindPageQuery parses pagination query parameters without silently accepting
// malformed numbers. Missing values use the documented defaults.
func BindPageQuery(c *gin.Context) (pagination.Query, error) {
	pageNo, err := positiveQueryInt(c, "pageNo", pagination.DefaultPageNo)
	if err != nil {
		return pagination.Query{}, err
	}
	pageSize, err := positiveQueryInt(c, "pageSize", pagination.DefaultPageSize)
	if err != nil {
		return pagination.Query{}, err
	}
	if pageSize > pagination.MaxPageSize {
		return pagination.Query{}, badRequest("字段 pageSize 不能超过 " + strconv.Itoa(pagination.MaxPageSize))
	}
	return pagination.Query{PageNo: pageNo, PageSize: pageSize}, nil
}

func positiveQueryInt(c *gin.Context, name string, defaultValue int) (int, error) {
	raw, exists := c.GetQuery(name)
	if !exists {
		return defaultValue, nil
	}
	raw = strings.TrimSpace(raw)
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0, badRequest("字段 " + name + " 必须是正整数")
	}
	return value, nil
}
