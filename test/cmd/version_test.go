package cmd

import (
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/config"
	"os/exec"
	"testing"
)

var _ = Describe("version", Ordered, func() {
	It("outputs version", func() {
		stdout, err := exec.Command("webhookx", "version").Output()
		assert.Nil(GinkgoT(), err)
		assert.Equal(GinkgoT(),
			fmt.Sprintf("WebhookX %s (%s) \n", config.VERSION, config.COMMIT),
			string(stdout))
	})
})

func TestCommandVersion(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CMD version suite")
}
