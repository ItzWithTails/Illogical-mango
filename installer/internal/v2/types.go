package v2

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const ManifestVersion = 2

type Operation string

const (
	Install   Operation = "install"
	Update    Operation = "update"
	Uninstall Operation = "uninstall"
	Rollback  Operation = "rollback"
	Status    Operation = "status"
)

func (o Operation) Verb() string {
	switch o {
	case Update:
		return "Update"
	case Uninstall:
		return "Uninstall"
	case Rollback:
		return "Rollback"
	case Status:
		return "Inspect"
	default:
		return "Install"
	}
}

type Preset string

const (
	Minimal     Preset = "minimal"
	Recommended Preset = "recommended"
	Full        Preset = "full"
)

type ConflictPolicy string

const (
	Preserve ConflictPolicy = "preserve"
	Replace  ConflictPolicy = "replace"
)

// Config contains user intent only. Paths and detected host state live in Plan.
type Config struct {
	Operation      Operation
	Preset         Preset
	ConflictPolicy ConflictPolicy
	KeyboardLayout string
	Language       string
	Packages       bool
	SystemUpgrade  bool
	MangoHook      bool
	DryRun         bool
	Yes            bool
	Verbose        bool
	NoColor        bool
	Root           string
	Home           string
	Repo           string
	FailAfter      int // test-only fault injection; zero disables it
}

func DefaultConfig() Config {
	home, _ := os.UserHomeDir()
	return Config{
		Operation:      Install,
		Preset:         Recommended,
		ConflictPolicy: Preserve,
		KeyboardLayout: "system",
		Language:       "auto",
		Packages:       true,
		MangoHook:      true,
		Home:           home,
	}
}

func (c Config) Validate() error {
	if c.Home == "" || !filepath.IsAbs(c.Home) {
		return fmt.Errorf("home must be an absolute path")
	}
	if filepath.Clean(c.Home) == string(filepath.Separator) {
		return fmt.Errorf("refusing to use the filesystem root as home")
	}
	if c.Root != "" && !filepath.IsAbs(c.Root) {
		return fmt.Errorf("--root must be an absolute path")
	}
	if c.Root != "" && filepath.Clean(c.Root) == string(filepath.Separator) {
		return fmt.Errorf("refusing --root / because it is not a sandbox")
	}
	switch c.Operation {
	case Install, Update, Uninstall, Rollback, Status:
	default:
		return fmt.Errorf("unknown operation %q", c.Operation)
	}
	switch c.Preset {
	case Minimal, Recommended, Full:
	default:
		return fmt.Errorf("unknown preset %q", c.Preset)
	}
	switch c.ConflictPolicy {
	case Preserve, Replace:
	default:
		return fmt.Errorf("unknown conflict policy %q", c.ConflictPolicy)
	}
	if strings.TrimSpace(c.KeyboardLayout) == "" {
		return fmt.Errorf("keyboard layout cannot be empty")
	}
	for _, r := range c.KeyboardLayout {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == ',' || r == '_' || r == '-') {
			return fmt.Errorf("keyboard layout contains unsafe character %q", r)
		}
	}
	if c.Language != "auto" && c.Language != "en" && c.Language != "ru" {
		return fmt.Errorf("language must be auto, en or ru")
	}
	if c.Root != "" && (c.Packages || c.SystemUpgrade) {
		return fmt.Errorf("--root is a filesystem sandbox: packages and system upgrade must be disabled")
	}
	if c.SystemUpgrade && !c.Packages {
		return fmt.Errorf("system upgrade requires package installation")
	}
	return nil
}

func (c Config) ConfigHome() string { return filepath.Join(c.Home, ".config") }
func (c Config) DataHome() string   { return filepath.Join(c.Home, ".local", "share") }
func (c Config) StateHome() string  { return filepath.Join(c.Home, ".local", "state") }
func (c Config) BinHome() string    { return filepath.Join(c.Home, ".local", "bin") }
func (c Config) ShellDir() string   { return filepath.Join(c.ConfigHome(), "quickshell", "ilmango") }

func (c Config) CommandLine() string {
	args := []string{"ilmango-installer-v2", string(c.Operation), "--home", shellQuote(c.Home)}
	if c.Repo != "" && (c.Operation == Install || c.Operation == Update) {
		args = append(args, "--repo", shellQuote(c.Repo))
	}
	if c.Root != "" {
		args = append(args, "--root", shellQuote(c.Root))
	}
	if c.Operation == Install || c.Operation == Update {
		args = append(args, "--preset", string(c.Preset), "--conflict", string(c.ConflictPolicy),
			"--layout", shellQuote(c.KeyboardLayout), "--language", c.Language)
		if !c.Packages {
			args = append(args, "--no-packages")
		}
		if !c.MangoHook {
			args = append(args, "--mango=false")
		}
		if c.SystemUpgrade {
			args = append(args, "--system-upgrade")
		}
	}
	if c.DryRun {
		args = append(args, "--dry-run")
	}
	args = append(args, "--yes")
	if c.NoColor {
		args = append(args, "--no-color")
	}
	return strings.Join(args, " ")
}

