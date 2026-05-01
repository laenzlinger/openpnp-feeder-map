// SPDX-License-Identifier: GPL-3.0-or-later

package openpnp

import (
	"fmt"
	"os"
	"strings"

	"github.com/laenzlinger/openpnp-tools/internal/generate"
)

// EnsurePartsResult reports what happened.
type EnsurePartsResult struct {
	Created []string
	Existed []string
}

// packageFromPartID extracts the package name from a part-id like "C_0805-100n".
func packageFromPartID(partID string) string {
	for i := len(partID) - 1; i > 0; i-- {
		if partID[i] == '-' {
			return partID[:i]
		}
	}
	return partID
}

// PackageFromPartID extracts the package name from a part-id using the package map.
// Falls back to last-dash split if no match found.
func PackageFromPartID(partID string, pkgMap *generate.PackageMap) string {
	if pkgMap != nil {
		for i := len(partID) - 1; i > 0; i-- {
			if partID[i] == '-' {
				candidate := partID[:i]
				if _, ok := pkgMap.LookupByPackage(candidate); ok {
					return candidate
				}
			}
		}
	}
	return packageFromPartID(partID)
}

// EnsurePartsWithMap ensures parts exist in parts.xml, using the package map for heights.
func EnsurePartsWithMap(
	partsPath string,
	partIDs []string,
	pkgMap *generate.PackageMap,
) (*EnsurePartsResult, error) {
	data, err := os.ReadFile(partsPath)
	if err != nil {
		return nil, fmt.Errorf("reading parts.xml: %w", err)
	}
	content := string(data)
	result := &EnsurePartsResult{}

	for _, partID := range partIDs {
		pkg := PackageFromPartID(partID, pkgMap)
		height := 0.0
		if info, ok := pkgMap.LookupByPackage(pkg); ok {
			height = info.Height
		}

		marker := fmt.Sprintf(`id="%s"`, partID)
		if strings.Contains(content, marker) {
			// Update height if package map has one
			if height > 0 {
				content = replacePartAttr(content, partID, "height", fmt.Sprintf("%.1f", height))
			}
			result.Existed = append(result.Existed, partID)
			continue
		}

		entry := fmt.Sprintf(
			`   <part id="%s" height-units="Millimeters"`+
				` height="%.1f" package-id="%s" speed="1.0" pick-retry-count="0"/>`,
			partID, height, pkg)

		content = strings.Replace(content, "</openpnp-parts>",
			entry+"\n</openpnp-parts>", 1)
		result.Created = append(result.Created, partID)
	}

	if err := os.WriteFile(partsPath, []byte(content), 0o600); err != nil {
		return nil, fmt.Errorf("writing parts.xml: %w", err)
	}
	return result, nil
}

// replacePartAttr replaces an attribute value on a part element.
func replacePartAttr(content, partID, attr, value string) string {
	marker := fmt.Sprintf(`id="%s"`, partID)
	idx := strings.Index(content, marker)
	if idx == -1 {
		return content
	}
	searchEnd := idx + 500
	if searchEnd > len(content) {
		searchEnd = len(content)
	}
	region := content[idx:searchEnd]
	attrMarker := fmt.Sprintf(`%s="`, attr)
	attrIdx := strings.Index(region, attrMarker)
	if attrIdx == -1 {
		return content
	}
	absStart := idx + attrIdx + len(attrMarker)
	absEnd := absStart + strings.Index(content[absStart:], `"`)
	return content[:absStart] + value + content[absEnd:]
}
