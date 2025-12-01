package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	jsonschemaV6 "github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/constants"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/plugins/jsonschema_validator"
	"github.com/webhookx-io/webhookx/plugins/jsonschema_validator/jsonschema"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/test/helper/factory"
	"github.com/webhookx-io/webhookx/utils"
	"strings"
	"time"
)

var jsonString = `{
	"type": "object",
	"required": ["id", "amount", "currency"],
	"properties": {
		"id": {
			"type": "string"
		},
		"amount": {
			"type": "integer",
			"minimum": 1
		},
		"currency": {
			"type": "string",
			"maxLength": 6,
			"minLength": 3
		}
	}
}`

type validateFieldTestContext struct {
	invalidData any
	expectedMsg string
	schemaDef   string
}

func getFullVersionFields(baseFields, versionFileds map[string]validateFieldTestContext) (map[string]validateFieldTestContext, map[string]any, map[string]string) {
	fields := versionFileds
	invalidData := make(map[string]any)
	expectedErrors := make(map[string]string)

	// Add base fields (all versions)
	for k, v := range baseFields {
		fields[k] = v
		if v.invalidData != nil {
			invalidData[k] = v.invalidData
		}
		expectedErrors[k] = v.expectedMsg
	}
	return fields, invalidData, expectedErrors
}

func buildSchema(version string, fields map[string]validateFieldTestContext) string {
	var props []string
	for _, field := range fields {
		props = append(props, field.schemaDef)
	}

	if version == jsonschema.OpenAPI_3_0 {
		return fmt.Sprintf(`{
						"type": "object",
						"required": ["requiredField"],
						"properties": {
							%s
						}
					}`, strings.Join(props, ","))
	} else {
		return fmt.Sprintf(`{
					"$schema": "https://json-schema.org/%s/schema",
					"type": "object",
					"required": ["requiredField"],
					"properties": {
						%s
					}
				}`, version, strings.Join(props, ","))
	}
}

