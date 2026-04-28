package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/webhookx-io/webhookx/db/errs"
	"github.com/webhookx-io/webhookx/db/transaction"
	"github.com/webhookx-io/webhookx/pkg/contextx"
	"github.com/webhookx-io/webhookx/pkg/tracing"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
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

type Schema interface {
	PrimaryKey() string
}

type DAO[T Schema] struct {
	log    *zap.SugaredLogger
	db     *sqlx.DB
	schema *SchemaMeta

	opts      Options
	workspace bool
}

func NewDAO[T Schema](db *sqlx.DB, opts Options) *DAO[T] {
	dao := DAO[T]{
		log:       zap.S().Named("dao"),
		db:        db,
		opts:      opts,
		workspace: opts.Workspace,
		schema:    NewSchema[T](opts.EntityName),
	}
	return &dao
}

func (dao *DAO[T]) debugSQL(sql string, args []interface{}) {
	dao.log.Debug(sql)
}

func (dao *DAO[T]) trace(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if dao.opts.Instrumented {
		return tracing.Start(ctx, spanName, opts...)
	}
	return tracing.NoopTracer.Start(ctx, spanName, opts...)
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

func (dao *DAO[T]) Get(ctx context.Context, id string) (entity *T, err error) {
	ctx, span := dao.trace(ctx, fmt.Sprintf("dao.%s.get", dao.opts.Table))
	defer span.End()

	return dao.Select(ctx, "id", id)
}

func (dao *DAO[T]) Select(ctx context.Context, field string, value string) (entity *T, err error) {
	builder := psql.Select("*").From(dao.opts.Table).Where(sq.Eq{field: value})
	builder = dao.workspaceFilter(ctx, builder)
	statement, args := builder.MustSql()
	dao.debugSQL(statement, args)
	entity = new(T)
	err = dao.DB(ctx).GetContext(ctx, entity, statement, args...)
	if errors.Is(err, ErrNoRows) {
		return nil, nil
	}
	return
}

func (dao *DAO[T]) Delete(ctx context.Context, id string) (bool, error) {
	ctx, span := dao.trace(ctx, fmt.Sprintf("dao.%s.delete", dao.opts.Table))
	defer span.End()

	builder := psql.Delete(dao.opts.Table).Where(sq.Eq{"id": id})
	if dao.workspace {
		wid := contextx.GetWorkspaceID(ctx)
		builder = builder.Where(sq.Eq{"ws_id": wid})
	}
	err := dao.executePropagate(ctx, builder, new(T))
	if err != nil {
		if errors.Is(err, ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func appendWhere(schema *SchemaMeta, builder sq.SelectBuilder, conditions []Condition) sq.SelectBuilder {
	for _, condition := range conditions {
		if schema.HasColumn(condition.Column) {
			switch condition.Op {
			case Equal:
				builder = builder.Where(sq.Eq{condition.Column: condition.Value})
			case JsonContain:
				builder = builder.Where(condition.Column+" @> ?", condition.Value)
			case GreaterThan:
				builder = builder.Where(sq.Gt{condition.Column: condition.Value})
			case GreaterThanOrEqual:
				builder = builder.Where(sq.GtOrEq{condition.Column: condition.Value})
			case LessThan:
				builder = builder.Where(sq.Lt{condition.Column: condition.Value})
			case LessThanOrEqual:
				builder = builder.Where(sq.LtOrEq{condition.Column: condition.Value})
			}
		}
	}
	return builder
}

func appendOrder(builder sq.SelectBuilder, orders []Order) sq.SelectBuilder {
	for _, order := range orders {
		builder = builder.OrderBy(order.SQL())
	}
	return builder
}

func (dao *DAO[T]) workspaceFilter(ctx context.Context, builder sq.SelectBuilder) sq.SelectBuilder {
	if dao.workspace {
		wid := contextx.GetWorkspaceID(ctx)
		builder = builder.Where("ws_id = ?", wid)
	}
	return builder
}

func (dao *DAO[T]) Count(ctx context.Context, query *Query) (total int64, err error) {
	if query == nil {
		panic("query is nil")
	}

	ctx, span := dao.trace(ctx, fmt.Sprintf("dao.%s.count", dao.opts.Table))
	defer span.End()

	builder := psql.Select("COUNT(*)").From(dao.opts.Table)
	builder = appendWhere(dao.schema, builder, query.Wheres)
	builder = dao.workspaceFilter(ctx, builder)
	statement, args := builder.MustSql()
	dao.debugSQL(statement, args)
	err = dao.DB(ctx).GetContext(ctx, &total, statement, args...)
	return
}

func (dao *DAO[T]) List(ctx context.Context, query *Query) (list []*T, err error) {
	if query == nil {
		panic("query is nil")
	}

	ctx, span := dao.trace(ctx, fmt.Sprintf("dao.%s.list", dao.opts.Table))
	defer span.End()

	builder := psql.Select("*").From(dao.opts.Table)
	builder = appendWhere(dao.schema, builder, query.Wheres)
	builder = dao.workspaceFilter(ctx, builder)
	builder = appendOrder(builder, query.Orders)

	if query.Limit != 0 {
		builder = builder.Limit(uint64(query.Limit))
	}
	if query.Offset != 0 {
		builder = builder.Offset(uint64(query.Offset))
	}

	statement, args := builder.MustSql()
	dao.debugSQL(statement, args)
	list = make([]*T, 0)
	err = dao.DB(ctx).SelectContext(ctx, &list, statement, args...)
	return
}

func (dao *DAO[T]) Cursor(ctx context.Context, query *Query) (cursor Cursor[*T], err error) {
	if query == nil {
		panic("query is nil")
	}
	if query.Limit <= 0 {
		panic("query.limit must be positive")
	}
	var spanName string
	if query.CursorModel {
		spanName = fmt.Sprintf("dao.%s.cursor", dao.opts.Table)
	} else {
		spanName = fmt.Sprintf("dao.%s.page", dao.opts.Table)
	}
	ctx, span := dao.trace(ctx, spanName)
	defer span.End()

	builder := psql.Select("*").From(dao.opts.Table)
	builder = appendWhere(dao.schema, builder, query.Wheres)
	builder = dao.workspaceFilter(ctx, builder)
	builder = appendOrder(builder, query.Orders)
	builder = builder.Limit(uint64(query.Limit + 1))
	if query.Offset > 0 {
		builder = builder.Offset(uint64(query.Offset))
	}

	statement, args := builder.MustSql()
	dao.debugSQL(statement, args)

	cursor.Data = make([]*T, 0)
	err = dao.DB(ctx).SelectContext(ctx, &cursor.Data, statement, args...)
	if err != nil {
		return
	}

	if len(cursor.Data) > query.Limit {
		cursor.HasMore = true
		cursor.Data = cursor.Data[:query.Limit]
	}

	if query.Reverse {
		slices.Reverse(cursor.Data)
		cursor.Reversed = true
	}

	if len(cursor.Data) > 0 {
		first := cursor.Data[0]
		firstId := reflect.ValueOf(*first).FieldByName("ID")
		if firstId.IsValid() {
			cursor.FirstId = new(firstId.String())
		}

		last := cursor.Data[len(cursor.Data)-1]
		lastId := reflect.ValueOf(*last).FieldByName("ID")
		if lastId.IsValid() {
			cursor.LastId = new(lastId.String())
		}
	}

	if !query.CursorModel {
		totoal, err := dao.Count(ctx, query)
		if err != nil {
			return cursor, err
		}
		cursor.Total = totoal
	}
	cursor.Cursor = query.CursorModel

	return
}

func (dao *DAO[T]) Insert(ctx context.Context, entity *T) error {
	ctx, span := dao.trace(ctx, fmt.Sprintf("dao.%s.insert", dao.opts.Table))
	defer span.End()

	values := make([]interface{}, 0)
	EachField(entity, func(f reflect.StructField, v reflect.Value, column string) {
		if column == "created_at" || column == "updated_at" {
			return
		}
		value := v.Interface()
		if column == "ws_id" && dao.workspace {
			value = contextx.GetWorkspaceID(ctx)
		}
		values = append(values, value)
	})
	builder := psql.Insert(dao.opts.Table).
		Columns(dao.schema.InsertColumns()...).
		Values(values...)
	return dao.executePropagate(ctx, builder, entity)
}

func (dao *DAO[T]) BatchInsert(ctx context.Context, entities []*T) error {
	ctx, span := dao.trace(ctx, fmt.Sprintf("dao.%s.batch_insert", dao.opts.Table))
	defer span.End()

	if len(entities) == 0 {
		return nil
	}

	builder := psql.Insert(dao.opts.Table).Columns(dao.schema.InsertColumns()...)

	for _, entity := range entities {
		values := make([]interface{}, 0)
		EachField(entity, func(f reflect.StructField, v reflect.Value, column string) {
			if column == "created_at" || column == "updated_at" {
				return
			}
			value := v.Interface()
			if column == "ws_id" && dao.workspace {
				value = contextx.GetWorkspaceID(ctx)
			}
			values = append(values, value)
		})
		builder = builder.Values(values...)
	}

	statement, args := builder.Suffix("RETURNING *").MustSql()
	rows, err := dao.DB(ctx).QueryxContext(ctx, statement, args...)
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

func (dao *DAO[T]) update(ctx context.Context, id string, maps map[string]interface{}) (entity *T, err error) {
	builder := psql.Update(dao.opts.Table).SetMap(maps).Where(sq.Eq{"id": id})
	if dao.workspace {
		wid := contextx.GetWorkspaceID(ctx)
		builder = builder.Where(sq.Eq{"ws_id": wid})
	}

	entity = new(T)
	err = dao.executePropagate(ctx, builder, entity)
	return entity, err
}

func (dao *DAO[T]) executeReturning(ctx context.Context, builder interface{}, obj *T) error {
	var statement string
	var args []interface{}

	switch t := builder.(type) {
	case sq.UpdateBuilder:
		statement, args = t.Suffix("RETURNING *").MustSql()
	case sq.InsertBuilder:
		statement, args = t.Suffix("RETURNING *").MustSql()
	case sq.DeleteBuilder:
		statement, args = t.Suffix("RETURNING *").MustSql()
	default:
		panic("invalid builder: " + reflect.TypeOf(t).String())
	}

	dao.debugSQL(statement, args)

	err := dao.DB(ctx).QueryRowxContext(ctx, statement, args...).StructScan(obj)
	return errs.ConvertError(err)
}

func (dao *DAO[T]) executePropagate(ctx context.Context, builder interface{}, entity *T) error {
	err := dao.executeReturning(ctx, builder, entity)
	if err != nil {
		return err
	}

	if dao.opts.CachePropagate {
		go dao.propagateEvent(ctx, entity)
	}
	return nil
}

func (dao *DAO[T]) Update(ctx context.Context, entity *T) error {
	ctx, span := dao.trace(ctx, fmt.Sprintf("dao.%s.update", dao.opts.Table))
	defer span.End()

	builder := psql.Update(dao.opts.Table)
	EachField(entity, func(f reflect.StructField, v reflect.Value, column string) {
		switch column {
		case "created_at": // ignore
		case "updated_at":
			builder = builder.Set(column, sq.Expr("NOW()"))
		default:
			builder = builder.Set(column, v.Interface())
		}
	})
	if dao.workspace {
		wid := contextx.GetWorkspaceID(ctx)
		builder = builder.Where(sq.Eq{"ws_id": wid})
	}

	builder = builder.Where(sq.Eq{"id": (*entity).PrimaryKey()})
	return dao.executePropagate(ctx, builder, entity)
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
				value = contextx.GetWorkspaceID(ctx)
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

	bulder := psql.Insert(dao.opts.Table).
		Columns(columns...).
		Values(values...).
		Suffix("ON CONFLICT (" + strings.Join(fields, ",") + ") DO UPDATE SET " + clause.String())

	return dao.executePropagate(ctx, bulder, entity)
}

func (dao *DAO[T]) propagateEvent(ctx context.Context, entity *T) {
	ctx, span := dao.trace(ctx, fmt.Sprintf("%s.crud.broadcast", dao.opts.Table))
	defer span.End()
	dao.opts.PropagateHandler(ctx, &dao.opts, (*entity).PrimaryKey(), entity)
}
