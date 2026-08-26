package fsx

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPlanModeWritesNothing(t *testing.T) {
	root := t.TempDir()
	var logged int
	tree := Tree{Root: root, Mode: ModePlan, Log: func(string) { logged++ }}

	if err := tree.WriteFile("/etc/ilmango.conf", []byte("data"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := tree.EnsureDir("/etc/ilmango", 0o755); err != nil {
		t.Fatalf("EnsureDir: %v", err)
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("plan mode created %d entries, want none", len(entries))
	}
	if logged == 0 {
		t.Error("plan mode should still describe what it would do")
	}
}

func TestApplyModeWritesUnderRoot(t *testing.T) {
	root := t.TempDir()
	tree := Tree{Root: root, Mode: ModeApply}

	if err := tree.WriteFile("/home/user/.config/a.conf", []byte("hello"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(root, "home/user/.config/a.conf"))
	if err != nil {
		t.Fatalf("reading written file: %v", err)
	}
	if string(got) != "hello" {
		t.Errorf("content = %q, want %q", got, "hello")
	}
}

func TestBackupPreservesReplacedFile(t *testing.T) {
	root := t.TempDir()
	backupBase := t.TempDir()

	// Seed an existing config that the install is about to replace.
	existing := filepath.Join(root, "home/user/.config/a.conf")
	if err := os.MkdirAll(filepath.Dir(existing), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(existing, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}

	backup := NewBackup(backupBase)
	tree := Tree{Root: root, Mode: ModeApply, Backup: backup}

	if err := tree.WriteFile("/home/user/.config/a.conf", []byte("replacement"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if backup.Count() != 1 {
		t.Fatalf("backup kept %d files, want 1", backup.Count())
	}
	kept, err := os.ReadFile(filepath.Join(backup.Path(), "home/user/.config/a.conf"))
	if err != nil {
		t.Fatalf("reading backup: %v", err)
	}
	if string(kept) != "original" {
		t.Errorf("backup content = %q, want the original", kept)
	}
}

func TestBackupIgnoresAbsentFile(t *testing.T) {
	root := t.TempDir()
	backup := NewBackup(t.TempDir())
	tree := Tree{Root: root, Mode: ModeApply, Backup: backup}

	if err := tree.WriteFile("/home/user/new.conf", []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if backup.Count() != 0 {
		t.Errorf("nothing was replaced, but %d files were backed up", backup.Count())
	}
	if backup.Path() != "" {
		t.Error("an empty backup should not report a path")
	}
}

func TestCopyTreeMirrorsStructure(t *testing.T) {
	src := t.TempDir()
	root := t.TempDir()

	if err := os.MkdirAll(filepath.Join(src, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "nested/file.txt"), []byte("v"), 0o644); err != nil {
		t.Fatal(err)
	}

	tree := Tree{Root: root, Mode: ModeApply}
	if err := tree.CopyTree(src, "/dest"); err != nil {
		t.Fatalf("CopyTree: %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, "dest/nested/file.txt")); err != nil {
		t.Errorf("copied tree is missing the nested file: %v", err)
	}
}

func TestZeroTreeIsInert(t *testing.T) {
	// The zero value must not write to the real filesystem.
	var tree Tree
	if tree.Mode != ModePlan {
		t.Error("the zero Tree must default to plan mode")
	}
}

func TestCopyTreeExceptPrunesSkippedDirectories(t *testing.T) {
	src := t.TempDir()
	root := t.TempDir()

	// A development artefact directory that must never be installed.
	if err := os.MkdirAll(filepath.Join(src, ".claude/deep"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, ".claude/deep/secret.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "keep.qml"), []byte("y"), 0o644); err != nil {
		t.Fatal(err)
	}

	tree := Tree{Root: root, Mode: ModeApply}
	skip := func(name string) bool { return name == ".claude" }

	if err := tree.CopyTreeExcept(src, "/dest", skip); err != nil {
		t.Fatalf("CopyTreeExcept: %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, "dest/keep.qml")); err != nil {
		t.Errorf("wanted file was not copied: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "dest/.claude")); err == nil {
		t.Error("a skipped directory was copied anyway")
	}
}

func TestCopyTreeRecreatesSymlinks(t *testing.T) {
	src := t.TempDir()
	root := t.TempDir()

	if err := os.WriteFile(filepath.Join(src, "real.txt"), []byte("v"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("real.txt", filepath.Join(src, "link.txt")); err != nil {
		t.Fatal(err)
	}

	tree := Tree{Root: root, Mode: ModeApply}
	if err := tree.CopyTree(src, "/dest"); err != nil {
		t.Fatalf("CopyTree: %v", err)
	}

	info, err := os.Lstat(filepath.Join(root, "dest/link.txt"))
	if err != nil {
		t.Fatalf("link was not created: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("the link was followed and copied as a regular file")
	}
}

func TestRemoveKeepsFilesTheUserChanged(t *testing.T) {
	root := t.TempDir()
	tree := Tree{Root: root, Mode: ModeApply}

	if err := tree.WriteFile("/home/u/a.conf", []byte("as installed"), 0o644); err != nil {
		t.Fatal(err)
	}
	installedSum := Sum([]byte("as installed"))

	// The user edits the file after installing.
	if err := os.WriteFile(filepath.Join(root, "home/u/a.conf"), []byte("my own edits"), 0o644); err != nil {
		t.Fatal(err)
	}

	outcome, err := tree.Remove("/home/u/a.conf", installedSum)
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if outcome != Modified {
		t.Errorf("outcome = %v, want Modified", outcome)
	}
	if _, err := os.Stat(filepath.Join(root, "home/u/a.conf")); err != nil {
		t.Error("an edited file must survive an uninstall")
	}
}

func TestRemoveDeletesUntouchedFile(t *testing.T) {
	root := t.TempDir()
	tree := Tree{Root: root, Mode: ModeApply}

	if err := tree.WriteFile("/home/u/a.conf", []byte("as installed"), 0o644); err != nil {
		t.Fatal(err)
	}

	outcome, err := tree.Remove("/home/u/a.conf", Sum([]byte("as installed")))
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if outcome != Removed {
		t.Errorf("outcome = %v, want Removed", outcome)
	}
	if _, err := os.Stat(filepath.Join(root, "home/u/a.conf")); !os.IsNotExist(err) {
		t.Error("an untouched file should have been removed")
	}
}

func TestRemoveReportsAbsentPath(t *testing.T) {
	tree := Tree{Root: t.TempDir(), Mode: ModeApply}

	outcome, err := tree.Remove("/never/existed", "")
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if outcome != Absent {
		t.Errorf("outcome = %v, want Absent", outcome)
	}
}

func TestRecordCapturesWrites(t *testing.T) {
	var written []Written
	tree := Tree{Root: t.TempDir(), Mode: ModeApply, Record: func(w Written) { written = append(written, w) }}

	if err := tree.WriteFile("/a.conf", []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	if len(written) != 1 || written[0].Path != "/a.conf" {
		t.Fatalf("recorded %+v, want one entry for /a.conf", written)
	}
	if written[0].Sum != Sum([]byte("x")) {
		t.Error("the recorded checksum does not match what was written")
	}
}

func TestPruneEmptyDirsStopsAtBoundary(t *testing.T) {
	root := t.TempDir()
	tree := Tree{Root: root, Mode: ModeApply}

	if err := tree.EnsureDir("/home/u/.config/deep/nested", 0o755); err != nil {
		t.Fatal(err)
	}

	tree.PruneEmptyDirs("/home/u/.config/deep/nested", "/home/u")

	if _, err := os.Stat(filepath.Join(root, "home/u/.config")); !os.IsNotExist(err) {
		t.Error("empty directories below the boundary should have been pruned")
	}
	if _, err := os.Stat(filepath.Join(root, "home/u")); err != nil {
		t.Error("pruning must stop at the boundary directory")
	}
}

func TestBackupSkipsDanglingSymlink(t *testing.T) {
	root := t.TempDir()
	backup := NewBackup(t.TempDir())
	tree := Tree{Root: root, Mode: ModeApply, Backup: backup}

	// A link to something that is not there — exactly what an uninstall meets
	// once it has already removed the target.
	linkPath := filepath.Join(root, "home/u/link")
	if err := os.MkdirAll(filepath.Dir(linkPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("gone.txt", linkPath); err != nil {
		t.Fatal(err)
	}

	outcome, err := tree.Remove("/home/u/link", "")
	if err != nil {
		t.Fatalf("removing a dangling link must not fail: %v", err)
	}
	if outcome != Removed {
		t.Errorf("outcome = %v, want Removed", outcome)
	}
}
