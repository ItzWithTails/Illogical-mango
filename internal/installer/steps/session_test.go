package steps

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ilmango/internal/fsx"
	"ilmango/internal/installer"
)

func TestDesktopSettingsPublishesTheCursorToTheCompositor(t *testing.T) {
	// gsettings only reaches GTK applications. The pointer the user moves is
	// drawn by the compositor, which resolves the XCursor theme called
	// "default" — so that name has to be pointed at our theme too.
	home := t.TempDir()
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))

	// An empty Root writes at the paths as given, which the XDG override has
	// already pointed inside the temporary directory.
	env := &installer.Env{Home: fsx.Tree{Mode: fsx.ModeApply}}
	if err := (desktopSettingsStep{}).writeCursorDefault(env); err != nil {
		t.Fatalf("writeCursorDefault() error = %v", err)
	}

	written, err := os.ReadFile(filepath.Join(home, ".local", "share", "icons", "default", "index.theme"))
	if err != nil {
		t.Fatalf("no default cursor theme was written: %v", err)
	}
	if want := "Inherits=" + cursorTheme; !strings.Contains(string(written), want) {
		t.Fatalf("default cursor theme is\n%s\nwant it to contain %q", written, want)
	}
}

func TestMangoTemplateSetsTheCursorEnvironment(t *testing.T) {
	// The compositor reads XCURSOR_THEME before anything else can correct it,
	// so the shipped config has to carry it.
	config, err := os.ReadFile(filepath.Join("..", "..", "..", "defaults", "mango", "config.conf"))
	if err != nil {
		t.Skipf("template not readable from here: %v", err)
	}

	for _, want := range []string{"env=XCURSOR_THEME," + cursorTheme, "env=XCURSOR_SIZE," + cursorSize} {
		if !strings.Contains(string(config), want) {
			t.Errorf("the mango template does not carry %q", want)
		}
	}
}
