// Package fsx is the installer's single gateway to modifying the filesystem.
//
// It exists for the same reasons as package run: every write is logged, every
// write can be simulated, and every destination can be redirected under a test
// root, so the installer can be exercised end to end without touching a real
// home directory. Nothing else may write files directly.
package fsx

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Mode decides whether writes actually happen.
type Mode int

const (
	// ModePlan logs intended writes and performs none.
	ModePlan Mode = iota
	// ModeApply performs writes for real.
	ModeApply
)

// Tree is a destination filesystem. The zero Tree is in ModePlan and writes
// nothing, so an unconfigured Tree cannot damage a machine.
type Tree struct {
	// Root prefixes every destination path. An empty Root means the real
	// filesystem; tests set it to a temporary directory.
	Root string
	Mode Mode
	// Log receives one line per operation. It may be nil.
	Log func(string)
	// Backup, when set, preserves whatever a write is about to replace.
	Backup *Backup

	// Record, when set, is told about every path written. It is what lets an
	// uninstall know exactly what an install created, rather than guessing
	// from a hardcoded list of paths that drifts out of date.
	Record func(Written)
}

// Written describes one path an installation created.
type Written struct {
	// Path is the logical destination, without any test root prefix.
	Path string
	// Sum is the SHA-256 of the content written, hex encoded. An uninstall
	// compares it against what is on disk to tell an untouched file from one
	// the user has since edited.
	Sum string
	// Link is the target, for symbolic links; Sum is empty for those.
	Link string
}

// resolve maps a destination path into the tree.
func (t Tree) resolve(path string) string {
	if t.Root == "" {
		return path
	}
	return filepath.Join(t.Root, path)
}

// EnsureDir creates a directory and any missing parents.
func (t Tree) EnsureDir(path string, perm fs.FileMode) error {
	target := t.resolve(path)
	if t.Mode == ModePlan {
		t.logf("would create directory %s", path)
		return nil
	}
	if err := os.MkdirAll(target, perm); err != nil {
		return fmt.Errorf("creating %s: %w", path, err)
	}
	return nil
}

// WriteFile writes data, backing up and replacing anything already there.
func (t Tree) WriteFile(path string, data []byte, perm fs.FileMode) error {
	target := t.resolve(path)

	if t.Mode == ModePlan {
		t.logf("would write %s (%d bytes)", path, len(data))
		return nil
	}
	if err := t.preserve(path, target); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("creating parent of %s: %w", path, err)
	}

	// Write to a sibling and rename, so an interrupted install never leaves a
	// half-written config behind.
	tmp, err := os.CreateTemp(filepath.Dir(target), ".ilmango-*")
	if err != nil {
		return fmt.Errorf("staging %s: %w", path, err)
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("writing %s: %w", path, err)
	}
	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		return fmt.Errorf("setting mode on %s: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing %s: %w", path, err)
	}
	if err := os.Rename(tmp.Name(), target); err != nil {
		return fmt.Errorf("installing %s: %w", path, err)
	}

	t.record(Written{Path: path, Sum: Sum(data)})
	t.logf("wrote %s", path)
	return nil
}

// Sum returns the hex-encoded SHA-256 of data.
func Sum(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

// SumFile returns the hex-encoded SHA-256 of a file's current content.
func SumFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return Sum(data), nil
}

func (t Tree) record(w Written) {
	if t.Record != nil {
		t.Record(w)
	}
}

// CopyFile copies src from the real filesystem to dst inside the tree.
func (t Tree) CopyFile(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("reading %s: %w", src, err)
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("reading %s: %w", src, err)
	}
	return t.WriteFile(dst, data, info.Mode().Perm())
}

// CopyTree mirrors the directory src into dst, creating directories as needed.
func (t Tree) CopyTree(src, dst string) error {
	return t.CopyTreeExcept(src, dst, nil)
}

// CopyTreeExcept mirrors src into dst, skipping any entry whose base name skip
// reports. A skipped directory is pruned along with everything beneath it.
//
// Symlinks are recreated rather than followed, so links within the source keep
// pointing where they were written to point.
func (t Tree) CopyTreeExcept(src, dst string, skip func(name string) bool) error {
	return filepath.WalkDir(src, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if skip != nil && path != src && skip(entry.Name()) {
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		switch {
		case entry.IsDir():
			info, err := entry.Info()
			if err != nil {
				return err
			}
			return t.EnsureDir(target, info.Mode().Perm())

		case entry.Type()&fs.ModeSymlink != 0:
			dest, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return t.Symlink(dest, target)

		default:
			return t.CopyFile(path, target)
		}
	})
}

// Symlink creates or replaces a symbolic link.
func (t Tree) Symlink(dest, path string) error {
	target := t.resolve(path)

	if t.Mode == ModePlan {
		t.logf("would link %s -> %s", path, dest)
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("creating parent of %s: %w", path, err)
	}
	if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("replacing %s: %w", path, err)
	}
	if err := os.Symlink(dest, target); err != nil {
		return fmt.Errorf("linking %s: %w", path, err)
	}

	t.record(Written{Path: path, Link: dest})
	t.logf("linked %s -> %s", path, dest)
	return nil
}

// RemovalOutcome says what happened to a path an uninstall considered.
type RemovalOutcome int

const (
	// Removed means the path was deleted.
	Removed RemovalOutcome = iota
	// Absent means it was already gone.
	Absent
	// Modified means the content no longer matches what was installed, so it
	// was left alone: it holds the user's own edits.
	Modified
)

