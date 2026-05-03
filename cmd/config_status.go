// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/laenzlinger/openpnp-tools/internal/config"
	"github.com/spf13/cobra"
)

var statusRepoFlag string
var statusDiffFlag bool

var configStatusCmd = &cobra.Command{
	Use:           "status",
	Short:         "Show OpenPnP process status and config drift",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		running := config.IsOpenPnPRunning()
		if running {
			fmt.Println("OpenPnP: RUNNING")
		} else {
			fmt.Println("OpenPnP: STOPPED")
		}

		repo := repoDir(statusRepoFlag)
		results, err := config.CompareFiles(configDirFlag, repo)
		if err != nil {
			return err
		}

		fmt.Println("Config drift:")
		for _, r := range results {
			fmt.Printf("  %-25s %s\n", r.Name, r.Status)
		}

		if statusDiffFlag {
			for _, r := range results {
				if r.Status != "modified" {
					continue
				}
				repoFile := filepath.Join(repo, r.Name)
				liveFile := filepath.Join(configDirFlag, r.Name)
				diff := exec.Command("diff", "-u", repoFile, liveFile)
				diff.Stdout = os.Stdout
				diff.Stderr = os.Stderr
				_ = diff.Run() // exit 1 means files differ, not an error
			}
		}

		if running {
			return fmt.Errorf("OpenPnP is running")
		}
		return nil
	},
}

func init() {
	configCmd.AddCommand(configStatusCmd)
	configStatusCmd.Flags().StringVar(&statusRepoFlag, "repo", "",
		"repo directory to compare against (default: current directory)")
	configStatusCmd.Flags().BoolVarP(&statusDiffFlag, "diff", "d", false,
		"show unified diff for modified files")
}
