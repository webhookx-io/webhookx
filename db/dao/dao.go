package dao

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/webhookx-io/webhookx/db/errs"
	"github.com/webhookx-io/webhookx/db/query"
	"github.com/webhookx-io/webhookx/db/transaction"
	"github.com/webhookx-io/webhookx/eventbus"
	"github.com/webhookx-io/webhookx/pkg/tracing"
	"github.com/webhookx-io/webhookx/pkg/ucontext"
	"github.com/webhookx-io/webhookx/utils"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"reflect"
	"strings"
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
	bus *eventbus.EventBus

	workspace bool
	opts      Options
	columns   []string
}

type Options struct {
	Table          string
	EntityName     string
	Workspace      bool
	CachePropagate bool
	CacheName      string
}

func NewDAO[T any](db *sqlx.DB, bus *eventbus.EventBus, opts Options) *DAO[T] {
	dao := DAO[T]{
		log:       zap.S().Named("dao"),
		db:        db,
		bus:       bus,
		workspace: opts.Workspace,
		opts:      opts,
	}
	EachField(new(T), func(f reflect.StructField, _ reflect.Value, column string) {
		if column == "created_at" || column == "updated_at" {
			return
		}
		dao.columns = append(dao.columns, column)
	})
	return &dao
}

func (dao *DAO[T]) debugSQL(sql string, args []interface{}) {
	dao.log.Debug(sql)
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
	ctx, span := tracing.Start(ctx, fmt.Sprintf("dao.%s.get", dao.opts.Table), trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

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
	ctx, span := tracing.Start(ctx, fmt.Sprintf("dao.%s.delete", dao.opts.Table), trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	builder := psql.Delete(dao.opts.Table).Where(sq.Eq{"id": id})
	if dao.workspace {
		wid := ucontext.GetWorkspaceID(ctx)
		builder = builder.Where(sq.Eq{"ws_id": wid})
	}
	statement, args := builder.Suffix("RETURNING *").MustSql()
	dao.debugSQL(statement, args)
	entity := new(T)
	err := dao.UnsafeDB(ctx).QueryRowxContext(ctx, statement, args...).StructScan(entity)
	if err != nil {
		if errors.Is(err, ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	if dao.opts.CachePropagate {
		go dao.propagateEvent(id, entity)
	}
	return true, nil
}

func (dao *DAO[T]) Page(ctx context.Context, q query.Queryer) (list []*T, total int64, err error) {
	ctx, span := tracing.Start(ctx, fmt.Sprintf("dao.%s.page", dao.opts.Table), trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	total, err = dao.Count(ctx, q.WhereMap())
	if err != nil {
		return
	}
	list, err = dao.List(ctx, q)
	return
}

func (dao *DAO[T]) Count(ctx context.Context, where map[string]interface{}) (total int64, err error) {
	ctx, span := tracing.Start(ctx, fmt.Sprintf("dao.%s.count", dao.opts.Table), trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

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
	ctx, span := tracing.Start(ctx, fmt.Sprintf("dao.%s.list", dao.opts.Table), trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

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

func (dao *DAO[T]) Insert(ctx context.Context, entity *T) error {
	ctx, span := tracing.Start(ctx, fmt.Sprintf("dao.%s.insert", dao.opts.Table), trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	values := make([]interface{}, 0)
	EachField(entity, func(f reflect.StructField, v reflect.Value, column string) {
		if column == "created_at" || column == "updated_at" {
			return
		}
		value := v.Interface()
		if column == "ws_id" && dao.workspace {
			value = ucontext.GetWorkspaceID(ctx)
		}
		values = append(values, value)
	})
	statement, args := psql.Insert(dao.opts.Table).Columns(dao.columns...).Values(values...).
		Suffix("RETURNING *").
		MustSql()
	dao.debugSQL(statement, args)
	err := dao.UnsafeDB(ctx).QueryRowxContext(ctx, statement, args...).StructScan(entity)
	if dao.opts.CachePropagate && err == nil {
		id := reflect.ValueOf(*entity).FieldByName("ID")
		if id.IsValid() {
			go dao.propagateEvent(id.String(), entity)
		}
	}
	return errs.ConvertError(err)
}

func (dao *DAO[T]) BatchInsert(ctx context.Context, entities []*T) error {
	ctx, span := tracing.Start(ctx, fmt.Sprintf("dao.%s.batch_insert", dao.opts.Table), trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	if len(entities) == 0 {
		return nil
	}

	builder := psql.Insert(dao.opts.Table).Columns(dao.columns...)

	for _, entity := range entities {
		values := make([]interface{}, 0)
		EachField(entity, func(f reflect.StructField, v reflect.Value, column string) {
			if column == "created_at" || column == "updated_at" {
				return
			}
			value := v.Interface()
			if column == "ws_id" && dao.workspace {
				value = ucontext.GetWorkspaceID(ctx)
			}
			values = append(values, value)
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
	return rows, err
}

func (dao *DAO[T]) Update(ctx context.Context, entity *T) error {
	ctx, span := tracing.Start(ctx, fmt.Sprintf("dao.%s.update", dao.opts.Table), trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	var id string
	builder := psql.Update(dao.opts.Table)
	EachField(entity, func(f reflect.StructField, v reflect.Value, column string) {
		switch column {
		case "id":
			id = v.Interface().(string)
		case "created_at": // ignore
		case "updated_at":
			builder = builder.Set(column, sq.Expr("NOW()"))
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
		go dao.propagateEvent(id, entity)
	}
	return errs.ConvertError(err)
}

func (dao *DAO[T]) Upsert(ctx context.Context, fields []string, entity *T) error {
	columns := make([]string, 0)
	values := make([]interface{}, 0)
	EachField(entity, func(f reflect.StructField, v reflect.Value, column string) {
		switch column {
		case "created_at", "updated_at":
			columns = append(columns, column)
			values = append(values, sq.Expr("NOW()"))
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
			go dao.propagateEvent(id.String(), entity)
		}
	}
	return errs.ConvertError(err)
}

func (dao *DAO[T]) propagateEvent(id string, entity *T) {
	data := &eventbus.CrudData{
		ID:        id,
		CacheName: dao.opts.CacheName,
		Entity:    dao.opts.EntityName,
		Data:      utils.Must(json.Marshal(entity)),
	}
	wid := reflect.ValueOf(*entity).FieldByName("WorkspaceId")
	if wid.IsValid() {
		data.WID = wid.String()
	}
	_ = dao.bus.ClusteringBroadcast(eventbus.EventCRUD, data)
}
