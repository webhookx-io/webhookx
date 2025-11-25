package license

import (
	"github.com/webhookx-io/webhookx/api/license"
)

type Licenser interface {
	Allow(feature string) bool
	License() *license.License
}

type DefaultLicenser struct {
	license *license.License
}

func NewLicenser(license *license.License) *DefaultLicenser {
	return &DefaultLicenser{
		license: license,
	}
}

func (l *DefaultLicenser) Allow(feature string) bool {
	if l.license.Expired() {
		return false
	}
	plan := l.license.Plan
	return plans[plan].HasFeature(feature)
}

func (l *DefaultLicenser) License() *license.License {
	return l.license
}
