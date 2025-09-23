package db

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/test/helper/factory"
	"github.com/webhookx-io/webhookx/utils"
	"testing"
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
				id4 := utils.KSUID()
				err := db.Events.Insert(context.TODO(), factory.EventP(func(o *entities.Event) { o.ID = id1; o.UniqueId = utils.Pointer("key1") }))
				assert.Nil(GinkgoT(), err)
				err = db.Events.Insert(context.TODO(), factory.EventP(func(o *entities.Event) { o.ID = id2; o.UniqueId = utils.Pointer("key2") }))
				assert.Nil(GinkgoT(), err)
				ids, err := db.Events.BatchInsertIgnoreConflict(context.TODO(), []*entities.Event{
					factory.EventP(func(o *entities.Event) { o.ID = id1; o.UniqueId = utils.Pointer("key0") }),           // id duplicated
					factory.EventP(func(o *entities.Event) { o.ID = utils.KSUID(); o.UniqueId = utils.Pointer("key1") }), // key duplicated
					factory.EventP(func(o *entities.Event) { o.ID = utils.KSUID(); o.UniqueId = utils.Pointer("key2") }), // key duplicated
					factory.EventP(func(o *entities.Event) { o.ID = id3; o.UniqueId = utils.Pointer("key3") }),           // this should be returned
					factory.EventP(func(o *entities.Event) { o.ID = id4; o.UniqueId = utils.Pointer("key4") }),           // this should be returned
				})
				assert.Equal(GinkgoT(), id3, ids[0])
				assert.Equal(GinkgoT(), id4, ids[1])
			})
		})
	})
})

func TestDB(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "DB Suite")
}
