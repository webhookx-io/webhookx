package plugins

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"testing"
)

func TestAdmin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plugins Suite")
}
