package api

import (
	"net/url"

	"github.com/webhookx-io/webhookx/db/dao"
)

type Pagination[T any] struct {
	Total int64 `json:"total"`
	Data  []T   `json:"data"`
}

type CursorPagination[T any] struct {
	Data []T     `json:"data"`
	Next *string `json:"next"`
}

func BuildPaginationResponse[T any](cursor bool, result dao.CursorResult[T], url *url.URL) interface{} {
	if !cursor {
		return Pagination[T]{
			Total: result.Total,
			Data:  result.Data,
		}

	}

	var next *string

	if result.Cursor.HasMore {
		values := url.Query()
		values.Set("after", *result.Cursor.LastId)
		next = new(url.Path + "?" + values.Encode())
	}

	return CursorPagination[T]{
		Data: result.Data,
		Next: next,
	}
}
