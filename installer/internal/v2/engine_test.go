package v2

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixtureRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "src", "shell.qml"), "// shell\n", 0o644)
	writeTestFile(t, filepath.Join(root, "src", "qmldir"), "module ilmango\n", 0o644)
	writeTestFile(t, filepath.Join(root, "src", "VERSION"), "9.9.9\n", 0o644)
	writeTestFile(t, filepath.Join(root, "src", "scripts", "ilmango"), "#!/bin/sh\n", 0o755)
	writeTestFile(t, filepath.Join(root, "src", "modules", "Widget.qml"), "widget-v1\n", 0o644)
	writeTestFile(t, filepath.Join(root, "src", "defaults", "mango", "config.conf"), "exec-once=ilmango run --daemon\n", 0o644)
	if err := os.MkdirAll(filepath.Join(root, "src", "dots"), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func testConfig(t *testing.T, repo string) Config {
	t.Helper()
	cfg := DefaultConfig()
	cfg.Home = "/home/tester"
	cfg.Root = t.TempDir()
	cfg.Repo = repo
	cfg.Packages = false
	cfg.Preset = Recommended
	return cfg
}

func TestFailureRollsBackEveryCreatedFileAndManifest(t *testing.T) {
	cfg := testConfig(t, fixtureRepo(t))
	cfg.FailAfter = 3
	plan, err := BuildPlan(cfg)
	if err != nil {
		t.Fatal(err)
	}
	result := (Engine{}).Run(context.Background(), plan)
	if result.Err == nil || !strings.Contains(result.Err.Error(), "rolled back") {
		t.Fatalf("Run() error = %v, want an automatic rollback", result.Err)
	}
	manifest, _ := cfg.Resolve(ManifestPath(cfg))
	if _, err := os.Lstat(manifest); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("partial run left manifest %s: %v", manifest, err)
	}
	shell, _ := cfg.Resolve(cfg.ShellDir())
	if entries, err := os.ReadDir(shell); err == nil && len(entries) != 0 {
		t.Fatalf("partial run left %d payload entries", len(entries))
	}
	active, _ := cfg.Resolve(filepath.Join(cfg.StateHome(), "ilmango-v2", "active-transaction.json"))
	if _, err := os.Stat(active); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("active transaction pointer survived rollback: %v", err)
	}
}

