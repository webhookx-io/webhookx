package dao

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPage(t *testing.T) {
	q := Query{}
	q.Page(0, 10)
	assert.Equal(t, 0, q.Offset)
	assert.Equal(t, 10, q.Limit)
}

func TestClone(t *testing.T) {
	q := Query{}
	q.Page(1, 20)
	q.Order("id", "desc")
	q.Where("foo", Equal, "bar")

	q2 := q.clone()

	// modify q
	q.Where("field", Equal, "value")
	q.Limit = 40

	// assert q
	assert.Equal(t, 0, q.Offset)
	assert.Equal(t, 40, q.Limit)
	assert.EqualValues(t, []Order{{"id", "desc"}}, q.Orders)
	assert.EqualValues(t,
		[]Condition{{"foo", Equal, "bar"}, {"field", Equal, "value"}},
		q.Wheres)

	// assert q2
	assert.Equal(t, 0, q2.Offset)
	assert.Equal(t, 20, q2.Limit)
	assert.EqualValues(t, []Order{{"id", "desc"}}, q2.Orders)
	assert.EqualValues(t, []Condition{{"foo", Equal, "bar"}}, q2.Wheres)
}
