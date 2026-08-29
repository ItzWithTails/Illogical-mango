package v2

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"ilmango/internal/pkg"
	"ilmango/internal/run"
	"ilmango/internal/system"
)

var payloadDirs = []string{
	"shell", "modules", "services", "scripts", "assets", "translations",
	"defaults", "dots", "sdata",
}

var excludedNames = map[string]bool{
	"AGENTS.md": true, "CLAUDE.md": true, "CODEX.md": true, "PI.md": true,
	"codemap.md": true, ".mcp.json": true, "opencode.json": true,
	"skills-lock.json": true, ".agents": true, ".claude": true,
	".codex": true, ".factory": true, ".opencode": true,
	".codebase-memory": true, ".impeccable": true, ".pi-subagents": true,
}

const (
	mangoHookBegin     = "# >>> Illogical-mango installer (managed; do not edit) >>>"
	mangoHookEnd       = "# <<< Illogical-mango installer <<<"
	mangoKeyboardBegin = "# >>> Illogical-mango keyboard (managed; do not edit) >>>"
	mangoKeyboardEnd   = "# <<< Illogical-mango keyboard <<<"
)

func LocateRepo(hint string) (RepoInfo, error) {
	r, err := system.FindRepo(hint)
	if err != nil {
		return RepoInfo{}, err
	}
	return RepoInfo{Root: r.Root, Payload: r.Payload, Version: r.Version,
		Commit: r.Commit, Branch: r.Branch}, nil
}

// PreparePlan performs required crash recovery before observing destination
// state. A plan built from half-applied files would be internally consistent
// but wrong, so callers should use this entry point for real operations.
func PreparePlan(cfg Config) (*Plan, error) {
	recovered := false
	active, err := HasActiveTransaction(cfg)
	if err != nil {
		return nil, err
	}
	if cfg.DryRun && active {
		return nil, errors.New("an interrupted filesystem transaction needs recovery; dry-run cannot modify it, so rerun without --dry-run")
	}
	if !cfg.DryRun && cfg.Operation != Status {
		var err error
		recovered, err = RecoverActive(cfg)
		if err != nil {
			return nil, fmt.Errorf("recovering interrupted transaction: %w", err)
		}
	}
	p, err := BuildPlan(cfg)
	if err != nil {
		return nil, err
	}
	if recovered {
		p.Impact.Warnings = append([]string{"Recovered and rolled back an interrupted previous transaction before computing this plan."}, p.Impact.Warnings...)
	}
	if active && cfg.Operation == Status {
		p.Impact.Warnings = append([]string{"An interrupted transaction is present. Status did not recover it because status is read-only."}, p.Impact.Warnings...)
	}
	return p, nil
}

