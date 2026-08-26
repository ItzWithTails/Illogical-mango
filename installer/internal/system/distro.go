// Package system inspects the host: which distribution it runs, where the Illogical-mango
// checkout lives, and whether the machine is ready to be installed onto.
package system

import (
	"bufio"
	"os"
	"strings"
)

// Family groups distributions that share a dependency installer. It mirrors
// OS_GROUP_ID in sdata/lib/dist-determine.sh — the shell phases branch on the
// same values, so the two must agree.
type Family string

const (
	FamilyArch     Family = "arch"
	FamilyFedora   Family = "fedora"
	FamilyDebian   Family = "debian"
	FamilyUbuntu   Family = "ubuntu"
	FamilyOpenSUSE Family = "opensuse"
	FamilyVoid     Family = "void"
	FamilyGentoo   Family = "gentoo"
	FamilyNixOS    Family = "nixos"
	FamilyAlpine   Family = "alpine"
	FamilyGeneric  Family = "generic"
)

// SupportLevel describes how well tested a family is.
type SupportLevel int

const (
	// SupportFull means the family has a maintained dependency installer.
	SupportFull SupportLevel = iota
	// SupportExperimental means installation falls back to the generic path.
	SupportExperimental
	// SupportManual means the family cannot be installed automatically.
	SupportManual
)

func (s SupportLevel) String() string {
	switch s {
	case SupportFull:
		return "supported"
	case SupportExperimental:
		return "experimental"
	default:
		return "manual setup required"
	}
}

// Distro is the identified host distribution.
type Distro struct {
	Family   Family
	ID       string // the specific derivative, e.g. "artix", "pop"
	Name     string // PRETTY_NAME from os-release
	Version  string
	Detected bool
}

// Support reports how well this installer handles the distribution.
func (d Distro) Support() SupportLevel {
	switch d.Family {
	case FamilyArch, FamilyFedora, FamilyDebian, FamilyUbuntu:
		return SupportFull
	case FamilyNixOS:
		return SupportManual
	default:
		return SupportExperimental
	}
}

// osReleasePaths are probed in order; the first readable file wins.
var osReleasePaths = []string{"/etc/os-release", "/usr/lib/os-release"}

// DetectDistro identifies the host from os-release. An unrecognised or missing
// os-release yields the generic family rather than an error: the generic
// dependency installer exists precisely for that case.
func DetectDistro() Distro {
	fields := readOSRelease()
	if len(fields) == 0 {
		return Distro{Family: FamilyGeneric, ID: "unknown", Name: "Unknown distribution"}
	}

	d := Distro{
		ID:       fields["ID"],
		Name:     firstNonEmpty(fields["PRETTY_NAME"], fields["NAME"], fields["ID"]),
		Version:  firstNonEmpty(fields["VERSION_ID"], fields["VERSION"]),
		Detected: true,
	}
	d.Family = classify(fields["ID"], fields["ID_LIKE"])
	if d.Family == FamilyGeneric {
		d.Detected = false
	}
	return d
}

// classify maps an os-release ID and ID_LIKE onto a family. ID is checked
// first so a derivative that also lists a parent in ID_LIKE lands correctly.
func classify(id, idLike string) Family {
	tokens := append([]string{strings.ToLower(id)}, strings.Fields(strings.ToLower(idLike))...)

	// Ubuntu derivatives must be tested before Debian: they list both.
	for _, want := range []struct {
		token  string
		family Family
	}{
		{"arch", FamilyArch},
		{"archarm", FamilyArch},
		{"artix", FamilyArch},
		{"manjaro", FamilyArch},
		{"cachyos", FamilyArch},
		{"endeavouros", FamilyArch},
		{"garuda", FamilyArch},
		{"nixos", FamilyNixOS},
		{"fedora", FamilyFedora},
		{"rhel", FamilyFedora},
		{"ubuntu", FamilyUbuntu},
		{"debian", FamilyDebian},
		{"opensuse", FamilyOpenSUSE},
		{"suse", FamilyOpenSUSE},
		{"void", FamilyVoid},
		{"gentoo", FamilyGentoo},
		{"alpine", FamilyAlpine},
	} {
		for _, tok := range tokens {
			if tok == want.token || strings.HasPrefix(tok, want.token) {
				return want.family
			}
		}
	}
	return FamilyGeneric
}

// readOSRelease parses the shell-style key=value os-release format.
func readOSRelease() map[string]string {
	for _, path := range osReleasePaths {
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		defer f.Close()

		fields := map[string]string{}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			key, value, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			fields[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"'`)
		}
		if scanner.Err() == nil && len(fields) > 0 {
			return fields
		}
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
