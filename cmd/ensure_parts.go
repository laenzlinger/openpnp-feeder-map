// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/laenzlinger/openpnp-tools/internal/generate"
	"github.com/laenzlinger/openpnp-tools/internal/openpnp"
	"github.com/spf13/cobra"
)

var partsFileFlag string

var ensurePartsCmd = &cobra.Command{
	Use:   "ensure-parts <board.xml>",
	Short: "Ensure all board parts exist in parts.xml",
	Long: `Reads an OpenPnP board XML and ensures all referenced parts exist in parts.xml
with correct package-id and heights from the package map.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		boardPath := args[0]
		home, _ := os.UserHomeDir()

		if partsFileFlag == "" {
			partsFileFlag = filepath.Join(home, ".openpnp2", "parts.xml")
		}
		if packageMapFlag == "" {
			packageMapFlag = filepath.Join(home, ".openpnp2", "openpnp-package-map.csv")
		}

		pkgMap, err := generate.LoadPackageMap(packageMapFlag)
		if err != nil {
			return fmt.Errorf("loading package map: %w", err)
		}

		board, err := openpnp.ParseBoard(boardPath)
		if err != nil {
			return fmt.Errorf("loading board: %w", err)
		}

		seen := make(map[string]bool)
		var partIDs []string
		for _, p := range board.Placements {
			if p.PartID != "" && !seen[p.PartID] {
				seen[p.PartID] = true
				partIDs = append(partIDs, p.PartID)
			}
		}

		result, err := openpnp.EnsurePartsWithMap(partsFileFlag, partIDs, pkgMap)
		if err != nil {
			return err
		}

		for _, id := range result.Created {
			fmt.Printf("  + %s\n", id)
		}
		fmt.Printf("\n%d created, %d existed\n",
			len(result.Created), len(result.Existed))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(ensurePartsCmd)
	ensurePartsCmd.Flags().StringVar(&partsFileFlag, "parts", "",
		"path to parts.xml (default: ~/.openpnp2/parts.xml)")
}
