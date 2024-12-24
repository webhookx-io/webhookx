package dao

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/constants"
	"github.com/webhookx-io/webhookx/db/errs"
	"github.com/webhookx-io/webhookx/db/query"
	"github.com/webhookx-io/webhookx/db/transaction"
	"github.com/webhookx-io/webhookx/eventbus"
	"github.com/webhookx-io/webhookx/mcache"
	"github.com/webhookx-io/webhookx/pkg/tracing"
	"github.com/webhookx-io/webhookx/pkg/types"
	"github.com/webhookx-io/webhookx/pkg/ucontext"
	"github.com/webhookx-io/webhookx/utils"
	"go.opentelemetry.io/otel/trace"
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
	log *zap.SugaredLogger
	db  *sqlx.DB

	workspace bool
	opts      Options
}

type Options struct {
	Table          string
	EntityName     string
	Workspace      bool
	CachePropagate bool
	CacheKey       constants.CacheKey
}

func NewDAO[T any](db *sqlx.DB, opts Options) *DAO[T] {
	dao := DAO[T]{
		log:       zap.S(),
		db:        db,
		workspace: opts.Workspace,
		opts:      opts,
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
	tracingCtx, span := tracing.Start(ctx, fmt.Sprintf("dao.%s.get", dao.opts.Table), trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()
	ctx = tracingCtx

	return dao.Select(ctx, "id", id)
}

func (dao *DAO[T]) Select(ctx context.Context, field string, value string) (entity *T, err error) {
	builder := psql.Select("*").From(dao.opts.Table).Where(sq.Eq{field: value})
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
	builder := psql.Select("*").From(dao.opts.Table).Where(sq.Eq{field: value})
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
	tracingCtx, span := tracing.Start(ctx, fmt.Sprintf("dao.%s.delete", dao.opts.Table), trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()
	ctx = tracingCtx
	builder := psql.Delete(dao.opts.Table).Where(sq.Eq{"id": id})
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
	if dao.opts.CachePropagate && rows > 0 {
		go dao.handleCachePropagate(id)
	}
	return rows > 0, nil
}

func (dao *DAO[T]) Page(ctx context.Context, q query.Queryer) (list []*T, total int64, err error) {
	tracingCtx, span := tracing.Start(ctx, fmt.Sprintf("dao.%s.page", dao.opts.Table), trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()
	ctx = tracingCtx
	total, err = dao.Count(ctx, q.WhereMap())
	if err != nil {
		return
	}
	list, err = dao.List(ctx, q)
	return
}

func (dao *DAO[T]) Count(ctx context.Context, where map[string]interface{}) (total int64, err error) {
	tracingCtx, span := tracing.Start(ctx, fmt.Sprintf("dao.%s.count", dao.opts.Table), trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()
	ctx = tracingCtx
	builder := psql.Select("COUNT(*)").From(dao.opts.Table)
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
	tracingCtx, span := tracing.Start(ctx, fmt.Sprintf("dao.%s.list", dao.opts.Table), trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()
	ctx = tracingCtx
	builder := psql.Select("*").From(dao.opts.Table)
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
	tracingCtx, span := tracing.Start(ctx, fmt.Sprintf("dao.%s.insert", dao.opts.Table), trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()
	ctx = tracingCtx
	columns := make([]string, 0)
	values := make([]interface{}, 0)
	travel(entity, func(f reflect.StructField, v reflect.Value) {
		column := utils.DefaultIfZero(f.Tag.Get("db"), strings.ToLower(f.Name))
		switch column {
		case "created_at", "updated_at": // ignore
		default:
			columns = append(columns, column)
			value := v.Interface()
			if column == "ws_id" && dao.workspace {
				value = ucontext.GetWorkspaceID(ctx)
			}
			values = append(values, value)
		}
	})
	statement, args := psql.Insert(dao.opts.Table).Columns(columns...).Values(values...).
		Suffix("RETURNING *").
		MustSql()
	dao.debugSQL(statement, args)
	err := dao.UnsafeDB(ctx).QueryRowxContext(ctx, statement, args...).StructScan(entity)
	return errs.ConvertError(err)
}

func (dao *DAO[T]) BatchInsert(ctx context.Context, entities []*T) error {
	tracingCtx, span := tracing.Start(ctx, fmt.Sprintf("dao.%s.batch_insert", dao.opts.Table), trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()
	ctx = tracingCtx
	if len(entities) == 0 {
		return nil
	}

	builder := psql.Insert(dao.opts.Table)
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
				value := v.Interface()
				if column == "ws_id" && dao.workspace {
					value = ucontext.GetWorkspaceID(ctx)
				}
				values = append(values, value)
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
	builder := psql.Update(dao.opts.Table).SetMap(maps).Where(sq.Eq{"id": id})
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
	rows, err := result.RowsAffected()
	if dao.opts.CachePropagate && rows == 1 {
		go dao.handleCachePropagate(id)
	}
	return rows, err
}

func (dao *DAO[T]) Update(ctx context.Context, entity *T) error {
	tracingCtx, span := tracing.Start(ctx, fmt.Sprintf("dao.%s.update", dao.opts.Table), trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()
	ctx = tracingCtx
	var id string
	builder := psql.Update(dao.opts.Table)
	travel(entity, func(f reflect.StructField, v reflect.Value) {
		column := utils.DefaultIfZero(f.Tag.Get("db"), strings.ToLower(f.Name))
		switch column {
		case "id":
			id = v.Interface().(string)
		case "created_at": // ignore
		case "updated_at":
			builder = builder.Set(column, types.NewTime(time.Now()))
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
	err := dao.UnsafeDB(ctx).QueryRowxContext(ctx, statement, args...).StructScan(entity)
	if dao.opts.CachePropagate && err == nil {
		go dao.handleCachePropagate(id)
	}
	return errs.ConvertError(err)
}

func (dao *DAO[T]) Upsert(ctx context.Context, fields []string, entity *T) error {
	now := time.Now()
	columns := make([]string, 0)
	values := make([]interface{}, 0)
	travel(entity, func(f reflect.StructField, v reflect.Value) {
		column := utils.DefaultIfZero(f.Tag.Get("db"), strings.ToLower(f.Name))
		switch column {
		case "created_at", "updated_at":
			columns = append(columns, column)
			values = append(values, types.NewTime(now))
		default:
			columns = append(columns, column)
			value := v.Interface()
			if column == "ws_id" && dao.workspace {
				value = ucontext.GetWorkspaceID(ctx)
			}
			values = append(values, value)
		}
	})
	var clause strings.Builder
	for i := range columns {
		column := columns[i]
		if column == "created_at" || column == "id" {
			continue
		}
		clause.WriteString(column)
		clause.WriteString(" = EXCLUDED.")
		clause.WriteString(column)
		if i < len(columns)-1 {
			clause.WriteString(", ")
		}
	}
	statement, args := psql.Insert(dao.opts.Table).Columns(columns...).Values(values...).
		Suffix("ON CONFLICT (" + strings.Join(fields, ",") + ") DO UPDATE SET " + clause.String()).
		Suffix("RETURNING *").
		MustSql()
	dao.debugSQL(statement, args)
	err := dao.UnsafeDB(ctx).QueryRowxContext(ctx, statement, args...).StructScan(entity)
	if dao.opts.CachePropagate && err == nil {
		id := reflect.ValueOf(*entity).FieldByName("ID")
		if id.IsValid() {
			go dao.handleCachePropagate(id.String())
		}
	}
	return errs.ConvertError(err)
}

func (dao *DAO[T]) handleCachePropagate(id string) {
	key := dao.opts.CacheKey.Build(id)
	if e := mcache.Invalidate(context.TODO(), key); e != nil {
		dao.log.Warnf("failed to invalidate mcache: key=%s, %v", key, e)
	}
	dao.publishEvent(eventbus.EventInvalidation, map[string]interface{}{
		"cache_key": key,
	})
}

func (dao *DAO[T]) publishEvent(event string, data interface{}) {
	bytes, err := json.Marshal(data)
	if err != nil {
		dao.log.Warnf("failed to marshal data: %v", err)
		return
	}
	payload := eventbus.EventPayload{
		Event: event,
		Data:  bytes,
		Time:  time.Now().UnixMilli(),
		Node:  config.NODE,
	}
	bytes, err = json.Marshal(payload)
	if err != nil {
		dao.log.Warnf("failed to marshal payload: %v", err)
		return
	}

	dao.log.Debugf("broadcasting event: %s", string(bytes))
	statement := fmt.Sprintf("NOTIFY %s, '%s'", "webhookx", string(bytes))
	_, err = dao.DB(context.TODO()).ExecContext(context.TODO(), statement)
	if err != nil {
		dao.log.Warnf("failed to publish event: %v", err)
	}
}
