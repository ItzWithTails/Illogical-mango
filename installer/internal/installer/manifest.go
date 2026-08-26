package installer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"ilmango/internal/fsx"
)

// ManifestVersion is bumped when the on-disk shape changes incompatibly.
const ManifestVersion = 1

// Manifest is the record of what an installation put on the machine.
//
// It is the difference between an uninstaller that knows and one that guesses.
// Every entry carries the checksum of what was written, so removal can tell an
// untouched file from one the user has since edited and leave the latter alone.
type Manifest struct {
	Version     int             `json:"version"`
	InstalledAt string          `json:"installed_at"`
	Release     string          `json:"release"`
	RepoPath    string          `json:"repo_path"`
	Entries     []ManifestEntry `json:"entries"`
}

// ManifestEntry is one path an installation created.
type ManifestEntry struct {
	Path string `json:"path"`
	Sum  string `json:"sum,omitempty"`
	Link string `json:"link,omitempty"`
}

// NewManifest starts an empty record for a release.
func NewManifest(release, repoPath string) *Manifest {
	return &Manifest{
		Version:     ManifestVersion,
		InstalledAt: time.Now().Format(time.RFC3339),
		Release:     release,
		RepoPath:    repoPath,
	}
}

// Add records one written path. Re-writing a path replaces its entry rather
// than duplicating it, so reinstalling does not grow the manifest without end.
func (m *Manifest) Add(w fsx.Written) {
	for i, existing := range m.Entries {
		if existing.Path == w.Path {
			m.Entries[i] = ManifestEntry{Path: w.Path, Sum: w.Sum, Link: w.Link}
			return
		}
	}
	m.Entries = append(m.Entries, ManifestEntry{Path: w.Path, Sum: w.Sum, Link: w.Link})
}

// Len is how many paths were recorded.
func (m *Manifest) Len() int { return len(m.Entries) }

// RemovalOrder returns the entries deepest-first, so a directory is only
// considered once everything inside it has been dealt with.
func (m *Manifest) RemovalOrder() []ManifestEntry {
	out := make([]ManifestEntry, len(m.Entries))
	copy(out, m.Entries)
	sort.Slice(out, func(i, j int) bool {
		di, dj := depth(out[i].Path), depth(out[j].Path)
		if di != dj {
			return di > dj
		}
		return out[i].Path > out[j].Path
	})
	return out
}

func depth(path string) int {
	n := 0
	for _, r := range path {
		if r == filepath.Separator {
			n++
		}
	}
	return n
}

// Save writes the manifest through the tree, so it honours dry run like every
// other write. It deliberately does not record itself as an entry.
func (m *Manifest) Save(tree fsx.Tree, path string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding manifest: %w", err)
	}

	// Saving must not append the manifest to itself.
	tree.Record = nil
	return tree.WriteFile(path, append(data, '\n'), 0o644)
}

// LoadManifest reads a manifest from the real filesystem.
func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	if m.Version > ManifestVersion {
		return nil, fmt.Errorf("%s was written by a newer installer (format %d)", path, m.Version)
	}
	return &m, nil
}
