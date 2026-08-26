package steps

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ilmango/internal/fsx"
	"ilmango/internal/installer"
	"ilmango/internal/run"
)

// Extras are the optional downloads the old shell installer hid behind a menu:
// art and themes that are pleasant to have and irrelevant to whether the shell
// works. They share three properties, which is why they live together here.
//
// Every one of them is off by default, because every one of them fetches tens
// or hundreds of megabytes. Every one is Optional, so a dead mirror costs the
// download and nothing else. And every one refuses to overwrite something the
// user already has, because an extra that clobbers your wallpapers is not an
// extra.
const (
	wallpapersRepo = "https://github.com/snowarch/iNiR-Walls.git"
	iconThemeRepo  = "https://bitbucket.org/dirn-typo/yet-another-monochrome-icon-set.git"
	iconThemeName  = "yet-another-monochrome-icon-set"
	mascotBaseURL  = "https://github.com/snowarch/inir-mascot/releases/latest/download"
)

// wallpaperExtensions are the image formats the pack ships. Four of the
// wallpapers are genuinely WebP, so extension alone cannot separate them from
// the gallery thumbnails — see wallpapersStep.Run.
var wallpaperExtensions = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".avif": true,
}

// wallpapersStep copies an optional wallpaper pack into the user's wallpapers.
type wallpapersStep struct{ base }

func newWallpapersStep() installer.Step {
	return wallpapersStep{base{
		id:     "wallpapers",
		title:  "Download the wallpaper pack",
		detail: "About 148 wallpapers, cloned once and then discarded.",
		applies: func(c installer.Config) bool {
			return c.Effective(installer.OptWallpapers)
		},
	}}
}

func (wallpapersStep) Optional() bool { return true }

func (s wallpapersStep) Run(ctx context.Context, env *installer.Env) error {
	if !run.Exists("git") {
		env.Note("git is not installed, so the wallpaper pack was not downloaded.")
		return nil
	}

	target := filepath.Join(pictureHome(), "Wallpapers")
	if err := env.Home.EnsureDir(target, 0o755); err != nil {
		return err
	}

	// The clone is a means, not a result: it goes to a temporary directory and
	// is deleted, so the user's disk holds images rather than a git history
	// several times their size.
	tmp, err := os.MkdirTemp("", "ilmango-walls-")
	if err != nil {
		return fmt.Errorf("preparing the download: %w", err)
	}
	defer os.RemoveAll(tmp)

	clone := filepath.Join(tmp, "walls")
	env.Detail("cloning " + wallpapersRepo)
	if err := env.Runner.Run(ctx, run.Command{
		Name:    "git",
		Args:    []string{"clone", "--depth", "1", wallpapersRepo, clone},
		Timeout: wallpaperBudget,
	}); err != nil {
		env.Note("The wallpaper pack could not be downloaded: " + err.Error())
		return nil
	}

	if env.Config.DryRun {
		// Nothing was cloned, so there is nothing to walk. Saying so is more
		// honest than reporting a copy that did not happen.
		env.Detail("would copy every image that is not already in " + target)
		return nil
	}

	copied, skipped, err := s.copyImages(env, clone, target)
	if err != nil {
		return err
	}
	env.Detail(fmt.Sprintf("%d copied, %d already present", copied, skipped))
	if copied == 0 && skipped == 0 {
		env.Note("The wallpaper pack downloaded but contained no images.")
	}
	return nil
}

// copyImages walks the clone and copies every wallpaper the user does not
// already have under that name.
func (s wallpapersStep) copyImages(env *installer.Env, clone, target string) (copied, skipped int, err error) {
	err = filepath.WalkDir(clone, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if !wallpaperExtensions[strings.ToLower(filepath.Ext(path))] {
			return nil
		}
		// The pack generates a 640px preview per wallpaper under images/thumbs/
		// with the same basename as the image it previews. Nothing but
		// directory order keeps a thumbnail from landing on top of the real
		// wallpaper, so the thumbnails are excluded by path.
		if strings.Contains(filepath.ToSlash(path), "/thumbs/") {
			return nil
		}

		dest := filepath.Join(target, filepath.Base(path))
		if info, statErr := os.Stat(dest); statErr == nil && info.Size() > 0 {
			skipped++
			return nil
		}
		// Wallpapers are the user's own library from the moment they land, so
		// they are not recorded in the manifest: uninstalling the shell must
		// not delete someone's pictures.
		if err := env.Home.Unrecorded().CopyFile(path, dest); err != nil {
			return err
		}
		copied++
		return nil
	})
	return copied, skipped, err
}

