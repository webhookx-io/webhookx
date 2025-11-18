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
	})

	Context("ENV", func() {
		It("references should be resolved #token", func() {
			config, err := helper.NewConfig(map[string]string{
				"WEBHOOKX_SECRET_VAULT_AUTHN_TOKEN_TOKEN": "root",

				"WEBHOOKX_DATABASE_HOST":       "{secret://vault/webhookx/config.key_boolean}",
				"WEBHOOKX_DATABASE_USERNAME":   "{secret://vault/webhookx/config.key_string}",
				"WEBHOOKX_DATABASE_PASSWORD":   "{secret://vault/webhookx/config.key_integer}",
				"WEBHOOKX_DATABASE_DATABASE":   "{secret://vault/webhookx/config.key_float}",
				"WEBHOOKX_DATABASE_PARAMETERS": "{secret://vault/webhookx/config.key_array.2}",

				"WEBHOOKX_REDIS_HOST":     "{secret://vault/webhookx/config.key_nested.key_boolean}",
				"WEBHOOKX_REDIS_PASSWORD": "{secret://vault/webhookx/config.key_nested.key_string}",
			})
			assert.NoError(GinkgoT(), err)

			assert.Equal(GinkgoT(), "true", config.Database.Host)
			assert.Equal(GinkgoT(), "value", config.Database.Username)
			assert.EqualValues(GinkgoT(), "1", config.Database.Password)
			assert.EqualValues(GinkgoT(), "0.5", config.Database.Database)
			assert.EqualValues(GinkgoT(), "c", config.Database.Parameters)

			assert.Equal(GinkgoT(), "false", config.Redis.Host)
			assert.EqualValues(GinkgoT(), "nested value", config.Redis.Password)
		})

		It("references should be resolved #approle", func() {
			config, err := helper.NewConfig(map[string]string{
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
			assert.NoError(GinkgoT(), err)

			assert.Equal(GinkgoT(), "true", config.Database.Host)
			assert.Equal(GinkgoT(), "value", config.Database.Username)
			assert.EqualValues(GinkgoT(), "1", config.Database.Password)
			assert.EqualValues(GinkgoT(), "0.5", config.Database.Database)
			assert.EqualValues(GinkgoT(), "c", config.Database.Parameters)

			assert.Equal(GinkgoT(), "false", config.Redis.Host)
			assert.EqualValues(GinkgoT(), "nested value", config.Redis.Password)
		})

		It("references should be resolved #kubernetes", func() {
			server := startHTTP(func(w http.ResponseWriter, request *http.Request) {
				fmt.Println(request.URL)
				fmt.Println("doge")
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
			config, err := helper.NewConfig(map[string]string{
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
			assert.NoError(GinkgoT(), err)

			assert.Equal(GinkgoT(), "true", config.Database.Host)
			assert.Equal(GinkgoT(), "value", config.Database.Username)
			assert.EqualValues(GinkgoT(), "1", config.Database.Password)
			assert.EqualValues(GinkgoT(), "0.5", config.Database.Database)
			assert.EqualValues(GinkgoT(), "c", config.Database.Parameters)

			assert.Equal(GinkgoT(), "false", config.Redis.Host)
			assert.EqualValues(GinkgoT(), "nested value", config.Redis.Password)
			assert.Nil(GinkgoT(), server.Shutdown(context.TODO()))
		})
	})

	Context("YAML", func() {
		yaml1 := `
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
				cfg, err = config.New(&config.Options{
					YAML: []byte(yaml1),
				})
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
	})
})
