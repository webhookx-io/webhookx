package cmd

import (
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/test"
	"github.com/webhookx-io/webhookx/test/helper"
)

var _ = Describe("start", Ordered, func() {
	Context("errors", func() {
		It("should return error when configuration file is invalid", func() {
			output, err := helper.ExecAppCommand("start", "--config", "config.yml")
			assert.NotNil(GinkgoT(), err)
			assert.Equal(GinkgoT(), "Error: could not load configuration: open config.yml: no such file or directory\n", output)
		})

		It("should return error when configuration file is invalid", func() {
			output, err := helper.ExecAppCommand("start", "--config", test.FilePath("fixtures/malformed-config.yml"))
			assert.NotNil(GinkgoT(), err)
			assert.Equal(GinkgoT(), "Error: could not load configuration: yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `👻` into modules.SecretConfig\n", output)
		})

		It("should return error when configuration is invalid", func() {
			output, err := helper.ExecAppCommand("start", "--config", test.FilePath("fixtures/invalid-config.yml"))
			assert.NotNil(GinkgoT(), err)
			assert.Equal(GinkgoT(), "Error: invalid configuration: port must be in the range [0, 65535]\n", output)
		})
	})
})
