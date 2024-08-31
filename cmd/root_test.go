package cmd

import (
	"bytes"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
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

func TestCMD(t *testing.T) {
	output, err := executeCommand(cmd, "")
	assert.Nil(t, err)
	assert.NotNil(t, output)
}
