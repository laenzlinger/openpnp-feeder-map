// SPDX-License-Identifier: GPL-3.0-or-later

package feedermap

import (
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"sort"
	"strings"

	"github.com/laenzlinger/openpnp-tools/internal/generate"
	"github.com/laenzlinger/openpnp-tools/internal/openpnp"
)

// LoadIPNMapFromBOM reads a KiCad BOM CSV (with Value, Footprint, IPN columns)
// and a package map to build a mapping from OpenPnP PartID to IPN.
// The BOM is exported with: kicad-cli sch export bom --fields "Value,Footprint,IPN" --group-by "Value,Footprint,IPN"
func LoadIPNMapFromBOM(r io.Reader, pkgMap *generate.PackageMap) (map[string]string, error) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("reading BOM: %w", err)
	}
	if len(records) == 0 {
		return nil, nil
	}

	// Find column indices from header
	header := records[0]
	col := make(map[string]int)
	for i, h := range header {
		col[strings.TrimSpace(h)] = i
	}
	valIdx, valOK := col["Value"]
	fpIdx, fpOK := col["Footprint"]
	ipnIdx, ipnOK := col["IPN"]
	if !valOK || !fpOK || !ipnOK {
		return nil, fmt.Errorf("BOM CSV must have Value, Footprint, and IPN columns")
	}

	m := make(map[string]string)
	for _, row := range records[1:] {
		if len(row) <= ipnIdx {
			continue
		}
		value := strings.TrimSpace(row[valIdx])
		fpFull := strings.TrimSpace(row[fpIdx])
		ipn := strings.TrimSpace(row[ipnIdx])
		if ipn == "" || value == "" {
			continue
		}
		// Strip library prefix (e.g. "Capacitor_SMD:C_0805_2012Metric" → "C_0805_2012Metric")
		fp := fpFull
		if i := strings.LastIndex(fp, ":"); i >= 0 {
			fp = fp[i+1:]
		}
		pkg := pkgMap.PackageName(fp)
		partID := fmt.Sprintf("%s-%s", pkg, value)
		m[partID] = ipn
	}
	return m, nil
}

// FeederEntry is one row in the feeder map output.
type FeederEntry struct {
	Feeder   openpnp.Feeder
	IPN      string  // InvenTree IPN (empty if not mapped)
	Count    int     // placements in this job
	Capacity int     // max parts on strip (strip_length / part_pitch)
	StartX   float64 // reference hole / pick location X
	StartY   float64 // reference hole / pick location Y
	EndX     float64 // last hole X (strip feeders)
	EndY     float64 // last hole Y (strip feeders)
	HasEnd   bool    // true if strip/pushpull with two points
}

// MapData holds everything the HTML template needs.
type MapData struct {
	JobFile          string
	Feeders          []FeederEntry // feeders needed for this job
	MissingParts     []MissingPart // parts with no feeder
	UnusedFeeders    []FeederEntry // enabled feeders not needed by this job
	Boards           []openpnp.BoardEntry
	BedXMin, BedXMax float64
	BedYMin, BedYMax float64
	HasBed           bool
	HasIPN           bool    // true if any feeder has an IPN
	StripLength      float64 // default strip feeder slot length in mm
}

// MissingPart is a job part that has no matching feeder.
type MissingPart struct {
	PartID string
	IPN    string
	Count  int
}

// pushPullEnd computes the tape end point from pick location, rotation, and strip length.
// Push-pull feeders define feed direction via location.rotation (0°=+X, 90°=+Y).
func pushPullEnd(f *openpnp.Feeder, length float64) (float64, float64) {
	if length <= 0 {
		length = 120 // default
	}
	rad := f.Location.Rotation * math.Pi / 180
	return f.PickX() - math.Cos(rad)*length, f.PickY() - math.Sin(rad)*length
}

