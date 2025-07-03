package queue

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/pkg/log"
	"github.com/webhookx-io/webhookx/pkg/taskqueue"
	"testing"
	"time"
)

var _ = Describe("processRequeue", Ordered, func() {

	var queue taskqueue.TaskQueue

	BeforeAll(func() {
		cfg, err := config.Init()
		assert.Nil(GinkgoT(), err)
		log, err := log.NewZapLogger(&cfg.Log)

		client := cfg.Redis.GetClient()
		client.Del(context.TODO(), "webhookx:test-queue")
		client.Del(context.TODO(), "webhookx:test-queue_invisible")
		client.Del(context.TODO(), "webhookx:test-queue_data")

		queue = taskqueue.NewRedisQueue(taskqueue.RedisTaskQueueOptions{
			QueueName:          "webhookx:test-queue",
			InvisibleQueueName: "webhookx:test-queue_invisible",
			QueueDataName:      "webhookx:test-queue_data",
			VisibilityTimeout:  time.Second * 3,
			Client:             client,
		}, log, nil)
	})

	AfterAll(func() {

	})

	It("sanity", func() {
		messages := []*taskqueue.TaskMessage{
			{
				ID:   "one",
				Data: "data-one",
			},
			{
				ID:   "two",
				Data: "data-two",
			},
			{
				ID:   "three",
				Data: "data-three",
			},
		}

		for _, msg := range messages {
			msg.ScheduledAt = time.Now()
			err := queue.Add(context.TODO(), []*taskqueue.TaskMessage{msg})
			assert.Nil(GinkgoT(), err)
			time.Sleep(time.Millisecond * 1)
		}

		size, err := queue.Size(context.TODO())
		assert.Nil(GinkgoT(), err)
		assert.EqualValues(GinkgoT(), len(messages), size)

		tasks, err := queue.Get(context.TODO(), &taskqueue.GetOptions{Count: 1})
		assert.Nil(GinkgoT(), err)
		assert.Len(GinkgoT(), tasks, 1)
		assert.Equal(GinkgoT(), tasks[0].ID, "one")
		var data string
		tasks[0].UnmarshalData(&data)
		assert.Equal(GinkgoT(), data, "data-one")
		err = queue.Delete(context.TODO(), tasks[0])
		assert.Nil(GinkgoT(), err)

		tasks, err = queue.Get(context.TODO(), &taskqueue.GetOptions{Count: 10})
		assert.Nil(GinkgoT(), err)
		assert.Len(GinkgoT(), tasks, 2)
		assert.Equal(GinkgoT(), tasks[0].ID, "two")
		assert.Equal(GinkgoT(), tasks[1].ID, "three")
		err = queue.Delete(context.TODO(), tasks[0])
		assert.Nil(GinkgoT(), err)
		err = queue.Delete(context.TODO(), tasks[1])
		assert.Nil(GinkgoT(), err)

		size, err = queue.Size(context.TODO())
		assert.Nil(GinkgoT(), err)
		assert.EqualValues(GinkgoT(), 0, size)
	})

	It("in-flight messages should be consumable after reaching the timeout", func() {
		err := queue.Add(context.TODO(), []*taskqueue.TaskMessage{
			{ID: "task-timeout", Data: "data", ScheduledAt: time.Now()},
		})
		assert.Nil(GinkgoT(), err)

		tasks, err := queue.Get(context.TODO(), &taskqueue.GetOptions{Count: 1})
		assert.Nil(GinkgoT(), err)
		assert.Len(GinkgoT(), tasks, 1)

		size, err := queue.Size(context.TODO())
		assert.Nil(GinkgoT(), err)
		assert.EqualValues(GinkgoT(), 0, size)

		time.Sleep(time.Second * 2)

		// this should be zero as we're not reach the timeout yet
		size, err = queue.Size(context.TODO())
		assert.Nil(GinkgoT(), err)
		assert.EqualValues(GinkgoT(), 0, size)

		time.Sleep(time.Second * 3) // 5 seconds later

		tasks, err = queue.Get(context.TODO(), &taskqueue.GetOptions{Count: 1})
		assert.Nil(GinkgoT(), err)
		assert.Len(GinkgoT(), tasks, 1)

		size, err = queue.Size(context.TODO())
		assert.Nil(GinkgoT(), err)
		assert.EqualValues(GinkgoT(), 0, size)
	})
})

func TestQueue(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Queue Suite")
}