// BuildPlan computes the complete target state before anything is changed.
// This makes dry-run and the review screen truthful and gives updates exact
// old-minus-new stale-file semantics.
func BuildPlan(cfg Config) (*Plan, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	p := &Plan{Config: cfg}

	switch cfg.Operation {
	case Rollback:
		return buildRollbackPlan(p)
	case Status, Uninstall:
		old, legacy, err := loadAnyManifest(cfg)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) && cfg.Operation == Status {
				p.Impact.Warnings = append(p.Impact.Warnings, "No v2 installation record exists.")
				return p, nil
			}
			return nil, fmt.Errorf("loading installation record: %w", err)
		}
		p.OldManifest = old
		p.LegacyPath = legacy
		if legacy != "" {
			p.Impact.Warnings = append(p.Impact.Warnings, "A legacy v1 installation record will be migrated transactionally.")
		}
		if cfg.Operation == Status {
			return buildStatusPlan(p)
		}
		return buildUninstallPlan(p)
	}

	repo, err := LocateRepo(cfg.Repo)
	if err != nil {
		return nil, err
	}
	p.Repo = repo
	old, legacy, err := loadAnyManifest(cfg)
	if err == nil {
		p.OldManifest = old
		p.LegacyPath = legacy
		if legacy != "" {
			p.Impact.Warnings = append(p.Impact.Warnings, "A legacy v1 installation record will be migrated transactionally.")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("loading installation record: %w", err)
	}
	if cfg.Operation == Update && p.OldManifest == nil {
		return nil, fmt.Errorf("cannot update: no v2 installation record exists; use install")
	}
	if cfg.Root == "" && !run.Exists("mango") && !run.Exists("mmsg") {
		p.Impact.Warnings = append(p.Impact.Warnings,
			"MangoWM was not found on PATH. The shell files can be installed, but you need a working Mango session separately.")
	}

	desired, unmanaged, seededMain, err := buildDesired(cfg, repo)
	if err != nil {
		return nil, err
	}
	p.Manifest = NewManifest(repo.Version, repo.Commit, repo.Root)
	if err := compareDesired(p, desired); err != nil {
		return nil, err
	}
	reconcileSeededMain(p, seededMain, unmanaged)
	for _, a := range unmanaged {
		p.Actions = append(p.Actions, a)
		actual, _ := cfg.Resolve(a.Path)
		if _, err := os.Lstat(actual); errors.Is(err, os.ErrNotExist) {
			p.Impact.Create++
		} else {
			p.Impact.Replace++
		}
		p.Impact.Details = append(p.Impact.Details, a.Reason+": "+a.Path)
	}
	if cfg.ConflictPolicy == Replace {
		p.Impact.Warnings = append(p.Impact.Warnings,
			"Replace policy is active: conflicting files will be overwritten after their exact previous state is journalled.")
	}
	p.Impact.Details = append(p.Impact.Details, "persistent transcript: "+filepath.Join(cfg.StateHome(), "ilmango-v2", "logs"))
	if err := appendPackageImpact(p); err != nil {
		return nil, err
	}
	sort.Slice(p.Actions, func(i, j int) bool { return p.Actions[i].Path < p.Actions[j].Path })
	return p, nil
}

func buildRollbackPlan(p *Plan) (*Plan, error) {
	j, err := LoadLastJournal(p.Config)
	if err != nil {
		return nil, err
	}
	for _, before := range j.Actions {
		if before.Existed {
			p.Actions = append(p.Actions, Action{Kind: Restore, Path: before.Path,
				Reason: "restore exact state before the last operation"})
			p.Impact.RestorePrevious++
		} else {
			p.Actions = append(p.Actions, Action{Kind: Remove, Path: before.Path,
				Reason: "remove a path created by the last operation"})
			p.Impact.RemoveCreated++
		}
	}
	p.Impact.Details = append(p.Impact.Details, "persistent transcript: "+filepath.Join(p.Config.StateHome(), "ilmango-v2", "logs"))
	sort.Slice(p.Actions, func(i, j int) bool { return p.Actions[i].Path < p.Actions[j].Path })
	return p, nil
}

