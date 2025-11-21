package secret_reference

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/test"
	"github.com/webhookx-io/webhookx/test/helper"
)

func withCleanEnv(f func()) {
	oldEnv := os.Environ()

	os.Clearenv()
	f()

	os.Clearenv()
	for _, e := range oldEnv {
		parts := strings.SplitN(e, "=", 2)
		os.Setenv(parts[0], parts[1])
	}
}

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
	})

	Context("ENV", func() {
		It("references should be resolved #token", func() {
			var cfg *config.Config
			var err error
			withCleanEnv(func() {
				cancel := helper.SetEnvs(nil, map[string]string{
					"WEBHOOKX_SECRET_VAULT_AUTHN_TOKEN_TOKEN": "root",

					"WEBHOOKX_DATABASE_HOST":       "{secret://vault/webhookx/config.key_boolean}",
					"WEBHOOKX_DATABASE_USERNAME":   "{secret://vault/webhookx/config.key_string}",
					"WEBHOOKX_DATABASE_PASSWORD":   "{secret://vault/webhookx/config.key_integer}",
					"WEBHOOKX_DATABASE_DATABASE":   "{secret://vault/webhookx/config.key_float}",
					"WEBHOOKX_DATABASE_PARAMETERS": "{secret://vault/webhookx/config.key_array.2}",

					"WEBHOOKX_REDIS_HOST":     "{secret://vault/webhookx/config.key_nested.key_boolean}",
					"WEBHOOKX_REDIS_PASSWORD": "{secret://vault/webhookx/config.key_nested.key_string}",
				})
				defer cancel()
				cfg = config.New()
				err = config.NewLoader(cfg).Load()
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

		It("references should be resolved #approle", func() {
			var cfg *config.Config
			var err error
			withCleanEnv(func() {
				cancel := helper.SetEnvs(nil, map[string]string{
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
				})
				defer cancel()
				cfg = config.New()
				err = config.NewLoader(cfg).Load()
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
				w.Write([]byte(`{
  "apiVersion": "authentication.k8s.io/v1",
  "kind": "TokenReview",
  "status": {
    "authenticated": true,
    "user": {
      "username": "system:serviceaccount:default:webhookx",
      "uid": "00000000-0000-0000-0000-000000000000"
    },
	"audiences": ["webhookx"]
  }
}`))
			}, ":18888")
			var cfg *config.Config
			var err error
			withCleanEnv(func() {
				cancel := helper.SetEnvs(nil, map[string]string{
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
				})
				defer cancel()
				cfg = config.New()
				err = config.NewLoader(cfg).Load()
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
				_, err := helper.NewConfig(map[string]string{
					"WEBHOOKX_SECRET_VAULT_AUTH_METHOD":             "approle",
					"WEBHOOKX_SECRET_VAULT_AUTHN_APPROLE_ROLE_ID":   "test-role-id",
					"WEBHOOKX_SECRET_VAULT_AUTHN_APPROLE_SECRET_ID": "unknown",
				})
				assert.NotNil(GinkgoT(), err)
			})

			It("should return error when kubernetes auth failed", func() {
				_, err := helper.NewConfig(map[string]string{
					"WEBHOOKX_SECRET_VAULT_AUTH_METHOD":           "kubernetes",
					"WEBHOOKX_SECRET_VAULT_AUTHN_KUBERNETES_ROLE": "test-role",
				})
				assert.NotNil(GinkgoT(), err)
			})

			It("returns error when secret is not found", func() {
				var cfg *config.Config
				var err error
				withCleanEnv(func() {
					cancel := helper.SetEnvs(nil, map[string]string{
						"WEBHOOKX_SECRET_VAULT_AUTHN_TOKEN_TOKEN": "root",

						"WEBHOOKX_DATABASE_HOST": "{secret://vault/webhookx/notfound}",
					})
					defer cancel()
					cfg = config.New()
					err = config.NewLoader(cfg).Load()
				})
				assert.EqualError(GinkgoT(), err, "failed to resolve reference value '{secret://vault/webhookx/notfound}': secret not found")
			})

			It("should return error when reading a deleted secret", func() {
				var cfg *config.Config
				var err error
				withCleanEnv(func() {
					cancel := helper.SetEnvs(nil, map[string]string{
						"WEBHOOKX_SECRET_VAULT_AUTHN_TOKEN_TOKEN": "root",

						"WEBHOOKX_DATABASE_HOST": "{secret://vault/webhookx/secret-deleted.data}",
					})
					defer cancel()
					cfg = config.New()
					err = config.NewLoader(cfg).Load()
				})
				assert.EqualError(GinkgoT(), err, "failed to resolve reference value '{secret://vault/webhookx/secret-deleted.data}': secret no data")
			})
		})
	})

	Context("YAML", func() {
		configFile := `
database:
  host: "{secret://vault/webhookx/config.key_boolean}"
  port: 5432
  username: "{secret://vault/webhookx/config.key_string}"
  password: "{secret://vault/webhookx/config.key_integer}"
  database: "{secret://vault/webhookx/config.key_float}"
  parameters: "{secret://vault/webhookx/config.key_array.2}"

redis:
  host: "{secret://vault/webhookx/config.key_nested.key_boolean}"
  port: 6379
  password: "{secret://vault/webhookx/config.key_nested.key_string}"
`
		It("references should be resolved", func() {
			var cfg *config.Config
			var err error

			withCleanEnv(func() {
				cancel := helper.SetEnvs(nil, map[string]string{
					"WEBHOOKX_SECRET_VAULT_AUTHN_TOKEN_TOKEN": "root",
				})
				defer cancel()
				cfg = config.New()
				err = config.NewLoader(cfg).WithFileContent([]byte(configFile)).Load()
			})
			assert.NoError(GinkgoT(), err)

			assert.NoError(GinkgoT(), err)

			assert.Equal(GinkgoT(), "true", cfg.Database.Host)
			assert.Equal(GinkgoT(), "value", cfg.Database.Username)
			assert.EqualValues(GinkgoT(), "1", cfg.Database.Password)
			assert.EqualValues(GinkgoT(), "0.5", cfg.Database.Database)
			assert.EqualValues(GinkgoT(), "c", cfg.Database.Parameters)

			assert.Equal(GinkgoT(), "false", cfg.Redis.Host)
			assert.EqualValues(GinkgoT(), "nested value", cfg.Redis.Password)
		})
	})
})
