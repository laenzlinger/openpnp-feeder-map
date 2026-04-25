// SPDX-License-Identifier: GPL-3.0-or-later

package feedermap

import (
	"sort"

	"github.com/laenzlinger/openpnp-feeder-map/internal/openpnp"
)

// FeederEntry is one row in the feeder map output.
type FeederEntry struct {
	Feeder   openpnp.Feeder
	Count    int     // placements in this job
	StartX   float64 // reference hole / pick location X
	StartY   float64 // reference hole / pick location Y
	EndX     float64 // last hole X (strip feeders)
	EndY     float64 // last hole Y (strip feeders)
	HasEnd   bool    // true if strip/pushpull with two points
}

// MapData holds everything the HTML template needs.
type MapData struct {
	JobFile        string
	Feeders        []FeederEntry // feeders needed for this job
	MissingParts   []MissingPart // parts with no feeder
	UnusedFeeders  []FeederEntry // enabled feeders not needed by this job
	BedXMin, BedXMax float64
	BedYMin, BedYMax float64
	HasBed         bool
}

// MissingPart is a job part that has no matching feeder.
type MissingPart struct {
	PartID string
	Count  int
}

// Build creates the feeder map data by matching job parts to machine feeders.
func Build(jobParts map[string]int, machine *openpnp.Machine) *MapData {
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

	usedParts := make(map[string]bool)
	var feeders []FeederEntry
	var missing []MissingPart

	for partID, count := range jobParts {
		f, ok := feederByPart[partID]
		if !ok {
			missing = append(missing, MissingPart{PartID: partID, Count: count})
			continue
		}
		usedParts[partID] = true
		entry := FeederEntry{
			Feeder: *f,
			Count:  count,
			StartX: f.PickX(),
			StartY: f.PickY(),
		}
		if f.LastHoleLocation != nil {
			entry.EndX = f.LastHoleLocation.X
			entry.EndY = f.LastHoleLocation.Y
			entry.HasEnd = true
		} else if f.Hole2Location != nil {
			entry.EndX = f.Hole2Location.X
			entry.EndY = f.Hole2Location.Y
			entry.HasEnd = true
		}
		feeders = append(feeders, entry)
	}

	// Collect enabled feeders not used by this job.
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
		}
		unused = append(unused, entry)
	}

	sort.Slice(feeders, func(i, j int) bool { return feeders[i].Feeder.Name < feeders[j].Feeder.Name })
	sort.Slice(missing, func(i, j int) bool { return missing[i].PartID < missing[j].PartID })
	sort.Slice(unused, func(i, j int) bool { return unused[i].Feeder.Name < unused[j].Feeder.Name })

	return &MapData{
		Feeders:       feeders,
		MissingParts:  missing,
		UnusedFeeders: unused,
		BedXMin:       machine.BedXMin,
		BedXMax:       machine.BedXMax,
		BedYMin:       machine.BedYMin,
		BedYMax:       machine.BedYMax,
		HasBed:        machine.HasBed,
	}
}
