// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/laenzlinger/openpnp-tools/internal/config"
	"github.com/spf13/cobra"
)

var configBackupCmd = &cobra.Command{
	Use:          "backup",
	Short:        "Backup ~/.openpnp2 config files (same format as OpenPnP)",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := config.Backup(configDirFlag)
		if err != nil {
			return err
		}
		fmt.Printf("  backup → %s\n", dir)
		return nil
	},
}

func init() {
	configCmd.AddCommand(configBackupCmd)
}
