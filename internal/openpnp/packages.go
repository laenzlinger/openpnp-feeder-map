// SPDX-License-Identifier: GPL-3.0-or-later

package openpnp

import (
	"fmt"
	"os"
	"strings"
)

// EnsurePackagesResult reports what happened.
type EnsurePackagesResult struct {
	Created []string
	Existed []string
}

// EnsurePackages ensures package entries exist in packages.xml.
func EnsurePackages(packagesPath string, packageIDs []string) (*EnsurePackagesResult, error) {
	data, err := os.ReadFile(packagesPath)
	if err != nil {
		return nil, fmt.Errorf("reading packages.xml: %w", err)
	}
	content := string(data)
	result := &EnsurePackagesResult{}

	for _, pkgID := range packageIDs {
		marker := fmt.Sprintf(`id="%s"`, pkgID)
		if strings.Contains(content, marker) {
			result.Existed = append(result.Existed, pkgID)
			continue
		}

		entry := fmt.Sprintf(
			`   <package version="1.1" id="%s"`+
				` pick-vacuum-level="0.0" place-blow-off-level="0.0"/>`,
			pkgID)
		content = strings.Replace(content,
			"</openpnp-packages>", entry+"\n</openpnp-packages>", 1)
		result.Created = append(result.Created, pkgID)
	}

	if len(result.Created) > 0 {
		if err := os.WriteFile(packagesPath, []byte(content), 0o600); err != nil {
			return nil, fmt.Errorf("writing packages.xml: %w", err)
		}
	}
	return result, nil
}
