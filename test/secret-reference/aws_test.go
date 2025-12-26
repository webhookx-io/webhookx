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
	"github.com/webhookx-io/webhookx/test"
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

	var licenserCancel func()
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
		licenserCancel = helper.ReplaceLicenser(nil)
	})

	AfterAll(func() {
		licenserCancel()
	})

	Context("ENV", func() {

		It("references should be resolved", func() {
			cfg, err := helper.LoadConfig(helper.LoadConfigOptions{
				Envs: helper.NewTestEnv(map[string]string{
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

		Context("errors", func() {
			It("returns error when extracting a value from a invalid json", func() {
				_, err := helper.LoadConfig(helper.LoadConfigOptions{
					Envs: helper.NewTestEnv(map[string]string{
						"AWS_ACCESS_KEY_ID":          "test",
						"AWS_SECRET_ACCESS_KEY":      "test",
						"WEBHOOKX_SECRET_AWS_REGION": "us-east-1",
						"WEBHOOKX_SECRET_AWS_URL":    "http://localhost:4566",

						"WEBHOOKX_DATABASE_HOST": "{secret://aws/webhookx/value.key}",
					}),
				})
				assert.EqualError(GinkgoT(), err, "failed to resolve reference value '{secret://aws/webhookx/value.key}': value is not a valid JSON string")
			})

			It("returns error when json path has no value", func() {
				_, err := helper.LoadConfig(helper.LoadConfigOptions{
					Envs: helper.NewTestEnv(map[string]string{
						"AWS_ACCESS_KEY_ID":          "test",
						"AWS_SECRET_ACCESS_KEY":      "test",
						"WEBHOOKX_SECRET_AWS_REGION": "us-east-1",
						"WEBHOOKX_SECRET_AWS_URL":    "http://localhost:4566",

						"WEBHOOKX_DATABASE_HOST": "{secret://aws/webhookx/config.key_nested.no-value}",
					}),
				})
				assert.EqualError(GinkgoT(), err, "failed to resolve reference value '{secret://aws/webhookx/config.key_nested.no-value}': no value for json path 'key_nested.no-value'")
			})

		})
	})

	Context("YAML", func() {
		It("references should be resolved", func() {
			reset := helper.SetEnvs(map[string]string{
				"AWS_ACCESS_KEY_ID":     "test",
				"AWS_SECRET_ACCESS_KEY": "test",
			})
			defer reset()
			cfg, err := helper.LoadConfig(helper.LoadConfigOptions{
				File:       test.FilePath("secret-reference/testdata/aws-secrets.yml"),
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

		Context("errors", func() {
			It("returns error when secret is not found", func() {
				reset := helper.SetEnvs(map[string]string{
					"AWS_ACCESS_KEY_ID":     "test",
					"AWS_SECRET_ACCESS_KEY": "test",
				})
				defer reset()
				_, err := helper.LoadConfig(helper.LoadConfigOptions{
					File:       test.FilePath("secret-reference/testdata/aws-secrets-not-found.yml"),
					ExcludeEnv: true,
				})
				assert.EqualError(GinkgoT(), err, "failed to resolve reference value '{secret://aws/webhookx/notfound}': secret not found")
			})
		})
	})
})
