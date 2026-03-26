package dao

// Operator SQL operator
type Operator int

const (
	// Equal =
	Equal Operator = iota
	// GreaterThan >
	GreaterThan
	// GreaterThanOrEqual >=
	GreaterThanOrEqual
	// LessThan <
	LessThan
	// LessThanOrEqual <=
	LessThanOrEqual
	// JsonContain @>
	JsonContain
)

type Condition struct {
	Column string
	Op     Operator
	Value  interface{}
}

type Sort = string

const (
	ASC  Sort = "ASC"
	DESC Sort = "DESC"
)

type Order struct {
	Column string
	Sort   Sort
}

func (o Order) String() string {
	return o.SQL()
}

func (o Order) SQL() string {
	return o.Column + " " + o.Sort
}

type Query struct {
	offset      int
	limit       int
	wheres      []Condition
	orders      []Order
	CursorModel bool
}

func (q *Query) Where(column string, op Operator, value any) *Query {
	q.wheres = append(q.wheres, Condition{column, op, value})
	return q
}

func (q *Query) Order(column string, sort Sort) *Query {
	q.orders = append(q.orders, Order{column, sort})
	return q
}

func (q *Query) Page(pageNo, pageSize int) {
	if pageNo < 1 {
		pageNo = 1
	}
	q.offset = (pageNo - 1) * pageSize
	q.limit = pageSize
}

func (q Query) clone() Query {
	cloned := q
	if len(q.wheres) > 0 {
		cloned.wheres = append([]Condition(nil), q.wheres...)
	}
	if len(q.orders) > 0 {
		cloned.orders = append([]Order(nil), q.orders...)
	}
	return cloned
}

type Cursor = string

type CursorResult[T any] struct {
	Data    []T
	Total   int64
	HasMore bool
	Cursor  *Cursor
}
