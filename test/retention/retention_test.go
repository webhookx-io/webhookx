package retention

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/db/dao"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/services/retention"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/test/helper/factory"
	"github.com/webhookx-io/webhookx/utils"
)

var _ = Describe("retention", Ordered, func() {

	Context("sanity", func() {
		var app *app.Application

		BeforeAll(func() {
			db := helper.InitDB(true, nil)
			ws, err := helper.GetDeafultWorkspace()
			assert.NoError(GinkgoT(), err)
			eventTTL := 30*24*time.Hour + 30*time.Minute
			attemptTTL := 60*24*time.Hour + 30*time.Minute
			boundaryMargin := 5 * time.Minute

			// add 1001 expired events
			for i := 0; i < retention.BatchSize+1; i++ {
				expiredEvent := factory.EventWS(ws.ID)
				assert.NoError(GinkgoT(), db.Events.Insert(context.TODO(), expiredEvent))
				_, err = db.SqlDB().Exec("UPDATE events SET created_at = $1 WHERE id = $2", time.Now().Add(-eventTTL-boundaryMargin), expiredEvent.ID)
				assert.NoError(GinkgoT(), err)
			}

			// Active event
			activeEvent := factory.EventWS(ws.ID)
			assert.NoError(GinkgoT(), db.Events.Insert(context.TODO(), activeEvent))
			_, err = db.SqlDB().Exec("UPDATE events SET created_at = $1 WHERE id = $2", time.Now().Add(-eventTTL+boundaryMargin), activeEvent.ID)
			assert.NoError(GinkgoT(), err)

			// add 1001 expired attempts
			for i := 0; i < retention.BatchSize+1; i++ {
				expiredAttempt := &entities.Attempt{
					ID:            utils.KSUID(),
					EventId:       activeEvent.ID,
					Status:        entities.AttemptStatusSuccess,
					AttemptNumber: 1,
					BaseModel: entities.BaseModel{
						WorkspaceId: ws.ID,
					},
				}
				assert.NoError(GinkgoT(), db.Attempts.Insert(context.TODO(), expiredAttempt))
				_, err = db.SqlDB().Exec("UPDATE attempts SET created_at = $1 WHERE id = $2", time.Now().Add(-attemptTTL-boundaryMargin), expiredAttempt.ID)
				assert.NoError(GinkgoT(), err)
			}

			// Active attempt
			activeAttempt := &entities.Attempt{
				ID:            utils.KSUID(),
				EventId:       activeEvent.ID,
				Status:        entities.AttemptStatusSuccess,
				AttemptNumber: 1,
				BaseModel: entities.BaseModel{
					WorkspaceId: ws.ID,
				},
			}
			assert.NoError(GinkgoT(), db.Attempts.Insert(context.TODO(), activeAttempt))
			_, err = db.SqlDB().Exec("UPDATE attempts SET created_at = $1 WHERE id = $2", time.Now().Add(-attemptTTL+boundaryMargin), activeAttempt.ID)
			assert.NoError(GinkgoT(), err)

			app = utils.Must(helper.Start(map[string]string{
				"WEBHOOKX_RETENTION_ENABLED":      "true",
				"WEBHOOKX_RETENTION_TTL_EVENTS":   "30d30m",
				"WEBHOOKX_RETENTION_TTL_ATTEMPTS": "60d30m",
			}))
		})

		AfterAll(func() {
			app.Stop()
		})

		It("retention service should start", func() {
			assert.Eventually(GinkgoT(), func() bool {
				matched, err := helper.FileHasLine(helper.LogFile, "^.*\\[retention\\]\\s+service started.*$")
				return err == nil && matched
			}, time.Second*5, time.Microsecond*100)
		})

		It("should purge expired events and attempts when triggered", func() {
			app.Scheduler().RunNow("retention")

			db := app.DB()

			// verify expired event is deleted, active event remains
			events, err := db.Events.List(context.TODO(), &dao.Query{})
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 1, len(events))

			// verify expired attempt is deleted, active attempt remains
			attempts, err := db.Attempts.List(context.TODO(), &dao.Query{})
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 1, len(attempts))
		})
	})

	Context("errors", func() {
		var app *app.Application

		BeforeAll(func() {
			app = utils.Must(helper.Start(map[string]string{
				"WEBHOOKX_RETENTION_ENABLED": "true",
				"WEBHOOKX_ROLE":              "dp_proxy",
			}))
		})

		AfterAll(func() {
			app.Stop()
		})

		It("ignore retention configuration on data-planes", func() {
			assert.Eventually(GinkgoT(), func() bool {
				matched, err := helper.FileHasLine(helper.LogFile, "retention configuration is ignored on data-plane nodes")
				return err == nil && matched
			}, time.Second*5, time.Microsecond*100)
		})
	})

	Context("partial ttl - only events retention enabled", func() {
		var app *app.Application

		BeforeAll(func() {
			db := helper.InitDB(true, nil)
			ws, err := helper.GetDeafultWorkspace()
			assert.NoError(GinkgoT(), err)
			eventTTL := 30*24*time.Hour + 30*time.Minute
			boundaryMargin := 5 * time.Minute

			// add expired event
			expiredEvent := factory.EventWS(ws.ID)
			assert.NoError(GinkgoT(), db.Events.Insert(context.TODO(), expiredEvent))
			_, err = db.SqlDB().Exec("UPDATE events SET created_at = $1 WHERE id = $2", time.Now().Add(-eventTTL-boundaryMargin), expiredEvent.ID)
			assert.NoError(GinkgoT(), err)

			// add expired attempt
			expiredAttempt := &entities.Attempt{
				ID:            utils.KSUID(),
				EventId:       expiredEvent.ID,
				Status:        entities.AttemptStatusSuccess,
				AttemptNumber: 1,
				BaseModel: entities.BaseModel{
					WorkspaceId: ws.ID,
				},
			}
			assert.NoError(GinkgoT(), db.Attempts.Insert(context.TODO(), expiredAttempt))
			_, err = db.SqlDB().Exec("UPDATE attempts SET created_at = $1 WHERE id = $2", time.Now().Add(-eventTTL-boundaryMargin), expiredAttempt.ID)
			assert.NoError(GinkgoT(), err)

			app = utils.Must(helper.Start(map[string]string{
				"WEBHOOKX_RETENTION_ENABLED":      "true",
				"WEBHOOKX_RETENTION_TTL_EVENTS":   "30d30m",
				"WEBHOOKX_RETENTION_TTL_ATTEMPTS": "0",
			}))
		})

		AfterAll(func() {
			app.Stop()
		})

		It("should purge expired events only and keep attempts when TTL.Attempts is 0", func() {
			app.Scheduler().RunNow("retention")

			db := app.DB()

			// expired event is deleted
			events, err := db.Events.List(context.TODO(), &dao.Query{})
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 0, len(events))

			// expired attempt remains because attempts retention is disabled (TTL.Attempts == 0)
			attempts, err := db.Attempts.List(context.TODO(), &dao.Query{})
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 1, len(attempts))
		})
	})

	Context("partial ttl - only attempts retention enabled", func() {
		var app *app.Application

		BeforeAll(func() {
			db := helper.InitDB(true, nil)
			ws, err := helper.GetDeafultWorkspace()
			assert.NoError(GinkgoT(), err)
			attemptTTL := 60*24*time.Hour + 30*time.Minute
			boundaryMargin := 5 * time.Minute

			// add expired event
			expiredEvent := factory.EventWS(ws.ID)
			assert.NoError(GinkgoT(), db.Events.Insert(context.TODO(), expiredEvent))
			_, err = db.SqlDB().Exec("UPDATE events SET created_at = $1 WHERE id = $2", time.Now().Add(-attemptTTL-boundaryMargin), expiredEvent.ID)
			assert.NoError(GinkgoT(), err)

			// add expired attempt
			expiredAttempt := &entities.Attempt{
				ID:            utils.KSUID(),
				EventId:       expiredEvent.ID,
				Status:        entities.AttemptStatusSuccess,
				AttemptNumber: 1,
				BaseModel: entities.BaseModel{
					WorkspaceId: ws.ID,
				},
			}
			assert.NoError(GinkgoT(), db.Attempts.Insert(context.TODO(), expiredAttempt))
			_, err = db.SqlDB().Exec("UPDATE attempts SET created_at = $1 WHERE id = $2", time.Now().Add(-attemptTTL-boundaryMargin), expiredAttempt.ID)
			assert.NoError(GinkgoT(), err)

			app = utils.Must(helper.Start(map[string]string{
				"WEBHOOKX_RETENTION_ENABLED":      "true",
				"WEBHOOKX_RETENTION_TTL_EVENTS":   "0",
				"WEBHOOKX_RETENTION_TTL_ATTEMPTS": "60d30m",
			}))
		})

		AfterAll(func() {
			app.Stop()
		})

		It("should purge expired attempts only and keep events when TTL.Events is 0", func() {
			app.Scheduler().RunNow("retention")

			db := app.DB()

			// expired attempt is deleted
			attempts, err := db.Attempts.List(context.TODO(), &dao.Query{})
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 0, len(attempts))

			// expired event remains because events retention is disabled (TTL.Events == 0)
			events, err := db.Events.List(context.TODO(), &dao.Query{})
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 1, len(events))
		})
	})
})

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Retention Suite")
}
