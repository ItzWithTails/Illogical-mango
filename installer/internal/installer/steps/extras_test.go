package steps

import (
	"os"
	"path/filepath"
	"testing"

	"ilmango/internal/fsx"
	"ilmango/internal/installer"
)

// writeFile creates a file with content, making its parents.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestWallpaperCopySkipsThumbnailsAndKeepsExistingFiles(t *testing.T) {
	clone := t.TempDir()
	target := t.TempDir()

	// The pack's gallery thumbnails share the basename of the wallpaper they
	// preview, so a thumbnail that slipped through would silently replace a
	// full-size image with a 640px one.
	writeFile(t, filepath.Join(clone, "images", "sunset.jpg"), "the real wallpaper")
	writeFile(t, filepath.Join(clone, "images", "thumbs", "sunset.jpg"), "640px preview")
	writeFile(t, filepath.Join(clone, "images", "wide.webp"), "genuinely webp")
	writeFile(t, filepath.Join(clone, "README.md"), "not an image")
	writeFile(t, filepath.Join(clone, "images", "mine.png"), "from the pack")

	// One the user already has, which must survive untouched.
	writeFile(t, filepath.Join(target, "mine.png"), "the user's own")

	env := &installer.Env{Home: fsx.Tree{Mode: fsx.ModeApply}}
	copied, skipped, err := wallpapersStep{}.copyImages(env, clone, target)
	if err != nil {
		t.Fatalf("copyImages() error = %v", err)
	}

	if copied != 2 || skipped != 1 {
		t.Fatalf("copied %d and skipped %d, want 2 copied (sunset, wide) and 1 skipped (mine)", copied, skipped)
	}

	if got := readFile(t, filepath.Join(target, "sunset.jpg")); got != "the real wallpaper" {
		t.Errorf("sunset.jpg is %q — a thumbnail overwrote the wallpaper", got)
	}
	if got := readFile(t, filepath.Join(target, "mine.png")); got != "the user's own" {
		t.Errorf("mine.png is %q — the pack overwrote a file the user already had", got)
	}
	if _, err := os.Stat(filepath.Join(target, "README.md")); err == nil {
		t.Error("a non-image was copied into the wallpapers directory")
	}
}

func TestWallpapersAreNotRecordedForRemoval(t *testing.T) {
	// Wallpapers become the user's library the moment they land. Recording
	// them would make uninstalling the shell delete someone's pictures.
	clone, target := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(clone, "a.png"), "image")

	var recorded []fsx.Written
	env := &installer.Env{Home: fsx.Tree{
		Mode:   fsx.ModeApply,
		Record: func(w fsx.Written) { recorded = append(recorded, w) },
	}}

	if _, _, err := (wallpapersStep{}).copyImages(env, clone, target); err != nil {
		t.Fatalf("copyImages() error = %v", err)
	}
	if len(recorded) != 0 {
		t.Fatalf("wallpapers were recorded in the manifest: %v", recorded)
	}
}

func TestExtrasAreOffByDefaultAndSurviveFailure(t *testing.T) {
	cfg := installer.NewConfig()
	for _, id := range []installer.OptionID{
		installer.OptWallpapers, installer.OptIconTheme,
		installer.OptMascot, installer.OptSDDMTheme,
	} {
		if cfg.Effective(id) {
			t.Errorf("%s is on by default; every extra is a large download and must be opted into", id)
		}
	}

	for _, step := range []installer.Step{
		newWallpapersStep(), newIconThemeStep(), newMascotStep(), newSDDMThemeStep(),
	} {
		optional, ok := step.(interface{ Optional() bool })
		if !ok || !optional.Optional() {
			t.Errorf("%s is not optional; a dead mirror would fail the whole install", step.ID())
		}
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
