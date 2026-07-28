package set

// Set 泛型集合，基于 map 实现去重
type Set[T comparable] struct {
	items map[T]struct{}
}

// New 创建空集合；可选传入初始容量提示（仅预分配，map 仍会按需自动扩容）
func New[T comparable](capHint ...int) *Set[T] {
	n := 0
	if len(capHint) > 0 && capHint[0] > 0 {
		n = capHint[0]
	}
	return &Set[T]{items: make(map[T]struct{}, n)}
}

// FromSlice 从切片创建集合（自动去重）
func FromSlice[T comparable](items []T) *Set[T] {
	s := New[T](len(items))
	s.Add(items...)
	return s
}

// Add 添加一个或多个元素（底层 map 自动扩容，无需手动扩容）
func (s *Set[T]) Add(items ...T) {
	for _, item := range items {
		s.items[item] = struct{}{}
	}
}

// Remove 删除元素
func (s *Set[T]) Remove(item T) {
	delete(s.items, item)
}

// Contains 检查元素是否存在
func (s *Set[T]) Contains(item T) bool {
	_, ok := s.items[item]
	return ok
}

// Size 返回集合大小
func (s *Set[T]) Size() int {
	return len(s.items)
}

// ToSlice 转换为切片（顺序不稳定）
func (s *Set[T]) ToSlice() []T {
	result := make([]T, 0, len(s.items))
	for key := range s.items {
		result = append(result, key)
	}
	return result
}
