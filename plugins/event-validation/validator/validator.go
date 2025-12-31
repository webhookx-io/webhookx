package validator

import (
	"fmt"
	"hash/fnv"
	"strings"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

type ValidateError []error

func (ve ValidateError) Error() string {
	sb := strings.Builder{}
	for i, e := range ve {
		sb.WriteString(e.Error())
		if i != len(ve)-1 {
			sb.WriteString(" | ")
		}
	}
	return sb.String()
}

func (ve ValidateError) Errors() []string {
	strs := make([]string, len(ve))
	for i, e := range ve {
		strs[i] = e.Error()
	}
	return strs
}

type Validator interface {
	Validate(value interface{}) error
}

var cache, _ = lru.New[uint64, Validator](128)

func hash(str string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(str))
	return h.Sum64()
}

func NewValidator(version string, schemaDef string) (Validator, error) {
	cacheKey := hash(version + "." + schemaDef)
	validator, exist := cache.Get(cacheKey)
	if exist {
		return validator, nil
	}

	var err error
	switch version {
	case "draft-04":
		validator, err = NewJsonSchemaValidator(jsonschema.Draft4, schemaDef)
	case "draft-06":
		validator, err = NewJsonSchemaValidator(jsonschema.Draft6, schemaDef)
	case "draft-07":
		validator, err = NewJsonSchemaValidator(jsonschema.Draft7, schemaDef)
	case "draft-2019-09":
		validator, err = NewJsonSchemaValidator(jsonschema.Draft2019, schemaDef)
	case "draft-2020-12":
		validator, err = NewJsonSchemaValidator(jsonschema.Draft2020, schemaDef)
	case "openapi-3.0":
		validator, err = NewOpenApiValidator(schemaDef)
	default:
		err = fmt.Errorf("unsupported version: %s", version)
	}

	if err != nil {
		return nil, err
	}

	_ = cache.Add(cacheKey, validator)
	return validator, nil
}
