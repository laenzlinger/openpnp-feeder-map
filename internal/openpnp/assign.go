// SPDX-License-Identifier: GPL-3.0-or-later

package openpnp

import (
	"fmt"
	"os"
	"strings"
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

// AssignFeeders updates part-id attributes in machine.xml based on the given assignments.
// It uses string replacement to preserve the original XML formatting.
func AssignFeeders(machinePath string, assignments []FeederAssignment) ([]AssignResult, error) {
	data, err := os.ReadFile(machinePath)
	if err != nil {
		return nil, fmt.Errorf("reading machine.xml: %w", err)
	}
	content := string(data)
	var results []AssignResult

	for _, a := range assignments {
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

		if oldPartID == a.PartID {
			results = append(results, AssignResult{
				FeederName: a.FeederName,
				PartID:     a.PartID,
				OldPartID:  oldPartID,
				Status:     "unchanged",
			})
			continue
		}

		content = content[:start] + a.PartID + content[end:]
		results = append(results, AssignResult{
			FeederName: a.FeederName,
			PartID:     a.PartID,
			OldPartID:  oldPartID,
			Status:     "assigned",
		})
	}

	if err := os.WriteFile(machinePath, []byte(content), 0o644); err != nil {
		return nil, fmt.Errorf("writing machine.xml: %w", err)
	}
	return results, nil
}
