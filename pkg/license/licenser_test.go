package license

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/api/license"
)

func Test(t *testing.T) {
	license := license.New()
	license.Plan = "enterprise"
	licenser := NewLicenser(license)
	assert.Equal(t, true, licenser.Allow("secret"))

	license.ExpiredAt = time.Now()
	assert.Equal(t, false, licenser.Allow("secret")) // should be false when license is expired
}
