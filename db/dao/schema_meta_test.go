package dao

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/db/entities"
)

type TestEntity struct {
	ID     string  `json:"id" db:"id"`
	Name   *string `json:"name" db:"name"`
	Ignore string  `json:"ignore" db:"-"`
	entities.BaseModel
}

func (m TestEntity) PrimaryKey() string {
	return m.ID
}

func TestSchemaMeta(t *testing.T) {
	schema := NewSchema[TestEntity]("test_schema")

	assert.Equal(t, "test_schema", schema.Name)
	assert.EqualValues(t, []string{"id", "name", "ws_id"}, schema.InsertColumns())
	assert.EqualValues(t, []string{"created_at", "id", "name", "updated_at", "ws_id"}, schema.Columns())
	assert.Equal(t, false, schema.HasColumn("foo"))
}