func TestUpdateRemovesUnchangedStaleFiles(t *testing.T) {
	repo := fixtureRepo(t)
	cfg := testConfig(t, repo)
	install, err := BuildPlan(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if result := (Engine{}).Run(context.Background(), install); !result.Success {
		t.Fatal(result.Err)
	}

	source := filepath.Join(repo, "src", "modules", "Widget.qml")
	if err := os.Remove(source); err != nil {
		t.Fatal(err)
	}
	cfg.Operation = Update
	update, err := BuildPlan(cfg)
	if err != nil {
		t.Fatal(err)
	}
	staleLogical := filepath.Join(cfg.ShellDir(), "modules", "Widget.qml")
	if !hasAction(update, Remove, staleLogical) {
		t.Fatalf("update did not plan removal of %s", staleLogical)
	}
	if result := (Engine{}).Run(context.Background(), update); !result.Success {
		t.Fatal(result.Err)
	}
	stale, _ := cfg.Resolve(staleLogical)
	if _, err := os.Lstat(stale); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("stale file survived update: %v", err)
	}
	m, err := LoadManifest(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, found := m.Map()[staleLogical]; found {
		t.Fatal("stale file survived in the new manifest")
	}
}

func TestUpdateAndUninstallPreserveModifiedSymlink(t *testing.T) {
	repo := fixtureRepo(t)
	linkSource := filepath.Join(repo, "src", "modules", "current")
	if err := os.Symlink("release-a", linkSource); err != nil {
		t.Fatal(err)
	}
	cfg := testConfig(t, repo)
	plan, err := BuildPlan(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if result := (Engine{}).Run(context.Background(), plan); !result.Success {
		t.Fatal(result.Err)
	}
	logical := filepath.Join(cfg.ShellDir(), "modules", "current")
	actual, _ := cfg.Resolve(logical)
	if err := os.Remove(actual); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("my-release", actual); err != nil {
		t.Fatal(err)
	}

	cfg.Operation = Update
	update, err := BuildPlan(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if hasAction(update, Link, logical) || hasAction(update, Remove, logical) {
		t.Fatal("update planned to overwrite a user-modified symlink")
	}
	if result := (Engine{}).Run(context.Background(), update); !result.Success {
		t.Fatal(result.Err)
	}

	cfg.Operation = Uninstall
	uninstall, err := BuildPlan(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if hasAction(uninstall, Remove, logical) {
		t.Fatal("uninstall planned to delete a user-modified symlink")
	}
	if result := (Engine{}).Run(context.Background(), uninstall); !result.Success {
		t.Fatal(result.Err)
	}
	if got, err := os.Readlink(actual); err != nil || got != "my-release" {
		t.Fatalf("modified symlink after uninstall = %q, %v", got, err)
	}
}

func TestFullPresetDoesNotOverwriteForeignDotfileByDefault(t *testing.T) {
	repo := fixtureRepo(t)
	writeTestFile(t, filepath.Join(repo, "src", "dots", ".config", "foot", "foot.ini"), "from-project\n", 0o644)
	cfg := testConfig(t, repo)
	cfg.Preset = Full
	foreignLogical := filepath.Join(cfg.Home, ".config", "foot", "foot.ini")
	foreign, _ := cfg.Resolve(foreignLogical)
	writeTestFile(t, foreign, "my-settings\n", 0o644)

	plan, err := BuildPlan(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if hasAction(plan, Write, foreignLogical) {
		t.Fatal("preserve policy planned to overwrite a foreign dotfile")
	}
	if result := (Engine{}).Run(context.Background(), plan); !result.Success {
		t.Fatal(result.Err)
	}
	data, err := os.ReadFile(foreign)
	if err != nil || string(data) != "my-settings\n" {
		t.Fatalf("foreign dotfile changed to %q: %v", data, err)
	}
	m, err := LoadManifest(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, found := m.Map()[foreignLogical]; found {
		t.Fatal("foreign preserved dotfile was incorrectly claimed in manifest")
	}
}

func TestIdenticalForeignFileIsNotClaimedOrUninstalled(t *testing.T) {
	repo := fixtureRepo(t)
	cfg := testConfig(t, repo)
	logical := filepath.Join(cfg.ShellDir(), "modules", "Widget.qml")
	actual, _ := cfg.Resolve(logical)
	writeTestFile(t, actual, "widget-v1\n", 0o644)
	plan, err := BuildPlan(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if result := (Engine{}).Run(context.Background(), plan); !result.Success {
		t.Fatal(result.Err)
	}
	m, err := LoadManifest(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, claimed := m.Map()[logical]; claimed {
		t.Fatal("installer claimed an identical file it did not create")
	}
	cfg.Operation = Uninstall
	uninstall, err := BuildPlan(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if hasAction(uninstall, Remove, logical) {
		t.Fatal("uninstall planned to delete an identical foreign file")
	}
	if result := (Engine{}).Run(context.Background(), uninstall); !result.Success {
		t.Fatal(result.Err)
	}
	if data, err := os.ReadFile(actual); err != nil || string(data) != "widget-v1\n" {
		t.Fatalf("foreign file after uninstall = %q, %v", data, err)
	}
}

func TestUninstallTreatsPermissionChangeAsUserModification(t *testing.T) {
	cfg := testConfig(t, fixtureRepo(t))
	plan, err := BuildPlan(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if result := (Engine{}).Run(context.Background(), plan); !result.Success {
		t.Fatal(result.Err)
	}
	logical := filepath.Join(cfg.BinHome(), "ilmango")
	actual, _ := cfg.Resolve(logical)
	if err := os.Chmod(actual, 0o700); err != nil {
		t.Fatal(err)
	}
	cfg.Operation = Uninstall
	uninstall, err := BuildPlan(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if hasAction(uninstall, Remove, logical) {
		t.Fatal("uninstall ignored a user permission change")
	}
	if result := (Engine{}).Run(context.Background(), uninstall); !result.Success {
		t.Fatal(result.Err)
	}
	if info, err := os.Stat(actual); err != nil || info.Mode().Perm() != 0o700 {
		t.Fatalf("permission-modified launcher was not kept: %v, %v", info, err)
	}
}

func TestRootRedirectsAllReadsAndWrites(t *testing.T) {
	cfg := testConfig(t, fixtureRepo(t))
	logical := filepath.Join(cfg.ConfigHome(), "mango", "config.conf")
	inside, _ := cfg.Resolve(logical)
	writeTestFile(t, inside, "inside-root\n", 0o644)
	plan, err := BuildPlan(cfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, action := range plan.Actions {
		if action.Path == logical && action.Desired != nil && !strings.Contains(string(action.Desired.Data), "inside-root") {
			t.Fatal("planner read Mango config from the host instead of --root")
		}
	}
	if result := (Engine{}).Run(context.Background(), plan); !result.Success {
		t.Fatal(result.Err)
	}
	if _, err := os.Stat(filepath.Join(cfg.Root, "home", "tester", ".local", "bin", "ilmango")); err != nil {
		t.Fatalf("launcher was not redirected: %v", err)
	}
}

func TestModifiedSeededMangoConfigSurvivesUninstallWithoutManagedBlocks(t *testing.T) {
	cfg := testConfig(t, fixtureRepo(t))
	plan, err := BuildPlan(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if result := (Engine{}).Run(context.Background(), plan); !result.Success {
		t.Fatal(result.Err)
	}

	logical := filepath.Join(cfg.ConfigHome(), "mango", "config.conf")
	actual, _ := cfg.Resolve(logical)
	data, err := os.ReadFile(actual)
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, []byte("user_setting=keep-me\n")...)
	if err := os.WriteFile(actual, data, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg.Operation = Uninstall
	uninstall, err := BuildPlan(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if hasAction(uninstall, Remove, logical) {
		t.Fatal("uninstall planned to remove a user-modified seeded config")
	}
	if result := (Engine{}).Run(context.Background(), uninstall); !result.Success {
		t.Fatal(result.Err)
	}
	clean, err := os.ReadFile(actual)
	if err != nil {
		t.Fatalf("modified seeded config was removed: %v", err)
	}
	if string(clean) != "user_setting=keep-me\n" {
		t.Fatalf("cleaned seeded config = %q", clean)
	}
}

func TestRecoverActiveRestoresRegularFile(t *testing.T) {
	cfg := testConfig(t, fixtureRepo(t))
	logical := filepath.Join(cfg.Home, ".config", "sample")
	actual, _ := cfg.Resolve(logical)
	writeTestFile(t, actual, "before", 0o640)
	tx, err := Begin(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.WriteFile(logical, []byte("after"), 0o644); err != nil {
		t.Fatal(err)
	}
	recovered, err := RecoverActive(cfg)
	if err != nil || !recovered {
		t.Fatalf("RecoverActive() = %v, %v", recovered, err)
	}
	data, err := os.ReadFile(actual)
	if err != nil || string(data) != "before" {
		t.Fatalf("recovered content = %q, %v", data, err)
	}
	info, _ := os.Stat(actual)
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("recovered mode = %o", info.Mode().Perm())
	}
}

func TestMangoHookRemovalIsBoundedAndIdempotent(t *testing.T) {
	original := "bind=SUPER,Return,spawn,foot\nsource-optional=/elsewhere.conf\n"
	withHook := setMangoHook(original, "/home/u/.config/mango/ilmango.conf", "us,de")
	if !strings.HasPrefix(withHook, mangoHookBegin) || strings.Index(withHook, mangoHookBegin) > strings.Index(withHook, "bind=") {
		t.Fatalf("managed include is not before existing bindings:\n%s", withHook)
	}
	if strings.Index(withHook, mangoKeyboardBegin) < strings.Index(withHook, "bind=") {
		t.Fatalf("keyboard override is not after the user's settings:\n%s", withHook)
	}
	clean := removeMangoHook(withHook)
	if clean != original {
		t.Fatalf("cleaned config = %q, want %q", clean, original)
	}
	if again := removeMangoHook(clean); again != clean {
		t.Fatal("hook removal is not idempotent")
	}
	if duplicate := removeMangoHook(withHook + withHook); strings.Contains(duplicate, mangoHookBegin) {
		t.Fatal("duplicate managed blocks survived removal")
	}
}

func TestKeyboardLayoutKeepsUnrelatedXKBOptions(t *testing.T) {
	got := keyboardSettings("xkb_rules_options=caps:escape,grp:caps_toggle,compose:ralt\n", "us,de")
	for _, want := range []string{
		"xkb_rules_layout=us,de",
		"xkb_rules_options=caps:escape,compose:ralt,grp:alt_shift_toggle",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("generated Mango config lacks %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "grp:caps_toggle") {
		t.Fatal("old layout-switch option was not replaced")
	}
}

func TestLegacyManifestMigratesWithoutLosingOwnership(t *testing.T) {
	repo := fixtureRepo(t)
	cfg := testConfig(t, repo)
	logical := filepath.Join(cfg.ShellDir(), "modules", "Widget.qml")
	actual, _ := cfg.Resolve(logical)
	writeTestFile(t, actual, "widget-v1\n", 0o644)
	legacyLogical := legacyManifestPath(cfg)
	legacy, _ := cfg.Resolve(legacyLogical)
	writeTestFile(t, legacy, `{
  "version": 1,
  "installed_at": "2025-01-01T00:00:00Z",
  "release": "old",
  "repo_path": "/old/repo",
  "entries": [{"path": "`+logical+`", "sum": "`+Sum([]byte("widget-v1\n"))+`"}]
}`, 0o644)

	cfg.Operation = Update
	plan, err := BuildPlan(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if plan.LegacyPath != legacyLogical {
		t.Fatalf("legacy path = %q", plan.LegacyPath)
	}
	if result := (Engine{}).Run(context.Background(), plan); !result.Success {
		t.Fatal(result.Err)
	}
	if _, err := os.Stat(legacy); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("legacy manifest survived migration: %v", err)
	}
	m, err := LoadManifest(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := m.Map()[logical]; !ok {
		t.Fatal("migrated manifest lost ownership of an unchanged file")
	}
}

type failingPackages struct{ called bool }

func (f *failingPackages) Install(context.Context, Config, []string, func(Event)) error {
	f.called = true
	return errors.New("package transaction failed")
}

func TestPackageFailureHappensBeforeFilesystemTransaction(t *testing.T) {
	cfg := testConfig(t, fixtureRepo(t))
	cfg.Root = "" // Package execution is intentionally host-only; use a fake.
	cfg.Home = filepath.Join(t.TempDir(), "home")
	cfg.Packages = true
	plan, err := BuildPlan(cfg)
	if err != nil {
		t.Fatal(err)
	}
	plan.Impact.Packages = []string{"required-test-package"}
	fake := &failingPackages{}
	result := (Engine{Packages: fake}).Run(context.Background(), plan)
	if !fake.called || result.Err == nil {
		t.Fatalf("package failure was ignored: called=%v result=%+v", fake.called, result)
	}
	if result.LogPath == "" {
		t.Fatal("failed run did not retain a transcript path")
	}
	logPath, err := cfg.Resolve(result.LogPath)
	if err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(logPath); err != nil || !strings.Contains(string(data), "operation=install") {
		t.Fatalf("persistent transcript = %q, %v", data, err)
	}
	if _, err := os.Stat(filepath.Join(cfg.BinHome(), "ilmango")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("files changed after package failure: %v", err)
	}
	if _, err := os.Stat(ManifestPath(cfg)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("manifest exists after package failure: %v", err)
	}
}

func TestCancellationRollsBackBeforeReporting(t *testing.T) {
	cfg := testConfig(t, fixtureRepo(t))
	plan, err := BuildPlan(cfg)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	result := (Engine{Emit: func(ev Event) {
		if ev.Step == string(Write) {
			cancel()
		}
	}}).Run(ctx, plan)
	if !errors.Is(result.Err, context.Canceled) && (result.Err == nil || !strings.Contains(result.Err.Error(), "context canceled")) {
		t.Fatalf("cancel result = %v", result.Err)
	}
	launcher, _ := cfg.Resolve(filepath.Join(cfg.BinHome(), "ilmango"))
	if _, err := os.Stat(launcher); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cancel returned before rollback removed launcher: %v", err)
	}
}

func TestOnlyOneCommittedRollbackPayloadIsRetained(t *testing.T) {
	cfg := testConfig(t, fixtureRepo(t))
	logical := filepath.Join(cfg.Home, ".config", "sample")
	for _, body := range []string{"one", "two"} {
		tx, err := Begin(cfg)
		if err != nil {
			t.Fatal(err)
		}
		if err := tx.WriteFile(logical, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := tx.Commit(); err != nil {
			t.Fatal(err)
		}
	}
	transactions, _ := cfg.Resolve(filepath.Join(cfg.StateHome(), "ilmango-v2", "transactions"))
	entries, err := os.ReadDir(transactions)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("retained %d transaction payloads, want 1", len(entries))
	}
}

func TestPreparePlanRecoversBeforeComparingFiles(t *testing.T) {
	cfg := testConfig(t, fixtureRepo(t))
	logical := filepath.Join(cfg.ShellDir(), "shell.qml")
	actual, _ := cfg.Resolve(logical)
	writeTestFile(t, actual, "original-user-file\n", 0o644)
	tx, err := Begin(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.WriteFile(logical, []byte("half-installed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan, err := PreparePlan(cfg)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(actual)
	if err != nil || string(data) != "original-user-file\n" {
		t.Fatalf("recovery happened after planning: content %q, %v", data, err)
	}
	if hasAction(plan, Write, logical) {
		t.Fatal("fresh plan would overwrite the recovered foreign file under preserve policy")
	}
	if len(plan.Impact.Warnings) == 0 || !strings.Contains(plan.Impact.Warnings[0], "Recovered") {
		t.Fatalf("recovery was not disclosed: %v", plan.Impact.Warnings)
	}
}

func TestDryRunRefusesToHidePendingRecovery(t *testing.T) {
	cfg := testConfig(t, fixtureRepo(t))
	tx, err := Begin(cfg)
	if err != nil {
		t.Fatal(err)
	}
	logical := filepath.Join(cfg.Home, ".config", "half")
	if err := tx.WriteFile(logical, []byte("partial"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg.DryRun = true
	if _, err := PreparePlan(cfg); err == nil || !strings.Contains(err.Error(), "needs recovery") {
		t.Fatalf("dry-run recovery error = %v", err)
	}
}

func TestRollbackPlanDisclosesEveryRestoration(t *testing.T) {
	cfg := testConfig(t, fixtureRepo(t))
	existing := filepath.Join(cfg.Home, ".config", "existing")
	actualExisting, _ := cfg.Resolve(existing)
	writeTestFile(t, actualExisting, "before", 0o644)
	created := filepath.Join(cfg.Home, ".config", "created")
	tx, err := Begin(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.WriteFile(existing, []byte("after"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := tx.WriteFile(created, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	cfg.Operation = Rollback
	plan, err := PreparePlan(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Impact.RestorePrevious != 1 || plan.Impact.RemoveCreated != 1 || len(plan.Actions) != 2 {
		t.Fatalf("rollback impact = %+v, actions = %v", plan.Impact, plan.Actions)
	}
}

func TestPathsAndJournalPointersCannotEscapeManagedHome(t *testing.T) {
	cfg := testConfig(t, fixtureRepo(t))
	if _, err := cfg.Resolve("/etc/passwd"); err == nil {
		t.Fatal("Resolve accepted a destination outside home")
	}
	if err := validateJournalPath(cfg, "/tmp/foreign/journal.json"); err == nil {
		t.Fatal("foreign journal pointer was accepted")
	}
	cfg.Root = "/"
	if err := cfg.Validate(); err == nil {
		t.Fatal("--root / was accepted as a sandbox")
	}
}

func TestRootSandboxRejectsSymlinkedParentEscape(t *testing.T) {
	cfg := testConfig(t, fixtureRepo(t))
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(cfg.Root, "home")); err != nil {
		t.Fatal(err)
	}
	if _, err := cfg.Resolve(filepath.Join(cfg.Home, ".config", "target")); err == nil || !strings.Contains(err.Error(), "symlinked parent") {
		t.Fatalf("sandbox parent escape error = %v", err)
	}
}

func hasAction(plan *Plan, kind ActionKind, path string) bool {
	for _, action := range plan.Actions {
		if action.Kind == kind && action.Path == path {
			return true
		}
	}
	return false
}

func writeTestFile(t *testing.T, path, body string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), mode); err != nil {
		t.Fatal(err)
	}
}
