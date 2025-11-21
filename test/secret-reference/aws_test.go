package secret_reference

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	awstypes "github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/test/helper"
)

func upsertSecret(client *secretsmanager.Client, name string, value string) error {
	_, err := client.CreateSecret(context.TODO(), &secretsmanager.CreateSecretInput{
		Name:         aws.String(name),
		SecretString: aws.String(value),
	})
	var exists *awstypes.ResourceExistsException
	if errors.As(err, &exists) {
		err = nil
		_, err = client.PutSecretValue(context.TODO(), &secretsmanager.PutSecretValueInput{
			SecretId:     aws.String(name),
			SecretString: aws.String(value),
		})
	}
	return err
}

var _ = Describe("AWS SecretManager", Ordered, func() {

	BeforeAll(func() {
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
		b, err := json.Marshal(data)
		assert.NoError(GinkgoT(), err)
		smClient := helper.SecretManangerClient()
		err = upsertSecret(smClient, "webhookx/config", string(b))
		assert.NoError(GinkgoT(), err)

		err = upsertSecret(smClient, "webhookx/value", "string value")
		assert.NoError(GinkgoT(), err)
	})

	Context("ENV", func() {

		It("references should be resolved", func() {
			var cfg *config.Config
			var err error
			withCleanEnv(func() {
				cancel := helper.SetEnvs(nil, map[string]string{
					"AWS_ACCESS_KEY_ID":          "test",
					"AWS_SECRET_ACCESS_KEY":      "test",
					"WEBHOOKX_SECRET_AWS_REGION": "us-east-1",
					"WEBHOOKX_SECRET_AWS_URL":    "http://localhost:4566",

					"WEBHOOKX_DATABASE_HOST":       "{secret://aws/webhookx/config.key_boolean}",
					"WEBHOOKX_DATABASE_USERNAME":   "{secret://aws/webhookx/config.key_string}",
					"WEBHOOKX_DATABASE_PASSWORD":   "{secret://aws/webhookx/config.key_integer}",
					"WEBHOOKX_DATABASE_DATABASE":   "{secret://aws/webhookx/config.key_float}",
					"WEBHOOKX_DATABASE_PARAMETERS": "{secret://aws/webhookx/config.key_array.2}",

					"WEBHOOKX_REDIS_HOST":     "{secret://aws/webhookx/config.key_nested.key_boolean}",
					"WEBHOOKX_REDIS_PASSWORD": "{secret://aws/webhookx/config.key_nested.key_string}",
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

		Context("errors", func() {
			It("returns error when extracting a value from a invalid json", func() {
				var cfg *config.Config
				var err error
				withCleanEnv(func() {
					cancel := helper.SetEnvs(nil, map[string]string{
						"AWS_ACCESS_KEY_ID":          "test",
						"AWS_SECRET_ACCESS_KEY":      "test",
						"WEBHOOKX_SECRET_AWS_REGION": "us-east-1",
						"WEBHOOKX_SECRET_AWS_URL":    "http://localhost:4566",

						"WEBHOOKX_DATABASE_HOST": "{secret://aws/webhookx/value.key}",
					})
					defer cancel()
					cfg = config.New()
					err = config.NewLoader(cfg).Load()
				})
				assert.EqualError(GinkgoT(), err, "value is not a valid json")
			})

		})
	})

	Context("YAML", func() {
		yaml1 := `
database:
  host: "{secret://aws/webhookx/config.key_boolean}"
  port: 5432
  username: "{secret://aws/webhookx/config.key_string}"
  password: "{secret://aws/webhookx/config.key_integer}"
  database: "{secret://aws/webhookx/config.key_float}"
  parameters: "{secret://aws/webhookx/config.key_array.2}"

redis:
  host: "{secret://aws/webhookx/config.key_nested.key_boolean}"
  port: 6379
  password: "{secret://aws/webhookx/config.key_nested.key_string}"
`
		It("references should be resolved", func() {
			var cfg *config.Config
			var err error

			withCleanEnv(func() {
				cancel := helper.SetEnvs(nil, map[string]string{
					"AWS_ACCESS_KEY_ID":          "test",
					"AWS_SECRET_ACCESS_KEY":      "test",
					"WEBHOOKX_SECRET_AWS_REGION": "us-east-1",
					"WEBHOOKX_SECRET_AWS_URL":    "http://localhost:4566",
				})
				defer cancel()
				cfg = config.New()
				err = config.NewLoader(cfg).WithFileContent([]byte(yaml1)).Load()
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

		Context("errors", func() {
			It("returns error when secret is not found", func() {
				configFile := `
database:
  host: "{secret://aws/webhookx/notfound}"
`
				var cfg *config.Config
				var err error

				withCleanEnv(func() {
					cancel := helper.SetEnvs(nil, map[string]string{
						"AWS_ACCESS_KEY_ID":          "test",
						"AWS_SECRET_ACCESS_KEY":      "test",
						"WEBHOOKX_SECRET_AWS_REGION": "us-east-1",
						"WEBHOOKX_SECRET_AWS_URL":    "http://localhost:4566",
					})
					defer cancel()
					cfg = config.New()
					err = config.NewLoader(cfg).WithFileContent([]byte(configFile)).Load()
				})
				assert.EqualError(GinkgoT(), err, "failed to resolve reference value '{secret://aws/webhookx/notfound}': secret not found")
			})
		})
	})
})
