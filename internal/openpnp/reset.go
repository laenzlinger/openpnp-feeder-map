// SPDX-License-Identifier: GPL-3.0-or-later

package openpnp

import (
	"fmt"
	"os"
	"strings"
)

// Status constants for feeder operations.
const (
	StatusReset     = "reset"
	StatusUnchanged = "unchanged"
	StatusAssigned  = "assigned"
	StatusNotFound  = "not_found"
)

// ResetResult reports what happened for each feeder.
type ResetResult struct {
	FeederName string
	OldPartID  string
	Status     string // StatusReset, StatusUnchanged
}

// ResetFeeders sets all feeder part-ids to dummyPartID and resets feed-count to 0.
// Leaves positions, calibration, tape-type, rotation, pitch, and everything else untouched.
func ResetFeeders(machinePath string, dummyPartID string) ([]ResetResult, error) {
	data, err := os.ReadFile(machinePath)
	if err != nil {
		return nil, fmt.Errorf("reading machine.xml: %w", err)
	}
	content := string(data)
	var results []ResetResult

	offset := 0
	for {
		nameMarker := `name="`
		idx := strings.Index(content[offset:], `ReferenceStripFeeder`)
		if idx == -1 {
			break
		}
		absIdx := offset + idx

		// Find feeder name
		nameIdx := strings.Index(content[absIdx:], nameMarker)
		if nameIdx == -1 {
			offset = absIdx + 20
			continue
		}
		nameStart := absIdx + nameIdx + len(nameMarker)
		nameEnd := strings.Index(content[nameStart:], `"`)
		if nameEnd == -1 {
			offset = nameStart
			continue
		}
		feederName := content[nameStart : nameStart+nameEnd]

		// Reset part-id
		partMarker := `part-id="`
		partIdx := strings.Index(content[absIdx:], partMarker)
		if partIdx == -1 {
			offset = nameStart + nameEnd
			continue
		}
		partStart := absIdx + partIdx + len(partMarker)
		partEnd := strings.Index(content[partStart:], `"`)
		if partEnd == -1 {
			offset = partStart
			continue
		}
		oldPartID := content[partStart : partStart+partEnd]

		// Reset feed-count
		feedMarker := `feed-count="`
		feedIdx := strings.Index(content[absIdx:], feedMarker)
		if feedIdx != -1 {
			feedStart := absIdx + feedIdx + len(feedMarker)
			feedEnd := strings.Index(content[feedStart:], `"`)
			if feedEnd != -1 {
				content = content[:feedStart] + "0" + content[feedStart+feedEnd:]
			}
		}

		// Set part-id (re-find after possible content shift)
		partIdx = strings.Index(content[absIdx:], partMarker)
		partStart = absIdx + partIdx + len(partMarker)
		partEnd = strings.Index(content[partStart:], `"`)
		oldPartID2 := content[partStart : partStart+partEnd]
		_ = oldPartID2

		status := StatusUnchanged
		if oldPartID != dummyPartID {
			content = content[:partStart] + dummyPartID + content[partStart+partEnd:]
			status = StatusReset
		}

		results = append(results, ResetResult{
			FeederName: feederName,
			OldPartID:  oldPartID,
			Status:     status,
		})

		offset = partStart + len(dummyPartID) + 1
	}

	if err := os.WriteFile(machinePath, []byte(content), 0o600); err != nil {
		return nil, fmt.Errorf("writing machine.xml: %w", err)
	}
	return results, nil
}
