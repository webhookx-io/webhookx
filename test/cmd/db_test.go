package cmd

import (
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/cmd"
	"github.com/webhookx-io/webhookx/db/migrator"
	"github.com/webhookx-io/webhookx/test/helper"
)

var statusOutputInit = `1 init (⏳ pending)
2 attempts (⏳ pending)
3 create_attempt_details_table (⏳ pending)
4 plugins (⏳ pending)
5 fix_attempt_details (⏳ pending)
6 async_ingestion (⏳ pending)
7 plugins_source_id (⏳ pending)
8 metadata (⏳ pending)
9 timestamp (⏳ pending)
10 ratelimit (⏳ pending)
Summary:
  Current version: 0
  Dirty: false
  Executed: 0
  Pending: 10
`

var statusOutputDone = `1 init (✅ executed)
2 attempts (✅ executed)
3 create_attempt_details_table (✅ executed)
4 plugins (✅ executed)
5 fix_attempt_details (✅ executed)
6 async_ingestion (✅ executed)
7 plugins_source_id (✅ executed)
8 metadata (✅ executed)
9 timestamp (✅ executed)
10 ratelimit (✅ executed)
Summary:
  Current version: 10
  Dirty: false
  Executed: 10
  Pending: 0
`

var _ = Describe("db", Ordered, func() {

	var m *migrator.Migrator
	BeforeAll(func() {
		m = migrator.New(helper.DB().DB.DB, nil)
	})

	Context("status", func() {
		It("sanity", func() {
			assert.Nil(GinkgoT(), m.Reset())
			output, err := executeCommand(cmd.NewRootCmd(), "db", "status")
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), statusOutputInit, output)

			assert.Nil(GinkgoT(), m.Up())
			output, err = executeCommand(cmd.NewRootCmd(), "db", "status")
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), statusOutputDone, output)
		})
	})

	Context("up", func() {
		It("sanity", func() {
			assert.Nil(GinkgoT(), m.Reset())
			output, err := executeCommand(cmd.NewRootCmd(), "db", "up")
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), "database is up-to-date\n", output)

			// runs up again
			output, err = executeCommand(cmd.NewRootCmd(), "db", "up")
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), "database is up-to-date\n", output)
		})
	})

	Context("reset", func() {
		It("with --yes flag", func() {
			output, err := executeCommand(cmd.NewRootCmd(), "db", "reset", "--yes")
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), "resetting database...\ndatabase successfully reset\n", output)
		})

		It("without --yes flag", func() {
			output, err := executeCommand(cmd.NewRootCmd(), "db", "reset")
			assert.NotNil(GinkgoT(), err)
			assert.Equal(GinkgoT(), "canceled", err.Error())
			assert.Equal(GinkgoT(), "> Are you sure? This operation is irreversible. [Y/N] Error: canceled\n", output)
		})
	})
})
