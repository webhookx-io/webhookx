package jsonschema

import (
	"encoding/json"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestJSONSchema(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Schema Validator Suite")
}

var _ = Describe("Schema Validator Plugin", func() {

	Context("JSONSchema Validator", func() {
		It("should validate valid JSON data against the schema", func() {
			schemaDef := `{
				"type": "object",
				"properties": {
					"name": { "type": "string" },
					"age": { "type": "integer", "minimum": 0 }
				},
				"required": ["name", "age"]
			}`

			validator := New([]byte(schemaDef))

			validData := map[string]interface{}{"name": "John Doe", "age": 30}
			ctx := &ValidatorContext{
				HTTPRequest: &HTTPRequest{
					Data: validData,
				},
			}

			err := validator.Validate(ctx)
			Expect(err).To(BeNil())
		})

		It("should return an error for invalid JSON data against the schema", func() {
			schemaDef := `{
				"type": "object",
				"properties": {
					"name": { "type": "string" },
					"age": { "type": "integer", "minimum": 0 }
				},
				"required": ["name", "age"]
			}`

			validator := New([]byte(schemaDef))

			invalidData := map[string]interface{}{"name": "John Doe", "age": -5}
			ctx := &ValidatorContext{
				HTTPRequest: &HTTPRequest{
					Data: invalidData,
				},
			}

			err := validator.Validate(ctx)
			Expect(err).ToNot(BeNil())
			b, _ := json.Marshal(err)
			Expect(string(b)).To(Equal(`{"message":"request validation","fields":{"age":"number must be at least 0"}}`))
		})
	})
})
