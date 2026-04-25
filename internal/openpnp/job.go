// SPDX-License-Identifier: GPL-3.0-or-later

package openpnp

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
)

type Job struct {
	Boards []BoardRef
}

type BoardRef struct {
	ID       string
	FileName string
	Side     string
	Enabled  bool
	Location Location
}

type Board struct {
	Name       string
	Width      float64 // from <dimensions> x
	Height     float64 // from <dimensions> y
	Placements []Placement
}

// BoardEntry combines a board's dimensions with its job placement location.
type BoardEntry struct {
	Name     string
	Width    float64
	Height   float64
	Location Location
}

type Placement struct {
	ID      string
	PartID  string
	Side    string
	Type    string // "Placement" or "Fiducial"
	Enabled bool
}

// xmlJob mirrors the OpenPnP job XML (supports both old and new format).
type xmlJob struct {
	XMLName   xml.Name     `xml:"openpnp-job"`
	RootPanel xmlRootPanel `xml:"root-panel"`
	// Old format
	BoardLocations xmlBoardLocations `xml:"board-locations"`
}

type xmlRootPanel struct {
	Children xmlChildren `xml:"children"`
}

type xmlChildren struct {
	Objects []xmlBoardLocation `xml:"object"`
}

type xmlBoardLocations struct {
	Boards []xmlOldBoardLocation `xml:"board-location"`
}

type xmlBoardLocation struct {
	Class    string   `xml:"class,attr"`
	ID       string   `xml:"id,attr"`
	Side     string   `xml:"side,attr"`
	FileName string   `xml:"file-name,attr"`
	Enabled  bool     `xml:"locally-enabled,attr"`
	Location Location `xml:"location"`
}

type xmlOldBoardLocation struct {
	Side      string   `xml:"side,attr"`
	BoardFile string   `xml:"board-file,attr"`
	Enabled   bool     `xml:"enabled,attr"`
	Location  Location `xml:"location"`
}

// xmlBoard mirrors the OpenPnP board XML.
type xmlBoard struct {
	XMLName    xml.Name        `xml:"openpnp-board"`
	Name       string          `xml:"name,attr"`
	Dimensions Location        `xml:"dimensions"`
	Placements xmlPlacements   `xml:"placements"`
}

type xmlPlacements struct {
	Placements []xmlPlacement `xml:"placement"`
}

type xmlPlacement struct {
	ID      string `xml:"id,attr"`
	PartID  string `xml:"part-id,attr"`
	Side    string `xml:"side,attr"`
	Type    string `xml:"type,attr"`
	Enabled bool   `xml:"enabled,attr"`
}

func ParseJob(path string) (*Job, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading job file: %w", err)
	}
	var xj xmlJob
	if err := xml.Unmarshal(data, &xj); err != nil {
		return nil, fmt.Errorf("parsing job file: %w", err)
	}
	job := &Job{}
	// New format (root-panel > children > object)
	for _, obj := range xj.RootPanel.Children.Objects {
		job.Boards = append(job.Boards, BoardRef{
			ID:       obj.ID,
			FileName: obj.FileName,
			Side:     obj.Side,
			Enabled:  obj.Enabled,
			Location: obj.Location,
		})
	}
	// Old format (board-locations > board-location)
	for _, bl := range xj.BoardLocations.Boards {
		job.Boards = append(job.Boards, BoardRef{
			FileName: bl.BoardFile,
			Side:     bl.Side,
			Enabled:  bl.Enabled,
			Location: bl.Location,
		})
	}
	return job, nil
}

func ParseBoard(path string) (*Board, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading board file: %w", err)
	}
	var xb xmlBoard
	if err := xml.Unmarshal(data, &xb); err != nil {
		return nil, fmt.Errorf("parsing board file: %w", err)
	}
	board := &Board{Name: xb.Name, Width: xb.Dimensions.X, Height: xb.Dimensions.Y}
	for _, xp := range xb.Placements.Placements {
		board.Placements = append(board.Placements, Placement(xp))
	}
	return board, nil
}

// LoadJobParts reads a job file and all referenced boards, returning
// a map of part-id → count of enabled placements (excluding fiducials)
// and a list of board entries with their locations and dimensions.
func LoadJobParts(jobPath string) (map[string]int, []BoardEntry, error) {
	job, err := ParseJob(jobPath)
	if err != nil {
		return nil, nil, err
	}
	jobDir := filepath.Dir(jobPath)
	parts := make(map[string]int)
	var boards []BoardEntry
	for _, br := range job.Boards {
		if !br.Enabled {
			continue
		}
		boardPath := br.FileName
		if !filepath.IsAbs(boardPath) {
			boardPath = filepath.Join(jobDir, boardPath)
		}
		board, err := ParseBoard(boardPath)
		if err != nil {
			return nil, nil, fmt.Errorf("board %s: %w", br.ID, err)
		}
		boards = append(boards, BoardEntry{
			Name:     board.Name,
			Width:    board.Width,
			Height:   board.Height,
			Location: br.Location,
		})
		for _, p := range board.Placements {
			if p.Enabled && p.Type == "Placement" {
				parts[p.PartID]++
			}
		}
	}
	return parts, boards, nil
}
