package v2

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Before struct {
	Path    string   `json:"path"`
	Existed bool     `json:"existed"`
	Kind    FileKind `json:"kind,omitempty"`
	Backup  string   `json:"backup,omitempty"`
	Link    string   `json:"link,omitempty"`
	Mode    uint32   `json:"mode,omitempty"`
}

type Journal struct {
	Version   int       `json:"version"`
	ID        string    `json:"id"`
	Operation Operation `json:"operation"`
	StartedAt string    `json:"started_at"`
	State     string    `json:"state"`
	Actions   []Before  `json:"actions"`
	Error     string    `json:"error,omitempty"`
}

// Transaction is a write-ahead filesystem transaction. Every previous state is
// durable before the destination is changed, including symlink targets.
type Transaction struct {
	cfg         Config
	dir         string
	journalPath string
	activePath  string
	lastPath    string
	j           Journal
	seen        map[string]bool
	mutations   int
}

func Begin(cfg Config) (*Transaction, error) {
	baseLogical := filepath.Join(cfg.StateHome(), "ilmango-v2", "transactions")
	base, err := cfg.Resolve(baseLogical)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(base, 0o700); err != nil {
		return nil, fmt.Errorf("creating transaction directory: %w", err)
	}
	id := time.Now().UTC().Format("20060102T150405.000000000Z")
	dir := filepath.Join(base, id)
	if err := os.Mkdir(dir, 0o700); err != nil {
		return nil, err
	}
	stateBase := filepath.Dir(base)
	t := &Transaction{
		cfg: cfg, dir: dir,
		journalPath: filepath.Join(dir, "journal.json"),
		activePath:  filepath.Join(stateBase, "active-transaction.json"),
		lastPath:    filepath.Join(stateBase, "last-transaction.json"),
		seen:        map[string]bool{},
		j: Journal{Version: 1, ID: id, Operation: cfg.Operation,
			StartedAt: time.Now().Format(time.RFC3339), State: "active"},
	}
	if err := t.flush(); err != nil {
		return nil, err
	}
	if err := atomicJSON(t.activePath, map[string]string{"journal": t.journalPath}, 0o600); err != nil {
		return nil, err
	}
	return t, nil
}

func RecoverActive(cfg Config) (bool, error) {
	active, err := cfg.Resolve(filepath.Join(cfg.StateHome(), "ilmango-v2", "active-transaction.json"))
	if err != nil {
		return false, err
	}
	data, err := os.ReadFile(active)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var pointer map[string]string
	if json.Unmarshal(data, &pointer) != nil || pointer["journal"] == "" {
		return false, fmt.Errorf("invalid active transaction pointer at %s", active)
	}
	if err := validateJournalPath(cfg, pointer["journal"]); err != nil {
		return false, err
	}
	t, err := loadTransaction(cfg, pointer["journal"])
	if err != nil {
		return false, err
	}
	if t.j.State == "active" || t.j.State == "failed" {
		if err := t.Rollback(fmt.Errorf("recovering interrupted transaction")); err != nil {
			return false, err
		}
	}
	return true, nil
}

