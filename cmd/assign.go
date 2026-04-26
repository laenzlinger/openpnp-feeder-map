// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"

	"github.com/laenzlinger/openpnp-tools/internal/generate"
	"github.com/laenzlinger/openpnp-tools/internal/openpnp"
	"github.com/spf13/cobra"
)

var dryRunFlag bool
var resetUnusedFlag bool
var dummyPartFlag string
var assignStripLength float64

var assignCmd = &cobra.Command{
	Use:   "assign <feeders.csv>",
	Short: "Assign parts to feeders in machine.xml",
	Long: `Reads a CSV file mapping feeder names to part IDs and updates machine.xml accordingly.

The CSV must have columns: feeder,part

Example:
  feeder,part
  LV08-01,C_0805-100n
  RH12-01,SOT-223-NCP1117-3.3

Feeders not listed in the CSV are left unchanged.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		csvPath := args[0]

		assignments, err := parseFeederCSV(csvPath)
		if err != nil {
			return err
		}

		if dryRunFlag {
			fmt.Printf("Dry run: %d assignments from %s\n", len(assignments), csvPath)
			for _, a := range assignments {
				fmt.Printf("  %s → %s\n", a.FeederName, a.PartID)
			}
			return nil
		}

		home, _ := os.UserHomeDir()
		if packageMapFlag == "" {
			packageMapFlag = filepath.Join(home, ".openpnp2", "openpnp-package-map.csv")
		}
		pkgMap, err := generate.LoadPackageMap(packageMapFlag)
		if err != nil {
			return fmt.Errorf("loading package map: %w", err)
		}

		results, err := openpnp.AssignFeeders(machineFlag, assignments, pkgMap, resetUnusedFlag, dummyPartFlag, assignStripLength)
		if err != nil {
			return err
		}

		assigned, unchanged, notFound, reset := 0, 0, 0, 0
		for _, r := range results {
			switch r.Status {
			case "assigned":
				fmt.Printf("  ✓ %s: %s → %s\n", r.FeederName, r.OldPartID, r.PartID)
				assigned++
			case "unchanged":
				fmt.Printf("  = %s: %s\n", r.FeederName, r.PartID)
				unchanged++
			case "not_found":
				fmt.Printf("  ? %s: not found in machine.xml\n", r.FeederName)
				notFound++
			case "reset":
				fmt.Printf("  ✗ %s: %s → %s\n", r.FeederName, r.OldPartID, r.PartID)
				reset++
			}
		}
		fmt.Printf("\n%d assigned, %d unchanged, %d not found, %d reset\n",
			assigned, unchanged, notFound, reset)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(assignCmd)
	assignCmd.Flags().BoolVarP(&resetUnusedFlag, "reset-unused", "r", false,
		"reset feeders not in CSV to dummy part")
	assignCmd.Flags().StringVar(&dummyPartFlag, "dummy-part", "CALIBRATION-DUMMY",
		"part-id to use when resetting unused feeders")
	assignCmd.Flags().Float64Var(&assignStripLength, "strip-length", 120, "strip feeder slot length in mm")
	assignCmd.Flags().BoolVarP(&dryRunFlag, "dry-run", "n", false,
		"show what would be changed without modifying machine.xml")
}

func parseFeederCSV(path string) ([]openpnp.FeederAssignment, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening CSV: %w", err)
	}
	defer func() { _ = f.Close() }()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("reading CSV: %w", err)
	}

	var assignments []openpnp.FeederAssignment
	for i, row := range records {
		if i == 0 && row[0] == "feeder" {
			continue // skip header
		}
		if len(row) < 2 {
			continue
		}
		assignments = append(assignments, openpnp.FeederAssignment{
			FeederName: row[0],
			PartID:     row[1],
		})
	}
	return assignments, nil
}
