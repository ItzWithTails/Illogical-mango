package v2

import (
	"slices"
	"testing"
)

func TestEveryArchPresetInstallsMangoWM(t *testing.T) {
	for _, preset := range []Preset{Minimal, Recommended, Full} {
		if packages := archPackages(preset); !slices.Contains(packages, "mangowm") {
			t.Errorf("%s packages omit mangowm: %v", preset, packages)
		}
	}
}
