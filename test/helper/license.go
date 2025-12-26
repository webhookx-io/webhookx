package helper

import (
	apilicense "github.com/webhookx-io/webhookx/api/license"
	"github.com/webhookx-io/webhookx/pkg/license"
)

type MockLicenser struct{}

func (l *MockLicenser) License() *apilicense.License {
	license := apilicense.NewFree()
	license.Plan = "enterprise"
	return license
}
func (l *MockLicenser) Allow(feature string) bool                                  { return true }
func (l *MockLicenser) AllowAPI(workspace string, path string, method string) bool { return true }
func (l *MockLicenser) AllowPlugin(plugin string) bool                             { return true }

func ReplaceLicenser(licenser license.Licenser) func() {
	if licenser == nil {
		licenser = &MockLicenser{}
	}
	original := license.GetLicenser()
	reset := func() { license.SetLicenser(original) }
	license.SetLicenser(licenser)
	return reset
}