func buildDesired(cfg Config, repo RepoInfo) (map[string]DesiredFile, []Action, *DesiredFile, error) {
	out := map[string]DesiredFile{}
	addFile := func(src, dst string, transform func([]byte) []byte) error {
		data, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		if transform != nil {
			data = transform(data)
		}
		info, err := os.Stat(src)
		if err != nil {
			return err
		}
		out[dst] = DesiredFile{Path: dst, Kind: Regular, Data: data, Mode: info.Mode().Perm()}
		return nil
	}
	addTree := func(src, dst string) error {
		return filepath.WalkDir(src, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if path == src {
				return nil
			}
			if excludedNames[d.Name()] {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if d.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(src, path)
			if err != nil {
				return err
			}
			info, err := os.Lstat(path)
			if err != nil {
				return err
			}
			target := filepath.Join(dst, rel)
			if info.Mode()&fs.ModeSymlink != 0 {
				link, err := os.Readlink(path)
				if err != nil {
					return err
				}
				out[target] = DesiredFile{Path: target, Kind: Symlink, Link: link, Mode: 0o777}
				return nil
			}
			return addFile(path, target, nil)
		})
	}

	qml, err := filepath.Glob(filepath.Join(repo.Payload, "*.qml"))
	if err != nil || len(qml) == 0 {
		return nil, nil, nil, fmt.Errorf("payload has no top-level QML files")
	}
	for _, src := range qml {
		if err := addFile(src, filepath.Join(cfg.ShellDir(), filepath.Base(src)), nil); err != nil {
			return nil, nil, nil, err
		}
	}
	for _, name := range []string{"qmldir", "VERSION"} {
		src := filepath.Join(repo.Payload, name)
		if _, err := os.Stat(src); err == nil {
			if err := addFile(src, filepath.Join(cfg.ShellDir(), name), nil); err != nil {
				return nil, nil, nil, err
			}
		}
	}
	if src := filepath.Join(repo.Root, "docs", "CHANGELOG.md"); fileExists(src) {
		if err := addFile(src, filepath.Join(cfg.ShellDir(), "CHANGELOG.md"), nil); err != nil {
			return nil, nil, nil, err
		}
	}
	for _, name := range payloadDirs {
		src := filepath.Join(repo.Payload, name)
		if dirExists(src) {
			if err := addTree(src, filepath.Join(cfg.ShellDir(), name)); err != nil {
				return nil, nil, nil, err
			}
		}
	}

	launcher := filepath.Join(repo.Payload, "scripts", "ilmango")
	if err := addFile(launcher, filepath.Join(cfg.BinHome(), "ilmango"), nil); err != nil {
		return nil, nil, nil, fmt.Errorf("adding launcher: %w", err)
	}
	launcherPath := filepath.Join(cfg.BinHome(), "ilmango")
	for name, command := range map[string]string{
		"ilmango.desktop": "service restart", "ilmango-settings.desktop": "settings",
	} {
		src := filepath.Join(repo.Payload, "assets", "applications", name)
		dst := filepath.Join(cfg.DataHome(), "applications", name)
		err := addFile(src, dst, func(data []byte) []byte { return rewriteDesktopExec(data, launcherPath+" "+command) })
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, nil, nil, err
		}
	}
	icon := filepath.Join(repo.Payload, "assets", "icons", "desktop-symbolic.svg")
	if fileExists(icon) {
		if err := addFile(icon, filepath.Join(cfg.DataHome(), "icons", "hicolor", "scalable", "apps", "ilmango.svg"), nil); err != nil {
			return nil, nil, nil, err
		}
	}
	unit := filepath.Join(repo.Payload, "assets", "systemd", "ilmango.service")
	if fileExists(unit) {
		if err := addFile(unit, filepath.Join(cfg.ConfigHome(), "systemd", "user", "ilmango.service"), func(data []byte) []byte {
			s := strings.ReplaceAll(string(data), "/usr/local/bin/ilmango", launcherPath)
			s = strings.ReplaceAll(s, "/usr/bin/ilmango", launcherPath)
			return []byte(s)
		}); err != nil {
			return nil, nil, nil, err
		}
	}

	// Full is the only preset allowed to mirror broad desktop preferences.
	// Recommended installs only the compositor integration explicitly selected.
	if cfg.Preset == Full {
		dots := filepath.Join(repo.Payload, "dots")
		entries, err := os.ReadDir(dots)
		if err != nil {
			return nil, nil, nil, err
		}
		for _, entry := range entries {
			if entry.IsDir() && strings.HasPrefix(entry.Name(), ".") {
				if err := addTree(filepath.Join(dots, entry.Name()), filepath.Join(cfg.Home, entry.Name())); err != nil {
					return nil, nil, nil, err
				}
			}
		}
	}

	var unmanaged []Action
	var seededMain *DesiredFile
	if cfg.MangoHook && cfg.Preset != Minimal {
		main := filepath.Join(cfg.ConfigHome(), "mango", "config.conf")
		current, err := readLogical(cfg, main)
		missingMain := errors.Is(err, os.ErrNotExist)
		if err != nil && !missingMain {
			return nil, nil, nil, err
		}
		copiedSystemConfig := false
		if missingMain && cfg.Root == "" {
			if systemConfig, readErr := os.ReadFile("/etc/mango/config.conf"); readErr == nil {
				current, copiedSystemConfig = systemConfig, true
			}
		}
		src := filepath.Join(repo.Payload, "defaults", "mango", "config.conf")
		dst := filepath.Join(cfg.ConfigHome(), "mango", "ilmango.conf")
		if err := addFile(src, dst, func(data []byte) []byte {
			body := strings.ReplaceAll(string(data), "spawn,ilmango ", "spawn,"+launcherPath+" ")
			body = strings.ReplaceAll(body, "exec-once=ilmango ", "exec-once="+launcherPath+" ")
			return []byte(body)
		}); err != nil {
			return nil, nil, nil, err
		}
		updated := setMangoHook(string(current), dst, cfg.KeyboardLayout)
		if updated != string(current) {
			d := DesiredFile{Path: main, Kind: Regular, Data: []byte(updated), Mode: 0o644}
			reason := "add a reversible, marked Mango include before existing bindings"
			if copiedSystemConfig {
				reason = "seed the user config from /etc/mango/config.conf and add the marked include first"
			}
			unmanaged = append(unmanaged, Action{Kind: Write, Path: main, Desired: &d,
				Reason: reason, Owned: false})
			if missingMain {
				seededCopy := d
				seededMain = &seededCopy
			}
		}
	}

	versionData, _ := json.MarshalIndent(map[string]string{
		"version": repo.Version, "commit": repo.Commit,
		"installed_at": time.Now().Format(time.RFC3339), "source": "installer-v2",
		"repo_path": repo.Root,
	}, "", "  ")
	versionPath := filepath.Join(cfg.ConfigHome(), "ilmango", "version.json")
	out[versionPath] = DesiredFile{Path: versionPath, Kind: Regular, Data: append(versionData, '\n'), Mode: 0o644}
	return out, unmanaged, seededMain, nil
}

