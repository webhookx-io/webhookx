package license

import (
	"sync/atomic"

	"github.com/webhookx-io/webhookx/api/license"
)

var (
	globalLicenser = defaultGlobalLicenser()
)

type licenseHolder struct{ value Licenser }

func defaultGlobalLicenser() *atomic.Value {
	v := &atomic.Value{}
	v.Store(licenseHolder{NewLicenser(license.NewFree())})
	return v
}

func GetLicenser() Licenser {
	return globalLicenser.Load().(licenseHolder).value
}

func SetLicenser(licenser Licenser) {
	globalLicenser.Store(licenseHolder{licenser})
}
