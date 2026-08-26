package system

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// PayloadDir is the checkout subdirectory holding everything that ships to a
// user's machine. The repository root above it carries only the things a
// developer needs — the installer's own source, packaging, documentation — none
// of which a running shell has any use for.
const PayloadDir = "src"

// repoMarkers are the paths that together identify an Illogical-mango checkout: the shell
// entry point, the dotfiles it installs alongside itself, and the version it
// reports. All must be present, so a bare directory named "ilmango" is never
// mistaken for one.
//
// These are what the installer actually reads. They deliberately do not
// include the shell installer under sdata/, which this program replaced.
var repoMarkers = []string{
	filepath.Join(PayloadDir, "shell.qml"),
	filepath.Join(PayloadDir, "dots"),
	filepath.Join(PayloadDir, "VERSION"),
}

// ErrRepoNotFound is returned when no Illogical-mango checkout could be located.
var ErrRepoNotFound = errors.New("Illogical-mango repository not found")

// Repo is a located Illogical-mango checkout.
type Repo struct {
	// Root is the checkout itself: what git reports on, and what the install
	// manifest records so a later run can find the same source.
	Root string
	// Payload is the subtree that actually gets installed. Every step that
	// copies a file into the user's home reads from here, never from Root.
	Payload string
	Version string
	Commit  string
	Branch  string
}

// FindRepo locates the checkout to install from. It prefers an explicit hint,
// then the working directory and its ancestors, then the directory holding the
// running binary and its ancestors.
func FindRepo(hint string) (Repo, error) {
	var candidates []string
	if hint != "" {
		candidates = append(candidates, hint)
	}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, wd)
	}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Dir(exe))
	}

	for _, start := range candidates {
		if root, ok := walkUpToRepo(start); ok {
			return describeRepo(root), nil
		}
	}
	return Repo{}, fmt.Errorf("%w: looked in %s", ErrRepoNotFound, strings.Join(candidates, ", "))
}

// walkUpToRepo climbs from dir towards the filesystem root looking for markers.
func walkUpToRepo(dir string) (string, bool) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return "", false
	}
	for {
		if isRepo(dir) {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func isRepo(dir string) bool {
	for _, marker := range repoMarkers {
		if _, err := os.Stat(filepath.Join(dir, marker)); err != nil {
			return false
		}
	}
	return true
}

// describeRepo reads the version and git metadata. Missing git information is
// not an error — a tarball install has no .git directory.
func describeRepo(root string) Repo {
	repo := Repo{
		Root:    root,
		Payload: filepath.Join(root, PayloadDir),
		Version: "unknown", Commit: "unknown", Branch: "",
	}

	if data, err := os.ReadFile(filepath.Join(repo.Payload, "VERSION")); err == nil {
		if v := strings.TrimSpace(string(data)); v != "" {
			repo.Version = v
		}
	}
	if out, err := gitOutput(root, "rev-parse", "--short", "HEAD"); err == nil {
		repo.Commit = out
	}
	if out, err := gitOutput(root, "rev-parse", "--abbrev-ref", "HEAD"); err == nil {
		repo.Branch = out
	}
	return repo
}

func gitOutput(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
