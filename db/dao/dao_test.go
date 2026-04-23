package dao

import (
	"context"
	"maps"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/db/entities"
)

type TestEntity struct {
	ID   string  `json:"id" db:"id"`
	Name *string `json:"name" db:"name"`
	entities.BaseModel
}

func TestNewDAO(t *testing.T) {
	dao := NewDAO[TestEntity](nil, Options{
		Table: "test_table",
	})

	assert.Equal(t, "test_table", dao.opts.Table)
	assert.Equal(t, []string{"id", "name", "ws_id"}, dao.insertionColumns)

	assert.Equal(t, 5, len(dao.columns))
	columns := slices.Collect(maps.Keys(dao.columns))
	slices.Sort(columns)
	assert.EqualValues(t, []string{"created_at", "id", "name", "updated_at", "ws_id"}, columns)
}

func TestList(t *testing.T) {
	dao := NewDAO[TestEntity](nil, Options{
		Table: "test_table",
	})

	t.Run("should panic when query is nil", func(t *testing.T) {
		defer func() {
			if r := recover(); r != "query is nil" {
				t.Errorf("unexpected panic: %v", r)
			}
		}()
		dao.List(context.TODO(), nil)
	})
}

func TestCursor(t *testing.T) {
	dao := NewDAO[TestEntity](nil, Options{
		Table: "test_table",
	})

	t.Run("should panic when query is nil", func(t *testing.T) {
		defer func() {
			if r := recover(); r != "query is nil" {
				t.Errorf("unexpected panic: %v", r)
			}
		}()
		dao.Cursor(context.TODO(), nil)
	})

	t.Run("should panic when query.limit is negative", func(t *testing.T) {
		defer func() {
			if r := recover(); r != "query.limit must be positive" {
				t.Errorf("unexpected panic: %v", r)
			}
		}()
		query := Query{
			Limit: -1,
		}
		dao.Cursor(context.TODO(), &query)
	})
}

func TestCount(t *testing.T) {
	dao := NewDAO[TestEntity](nil, Options{
		Table: "test_table",
	})

	t.Run("should panic when query is nil", func(t *testing.T) {
		defer func() {
			if r := recover(); r != "query is nil" {
				t.Errorf("unexpected panic: %v", r)
			}
		}()
		dao.Count(context.TODO(), nil)
	})

}
