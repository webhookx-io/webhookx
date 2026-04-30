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

func (o Order) SQL() string {
	return o.Column + " " + o.Sort
}

type Query struct {
	Offset      int
	Limit       int
	Wheres      []Condition
	Orders      []Order
	CursorModel bool
	Reverse     bool
}

func (q *Query) Where(column string, op Operator, value any) *Query {
	q.Wheres = append(q.Wheres, Condition{column, op, value})
	return q
}

func (q *Query) Order(column string, sort Sort) *Query {
	q.Orders = append(q.Orders, Order{column, sort})
	return q
}

func (q *Query) Page(pageNo, pageSize int) {
	if pageNo < 1 {
		pageNo = 1
	}
	q.Offset = (pageNo - 1) * pageSize
	q.Limit = pageSize
}

func (q *Query) clone() *Query {
	cloned := *q
	if len(q.Wheres) > 0 {
		cloned.Wheres = append([]Condition(nil), q.Wheres...)
	}
	if len(q.Orders) > 0 {
		cloned.Orders = append([]Order(nil), q.Orders...)
	}
	return &cloned
}

type Cursor[T any] struct {
	Cursor   bool
	Reversed bool
	Data     []T
	// Deprecated
	Total   int64
	HasMore bool
	FirstId *string
	LastId  *string
}
