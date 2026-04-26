// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/laenzlinger/openpnp-feeder-map/internal/feedermap"
	"github.com/laenzlinger/openpnp-feeder-map/internal/openpnp"
	"github.com/spf13/cobra"
)

var mapOutputFlag string

var mapCmd = &cobra.Command{
	Use:   "map <job.xml>",
	Short: "Generate an interactive HTML feeder map",
	Long:  `Generates a self-contained HTML page showing feeder positions, job parts, and missing feeders.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		jobPath := args[0]

		jobParts, boards, err := openpnp.LoadJobParts(jobPath)
		if err != nil {
			return fmt.Errorf("loading job: %w", err)
		}

		machine, err := openpnp.ParseMachine(machineFlag)
		if err != nil {
			return fmt.Errorf("loading machine: %w", err)
		}

		data := feedermap.Build(jobParts, boards, machine)
		data.JobFile = filepath.Base(jobPath)

		f, err := os.Create(mapOutputFlag)
		if err != nil {
			return fmt.Errorf("creating output: %w", err)
		}
		defer func() { _ = f.Close() }()

		if err := feedermap.Render(f, data); err != nil {
			return fmt.Errorf("rendering: %w", err)
		}

		fmt.Printf("Feeder map: %d feeders, %d missing parts, %d unused → %s\n",
			len(data.Feeders), len(data.MissingParts), len(data.UnusedFeeders), mapOutputFlag)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(mapCmd)
	mapCmd.Flags().StringVarP(&mapOutputFlag, "output", "o", "feeder-map.html", "output HTML file path")
}