func HasActiveTransaction(cfg Config) (bool, error) {
	active, err := cfg.Resolve(filepath.Join(cfg.StateHome(), "ilmango-v2", "active-transaction.json"))
	if err != nil {
		return false, err
	}
	_, err = os.Stat(active)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

func loadTransaction(cfg Config, path string) (*Transaction, error) {
	if err := validateJournalPath(cfg, path); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var j Journal
	if err := json.Unmarshal(data, &j); err != nil {
		return nil, err
	}
	stateBase := filepath.Dir(filepath.Dir(filepath.Dir(path)))
	return &Transaction{cfg: cfg, dir: filepath.Dir(path), journalPath: path,
		activePath: filepath.Join(stateBase, "active-transaction.json"),
		lastPath:   filepath.Join(stateBase, "last-transaction.json"), j: j, seen: map[string]bool{}}, nil
}

func (t *Transaction) capture(path string) error {
	if t.seen[path] {
		return nil
	}
	actual, err := t.cfg.Resolve(path)
	if err != nil {
		return err
	}
	b := Before{Path: path}
	info, err := os.Lstat(actual)
	switch {
	case errors.Is(err, os.ErrNotExist):
		// A non-existent path is still journaled: rollback must delete what we create.
	case err != nil:
		return fmt.Errorf("inspecting %s: %w", path, err)
	case info.Mode().IsRegular():
		b.Existed, b.Kind, b.Mode = true, Regular, uint32(info.Mode().Perm())
		name := fmt.Sprintf("backup-%06d", len(t.j.Actions))
		backup := filepath.Join(t.dir, name)
		if err := copyRegular(actual, backup, info.Mode().Perm()); err != nil {
			return fmt.Errorf("backing up %s: %w", path, err)
		}
		b.Backup = name
	case info.Mode()&fs.ModeSymlink != 0:
		b.Existed, b.Kind = true, Symlink
		b.Link, err = os.Readlink(actual)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("refusing to replace non-file path %s", path)
	}
	t.j.Actions = append(t.j.Actions, b)
	if err := t.flush(); err != nil {
		return err
	}
	t.seen[path] = true
	return nil
}

func (t *Transaction) WriteFile(path string, data []byte, mode os.FileMode) error {
	if err := t.capture(path); err != nil {
		return err
	}
	actual, _ := t.cfg.Resolve(path)
	if err := atomicWrite(actual, data, mode.Perm()); err != nil {
		return err
	}
	t.mutations++
	return t.injectFailure()
}

func (t *Transaction) Symlink(path, target string) error {
	if err := t.capture(path); err != nil {
		return err
	}
	actual, _ := t.cfg.Resolve(path)
	if err := os.MkdirAll(filepath.Dir(actual), 0o755); err != nil {
		return err
	}
	tmp := actual + ".ilmango-link"
	_ = os.Remove(tmp)
	if err := os.Symlink(target, tmp); err != nil {
		return err
	}
	if err := os.Rename(tmp, actual); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	t.mutations++
	return t.injectFailure()
}

func (t *Transaction) Remove(path string) error {
	if err := t.capture(path); err != nil {
		return err
	}
	actual, _ := t.cfg.Resolve(path)
	if err := os.Remove(actual); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	t.mutations++
	return t.injectFailure()
}

func (t *Transaction) injectFailure() error {
	if t.cfg.FailAfter > 0 && t.mutations >= t.cfg.FailAfter {
		return fmt.Errorf("injected failure after %d mutations", t.mutations)
	}
	return nil
}

func (t *Transaction) Commit() error {
	previous := previousJournal(t.lastPath)
	t.j.State = "committed"
	if err := t.flush(); err != nil {
		return err
	}
	if err := atomicJSON(t.lastPath, map[string]string{"journal": t.journalPath}, 0o600); err != nil {
		return err
	}
	if err := os.Remove(t.activePath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	// Only the immediately previous operation can be rolled back. Once the new
	// pointer is durable, its predecessor's potentially large payload backups
	// no longer serve a recovery purpose.
	cleanupTransaction(previous, t.dir, filepath.Dir(t.dir))
	return nil
}

func (t *Transaction) Rollback(cause error) error {
	var failures []string
	for i := len(t.j.Actions) - 1; i >= 0; i-- {
		if err := t.restore(t.j.Actions[i]); err != nil {
			failures = append(failures, err.Error())
		}
	}
	if len(failures) > 0 {
		t.j.State = "rollback-failed"
		t.j.Error = strings.Join(failures, "; ")
		_ = t.flush()
		return fmt.Errorf("rollback incomplete after %v: %s", cause, t.j.Error)
	}
	t.j.State = "rolled-back"
	if cause != nil {
		t.j.Error = cause.Error()
	}
	if err := t.flush(); err != nil {
		return err
	}
	_ = os.Remove(t.activePath)
	return nil
}

func RollbackLast(cfg Config) (int, error) {
	j, journalPath, err := loadLastJournal(cfg)
	if err != nil {
		return 0, err
	}
	last, err := cfg.Resolve(filepath.Join(cfg.StateHome(), "ilmango-v2", "last-transaction.json"))
	if err != nil {
		return 0, err
	}
	t, err := loadTransaction(cfg, journalPath)
	if err != nil {
		return 0, err
	}
	count := len(j.Actions)
	if err := t.Rollback(fmt.Errorf("user requested rollback")); err != nil {
		return 0, err
	}
	_ = os.Remove(last)
	_ = os.RemoveAll(t.dir)
	return count, nil
}

func LoadLastJournal(cfg Config) (Journal, error) {
	j, _, err := loadLastJournal(cfg)
	return j, err
}

func loadLastJournal(cfg Config) (Journal, string, error) {
	path := journalPathFromLast(cfg)
	if path == "" {
		return Journal{}, "", errors.New("no committed transaction is available to roll back")
	}
	if err := validateJournalPath(cfg, path); err != nil {
		return Journal{}, "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Journal{}, "", err
	}
	var j Journal
	if err := json.Unmarshal(data, &j); err != nil {
		return Journal{}, "", err
	}
	if j.State != "committed" {
		return Journal{}, "", fmt.Errorf("last transaction is %s, not committed", j.State)
	}
	return j, path, nil
}

func journalPathFromLast(cfg Config) string {
	last, err := cfg.Resolve(filepath.Join(cfg.StateHome(), "ilmango-v2", "last-transaction.json"))
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(last)
	if err != nil {
		return ""
	}
	var pointer map[string]string
	if json.Unmarshal(data, &pointer) != nil {
		return ""
	}
	return pointer["journal"]
}

func previousJournal(pointerPath string) string {
	data, err := os.ReadFile(pointerPath)
	if err != nil {
		return ""
	}
	var pointer map[string]string
	if json.Unmarshal(data, &pointer) != nil {
		return ""
	}
	return pointer["journal"]
}

func cleanupTransaction(journal, keepDir, allowedRoot string) {
	if journal == "" {
		return
	}
	if filepath.Base(journal) != "journal.json" {
		return
	}
	dir := filepath.Clean(filepath.Dir(journal))
	rel, err := filepath.Rel(filepath.Clean(allowedRoot), dir)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) ||
		strings.Contains(rel, string(filepath.Separator)) || dir == filepath.Clean(keepDir) {
		return
	}
	_ = os.RemoveAll(dir)
}

func validateJournalPath(cfg Config, journal string) error {
	base, err := cfg.Resolve(filepath.Join(cfg.StateHome(), "ilmango-v2", "transactions"))
	if err != nil {
		return err
	}
	clean := filepath.Clean(journal)
	if !filepath.IsAbs(clean) || filepath.Base(clean) != "journal.json" {
		return fmt.Errorf("invalid transaction journal path %q", journal)
	}
	rel, err := filepath.Rel(base, clean)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("transaction journal escapes state directory: %q", journal)
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) != 2 || parts[0] == "" || parts[0] == "." || parts[1] != "journal.json" {
		return fmt.Errorf("invalid transaction journal layout %q", journal)
	}
	return nil
}

