// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/laenzlinger/openpnp-tools/internal/config"
	"github.com/laenzlinger/openpnp-tools/internal/openpnp"
	"github.com/spf13/cobra"
)

var saveMessageFlag string

var configSaveCmd = &cobra.Command{
	Use:          "save",
	Short:        "Pull tuning, reset feeders, commit and push config repo",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.CheckNotRunning(); err != nil {
			return err
		}

		repo := repoDir("")
		if repo == "." {
			return fmt.Errorf("repo-dir not configured — set it in %s/%s", configDirFlag, config.FileName)
		}

		copied, err := config.CopyManagedFiles(configDirFlag, repo)
		if err != nil {
			return err
		}
		for _, f := range copied {
			fmt.Printf("  %s synced\n", f)
		}

		machPath := filepath.Join(repo, "machine.xml")
		results, err := openpnp.ResetFeeders(machPath, "CALIBRATION-DUMMY")
		if err != nil {
			return fmt.Errorf("resetting feeders: %w", err)
		}
		resetCount := 0
		for _, r := range results {
			if r.Status == openpnp.StatusReset {
				resetCount++
			}
		}
		fmt.Printf("  %d feeders reset\n", resetCount)

		// git add + commit + push
		if err := gitRun(repo, "add", "-A"); err != nil {
			return fmt.Errorf("git add: %w", err)
		}

		if err := gitRun(repo, "diff", "--cached", "--quiet"); err == nil {
			fmt.Println("Nothing to commit")
			return nil
		}

		msg := saveMessageFlag
		if msg == "" {
			msg = "tuning: update config"
		}
		if err := gitRun(repo, "commit", "-m", msg); err != nil {
			return fmt.Errorf("git commit: %w", err)
		}
		if err := gitRun(repo, "push"); err != nil {
			return fmt.Errorf("git push: %w", err)
		}

		fmt.Println("Saved and pushed to config repo")
		return nil
	},
}

func gitRun(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	return cmd.Run()
}

func init() {
	configCmd.AddCommand(configSaveCmd)
	configSaveCmd.Flags().StringVarP(&saveMessageFlag, "message", "m", "",
		"commit message (default: \"tuning: update config\")")
}
