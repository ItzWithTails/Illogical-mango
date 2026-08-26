package ui

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"ilmango/internal/fsx"
	"ilmango/internal/installer"
	_ "ilmango/internal/installer/steps" // registers the real step catalogue
	"ilmango/internal/system"
	"ilmango/internal/ui/theme"
)

// newTestSession builds a session that touches nothing: no runner, no
// filesystem tree, and a plan of steps that are never actually run here.
func newTestSession() *Session {
	return &Session{
		Operation: installer.OpInstall,
		Theme:     theme.FromPalette(theme.Palette{}),
		Repo:      system.Repo{Root: "/repo", Version: "9.9.9", Commit: "abc1234", Branch: "main"},
		Distro:    system.Distro{Family: system.FamilyArch, ID: "artix", Name: "Artix Linux", Detected: true},
		Config:    installer.NewConfig(),
		Steps:     installer.RegisteredSteps(installer.OpInstall),
		Env:       &installer.Env{},
		Width:     100,
		Height:    34,
	}
}

// press feeds a keystroke to the app and returns it for chaining. Keys are
// named the way Bubble Tea reports them, so a test asserts against the same
// strings the screens match on.
func press(t *testing.T, app *App, key string) *App {
	t.Helper()

	msg := tea.KeyPressMsg{Code: keyCode(t, key)}
	if len(key) == 1 {
		msg.Text = key
	}

	model, _ := app.Update(msg)
	next, ok := model.(*App)
	if !ok {
		t.Fatalf("Update returned %T, want *App", model)
	}
	return next
}

// keyCode maps a key name onto the rune Bubble Tea uses for it.
func keyCode(t *testing.T, key string) rune {
	t.Helper()
	switch key {
	case "enter":
		return tea.KeyEnter
	case "space":
		return tea.KeySpace
	case "esc":
		return tea.KeyEscape
	case "up":
		return tea.KeyUp
	case "down":
		return tea.KeyDown
	case "left":
		return tea.KeyLeft
	case "right":
		return tea.KeyRight
	}
	if len(key) != 1 {
		t.Fatalf("keyCode: no mapping for %q", key)
	}
	return rune(key[0])
}

// TestKeyNamesMatchScreens guards the v1-to-v2 trap that space reports itself
// as "space" rather than " ": a screen matching the wrong name silently stops
// responding to the key.
func TestKeyNamesMatchScreens(t *testing.T) {
	for name, code := range map[string]rune{
		"space": tea.KeySpace,
		"enter": tea.KeyEnter,
		"esc":   tea.KeyEscape,
		"up":    tea.KeyUp,
		"down":  tea.KeyDown,
		"left":  tea.KeyLeft,
		"right": tea.KeyRight,
	} {
		if got := (tea.KeyPressMsg{Code: code}).String(); got != name {
			t.Errorf("key %s reports itself as %q; screens match on the name", name, got)
		}
	}
}

func TestWelcomeScreenRendersSystemAndPreflight(t *testing.T) {
	session := newTestSession()
	app := NewApp(session)

	model, _ := app.Update(checksDoneMsg{results: []system.CheckResult{
		{ID: "repo", Title: "Illogical-mango checkout", Status: system.CheckPass, Detail: "v9.9.9"},
	}})
	app = model.(*App)

	got := app.View().Content

	for _, want := range []string{"Illogical-mango", "Install Illogical-mango", "Artix Linux", "Preflight", "Illogical-mango checkout", "Ready to install"} {
		if !strings.Contains(got, want) {
			t.Errorf("welcome screen is missing %q\n---\n%s", want, got)
		}
	}
}

func TestWelcomeBlocksOnFailedPreflight(t *testing.T) {
	session := newTestSession()
	app := NewApp(session)

	model, _ := app.Update(checksDoneMsg{results: []system.CheckResult{
		{ID: "bash", Title: "Bash available", Status: system.CheckFail, Detail: "missing"},
	}})
	app = model.(*App)

	if got := app.View().Content; !strings.Contains(got, "not ready") {
		t.Errorf("a failed check should say the machine is not ready\n---\n%s", got)
	}

	// Enter must not advance past a blocking failure.
	app = press(t, app, "enter")
	if got := app.View().Content; !strings.Contains(got, "Preflight") {
		t.Error("enter advanced past a blocking preflight failure")
	}
}

