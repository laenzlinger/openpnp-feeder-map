// SPDX-License-Identifier: GPL-3.0-or-later

package openpnp

import (
	"testing"
)

func TestParseBoard(t *testing.T) {
	b, err := ParseBoard(testdataPath("board.xml"))
	if err != nil {
		t.Fatalf("ParseBoard: %v", err)
	}
	if b.Name != "test-board" {
		t.Errorf("Name = %q, want test-board", b.Name)
	}
	if b.Width != 50.0 || b.Height != 50.0 {
		t.Errorf("Dimensions = (%.1f, %.1f), want (50.0, 50.0)", b.Width, b.Height)
	}
	if got := len(b.Placements); got != 7 {
		t.Fatalf("expected 7 placements, got %d", got)
	}
	// Check enabled placement
	if p := b.Placements[0]; p.ID != "R1" || p.PartID != "R_0805-10K" || !p.Enabled {
		t.Errorf("placement[0] = %+v", p)
	}
	// Check disabled placement
	if p := b.Placements[4]; p.Enabled {
		t.Errorf("placement[4] (J1) should be disabled")
	}
	// Check fiducial
	if p := b.Placements[5]; p.Type != "Fiducial" {
		t.Errorf("placement[5].Type = %q, want Fiducial", p.Type)
	}
}

func TestLoadJobParts_NewFormat(t *testing.T) {
	parts, boards, err := LoadJobParts(testdataPath("job-new.xml"))
	if err != nil {
		t.Fatalf("LoadJobParts (new): %v", err)
	}
	assertParts(t, parts)
	assertBoards(t, boards)
}

func TestLoadJobParts_OldFormat(t *testing.T) {
	parts, _, err := LoadJobParts(testdataPath("job-old.xml"))
	if err != nil {
		t.Fatalf("LoadJobParts (old): %v", err)
	}
	assertParts(t, parts)
}

func assertParts(t *testing.T, parts map[string]int) {
	t.Helper()
	// R_0805-10K: R1 + R2 = 2
	if got := parts["R_0805-10K"]; got != 2 {
		t.Errorf("R_0805-10K count = %d, want 2", got)
	}
	// C_0805-100n: C1 = 1
	if got := parts["C_0805-100n"]; got != 1 {
		t.Errorf("C_0805-100n count = %d, want 1", got)
	}
	// QFN-48: U1 = 1
	if got := parts["QFN-48"]; got != 1 {
		t.Errorf("QFN-48 count = %d, want 1", got)
	}
	// LED-0805: D1 = 1
	if got := parts["LED-0805"]; got != 1 {
		t.Errorf("LED-0805 count = %d, want 1", got)
	}
	// Disabled (USB-C) and fiducials should not appear
	if _, ok := parts["USB-C"]; ok {
		t.Error("disabled placement USB-C should not be in parts")
	}
	if _, ok := parts["Fiducial"]; ok {
		t.Error("fiducial should not be in parts")
	}
	// Total: 4 unique parts
	if got := len(parts); got != 4 {
		t.Errorf("expected 4 unique parts, got %d", got)
	}
}

func assertBoards(t *testing.T, boards []BoardEntry) {
	t.Helper()
	if got := len(boards); got != 1 {
		t.Fatalf("expected 1 board, got %d", got)
	}
	b := boards[0]
	if b.Name != "test-board" {
		t.Errorf("board name = %q, want test-board", b.Name)
	}
	if b.Width != 50.0 || b.Height != 50.0 {
		t.Errorf("board dimensions = (%.1f, %.1f), want (50.0, 50.0)", b.Width, b.Height)
	}
}