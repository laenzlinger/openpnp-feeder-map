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
var packagesFileFlag string

var ensurePartsCmd = &cobra.Command{
	Use:   "ensure-parts <board.xml>",
	Short: "Ensure all board parts and packages exist in OpenPnP config",
	Long: `Reads an OpenPnP board XML and ensures all referenced parts and packages
exist in parts.xml and packages.xml with correct metadata from the package map.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		boardPath := args[0]
		home, _ := os.UserHomeDir()

		if partsFileFlag == "" {
			partsFileFlag = filepath.Join(home, ".openpnp2", "parts.xml")
		}
		if packagesFileFlag == "" {
			packagesFileFlag = filepath.Join(home, ".openpnp2", "packages.xml")
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

		// Extract package using package map for correct split
		seenParts := make(map[string]bool)
		seenPkgs := make(map[string]bool)
		var partIDs []string
		var packageIDs []string
		for _, p := range board.Placements {
			if p.PartID == "" {
				continue
			}
			if !seenParts[p.PartID] {
				seenParts[p.PartID] = true
				partIDs = append(partIDs, p.PartID)
			}
			pkg := extractPackage(p.PartID, pkgMap)
			if !seenPkgs[pkg] {
				seenPkgs[pkg] = true
				packageIDs = append(packageIDs, pkg)
			}
		}

		// Ensure packages first
		pkgResult, err := openpnp.EnsurePackages(packagesFileFlag, packageIDs, pkgMap)
		if err != nil {
			return err
		}
		for _, id := range pkgResult.Created {
			fmt.Printf("  + package: %s\n", id)
		}
		for _, id := range pkgResult.Updated {
			fmt.Printf("  ~ package: %s (tape-specification updated)\n", id)
		}

		// Then ensure parts
		partResult, err := openpnp.EnsurePartsWithMap(partsFileFlag, partIDs, pkgMap)
		if err != nil {
			return err
		}
		for _, id := range partResult.Created {
			fmt.Printf("  + part: %s\n", id)
		}

		fmt.Printf("\nPackages: %d created, %d existed\n",
			len(pkgResult.Created), len(pkgResult.Existed))
		fmt.Printf("Parts: %d created, %d existed\n",
			len(partResult.Created), len(partResult.Existed))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(ensurePartsCmd)
	ensurePartsCmd.Flags().StringVar(&partsFileFlag, "parts", "",
		"path to parts.xml (default: ~/.openpnp2/parts.xml)")
	ensurePartsCmd.Flags().StringVar(&packagesFileFlag, "packages", "",
		"path to packages.xml (default: ~/.openpnp2/packages.xml)")
}

// extractPackage finds the package name from a part-id using the package map.
// Falls back to last-dash split if no match found.
func extractPackage(partID string, pkgMap *generate.PackageMap) string {
	// Try each possible split point, check if the prefix is a known package
	for i := len(partID) - 1; i > 0; i-- {
		if partID[i] == '-' {
			candidate := partID[:i]
			if _, ok := pkgMap.LookupByPackage(candidate); ok {
				return candidate
			}
		}
	}
	// Fallback: last dash
	for i := len(partID) - 1; i > 0; i-- {
		if partID[i] == '-' {
			return partID[:i]
		}
	}
	return partID
}
