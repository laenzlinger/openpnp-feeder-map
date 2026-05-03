// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"github.com/laenzlinger/openpnp-tools/internal/config"
	"github.com/spf13/cobra"
)

var configDirFlag string

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage OpenPnP configuration (backup, apply, pull, status)",
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.PersistentFlags().StringVar(&configDirFlag, "config-dir", config.DefaultConfigDir(),
		"path to OpenPnP config directory")
}
