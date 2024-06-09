package dao

import (
	"context"
	"database/sql"
	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/webhookx-io/webhookx/internal/db/entities"
	"github.com/webhookx-io/webhookx/internal/db/query"
	"github.com/webhookx-io/webhookx/internal/utils"
	"reflect"
	"strings"
	"time"
)

var ErrNoRows = sql.ErrNoRows
var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

type DAO[T any] struct {
	db    *sqlx.DB
	table string
}

func NewDAO[T any](table string, db *sqlx.DB) *DAO[T] {
	dao := DAO[T]{
		db:    db,
		table: table,
	}
	return &dao
}

func (dao *DAO[T]) Get(ctx context.Context, id string) (entity *T, err error) {
	if ok := utils.IsValidUUID(id); !ok {
		return nil, nil
	}
	statement, args := psql.Select("*").From(dao.table).Where(sq.Eq{"id": id}).MustSql()
	entity = new(T)
	err = dao.db.Unsafe().GetContext(ctx, entity, statement, args...)
	if errors.Is(err, ErrNoRows) {
		return nil, nil
	}
	return
}

func (dao *DAO[T]) Delete(ctx context.Context, id string) (bool, error) {
	if ok := utils.IsValidUUID(id); !ok {
		return false, nil
	}
	statement, args := psql.Delete(dao.table).Where(sq.Eq{"id": id}).
		MustSql()
	result, err := dao.db.ExecContext(ctx, statement, args...)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}

func (dao *DAO[T]) Page(ctx context.Context, q query.DatabaseQuery) (list []*T, total int64, err error) {
	total, err = dao.Count(ctx, q.WhereMap())
	if err != nil {
		return
	}
	list, err = dao.List(ctx, q)
	return
}

func (dao *DAO[T]) Count(ctx context.Context, where map[string]interface{}) (total int64, err error) {
	statement, args := psql.Select("COUNT(*)").From(dao.table).Where(where).MustSql()
	err = dao.db.GetContext(ctx, &total, statement, args...)
	return
}

func (dao *DAO[T]) List(ctx context.Context, q query.DatabaseQuery) (list []*T, err error) {
	builder := psql.Select("*").From(dao.table).Where(q.WhereMap())
	if q.GetLimit() != 0 {
		builder = builder.Offset(uint64(q.GetOffset()))
		builder = builder.Limit(uint64(q.GetLimit()))
	}
	// TODO: add order by support
	statement, args := builder.MustSql()
	list = make([]*T, 0)
	err = dao.db.Unsafe().SelectContext(ctx, &list, statement, args...)
	return
}

func travel(entity interface{}, fn func(field reflect.StructField, value reflect.Value)) {
	t := reflect.TypeOf(entity)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	v := reflect.ValueOf(entity)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)
		if field.Anonymous {
			travel(value.Interface(), fn)
		} else {
			fn(field, value)
		}
	}
}

func (dao *DAO[T]) Insert(ctx context.Context, entity *T) error {
	columns := make([]string, 0)
	values := make([]interface{}, 0)
	travel(entity, func(f reflect.StructField, v reflect.Value) {
		column := utils.DefaultIfZero(f.Tag.Get("db"), strings.ToLower(f.Name))
		switch column {
		case "created_at", "updated_at": // ignore
		default:
			columns = append(columns, column)
			values = append(values, v.Interface())
		}
	})
	statement, args := psql.Insert(dao.table).Columns(columns...).Values(values...).
		Suffix("RETURNING *").
		MustSql()
	return dao.db.Unsafe().QueryRowxContext(ctx, statement, args...).StructScan(entity)
}

func (dao *DAO[T]) BatchInsert(ctx context.Context, entities []*T) error {
	if len(entities) == 0 {
		return nil
	}

	builder := psql.Insert(dao.table)
	travel(entities[0], func(f reflect.StructField, v reflect.Value) {
		column := utils.DefaultIfZero(f.Tag.Get("db"), strings.ToLower(f.Name))
		switch column {
		case "created_at", "updated_at": // ignore
		default:
			builder = builder.Columns(column)
		}
	})

	for _, entity := range entities {
		values := make([]interface{}, 0)
		travel(entity, func(f reflect.StructField, v reflect.Value) {
			column := utils.DefaultIfZero(f.Tag.Get("db"), strings.ToLower(f.Name))
			switch column {
			case "created_at", "updated_at":
				// ignore
			default:
				values = append(values, v.Interface())
			}
		})
		builder = builder.Values(values...)
	}

	statement, args := builder.Suffix("RETURNING *").MustSql()
	rows, err := dao.db.Unsafe().QueryxContext(ctx, statement, args...)
	if err != nil {
		return err
	}
	i := 0
	for rows.Next() {
		err = rows.StructScan(entities[i])
		if err != nil {
			return err
		}
		i++
	}
	return rows.Err()
}

func (dao *DAO[T]) update(ctx context.Context, id string, maps map[string]interface{}) (int64, error) {
	statement, args := psql.Update(dao.table).SetMap(maps).Where(sq.Eq{"id": id}).MustSql()
	result, err := dao.db.ExecContext(ctx, statement, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (dao *DAO[T]) Update(ctx context.Context, entity *T) error {
	var id string
	builder := psql.Update(dao.table)
	travel(entity, func(f reflect.StructField, v reflect.Value) {
		column := utils.DefaultIfZero(f.Tag.Get("db"), strings.ToLower(f.Name))
		switch column {
		case "id":
			id = v.Interface().(string)
		case "created_at": // ignore
		case "updated_at":
			builder = builder.Set(column, entities.NewUnixTime(time.Now()))
		default:
			builder = builder.Set(column, v.Interface())
		}
	})
	statement, args := builder.Where(sq.Eq{"id": id}).Suffix("RETURNING *").MustSql()
	return dao.db.Unsafe().QueryRowxContext(ctx, statement, args...).StructScan(entity)
}