// calcCapacity returns the max number of parts a strip feeder can hold.
// Subtracts 2 for cut waste at both ends of the strip.
func calcCapacity(stripLength float64, f *openpnp.Feeder) int {
	if stripLength <= 0 || f.PartPitch == nil || f.PartPitch.Value <= 0 {
		return 0
	}
	cap := int(stripLength/f.PartPitch.Value) - 2
	if cap < 0 {
		return 0
	}
	return cap
}

func collectUnused(machine *openpnp.Machine, usedParts map[string]bool, stripLength float64) []FeederEntry {
	var unused []FeederEntry
	for i := range machine.Feeders {
		f := &machine.Feeders[i]
		if !f.Enabled || f.PartID == "" || usedParts[f.PartID] {
			continue
		}
		if f.PickX() == 0 && f.PickY() == 0 {
			continue
		}
		entry := FeederEntry{
			Feeder: *f,
			StartX: f.PickX(),
			StartY: f.PickY(),
		}
		if f.LastHoleLocation != nil {
			entry.EndX = f.LastHoleLocation.X
			entry.EndY = f.LastHoleLocation.Y
			entry.HasEnd = true
		} else if f.FeederType() == "PushPull" {
			entry.EndX, entry.EndY = pushPullEnd(f, stripLength)
			entry.HasEnd = true
		}
		entry.Capacity = calcCapacity(stripLength, f)
		unused = append(unused, entry)
	}
	return unused
}

func Build(jobParts map[string]int, boards []openpnp.BoardEntry, machine *openpnp.Machine,
	stripLength float64, ipnMap map[string]string,
) *MapData {
	// Index feeders by part-id (only enabled feeders with a position).
	feederByPart := make(map[string]*openpnp.Feeder)
	for i := range machine.Feeders {
		f := &machine.Feeders[i]
		if !f.Enabled || f.PartID == "" {
			continue
		}
		if f.PickX() == 0 && f.PickY() == 0 {
			continue
		}
		feederByPart[f.PartID] = f
	}

	if ipnMap == nil {
		ipnMap = make(map[string]string)
	}

	usedParts := make(map[string]bool)
	var feeders []FeederEntry
	var missing []MissingPart

	for partID, count := range jobParts {
		f, ok := feederByPart[partID]
		if !ok {
			missing = append(missing, MissingPart{PartID: partID, IPN: ipnMap[partID], Count: count})
			continue
		}
		usedParts[partID] = true
		entry := FeederEntry{
			Feeder: *f,
			IPN:    ipnMap[partID],
			Count:  count,
			StartX: f.PickX(),
			StartY: f.PickY(),
		}
		if f.LastHoleLocation != nil {
			entry.EndX = f.LastHoleLocation.X
			entry.EndY = f.LastHoleLocation.Y
			entry.HasEnd = true
		} else if f.FeederType() == "PushPull" {
			// Push-pull: compute end from pick location + rotation + strip length
			entry.EndX, entry.EndY = pushPullEnd(f, stripLength)
			entry.HasEnd = true
		}
		entry.Capacity = calcCapacity(stripLength, f)
		feeders = append(feeders, entry)
	}

	// Collect enabled feeders not used by this job.
	unused := collectUnused(machine, usedParts, stripLength)

	sort.Slice(feeders, func(i, j int) bool { return feeders[i].Feeder.Name < feeders[j].Feeder.Name })
	sort.Slice(missing, func(i, j int) bool { return missing[i].PartID < missing[j].PartID })
	sort.Slice(unused, func(i, j int) bool { return unused[i].Feeder.Name < unused[j].Feeder.Name })

	return &MapData{
		Feeders:       feeders,
		MissingParts:  missing,
		UnusedFeeders: unused,
		Boards:        boards,
		BedXMin:       machine.BedXMin,
		BedXMax:       machine.BedXMax,
		BedYMin:       machine.BedYMin,
		BedYMax:       machine.BedYMax,
		HasBed:        machine.HasBed,
		HasIPN:        len(ipnMap) > 0,
	}
}
