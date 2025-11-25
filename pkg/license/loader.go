package license

import (
	"errors"
	"fmt"
	"os"

	"github.com/webhookx-io/webhookx/api/license"
)

var (
	ErrInvalidLicense = errors.New("license is invalid")
)

func init() {
	license.PublicKey = "1c085d754c3343b8e9ad280ec5470111e47a422066071c1723b609a917e5b772"
}

// Load loads license
func Load() (*license.License, error) {
	licenseJSON := os.Getenv("WEBHOOKX_LICENSE")
	if licenseJSON != "" {
		lic, err := license.ParseLicense(licenseJSON)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrInvalidLicense, err)
		}
		err = lic.Validate()
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrInvalidLicense, err)
		}
		return lic, nil
	}

	return license.NewFree(), nil
}
