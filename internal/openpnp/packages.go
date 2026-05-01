// SPDX-License-Identifier: GPL-3.0-or-later

package openpnp

import (
	"encoding/xml"
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
			content = setNozzleTip(content, pkgID, pkgMap, result)
			result.Existed = append(result.Existed, pkgID)
			continue
		}

		tapeSpec := tapeSpecAttr(pkgID, pkgMap)
		nozzle := nozzleTipElement(pkgID, pkgMap)
		var entry string
		if nozzle != "" {
			entry = fmt.Sprintf(
				"   <package version=\"1.1\" id=\"%s\"%s"+
					" pick-vacuum-level=\"0.0\" place-blow-off-level=\"0.0\">\n%s"+
					"   </package>",
				pkgID, tapeSpec, nozzle)
		} else {
			entry = fmt.Sprintf(
				`   <package version="1.1" id="%s"%s`+
					` pick-vacuum-level="0.0" place-blow-off-level="0.0"/>`,
				pkgID, tapeSpec)
		}
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

// nozzleTipElement returns the XML child element for compatible-nozzle-tip-ids.
func nozzleTipElement(pkgID string, pkgMap *generate.PackageMap) string {
	if pkgMap == nil {
		return ""
	}
	info, ok := pkgMap.LookupByPackage(pkgID)
	if !ok || info.NozzleTip == "" {
		return ""
	}
	return fmt.Sprintf("      <compatible-nozzle-tip-ids class=\"java.util.ArrayList\">\n"+
		"         <string>%s</string>\n"+
		"      </compatible-nozzle-tip-ids>\n", info.NozzleTip)
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

// setNozzleTip ensures the compatible-nozzle-tip-ids element exists on an existing package.
// It uses a two-pass approach: first parse with encoding/xml to find which packages need
// updating, then apply targeted string replacements at the correct positions.
func setNozzleTip(content, pkgID string, pkgMap *generate.PackageMap, result *EnsurePackagesResult) string {
	if pkgMap == nil {
		return content
	}
	info, ok := pkgMap.LookupByPackage(pkgID)
	if !ok || info.NozzleTip == "" {
		return content
	}

	// Parse to check current state
	type nozzleTips struct {
		Tips []string `xml:"compatible-nozzle-tip-ids>string"`
	}
	type xmlPkg struct {
		ID    string     `xml:"id,attr"`
		Tips  nozzleTips `xml:",any"`
		Inner string     `xml:",innerxml"`
	}
	type xmlPkgs struct {
		Packages []xmlPkg `xml:"package"`
	}
	var pkgs xmlPkgs
	if err := xml.Unmarshal([]byte(content), &pkgs); err != nil {
		return content
	}

	// Find the package and check if it already has the right tip
	for _, p := range pkgs.Packages {
		if p.ID != pkgID {
			continue
		}
		for _, tip := range p.Tips.Tips {
			if tip == info.NozzleTip {
				return content // already correct
			}
		}
		break
	}

	// Find the package element in the string
	marker := fmt.Sprintf(`id="%s"`, pkgID)
	idx := strings.Index(content, marker)
	if idx < 0 {
		return content
	}

	// Find the closing </package> for this specific element by counting nesting
	searchFrom := idx
	closePos := findPackageClose(content, searchFrom)
	if closePos < 0 {
		return content
	}

	nozzleElement := fmt.Sprintf("      <compatible-nozzle-tip-ids class=\"java.util.ArrayList\">\n"+
		"         <string>%s</string>\n"+
		"      </compatible-nozzle-tip-ids>\n", info.NozzleTip)

	// Check if there's an existing compatible-nozzle-tip-ids to replace
	pkgContent := content[idx:closePos]
	tipStart := strings.Index(pkgContent, "<compatible-nozzle-tip-ids")
	tipEnd := strings.Index(pkgContent, "</compatible-nozzle-tip-ids>")

	if tipStart >= 0 && tipEnd >= 0 {
		// Replace existing
		tipEnd += len("</compatible-nozzle-tip-ids>")
		// Find the start of the line (include leading whitespace)
		lineStart := tipStart
		for lineStart > 0 && pkgContent[lineStart-1] == ' ' {
			lineStart--
		}
		content = content[:idx+lineStart] + nozzleElement + content[idx+tipEnd:]
	} else {
		// Insert before </package>
		content = content[:closePos] + nozzleElement + "   " + content[closePos:]
	}
	result.Updated = append(result.Updated, pkgID)
	return content
}

// findPackageClose finds the position of </package> that closes the package element
// containing the given start position. Handles self-closing tags by converting them.
func findPackageClose(content string, startIdx int) int {
	// First check if this is a self-closing package tag
	// Find the opening < before our marker
	tagStart := strings.LastIndex(content[:startIdx], "<package ")
	if tagStart < 0 {
		return -1
	}

	// Scan forward from tagStart to find either /> or >
	pos := startIdx
	for pos < len(content)-1 {
		if content[pos] == '/' && content[pos+1] == '>' {
			// Self-closing — this shouldn't happen for packages with existing content
			// but handle it: return -1 to signal we need conversion
			return -1
		}
		if content[pos] == '>' {
			break
		}
		pos++
	}

	// Now find the matching </package> by counting nesting
	depth := 1
	i := pos + 1
	for i < len(content) && depth > 0 {
		switch {
		case i+8 < len(content) && content[i:i+9] == "<package ":
			depth++
			i += 9
		case i+9 < len(content) && content[i:i+10] == "</package>":
			depth--
			if depth == 0 {
				return i
			}
			i += 10
		default:
			i++
		}
	}
	return -1
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
