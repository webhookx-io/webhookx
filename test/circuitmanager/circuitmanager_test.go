package circuitmanager

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/worker/circuitbreaker"
	"github.com/webhookx-io/webhookx/worker/circuitbreaker/metrics"
)

func redisClient() *redis.Client {
	cfg, err := helper.LoadConfig(helper.LoadConfigOptions{
		Envs: helper.NewTestEnv(nil),
	})
	if err != nil {
		panic(err)
	}
	return cfg.Redis.GetClient()
}

var _ = Describe("CircuitBreaker Manager", Ordered, func() {

	Context("windowsize < 3600", func() {
		manager := circuitbreaker.NewManager(
			circuitbreaker.WithRedisClient(redisClient()),
			circuitbreaker.WithTimeWindowSize(60),
			circuitbreaker.WithFailureRateThreshold(80),
			circuitbreaker.WithMinimumRequestThreshold(5),
			circuitbreaker.WithFlushInterval(time.Second),
		)

		It("sanity", func() {
			redisClient().FlushDB(context.TODO())

			manager.Record(time.Now().Add(-time.Second), "test", metrics.Success)
			manager.Record(time.Now().Add(-time.Second), "test", metrics.Error)
			manager.Record(time.Now().Add(-time.Second), "test", metrics.Error)
			manager.Record(time.Now().Add(-time.Second), "test", metrics.Error)
			manager.Record(time.Now().Add(-time.Second), "test", metrics.Error)

			err := manager.Flush(context.TODO())
			assert.NoError(GinkgoT(), err)

			cb, err := manager.GetCircuitBreaker(context.TODO(), "test")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), "test", cb.Name())
			assert.EqualValues(GinkgoT(), 1, cb.Metric().Success)
			assert.EqualValues(GinkgoT(), 4, cb.Metric().Error)
			assert.EqualValues(GinkgoT(), 5, cb.Metric().TotalRequest())
			assert.Equal(GinkgoT(), 0.8, cb.Metric().FailureRate())
			assert.Equal(GinkgoT(), circuitbreaker.StateOpen, cb.State())
		})
	})

	Context("windowsize >= 3600", func() {
		manager := circuitbreaker.NewManager(
			circuitbreaker.WithRedisClient(redisClient()),
			circuitbreaker.WithTimeWindowSize(3600),
			circuitbreaker.WithFailureRateThreshold(80),
			circuitbreaker.WithMinimumRequestThreshold(5),
			circuitbreaker.WithFlushInterval(time.Second),
		)

		BeforeAll(func() {
			redisClient().FlushDB(context.TODO())
			m := circuitbreaker.NewManager(
				circuitbreaker.WithRedisClient(redisClient()),
				circuitbreaker.WithTimeWindowSize(3600),
				circuitbreaker.WithFailureRateThreshold(80),
				circuitbreaker.WithMinimumRequestThreshold(5),
				circuitbreaker.WithFlushInterval(time.Second),
			)
			prevHour := time.Now().Add(-time.Hour)
			for i := 0; i < 100; i++ {
				m.Record(prevHour, "test", metrics.Success)
			}
			err := m.Flush(context.TODO())
			assert.NoError(GinkgoT(), err)
		})

		It("sanity", func() {
			manager.Record(time.Now().Add(-time.Second), "test", metrics.Success)
			manager.Record(time.Now().Add(-time.Second), "test", metrics.Error)
			manager.Record(time.Now().Add(-time.Second), "test", metrics.Error)
			manager.Record(time.Now().Add(-time.Second), "test", metrics.Error)
			manager.Record(time.Now().Add(-time.Second), "test", metrics.Error)
			err := manager.Flush(context.TODO())
			assert.NoError(GinkgoT(), err)

			cb, err := manager.GetCircuitBreaker(context.TODO(), "test")
			assert.NoError(GinkgoT(), err)
			assert.True(GinkgoT(), cb.Metric().Success > 1)
			assert.EqualValues(GinkgoT(), 4, cb.Metric().Error)
			assert.True(GinkgoT(), cb.Metric().TotalRequest() > 5)
			assert.True(GinkgoT(), cb.Metric().FailureRate() < 0.8)
			assert.Equal(GinkgoT(), circuitbreaker.StateClosed, cb.State())

		})
	})
})

func Test(t *testing.T) {
	gomega.RegisterFailHandler(Fail)
	RunSpecs(t, "CircuitBreaker Manager Suite")
}
