package installer

import (
	"path/filepath"
	"testing"

	"ilmango/internal/fsx"
)

func TestManifestAddReplacesRatherThanDuplicates(t *testing.T) {
	m := NewManifest("1.0.0", "/repo")

	m.Add(fsx.Written{Path: "/a.conf", Sum: "first"})
	m.Add(fsx.Written{Path: "/a.conf", Sum: "second"})

	if m.Len() != 1 {
		t.Fatalf("Len() = %d, want 1 after rewriting the same path", m.Len())
	}
	if m.Entries[0].Sum != "second" {
		t.Errorf("sum = %q, want the most recent write", m.Entries[0].Sum)
	}
}

func TestRemovalOrderIsDeepestFirst(t *testing.T) {
	m := NewManifest("1.0.0", "/repo")
	m.Add(fsx.Written{Path: "/a/b.conf"})
	m.Add(fsx.Written{Path: "/a/b/c/d.conf"})
	m.Add(fsx.Written{Path: "/a/b/c.conf"})

	order := m.RemovalOrder()

	// A directory can only be pruned once what is inside it is gone.
	for i := 1; i < len(order); i++ {
		if depth(order[i-1].Path) < depth(order[i].Path) {
			t.Fatalf("removal order is not deepest-first: %v", order)
		}
	}
}

func TestManifestRoundTripsThroughDisk(t *testing.T) {
	root := t.TempDir()
	tree := fsx.Tree{Root: root, Mode: fsx.ModeApply}

	original := NewManifest("2.0.0", "/repo")
	original.Add(fsx.Written{Path: "/home/u/a.conf", Sum: "abc"})
	original.Add(fsx.Written{Path: "/home/u/link", Link: "a.conf"})

	if err := original.Save(tree, "/state/manifest.json"); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := LoadManifest(filepath.Join(root, "state/manifest.json"))
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}

	if loaded.Release != "2.0.0" || loaded.Len() != 2 {
		t.Errorf("round trip lost data: %+v", loaded)
	}
	if loaded.Entries[1].Link != "a.conf" {
		t.Errorf("symlink target was not preserved: %+v", loaded.Entries[1])
	}
}

func TestSaveDoesNotRecordItself(t *testing.T) {
	root := t.TempDir()
	m := NewManifest("1.0.0", "/repo")

	// A tree that would append every write back into the manifest.
	tree := fsx.Tree{Root: root, Mode: fsx.ModeApply, Record: m.Add}

	if err := m.Save(tree, "/state/manifest.json"); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if m.Len() != 0 {
		t.Errorf("saving the manifest recorded itself as an entry: %+v", m.Entries)
	}
}

func TestLoadManifestRejectsNewerFormat(t *testing.T) {
	root := t.TempDir()
	tree := fsx.Tree{Root: root, Mode: fsx.ModeApply}

	future := NewManifest("1.0.0", "/repo")
	future.Version = ManifestVersion + 1
	if err := future.Save(tree, "/m.json"); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadManifest(filepath.Join(root, "m.json")); err == nil {
		t.Error("a manifest from a newer installer must be refused, not misread")
	}
}
