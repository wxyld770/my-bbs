package pagination

const (
	DefaultPageNo   = 1
	DefaultPageSize = 10
	MaxPageSize     = 50
	// MaxOffset limits offset-based pagination to a bounded database scan.
	// Requests whose first row would start after this offset are rejected by
	// the HTTP binding layer; Normalize applies the same bound defensively for
	// non-HTTP callers.
	MaxOffset = 5000
)

// Query 分页查询参数（适合无限下拉）
type Query struct {
	PageNo   int `form:"pageNo" json:"pageNo"`
	PageSize int `form:"pageSize" json:"pageSize"`
}

// Normalize 校正非法分页参数
func (q *Query) Normalize() {
	if q.PageNo < 1 {
		q.PageNo = DefaultPageNo
	}
	if q.PageSize < 1 {
		q.PageSize = DefaultPageSize
	}
	if q.PageSize > MaxPageSize {
		q.PageSize = MaxPageSize
	}
	maxPageNo := MaxOffset/q.PageSize + 1
	if q.PageNo > maxPageNo {
		q.PageNo = maxPageNo
	}
}

// IsOffsetAllowed reports whether calculating the SQL offset is safe and the
// result stays within MaxOffset. The division-first check avoids integer
// overflow even when PageNo came from an untrusted query string.
func (q Query) IsOffsetAllowed() bool {
	return q.PageNo > 0 && q.PageSize > 0 && q.PageNo-1 <= MaxOffset/q.PageSize
}

// Offset 计算 SQL offset；先归一化以确保非 HTTP 调用也不会产生过深或溢出的 offset。
func (q Query) Offset() int {
	q.Normalize()
	return (q.PageNo - 1) * q.PageSize
}

// Result 分页结果：用 hasMore 判断是否还能继续下拉，不做 total count
type Result[T any] struct {
	List     []T  `json:"list"`
	PageNo   int  `json:"pageNo"`
	PageSize int  `json:"pageSize"`
	HasMore  bool `json:"hasMore"`
}

// NewResult 构造分页结果；本页条数 == pageSize 时认为可能还有下一页
func NewResult[T any](list []T, q Query) Result[T] {
	q.Normalize()
	if list == nil {
		list = []T{}
	}
	// Do not advertise a next page that BindPageQuery will necessarily reject
	// for crossing MaxOffset.
	hasMore := len(list) >= q.PageSize && q.PageNo <= MaxOffset/q.PageSize
	return Result[T]{
		List:     list,
		PageNo:   q.PageNo,
		PageSize: q.PageSize,
		HasMore:  hasMore,
	}
}
