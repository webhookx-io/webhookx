package helper

import (
	api_license "github.com/webhookx-io/webhookx/api/license"
	"github.com/webhookx-io/webhookx/pkg/license"
)

type mockLicenser struct{}

var _ license.Licenser = mockLicenser{}

func (l mockLicenser) Allow(feature string) bool { return true }
func (l mockLicenser) License() *api_license.License {
	license := api_license.NewFree()
	license.Plan = "enterprise"
	return license
}
func (l mockLicenser) AllowAPI(workspace string, path string, method string) bool { return true }

func MockLicenser(licenser license.Licenser) func() {
	if licenser == nil {
		licenser = mockLicenser{}
	}
	def := license.GetLicenser()
	reset := func() {
		license.SetLicenser(def)
	}
	license.SetLicenser(licenser)
	return reset
}
