package api

import "github.com/webhookx-io/webhookx/db/dao"

type Pagination[T any] struct {
	Total int64 `json:"total"`
	Data  []T   `json:"data"`
}

type PaginationCursor[T any] struct {
	Next *string `json:"next"`
	Data []T     `json:"data"`
}

func NewPagination[T any](q *dao.Query, cursor dao.CursorResult[T]) interface{} {
	if q.CursorModel {
		return PaginationCursor[T]{
			Next: cursor.Cursor,
			Data: cursor.Data,
		}
	}
	return Pagination[T]{
		Total: cursor.Total,
		Data:  cursor.Data,
	}
}
