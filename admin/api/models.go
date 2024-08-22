package api

type Pagination[T any] struct {
	Total int64 `json:"total"`
	Data  []T   `json:"data"`
}

func NewPagination[T any](total int64, data []T) *Pagination[T] {
	return &Pagination[T]{
		Total: total,
		Data:  data,
	}
}
