// SPDX-License-Identifier: GPL-3.0-or-later

package generate

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
)

// Placement holds one component from the KiCad position export.
type Placement struct {
	Ref     string
	Val     string
	KiCadFP string // original KiCad footprint
	Package string // remapped OpenPnP package
	PartID  string // {KiCadFP}-{Val} for part matching
	X, Y    string
	Rot     string
	Side    string
	Type    string // "Placement" or "Fiducial"
	Enabled bool
}

// PlacementStats summarizes the placements.
type PlacementStats struct {
	Total, Active, Fiducials, Disabled int
}

// LoadPackageMap reads a CSV mapping KiCad footprints to OpenPnP package names.
func LoadPackageMap(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	m := make(map[string]string)
	reader := csv.NewReader(f)
	reader.Comment = '#'
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(row) >= 2 && row[0] != "kicad_footprint" {
			m[row[0]] = row[1]
		}
	}
	return m, nil
}

// ParseKiCadCSV reads a KiCad position CSV and returns placements with remapped packages.
func ParseKiCadCSV(r io.Reader, pkgMap map[string]string) ([]Placement, error) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1 // allow variable field count
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("reading CSV: %w", err)
	}

	var placements []Placement
	for i, row := range records {
		if i == 0 {
			continue // skip header
		}
		if len(row) < 7 {
			continue
		}
		ref := strings.TrimSpace(row[0])
		val := strings.TrimSpace(row[1])
		kicadFP := strings.TrimSpace(row[2])
		if ref == "" || kicadFP == "" {
			continue
		}

		pkg := kicadFP
		if mapped, ok := pkgMap[kicadFP]; ok {
			pkg = mapped
		}

		ptype := "Placement"
		enabled := true
		if strings.Contains(strings.ToLower(kicadFP), "fiducial") {
			ptype = "Fiducial"
		}

		placements = append(placements, Placement{
			Ref:     ref,
			Val:     val,
			KiCadFP: kicadFP,
			Package: pkg,
			PartID:  fmt.Sprintf("%s-%s", pkg, val),
			X:       strings.TrimSpace(row[3]),
			Y:       strings.TrimSpace(row[4]),
			Rot:     strings.TrimSpace(row[5]),
			Side:    strings.TrimSpace(row[6]),
			Type:    ptype,
			Enabled: enabled,
		})
	}
	return placements, nil
}

// WriteBoardXML writes an OpenPnP board XML file.
func WriteBoardXML(placements []Placement, path, name string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	fmt.Fprintf(f, "<openpnp-board version=\"1.1\" name=\"%s\">\n", name)
	fmt.Fprintf(f, "   <dimensions units=\"Millimeters\" x=\"0.0\" y=\"0.0\" z=\"1.6\" rotation=\"0.0\"/>\n")
	fmt.Fprintf(f, "   <placements>\n")
	for _, p := range placements {
		side := strings.ToUpper(p.Side[:1]) + strings.ToLower(p.Side[1:])
		fmt.Fprintf(f, "      <placement version=\"1.4\" side=\"%s\" id=\"%s\" part-id=\"%s\" type=\"%s\" enabled=\"%t\">\n",
			side, p.Ref, p.PartID, p.Type, p.Enabled)
		fmt.Fprintf(f, "         <location units=\"Millimeters\" x=\"%s\" y=\"%s\" z=\"0.0\" rotation=\"%s\"/>\n",
			p.X, p.Y, p.Rot)
		fmt.Fprintf(f, "      </placement>\n")
	}
	fmt.Fprintf(f, "   </placements>\n")
	fmt.Fprintf(f, "   <fiducials/>\n")
	fmt.Fprintf(f, "   <solder-paste-pads/>\n")
	fmt.Fprintf(f, "</openpnp-board>\n")
	return nil
}

// WritePosFile writes a KiCad-compatible ASCII position file with remapped packages.
func WritePosFile(placements []Placement, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	fmt.Fprintf(f, "### Footprint positions ###\n")
	fmt.Fprintf(f, "## Unit = mm, Angle = deg.\n")
	fmt.Fprintf(f, "## Side : All\n")
	fmt.Fprintf(f, "# Ref     Val                          Package                                    PosX       PosY       Rot  Side\n")
	for _, p := range placements {
		fmt.Fprintf(f, "%-10s%-29s%-43s%10s%11s%10s  %s\n",
			p.Ref, p.Val, p.Package, p.X, p.Y, p.Rot, p.Side)
	}
	fmt.Fprintf(f, "## End\n")
	return nil
}

// Stats returns placement statistics.
func Stats(placements []Placement) PlacementStats {
	var s PlacementStats
	for _, p := range placements {
		s.Total++
		switch {
		case p.Type == "Fiducial":
			s.Fiducials++
		case p.Enabled:
			s.Active++
		default:
			s.Disabled++
		}
	}
	return s
}
