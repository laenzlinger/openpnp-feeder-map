// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var machineFlag string
var packageMapFlag string

var rootCmd = &cobra.Command{
	Use:               "openpnp-tools",
	Short:             "OpenPnP feeder map and management tool",
	Long:              `Generates interactive feeder maps and manages feeder assignments for OpenPnP jobs.`,
	DisableAutoGenTag: true,
}

func Execute(version string) {
	rootCmd.Version = version
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	home, _ := os.UserHomeDir()
	defaultMachine := filepath.Join(home, ".openpnp2", "machine.xml")
	rootCmd.PersistentFlags().StringVar(&machineFlag, "machine", defaultMachine, "path to machine.xml")
	defaultPkgMap := filepath.Join(home, ".openpnp2", "openpnp-package-map.csv")
	rootCmd.PersistentFlags().StringVar(&packageMapFlag, "package-map", defaultPkgMap, "path to package map CSV")
}