// iconThemeStep installs a monochrome icon theme beside the user's own.
type iconThemeStep struct{ base }

func newIconThemeStep() installer.Step {
	return iconThemeStep{base{
		id:     "icons",
		title:  "Install the monochrome icon theme",
		detail: "YAMIS, in your user icon directory. Your current theme is left alone.",
		applies: func(c installer.Config) bool {
			return c.Effective(installer.OptIconTheme)
		},
	}}
}

func (iconThemeStep) Optional() bool { return true }

func (s iconThemeStep) Run(ctx context.Context, env *installer.Env) error {
	if !run.Exists("git") {
		env.Note("git is not installed, so the icon theme was not downloaded.")
		return nil
	}

	dest := filepath.Join(dataHome(), "icons", iconThemeName)
	if err := env.Home.EnsureDir(filepath.Dir(dest), 0o755); err != nil {
		return err
	}

	switch {
	case dirExists(filepath.Join(dest, ".git")):
		// Already a checkout: fast-forward it rather than cloning again.
		env.Detail("updating " + dest)
		if err := env.Runner.Run(ctx, run.Command{
			Name: "git", Args: []string{"-C", dest, "pull", "--ff-only", "--quiet"},
			Timeout: iconBudget,
		}); err != nil {
			env.Note("The icon theme could not be updated; the copy you have was left as it is.")
		}
		return nil

	case dirExists(dest):
		// Something is there that git did not put there. It is not ours to
		// replace.
		env.Note(fmt.Sprintf("%s already exists and is not a checkout, so the icon theme was left alone.", dest))
		return nil
	}

	env.Detail("cloning " + iconThemeRepo)
	if err := env.Runner.Run(ctx, run.Command{
		Name:    "git",
		Args:    []string{"clone", "--depth", "1", "--quiet", iconThemeRepo, dest},
		Timeout: iconBudget,
	}); err != nil {
		env.Note("The icon theme could not be downloaded: " + err.Error())
		_ = os.RemoveAll(dest)
		return nil
	}

	if run.Exists("gtk-update-icon-cache") {
		_ = env.Runner.Run(ctx, run.Command{Name: "gtk-update-icon-cache", Args: []string{"-q", dest}})
	}
	env.NoteApplied("The monochrome icon theme is installed. Switch to it in Settings → Appearance if you want it.")
	return nil
}

// mascotStep downloads the mascot art pack into the installed shell.
type mascotStep struct{ base }

func newMascotStep() installer.Step {
	return mascotStep{base{
		id:     "mascot",
		title:  "Download the mascot art pack",
		detail: "354 poses and animations, verified against the published checksum.",
		applies: func(c installer.Config) bool {
			return c.Effective(installer.OptMascot)
		},
	}}
}

func (mascotStep) Optional() bool { return true }

func (s mascotStep) Run(ctx context.Context, env *installer.Env) error {
	if !run.Exists("curl") {
		env.Note("curl is not installed, so the mascot pack was not downloaded.")
		return nil
	}

	dest := filepath.Join(shellDir(), "assets", "images", "mascot")
	if err := env.Home.EnsureDir(dest, 0o755); err != nil {
		return err
	}

	tmp, err := os.MkdirTemp("", "ilmango-mascot-")
	if err != nil {
		return fmt.Errorf("preparing the download: %w", err)
	}
	defer os.RemoveAll(tmp)

	archive := filepath.Join(tmp, "mascot-pack.tar.gz")
	env.Detail("downloading the pack")
	if err := env.Runner.Run(ctx, run.Command{
		Name:    "curl",
		Args:    []string{"-fsSL", "--retry", "2", "-o", archive, mascotBaseURL + "/inir-mascot-pack.tar.gz"},
		Timeout: mascotBudget,
	}); err != nil {
		env.Note("The mascot pack could not be downloaded; the art you have was left as it is.")
		return nil
	}

	if env.Config.DryRun {
		env.Detail("would verify the checksum and unpack into " + dest)
		return nil
	}

	// The pack replaces live art, so it is verified before anything is
	// unpacked. A truncated download that still extracts would leave the
	// mascot half-drawn, which is worse than not downloading it at all.
	if ok, err := s.verify(ctx, env, tmp, archive); err != nil {
		return err
	} else if !ok {
		env.Note("The mascot pack did not match its published checksum, so it was discarded.")
		return nil
	}

	// Unpack to a staging directory and copy from there, rather than
	// extracting straight into the shell. The copy goes through the same
	// gateway as every other installed file, so the art lands in the manifest
	// and uninstalling removes it instead of leaving 32 MiB behind.
	stage := filepath.Join(tmp, "stage")
	if err := os.MkdirAll(stage, 0o755); err != nil {
		return fmt.Errorf("preparing the unpack: %w", err)
	}
	env.Detail("unpacking")
	if err := env.Runner.Run(ctx, run.Command{
		Name:    "tar",
		Args:    []string{"-xzf", archive, "-C", stage},
		Timeout: mascotBudget,
	}); err != nil {
		env.Note("The mascot pack downloaded but could not be unpacked: " + err.Error())
		return nil
	}

	env.Detail("installing into " + dest)
	if err := env.Home.CopyTree(stage, dest); err != nil {
		return err
	}
	env.NoteApplied("The mascot art is installed. Kira herself stays off until you enable her in Settings.")
	return nil
}

