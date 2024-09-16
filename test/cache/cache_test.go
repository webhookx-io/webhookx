package cache

import (
	"context"
	"encoding/json"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/pkg/cache"
	"github.com/webhookx-io/webhookx/pkg/types"
	"github.com/webhookx-io/webhookx/utils"
	"testing"
	"time"
)

var _ = Describe("/attempts", Ordered, func() {

	var redisCache cache.Cache

	BeforeAll(func() {
		cfg, err := config.Init()
		assert.NoError(GinkgoT(), err)
		redisCache = cache.NewRedisCache(cfg.RedisConfig.GetClient())
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

	Context("cache.Get", func() {
		counter := 0
		mockWorkspace := &entities.Workspace{
			ID:          utils.KSUID(),
			Name:        utils.Pointer("mock workpsace"),
			Description: utils.Pointer("mock workpsace"),
			CreatedAt:   types.NewTime(time.Now()),
			UpdatedAt:   types.NewTime(time.Now()),
		}
		key := "workspaces:" + mockWorkspace.ID

		It("sanity", func() {
			_, err := cache.Get(redisCache, context.TODO(), key, func(ctx context.Context) (*entities.Workspace, error) {
				counter++
				return mockWorkspace, nil
			}, nil)
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 1, counter)

			exist, err := redisCache.Exist(context.TODO(), key)
			assert.NoError(GinkgoT(), err)
			assert.True(GinkgoT(), exist)

			workspace, err := cache.Get(redisCache, context.TODO(), key, func(ctx context.Context) (*entities.Workspace, error) {
				counter++
				return mockWorkspace, nil
			}, nil)
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 1, counter)

			assert.JSONEq(GinkgoT(),
				string(utils.Must(json.Marshal(mockWorkspace))),
				string(utils.Must(json.Marshal(workspace))))
		})

	})

})

func TestCache(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cache Suite")
}