// Remove deletes a path that an installation created.
//
// wantSum is the checksum recorded at install time. If the file on disk no
// longer matches it, the file is kept: an uninstaller that deletes work the
// user did after installing is worse than one that leaves a file behind. Pass
// an empty wantSum to remove regardless.
func (t Tree) Remove(path, wantSum string) (RemovalOutcome, error) {
	target := t.resolve(path)

	info, err := os.Lstat(target)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Absent, nil
		}
		return Absent, fmt.Errorf("inspecting %s: %w", path, err)
	}

	if wantSum != "" && info.Mode().IsRegular() {
		got, err := SumFile(target)
		if err != nil {
			return Absent, fmt.Errorf("checksumming %s: %w", path, err)
		}
		if got != wantSum {
			t.logf("kept %s (changed since it was installed)", path)
			return Modified, nil
		}
	}

	if t.Mode == ModePlan {
		t.logf("would remove %s", path)
		return Removed, nil
	}

	if err := t.preserve(path, target); err != nil {
		return Absent, err
	}
	if err := os.Remove(target); err != nil {
		return Absent, fmt.Errorf("removing %s: %w", path, err)
	}

	t.logf("removed %s", path)
	return Removed, nil
}

// RemoveAll deletes a directory and everything under it. It is for discarding
// something the installer itself created and knows to be unusable — a broken
// virtualenv — never for user data.
func (t Tree) RemoveAll(path string) error {
	if t.Mode == ModePlan {
		t.logf("would remove directory tree %s", path)
		return nil
	}
	if err := os.RemoveAll(t.resolve(path)); err != nil {
		return fmt.Errorf("removing %s: %w", path, err)
	}

	t.logf("removed directory tree %s", path)
	return nil
}

// PruneEmptyDirs removes dir and its parents while they are empty, stopping
// before it would touch stop. It tidies up the directory skeleton an install
// created without ever deleting a directory that still holds something.
func (t Tree) PruneEmptyDirs(dir, stop string) {
	for dir != stop && dir != "/" && dir != "." {
		target := t.resolve(dir)

		entries, err := os.ReadDir(target)
		if err != nil || len(entries) > 0 {
			return
		}

		if t.Mode == ModePlan {
			t.logf("would remove empty directory %s", dir)
		} else if err := os.Remove(target); err != nil {
			return
		} else {
			t.logf("removed empty directory %s", dir)
		}

		dir = filepath.Dir(dir)
	}
}

// Unrecorded returns a copy of the tree that does not report its writes.
//
// It is for files the installer edits but does not own — a compositor config
// it appends one line to. Recording such a file would list it as ours, and an
// uninstall would then delete the user's configuration wholesale.
func (t Tree) Unrecorded() Tree {
	t.Record = nil
	return t
}

// Chmod sets a path's permissions. Copying preserves the source mode, but a
// file that must be executable is worth stating outright rather than trusting
// whatever bit survived a checkout, an archive or a file share.
func (t Tree) Chmod(path string, perm fs.FileMode) error {
	if t.Mode == ModePlan {
		t.logf("would set mode %o on %s", perm, path)
		return nil
	}
	if err := os.Chmod(t.resolve(path), perm); err != nil {
		return fmt.Errorf("setting mode on %s: %w", path, err)
	}
	return nil
}

// ReadFile reads a path inside the tree. A missing file reads as empty, since
// every caller here is about to create it.
func (t Tree) ReadFile(path string) ([]byte, error) {
	data, err := os.ReadFile(t.resolve(path))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return data, err
}

// Exists reports whether a path is present inside the tree.
func (t Tree) Exists(path string) bool {
	_, err := os.Lstat(t.resolve(path))
	return err == nil
}

// preserve hands an existing file to the backup set before it is replaced.
func (t Tree) preserve(logical, actual string) error {
	if t.Backup == nil {
		return nil
	}
	return t.Backup.Preserve(logical, actual)
}

func (t Tree) logf(format string, args ...any) {
	if t.Log != nil {
		t.Log(fmt.Sprintf(format, args...))
	}
}

// Backup collects the files an installation replaces, so a user can get their
// previous configuration back.
type Backup struct {
	// Dir is where copies are written. It is created lazily, so a run that
	// replaces nothing leaves no empty directory behind.
	Dir string

	created bool
	count   int
}

// NewBackup returns a backup set rooted at a timestamped directory under base.
func NewBackup(base string) *Backup {
	return &Backup{Dir: filepath.Join(base, "backup-"+time.Now().Format("20060102-150405"))}
}

// Preserve copies actual into the backup set, keeping its logical path so the
// layout of the backup mirrors the layout of what it replaced.
func (b *Backup) Preserve(logical, actual string) error {
	info, err := os.Lstat(actual)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil // nothing was there; nothing to preserve
		}
		return fmt.Errorf("inspecting %s: %w", logical, err)
	}
	if info.IsDir() {
		return nil // directories are merged into, never replaced wholesale
	}
	if !info.Mode().IsRegular() {
		// Symlinks, sockets and devices have no content worth copying, and a
		// dangling link would fail to open at all. Recreating one is trivial;
		// aborting an uninstall over it is not acceptable.
		return nil
	}

	dest := filepath.Join(b.Dir, strings.TrimPrefix(logical, string(filepath.Separator)))
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("preparing backup for %s: %w", logical, err)
	}

	src, err := os.Open(actual)
	if err != nil {
		return fmt.Errorf("reading %s for backup: %w", logical, err)
	}
	defer src.Close()

	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return fmt.Errorf("writing backup of %s: %w", logical, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, src); err != nil {
		return fmt.Errorf("copying %s to backup: %w", logical, err)
	}

	b.created = true
	b.count++
	return nil
}

// Count is how many files were preserved.
func (b *Backup) Count() int { return b.count }

// Path returns the backup directory, or an empty string if nothing was kept.
func (b *Backup) Path() string {
	if b == nil || !b.created {
		return ""
	}
	return b.Dir
}
