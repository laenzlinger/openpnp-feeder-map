// SPDX-License-Identifier: GPL-3.0-or-later

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseSettings(t *testing.T) {
	input := "# comment\nrepo-dir: /home/user/openpnp-config\n"
	s, err := parseSettings(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parseSettings: %v", err)
	}
	if s.RepoDir != "/home/user/openpnp-config" {
		t.Errorf("RepoDir = %q, want /home/user/openpnp-config", s.RepoDir)
	}
}

func TestParseSettingsEmpty(t *testing.T) {
	s, err := parseSettings(strings.NewReader(""))
	if err != nil {
		t.Fatalf("parseSettings: %v", err)
	}
	if s.RepoDir != "" {
		t.Errorf("RepoDir = %q, want empty", s.RepoDir)
	}
}

func TestLoadSettingsMissing(t *testing.T) {
	s, err := LoadSettings(t.TempDir())
	if err != nil {
		t.Fatalf("LoadSettings: %v", err)
	}
	if s.RepoDir != "" {
		t.Errorf("RepoDir = %q, want empty", s.RepoDir)
	}
}

func TestLoadSettingsFromFile(t *testing.T) {
	dir := t.TempDir()
	content := "repo-dir: /tmp/my-config\n"
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := LoadSettings(dir)
	if err != nil {
		t.Fatalf("LoadSettings: %v", err)
	}
	if s.RepoDir != "/tmp/my-config" {
		t.Errorf("RepoDir = %q, want /tmp/my-config", s.RepoDir)
	}
}

func setupTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, f := range ManagedFiles {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("<"+f+"/>"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestBackup(t *testing.T) {
	dir := setupTestDir(t)

	backupDir, err := Backup(dir)
	if err != nil {
		t.Fatalf("Backup: %v", err)
	}

	for _, f := range ManagedFiles {
		if _, err := os.Stat(filepath.Join(backupDir, f)); err != nil {
			t.Errorf("missing backup file %s: %v", f, err)
		}
	}
}

func TestBackupIncludesExtraXMLs(t *testing.T) {
	dir := setupTestDir(t)
	// Add boards.xml and panels.xml like OpenPnP does
	for _, f := range []string{"boards.xml", "panels.xml"} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("<"+f+"/>"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	backupDir, err := Backup(dir)
	if err != nil {
		t.Fatalf("Backup: %v", err)
	}

	for _, f := range backupFiles {
		if _, err := os.Stat(filepath.Join(backupDir, f)); err != nil {
			t.Errorf("missing backup file %s: %v", f, err)
		}
	}
}

func TestBackupNoFiles(t *testing.T) {
	dir := t.TempDir()
	_, err := Backup(dir)
	if err == nil {
		t.Fatal("expected error for empty dir")
	}
}

func TestCopyManagedFiles(t *testing.T) {
	src := setupTestDir(t)
	dst := t.TempDir()

	copied, err := CopyManagedFiles(src, dst)
	if err != nil {
		t.Fatalf("CopyManagedFiles: %v", err)
	}
	if len(copied) != len(ManagedFiles) {
		t.Errorf("copied %d files, want %d", len(copied), len(ManagedFiles))
	}
	for _, f := range ManagedFiles {
		data, err := os.ReadFile(filepath.Join(dst, f))
		if err != nil {
			t.Errorf("missing %s in dst: %v", f, err)
			continue
		}
		if string(data) != "<"+f+"/>" {
			t.Errorf("%s content mismatch", f)
		}
	}
}

func TestCompareFiles(t *testing.T) {
	live := setupTestDir(t)
	repo := setupTestDir(t)

	// Identical
	results, err := CompareFiles(live, repo)
	if err != nil {
		t.Fatalf("CompareFiles: %v", err)
	}
	for _, r := range results {
		if r.Status != "unchanged" {
			t.Errorf("%s: got %s, want unchanged", r.Name, r.Status)
		}
	}

	// Modify one file in live
	if err := os.WriteFile(filepath.Join(live, "machine.xml"), []byte("<changed/>"), 0o600); err != nil {
		t.Fatal(err)
	}
	results, _ = CompareFiles(live, repo)
	for _, r := range results {
		if r.Name == "machine.xml" && r.Status != "modified" {
			t.Errorf("machine.xml: got %s, want modified", r.Status)
		}
	}

	// Remove from live
	if err := os.Remove(filepath.Join(live, "parts.xml")); err != nil {
		t.Fatal(err)
	}
	results, _ = CompareFiles(live, repo)
	for _, r := range results {
		if r.Name == "parts.xml" && r.Status != "missing-live" {
			t.Errorf("parts.xml: got %s, want missing-live", r.Status)
		}
	}
}
