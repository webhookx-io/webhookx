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
	"github.com/webhookx-io/webhookx/pkg/errs"
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

	for k, v := range fields {
		invalidData[k] = v.invalidData
		expectedErrors[k] = v.expectedMsg
	}

	// Add base fields (all versions)
	for k, v := range baseFields {
		fields[k] = v
		invalidData[k] = v.invalidData
		expectedErrors[k] = v.expectedMsg
	}
	return fields, invalidData, expectedErrors
}

func buildSchema(version string, fields map[string]validateFieldTestContext) map[string]string {
	var results = make(map[string]string)
	for key, field := range fields {
		def := field.schemaDef
		if key == "requiredField" {
			def = fmt.Sprintf(`"required": ["requiredField"], "properties":{"%s":{%s}}`, key, def)
		}
		results[key] = fmt.Sprintf(`{
					"$schema": "https://json-schema.org/%s/schema",
					%s
					}`, version, def)
	}
	return results
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

	Context("validation", func() {
		var c *jsonschemaV6.Compiler
		versions := []string{
			jsonschema.Draft_4,
			jsonschema.Draft_6,
			jsonschema.Draft_7,
			jsonschema.Draft_2019,
			jsonschema.Draft_2020,
			jsonschema.OpenAPI_3_0,
		}
		BeforeAll(func() {
			c = jsonschemaV6.NewCompiler()
			c.AssertContent()
			c.AssertFormat()
		})
		for _, version := range versions {
			schema := fmt.Sprintf(`"$schema": "https://json-schema.org/%s/schema"`, version)
			It(version+" sanity", func() {
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
		}
	})

	Context("format error consistent for version", func() {
		var c *jsonschemaV6.Compiler
		baseFields := map[string]validateFieldTestContext{
			"requiredField": {
				invalidData: map[string]interface{}{},
				expectedMsg: "required field missing",
				schemaDef:   `"type": "object"`,
			},
			"stringField": {
				invalidData: 123,
				expectedMsg: "value must be a string",
				schemaDef:   `"type": "string"`,
			},

			"integerField": {
				invalidData: "not-a-number",
				expectedMsg: "value must be an integer",
				schemaDef:   `"type": "integer"`,
			},
			"arrayField": {
				invalidData: "not-an-array",
				expectedMsg: "value must be an array",
				schemaDef:   `"type": "array"`,
			},
			"enumField": {
				invalidData: "invalid",
				expectedMsg: `value is not one of the allowed values ["valid1","valid2"]`,
				schemaDef:   `"type": "string", "enum": ["valid1", "valid2"]`,
			},
			"minimumField": {
				invalidData: -5,
				expectedMsg: "number must be at least 0",
				schemaDef:   `"type": "integer", "minimum": 0`,
			},
			"maximumField": {
				invalidData: 150,
				expectedMsg: "number must be at most 100",
				schemaDef:   `"type": "integer", "maximum": 100`,
			},
			"multipleOfField": {
				invalidData: 7,
				expectedMsg: "number must be a multiple of 5",
				schemaDef:   `"type": "integer", "multipleOf": 5`,
			},
			"minLengthField": {
				invalidData: "",
				expectedMsg: "minimum string length is 3",
				schemaDef:   `"type": "string", "minLength": 3`,
			},
			"maxLengthField": {
				invalidData: "too-long",
				expectedMsg: "maximum string length is 5",
				schemaDef:   `"type": "string", "maxLength": 5`,
			},
			"patternField": {
				invalidData: "invalid123",
				expectedMsg: `string doesn't match the regular expression "^[a-z]+$"`,
				schemaDef:   `"type": "string", "pattern": "^[a-z]+$"`,
			},
			"minItemsField": {
				invalidData: []any{1},
				expectedMsg: "minimum number of items is 2",
				schemaDef:   `"type": "array", "minItems": 2`,
			},
			"maxItemsField": {
				invalidData: []any{1, 2, 3},
				expectedMsg: "maximum number of items is 2",
				schemaDef:   `"type": "array", "maxItems": 2`,
			},
			"uniqueItemsField": {
				invalidData: []any{1, 2, 2},
				expectedMsg: "duplicate items found",
				schemaDef:   `"type": "array", "uniqueItems": true`,
			},
			"minPropertiesField": {
				invalidData: map[string]any{"a": 1},
				expectedMsg: "there must be at least 2 properties",
				schemaDef:   `"type": "object", "minProperties": 2`,
			},
			"maxPropertiesField": {
				invalidData: map[string]any{"a": 1, "b": 2, "c": 3, "d": 4},
				expectedMsg: "there must be at most 3 properties",
				schemaDef:   `"type": "object", "maxProperties": 3`,
			},
			"additionalPropsField": {
				invalidData: map[string]any{"allowed": "value", "extra": "not-allowed"},
				expectedMsg: `property "extra" is unsupported`,
				schemaDef:   `"type": "object", "properties": { "allowed": { "type": "string" } }, "additionalProperties": false `,
			},
		}

		jsonschemaExtraFields := map[string]validateFieldTestContext{
			"formatField": {
				invalidData: "te..st@example.com",
				expectedMsg: `string doesn't match the format "email"`,
				schemaDef:   `"type": "string", "format": "email"`,
			},
			"oneOf": {
				invalidData: map[string]any{"type": "object"},
				expectedMsg: `value doesn't match any schema from "oneOf"`,
				schemaDef: `"definitions": {
						"test1": { "type":"string", "minLength": 10000 },
						"test2": { "type":"string", "minLength": 100000}
					}, "oneOf": [ {"$ref": "#/definitions/test1"},{"$ref": "#/definitions/test2"}]`,
			},
			"anyOf": {
				invalidData: false,
				expectedMsg: `value doesn't match any schema from "anyOf"`,
				schemaDef: `"definitions": {
						"test1": { "type":"string", "minLength": 10000 },
						"test2": { "type":"string", "minLength": 100000}
					}, "anyOf": [{"$ref": "#/definitions/test1"},{"$ref": "#/definitions/test2"}]`,
			},
			"allOf": {
				invalidData: "short",
				expectedMsg: `value doesn't match all schemas from "allOf"`,
				schemaDef: `"definitions": {
						"test1": { "type":"string", "minLength": 10000 },
						"test2": { "type":"string", "minLength": 100000}
					}, "allOf": [{"$ref": "#/definitions/test1"},{"$ref": "#/definitions/test2"}]`,
			},
		}

		draft4SpecificFields := map[string]validateFieldTestContext{
			"exclusiveMinimum": {
				invalidData: 0,
				expectedMsg: "number must be more than 0",
				schemaDef:   `"type": "integer", "minimum": 0, "exclusiveMinimum": true`,
			},
			"exclusiveMaximum": {
				invalidData: 100,
				expectedMsg: "number must be less than 100",
				schemaDef:   `"type": "integer", "maximum": 100, "exclusiveMaximum": true`,
			},
		}

		// Draft 6+ fields
		draft6PlusFields := map[string]validateFieldTestContext{
			"constField": {
				invalidData: "wrong-value",
				expectedMsg: `value must be equal to "expected-value"`,
				schemaDef:   `"const": "expected-value"`,
			},
			"containsField": {
				invalidData: []any{"foo", true},
				expectedMsg: "no items match contains schema",
				schemaDef:   `"contains": {"type": "number","multipleOf": 2}`,
			},
			"exclusiveMinField": {
				invalidData: 0,
				expectedMsg: "number must be more than 0",
				schemaDef:   `"type": "integer", "exclusiveMinimum": 0`,
			},
			"exclusiveMaxField": {
				invalidData: 100,
				expectedMsg: "number must be less than 100",
				schemaDef:   `"type": "integer","exclusiveMaximum": 100`,
			},
			"propertyNamesField": {
				invalidData: map[string]any{"validName": 1, "invalid-name!": 2},
				expectedMsg: `invalid propertyName "invalid-name!"`,
				schemaDef:   `"type": "object", "propertyNames": { "type": "string", "pattern": "^[a-zA-Z_][a-zA-Z0-9_]*$" }`,
			},
		}

		// Draft 2019-09+ fields
		draft2019PlusFields := map[string]validateFieldTestContext{
			"minContains": {
				invalidData: []any{1},
				expectedMsg: "min 2 items required to match contains schema, but matched 1 items",
				schemaDef:   `"contains": {"const": 1}, "minContains": 2`,
			},
			"maxContains": {
				invalidData: []any{1, 1, 1},
				expectedMsg: "max 2 items required to match contains schema, but matched 3 items",
				schemaDef:   `"contains": {"const": 1}, "maxContains": 2`,
			},
			"dependentRequired": {
				invalidData: map[string]any{"a": 1},
				expectedMsg: `{"a":"properties b required, if a exists"}`,
				schemaDef:   `"dependentRequired": { "a": ["b"] }`,
			},
		}

		BeforeAll(func() {
			c = jsonschemaV6.NewCompiler()
			c.AssertContent()
			c.AssertFormat()
		})

		It("openapi3.0", func() {
			versionFields, invalidData, expectedErrors := getFullVersionFields(baseFields, draft4SpecificFields)
			delete(invalidData, "requiredField")
			var fieldsSchemas []string
			for field, ctx := range versionFields {
				fieldsSchemas = append(fieldsSchemas, fmt.Sprintf(`"%s": { %s }`, field, ctx.schemaDef))
			}
			schemaDef := fmt.Sprintf(`{
						"type": "object",
						"required": ["requiredField"],
						"properties": {
							%s
						}
					}`, strings.Join(fieldsSchemas, ","))
			validator := jsonschema.New(schemaDef, c)
			err := validator.Validate(&jsonschema.ValidatorContext{
				HTTPRequest: &jsonschema.HTTPRequest{
					Data: invalidData,
				},
			})
			Expect(err).ToNot(BeNil(), "Expect not nil error")

			var errMap map[string]interface{}
			b, _ := json.Marshal(err)
			json.Unmarshal(b, &errMap)

			actualFields, ok := errMap["fields"].(map[string]interface{})
			Expect(ok).To(BeTrue(), "Error should have 'fields' map for %s", string(b))
			Expect(len(actualFields)).To(Equal(19))

			// Verify each expected field error message
			for field, expectedMsg := range expectedErrors {
				actualMsg, exists := actualFields[field]

				Expect(exists).To(BeTrue(), "Field '%s' should have error for %s", field, actualMsg)
				Expect(expectedMsg).To(Equal(actualMsg), "Field '%s' error message should be consistent for [%s]. Got: [%v]", field, expectedMsg, actualMsg)
			}
		})

		It("draft4", func() {
			draft4Fields := utils.MergeGenericMap(nil, draft4SpecificFields)
			draft4Fields = utils.MergeGenericMap(draft4Fields, jsonschemaExtraFields)
			versionFields, invalidData, expectedErrors := getFullVersionFields(baseFields, draft4Fields)

			actualFields := make(map[string]interface{})
			schemaDefs := buildSchema(jsonschema.Draft_4, versionFields)
			for field, schemaDef := range schemaDefs {
				validator := jsonschema.New(schemaDef, c)
				err := validator.Validate(&jsonschema.ValidatorContext{
					HTTPRequest: &jsonschema.HTTPRequest{
						Data: invalidData[field],
					},
				})
				if vErr, ok := err.(*errs.ValidateError); ok {
					for _, v := range vErr.Fields {
						actualFields[field] = v
					}
				}
			}

			Expect(len(actualFields)).To(Equal(23))

			// Verify each expected field error message
			for field, expectedMsg := range expectedErrors {
				actualMsg, exists := actualFields[field]

				Expect(exists).To(BeTrue(), "Field '%s' should have error for %s", field, actualMsg)
				Expect(actualMsg).To(Equal(expectedMsg), "Field '%s' error message should be consistent for [%s]. Got: [%v]", field, expectedMsg, actualMsg)
			}
		})

		// draft 6 and 7 are similar
		It("draft6 or draft7", func() {
			draft6Fields := draft6PlusFields
			draft6Fields = utils.MergeGenericMap(draft6Fields, jsonschemaExtraFields)
			versionFields, invalidData, expectedErrors := getFullVersionFields(baseFields, draft6Fields)
			actualFields := make(map[string]interface{})
			schemaDefs := buildSchema(jsonschema.Draft_6, versionFields)
			for field, schemaDef := range schemaDefs {
				validator := jsonschema.New(schemaDef, c)
				err := validator.Validate(&jsonschema.ValidatorContext{
					HTTPRequest: &jsonschema.HTTPRequest{
						Data: invalidData[field],
					},
				})

				if vErr, ok := err.(*errs.ValidateError); ok {
					for _, v := range vErr.Fields {
						actualFields[field] = v
					}
				}
			}
			Expect(len(actualFields)).To(Equal(26))

			for field, expectedMsg := range expectedErrors {
				actualMsg, exists := actualFields[field]
				Expect(exists).To(BeTrue(), "Field '%s' should have error", field)
				var msg interface{}
				switch actualMsg.(type) {
				case map[string]interface{}, []interface{}:
					b, _ := json.Marshal(actualMsg)
					msg = string(b)
				case string:
					msg = actualMsg.(string)
				default:
					msg = actualMsg
				}
				Expect(msg).To(Equal(expectedMsg), "Field '%s' error message should be consistent for [%s]. Got: [%v]", field, expectedMsg, msg)
			}
		})

		// draft 2019 and 2020 are similar
		It("draft2019 or draft2020", func() {
			draft2019Fields := draft2019PlusFields
			draft2019Fields = utils.MergeGenericMap(draft2019Fields, draft6PlusFields)
			draft2019Fields = utils.MergeGenericMap(draft2019Fields, jsonschemaExtraFields)
			versionFields, invalidData, expectedErrors := getFullVersionFields(nil, draft2019Fields)
			actualFields := make(map[string]interface{})
			schemaDefs := buildSchema(jsonschema.Draft_2019, versionFields)
			for field, schemaDef := range schemaDefs {
				validator := jsonschema.New(schemaDef, c)
				err := validator.Validate(&jsonschema.ValidatorContext{
					HTTPRequest: &jsonschema.HTTPRequest{
						Data: invalidData[field],
					},
				})

				if vErr, ok := err.(*errs.ValidateError); ok {
					for _, v := range vErr.Fields {
						actualFields[field] = v
					}
				}
			}
			Expect(len(actualFields)).To(Equal(29))

			for field, expectedMsg := range expectedErrors {
				actualMsg, exists := actualFields[field]
				Expect(exists).To(BeTrue(), "Field '%s' should have error", field)
				var msg interface{}
				switch actualMsg.(type) {
				case map[string]interface{}, []interface{}:
					b, _ := json.Marshal(actualMsg)
					msg = string(b)
				case string:
					msg = actualMsg.(string)
				default:
					msg = actualMsg
				}
				Expect(msg).To(Equal(expectedMsg), "Field '%s' error message should be consistent for [%s]. Got: [%v]", field, expectedMsg, msg)
			}
		})
	})
})
