package cmd

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"os/exec"
	"testing"
)

var _ = Describe("version", Ordered, func() {
	It("outputs version", func() {
		stdout, err := exec.Command("webhookx", "version").Output()
		assert.Nil(GinkgoT(), err)
		assert.Regexp(GinkgoT(), "WebhookX \\w* \\([0-9a-z]{7}\\) \n", string(stdout))
	})
})

func TestCommandVersion(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CMD version suite")
}
