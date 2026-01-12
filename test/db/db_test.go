package db

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/test/helper/factory"
	"github.com/webhookx-io/webhookx/utils"
)

var _ = Describe("DB", Ordered, func() {
	Context("DAO", func() {
		Context("events", func() {
			var db *db.DB
			BeforeEach(func() {
				db = helper.InitDB(true, nil)
			})
			It("BatchInsertIgnoreConflict", func() {
				id1 := utils.KSUID()
				id2 := utils.KSUID()
				id3 := utils.KSUID()
				err := db.Events.Insert(context.TODO(), factory.Event(func(o *entities.Event) { o.ID = id1 }))
				assert.Nil(GinkgoT(), err)
				ids, err := db.Events.BatchInsertIgnoreConflict(context.TODO(), []*entities.Event{
					factory.Event(func(o *entities.Event) { o.ID = id1 }), // id duplicated
					factory.Event(func(o *entities.Event) { o.ID = id2 }),
					factory.Event(func(o *entities.Event) { o.ID = id3 }),
				})
				assert.Equal(GinkgoT(), 2, len(ids))
				assert.Equal(GinkgoT(), id2, ids[0])
				assert.Equal(GinkgoT(), id3, ids[1])
			})
		})
	})
})

func TestDB(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "DB Suite")
}
