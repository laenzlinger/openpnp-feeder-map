// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/laenzlinger/openpnp-tools/internal/config"
	"github.com/laenzlinger/openpnp-tools/internal/openpnp"
	"github.com/spf13/cobra"
)

var pullToFlag string
var pullNoResetFlag bool
var pullDummyPartFlag string

var configPullCmd = &cobra.Command{
	Use:          "pull",
	Short:        "Pull live ~/.openpnp2 config into repo directory (OpenPnP must be closed)",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.CheckNotRunning(); err != nil {
			return err
		}

		to := pullToFlag
		if to == "" {
			to = "."
		}

		copied, err := config.CopyManagedFiles(configDirFlag, to)
		if err != nil {
			return err
		}
		for _, f := range copied {
			fmt.Printf("  %s synced\n", f)
		}

		if !pullNoResetFlag {
			machPath := filepath.Join(to, "machine.xml")
			results, err := openpnp.ResetFeeders(machPath, pullDummyPartFlag)
			if err != nil {
				return fmt.Errorf("resetting feeders: %w", err)
			}
			resetCount := 0
			for _, r := range results {
				if r.Status == openpnp.StatusReset {
					resetCount++
				}
			}
			fmt.Printf("  %d feeders reset\n", resetCount)
		}

		fmt.Println("Pulled " + configDirFlag + " → config repo (feeders reset)")
		return nil
	},
}

func init() {
	configCmd.AddCommand(configPullCmd)
	configPullCmd.Flags().StringVar(&pullToFlag, "to", "",
		"destination directory (default: current directory)")
	configPullCmd.Flags().BoolVar(&pullNoResetFlag, "no-reset", false,
		"skip reset-feeders after pulling")
	configPullCmd.Flags().StringVar(&pullDummyPartFlag, "dummy-part", "CALIBRATION-DUMMY",
		"part-id to use for reset feeders")
}
