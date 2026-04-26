// SPDX-License-Identifier: GPL-3.0-or-later

package openpnp

import (
	"encoding/xml"
	"fmt"
	"os"
)

type Machine struct {
	Feeders          []Feeder
	BedXMin, BedXMax float64
	BedYMin, BedYMax float64
	HasBed           bool
}

type Feeder struct {
	Class   string
	ID      string
	Name    string
	Enabled bool
	PartID  string

	// Location (rotation in tape for strip feeders)
	Location Location

	// ReferenceStripFeeder fields
	ReferenceHoleLocation *Location
	LastHoleLocation      *Location
	PartPitch             *Length
	TapeWidth             *Length
	TapeType              string

	// ReferencePushPullFeeder fields
	Hole1Location *Location
	Hole2Location *Location

	// ReferenceTrayFeeder fields
	TrayCountX int
	TrayCountY int
}

type Location struct {
	Units    string  `xml:"units,attr"`
	X        float64 `xml:"x,attr"`
	Y        float64 `xml:"y,attr"`
	Z        float64 `xml:"z,attr"`
	Rotation float64 `xml:"rotation,attr"`
}

type Length struct {
	Value float64 `xml:"value,attr"`
	Units string  `xml:"units,attr"`
}

// FeederType returns a short human-readable type name.
func (f *Feeder) FeederType() string {
	switch f.Class {
	case "org.openpnp.machine.reference.feeder.ReferenceStripFeeder":
		return "Strip"
	case "org.openpnp.machine.reference.feeder.ReferencePushPullFeeder":
		return "PushPull"
	case "org.openpnp.machine.reference.feeder.ReferenceTrayFeeder":
		return "Tray"
	default:
		return f.Class
	}
}

// PickX returns the primary X coordinate for map display.
func (f *Feeder) PickX() float64 {
	if f.ReferenceHoleLocation != nil {
		return f.ReferenceHoleLocation.X
	}
	if f.Hole1Location != nil {
		return f.Hole1Location.X
	}
	return f.Location.X
}

// PickY returns the primary Y coordinate for map display.
func (f *Feeder) PickY() float64 {
	if f.ReferenceHoleLocation != nil {
		return f.ReferenceHoleLocation.Y
	}
	if f.Hole1Location != nil {
		return f.Hole1Location.Y
	}
	return f.Location.Y
}

// xmlMachine mirrors the top-level machine.xml structure enough to extract feeders.
type xmlMachine struct {
	XMLName xml.Name        `xml:"openpnp-machine"`
	Machine xmlMachineInner `xml:"machine"`
}

type xmlMachineInner struct {
	Feeders xmlFeeders `xml:"feeders"`
	Axes    xmlAxes    `xml:"axes"`
}

type xmlAxes struct {
	Axes []xmlAxis `xml:"axis"`
}

type xmlAxis struct {
	Name                 string  `xml:"name,attr"`
	Type                 string  `xml:"type,attr"`
	SoftLimitLowEnabled  bool    `xml:"soft-limit-low-enabled,attr"`
	SoftLimitHighEnabled bool    `xml:"soft-limit-high-enabled,attr"`
	SoftLimitLow         *Length `xml:"soft-limit-low"`
	SoftLimitHigh        *Length `xml:"soft-limit-high"`
}

type xmlFeeders struct {
	Feeders []xmlFeeder `xml:"feeder"`
}

type xmlFeeder struct {
	Class      string `xml:"class,attr"`
	ID         string `xml:"id,attr"`
	Name       string `xml:"name,attr"`
	Enabled    bool   `xml:"enabled,attr"`
	PartID     string `xml:"part-id,attr"`
	TapeType   string `xml:"tape-type,attr"`
	TrayCountX int    `xml:"tray-count-x,attr"`
	TrayCountY int    `xml:"tray-count-y,attr"`

	Location              Location  `xml:"location"`
	ReferenceHoleLocation *Location `xml:"reference-hole-location"`
	LastHoleLocation      *Location `xml:"last-hole-location"`
	Hole1Location         *Location `xml:"hole-1-location"`
	Hole2Location         *Location `xml:"hole-2-location"`
	PartPitch             *Length   `xml:"part-pitch"`
	TapeWidth             *Length   `xml:"tape-width"`
}

func ParseMachine(path string) (*Machine, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading machine.xml: %w", err)
	}
	var m xmlMachine
	if err := xml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing machine.xml: %w", err)
	}
	machine := &Machine{}
	for _, xf := range m.Machine.Feeders.Feeders {
		f := Feeder{
			Class:                 xf.Class,
			ID:                    xf.ID,
			Name:                  xf.Name,
			Enabled:               xf.Enabled,
			PartID:                xf.PartID,
			Location:              xf.Location,
			ReferenceHoleLocation: xf.ReferenceHoleLocation,
			LastHoleLocation:      xf.LastHoleLocation,
			Hole1Location:         xf.Hole1Location,
			Hole2Location:         xf.Hole2Location,
			PartPitch:             xf.PartPitch,
			TapeWidth:             xf.TapeWidth,
			TapeType:              xf.TapeType,
			TrayCountX:            xf.TrayCountX,
			TrayCountY:            xf.TrayCountY,
		}
		machine.Feeders = append(machine.Feeders, f)
	}
	for _, ax := range m.Machine.Axes.Axes {
		if ax.SoftLimitLowEnabled && ax.SoftLimitHighEnabled && ax.SoftLimitLow != nil && ax.SoftLimitHigh != nil {
			switch ax.Type {
			case "X":
				machine.BedXMin = ax.SoftLimitLow.Value
				machine.BedXMax = ax.SoftLimitHigh.Value
				machine.HasBed = true
			case "Y":
				machine.BedYMin = ax.SoftLimitLow.Value
				machine.BedYMax = ax.SoftLimitHigh.Value
				machine.HasBed = true
			}
		}
	}
	return machine, nil
}
