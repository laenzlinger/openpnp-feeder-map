// SPDX-License-Identifier: GPL-3.0-or-later

package openpnp

import (
	"fmt"
	"os"
	"strings"

	"github.com/laenzlinger/openpnp-tools/internal/generate"
)

// EnsurePackagesResult reports what happened.
type EnsurePackagesResult struct {
	Created []string
	Existed []string
	Updated []string
}

// EnsurePackages ensures package entries exist in packages.xml and sets tape-specification.
func EnsurePackages(packagesPath string, packageIDs []string, pkgMap *generate.PackageMap) (*EnsurePackagesResult, error) {
	data, err := os.ReadFile(packagesPath)
	if err != nil {
		return nil, fmt.Errorf("reading packages.xml: %w", err)
	}
	content := string(data)
	result := &EnsurePackagesResult{}

	for _, pkgID := range packageIDs {
		marker := fmt.Sprintf(`id="%s"`, pkgID)
		if strings.Contains(content, marker) {
			content = setTapeSpec(content, pkgID, pkgMap, result)
			result.Existed = append(result.Existed, pkgID)
			continue
		}

		tapeSpec := tapeSpecAttr(pkgID, pkgMap)
		entry := fmt.Sprintf(
			`   <package version="1.1" id="%s"%s`+
				` pick-vacuum-level="0.0" place-blow-off-level="0.0"/>`,
			pkgID, tapeSpec)
		content = strings.Replace(content,
			"</openpnp-packages>", entry+"\n</openpnp-packages>", 1)
		result.Created = append(result.Created, pkgID)
	}

	if len(result.Created) > 0 || len(result.Updated) > 0 {
		if err := os.WriteFile(packagesPath, []byte(content), 0o600); err != nil {
			return nil, fmt.Errorf("writing packages.xml: %w", err)
		}
	}
	return result, nil
}

// tapeSpecAttr returns the tape-specification attribute string for a new package.
func tapeSpecAttr(pkgID string, pkgMap *generate.PackageMap) string {
	if pkgMap == nil {
		return ""
	}
	info, ok := pkgMap.LookupByPackage(pkgID)
	if !ok || info.TapeWidth == 0 {
		return ""
	}
	return fmt.Sprintf(` tape-specification="%dmm"`, info.TapeWidth)
}

// setTapeSpec updates tape-specification on an existing package if needed.
func setTapeSpec(content, pkgID string, pkgMap *generate.PackageMap, result *EnsurePackagesResult) string {
	if pkgMap == nil {
		return content
	}
	info, ok := pkgMap.LookupByPackage(pkgID)
	if !ok || info.TapeWidth == 0 {
		return content
	}

	spec := fmt.Sprintf(`%dmm`, info.TapeWidth)
	marker := fmt.Sprintf(`id="%s"`, pkgID)
	idx := strings.Index(content, marker)
	if idx < 0 {
		return content
	}

	// Find the end of this element's opening tag
	tagEnd := strings.Index(content[idx:], ">")
	if tagEnd < 0 {
		return content
	}
	tagEnd += idx

	return updateOrInsertTapeSpec(content, idx, tagEnd, marker, spec, result, pkgID)
}

func updateOrInsertTapeSpec(content string, idx, tagEnd int, marker, spec string, result *EnsurePackagesResult, pkgID string) string {
	tag := content[idx:tagEnd]
	tapeKey := `tape-specification="`
	tapeIdx := strings.Index(tag, tapeKey)

	if tapeIdx < 0 {
		// Insert tape-specification after id="..."
		insertAfter := marker + `"`
		insertPos := strings.Index(content[idx:], insertAfter)
		if insertPos >= 0 {
			pos := idx + insertPos + len(insertAfter)
			content = content[:pos] + fmt.Sprintf(` tape-specification="%s"`, spec) + content[pos:]
			result.Updated = append(result.Updated, pkgID)
		}
		return content
	}

	// Already has tape-specification — replace if different
	valStart := tapeIdx + len(tapeKey)
	valEnd := strings.Index(tag[valStart:], `"`)
	if valEnd < 0 {
		return content
	}
	current := tag[valStart : valStart+valEnd]
	if current == spec {
		return content
	}
	oldAttr := fmt.Sprintf(`tape-specification="%s"`, current)
	newAttr := fmt.Sprintf(`tape-specification="%s"`, spec)
	content = content[:idx] + strings.Replace(content[idx:tagEnd], oldAttr, newAttr, 1) + content[tagEnd:]
	result.Updated = append(result.Updated, pkgID)
	return content
}
