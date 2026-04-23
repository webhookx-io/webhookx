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
	Prev *string `json:"prev"`
}

func BuildPaginationResponse[T any](cursor dao.Cursor[T], url *url.URL) interface{} {
	if !cursor.Cursor {
		return Pagination[T]{
			Total: cursor.Total,
			Data:  cursor.Data,
		}

	}

	var hasNext, hasPrev bool
	var next, prev *string


	if cursor.Reversed {
		hasPrev = cursor.HasMore
		hasNext = url.Query().Get("before") != ""
	} else {
		hasNext = cursor.HasMore
		hasPrev = url.Query().Get("after") != ""
	}

	if hasNext && cursor.LastId != nil {
		values := url.Query()
		values.Del("before")
		values.Set("after", *cursor.LastId)
		next = new(url.Path + "?" + values.Encode())
	}

	if hasPrev && cursor.FirstId != nil {
		values := url.Query()
		values.Del("after")
		values.Set("before", *cursor.FirstId)
		prev = new(url.Path + "?" + values.Encode())
	}

	return CursorPagination[T]{
		Data: cursor.Data,
		Next: next,
		Prev: prev,
	}
}
