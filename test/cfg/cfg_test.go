package cfg

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/utils"
	"strings"
	"testing"
)

var _ = Describe("Configuration", Ordered, func() {

	var app *app.Application

	BeforeAll(func() {
		app = utils.Must(helper.Start(map[string]string{
			"WEBHOOKX_DATABASE_MAX_POOL_SIZE": "0",
			"WEBHOOKX_DATABASE_MAX_LIFETIME":  "3600",
			"WEBHOOKX_DATABASE_PARAMETERS":    "application_name=foo&sslmode=disable&connect_timeout=30",
		}))
	})

	AfterAll(func() {
		app.Stop()
	})

	It("database configuration", func() {
		assert.EqualValues(GinkgoT(), 0, app.Config().Database.MaxPoolSize)
		assert.EqualValues(GinkgoT(), 3600, app.Config().Database.MaxLifetime)
		assert.Equal(GinkgoT(), "application_name=foo&sslmode=disable&connect_timeout=30", app.Config().Database.Parameters)
		assert.True(GinkgoT(), strings.HasSuffix(app.Config().Database.GetDSN(), app.Config().Database.Parameters))
	})

})

func TestConfiguration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Configuration Suite")
}
