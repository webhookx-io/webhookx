package secret_reference

import (
	"context"
	"fmt"
	"net/http"
	"os"

	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/test"
	"github.com/webhookx-io/webhookx/test/helper"
)

func startHTTP(handler http.HandlerFunc, addr string) *http.Server {
	s := &http.Server{
		Handler: handler,
		Addr:    addr,
	}
	go func() {
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("failed to start HTTP server: %s\n", err.Error())
		}
	}()
	return s
}

var _ = Describe("Vault", Ordered, func() {

	var licenserCancel func()
	BeforeAll(func() {
		vaultClient := helper.VaultClient()
		data := map[string]interface{}{
			"key_string":  "value",
			"key_integer": 1,
			"key_boolean": true,
			"key_float":   0.5,
			"key_array":   []string{"a", "b", "c"},
			"key_nested": map[string]interface{}{
				"key_string":  "nested value",
				"key_integer": 100,
				"key_boolean": false,
				"key_float":   100.1,
			},
		}
		secret, err := vaultClient.KVv2("secret").Put(context.TODO(), "webhookx/config", data)
		assert.NoError(GinkgoT(), err)
		assert.NotNil(GinkgoT(), secret)

		secret, err = vaultClient.KVv2("secret").Put(context.TODO(), "webhookx/secret-deleted", map[string]interface{}{"data": "value"})
		assert.NoError(GinkgoT(), err)
		assert.NotNil(GinkgoT(), secret)

		err = vaultClient.KVv2("secret").Delete(context.TODO(), "webhookx/secret-deleted")
		assert.NoError(GinkgoT(), err)

		licenserCancel = helper.MockLicenser(nil)
	})

	AfterAll(func() {
		licenserCancel()
	})

	Context("ENV", func() {
		It("references should be resolved #token", func() {
			cfg, err := helper.LoadConfig(helper.LoadConfigOptions{
				Envs: helper.NewTestEnv(map[string]string{
					"WEBHOOKX_SECRET_VAULT_AUTHN_TOKEN_TOKEN": "root",

					"WEBHOOKX_DATABASE_HOST":       "{secret://vault/webhookx/config.key_boolean}",
					"WEBHOOKX_DATABASE_PORT":       "{secret://vault/webhookx/config.key_integer}",
					"WEBHOOKX_DATABASE_USERNAME":   "{secret://vault/webhookx/config.key_string}",
					"WEBHOOKX_DATABASE_PASSWORD":   "{secret://vault/webhookx/config.key_integer}",
					"WEBHOOKX_DATABASE_DATABASE":   "{secret://vault/webhookx/config.key_float}",
					"WEBHOOKX_DATABASE_PARAMETERS": "{secret://vault/webhookx/config.key_array.2}",

					"WEBHOOKX_REDIS_HOST":     "{secret://vault/webhookx/config.key_nested.key_boolean}",
					"WEBHOOKX_REDIS_PASSWORD": "{secret://vault/webhookx/config.key_nested.key_string}",

					"WEBHOOKX_WORKER_ENABLED":        "{secret://vault/webhookx/config.key_nested.key_boolean}",
					"WEBHOOKX_TRACING_SAMPLING_RATE": "{secret://vault/webhookx/config.key_float}",
				}),
			})
			assert.NoError(GinkgoT(), err)

			assert.Equal(GinkgoT(), "true", cfg.Database.Host)
			assert.EqualValues(GinkgoT(), 1, cfg.Database.Port)
			assert.Equal(GinkgoT(), "value", cfg.Database.Username)
			assert.EqualValues(GinkgoT(), "1", cfg.Database.Password)
			assert.EqualValues(GinkgoT(), "0.5", cfg.Database.Database)
			assert.EqualValues(GinkgoT(), "c", cfg.Database.Parameters)

			assert.Equal(GinkgoT(), "false", cfg.Redis.Host)
			assert.EqualValues(GinkgoT(), "nested value", cfg.Redis.Password)

			assert.Equal(GinkgoT(), false, cfg.Worker.Enabled)
			assert.EqualValues(GinkgoT(), 0.5, cfg.Tracing.SamplingRate)
		})

		It("references should be resolved #approle", func() {
			cfg, err := helper.LoadConfig(helper.LoadConfigOptions{
				Envs: helper.NewTestEnv(map[string]string{
					"WEBHOOKX_SECRET_VAULT_AUTH_METHOD":             "approle",
					"WEBHOOKX_SECRET_VAULT_AUTHN_APPROLE_ROLE_ID":   "test-role-id",
					"WEBHOOKX_SECRET_VAULT_AUTHN_APPROLE_SECRET_ID": "test-secret-id",

					"WEBHOOKX_DATABASE_HOST":       "{secret://vault/webhookx/config.key_boolean}",
					"WEBHOOKX_DATABASE_USERNAME":   "{secret://vault/webhookx/config.key_string}",
					"WEBHOOKX_DATABASE_PASSWORD":   "{secret://vault/webhookx/config.key_integer}",
					"WEBHOOKX_DATABASE_DATABASE":   "{secret://vault/webhookx/config.key_float}",
					"WEBHOOKX_DATABASE_PARAMETERS": "{secret://vault/webhookx/config.key_array.2}",

					"WEBHOOKX_REDIS_HOST":     "{secret://vault/webhookx/config.key_nested.key_boolean}",
					"WEBHOOKX_REDIS_PASSWORD": "{secret://vault/webhookx/config.key_nested.key_string}",
				}),
			})
			assert.NoError(GinkgoT(), err)

			assert.Equal(GinkgoT(), "true", cfg.Database.Host)
			assert.Equal(GinkgoT(), "value", cfg.Database.Username)
			assert.EqualValues(GinkgoT(), "1", cfg.Database.Password)
			assert.EqualValues(GinkgoT(), "0.5", cfg.Database.Database)
			assert.EqualValues(GinkgoT(), "c", cfg.Database.Parameters)

			assert.Equal(GinkgoT(), "false", cfg.Redis.Host)
			assert.EqualValues(GinkgoT(), "nested value", cfg.Redis.Password)
		})

		It("references should be resolved #kubernetes", func() {
			server := startHTTP(func(w http.ResponseWriter, request *http.Request) {
				w.WriteHeader(http.StatusOK)
				content, _ := os.ReadFile("testdata/vault-kubernetes-response.json")
				w.Write(content)
			}, ":18888")

			cfg, err := helper.LoadConfig(helper.LoadConfigOptions{
				Envs: helper.NewTestEnv(map[string]string{
					"WEBHOOKX_SECRET_VAULT_AUTH_METHOD":                 "kubernetes",
					"WEBHOOKX_SECRET_VAULT_AUTHN_KUBERNETES_ROLE":       "test-role",
					"WEBHOOKX_SECRET_VAULT_AUTHN_KUBERNETES_TOKEN_PATH": test.FilePath("vault-k8s-token.txt"),

					"WEBHOOKX_DATABASE_HOST":       "{secret://vault/webhookx/config.key_boolean}",
					"WEBHOOKX_DATABASE_USERNAME":   "{secret://vault/webhookx/config.key_string}",
					"WEBHOOKX_DATABASE_PASSWORD":   "{secret://vault/webhookx/config.key_integer}",
					"WEBHOOKX_DATABASE_DATABASE":   "{secret://vault/webhookx/config.key_float}",
					"WEBHOOKX_DATABASE_PARAMETERS": "{secret://vault/webhookx/config.key_array.2}",

					"WEBHOOKX_REDIS_HOST":     "{secret://vault/webhookx/config.key_nested.key_boolean}",
					"WEBHOOKX_REDIS_PASSWORD": "{secret://vault/webhookx/config.key_nested.key_string}",
				}),
			})
			assert.NoError(GinkgoT(), err)

			assert.Equal(GinkgoT(), "true", cfg.Database.Host)
			assert.Equal(GinkgoT(), "value", cfg.Database.Username)
			assert.EqualValues(GinkgoT(), "1", cfg.Database.Password)
			assert.EqualValues(GinkgoT(), "0.5", cfg.Database.Database)
			assert.EqualValues(GinkgoT(), "c", cfg.Database.Parameters)

			assert.Equal(GinkgoT(), "false", cfg.Redis.Host)
			assert.EqualValues(GinkgoT(), "nested value", cfg.Redis.Password)
			assert.Nil(GinkgoT(), server.Shutdown(context.TODO()))
		})

		Context("errors", func() {
			It("should return error when approle auth failed", func() {
				_, err := helper.LoadConfig(helper.LoadConfigOptions{
					Envs: map[string]string{
						"WEBHOOKX_SECRET_VAULT_AUTH_METHOD":             "approle",
						"WEBHOOKX_SECRET_VAULT_AUTHN_APPROLE_ROLE_ID":   "test-role-id",
						"WEBHOOKX_SECRET_VAULT_AUTHN_APPROLE_SECRET_ID": "unknown",
					},
				})
				assert.NotNil(GinkgoT(), err)
			})

			It("should return error when kubernetes auth failed", func() {
				_, err := helper.LoadConfig(helper.LoadConfigOptions{
					Envs: map[string]string{
						"WEBHOOKX_SECRET_VAULT_AUTH_METHOD":           "kubernetes",
						"WEBHOOKX_SECRET_VAULT_AUTHN_KUBERNETES_ROLE": "test-role",
					},
				})
				assert.NotNil(GinkgoT(), err)
			})

			It("returns error when secret is not found", func() {
				_, err := helper.LoadConfig(helper.LoadConfigOptions{
					Envs: helper.NewTestEnv(map[string]string{
						"WEBHOOKX_SECRET_VAULT_AUTHN_TOKEN_TOKEN": "root",

						"WEBHOOKX_DATABASE_HOST": "{secret://vault/webhookx/notfound}",
					}),
				})
				assert.EqualError(GinkgoT(), err, "failed to resolve reference value '{secret://vault/webhookx/notfound}': secret not found")
			})

			It("should return error when reading a deleted secret", func() {
				_, err := helper.LoadConfig(helper.LoadConfigOptions{
					Envs: helper.NewTestEnv(map[string]string{
						"WEBHOOKX_SECRET_VAULT_AUTHN_TOKEN_TOKEN": "root",

						"WEBHOOKX_DATABASE_HOST": "{secret://vault/webhookx/secret-deleted.data}",
					}),
				})
				assert.EqualError(GinkgoT(), err, "failed to resolve reference value '{secret://vault/webhookx/secret-deleted.data}': secret no data")
			})
		})
	})

	Context("YAML", func() {
		It("references should be resolved", func() {
			cfg, err := helper.LoadConfig(helper.LoadConfigOptions{
				File:       test.FilePath("secret-reference/testdata/vault-secrets.yml"),
				ExcludeEnv: true,
			})
			assert.NoError(GinkgoT(), err)

			assert.Equal(GinkgoT(), "true", cfg.Database.Host)
			assert.EqualValues(GinkgoT(), 1, cfg.Database.Port)
			assert.Equal(GinkgoT(), "value", cfg.Database.Username)
			assert.EqualValues(GinkgoT(), "1", cfg.Database.Password)
			assert.EqualValues(GinkgoT(), "0.5", cfg.Database.Database)
			assert.EqualValues(GinkgoT(), "c", cfg.Database.Parameters)

			assert.Equal(GinkgoT(), "false", cfg.Redis.Host)
			assert.EqualValues(GinkgoT(), "nested value", cfg.Redis.Password)
			assert.Equal(GinkgoT(), 100, int(cfg.Redis.Port))

			assert.Equal(GinkgoT(), false, cfg.Worker.Enabled)
			assert.EqualValues(GinkgoT(), 0.5, cfg.Tracing.SamplingRate)
		})
	})
})
