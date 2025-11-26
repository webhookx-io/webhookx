package license

import "slices"

var plans = map[string]Plan{
	"free": {
		Features: []string{},
		Limits:   map[string]int{},
	},
	"enterprise": {
		Features: []string{"secret", "workspace"},
		Limits:   map[string]int{},
	},
}

type Plan struct {
	Features []string
	Limits   map[string]int
}

func (p Plan) HasFeature(feature string) bool {
	return slices.Contains(p.Features, feature)
}
