// SPDX-License-Identifier: GPL-3.0-or-later

package openpnp

import (
	"fmt"
	"os"
	"strings"
)

// Part represents an OpenPnP part entry.
type Part struct {
	ID        string
	PackageID string
	Height    float64
}

// EnsurePartsResult reports what happened.
type EnsurePartsResult struct {
	Created []string
	Existed []string
}

// DefaultHeights maps package prefixes to default component heights in mm.
var DefaultHeights = map[string]float64{
	"C_0805":       0.9,
	"R_0805":       0.5,
	"LED_0805":     0.8,
	"SOT-23":       1.1,
	"SOT-23-5":     1.3,
	"SOT-23-6":     1.3,
	"SOT-223":      1.6,
	"SOIC-8":       1.75,
	"SOIC-8-EP":    0.65,
	"QFN-48-7x7":   0.9,
	"FUSE-2512":    1.0,
	"TANT-D":       1.9,
	"IND-6045":     4.2,
	"WS2812B-5050": 1.6,
	"XTAL-2016":    0.7,
	"FIDUCIAL-1X2": 0.0,
	"ECAP-6.3x7.7": 7.7,
}

// heightForPackage returns the default height for a package, or 0.0 if unknown.
func heightForPackage(pkg string) float64 {
	// Try exact match first, then prefix match (longest first)
	if h, ok := DefaultHeights[pkg]; ok {
		return h
	}
	best := ""
	for prefix := range DefaultHeights {
		if strings.HasPrefix(pkg, prefix) && len(prefix) > len(best) {
			best = prefix
		}
	}
	if best != "" {
		return DefaultHeights[best]
	}
	return 0.0
}

// packageFromPartID extracts the package name from a part-id like "C_0805-100n".
// It tries known package prefixes (longest match wins).
func packageFromPartID(partID string) string {
	// The part-id is {Package}-{Value}. Package can contain dashes (e.g. SOT-23-5).
	// Try progressively shorter prefixes until we find a dash that splits correctly.
	for i := len(partID) - 1; i > 0; i-- {
		if partID[i] == '-' {
			return partID[:i]
		}
	}
	return partID
}

// EnsureParts reads a board XML, extracts all part-ids, and ensures they exist
// in parts.xml with correct package-id and default heights.
func EnsureParts(partsPath string, partIDs []string) (*EnsurePartsResult, error) {
	data, err := os.ReadFile(partsPath)
	if err != nil {
		return nil, fmt.Errorf("reading parts.xml: %w", err)
	}
	content := string(data)
	result := &EnsurePartsResult{}

	for _, partID := range partIDs {
		marker := fmt.Sprintf(`id="%s"`, partID)
		if strings.Contains(content, marker) {
			result.Existed = append(result.Existed, partID)
			continue
		}

		pkg := packageFromPartID(partID)
		height := heightForPackage(pkg)
		entry := fmt.Sprintf(`   <part id="%s" height-units="Millimeters"`+
			` height="%.1f" package-id="%s" speed="1.0" pick-retry-count="0"/>`,
			partID, height, pkg)

		// Insert before closing tag
		content = strings.Replace(content, "</openpnp-parts>", entry+"\n</openpnp-parts>", 1)
		result.Created = append(result.Created, partID)
	}

	if len(result.Created) > 0 {
		if err := os.WriteFile(partsPath, []byte(content), 0o600); err != nil {
			return nil, fmt.Errorf("writing parts.xml: %w", err)
		}
	}
	return result, nil
}
