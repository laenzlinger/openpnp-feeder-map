// SPDX-License-Identifier: GPL-3.0-or-later

package openpnp

import (
	"path/filepath"
	"runtime"
	"testing"
)

func testdataPath(name string) string {
	_, f, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(f), "testdata", name)
}

func TestParseMachine(t *testing.T) {
	m, err := ParseMachine(testdataPath("machine.xml"))
	if err != nil {
		t.Fatalf("ParseMachine: %v", err)
	}

	if got := len(m.Feeders); got != 5 {
		t.Fatalf("expected 5 feeders, got %d", got)
	}

	// Check strip feeder
	f := m.Feeders[0]
	if f.Name != "LV8-01" {
		t.Errorf("feeder[0].Name = %q, want LV8-01", f.Name)
	}
	if f.PartID != "R_0805-10K" {
		t.Errorf("feeder[0].PartID = %q, want R_0805-10K", f.PartID)
	}
	if !f.Enabled {
		t.Error("feeder[0] should be enabled")
	}
	if f.FeederType() != "Strip" {
		t.Errorf("feeder[0].FeederType() = %q, want Strip", f.FeederType())
	}
	if f.ReferenceHoleLocation == nil {
		t.Fatal("feeder[0].ReferenceHoleLocation is nil")
	}
	if f.PickX() != 100.0 || f.PickY() != 200.0 {
		t.Errorf("feeder[0] pick = (%.1f, %.1f), want (100.0, 200.0)", f.PickX(), f.PickY())
	}
	if f.TapeWidth == nil || f.TapeWidth.Value != 8.0 {
		t.Errorf("feeder[0].TapeWidth = %v, want 8.0", f.TapeWidth)
	}
	if f.TapeType != "WhitePaper" {
		t.Errorf("feeder[0].TapeType = %q, want WhitePaper", f.TapeType)
	}

	// Check disabled feeder
	if m.Feeders[2].Enabled {
		t.Error("feeder[2] (disabled) should not be enabled")
	}

	// Check push-pull feeder
	pp := m.Feeders[3]
	if pp.FeederType() != "PushPull" {
		t.Errorf("feeder[3].FeederType() = %q, want PushPull", pp.FeederType())
	}
	if pp.Hole1Location == nil {
		t.Fatal("feeder[3].Hole1Location is nil")
	}
	if pp.PickX() != 196.0 || pp.PickY() != 48.0 {
		t.Errorf("pushpull pick = (%.1f, %.1f), want (196.0, 48.0)", pp.PickX(), pp.PickY())
	}

	// Check tray feeder
	tray := m.Feeders[4]
	if tray.FeederType() != "Tray" {
		t.Errorf("feeder[4].FeederType() = %q, want Tray", tray.FeederType())
	}
	if tray.PickX() != 300.0 || tray.PickY() != 150.0 {
		t.Errorf("tray pick = (%.1f, %.1f), want (300.0, 150.0)", tray.PickX(), tray.PickY())
	}

	// Check bed dimensions
	if !m.HasBed {
		t.Fatal("expected HasBed = true")
	}
	if m.BedXMax != 400.0 || m.BedYMax != 300.0 {
		t.Errorf("bed = (%.0f, %.0f), want (400, 300)", m.BedXMax, m.BedYMax)
	}
}
