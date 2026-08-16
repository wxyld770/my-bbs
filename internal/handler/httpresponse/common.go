package httpresponse

import "my-bbs/pkg/pagination"

// PageResponse 是 HTTP 分页响应模型，不直接暴露应用层的 pagination.Result。
type PageResponse[T any] struct {
	List     []T  `json:"list"`
	PageNo   int  `json:"pageNo"`
	PageSize int  `json:"pageSize"`
	HasMore  bool `json:"hasMore"`
}

func newPageResponse[S, T any](source pagination.Result[S], convert func(S) T) PageResponse[T] {
	list := make([]T, len(source.List))
	for i, item := range source.List {
		list[i] = convert(item)
	}
	return PageResponse[T]{
		List:     list,
		PageNo:   source.PageNo,
		PageSize: source.PageSize,
		HasMore:  source.HasMore,
	}
}

type HealthResponse struct {
	Status string `json:"status"`
}

func HealthOK() HealthResponse {
	return HealthResponse{Status: "ok"}
}

func HealthUnavailable() HealthResponse {
	return HealthResponse{Status: "unavailable"}
}
