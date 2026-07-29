package pagination

const (
	DefaultPageNo   = 1
	DefaultPageSize = 10
	MaxPageSize     = 50
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
}

// Offset 计算 SQL offset
func (q Query) Offset() int {
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
	if list == nil {
		list = []T{}
	}
	return Result[T]{
		List:     list,
		PageNo:   q.PageNo,
		PageSize: q.PageSize,
		HasMore:  len(list) >= q.PageSize,
	}
}