// reconcileSeededMain remembers a hybrid file that the installer had to create.
// It is not generally owned because users are expected to edit Mango's main
// config. If its exact installer-created state survives, uninstall may remove it
// and restore Mango's system-config fallback. Once the user changes it, the old
// baseline is retained forever so those changes can never be mistaken for ours.
func reconcileSeededMain(p *Plan, newlySeeded *DesiredFile, actions []Action) {
	if newlySeeded != nil {
		p.Manifest.Seeded = []FileRecord{desiredRecord(*newlySeeded)}
		return
	}
	if p.OldManifest == nil {
		return
	}

	for _, old := range p.OldManifest.Seeded {
		record := old
		actual, err := p.Config.Resolve(old.Path)
		if err == nil {
			kind, sum, link, mode, inspectErr := Inspect(actual)
			if inspectErr == nil && old.MatchesState(kind, sum, link, mode) {
				for _, action := range actions {
					if action.Path == old.Path && action.Desired != nil {
						record = desiredRecord(*action.Desired)
						break
					}
				}
			}
		}
		p.Manifest.Seeded = append(p.Manifest.Seeded, record)
	}
	sort.Slice(p.Manifest.Seeded, func(i, j int) bool {
		return p.Manifest.Seeded[i].Path < p.Manifest.Seeded[j].Path
	})
}

func compareDesired(p *Plan, desired map[string]DesiredFile) error {
	old := map[string]FileRecord{}
	if p.OldManifest != nil {
		old = p.OldManifest.Map()
	}
	next := map[string]FileRecord{}
	for _, path := range SortedPaths(desired) {
		d := desired[path]
		record := desiredRecord(d)
		actual, err := p.Config.Resolve(path)
		if err != nil {
			return err
		}
		kind, sum, link, mode, inspectErr := Inspect(actual)
		owned, wasOwned := old[path]
		if inspectErr == nil && record.MatchesState(kind, sum, link, mode) && wasOwned {
			next[path] = record
			p.Impact.Unchanged++
			continue
		}
		if inspectErr == nil && record.MatchesState(kind, sum, link, mode) && !wasOwned && p.Config.ConflictPolicy == Preserve {
			p.Impact.KeepModified++
			p.Impact.Warnings = append(p.Impact.Warnings, "Kept identical pre-existing file unowned: "+path)
			continue
		}
		modifiedOwned := wasOwned && inspectErr == nil && !owned.MatchesState(kind, sum, link, mode)
		foreign := !wasOwned && inspectErr == nil
		if p.Config.ConflictPolicy == Preserve && (modifiedOwned || foreign) {
			p.Impact.KeepModified++
			why := "pre-existing file"
			if modifiedOwned {
				why = "locally modified installed file"
				next[path] = owned
			}
			p.Impact.Warnings = append(p.Impact.Warnings, "Kept "+why+": "+path)
			continue
		}
		a := Action{Path: path, Desired: &d, Reason: "install current payload", Owned: true}
		if d.Kind == Symlink {
			a.Kind = Link
		} else {
			a.Kind = Write
		}
		if errors.Is(inspectErr, os.ErrNotExist) {
			p.Impact.Create++
		} else {
			p.Impact.Replace++
		}
		p.Actions = append(p.Actions, a)
		next[path] = record
	}

	for _, path := range SortedPaths(old) {
		if _, wanted := desired[path]; wanted {
			continue
		}
		record := old[path]
		actual, _ := p.Config.Resolve(path)
		kind, sum, link, mode, err := Inspect(actual)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil || !record.MatchesState(kind, sum, link, mode) {
			p.Impact.KeepModified++
			p.Impact.Warnings = append(p.Impact.Warnings, "Kept stale but modified path: "+path)
			continue
		}
		p.Actions = append(p.Actions, Action{Kind: Remove, Path: path, Reason: "remove stale file from previous release", Owned: true})
		p.Impact.RemoveStale++
	}
	p.Manifest.Set(next)
	return nil
}

