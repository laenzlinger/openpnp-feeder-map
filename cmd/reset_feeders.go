// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/laenzlinger/openpnp-tools/internal/openpnp"
	"github.com/spf13/cobra"
)

var resetDummyPartFlag string

var resetFeedersCmd = &cobra.Command{
	Use:   "reset-feeders",
	Short: "Reset all feeder assignments to prepare for version control",
	Long: `Resets all strip feeder part-ids to a dummy part and zeroes feed counts.

Leaves feeder positions, calibration, tape-type, rotation, pitch, and all
other tuned values untouched. Use this before committing machine.xml to
version control so the base config is project-neutral.

Workflow:
  1. Tune in OpenPnP, save and exit
  2. Copy live config to chezmoi source
  3. Run: openpnp-tools reset-feeders --machine path/to/chezmoi/machine.xml
  4. Commit — only calibration/tuning changes appear in the diff
  5. Run: openpnp-tools assign feeders.csv  (to restore project assignments)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		results, err := openpnp.ResetFeeders(machineFlag, resetDummyPartFlag)
		if err != nil {
			return err
		}

		resetCount, unchanged := 0, 0
		for _, r := range results {
			switch r.Status {
			case "reset":
				fmt.Printf("  ✗ %s: %s → %s\n", r.FeederName, r.OldPartID, resetDummyPartFlag)
				resetCount++
			case "unchanged":
				fmt.Printf("  = %s\n", r.FeederName)
				unchanged++
			}
		}
		fmt.Printf("\n%d reset, %d unchanged\n", resetCount, unchanged)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(resetFeedersCmd)
	resetFeedersCmd.Flags().StringVar(&resetDummyPartFlag, "dummy-part", "CALIBRATION-DUMMY",
		"part-id to use for reset feeders")
}
