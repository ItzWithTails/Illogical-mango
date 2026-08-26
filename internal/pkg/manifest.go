// Package pkg installs distribution packages.
//
// It knows two things: which packages a group needs on a given distribution
// family, and how to drive that family's package manager. Both are data —
// packages.json and the manager table — so supporting another distribution
// means adding entries, not writing control flow.
package pkg

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
)

//go:embed packages.json
var manifestJSON []byte

// Group names a set of packages that an installation option switches on.
type Group string

const (
	GroupCore          Group = "core"
	GroupQuickshell    Group = "quickshell"
	GroupAudio         Group = "audio"
	GroupToolkit       Group = "toolkit"
	GroupScreenCapture Group = "screencapture"
	GroupFonts         Group = "fonts"
)

// manifest is the parsed packages.json.
type manifest struct {
	Families map[string]map[Group][]string `json:"families"`
	// Critical names packages the shell visibly needs. It cuts across
	// families, since a package that is critical on one distribution is
	// critical everywhere it exists.
	Critical []string `json:"critical"`
}

var parsed manifest

func init() {
	if err := json.Unmarshal(manifestJSON, &parsed); err != nil {
		// The manifest is embedded at build time; a parse failure is a broken
		// build, not a runtime condition worth handling gracefully.
		panic(fmt.Sprintf("pkg: malformed packages.json: %v", err))
	}
}

// Packages returns the packages a family needs for the given groups,
// deduplicated and sorted. An unknown family yields no packages, which the
// caller reports rather than treating as an empty success.
func Packages(family string, groups ...Group) []string {
	sets, ok := parsed.Families[family]
	if !ok {
		return nil
	}

	seen := map[string]bool{}
	for _, group := range groups {
		for _, name := range sets[group] {
			seen[name] = true
		}
	}

	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// IsCritical reports whether the shell visibly breaks without a package.
func IsCritical(name string) bool {
	for _, critical := range parsed.Critical {
		if critical == name {
			return true
		}
	}
	return false
}

// SplitCritical partitions names into those the shell needs and the rest.
func SplitCritical(names []string) (critical, optional []string) {
	for _, name := range names {
		if IsCritical(name) {
			critical = append(critical, name)
		} else {
			optional = append(optional, name)
		}
	}
	return critical, optional
}

// KnownFamily reports whether the manifest covers a distribution family.
func KnownFamily(family string) bool {
	_, ok := parsed.Families[family]
	return ok
}

// Families lists the covered families, for help text and diagnostics.
func Families() []string {
	out := make([]string, 0, len(parsed.Families))
	for name := range parsed.Families {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
