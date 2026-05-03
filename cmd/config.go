// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"github.com/laenzlinger/openpnp-tools/internal/config"
	"github.com/spf13/cobra"
)

var configDirFlag string
var settings config.Settings

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage OpenPnP configuration (backup, apply, pull, status)",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		s, err := config.LoadSettings(configDirFlag)
		if err != nil {
			return err
		}
		settings = s
		return nil
	},
}

// repoDir returns the effective repo directory: explicit flag value, or settings, or ".".
func repoDir(flag string) string {
	if flag != "" {
		return flag
	}
	if settings.RepoDir != "" {
		return settings.RepoDir
	}
	return "."
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.PersistentFlags().StringVar(&configDirFlag, "config-dir", config.DefaultConfigDir(),
		"path to OpenPnP config directory")
}
