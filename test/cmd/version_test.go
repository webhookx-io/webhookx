package cmd

import (
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/cmd"
)

var _ = Describe("version", Ordered, func() {
	It("outputs version", func() {
		output, err := executeCommand(cmd.Command(), "version")
		assert.Nil(GinkgoT(), err)
		assert.Equal(GinkgoT(), "WebhookX dev (unknown)\n", output)
	})
})
