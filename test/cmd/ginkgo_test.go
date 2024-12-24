package cmd

import (
	"bytes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
	"testing"
)

func executeCommand(root *cobra.Command, args ...string) (output string, err error) {
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)

	_, err = root.ExecuteC()
	return buf.String(), err
}

func TestCommand(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CMD Suite")
}
