package dao

import (
	"maps"
	"reflect"
	"slices"
)


type SchemaMeta struct {
	Name          string
	insertColumns []string
	columns       map[string]bool
}

func NewSchema[T any](name string) *SchemaMeta {
	schema := &SchemaMeta{
		Name:    name,
		columns: make(map[string]bool),
	}

	EachField(new(T), func(f reflect.StructField, _ reflect.Value, column string) {
		schema.columns[column] = true
		if column != "created_at" && column != "updated_at" {
			schema.insertColumns = append(schema.insertColumns, column)
		}
	})

	return schema
}

func (s *SchemaMeta) HasColumn(column string) bool {
	return s.columns[column]
}

func (s *SchemaMeta) InsertColumns() []string {
	return s.insertColumns
}

func (s *SchemaMeta) Columns() []string {
	columns := slices.Collect(maps.Keys(s.columns))
	slices.Sort(columns)
	return columns
}
