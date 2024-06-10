package query

type DatabaseQuery interface {
	GetOffset() int64
	GetLimit() int64
	WhereMap() map[string]interface{}
}

type Query struct {
	Offset int64
	Limit  int64
}

func (q *Query) Page(pageNo, pageSize uint64) {
	if pageNo < 1 {
		pageNo = 1
	}
	offset := (pageNo - 1) * pageSize
	q.Offset = int64(offset)
	q.Limit = int64(int(pageSize))
}

func (q *Query) GetOffset() int64 {
	return q.Offset
}

func (q *Query) GetLimit() int64 {
	return q.Limit
}

func (q *Query) WhereMap() map[string]interface{} {
	return nil
}
