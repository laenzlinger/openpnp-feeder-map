// SPDX-License-Identifier: GPL-3.0-or-later

package feedermap

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/laenzlinger/openpnp-feeder-map/internal/openpnp"
)

func testdataPath(name string) string {
	_, f, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(f), "..", "openpnp", "testdata", name)
}

func TestBuild(t *testing.T) {
	machine, err := openpnp.ParseMachine(testdataPath("machine.xml"))
	if err != nil {
		t.Fatalf("ParseMachine: %v", err)
	}
	parts, err := openpnp.LoadJobParts(testdataPath("job-new.xml"))
	if err != nil {
		t.Fatalf("LoadJobParts: %v", err)
	}

	data := Build(parts, machine)

	// Job feeders: R_0805-10K (LV8-01), C_0805-100n (LV8-02), QFN-48 (TRAY-01) = 3
	if got := len(data.Feeders); got != 3 {
		t.Errorf("expected 3 job feeders, got %d", got)
	}

	// Check counts
	feederByName := map[string]FeederEntry{}
	for _, f := range data.Feeders {
		feederByName[f.Feeder.Name] = f
	}
	if f, ok := feederByName["LV8-01"]; !ok {
		t.Error("LV8-01 not in job feeders")
	} else if f.Count != 2 {
		t.Errorf("LV8-01 count = %d, want 2", f.Count)
	}
	if f, ok := feederByName["LV8-02"]; !ok {
		t.Error("LV8-02 not in job feeders")
	} else if f.Count != 1 {
		t.Errorf("LV8-02 count = %d, want 1", f.Count)
	}

	// Strip feeder should have end point
	if !feederByName["LV8-01"].HasEnd {
		t.Error("LV8-01 should have end point")
	}

	// Missing parts: LED-0805 (no feeder)
	if got := len(data.MissingParts); got != 1 {
		t.Errorf("expected 1 missing part, got %d", got)
	}
	if data.MissingParts[0].PartID != "LED-0805" {
		t.Errorf("missing part = %q, want LED-0805", data.MissingParts[0].PartID)
	}

	// Unused feeders: PP-01 (SOT-23-NPN not in job), disabled LV8-03 excluded
	if got := len(data.UnusedFeeders); got != 1 {
		t.Errorf("expected 1 unused feeder, got %d", got)
	}
	if data.UnusedFeeders[0].Feeder.Name != "PP-01" {
		t.Errorf("unused feeder = %q, want PP-01", data.UnusedFeeders[0].Feeder.Name)
	}

	// Bed dimensions passed through
	if !data.HasBed || data.BedXMax != 400.0 || data.BedYMax != 300.0 {
		t.Errorf("bed = HasBed:%v (%.0f, %.0f), want true (400, 300)", data.HasBed, data.BedXMax, data.BedYMax)
	}
}
