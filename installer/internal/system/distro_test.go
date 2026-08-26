package system

import "testing"

func TestClassifyPrefersDerivativeOverParent(t *testing.T) {
	tests := []struct {
		name   string
		id     string
		idLike string
		want   Family
	}{
		{"plain arch", "arch", "", FamilyArch},
		{"artix", "artix", "arch", FamilyArch},
		{"cachyos", "cachyos", "arch", FamilyArch},
		{"fedora", "fedora", "", FamilyFedora},
		{"nobara", "nobara", "fedora", FamilyFedora},
		// Ubuntu derivatives list debian in ID_LIKE; the more specific
		// family has to win or they get the wrong package names.
		{"ubuntu", "ubuntu", "debian", FamilyUbuntu},
		{"pop", "pop", "ubuntu debian", FamilyUbuntu},
		{"debian", "debian", "", FamilyDebian},
		{"opensuse tumbleweed", "opensuse-tumbleweed", "opensuse suse", FamilyOpenSUSE},
		{"void", "void", "", FamilyVoid},
		{"nixos", "nixos", "", FamilyNixOS},
		{"alpine", "alpine", "", FamilyAlpine},
		{"unheard of", "plan9", "", FamilyGeneric},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := classify(tc.id, tc.idLike); got != tc.want {
				t.Errorf("classify(%q, %q) = %q, want %q", tc.id, tc.idLike, got, tc.want)
			}
		})
	}
}

func TestSupportLevels(t *testing.T) {
	if got := (Distro{Family: FamilyArch}).Support(); got != SupportFull {
		t.Errorf("arch support = %v, want full", got)
	}
	if got := (Distro{Family: FamilyNixOS}).Support(); got != SupportManual {
		t.Errorf("nixos support = %v, want manual", got)
	}
	if got := (Distro{Family: FamilyVoid}).Support(); got != SupportExperimental {
		t.Errorf("void support = %v, want experimental", got)
	}
}

func TestBlockingOnlyOnFailures(t *testing.T) {
	if Blocking([]CheckResult{{Status: CheckPass}, {Status: CheckWarn}}) {
		t.Error("warnings must not block installation")
	}
	if !Blocking([]CheckResult{{Status: CheckPass}, {Status: CheckFail}}) {
		t.Error("a failure must block installation")
	}
}
