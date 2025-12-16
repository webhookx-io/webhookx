package license

import (
	"slices"

	"github.com/webhookx-io/webhookx/api/license"
)

type Licenser interface {
	Allow(feature string) bool
	AllowAPI(workspace string, path string, method string) bool
	AllowPlugin(plugin string) bool
	License() *license.License
}

var _ Licenser = &DefaultLicenser{}

type DefaultLicenser struct {
	license *license.License
}

func NewLicenser(license *license.License) *DefaultLicenser {
	return &DefaultLicenser{
		license: license,
	}
}

func (l *DefaultLicenser) Allow(feature string) bool {
	plan := l.plan()
	return plans[plan].HasFeature(feature)
}

func (l *DefaultLicenser) AllowPlugin(plugin string) bool {
	plan := l.plan()
	return plans[plan].HasPlugin(plugin)
}

func (l *DefaultLicenser) AllowAPI(workspace string, path string, method string) bool {
	plan := l.plan()
	if api := plans[plan].ForbiddenAPIs[path]; api != nil {
		if slices.Contains(api.Methods, method) {
			if !api.ExcludeDefaultWorkspace {
				return false
			}
			if workspace != "default" {
				return false
			}
		}
	}
	return true
}

func (l *DefaultLicenser) License() *license.License {
	return l.license
}

func (l *DefaultLicenser) plan() string {
	if l.license.Expired() {
		return "free"
	}
	return l.license.Plan
}
