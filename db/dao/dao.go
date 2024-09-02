package dao

import (
	"context"
	"database/sql"
	"errors"
	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/webhookx-io/webhookx/db/query"
	"github.com/webhookx-io/webhookx/db/transaction"
	"github.com/webhookx-io/webhookx/pkg/types"
	"github.com/webhookx-io/webhookx/pkg/ucontext"
	"github.com/webhookx-io/webhookx/utils"
	"go.uber.org/zap"
	"reflect"
	"strings"
	"time"
)

var (
	ErrNoRows              = sql.ErrNoRows
	ErrConstraintViolation = errors.New("constraint violation")
)
var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

// Queryable is an interface to be used interchangeably for sqlx.Db and sqlx.Tx
type Queryable interface {
	sqlx.ExtContext
	GetContext(context.Context, interface{}, string, ...interface{}) error
	SelectContext(context.Context, interface{}, string, ...interface{}) error
}

type DAO[T any] struct {
	log       *zap.SugaredLogger
	db        *sqlx.DB
	table     string
	workspace bool
}

func NewDAO[T any](table string, db *sqlx.DB, workspace bool) *DAO[T] {
	dao := DAO[T]{
		log:       zap.S(),
		db:        db,
		table:     table,
		workspace: workspace,
	}
	return &dao
}

func (dao *DAO[T]) debugSQL(sql string, args []interface{}) {
	dao.log.Debugf("[dao] execute: %s", sql)
}

func (dao *DAO[T]) DB(ctx context.Context) Queryable {
	if ctx == nil {
		ctx = context.TODO()
	}

	if tx, ok := transaction.FromContext(ctx); ok {
		return tx
	}

	return dao.db
}

func (dao *DAO[T]) UnsafeDB(ctx context.Context) Queryable {
	db := dao.DB(ctx)

	if tx, ok := db.(*sqlx.Tx); ok {
		return tx.Unsafe()
	}

	return db.(*sqlx.DB).Unsafe()
}

func (dao *DAO[T]) Get(ctx context.Context, id string) (entity *T, err error) {
	builder := psql.Select("*").From(dao.table).Where(sq.Eq{"id": id})
	if dao.workspace {
		wid := ucontext.GetWorkspaceID(ctx)
		builder = builder.Where(sq.Eq{"ws_id": wid})
	}
	statement, args := builder.MustSql()
	dao.debugSQL(statement, args)
	entity = new(T)
	err = dao.UnsafeDB(ctx).GetContext(ctx, entity, statement, args...)
	if errors.Is(err, ErrNoRows) {
		return nil, nil
	}
	return
}

func (dao *DAO[T]) selectByField(ctx context.Context, field string, value string) (entity *T, err error) {
	builder := psql.Select("*").From(dao.table).Where(sq.Eq{field: value})
	if dao.workspace {
		wid := ucontext.GetWorkspaceID(ctx)
		builder = builder.Where(sq.Eq{"ws_id": wid})
	}
	statement, args := builder.MustSql()
	dao.debugSQL(statement, args)
	entity = new(T)
	err = dao.UnsafeDB(ctx).GetContext(ctx, entity, statement, args...)
	if errors.Is(err, ErrNoRows) {
		return nil, nil
	}
	return
}

func (dao *DAO[T]) Delete(ctx context.Context, id string) (bool, error) {
	builder := psql.Delete(dao.table).Where(sq.Eq{"id": id})
	if dao.workspace {
		wid := ucontext.GetWorkspaceID(ctx)
		builder = builder.Where(sq.Eq{"ws_id": wid})
	}
	statement, args := builder.MustSql()
	dao.debugSQL(statement, args)
	result, err := dao.DB(ctx).ExecContext(ctx, statement, args...)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}

func (dao *DAO[T]) Page(ctx context.Context, q query.Queryer) (list []*T, total int64, err error) {
	total, err = dao.Count(ctx, q.WhereMap())
	if err != nil {
		return
	}
	list, err = dao.List(ctx, q)
	return
}

func (dao *DAO[T]) Count(ctx context.Context, where map[string]interface{}) (total int64, err error) {
	builder := psql.Select("COUNT(*)").From(dao.table)
	if len(where) > 0 {
		builder = builder.Where(where)
	}
	if dao.workspace {
		wid := ucontext.GetWorkspaceID(ctx)
		builder = builder.Where(sq.Eq{"ws_id": wid})
	}
	statement, args := builder.MustSql()
	dao.debugSQL(statement, args)
	err = dao.DB(ctx).GetContext(ctx, &total, statement, args...)
	return
}

func (dao *DAO[T]) List(ctx context.Context, q query.Queryer) (list []*T, err error) {
	builder := psql.Select("*").From(dao.table)
	where := q.WhereMap()
	if len(where) > 0 {
		builder = builder.Where(where)
	}
	if dao.workspace {
		wid := ucontext.GetWorkspaceID(ctx)
		builder = builder.Where(sq.Eq{"ws_id": wid})
	}
	if q.Limit() != 0 {
		builder = builder.Offset(uint64(q.Offset()))
		builder = builder.Limit(uint64(q.Limit()))
	}
	for _, order := range q.Orders() {
		builder = builder.OrderBy(order.Column + " " + order.Sort)
	}
	statement, args := builder.MustSql()
	dao.debugSQL(statement, args)
	list = make([]*T, 0)
	err = dao.UnsafeDB(ctx).SelectContext(ctx, &list, statement, args...)
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
	dao.debugSQL(statement, args)
	err := dao.UnsafeDB(ctx).QueryRowxContext(ctx, statement, args...).StructScan(entity)
	if err != nil {
		var pgErr *pq.Error
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrConstraintViolation // TODO attach the violated column
		}
	}
	return err
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
	rows, err := dao.UnsafeDB(ctx).QueryxContext(ctx, statement, args...)
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
	builder := psql.Update(dao.table).SetMap(maps).Where(sq.Eq{"id": id})
	if dao.workspace {
		wid := ucontext.GetWorkspaceID(ctx)
		builder = builder.Where(sq.Eq{"ws_id": wid})
	}
	statement, args := builder.MustSql()
	dao.debugSQL(statement, args)
	result, err := dao.DB(ctx).ExecContext(ctx, statement, args...)
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
			builder = builder.Set(column, types.NewUnixTime(time.Now()))
		default:
			builder = builder.Set(column, v.Interface())
		}
	})
	if dao.workspace {
		wid := ucontext.GetWorkspaceID(ctx)
		builder = builder.Where(sq.Eq{"ws_id": wid})
	}
	statement, args := builder.Where(sq.Eq{"id": id}).Suffix("RETURNING *").MustSql()
	dao.debugSQL(statement, args)
	return dao.UnsafeDB(ctx).QueryRowxContext(ctx, statement, args...).StructScan(entity)
}
