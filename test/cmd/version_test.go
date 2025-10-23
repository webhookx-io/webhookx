package cmd

import (
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/test/helper"
)

var _ = Describe("version", Ordered, func() {
	It("outputs version", func() {
		output, err := helper.ExecAppCommand("version")
		assert.Nil(GinkgoT(), err)
		assert.Equal(GinkgoT(), "WebhookX dev (unknown)\n", output)
	})
})
