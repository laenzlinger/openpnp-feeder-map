// SPDX-License-Identifier: GPL-3.0-or-later

package generate

import (
	"strings"
	"testing"
)

func TestParseKiCadCSV(t *testing.T) {
	csv := `Ref,Val,Package,PosX,PosY,Rot,Side
C1,100n,C_0805_2012Metric,10.0,-20.0,90.0,top
FID1,Fiducial,Fiducial_1mm_Mask2mm,5.0,-5.0,0.0,top
U1,ASM1061,QFN50P700X700X90-49N-D,30.0,-40.0,-45.0,top`

	pkgMap := map[string]string{
		"C_0805_2012Metric":       "C_0805",
		"QFN50P700X700X90-49N-D":  "QFN-48-7x7",
		"Fiducial_1mm_Mask2mm":    "FIDUCIAL-1X2",
	}

	placements, err := ParseKiCadCSV(strings.NewReader(csv), pkgMap)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(placements) != 3 {
		t.Fatalf("expected 3 placements, got %d", len(placements))
	}

	// C1: remapped package, normal placement
	if placements[0].Package != "C_0805" {
		t.Errorf("C1 package: got %q, want %q", placements[0].Package, "C_0805")
	}
	if placements[0].PartID != "C_0805-100n" {
		t.Errorf("C1 part-id: got %q, want %q", placements[0].PartID, "C_0805-100n")
	}
	if placements[0].Type != "Placement" {
		t.Errorf("C1 type: got %q, want %q", placements[0].Type, "Placement")
	}

	// FID1: auto-detected fiducial
	if placements[1].Type != "Fiducial" {
		t.Errorf("FID1 type: got %q, want %q", placements[1].Type, "Fiducial")
	}

	// U1: remapped QFN
	if placements[2].Package != "QFN-48-7x7" {
		t.Errorf("U1 package: got %q, want %q", placements[2].Package, "QFN-48-7x7")
	}
	if placements[2].PartID != "QFN-48-7x7-ASM1061" {
		t.Errorf("U1 part-id: got %q, want %q", placements[2].PartID, "QFN-48-7x7-ASM1061")
	}
}

func TestParseKiCadCSV_UnmappedFootprint(t *testing.T) {
	csv := `Ref,Val,Package,PosX,PosY,Rot,Side
J1,USB_C,SomeUnknownFootprint,10.0,-20.0,0.0,top`

	placements, err := ParseKiCadCSV(strings.NewReader(csv), map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if placements[0].Package != "SomeUnknownFootprint" {
		t.Errorf("unmapped package: got %q, want %q", placements[0].Package, "SomeUnknownFootprint")
	}
}

func TestParseKiCadCSV_TrailingJunk(t *testing.T) {
	csv := `Ref,Val,Package,PosX,PosY,Rot,Side
C1,100n,C_0805,10.0,-20.0,90.0,top
Wrote position data,,,,,,`

	placements, err := ParseKiCadCSV(strings.NewReader(csv), map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(placements) != 1 {
		t.Fatalf("expected 1 placement, got %d", len(placements))
	}
}

func TestStats(t *testing.T) {
	placements := []Placement{
		{Type: "Placement", Enabled: true},
		{Type: "Placement", Enabled: true},
		{Type: "Fiducial", Enabled: true},
		{Type: "Placement", Enabled: false},
	}

	s := Stats(placements)
	if s.Total != 4 {
		t.Errorf("total: got %d, want 4", s.Total)
	}
	if s.Active != 2 {
		t.Errorf("active: got %d, want 2", s.Active)
	}
	if s.Fiducials != 1 {
		t.Errorf("fiducials: got %d, want 1", s.Fiducials)
	}
	if s.Disabled != 1 {
		t.Errorf("disabled: got %d, want 1", s.Disabled)
	}
}
