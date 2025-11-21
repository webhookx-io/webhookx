package cache

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/pkg/cache"
)

var _ = Describe("cache", Ordered, func() {

	var redisCache cache.Cache

	BeforeAll(func() {
		cfg := config.New()
		err := config.Load("", cfg)
		assert.NoError(GinkgoT(), err)
		redisCache = cache.NewRedisCache(cfg.Redis.GetClient())
	})

	It("sanity", func() {
		ctx := context.TODO()

		err := redisCache.Put(ctx, "foo", "bar", time.Second*5)
		assert.NoError(GinkgoT(), err)

		var value string
		exist, err := redisCache.Get(ctx, "foo", &value)
		assert.NoError(GinkgoT(), err)
		assert.True(GinkgoT(), exist)
		assert.Equal(GinkgoT(), "bar", value)

		err = redisCache.Remove(ctx, "foo")
		assert.NoError(GinkgoT(), err)

		exist, err = redisCache.Get(ctx, "foo", &value)
		assert.NoError(GinkgoT(), err)
		assert.False(GinkgoT(), exist)
	})
})

func TestCache(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cache Suite")
}
