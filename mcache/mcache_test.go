package mcache

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type MockCache struct {
	data map[string]interface{}
}

func (m *MockCache) Put(ctx context.Context, key string, val interface{}, expiration time.Duration) error {
	m.data[key] = val
	return nil
}

func (m *MockCache) Get(ctx context.Context, key string, val interface{}) (exist bool, err error) {
	time.Sleep(time.Millisecond * 100)
	if v, ok := m.data[key]; ok {
		*val.(*string) = *v.(*string)
		return true, nil
	}

	return false, nil
}

func (m *MockCache) Remove(ctx context.Context, key string) error {
	delete(m.data, key)
	return nil
}

func (m *MockCache) Exist(ctx context.Context, key string) (bool, error) {
	_, ok := m.data[key]
	return ok, nil
}

var n atomic.Int64

type TestDao[T string] struct {
	data map[string]interface{}
}

func (d *TestDao[T]) Get(ctx context.Context, key string) (*T, error) {
	time.Sleep(time.Millisecond * 200)
	n.Add(1)
	if v, ok := d.data[key]; ok {
		cast := v.(T)
		return &cast, nil
	}
	return nil, nil
}

var _ = Describe("mcache", Ordered, func() {

	var mcache *MCache
	var testDao *TestDao[string]
	var mockCache *MockCache

	BeforeAll(func() {
		testDao = &TestDao[string]{
			data: map[string]interface{}{
				"foo": "bar",
			},
		}

		mockCache = &MockCache{
			data: make(map[string]interface{}),
		}

		mcache = NewMCache(&Options{
			L1Size: 100,
			L1TTL:  time.Second,
			L2:     mockCache,
		})
		Set(mcache)
	})

	BeforeEach(func() {
		mcache.Invalidate(context.TODO(), "foo")
	})

	It("sanity", func() {
		value, err := Load(context.TODO(), "foo", nil, testDao.Get, "foo")
		assert.NoError(GinkgoT(), err)
		assert.Equal(GinkgoT(), "bar", *value)

		// result should be cached to L2
		layer2value, ok := mockCache.data["foo"]
		assert.True(GinkgoT(), ok)
		assert.Equal(GinkgoT(), "bar", *layer2value.(*string))

		// result should be cached to L1
		layer1value, ok := mcache.l1.Get("foo")
		assert.True(GinkgoT(), ok)
		assert.Equal(GinkgoT(), "bar", *layer1value.(*string))
	})

	It("mutex", func() {
		group := sync.WaitGroup{}
		group2 := sync.WaitGroup{}
		group.Add(1)
		for i := 0; i < 1000; i++ {
			group2.Add(1)
			go func() {
				group.Wait()
				value, err := Load(context.TODO(), "foo", nil, testDao.Get, "foo")
				assert.NoError(GinkgoT(), err)
				assert.Equal(GinkgoT(), "bar", *value)
				group2.Done()
			}()
		}
		before := n.Load()
		group.Done()
		group2.Wait()
		assert.EqualValues(GinkgoT(), 1, n.Load()-before)

		// remove layer1
		before = n.Load()
		mcache.InvalidateL1(context.TODO(), "foo")
		value, err := Load(context.TODO(), "foo", nil, testDao.Get, "foo")
		assert.NoError(GinkgoT(), err)
		assert.Equal(GinkgoT(), "bar", *value)
		assert.EqualValues(GinkgoT(), 0, n.Load()-before)
	})

})

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "MCache Suite")
}
