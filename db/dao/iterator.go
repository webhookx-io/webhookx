package dao

import "reflect"

type Iterator[T any] struct {
	hasMore bool
	current *T
	err     error
	query   *Query
	values  []*T
	fetch   func(*Query) ([]*T, bool, error)
}

func (it *Iterator[T]) Next() bool {
	if len(it.values) == 0 && it.hasMore {
		id := reflect.ValueOf(it.current).Elem().FieldByName("ID").String()
		query := it.query.clone().Where("id", LessThan, id)
		it.fetchData(query)
	}

	if len(it.values) == 0 {
		return false
	}
	it.current = it.values[0]
	it.values = it.values[1:]
	return true
}

func (it *Iterator[T]) Current() *T {
	return it.current
}

func (it *Iterator[T]) Err() error {
	return it.err
}

func (it *Iterator[T]) fetchData(query *Query) {
	it.values, it.hasMore, it.err = it.fetch(query)
}
