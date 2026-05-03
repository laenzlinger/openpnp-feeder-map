// SPDX-License-Identifier: GPL-3.0-or-later

package config

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// ManagedFiles are the config files we copy during pull/apply.
var ManagedFiles = []string{
	"machine.xml",
	"parts.xml",
	"packages.xml",
	"vision-settings.xml",
}

// backupFiles are all XMLs OpenPnP backs up (superset of ManagedFiles).
var backupFiles = []string{
	"machine.xml",
	"parts.xml",
	"packages.xml",
	"vision-settings.xml",
	"boards.xml",
	"panels.xml",
}

const (
	backupSubdir = "backups"
	// OpenPnP's timestamp format: 2024-10-18_18.15.12
	timestampFmt = "2006-01-02_15.04.05"
	// FileName is the config file name looked up in ~/.openpnp2/.
	FileName = ".openpnp-tools.yaml"
)

// Settings holds values loaded from the config file.
type Settings struct {
	RepoDir string // repo-dir: path to the openpnp-config repo
}

// LoadSettings reads the config file from configDir/FileName.
// Returns empty Settings (no error) if the file doesn't exist.
func LoadSettings(configDir string) (Settings, error) {
	path := filepath.Join(configDir, FileName)
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return Settings{}, nil
	}
	if err != nil {
		return Settings{}, err
	}
	defer func() { _ = f.Close() }()
	return parseSettings(f)
}

func parseSettings(r io.Reader) (Settings, error) {
	var s Settings
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || line[0] == '#' {
			continue
		}
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		if strings.TrimSpace(key) == "repo-dir" {
			s.RepoDir = strings.TrimSpace(val)
		}
	}
	return s, scanner.Err()
}

// DefaultConfigDir returns ~/.openpnp2.
func DefaultConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".openpnp2")
}

// IsOpenPnPRunning checks whether an OpenPnP process is running.
func IsOpenPnPRunning() bool {
	if runtime.GOOS == "windows" {
		out, err := exec.Command("tasklist", "/FI", "IMAGENAME eq OpenPnP*").Output()
		return err == nil && strings.Contains(string(out), "OpenPnP")
	}
	// Match the launcher process name or the Java jar, but not unrelated
	// tools that happen to have "openpnp" in their arguments (e.g. kiro-cli --agent openpnp).
	if exec.Command("pgrep", "-x", "OpenPnP").Run() == nil {
		return true
	}
	return exec.Command("pgrep", "-f", "openpnp-gui").Run() == nil
}

// CheckNotRunning returns an error if OpenPnP is running.
func CheckNotRunning() error {
	if IsOpenPnPRunning() {
		return fmt.Errorf("OpenPnP is running — close it first")
	}
	return nil
}

// Backup copies all XML config files from configDir into configDir/backups/<timestamp>.
// Uses the same directory and timestamp format as OpenPnP itself.
// Returns the backup directory path.
func Backup(configDir string) (string, error) {
	stamp := time.Now().Format(timestampFmt)
	dir := filepath.Join(configDir, backupSubdir, stamp)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating backup dir: %w", err)
	}

	copied := 0
	for _, f := range backupFiles {
		src := filepath.Join(configDir, f)
		if _, err := os.Stat(src); os.IsNotExist(err) {
			continue
		}
		if err := copyFile(src, filepath.Join(dir, f)); err != nil {
			return "", fmt.Errorf("backing up %s: %w", f, err)
		}
		copied++
	}
	if copied == 0 {
		_ = os.Remove(dir)
		return "", fmt.Errorf("no config files found in %s", configDir)
	}
	return dir, nil
}

// CopyManagedFiles copies managed files from src to dst directory.
// Returns the list of files actually copied.
func CopyManagedFiles(srcDir, dstDir string) ([]string, error) {
	var copied []string
	for _, f := range ManagedFiles {
		src := filepath.Join(srcDir, f)
		if _, err := os.Stat(src); os.IsNotExist(err) {
			continue
		}
		if err := copyFile(src, filepath.Join(dstDir, f)); err != nil {
			return copied, fmt.Errorf("copying %s: %w", f, err)
		}
		copied = append(copied, f)
	}
	return copied, nil
}

// FileStatus describes the drift state of a single managed file.
type FileStatus struct {
	Name   string
	Status string // "unchanged", "modified", "missing-live", "missing-repo"
}

// CompareFiles compares managed files between configDir (live) and repoDir.
func CompareFiles(configDir, repoDir string) ([]FileStatus, error) {
	var results []FileStatus
	for _, f := range ManagedFiles {
		live, liveErr := os.ReadFile(filepath.Join(configDir, f))
		repo, repoErr := os.ReadFile(filepath.Join(repoDir, f))

		var status string
		switch {
		case os.IsNotExist(liveErr) && os.IsNotExist(repoErr):
			continue
		case os.IsNotExist(liveErr):
			status = "missing-live"
		case os.IsNotExist(repoErr):
			status = "missing-repo"
		case liveErr != nil:
			return nil, liveErr
		case repoErr != nil:
			return nil, repoErr
		case string(live) == string(repo):
			status = "unchanged"
		default:
			status = "modified"
		}
		results = append(results, FileStatus{Name: f, Status: status})
	}
	return results, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	_, err = io.Copy(out, in)
	return err
}
