// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/laenzlinger/openpnp-tools/internal/config"
	"github.com/spf13/cobra"
)

var applyFromFlag string
var applyNoBackupFlag bool

var configApplyCmd = &cobra.Command{
	Use:          "apply",
	Short:        "Apply base config to ~/.openpnp2 (backs up first, OpenPnP must be closed)",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.CheckNotRunning(); err != nil {
			return err
		}

		if !applyNoBackupFlag {
			dir, err := config.Backup(configDirFlag)
			if err != nil {
				return fmt.Errorf("backup failed: %w", err)
			}
			fmt.Printf("  backup → %s\n", dir)
		}

		from := repoDir(applyFromFlag)
		copied, err := config.CopyManagedFiles(from, configDirFlag)
		if err != nil {
			return err
		}
		for _, f := range copied {
			fmt.Printf("  %s applied\n", f)
		}
		fmt.Println("Applied config repo → " + configDirFlag)
		return nil
	},
}

func init() {
	configCmd.AddCommand(configApplyCmd)
	configApplyCmd.Flags().StringVar(&applyFromFlag, "from", "",
		"source directory (default: current directory)")
	configApplyCmd.Flags().BoolVar(&applyNoBackupFlag, "no-backup", false,
		"skip backup before applying")
}
