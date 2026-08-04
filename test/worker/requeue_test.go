package worker

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/config/modules"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/dao"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/pkg/metrics"
	"github.com/webhookx-io/webhookx/pkg/ratelimiter"
	"github.com/webhookx-io/webhookx/services"
	"github.com/webhookx-io/webhookx/services/schedule"
	"github.com/webhookx-io/webhookx/services/task"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/test/helper/factory"
	"github.com/webhookx-io/webhookx/test/mocks"
	"github.com/webhookx-io/webhookx/utils"
	"github.com/webhookx-io/webhookx/worker"
	"github.com/webhookx-io/webhookx/worker/circuitbreaker"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

var _ = Describe("processRequeue", Ordered, func() {

	var db *db.DB
	var w *worker.Worker
	var ctrl *gomock.Controller
	var queue *mocks.MockTaskQueue
	var scheduler = schedule.NewCronScheduler()
	endpoint := factory.Endpoint()
	var canceledAttemptID string

	BeforeAll(func() {
		cfg, err := helper.LoadConfig(helper.LoadConfigOptions{
			Envs: helper.NewTestEnv(nil),
		})
		assert.NoError(GinkgoT(), err)
		db = helper.InitDB(true, nil)

		// setup MockTaskQueue
		ctrl = gomock.NewController(GinkgoT())
		queue = mocks.NewMockTaskQueue(ctrl)
		queue.EXPECT().Get(gomock.Any(), gomock.Any()).AnyTimes()
		queue.EXPECT().Delete(gomock.Any(), gomock.Any()).AnyTimes()
		queue.EXPECT().Add(gomock.Any(), gomock.Any()).Times(1)

		metrics, err := metrics.New(modules.MetricsConfig{}, scheduler)
		assert.NoError(GinkgoT(), err)
		services := &services.Services{
			Scheduler:   scheduler,
			EventBus:    mocks.MockBus{},
			Metrics:     metrics,
			Task:        task.NewTaskService(zap.S(), db, queue),
			RateLimiter: ratelimiter.NewRedisLimiter(cfg.Redis.GetClient()),
		}
		w = worker.NewWorker(worker.Options{
			DB:                    db,
			CircuitBreakerManager: circuitbreaker.NewManager(),
		}, services)

		// data
		ws := utils.Must(db.Workspaces.GetDefault(context.TODO()))
		endpoint.WorkspaceId = ws.ID
		assert.NoError(GinkgoT(), db.Endpoints.Insert(context.TODO(), endpoint))

		for i := 1; i <= 10; i++ {
			event := factory.EventWS(ws.ID)
			assert.NoError(GinkgoT(), db.Events.Insert(context.TODO(), event))

			attempt := entities.Attempt{
				ID:            utils.KSUID(),
				EventId:       event.ID,
				EndpointId:    endpoint.ID,
				Status:        entities.AttemptStatusInit,
				AttemptNumber: 1,
			}
			attempt.WorkspaceId = ws.ID
			assert.NoError(GinkgoT(), db.Attempts.Insert(context.TODO(), &attempt))
		}

		canceledAttempt := entities.Attempt{
			ID:            utils.KSUID(),
			EventId:       utils.KSUID(),
			EndpointId:    endpoint.ID,
			Status:        entities.AttemptStatusInit,
			AttemptNumber: 1,
		}
		canceledAttempt.WorkspaceId = ws.ID
		assert.NoError(GinkgoT(), db.Attempts.Insert(context.TODO(), &canceledAttempt))
		canceledAttemptID = canceledAttempt.ID

		db.DB.MustExec("update attempts set created_at = created_at - INTERVAL '60 SECOND'")

		w.Start()
		scheduler.Start()
	})

	AfterAll(func() {
		w.Stop(context.TODO())
		ctrl.Finish()
	})

	It("all valid attempts should become QUEUED", func() {
		time.Sleep(time.Second * 1) // wait for timer to be executed
		var q dao.AttemptQuery
		q.EndpointId = new(endpoint.ID)
		q.Status = new(entities.AttemptStatusInit)
		count, err := db.Attempts.Count(context.TODO(), q.ToQuery())
		assert.NoError(GinkgoT(), err)
		assert.EqualValues(GinkgoT(), 0, count)

		q.Status = new(entities.AttemptStatusQueued)
		count, err = db.Attempts.Count(context.TODO(), q.ToQuery())
		assert.NoError(GinkgoT(), err)
		assert.EqualValues(GinkgoT(), 10, count)
	})

	It("attempt with missing event should become CANCELED with EVENT_NOT_FOUND error code", func() {
		att, err := db.Attempts.Get(context.TODO(), canceledAttemptID)
		assert.NoError(GinkgoT(), err)
		assert.NotNil(GinkgoT(), att)
		assert.Equal(GinkgoT(), entities.AttemptStatusCanceled, att.Status)
		assert.NotNil(GinkgoT(), att.ErrorCode)
		assert.Equal(GinkgoT(), entities.AttemptErrorCodeEventNotFound, *att.ErrorCode)
	})
})

func TestWorker(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Worker Suite")
}
