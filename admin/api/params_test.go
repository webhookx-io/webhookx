package api

import (
	"encoding/json"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/db/dao"
	"github.com/webhookx-io/webhookx/utils"
)

var _ = Describe("params to query", Ordered, func() {
	Context("EndpointListParams", func() {
		It("filters", func() {
			metadata := map[string]string{"foo": "bar"}
			metadataJson, _ := json.Marshal(metadata)
			params := EndpointListParams{
				Name:         utils.Pointer("test-endpoint"),
				Enabled:      utils.Pointer(true),
				CreatedAt:    utils.Pointer(int64(1000)),
				CreatedAtGT:  utils.Pointer(int64(2000)),
				CreatedAtGTE: utils.Pointer(int64(3000)),
				CreatedAtLT:  utils.Pointer(int64(4000)),
				CreatedAtLTE: utils.Pointer(int64(5000)),
				Metadata:     metadata,
			}
			query := params.Query()
			expectedWheres := []dao.Condition{
				{"name", dao.Equal, "test-endpoint"},
				{"enabled", dao.Equal, true},
				{"created_at", dao.Equal, time.UnixMilli(1000)},
				{"created_at", dao.GreaterThan, time.UnixMilli(2000)},
				{"created_at", dao.GreaterThanOrEqual, time.UnixMilli(3000)},
				{"created_at", dao.LessThan, time.UnixMilli(4000)},
				{"created_at", dao.LessThanOrEqual, time.UnixMilli(5000)},
				{"metadata", dao.JsonContain, string(metadataJson)},
			}
			assert.EqualValues(GinkgoT(), expectedWheres, query.Wheres)
		})
	})

	Context("SourceListParams", func() {
		It("filters", func() {
			metadata := map[string]string{"foo": "bar"}
			metadataJson, _ := json.Marshal(metadata)
			params := SourceListParams{
				Name:         utils.Pointer("test-source"),
				Enabled:      utils.Pointer(false),
				CreatedAt:    utils.Pointer(int64(1000)),
				CreatedAtGT:  utils.Pointer(int64(2000)),
				CreatedAtGTE: utils.Pointer(int64(3000)),
				CreatedAtLT:  utils.Pointer(int64(4000)),
				CreatedAtLTE: utils.Pointer(int64(5000)),
				Metadata:     metadata,
			}
			query := params.Query()
			expectedWheres := []dao.Condition{
				{"name", dao.Equal, "test-source"},
				{"enabled", dao.Equal, false},
				{"created_at", dao.Equal, time.UnixMilli(1000)},
				{"created_at", dao.GreaterThan, time.UnixMilli(2000)},
				{"created_at", dao.GreaterThanOrEqual, time.UnixMilli(3000)},
				{"created_at", dao.LessThan, time.UnixMilli(4000)},
				{"created_at", dao.LessThanOrEqual, time.UnixMilli(5000)},
				{"metadata", dao.JsonContain, string(metadataJson)},
			}
			assert.EqualValues(GinkgoT(), expectedWheres, query.Wheres)
		})
	})

	Context("PluginListParams", func() {
		It("filters", func() {
			metadata := map[string]string{"foo": "bar"}
			metadataJson, _ := json.Marshal(metadata)
			params := PluginListParams{
				Name:         utils.Pointer("test-plugin"),
				Enabled:      utils.Pointer(true),
				CreatedAt:    utils.Pointer(int64(1000)),
				CreatedAtGT:  utils.Pointer(int64(2000)),
				CreatedAtGTE: utils.Pointer(int64(3000)),
				CreatedAtLT:  utils.Pointer(int64(4000)),
				CreatedAtLTE: utils.Pointer(int64(5000)),
				Metadata:     metadata,
				EndpointId:   utils.Pointer("ep_123"),
				SourceId:     utils.Pointer("src_123"),
			}
			query := params.Query()
			expectedWheres := []dao.Condition{
				{"name", dao.Equal, "test-plugin"},
				{"enabled", dao.Equal, true},
				{"created_at", dao.Equal, time.UnixMilli(1000)},
				{"created_at", dao.GreaterThan, time.UnixMilli(2000)},
				{"created_at", dao.GreaterThanOrEqual, time.UnixMilli(3000)},
				{"created_at", dao.LessThan, time.UnixMilli(4000)},
				{"created_at", dao.LessThanOrEqual, time.UnixMilli(5000)},
				{"metadata", dao.JsonContain, string(metadataJson)},
				{"endpoint_id", dao.Equal, "ep_123"},
				{"source_id", dao.Equal, "src_123"},
			}
			assert.EqualValues(GinkgoT(), expectedWheres, query.Wheres)
		})
	})

	Context("AttemptListParams", func() {
		It("filters", func() {
			params := AttemptListParams{
				CreatedAt:      utils.Pointer(int64(1000)),
				CreatedAtGT:    utils.Pointer(int64(1100)),
				CreatedAtGTE:   utils.Pointer(int64(1200)),
				CreatedAtLT:    utils.Pointer(int64(1300)),
				CreatedAtLTE:   utils.Pointer(int64(1400)),
				EventId:        utils.Pointer("evt_123"),
				EndpointId:     utils.Pointer("ep_123"),
				Status:         utils.Pointer("SUCCESS"),
				AttemptedAt:    utils.Pointer(int64(2000)),
				AttemptedAtGT:  utils.Pointer(int64(2100)),
				AttemptedAtGTE: utils.Pointer(int64(2200)),
				AttemptedAtLT:  utils.Pointer(int64(2300)),
				AttemptedAtLTE: utils.Pointer(int64(2400)),
			}
			query := params.Query()
			expectedWheres := []dao.Condition{
				{"created_at", dao.Equal, time.UnixMilli(1000)},
				{"created_at", dao.GreaterThan, time.UnixMilli(1100)},
				{"created_at", dao.GreaterThanOrEqual, time.UnixMilli(1200)},
				{"created_at", dao.LessThan, time.UnixMilli(1300)},
				{"created_at", dao.LessThanOrEqual, time.UnixMilli(1400)},
				{"event_id", dao.Equal, "evt_123"},
				{"endpoint_id", dao.Equal, "ep_123"},
				{"status", dao.Equal, "SUCCESS"},
				{"attempted_at", dao.Equal, time.UnixMilli(2000)},
				{"attempted_at", dao.GreaterThan, time.UnixMilli(2100)},
				{"attempted_at", dao.GreaterThanOrEqual, time.UnixMilli(2200)},
				{"attempted_at", dao.LessThan, time.UnixMilli(2300)},
				{"attempted_at", dao.LessThanOrEqual, time.UnixMilli(2400)},
			}
			assert.EqualValues(GinkgoT(), expectedWheres, query.Wheres)
		})
	})

	Context("WorkspaceListParams", func() {
		It("filters", func() {
			metadata := map[string]string{"foo": "bar"}
			metadataJson, _ := json.Marshal(metadata)
			params := WorkspaceListParams{
				Name:         utils.Pointer("test-workspace"),
				CreatedAt:    utils.Pointer(int64(1000)),
				CreatedAtGT:  utils.Pointer(int64(2000)),
				CreatedAtGTE: utils.Pointer(int64(3000)),
				CreatedAtLT:  utils.Pointer(int64(4000)),
				CreatedAtLTE: utils.Pointer(int64(5000)),
				Metadata:     metadata,
			}
			query := params.Query()
			expectedWheres := []dao.Condition{
				{"name", dao.Equal, "test-workspace"},
				{"created_at", dao.Equal, time.UnixMilli(1000)},
				{"created_at", dao.GreaterThan, time.UnixMilli(2000)},
				{"created_at", dao.GreaterThanOrEqual, time.UnixMilli(3000)},
				{"created_at", dao.LessThan, time.UnixMilli(4000)},
				{"created_at", dao.LessThanOrEqual, time.UnixMilli(5000)},
				{"metadata", dao.JsonContain, string(metadataJson)},
			}
			assert.EqualValues(GinkgoT(), expectedWheres, query.Wheres)
		})
	})

	Context("EventListParams", func() {
		It("filters", func() {
			params := EventListParams{
				CreatedAt:     utils.Pointer(int64(1000)),
				CreatedAtGT:   utils.Pointer(int64(1100)),
				CreatedAtGTE:  utils.Pointer(int64(1200)),
				CreatedAtLT:   utils.Pointer(int64(1300)),
				CreatedAtLTE:  utils.Pointer(int64(1400)),
				EventType:     utils.Pointer("user.created"),
				UniqueId:      utils.Pointer("uid_123"),
				IngestedAt:    utils.Pointer(int64(2000)),
				IngestedAtGT:  utils.Pointer(int64(2100)),
				IngestedAtGTE: utils.Pointer(int64(2200)),
				IngestedAtLT:  utils.Pointer(int64(2300)),
				IngestedAtLTE: utils.Pointer(int64(2400)),
			}
			query := params.Query()
			expectedWheres := []dao.Condition{
				{"created_at", dao.Equal, time.UnixMilli(1000)},
				{"created_at", dao.GreaterThan, time.UnixMilli(1100)},
				{"created_at", dao.GreaterThanOrEqual, time.UnixMilli(1200)},
				{"created_at", dao.LessThan, time.UnixMilli(1300)},
				{"created_at", dao.LessThanOrEqual, time.UnixMilli(1400)},
				{"event_type", dao.Equal, "user.created"},
				{"unique_id", dao.Equal, "uid_123"},
				{"ingested_at", dao.Equal, time.UnixMilli(2000)},
				{"ingested_at", dao.GreaterThan, time.UnixMilli(2100)},
				{"ingested_at", dao.GreaterThanOrEqual, time.UnixMilli(2200)},
				{"ingested_at", dao.LessThan, time.UnixMilli(2300)},
				{"ingested_at", dao.LessThanOrEqual, time.UnixMilli(2400)},
			}
			assert.EqualValues(GinkgoT(), expectedWheres, query.Wheres)
		})
	})
})

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Params")
}
