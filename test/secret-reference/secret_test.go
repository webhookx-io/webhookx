package secret_reference

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/test/helper"
)

var _ = Describe("Secret", Ordered, func() {

	Context("errors", func() {
		It("should return error when setting unsupported provider", func() {
			app, err := helper.Start(map[string]string{
				"WEBHOOKX_SECRET_PROVIDERS": "@default,test",
			})
			assert.Nil(GinkgoT(), app)
			assert.EqualError(GinkgoT(), err, "invalid configuration: invalid provider: test")
		})

		It("should return error when using unknown providers", func() {
			_, err := helper.LoadConfig(helper.LoadConfigOptions{
				Envs: map[string]string{
					"WEBHOOKX_SECRET_VAULT_AUTHN_TOKEN_TOKEN": "root",
					"WEBHOOKX_DATABASE_HOST":                  "{secret://unknown/webhookx/config.key_boolean}",
				},
			})
			assert.EqualError(GinkgoT(), err, "failed to resolve reference value '{secret://unknown/webhookx/config.key_boolean}': provider 'unknown' is not supported")
		})

		It("should return error when provider is not enabled", func() {
			_, err := helper.LoadConfig(helper.LoadConfigOptions{
				Envs: map[string]string{
					"WEBHOOKX_SECRET_VAULT_AUTHN_TOKEN_TOKEN": "root",
					"WEBHOOKX_SECRET_PROVIDERS":               "vault",
					"WEBHOOKX_DATABASE_HOST":                  "{secret://aws/webhookx/config.key_boolean}",
				},
			})
			assert.EqualError(GinkgoT(), err, "failed to resolve reference value '{secret://aws/webhookx/config.key_boolean}': provider 'aws' is not supported")
		})
	})

})

func Test(t *testing.T) {
	gomega.RegisterFailHandler(Fail)
	RunSpecs(t, "SecretReference Suite")
}
