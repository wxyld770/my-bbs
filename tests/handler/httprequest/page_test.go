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
		{name: "invalid number", rawQuery: "pageNo=abc", message: "字段 pageNo 必须是正整数"},
		{name: "non-positive", rawQuery: "pageSize=0", message: "字段 pageSize 必须是正整数"},
		{name: "above maximum", rawQuery: "pageSize=51", message: "字段 pageSize 不能超过 50"},
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