func shellQuote(value string) string {
	if value != "" && !strings.ContainsAny(value, " \t\n'\"$`\\!&;|<>()[]{}*?") {
		return value
	}
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

// Resolve is the only mapping from logical host paths into a redirected root.
func (c Config) Resolve(path string) (string, error) {
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("refusing non-absolute destination %q", path)
	}
	clean := filepath.Clean(path)
	home := filepath.Clean(c.Home)
	rel, err := filepath.Rel(home, clean)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("refusing destination outside home: %s", path)
	}
	if c.Root == "" {
		if err := checkDestinationParents(home, filepath.Dir(clean), false); err != nil {
			return "", err
		}
		return clean, nil
	}
	actual := filepath.Join(c.Root, strings.TrimPrefix(clean, string(filepath.Separator)))
	if err := checkDestinationParents(filepath.Clean(c.Root), filepath.Dir(actual), true); err != nil {
		return "", err
	}
	return actual, nil
}

func checkDestinationParents(base, parent string, sandbox bool) error {
	rel, err := filepath.Rel(base, parent)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("destination parent escapes managed root: %s", parent)
	}
	resolvedBase := base
	if evaluated, err := filepath.EvalSymlinks(base); err == nil {
		resolvedBase = evaluated
	}
	current := base
	if rel == "." {
		return nil
	}
	for _, component := range strings.Split(rel, string(filepath.Separator)) {
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("inspecting destination parent %s: %w", current, err)
		}
		if info.Mode()&os.ModeSymlink == 0 {
			continue
		}
		if sandbox {
			return fmt.Errorf("refusing symlinked parent inside --root sandbox: %s", current)
		}
		resolved, err := filepath.EvalSymlinks(current)
		if err != nil {
			return fmt.Errorf("resolving destination parent %s: %w", current, err)
		}
		inside, err := filepath.Rel(resolvedBase, resolved)
		if err != nil || inside == ".." || strings.HasPrefix(inside, ".."+string(filepath.Separator)) {
			return fmt.Errorf("symlinked destination parent escapes home: %s -> %s", current, resolved)
		}
	}
	return nil
}

type FileKind string

const (
	Regular FileKind = "file"
	Symlink FileKind = "symlink"
)

type FileRecord struct {
	Path string   `json:"path"`
	Kind FileKind `json:"kind"`
	Sum  string   `json:"sum,omitempty"`
	Link string   `json:"link,omitempty"`
	Mode uint32   `json:"mode,omitempty"`
}

func (r FileRecord) Matches(sum, link string) bool {
	if r.Kind == Symlink {
		return r.Link == link
	}
	return r.Sum == sum
}

func (r FileRecord) MatchesState(kind FileKind, sum, link string, mode os.FileMode) bool {
	if kind != r.Kind || !r.Matches(sum, link) {
		return false
	}
	return r.Kind == Symlink || r.Mode == 0 || uint32(mode.Perm()) == r.Mode
}

type Manifest struct {
	Version     int          `json:"version"`
	Release     string       `json:"release"`
	Commit      string       `json:"commit"`
	Repo        string       `json:"repo"`
	InstalledAt string       `json:"installed_at"`
	Files       []FileRecord `json:"files"`
	Seeded      []FileRecord `json:"seeded_user_files,omitempty"`
}

func NewManifest(release, commit, repo string) *Manifest {
	return &Manifest{Version: ManifestVersion, Release: release, Commit: commit, Repo: repo, InstalledAt: time.Now().Format(time.RFC3339)}
}

func (m *Manifest) Map() map[string]FileRecord {
	out := make(map[string]FileRecord, len(m.Files))
	for _, f := range m.Files {
		out[f.Path] = f
	}
	return out
}

func (m *Manifest) Set(files map[string]FileRecord) {
	m.Files = m.Files[:0]
	for _, f := range files {
		m.Files = append(m.Files, f)
	}
	sort.Slice(m.Files, func(i, j int) bool { return m.Files[i].Path < m.Files[j].Path })
}

func (m *Manifest) SeededMap() map[string]FileRecord {
	out := make(map[string]FileRecord, len(m.Seeded))
	for _, file := range m.Seeded {
		out[file.Path] = file
	}
	return out
}

type DesiredFile struct {
	Path string
	Kind FileKind
	Data []byte
	Link string
	Mode os.FileMode
}

type ActionKind string

const (
	Write   ActionKind = "write"
	Link    ActionKind = "link"
	Remove  ActionKind = "remove"
	Restore ActionKind = "restore"
)

// Action is a fully resolved, reviewable filesystem mutation. Paths remain
// logical host paths; Transaction is the only layer allowed to map them into
// --root.
type Action struct {
	Kind    ActionKind
	Path    string
	Desired *DesiredFile
	Reason  string
	Owned   bool
}

type RepoInfo struct {
	Root    string
	Payload string
	Version string
	Commit  string
	Branch  string
}

type Plan struct {
	Config      Config
	Repo        RepoInfo
	OldManifest *Manifest
	Manifest    *Manifest
	Actions     []Action
	Impact      Impact
	LegacyPath  string
}

func (d DesiredFile) Record(sum string) FileRecord {
	return FileRecord{Path: d.Path, Kind: d.Kind, Sum: sum, Link: d.Link, Mode: uint32(d.Mode.Perm())}
}

type Impact struct {
	Create          int
	Replace         int
	RemoveStale     int
	RestorePrevious int
	RemoveCreated   int
	KeepModified    int
	Unchanged       int
	Packages        []string
	Warnings        []string
	Details         []string
}

func (i Impact) Mutations() int {
	return i.Create + i.Replace + i.RemoveStale + i.RestorePrevious + i.RemoveCreated
}

type Event struct {
	Step   string
	Detail string
	Done   int
	Total  int
	Err    error
}

type Result struct {
	Operation Operation
	Success   bool
	Changed   int
	Kept      int
	Warnings  []string
	LogPath   string
	Duration  time.Duration
	Err       error
}
