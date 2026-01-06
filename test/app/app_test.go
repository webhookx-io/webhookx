package app_test

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/test/helper"
)

var _ = Describe("app", Ordered, func() {
	Context("logo", func() {
		start := func(env map[string]string) string {
			old := os.Stdout
			r, w, err := os.Pipe()
			assert.NoError(GinkgoT(), err)
			os.Stdout = w

			app := helper.MustStart(env)

			assert.NoError(GinkgoT(), app.Stop())

			_ = w.Close()
			os.Stdout = old

			var buf bytes.Buffer
			_, _ = io.Copy(&buf, r)

			return buf.String()
		}

		It("should print logo", func() {
			stdout := start(map[string]string{
				"WEBHOOKX_LOG_FILE": "",
			})
			expectedstdout := `
 _       __     __    __                __  _  __
| |     / /__  / /_  / /_  ____  ____  / /_| |/ /
| | /| / / _ \/ __ \/ __ \/ __ \/ __ \/ //_/   /
| |/ |/ /  __/ /_/ / / / / /_/ / /_/ / ,< /   |
|__/|__/\___/_.___/_/ /_/\____/\____/_/|_/_/|_|

- Version: dev
- Proxy URL: http://127.0.0.1:9700
- Admin URL: http://127.0.0.1:9701
- Status URL: http://127.0.0.1:9702
- Worker: on

`
			assert.Equal(GinkgoT(), strings.TrimPrefix(expectedstdout, "\n"), stdout)

			stdout = start(map[string]string{
				"WEBHOOKX_LOG_FILE":       "",
				"WEBHOOKX_PROXY_LISTEN":   "",
				"WEBHOOKX_ADMIN_LISTEN":   "",
				"WEBHOOKX_STATUS_LISTEN":  "",
				"WEBHOOKX_WORKER_ENABLED": "false",
			})
			expectedstdout = `
 _       __     __    __                __  _  __
| |     / /__  / /_  / /_  ____  ____  / /_| |/ /
| | /| / / _ \/ __ \/ __ \/ __ \/ __ \/ //_/   /
| |/ |/ /  __/ /_/ / / / / /_/ / /_/ / ,< /   |
|__/|__/\___/_.___/_/ /_/\____/\____/_/|_/_/|_|

- Version: dev
- Proxy URL: disabled
- Admin URL: disabled
- Status URL: disabled
- Worker: off

`
			assert.Equal(GinkgoT(), strings.TrimPrefix(expectedstdout, "\n"), stdout)
		})
	})

})

func TestApp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "App Suite")
}