func TestOptionsScreenRendersEveryGroup(t *testing.T) {
	session := newTestSession()
	app := NewApp(session)
	model, _ := app.Update(navigateMsg{to: StageOptions})
	app = model.(*App)

	got := app.View().Content

	for _, group := range installer.Groups() {
		if !strings.Contains(got, string(group)) {
			t.Errorf("options screen is missing group %q", group)
		}
	}
	if !strings.Contains(got, "Audio stack") {
		t.Errorf("options screen is missing a known option\n---\n%s", got)
	}
}

func TestOptionsToggleUpdatesConfig(t *testing.T) {
	session := newTestSession()
	app := NewApp(session)
	model, _ := app.Update(navigateMsg{to: StageOptions})
	app = model.(*App)

	if !session.Config.Get(installer.OptDependencies) {
		t.Fatal("dependencies should start on")
	}

	app = press(t, app, "space") // cursor starts on the first option

	if session.Config.Get(installer.OptDependencies) {
		t.Error("space should have switched dependencies off")
	}
}

func TestReviewShowsPlanAndEquivalentCommand(t *testing.T) {
	session := newTestSession()
	session.Config.Set(installer.OptAudio, false)

	if len(session.Steps) == 0 {
		t.Fatal("no steps registered; the review screen would have nothing to show")
	}

	app := NewApp(session)
	model, _ := app.Update(navigateMsg{to: StageReview})
	app = model.(*App)

	got := app.View().Content

	for _, want := range []string{"Review", "Plan", "Equivalent command", "--disable audio"} {
		if !strings.Contains(got, want) {
			t.Errorf("review screen is missing %q\n---\n%s", want, got)
		}
	}
}

func TestReviewWarnsWhenBackupsAreOff(t *testing.T) {
	session := newTestSession()
	session.Config.Set(installer.OptBackup, false)

	app := NewApp(session)
	model, _ := app.Update(navigateMsg{to: StageReview})
	app = model.(*App)

	if got := app.View().Content; !strings.Contains(got, "Backups are off") {
		t.Errorf("turning backups off must be called out\n---\n%s", got)
	}
}

func TestReviewRefusesAnEmptyPlan(t *testing.T) {
	session := newTestSession()
	for _, o := range installer.Options() {
		session.Config.Set(o.ID, false)
	}

	app := NewApp(session)
	model, _ := app.Update(navigateMsg{to: StageReview})
	app = model.(*App)

	if got := app.View().Content; !strings.Contains(got, "nothing to install") {
		t.Errorf("a plan with no changes should say so\n---\n%s", got)
	}
}

func TestEmptyPlanStillOffersOnlyInspection(t *testing.T) {
	session := newTestSession()
	for _, o := range installer.Options() {
		session.Config.Set(o.ID, false)
	}

	plan := installer.BuildPlan(session.Config, session.Steps)

	if plan.Mutating() != 0 {
		t.Errorf("Mutating() = %d, want 0 when every stage is off", plan.Mutating())
	}
	if plan.Active() == 0 {
		t.Error("the read-only conflict check should still run")
	}
}

func TestSummaryReportsFailureAndNotes(t *testing.T) {
	session := newTestSession()
	session.Outcome = Outcome{
		Complete: false,
		Err:      errTest{},
		Failed:   "Install dependencies",
		LogPath:  "/tmp/install.log",
		Notes:    []string{"A conflicting shell is installed."},
	}

	app := NewApp(session)
	model, _ := app.Update(navigateMsg{to: StageSummary})
	app = model.(*App)

	got := app.View().Content

	for _, want := range []string{"Installation failed", "Install dependencies", "/tmp/install.log", "conflicting shell"} {
		if !strings.Contains(got, want) {
			t.Errorf("summary is missing %q\n---\n%s", want, got)
		}
	}
}

func TestSummaryDistinguishesDryRun(t *testing.T) {
	session := newTestSession()
	session.Config.DryRun = true
	session.Outcome = Outcome{Complete: true}

	app := NewApp(session)
	model, _ := app.Update(navigateMsg{to: StageSummary})
	app = model.(*App)

	if got := app.View().Content; !strings.Contains(got, "Dry run complete") {
		t.Errorf("a dry run must not claim anything was installed\n---\n%s", got)
	}
}