// verify checks the archive against the published SHA-256, when one is
// published. A missing checksum file is not a failure: it means the release
// does not offer one, and refusing the pack over that would be stricter than
// the project has ever been.
func (s mascotStep) verify(ctx context.Context, env *installer.Env, tmp, archive string) (bool, error) {
	sums := filepath.Join(tmp, "mascot-pack.sha256")
	if err := env.Runner.Run(ctx, run.Command{
		Name:    "curl",
		Args:    []string{"-fsSL", "--retry", "1", "-o", sums, mascotBaseURL + "/inir-mascot-pack.sha256"},
		Timeout: iconBudget,
	}); err != nil {
		env.Detail("no published checksum; skipping verification")
		return true, nil
	}

	published, err := os.ReadFile(sums)
	if err != nil {
		return true, nil
	}
	want, _, _ := strings.Cut(strings.TrimSpace(string(published)), " ")
	if want == "" {
		return true, nil
	}

	got, err := fsx.SumFile(archive)
	if err != nil {
		return false, fmt.Errorf("checksumming the mascot pack: %w", err)
	}
	if !strings.EqualFold(want, got) {
		env.Detail("checksum mismatch: expected " + want + ", got " + got)
		return false, nil
	}
	env.Detail("checksum verified")
	return true, nil
}

// sddmThemeStep applies the project's login screen theme.
type sddmThemeStep struct{ base }

func newSDDMThemeStep() installer.Step {
	return sddmThemeStep{base{
		id:     "sddm",
		title:  "Apply the login screen theme",
		detail: "Installs the ii-pixel theme and points SDDM at it.",
		applies: func(c installer.Config) bool {
			return c.Effective(installer.OptSDDMTheme)
		},
	}}
}

func (sddmThemeStep) Optional() bool { return true }

func (s sddmThemeStep) Run(ctx context.Context, env *installer.Env) error {
	if !run.Exists("sddm") {
		env.Note("SDDM is not installed, so the login screen theme was skipped.")
		return nil
	}

	script := filepath.Join(env.Repo.Payload, "scripts", "sddm", "install-pixel-sddm.sh")
	if _, err := os.Stat(script); err != nil {
		env.Note("The login screen theme installer is missing from this checkout, so it was skipped.")
		return nil
	}

	// The theme lives outside the home directory and rewrites the display
	// manager's configuration, which is why this option is off by default and
	// marked as touching the system on the review screen.
	env.Detail("running " + script)
	return env.Runner.Run(ctx, run.Command{
		Name:       "bash",
		Args:       []string{script},
		Privileged: true,
		Env:        []string{"ILMANGO_SDDM_AUTO_APPLY=yes"},
		Timeout:    iconBudget,
	})
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// Time budgets for the optional downloads. They are separate from the package
// budgets because these are single transfers over one connection, not a
// package manager working through a list: a wallpaper pack that has not
// finished in this long is not going to.
const (
	wallpaperBudget = 45 * time.Minute
	mascotBudget    = 20 * time.Minute
	iconBudget      = 15 * time.Minute
)

// pictureHome resolves the user's pictures directory, honouring XDG.
func pictureHome() string {
	if dir := os.Getenv("XDG_PICTURES_DIR"); dir != "" {
		return dir
	}
	return filepath.Join(home(), "Pictures")
}
