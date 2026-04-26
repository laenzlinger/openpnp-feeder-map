// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/laenzlinger/openpnp-tools/internal/generate"
	"github.com/spf13/cobra"
)

var (
	generateOutputDir string
	generateBoardName string
	packageMapFlag    string
	boardWidth        float64
	boardHeight       float64
)

var generateCmd = &cobra.Command{
	Use:   "generate [board.csv]",
	Short: "Generate OpenPnP board XML from KiCad position CSV",
	Long: `Reads a KiCad position CSV (from stdin or file) and generates an OpenPnP
board XML file with remapped package names.

Fiducials are auto-detected from footprint names containing "Fiducial".

Example (KiCad 10+ writes to file):
  kicad-cli pcb export pos --format csv --side both --units mm --smd-only --exclude-dnp board.kicad_pcb
  openpnp-tools generate -o pnp/ -n myboard board.csv

Example (KiCad ≤9 writes to stdout):
  kicad-cli pcb export pos --format csv --side both --units mm --smd-only --exclude-dnp board.kicad_pcb \
    | openpnp-tools generate -o pnp/ -n myboard`,
	RunE: func(cmd *cobra.Command, args []string) error {
		home, _ := os.UserHomeDir()
		if packageMapFlag == "" {
			packageMapFlag = filepath.Join(home, ".openpnp2", "openpnp-package-map.csv")
		}

		pkgMap, err := generate.LoadPackageMap(packageMapFlag)
		if err != nil {
			return fmt.Errorf("loading package map: %w", err)
		}

		var input *os.File
		if len(args) > 0 {
			input, err = os.Open(args[0])
			if err != nil {
				return fmt.Errorf("opening input: %w", err)
			}
			defer func() { _ = input.Close() }()
		} else {
			input = os.Stdin
		}

		placements, err := generate.ParseKiCadCSV(input, pkgMap)
		if err != nil {
			return fmt.Errorf("parsing CSV: %w", err)
		}

		if err := os.MkdirAll(generateOutputDir, 0o750); err != nil {
			return fmt.Errorf("creating output dir: %w", err)
		}

		w, h := boardWidth, boardHeight
		if w == 0 && h == 0 {
			w, h = generate.BoundingBox(placements)
		}

		boardPath := filepath.Join(generateOutputDir, generateBoardName+".board.xml")
		if err := generate.WriteBoardXML(placements, boardPath, generateBoardName, w, h); err != nil {
			return fmt.Errorf("writing board XML: %w", err)
		}

		posPath := filepath.Join(generateOutputDir, generateBoardName+".pos")
		if err := generate.WritePosFile(placements, posPath); err != nil {
			return fmt.Errorf("writing pos file: %w", err)
		}

		stats := generate.Stats(placements)
		fmt.Printf("Wrote %d placements (%d active, %d fiducials, %d disabled) to %s\n",
			stats.Total, stats.Active, stats.Fiducials, stats.Disabled, generateOutputDir)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)
	generateCmd.Flags().StringVarP(&generateOutputDir, "output-dir", "o", "pnp", "output directory")
	generateCmd.Flags().StringVarP(&generateBoardName, "name", "n", "granit", "board name (used for filenames)")
	generateCmd.Flags().Float64Var(&boardWidth, "board-width", 0, "board width in mm")
	generateCmd.Flags().Float64Var(&boardHeight, "board-height", 0, "board height in mm")
	generateCmd.Flags().StringVar(&packageMapFlag, "package-map", "",
		"path to package map CSV (default: ~/.openpnp2/openpnp-package-map.csv)")
}