var _ = Describe("jsonschema-validator", Ordered, func() {

	Context("schema string", func() {
		var proxyClient *resty.Client

		var app *app.Application
		var db *db.DB

		entitiesConfig := helper.EntitiesConfig{
			Endpoints: []*entities.Endpoint{factory.EndpointP()},
			Sources:   []*entities.Source{factory.SourceP()},
		}
		entitiesConfig.Plugins = []*entities.Plugin{
			factory.PluginP(
				factory.WithPluginSourceID(entitiesConfig.Sources[0].ID),
				factory.WithPluginName("jsonschema-validator"),
				factory.WithPluginConfig(jsonschema_validator.Config{
					DefaultSchema:   jsonString,
					VerboseResponse: true,
					Schemas: map[string]*jsonschema_validator.Schema{
						"charge.succeeded": {
							Schema: jsonString,
						},
						"reuse.default_schema": nil,
					},
				}),
			),
		}

		BeforeAll(func() {
			db = helper.InitDB(true, &entitiesConfig)
			proxyClient = helper.ProxyClient()

			app = utils.Must(helper.Start(map[string]string{}))

			err := helper.WaitForServer(helper.ProxyHttpURL, time.Second)
			assert.NoError(GinkgoT(), err)
		})

		AfterAll(func() {
			app.Stop()
		})

		It("sanity", func() {
			body := `{"event_type": "charge.succeeded","data": {"id": "ch_1234567890","amount": 1000,"currency": "usd"}}`
			resp, err := proxyClient.R().
				SetHeader("Content-Type", "application/json").
				SetBody(body).
				Post("/")
			Expect(err).To(BeNil())
			Expect(resp.StatusCode()).To(Equal(200))
			eventId := resp.Header().Get(constants.HeaderEventId)
			assert.NotEmpty(GinkgoT(), eventId)
			// get event from db
			event, err := db.Events.Get(context.TODO(), eventId)
			assert.Nil(GinkgoT(), err)
			assert.NotNil(GinkgoT(), event)
			assert.Equal(GinkgoT(), "charge.succeeded", event.EventType)
			assert.JSONEq(GinkgoT(), `{"id": "ch_1234567890","amount": 1000,"currency": "usd"}`, string(event.Data))
		})

		It("sanity if undeclared event type", func() {
			body := `{"event_type": "unknown.event", "data":{"foo": "bar"}}`
			resp, err := proxyClient.R().
				SetHeader("Content-Type", "application/json").
				SetBody(body).
				Post("/")
			Expect(err).To(BeNil())
			Expect(resp.StatusCode()).To(Equal(200))

			eventId := resp.Header().Get(constants.HeaderEventId)
			assert.NotEmpty(GinkgoT(), eventId)
			// get event from db
			event, err := db.Events.Get(context.TODO(), eventId)
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), "unknown.event", event.EventType)
			assert.JSONEq(GinkgoT(), `{"foo": "bar"}`, string(event.Data))
		})

		It("sanity if reuse default_schema", func() {
			body := `{"event_type": "reuse.default_schema","data": {"id": "ch_1234567890","amount": 1000,"currency": "usd"}}`
			resp, err := proxyClient.R().
				SetHeader("Content-Type", "application/json").
				SetBody(body).
				Post("/")
			Expect(err).To(BeNil())
			Expect(resp.StatusCode()).To(Equal(200))
			eventId := resp.Header().Get(constants.HeaderEventId)
			assert.NotEmpty(GinkgoT(), eventId)
			// get event from db
			event, err := db.Events.Get(context.TODO(), eventId)
			assert.Nil(GinkgoT(), err)
			assert.NotNil(GinkgoT(), event)
			assert.Equal(GinkgoT(), "reuse.default_schema", event.EventType)
			assert.JSONEq(GinkgoT(), `{"id": "ch_1234567890","amount": 1000,"currency": "usd"}`, string(event.Data))
		})

		It("invalid event - missing required field", func() {
			body := `{"event_type": "charge.succeeded","data": {"amount": 1000,"currency": "usd"}}`
			resp, err := proxyClient.R().
				SetHeader("Content-Type", "application/json").
				SetBody(body).
				Post("/")
			Expect(err).To(BeNil())
			Expect(resp.StatusCode()).To(Equal(400))
			Expect(string(resp.Body())).To(Equal(`{"message":"Request Validation","error":{"message":"request validation","fields":{"id":"required field missing"}}}`))
		})

		It("invalid event - field type mismatch", func() {
			body := `{"event_type": "charge.succeeded","data": {"id": "ch_1234567890","amount": "1000","currency": "usd"}}`
			resp, err := proxyClient.R().
				SetHeader("Content-Type", "application/json").
				SetBody(body).
				Post("/")

			Expect(err).To(BeNil())
			Expect(resp.StatusCode()).To(Equal(400))
			Expect(string(resp.Body())).To(Equal(`{"message":"Request Validation","error":{"message":"request validation","fields":{"amount":"value must be an integer"}}}`))
		})

		It("invalid event - reuse default schema", func() {
			body := `{"event_type": "reuse.default_schema","data": {"id": "ch_1234567890","amount": "1000","currency": "usd"}}`
			resp, err := proxyClient.R().
				SetHeader("Content-Type", "application/json").
				SetBody(body).
				Post("/")

			Expect(err).To(BeNil())
			Expect(resp.StatusCode()).To(Equal(400))
			Expect(string(resp.Body())).To(Equal(`{"message":"Request Validation","error":{"message":"request validation","fields":{"amount":"value must be an integer"}}}`))
		})
	})

	Context("verbose response", func() {
		var proxyClient *resty.Client

		var app *app.Application

		entitiesConfig := helper.EntitiesConfig{
			Endpoints: []*entities.Endpoint{factory.EndpointP()},
			Sources:   []*entities.Source{factory.SourceP()},
		}
		entitiesConfig.Plugins = []*entities.Plugin{
			factory.PluginP(
				factory.WithPluginSourceID(entitiesConfig.Sources[0].ID),
				factory.WithPluginName("jsonschema-validator"),
				factory.WithPluginConfig(jsonschema_validator.Config{
					DefaultSchema:   jsonString,
					VerboseResponse: false,
					Schemas: map[string]*jsonschema_validator.Schema{
						"charge.succeeded": {
							Schema: jsonString,
						},
						"reuse.default_schema": nil,
					},
				}),
			),
		}

		BeforeAll(func() {
			_ = helper.InitDB(true, &entitiesConfig)
			proxyClient = helper.ProxyClient()

			app = utils.Must(helper.Start(map[string]string{}))

			err := helper.WaitForServer(helper.ProxyHttpURL, time.Second)
			assert.NoError(GinkgoT(), err)
		})

		AfterAll(func() {
			app.Stop()
		})

		It("invalid event - inverbose response", func() {
			body := `{"event_type": "charge.succeeded","data": {"amount": 1000,"currency": "usd"}}`
			resp, err := proxyClient.R().
				SetHeader("Content-Type", "application/json").
				SetBody(body).
				Post("/")
			Expect(err).To(BeNil())
			Expect(resp.StatusCode()).To(Equal(400))
			Expect(string(resp.Body())).To(Equal(fmt.Sprintf(`{"message":"%s"}`, jsonschema_validator.InverboseResponseMessage)))
		})
	})

	versions := []string{
		jsonschema.Draft_4,
		jsonschema.Draft_6,
		jsonschema.Draft_7,
		jsonschema.Draft_2019,
		jsonschema.Draft_2020,
		jsonschema.OpenAPI_3_0,
	}
	for _, version := range versions {
		schema := fmt.Sprintf(`"$schema": "https://json-schema.org/%s/schema"`, version)

		Context(string(version)+" validation", func() {
			var c *jsonschemaV6.Compiler
			BeforeAll(func() {
				c = jsonschemaV6.NewCompiler()
				c.AssertContent()
				c.AssertFormat()
			})

			It("sanity", func() {
				schemaDef := `{
				"type": "object",
					"properties": {
						"user": {
							"type": "object",
							"properties": {
								"name": { "type": "string" },
								"email": { "type": "string", "format": "email" }
							},
							"required": ["name", "email"]
						},
						"age": { "type": "integer", "minimum": 0 },
						"gender": { "type": "string", "enum": ["male","female"]}
					},
					"required": ["user", "age"]
			}`

				if version != jsonschema.OpenAPI_3_0 {
					schemaDef = fmt.Sprintf(`{%s,%s`, schema, schemaDef[1:])
				}
				validator := jsonschema.New(schemaDef, c)
				validData := map[string]any{
					"user": map[string]any{
						"name":  "tester",
						"email": "test@test.com",
					},
					"age":    20,
					"gender": "male",
				}
				ctx := &jsonschema.ValidatorContext{
					HTTPRequest: &jsonschema.HTTPRequest{
						Data: validData,
					},
				}
				err := validator.Validate(ctx)
				Expect(err).To(BeNil())
			})

			It(string(version)+" - invalid data validation", func() {
				schemaDef := `{
					"type": "object",
						"properties": {
							"user": {
								"type": "object",
								"properties": {
									"name": { "type": "string" },
									"email": { "type": "string", "format": "email" }
								},
								"required": ["name", "email"]
							},
							"age": { "type": "integer", "minimum": 0 },
							"gender": { "type": "string", "enum": ["male","female"]}
						},
						"required": ["user", "age"]
				}`
				if version != jsonschema.OpenAPI_3_0 {
					schemaDef = fmt.Sprintf(`{%s,%s`, schema, schemaDef[1:])
				}

				validator := jsonschema.New(schemaDef, c)

				invalidData := map[string]any{
					"user": map[string]any{
						"name": 1234,
					},
					"age":    -5,
					"gender": "xxx",
				}
				ctx := &jsonschema.ValidatorContext{
					HTTPRequest: &jsonschema.HTTPRequest{
						Data: invalidData,
					},
				}

				err := validator.Validate(ctx)
				Expect(err).ToNot(BeNil())
				b, _ := json.Marshal(err)
				Expect(string(b)).To(Equal(`{"message":"request validation","fields":{"age":"number must be at least 0","gender":"value is not one of the allowed values [\"male\",\"female\"]","user":{"email":"required field missing","name":"value must be a string"}}}`))
			})
		})
	}

	Context("Validator version support", func() {
		var c *jsonschemaV6.Compiler
		baseFields := map[string]validateFieldTestContext{
			"requiredField": {
				invalidData: nil, // missing field
				expectedMsg: "required field missing",
				schemaDef:   `"requiredField": { "type": "string" }`,
			},
			"stringField": {
				invalidData: 123,
				expectedMsg: "value must be a string",
				schemaDef:   `"stringField": { "type": "string" }`,
			},

			"integerField": {
				invalidData: "not-a-number",
				expectedMsg: "value must be an integer",
				schemaDef:   `"integerField": { "type": "integer" }`,
			},
			"arrayField": {
				invalidData: "not-an-array",
				expectedMsg: "value must be an array",
				schemaDef:   `"arrayField": { "type": "array" }`,
			},
			"enumField": {
				invalidData: "invalid",
				expectedMsg: `value is not one of the allowed values ["valid1","valid2"]`,
				schemaDef:   `"enumField": { "type": "string", "enum": ["valid1", "valid2"] }`,
			},
			"minimumField": {
				invalidData: -5,
				expectedMsg: "number must be at least 0",
				schemaDef:   `"minimumField": { "type": "integer", "minimum": 0 }`,
			},
			"maximumField": {
				invalidData: 150,
				expectedMsg: "number must be at most 100",
				schemaDef:   `"maximumField": { "type": "integer", "maximum": 100 }`,
			},
			"multipleOfField": {
				invalidData: 7,
				expectedMsg: "number must be a multiple of 5",
				schemaDef:   `"multipleOfField": { "type": "integer", "multipleOf": 5 }`,
			},
			"minLengthField": {
				invalidData: "",
				expectedMsg: "minimum string length is 3",
				schemaDef:   `"minLengthField": { "type": "string", "minLength": 3 }`,
			},
			"maxLengthField": {
				invalidData: "too-long",
				expectedMsg: "maximum string length is 5",
				schemaDef:   `"maxLengthField": { "type": "string", "maxLength": 5 }`,
			},
			"patternField": {
				invalidData: "invalid123",
				expectedMsg: `string doesn't match the regular expression "^[a-z]+$"`,
				schemaDef:   `"patternField": { "type": "string", "pattern": "^[a-z]+$" }`,
			},
			"minItemsField": {
				invalidData: []any{1},
				expectedMsg: "minimum number of items is 2",
				schemaDef:   `"minItemsField": { "type": "array", "minItems": 2 }`,
			},
			"maxItemsField": {
				invalidData: []any{1, 2, 3},
				expectedMsg: "maximum number of items is 2",
				schemaDef:   `"maxItemsField": { "type": "array", "maxItems": 2 }`,
			},
			"uniqueItemsField": {
				invalidData: []any{1, 2, 2},
				expectedMsg: "duplicate items found",
				schemaDef:   `"uniqueItemsField": { "type": "array", "uniqueItems": true }`,
			},
			"minPropertiesField": {
				invalidData: map[string]any{"a": 1},
				expectedMsg: "there must be at least 2 properties",
				schemaDef:   `"minPropertiesField": { "type": "object", "minProperties": 2 }`,
			},
			"maxPropertiesField": {
				invalidData: map[string]any{"a": 1, "b": 2, "c": 3, "d": 4},
				expectedMsg: "there must be at most 3 properties",
				schemaDef:   `"maxPropertiesField": { "type": "object", "maxProperties": 3 }`,
			},
			"additionalPropsField": {
				invalidData: map[string]any{"allowed": "value", "extra": "not-allowed"},
				expectedMsg: `property "extra" is unsupported`,
				schemaDef:   `"additionalPropsField": { "type": "object", "properties": { "allowed": { "type": "string" } }, "additionalProperties": false }`,
			},
		}

		jsonschemaComposite := map[string]validateFieldTestContext{
			"formatField": {
				invalidData: "not-an-email",
				expectedMsg: `string doesn't match the format "email"`,
				schemaDef:   `"formatField": { "type": "string", "format": "email" }`,
			},
			"oneOfField": {
				invalidData: map[string]any{"type": "object"},
				expectedMsg: `value doesn't match any schema from "oneOf"`,
				schemaDef:   `"oneOfField": { "oneOf": [ { "type": "integer" }, { "type": "boolean" } ] }`,
			},
			"anyOfField": {
				invalidData: map[string]any{"type": "object"},
				expectedMsg: `value doesn't match any schema from "anyOf"`,
				schemaDef:   `"anyOfField": { "anyOf": [ { "type": "integer" }, { "type": "boolean" } ] }`,
			},
			"allOfField": {
				invalidData: "short",
				expectedMsg: `value doesn't match all schemas from "allOf"`,
				schemaDef:   `"allOfField": { "allOf": [ { "type": "string" }, { "minLength": 10 } ] }`,
			},
		}

		draft4SpecificFields := map[string]validateFieldTestContext{
			"exclusiveMinField": {
				invalidData: 0,
				expectedMsg: "number must be more than 0",
				schemaDef:   fmt.Sprintf(`"exclusiveMinField": { "type": "integer", %s }`, `"minimum": 0, "exclusiveMinimum": true`),
			},
			"exclusiveMaxField": {
				invalidData: 100,
				expectedMsg: "number must be less than 100",
				schemaDef:   fmt.Sprintf(`"exclusiveMaxField": { "type": "integer", %s }`, `"maximum": 100, "exclusiveMaximum": true`),
			},
		}

		// Draft 6+ fields
		draft6PlusFields := map[string]validateFieldTestContext{
			"constField": {
				invalidData: "wrong-value",
				expectedMsg: `value must be "expected-value"`,
				schemaDef:   `"constField": { "const": "expected-value" }`,
			},
			"containsField": {
				invalidData: []any{1, 2, 3},
				expectedMsg: "no items match contains schema",
				schemaDef:   `"containsField": { "type": "array", "contains": { "type": "integer", "minimum": 10 } }`,
			},
			"exclusiveMinField": {
				invalidData: 0,
				expectedMsg: "number must be more than 0",
				schemaDef:   fmt.Sprintf(`"exclusiveMinField": { "type": "integer", %s }`, `"exclusiveMinimum": 0`),
			},
			"exclusiveMaxField": {
				invalidData: 100,
				expectedMsg: "number must be less than 100",
				schemaDef:   fmt.Sprintf(`"exclusiveMaxField": { "type": "integer", %s }`, `"exclusiveMaximum": 100`),
			},
			"propertyNamesField": {
				invalidData: map[string]any{"validName": 1, "invalid-name!": 2},
				expectedMsg: `invalid propertyName "invalid-name!"`,
				schemaDef:   `"propertyNamesField": { "type": "object", "propertyNames": { "type": "string", "pattern": "^[a-zA-Z_][a-zA-Z0-9_]*$" } }`,
			},
		}

		// Draft 2019-09+ fields
		draft2019PlusFields := map[string]validateFieldTestContext{
			"minContainsField": {
				invalidData: []any{1, 2, 15},
				expectedMsg: "min 2 items required to match contains schema, but matched 1 items",
				schemaDef:   `"minContainsField": { "type": "array", "contains": { "type": "integer", "minimum": 10 }, "minContains": 2 }`,
			},
			"maxContainsField": {
				invalidData: []any{1, 15, 20, 25},
				expectedMsg: "max 2 items required to match contains schema, but matched 3 items",
				schemaDef:   `"maxContainsField": { "type": "array", "contains": { "type": "integer", "minimum": 10 }, "maxContains": 2 }`,
			},
			"dependentRequiredField": {
				invalidData: map[string]any{"a": 1},
				expectedMsg: "properties b required, if a exists",
				schemaDef:   `"dependentRequiredField": { "type": "object", "properties": { "a": { "type": "integer" }, "b": { "type": "integer" } }, "dependentRequired": { "a": ["b"] } }`,
			},
		}

		BeforeAll(func() {
			c = jsonschemaV6.NewCompiler()
			c.AssertContent()
			c.AssertFormat()
		})

		It(jsonschema.OpenAPI_3_0, func() {
			versionFields, invalidData, expectedErrors := getFullVersionFields(baseFields, draft4SpecificFields)
			schemaDef := buildSchema(jsonschema.OpenAPI_3_0, versionFields)
			validator := jsonschema.New(schemaDef, c)
			ctx := &jsonschema.ValidatorContext{
				HTTPRequest: &jsonschema.HTTPRequest{
					Data: invalidData,
				},
			}

			err := validator.Validate(ctx)
			Expect(err).ToNot(BeNil(), "Expect not nil error")

			var errMap map[string]interface{}
			b, _ := json.Marshal(err)
			json.Unmarshal(b, &errMap)

			actualFields, ok := errMap["fields"].(map[string]interface{})
			Expect(ok).To(BeTrue(), "Error should have 'fields' map for %s", string(b))

			// Verify each expected field error message
			for field, expectedMsg := range expectedErrors {
				actualMsg, exists := actualFields[field]

				Expect(exists).To(BeTrue(), "Field '%s' should have error for %s", field, actualMsg)
				Expect(expectedMsg).To(Equal(actualMsg), "Field '%s' error message should be consistent for [%s]. Got: [%v]", field, expectedMsg, actualMsg)
			}
		})

		It(jsonschema.Draft_4, func() {
			var draft4Fields map[string]validateFieldTestContext
			draft4Fields = utils.MergeGenericMap(draft4Fields, draft4SpecificFields)
			draft4Fields = utils.MergeGenericMap(draft4Fields, jsonschemaComposite)
			versionFields, invalidData, expectedErrors := getFullVersionFields(baseFields, draft4Fields)
			schemaDef := buildSchema(jsonschema.Draft_4, versionFields)
			validator := jsonschema.New(schemaDef, c)
			ctx := &jsonschema.ValidatorContext{
				HTTPRequest: &jsonschema.HTTPRequest{
					Data: invalidData,
				},
			}

			err := validator.Validate(ctx)
			Expect(err).ToNot(BeNil(), "Expect not nil error")

			var errMap map[string]interface{}
			b, _ := json.Marshal(err)
			json.Unmarshal(b, &errMap)

			actualFields, ok := errMap["fields"].(map[string]interface{})
			Expect(ok).To(BeTrue(), "Error should have 'fields' map for %s", string(b))

			// Verify each expected field error message
			for field, expectedMsg := range expectedErrors {
				actualMsg, exists := actualFields[field]

				Expect(exists).To(BeTrue(), "Field '%s' should have error for %s", field, actualMsg)
				Expect(expectedMsg).To(Equal(actualMsg), "Field '%s' error message should be consistent for [%s]. Got: [%v]", field, expectedMsg, actualMsg)
			}
		})

		// draft 6 and 7 are similar
		It(jsonschema.Draft_6, func() {
			draft6Fields := draft6PlusFields
			utils.MergeGenericMap(draft6Fields, jsonschemaComposite)
			versionFields, invalidData, expectedErrors := getFullVersionFields(baseFields, draft6Fields)
			schemaDef := buildSchema(jsonschema.Draft_6, versionFields)
			validator := jsonschema.New(schemaDef, c)
			ctx := &jsonschema.ValidatorContext{
				HTTPRequest: &jsonschema.HTTPRequest{
					Data: invalidData,
				},
			}

			err := validator.Validate(ctx)
			Expect(err).ToNot(BeNil(), "Expect not nil error")

			var errMap map[string]interface{}
			b, _ := json.Marshal(err)
			json.Unmarshal(b, &errMap)

			actualFields, ok := errMap["fields"].(map[string]interface{})
			Expect(ok).To(BeTrue(), "Error should have 'fields' map for %s", string(b))

			// Verify each expected field error message
			for field, expectedMsg := range expectedErrors {
				actualMsg, exists := actualFields[field]

				Expect(exists).To(BeTrue(), "Field '%s' should have error for %s", field, actualMsg)
				Expect(expectedMsg).To(Equal(actualMsg), "Field '%s' error message should be consistent for [%s]. Got: [%v]", field, expectedMsg, actualMsg)
			}
		})

		// draft 2019 and 2020 are similar
		It(jsonschema.Draft_2019, func() {
			draft2019Fields := draft2019PlusFields
			utils.MergeGenericMap(draft2019Fields, jsonschemaComposite)
			versionFields, invalidData, expectedErrors := getFullVersionFields(baseFields, draft2019Fields)
			schemaDef := buildSchema(jsonschema.Draft_6, versionFields)
			validator := jsonschema.New(schemaDef, c)
			ctx := &jsonschema.ValidatorContext{
				HTTPRequest: &jsonschema.HTTPRequest{
					Data: invalidData,
				},
			}

			err := validator.Validate(ctx)
			Expect(err).ToNot(BeNil(), "Expect not nil error")

			var errMap map[string]interface{}
			b, _ := json.Marshal(err)
			json.Unmarshal(b, &errMap)

			actualFields, ok := errMap["fields"].(map[string]interface{})
			Expect(ok).To(BeTrue(), "Error should have 'fields' map for %s", string(b))

			// Verify each expected field error message
			for field, expectedMsg := range expectedErrors {
				actualMsg, exists := actualFields[field]

				Expect(exists).To(BeTrue(), "Field '%s' should have error for %s", field, actualMsg)
				Expect(expectedMsg).To(Equal(actualMsg), "Field '%s' error message should be consistent for [%s]. Got: [%v]", field, expectedMsg, actualMsg)
			}
		})
	})

})
