package query

type Queryer interface {
	Offset() int64
	Limit() int64
	WhereMap() map[string]interface{}
	Orders() []*Order
}

type Query struct {
	offset int64
	limit  int64
	orders []*Order
}

func (q *Query) Page(pageNo, pageSize uint64) {
	if pageNo < 1 {
		pageNo = 1
	}
	offset := (pageNo - 1) * pageSize
	q.offset = int64(offset)
	q.limit = int64(int(pageSize))
}

func (q *Query) Offset() int64 {
	return q.offset
}

func (q *Query) Limit() int64 {
	return q.limit
}

func (q *Query) WhereMap() map[string]interface{} {
	return nil
}

func (q *Query) Orders() []*Order {
	return q.orders
}

func (q *Query) Order(column string, sort Sort) {
	q.orders = append(q.orders, &Order{column, sort})
}
