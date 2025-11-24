package reference

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

var (
	ErrReferenceInvalid = errors.New("invalid reference")
)

// Reference represents the definition of a reference.
// Syntax: {secret://<provider>/<name>[.<jsonpath>][?<parameters>]}
type Reference struct {
	Reference   string
	Provider    string
	Name        string
	JsonPointer string
	Properties  map[string]string
}

func (r *Reference) String() string {
	return r.Reference
}

func Parse(reference string) (*Reference, error) {
	s := strings.TrimPrefix(reference, "{")
	s = strings.TrimSuffix(s, "}")
	u, err := url.Parse(s)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrReferenceInvalid, err)
	}
	if u.Scheme != "secret" {
		return nil, fmt.Errorf("%w: %q", ErrReferenceInvalid, "invalid reference scheme")
	}
	if u.Host == "" {
		return nil, fmt.Errorf("%w: %q", ErrReferenceInvalid, "invalid reference provider")
	}
	if u.Path == "" || u.Path == "/" {
		return nil, fmt.Errorf("%w: %q", ErrReferenceInvalid, "invalid reference name")
	}
	values, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return nil, fmt.Errorf("%w: %q", ErrReferenceInvalid, "invalid reference properties")
	}

	ref := &Reference{
		Reference:  reference,
		Name:       strings.TrimPrefix(u.Path, "/"),
		Provider:   u.Host,
		Properties: make(map[string]string),
	}
	for k := range values {
		ref.Properties[k] = values.Get(k)
	}
	if parts := strings.SplitN(ref.Name, ".", 2); len(parts) == 2 {
		ref.Name = parts[0]
		ref.JsonPointer = parts[1]
	}
	return ref, err
}

func IsReference(s string) bool {
	return strings.HasPrefix(s, "{secret://") && strings.HasSuffix(s, "}")
}
