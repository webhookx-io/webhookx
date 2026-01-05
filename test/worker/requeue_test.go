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
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/db/query"
	"github.com/webhookx-io/webhookx/pkg/metrics"
	"github.com/webhookx-io/webhookx/pkg/schedule"
	"github.com/webhookx-io/webhookx/pkg/tracing"
	"github.com/webhookx-io/webhookx/service"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/test/helper/factory"
	"github.com/webhookx-io/webhookx/test/mocks"
	"github.com/webhookx-io/webhookx/utils"
	"github.com/webhookx-io/webhookx/worker"
	"go.uber.org/mock/gomock"
)

var _ = Describe("processRequeue", Ordered, func() {

	var db *db.DB
	var w *worker.Worker
	var ctrl *gomock.Controller
	var queue *mocks.MockTaskQueue
	var tracer *tracing.Tracer
	var scheduler = schedule.NewScheduler()
	endpoint := factory.Endpoint()

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
		srv := service.NewService(service.Options{
			DB:        db,
			TaskQueue: queue,
		})
		w = worker.NewWorker(worker.Options{
			DB:          db,
			Metrics:     metrics,
			Tracer:      tracer,
			EventBus:    mocks.MockBus{},
			Srv:         srv,
			RedisClient: cfg.Redis.GetClient(),
			Scheduler:   scheduler,
		})

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
		db.DB.MustExec("update attempts set created_at = created_at - INTERVAL '60 SECOND'")

		w.Start()
		scheduler.Start()
	})

	AfterAll(func() {
		w.Stop()
		ctrl.Finish()
	})

	It("all attempts should become QUEUED", func() {
		time.Sleep(time.Second * 1) // wait for timer to be executed
		var q query.AttemptQuery
		q.EndpointId = utils.Pointer(endpoint.ID)
		q.Status = utils.Pointer(entities.AttemptStatusInit)
		count, err := db.Attempts.Count(context.TODO(), q.WhereMap())
		assert.NoError(GinkgoT(), err)
		assert.EqualValues(GinkgoT(), 0, count)

		q.Status = utils.Pointer(entities.AttemptStatusQueued)
		count, err = db.Attempts.Count(context.TODO(), q.WhereMap())
		assert.NoError(GinkgoT(), err)
		assert.EqualValues(GinkgoT(), 10, count)
	})
})

func TestWorker(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Worker Suite")
}