func buildUninstallPlan(p *Plan) (*Plan, error) {
	for _, record := range p.OldManifest.Files {
		actual, _ := p.Config.Resolve(record.Path)
		kind, sum, link, mode, err := Inspect(actual)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil || !record.MatchesState(kind, sum, link, mode) {
			p.Impact.KeepModified++
			p.Impact.Warnings = append(p.Impact.Warnings, "Will keep modified path: "+record.Path)
			continue
		}
		p.Actions = append(p.Actions, Action{Kind: Remove, Path: record.Path, Reason: "remove installer-owned file", Owned: true})
		p.Impact.RemoveStale++
	}
	main := filepath.Join(p.Config.ConfigHome(), "mango", "config.conf")
	removeSeededMain := false
	for _, record := range p.OldManifest.Seeded {
		actual, err := p.Config.Resolve(record.Path)
		if err != nil {
			continue
		}
		kind, sum, link, mode, inspectErr := Inspect(actual)
		if errors.Is(inspectErr, os.ErrNotExist) {
			continue
		}
		if inspectErr == nil && record.MatchesState(kind, sum, link, mode) {
			p.Actions = append(p.Actions, Action{Kind: Remove, Path: record.Path,
				Reason: "remove unchanged user config created by the installer"})
			p.Impact.RemoveStale++
			if record.Path == main {
				removeSeededMain = true
			}
			continue
		}
		p.Impact.KeepModified++
		p.Impact.Warnings = append(p.Impact.Warnings,
			"Will preserve installer-seeded file because it was modified: "+record.Path)
	}
	current, err := readLogical(p.Config, main)
	if err == nil && !removeSeededMain {
		clean := removeMangoHook(string(current))
		if clean != string(current) {
			d := DesiredFile{Path: main, Kind: Regular, Data: []byte(clean), Mode: 0o644}
			p.Actions = append(p.Actions, Action{Kind: Write, Path: main, Desired: &d,
				Reason: "remove only installer-managed Mango blocks"})
			p.Impact.Replace++
		}
	}
	sort.Slice(p.Actions, func(i, j int) bool { return p.Actions[i].Path < p.Actions[j].Path })
	p.Impact.Details = append(p.Impact.Details, "persistent transcript: "+filepath.Join(p.Config.StateHome(), "ilmango-v2", "logs"))
	return p, nil
}

func buildStatusPlan(p *Plan) (*Plan, error) {
	records := make([]FileRecord, 0, len(p.OldManifest.Files)+len(p.OldManifest.Seeded))
	records = append(records, p.OldManifest.Files...)
	records = append(records, p.OldManifest.Seeded...)
	for _, record := range records {
		actual, _ := p.Config.Resolve(record.Path)
		kind, sum, link, mode, err := Inspect(actual)
		switch {
		case errors.Is(err, os.ErrNotExist):
			p.Impact.RemoveStale++
			p.Impact.Details = append(p.Impact.Details, "missing: "+record.Path)
		case err != nil || !record.MatchesState(kind, sum, link, mode):
			p.Impact.KeepModified++
			p.Impact.Details = append(p.Impact.Details, "modified: "+record.Path)
		default:
			p.Impact.Unchanged++
		}
	}
	return p, nil
}

