// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/laenzlinger/openpnp-tools/internal/config"
	"github.com/spf13/cobra"
)

var statusRepoFlag string

var configStatusCmd = &cobra.Command{
	Use:           "status",
	Short:         "Show OpenPnP process status and config drift",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		running := config.IsOpenPnPRunning()
		if running {
			fmt.Println("OpenPnP: RUNNING")
		} else {
			fmt.Println("OpenPnP: STOPPED")
		}

		repo := statusRepoFlag
		if repo == "" {
			repo = "."
		}
		results, err := config.CompareFiles(configDirFlag, repo)
		if err != nil {
			return err
		}

		fmt.Println("Config drift:")
		for _, r := range results {
			fmt.Printf("  %-25s %s\n", r.Name, r.Status)
		}

		if running {
			return fmt.Errorf("OpenPnP is running")
		}
		return nil
	},
}

func init() {
	configCmd.AddCommand(configStatusCmd)
	configStatusCmd.Flags().StringVar(&statusRepoFlag, "repo", "",
		"repo directory to compare against (default: current directory)")
}
