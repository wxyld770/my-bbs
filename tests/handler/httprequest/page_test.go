package httprequest_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	httprequest "my-bbs/internal/handler/httprequest"
	"my-bbs/pkg/bizerr"
	"my-bbs/pkg/pagination"

	"github.com/gin-gonic/gin"
)

func TestBindPageQuery(t *testing.T) {
	tests := []struct {
		name     string
		rawQuery string
		want     pagination.Query
		message  string
	}{
		{name: "defaults", want: pagination.Query{PageNo: 1, PageSize: 10}},
		{name: "values", rawQuery: "pageNo=2&pageSize=20", want: pagination.Query{PageNo: 2, PageSize: 20}},
		{name: "maximum offset", rawQuery: "pageNo=501&pageSize=10", want: pagination.Query{PageNo: 501, PageSize: 10}},
		{name: "invalid number", rawQuery: "pageNo=abc", message: "字段 pageNo 必须是正整数"},
		{name: "non-positive", rawQuery: "pageSize=0", message: "字段 pageSize 必须是正整数"},
		{name: "above maximum", rawQuery: "pageSize=51", message: "字段 pageSize 不能超过 50"},
		{name: "offset too deep", rawQuery: "pageNo=502&pageSize=10", message: "分页位置不能超过 5000 条"},
		{name: "large page number safely rejected", rawQuery: "pageNo=2147483647&pageSize=50", message: "分页位置不能超过 5000 条"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodGet, "/?"+tt.rawQuery, nil)

			got, err := httprequest.BindPageQuery(c)
			if tt.message == "" {
				if err != nil || got != tt.want {
					t.Fatalf("BindPageQuery() = (%+v, %v), want (%+v, nil)", got, err, tt.want)
				}
				return
			}
			assertBizError(t, err, http.StatusBadRequest, bizerr.ErrBadRequest.Code, tt.message)
		})
	}
}

func TestPaginationResultStopsAtMaximumOffset(t *testing.T) {
	fullPage := make([]int, 10)

	beforeBoundary := pagination.NewResult(fullPage, pagination.Query{PageNo: 500, PageSize: 10})
	if !beforeBoundary.HasMore {
		t.Fatal("page before maximum offset should still advertise the final allowed page")
	}

	atBoundary := pagination.NewResult(fullPage, pagination.Query{PageNo: 501, PageSize: 10})
	if atBoundary.HasMore {
		t.Fatal("page at maximum offset must not advertise a rejected next page")
	}
}
