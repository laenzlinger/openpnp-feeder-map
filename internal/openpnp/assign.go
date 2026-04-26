// SPDX-License-Identifier: GPL-3.0-or-later

package openpnp

import (
	"fmt"
	"os"
	"strings"

	"github.com/laenzlinger/openpnp-tools/internal/generate"
)

// FeederAssignment maps a feeder name to a part ID.
type FeederAssignment struct {
	FeederName string
	PartID     string
}

// AssignResult reports what happened for each assignment.
type AssignResult struct {
	FeederName string
	PartID     string
	OldPartID  string
	Status     string // "assigned", "unchanged", "not_found"
}

// AssignFeeders updates feeder attributes in machine.xml.
// Sets part-id, tape-type, and part-pitch based on the package map.
// If resetUnused is true, feeders not in the assignments list are reset to dummyPartID.
func AssignFeeders(
	machinePath string,
	assignments []FeederAssignment,
	pkgMap *generate.PackageMap,
	resetUnused bool,
	dummyPartID string,
	stripLength float64,
) ([]AssignResult, error) {
	data, err := os.ReadFile(machinePath)
	if err != nil {
		return nil, fmt.Errorf("reading machine.xml: %w", err)
	}
	content := string(data)
	var results []AssignResult

	assigned := make(map[string]bool)
	for _, a := range assignments {
		assigned[a.FeederName] = true
		marker := fmt.Sprintf(`name="%s" enabled="true" part-id="`, a.FeederName)
		idx := strings.Index(content, marker)
		if idx == -1 {
			results = append(results, AssignResult{
				FeederName: a.FeederName,
				PartID:     a.PartID,
				Status:     "not_found",
			})
			continue
		}

		start := idx + len(marker)
		end := strings.Index(content[start:], `"`)
		if end == -1 {
			continue
		}
		end += start
		oldPartID := content[start:end]

		changed := oldPartID != a.PartID
		content = content[:start] + a.PartID + content[end:]

		pkg := packageFromPartID(a.PartID)
		info, ok := pkgMap.LookupByPackage(pkg)
		if ok && info.TapeType != "" {
			content = replaceFeederAttr(content, a.FeederName,
				"tape-type", info.TapeType)
		}
		if ok && info.PartPitch > 0 {
			content = replacePartPitch(content, a.FeederName,
				info.PartPitch)
		}
		if ok && info.PartPitch > 0 && stripLength > 0 {
			cap := int(stripLength/info.PartPitch) - 2
			if cap > 0 {
				content = replaceFeederAttr(content, a.FeederName,
					"max-feed-count", fmt.Sprintf("%d", cap))
			}
		}

		status := "unchanged"
		if changed {
			status = "assigned"
		}
		results = append(results, AssignResult{
			FeederName: a.FeederName,
			PartID:     a.PartID,
			OldPartID:  oldPartID,
			Status:     status,
		})
	}

	if resetUnused {
		results = append(results,
			resetUnassignedFeeders(&content, assigned, dummyPartID)...)
	}

	if err := os.WriteFile(machinePath, []byte(content), 0o600); err != nil {
		return nil, fmt.Errorf("writing machine.xml: %w", err)
	}
	return results, nil
}

// resetUnassignedFeeders resets feeders not in the assigned set to dummyPartID.
func resetUnassignedFeeders(
	content *string,
	assigned map[string]bool,
	dummyPartID string,
) []AssignResult {
	var results []AssignResult
	offset := 0
	for {
		pattern := `enabled="true" part-id="`
		idx := strings.Index((*content)[offset:], pattern)
		if idx == -1 {
			break
		}
		absIdx := offset + idx

		// Find the feeder name before this point
		nameEnd := strings.LastIndex((*content)[:absIdx], `" `)
		if nameEnd == -1 {
			offset = absIdx + len(pattern)
			continue
		}
		nameMarker := `name="`
		nameStart := strings.LastIndex((*content)[:nameEnd], nameMarker)
		if nameStart == -1 {
			offset = absIdx + len(pattern)
			continue
		}
		nameStart += len(nameMarker)
		feederName := (*content)[nameStart:nameEnd]

		// Skip if already assigned
		if assigned[feederName] {
			offset = absIdx + len(pattern)
			continue
		}

		// Get current part-id
		partStart := absIdx + len(pattern)
		partEnd := partStart + strings.Index((*content)[partStart:], `"`)
		oldPartID := (*content)[partStart:partEnd]

		if oldPartID != dummyPartID {
			*content = (*content)[:partStart] + dummyPartID + (*content)[partEnd:]
			results = append(results, AssignResult{
				FeederName: feederName,
				PartID:     dummyPartID,
				OldPartID:  oldPartID,
				Status:     "reset",
			})
		}

		offset = partStart + len(dummyPartID) + 1
	}
	return results
}

// replaceFeederAttr replaces an attribute value on the feeder element.
func replaceFeederAttr(content, feederName, attr, value string) string {
	// Find the feeder by name
	nameMarker := fmt.Sprintf(`name="%s"`, feederName)
	idx := strings.Index(content, nameMarker)
	if idx == -1 {
		return content
	}
	// Search for the attribute within a reasonable range after the name
	searchStart := idx
	searchEnd := searchStart + 500
	if searchEnd > len(content) {
		searchEnd = len(content)
	}
	region := content[searchStart:searchEnd]

	attrMarker := fmt.Sprintf(`%s="`, attr)
	attrIdx := strings.Index(region, attrMarker)
	if attrIdx == -1 {
		return content
	}
	absStart := searchStart + attrIdx + len(attrMarker)
	absEnd := absStart + strings.Index(content[absStart:], `"`)
	return content[:absStart] + value + content[absEnd:]
}

// replacePartPitch updates the part-pitch value element for a feeder.
func replacePartPitch(content, feederName string, pitch float64) string {
	nameMarker := fmt.Sprintf(`name="%s"`, feederName)
	idx := strings.Index(content, nameMarker)
	if idx == -1 {
		return content
	}
	// Find <part-pitch value="..." after this feeder
	searchStart := idx
	searchEnd := searchStart + 1000
	if searchEnd > len(content) {
		searchEnd = len(content)
	}
	region := content[searchStart:searchEnd]

	pitchMarker := `<part-pitch value="`
	pitchIdx := strings.Index(region, pitchMarker)
	if pitchIdx == -1 {
		return content
	}
	absStart := searchStart + pitchIdx + len(pitchMarker)
	absEnd := absStart + strings.Index(content[absStart:], `"`)
	return content[:absStart] + fmt.Sprintf("%.1f", pitch) + content[absEnd:]
}
