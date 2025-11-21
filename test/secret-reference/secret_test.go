package secret_reference

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/test/helper"
)

var _ = Describe("Secret", Ordered, func() {

	Context("errors", func() {
		It("should return error when using unknown providers", func() {
			var cfg *config.Config
			var err error
			withCleanEnv(func() {
				cancel := helper.SetEnvs(nil, map[string]string{
					"WEBHOOKX_SECRET_VAULT_AUTHN_TOKEN_TOKEN": "root",

					"WEBHOOKX_DATABASE_HOST": "{secret://unknown/webhookx/config.key_boolean}",
				})
				defer cancel()
				cfg = config.New()
				err = config.NewLoader(cfg).Load()
			})
			assert.EqualError(GinkgoT(), err, "unsupported provider: {secret://unknown/webhookx/config.key_boolean}")
		})

		It("should return error when provider is not enabled", func() {
			var cfg *config.Config
			var err error
			withCleanEnv(func() {
				cancel := helper.SetEnvs(nil, map[string]string{
					"WEBHOOKX_SECRET_VAULT_AUTHN_TOKEN_TOKEN": "root",
					"WEBHOOKX_SECRET_PROVIDERS":               "vault",

					"WEBHOOKX_DATABASE_HOST": "{secret://aws/webhookx/config.key_boolean}",
				})
				defer cancel()
				cfg = config.New()
				err = config.NewLoader(cfg).Load()
			})
			// TODO change message
			assert.EqualError(GinkgoT(), err, "unsupported provider: {secret://aws/webhookx/config.key_boolean}")
		})
	})

})

func Test(t *testing.T) {
	gomega.RegisterFailHandler(Fail)
	RunSpecs(t, "SecretReference Suite")
}