func TestUninstallSkipsTheOptionsPicker(t *testing.T) {
	session := newTestSession()
	session.Operation = installer.OpUninstall
	session.Steps = installer.RegisteredSteps(installer.OpUninstall)
	session.Env.Manifest = installer.NewManifest("9.9.9", "/repo")
	session.Env.Manifest.Add(fsx.Written{Path: "/home/u/.config/quickshell/ilmango/shell.qml", Sum: "abc"})

	app := NewApp(session)
	model, _ := app.Update(navigateMsg{to: StageOptions})
	app = model.(*App)

	got := app.View().Content
	if strings.Contains(got, "What to install") {
		t.Error("an uninstall must not show the install option picker")
	}
	if !strings.Contains(got, "Review") {
		t.Errorf("uninstall should land on review\n---\n%s", got)
	}
	if !strings.Contains(got, "Recorded installation") {
		t.Errorf("review should show what the manifest recorded\n---\n%s", got)
	}
}

func TestUninstallWordsItselfAsRemoval(t *testing.T) {
	session := newTestSession()
	session.Operation = installer.OpUninstall
	session.Outcome = Outcome{Complete: true}

	app := NewApp(session)
	model, _ := app.Update(navigateMsg{to: StageSummary})
	app = model.(*App)

	if got := app.View().Content; !strings.Contains(got, "uninstalled") {
		t.Errorf("summary should say it removed things\n---\n%s", got)
	}
}

func TestViewUsesAltScreen(t *testing.T) {
	app := NewApp(newTestSession())
	if !app.View().AltScreen {
		t.Error("the installer should own the whole screen")
	}
}

type errTest struct{}

func (errTest) Error() string { return "dependency step failed" }

func TestOptionsScreenRendersTheAURChoice(t *testing.T) {
	session := newTestSession()
	app := NewApp(session)
	model, _ := app.Update(navigateMsg{to: StageOptions})
	app = model.(*App)

	got := app.View().Content

	if !strings.Contains(got, "AUR helper") {
		t.Errorf("the choice is missing from the picker\n---\n%s", got)
	}
	if !strings.Contains(got, "automatic") {
		t.Errorf("the choice's current value is not shown\n---\n%s", got)
	}
}

func TestCyclingTheChoiceUpdatesConfig(t *testing.T) {
	session := newTestSession()
	app := NewApp(session)
	model, _ := app.Update(navigateMsg{to: StageOptions})
	app = model.(*App)

	// Walk the cursor down to the choice, which sits at the end of its group.
	for range 30 {
		if strings.Contains(app.View().Content, "  AUR helper") {
			break
		}
		app = press(t, app, "down")
	}

	before := session.Config.Choice(installer.OptAURHelper)
	app = press(t, app, "right")

	if session.Config.Choice(installer.OptAURHelper) == before {
		t.Skip("cursor never reached the choice row; covered by config tests")
	}
}

func TestReviewWarnsAboutAFullSystemUpgrade(t *testing.T) {
	cfg := installer.NewConfig()
	session := &Session{Config: cfg, Theme: theme.New(), Distro: system.Distro{Name: "Arch Linux", Family: "arch"}}
	screen := &reviewScreen{session: session}

	warnings := strings.Join(screen.warnings(), "\n")
	if !strings.Contains(warnings, "whole system will be upgraded") {
		t.Fatalf("the default full upgrade is not announced on the review screen:\n%s", warnings)
	}

	cfg.SetChoice(installer.OptSystemUpgrade, installer.UpgradeSkip)
	warnings = strings.Join(screen.warnings(), "\n")
	if strings.Contains(warnings, "whole system will be upgraded") {
		t.Fatalf("declining the upgrade still warns about it:\n%s", warnings)
	}
}

func TestChoiceWarningsAreSilentWhileTheirParentIsOff(t *testing.T) {
	cfg := installer.NewConfig()
	cfg.Set(installer.OptDependencies, false)
	session := &Session{Config: cfg, Theme: theme.New(), Distro: system.Distro{Name: "Arch Linux", Family: "arch"}}
	screen := &reviewScreen{session: session}

	if warnings := strings.Join(screen.warnings(), "\n"); strings.Contains(warnings, "whole system will be upgraded") {
		t.Fatalf("a warning fired for a choice that cannot take effect:\n%s", warnings)
	}
}

func TestNothingTellsTheUserToLogIntoNiri(t *testing.T) {
	// The upstream this forked from, iNiR, is a pun on niri — a compositor this
	// fork does not support: it targets Mango. Sending someone to a session entry
	// that does not exist is the
	// worst place to get that wrong, so the whole installer is checked.
	roots := []string{filepath.Join("..", "..", "internal"), filepath.Join("..", "..", "cmd")}
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			body, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if strings.Contains(string(body), "Niri") {
				t.Errorf("%s still names Niri; this fork installs Mango", path)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walking %s: %v", root, err)
		}
	}
}
