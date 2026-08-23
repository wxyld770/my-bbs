package handler

import (
	"context"
	"errors"
	"time"

	httpreq "my-bbs/internal/handler/httprequest"
	httpresp "my-bbs/internal/handler/httpresponse"
	"my-bbs/internal/service"
	"my-bbs/pkg/bizerr"
	"my-bbs/pkg/response"

	"github.com/gin-gonic/gin"
)

const defaultSearchTimeout = time.Second

type SearchHandler struct {
	searchService *service.SearchService
	timeout       time.Duration
}

func NewSearchHandler(searchService *service.SearchService, timeout time.Duration) *SearchHandler {
	if timeout <= 0 {
		timeout = defaultSearchTimeout
	}
	return &SearchHandler{searchService: searchService, timeout: timeout}
}

func (h *SearchHandler) Search(c *gin.Context) {
	req, err := httpreq.BindSearchQuery(c)
	if err != nil {
		response.ReportError(c, err)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()
	result, err := h.searchService.Search(ctx, service.SearchQuery{
		Keyword:  req.Keyword,
		Scope:    service.SearchScope(req.Scope),
		PageNo:   req.PageNo,
		PageSize: req.PageSize,
	})
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			err = errors.Join(bizerr.ErrServiceUnavailable, err)
		}
		response.ReportError(c, err)
		return
	}

	response.OK(c, httpresp.NewSearchResponse(result))
}