func desiredRecord(d DesiredFile) FileRecord {
	if d.Kind == Symlink {
		return d.Record("")
	}
	return d.Record(Sum(d.Data))
}

func appendPackageImpact(p *Plan) error {
	if p.Config.Root != "" {
		p.Impact.Warnings = append(p.Impact.Warnings, "Filesystem sandbox active: no host commands or package tools will run.")
	}
	if !p.Config.Packages {
		return nil
	}
	distro := system.DetectDistro()
	if string(distro.Family) != "arch" {
		if p.Config.SystemUpgrade {
			return fmt.Errorf("system upgrade was requested, but automatic upgrades are supported only on Arch")
		}
		p.Impact.Warnings = append(p.Impact.Warnings,
			"Automatic packages are intentionally unavailable for "+string(distro.Family)+": no verified Mango/Quickshell recipe exists. Files can still be installed.")
		return nil
	}
	manager, err := pkg.FindManager(string(system.FamilyArch))
	if err != nil {
		return fmt.Errorf("automatic dependencies were selected, but %w; disable packages or install a supported manager", err)
	}
	p.Impact.Warnings = append(p.Impact.Warnings,
		"Package-manager changes are outside the filesystem transaction and are never automatically removed; the exact package list is shown below.")
	if p.Config.SystemUpgrade {
		p.Impact.Warnings = append(p.Impact.Warnings,
			"You explicitly enabled a full Arch system upgrade. This can change packages unrelated to Illogical-mango.")
	}
	core := []string{
		"bash", "curl", "jq", "python", "quickshell", "qt6-5compat", "qt6-base",
		"qt6-declarative", "qt6-imageformats", "qt6-multimedia", "qt6-positioning",
		"qt6-sensors", "qt6-svg", "qt6-wayland", "kirigami", "syntax-highlighting",
	}
	if p.Config.Preset != Minimal {
		core = append(core, "cliphist", "foot", "fuzzel", "grim", "matugen", "pipewire",
			"playerctl", "slurp", "ttf-material-symbols-variable-git", "ttf-roboto-flex",
			"wireplumber", "wl-clipboard")
	}
	if p.Config.Preset == Full {
		core = append(core, "brightnessctl", "cava", "ddcutil", "imagemagick", "mpv",
			"swappy", "upower", "wf-recorder", "ydotool")
	}
	p.Impact.Packages = uniqueSorted(core)
	if manager.Name == "pacman" {
		for _, name := range p.Impact.Packages {
			if strings.HasSuffix(name, "-git") || name == "ttf-roboto-flex" {
				return fmt.Errorf("%s needs an AUR helper for the selected %s preset; install paru/yay, choose minimal, or disable packages", name, p.Config.Preset)
			}
		}
	}
	p.Impact.Details = append(p.Impact.Details, "package manager: "+manager.Name)
	return nil
}

func keyboardSettings(mainConfig, layout string) string {
	if layout == "system" {
		return ""
	}
	options := findMangoValue(mainConfig, "xkb_rules_options")
	var kept []string
	for _, option := range strings.Split(options, ",") {
		option = strings.TrimSpace(option)
		if option != "" && !strings.HasPrefix(option, "grp:") {
			kept = append(kept, option)
		}
	}
	if strings.Contains(layout, ",") {
		kept = append(kept, "grp:alt_shift_toggle")
	}
	body := "xkb_rules_layout=" + layout + "\n"
	if len(kept) > 0 {
		body += "xkb_rules_options=" + strings.Join(unique(kept), ",") + "\n"
	}
	return body
}

