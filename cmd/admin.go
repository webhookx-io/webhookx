package cmd

import (
	"github.com/spf13/cobra"
)

func newAdminCmd() *cobra.Command {
	admin := &cobra.Command{
		Use:   "admin",
		Short: "Admin commands",
		Long:  ``,
	}

	admin.AddCommand(newAdminSyncCmd())
	admin.AddCommand(newAdminDumpCmd())

	return admin
}
