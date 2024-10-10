package types

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

var testTime, _ = time.Parse(time.RFC3339Nano, "2006-01-02T15:04:05.999Z")

func TestTime(t *testing.T) {
	t1 := NewTime(testTime)
	s, err := json.Marshal(t1)
	assert.Nil(t, err)
	assert.EqualValues(t, "1136214245999", string(s))

	var t2 Time
	assert.NoError(t, json.Unmarshal([]byte("1136214245999"), &t2))
	assert.True(t, t1.Equal(t2))
}

func TestZeroTime(t *testing.T) {
	var zeroTime1 Time
	s, err := json.Marshal(zeroTime1)
	assert.Nil(t, err)
	assert.EqualValues(t, "0", string(s))

	var zeroTime2 Time
	assert.NoError(t, json.Unmarshal([]byte("0"), &zeroTime2))
	assert.True(t, zeroTime1.Equal(zeroTime2))
}