func findMangoValue(config, key string) string {
	for _, line := range strings.Split(config, "\n") {
		line = strings.TrimSpace(line)
		if value, ok := strings.CutPrefix(line, key+"="); ok {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func rewriteDesktopExec(data []byte, command string) []byte {
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "Exec=") {
			lines[i] = "Exec=" + command
		}
	}
	return []byte(strings.Join(lines, "\n"))
}

func setMangoHook(current, include, layout string) string {
	clean := removeMangoHook(current)
	block := mangoHookBegin + "\nsource-optional=" + include + "\n" + mangoHookEnd + "\n"
	result := block + clean
	settings := keyboardSettings(clean, layout)
	if settings == "" {
		return result
	}
	marker := mangoKeyboardBegin
	if result != "" && !strings.HasSuffix(result, "\n") {
		result += "\n"
		marker += " [joined]"
	}
	return result + marker + "\n" + settings + mangoKeyboardEnd + "\n"
}

func removeMangoHook(current string) string {
	current = removeManagedBlocks(current, mangoKeyboardBegin, mangoKeyboardEnd)
	return removeManagedBlocks(current, mangoHookBegin, mangoHookEnd)
}

func removeManagedBlocks(current, begin, endMarker string) string {
	for {
		cleaned, removed := removeOneManagedBlock(current, begin, endMarker)
		if !removed {
			return current
		}
		current = cleaned
	}
}

func removeOneManagedBlock(current, begin, endMarker string) (string, bool) {
	start := strings.Index(current, begin)
	if start < 0 {
		return current, false
	}
	rest := current[start:]
	endRel := strings.Index(rest, endMarker)
	if endRel < 0 {
		return current, false // never guess when the marker is malformed
	}
	end := start + endRel + len(endMarker)
	if end < len(current) && current[end] == '\n' {
		end++
	}
	prefix := current[:start]
	markerLineEnd := strings.IndexByte(rest, '\n')
	if markerLineEnd >= 0 && strings.Contains(rest[:markerLineEnd], "[joined]") {
		prefix = strings.TrimSuffix(prefix, "\n")
	}
	return prefix + current[end:], true
}

func readLogical(cfg Config, logical string) ([]byte, error) {
	actual, err := cfg.Resolve(logical)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(actual)
}

type legacyManifest struct {
	Version     int    `json:"version"`
	InstalledAt string `json:"installed_at"`
	Release     string `json:"release"`
	RepoPath    string `json:"repo_path"`
	Entries     []struct {
		Path string `json:"path"`
		Sum  string `json:"sum,omitempty"`
		Link string `json:"link,omitempty"`
	} `json:"entries"`
}

func legacyManifestPath(cfg Config) string {
	return filepath.Join(cfg.StateHome(), "ilmango", "installed-files.json")
}

func loadAnyManifest(cfg Config) (*Manifest, string, error) {
	m, err := LoadManifest(cfg)
	if err == nil || !errors.Is(err, os.ErrNotExist) {
		return m, "", err
	}
	logical := legacyManifestPath(cfg)
	actual, resolveErr := cfg.Resolve(logical)
	if resolveErr != nil {
		return nil, "", resolveErr
	}
	data, readErr := os.ReadFile(actual)
	if readErr != nil {
		return nil, "", readErr
	}
	var old legacyManifest
	if err := json.Unmarshal(data, &old); err != nil {
		return nil, "", fmt.Errorf("reading legacy manifest: %w", err)
	}
	if old.Version != 1 {
		return nil, "", fmt.Errorf("unsupported legacy manifest version %d", old.Version)
	}
	converted := &Manifest{Version: ManifestVersion, Release: old.Release,
		Repo: old.RepoPath, InstalledAt: old.InstalledAt}
	for _, entry := range old.Entries {
		record := FileRecord{Path: entry.Path, Kind: Regular, Sum: entry.Sum}
		if entry.Link != "" {
			record.Kind, record.Link, record.Sum = Symlink, entry.Link, ""
		}
		converted.Files = append(converted.Files, record)
	}
	return converted, logical, nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}
func dirExists(path string) bool { info, err := os.Stat(path); return err == nil && info.IsDir() }

func uniqueSorted(in []string) []string {
	seen := map[string]bool{}
	for _, item := range in {
		seen[item] = true
	}
	out := make([]string, 0, len(seen))
	for item := range seen {
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

func unique(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, item := range in {
		if !seen[item] {
			seen[item] = true
			out = append(out, item)
		}
	}
	return out
}