func (t *Transaction) restore(b Before) error {
	actual, err := t.cfg.Resolve(b.Path)
	if err != nil {
		return err
	}
	if !b.Existed {
		if err := os.Remove(actual); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		stop, _ := t.cfg.Resolve(t.cfg.Home)
		prune(filepath.Dir(actual), stop)
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(actual), 0o755); err != nil {
		return err
	}
	switch b.Kind {
	case Symlink:
		tmp := actual + ".ilmango-restore-link"
		_ = os.Remove(tmp)
		if err := os.Symlink(b.Link, tmp); err != nil {
			return err
		}
		return os.Rename(tmp, actual)
	case Regular:
		return copyAtomic(filepath.Join(t.dir, b.Backup), actual, os.FileMode(b.Mode))
	default:
		return fmt.Errorf("unknown previous kind for %s", b.Path)
	}
}

func (t *Transaction) flush() error { return atomicJSON(t.journalPath, t.j, 0o600) }

func Sum(data []byte) string {
	d := sha256.Sum256(data)
	return hex.EncodeToString(d[:])
}

func Inspect(path string) (kind FileKind, sum, link string, mode os.FileMode, err error) {
	info, err := os.Lstat(path)
	if err != nil {
		return "", "", "", 0, err
	}
	mode = info.Mode().Perm()
	if info.Mode()&fs.ModeSymlink != 0 {
		link, err = os.Readlink(path)
		return Symlink, "", link, mode, err
	}
	if !info.Mode().IsRegular() {
		return "", "", "", mode, fmt.Errorf("not a regular file or symlink")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", "", mode, err
	}
	return Regular, Sum(data), "", mode, nil
}

func LoadManifest(cfg Config) (*Manifest, error) {
	path, err := cfg.Resolve(ManifestPath(cfg))
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if m.Version != ManifestVersion {
		return nil, fmt.Errorf("unsupported manifest version %d", m.Version)
	}
	return &m, nil
}

func ManifestPath(cfg Config) string {
	return filepath.Join(cfg.StateHome(), "ilmango-v2", "installed-files.json")
}

func atomicJSON(path string, value any, mode os.FileMode) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(path, append(data, '\n'), mode)
}

func atomicWrite(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".ilmango-v2-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, path); err != nil {
		return err
	}
	return syncDir(filepath.Dir(path))
}

func syncDir(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

func copyRegular(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Sync(); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func copyAtomic(src, dst string, mode os.FileMode) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return atomicWrite(dst, data, mode)
}

func prune(dir, stop string) {
	stop = filepath.Clean(stop)
	for dir != "/" && dir != "." && dir != stop {
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) != 0 {
			return
		}
		if os.Remove(dir) != nil {
			return
		}
		dir = filepath.Dir(dir)
	}
}

func SortedPaths[T any](m map[string]T) []string {
	out := make([]string, 0, len(m))
	for path := range m {
		out = append(out, path)
	}
	sort.Strings(out)
	return out
}
