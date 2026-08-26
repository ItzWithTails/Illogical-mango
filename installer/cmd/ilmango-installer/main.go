// Command ilmango-installer is the interactive installer for Illogical-mango.
//
// It presents the installation as a guided flow — preflight, options, review,
// progress, summary — and executes it through the shell phases under sdata/,
// which remain the single implementation of the installation itself.
//
// With --yes it runs the same plan without taking over the terminal, so CI and
// scripts share one code path with the interactive install.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	tea "charm.land/bubbletea/v2"

	"ilmango/internal/fsx"
	"ilmango/internal/installer"
	"ilmango/internal/installer/steps"
	"ilmango/internal/pkg"
	"ilmango/internal/run"
	"ilmango/internal/system"
	"ilmango/internal/ui"
	"ilmango/internal/ui/theme"
)

// Exit statuses. They are part of the CLI contract: scripts branch on them.
const (
	exitOK        = 0
	exitFailed    = 1
	exitUsage     = 2
	exitPreflight = 3
	exitCancelled = 130
)

type options struct {
	repo      string
	yes       bool
	dryRun    bool
	verbose   bool
	enable    string
	disable   string
	root      string
	uninstall bool
	update    bool
	rollback  bool
	changes   bool
	without   settingList
	settings  settingList
	list      bool
	listPkgs  bool
	showHelp  bool
}

func main() {
	os.Exit(start())
}

func start() int {
	opts, err := parseFlags(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n\n", err)
		usage(os.Stderr)
		return exitUsage
	}
	if opts.showHelp {
		usage(os.Stdout)
		return exitOK
	}
	if opts.list {
		listOptions(os.Stdout)
		return exitOK
	}
	if opts.listPkgs {
		listPackages(os.Stdout)
		return exitOK
	}

	repo, err := system.FindRepo(opts.repo)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		fmt.Fprintln(os.Stderr, "hint: run the installer from inside the Illogical-mango checkout, or pass --repo")
		return exitPreflight
	}

	cfg := installer.NewConfig()
	cfg.DryRun = opts.dryRun
	if err := applyOptionOverrides(&cfg, opts); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return exitUsage
	}

	operation := installer.OpInstall
	switch {
	case opts.uninstall:
		operation = installer.OpUninstall
	case opts.update:
		operation = installer.OpUpdate
	case opts.rollback:
		operation = installer.OpRollback
	case opts.changes:
		operation = installer.OpChanges
		// A report offers nothing to configure, so it never opens the picker.
		opts.yes = true
	}

	session := &ui.Session{
		Operation: operation,
		Theme:     theme.New(),
		Repo:      repo,
		Distro:    system.DetectDistro(),
		Config:    cfg,
		Steps:     installer.RegisteredSteps(operation),
		Width:     80,
	}

	env, err := buildEnv(session, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return exitPreflight
	}
	session.Env = env

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if opts.yes {
		return runHeadless(ctx, session, opts)
	}
	return runInteractive(session)
}

// runInteractive drives the Bubble Tea flow.
func runInteractive(session *ui.Session) int {
	app := ui.NewApp(session)

	program := tea.NewProgram(app)
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return exitFailed
	}

	switch {
	case app.Aborted():
		fmt.Fprintln(os.Stderr, "cancelled")
		return exitCancelled
	case app.Outcome().Err != nil:
		return exitFailed
	default:
		return exitOK
	}
}

// runHeadless performs preflight and the plan with plain output.
func runHeadless(ctx context.Context, session *ui.Session, opts options) int {
	head := ui.Headless{Out: os.Stdout, Theme: session.Theme, Verbose: opts.verbose}

	session.Checks = system.RunChecks(ctx, session.Repo)
	head.PrintChecks(session.Checks)
	if system.Blocking(session.Checks) {
		fmt.Fprintln(os.Stderr, "error: preflight failed; nothing was installed")
		return exitPreflight
	}

	if err := acquirePrivileges(ctx, session); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return exitPreflight
	}

	if err := head.Run(ctx, session); err != nil {
		if errors.Is(err, context.Canceled) {
			return exitCancelled
		}
		return exitFailed
	}
	return exitOK
}

// acquirePrivileges obtains a credential before the plan starts.
//
// Steps run privileged commands with prompting disabled, so the credential has
// to exist beforehand. Here the terminal is still the user's, so the escalation
// tool can ask for a password normally.
func acquirePrivileges(ctx context.Context, session *ui.Session) error {
	runner := session.Env.Runner

	if session.Config.DryRun || !session.Config.NeedsPrivileges() {
		return nil
	}
	if !runner.NeedsPrivileges() || runner.HasPrivileges(ctx) {
		return nil
	}

	name, args, ok := runner.AcquireCommand()
	if !ok {
		return nil // polkit-based tools authenticate per call
	}

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("could not obtain administrator privileges: %w", err)
	}
	return nil
}

// buildEnv assembles the only handles a step has on the machine. Both the
// command runner and the filesystem tree default to plan mode, so a missing
// --yes or an unset flag can never result in an unintended modification.
func buildEnv(session *ui.Session, opts options) (*installer.Env, error) {
	mode := run.ModeApply
	fsMode := fsx.ModeApply
	if opts.dryRun {
		mode, fsMode = run.ModePlan, fsx.ModePlan
	}

	var backup *fsx.Backup
	if session.Config.Effective(installer.OptBackup) && !opts.dryRun {
		backup = fsx.NewBackup(backupBase())
	}

	env := &installer.Env{
		Config: session.Config,
		Repo:   session.Repo,
		Distro: session.Distro,
		Runner: &run.Runner{Mode: mode},
		Home: fsx.Tree{
			Root:   opts.root,
			Mode:   fsMode,
			Backup: backup,
		},
		Backup: backup,
	}

	switch session.Operation {
	case installer.OpUninstall, installer.OpChanges:
		// Both read the record an install left behind: removal to know what it
		// owns, the change report to know what it wrote. Without it there is
		// nothing to work from, and guessing would be worse than refusing.
		// --root redirects writes, so the record of a redirected install
		// lives under that root too.
		manifest, err := steps.FindManifestUnder(opts.root)
		if err != nil {
			return nil, err
		}
		env.Manifest = manifest
		return env, nil

	case installer.OpRollback:
		// Restoring predates any record: it puts back what was there before.
		return env, nil
	}

	// An install builds the record as it writes, so every path it creates is
	// captured without any step having to remember to declare it.
	env.Manifest = installer.NewManifest(session.Repo.Version, session.Repo.Root)
	env.Home.Record = env.Manifest.Add
	return env, nil
}

// backupBase is where replaced files are kept.
func backupBase() string {
	if state := os.Getenv("XDG_STATE_HOME"); state != "" {
		return filepath.Join(state, "ilmango")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return os.TempDir()
	}
	return filepath.Join(home, ".local", "state", "ilmango")
}

func parseFlags(args []string) (options, error) {
	var opts options

	fs := flag.NewFlagSet("ilmango-installer", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&opts.repo, "repo", "", "path to the Illogical-mango checkout")
	fs.BoolVar(&opts.yes, "yes", false, "run without the interactive interface")
	fs.BoolVar(&opts.yes, "y", false, "shorthand for --yes")
	fs.BoolVar(&opts.dryRun, "dry-run", false, "walk the plan without executing any step")
	fs.BoolVar(&opts.verbose, "verbose", false, "stream all phase output (implies --yes)")
	fs.StringVar(&opts.enable, "enable", "", "comma-separated options to switch on")
	fs.StringVar(&opts.disable, "disable", "", "comma-separated options to switch off")
	fs.StringVar(&opts.root, "root", "", "write every file under this directory instead of / (for testing)")
	fs.Var(&opts.settings, "set", "select a value, as name=value; repeatable")
	fs.BoolVar(&opts.uninstall, "uninstall", false, "remove a previous installation")
	fs.BoolVar(&opts.update, "update", false, "pull the checkout forward, then reinstall from it")
	fs.BoolVar(&opts.rollback, "rollback", false, "restore the files the last run replaced")
	fs.BoolVar(&opts.changes, "changes", false, "list installed files you have edited")
	fs.Var(&opts.without, "without", "packages to leave out, comma-separated; repeatable")
	fs.BoolVar(&opts.listPkgs, "list-packages", false, "list the packages an install would install")
	fs.BoolVar(&opts.list, "list-options", false, "list the available options and exit")
	fs.BoolVar(&opts.showHelp, "help", false, "show this help")
	fs.BoolVar(&opts.showHelp, "h", false, "shorthand for --help")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			opts.showHelp = true
			return opts, nil
		}
		return opts, err
	}
	if rest := fs.Args(); len(rest) > 0 {
		return opts, fmt.Errorf("unexpected argument %q", rest[0])
	}
	if opts.verbose {
		opts.yes = true
	}
	return opts, nil
}

func applyOptionOverrides(cfg *installer.Config, opts options) error {
	if err := cfg.Apply(splitList(opts.enable), true); err != nil {
		return err
	}
	if err := cfg.Apply(splitList(opts.disable), false); err != nil {
		return err
	}

	for _, setting := range opts.settings {
		name, value, ok := strings.Cut(setting, "=")
		if !ok {
			return fmt.Errorf("--set expects name=value, got %q", setting)
		}
		if err := cfg.SetChoice(installer.OptionID(name), value); err != nil {
			return err
		}
	}

	known := knownPackages()
	for _, list := range opts.without {
		for _, name := range splitList(list) {
			// A typo here would silently install the package the user meant
			// to leave out, so an unknown name is refused rather than ignored.
			if !known[name] {
				return fmt.Errorf("--without: %q is not a package this installer offers (see --list-packages)", name)
			}
			cfg.SkipPackage(name, true)
		}
	}
	return nil
}

// knownPackages is every package the installer might install, on any
// distribution: --without is checked against all of them so that a name valid
// on another machine is not rejected here.
func knownPackages() map[string]bool {
	known := map[string]bool{}
	for _, family := range pkg.Families() {
		for _, name := range pkg.Packages(family, pkg.AllGroups()...) {
			known[name] = true
		}
	}
	return known
}

// settingList collects repeated --set flags.
type settingList []string

func (s *settingList) String() string { return strings.Join(*s, ",") }

func (s *settingList) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func splitList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return strings.Split(value, ",")
}

func listOptions(w io.Writer) {
	fmt.Fprintln(w, "Installation options (use --enable / --disable):")
	fmt.Fprintln(w)
	for _, o := range installer.Options() {
		state := "off"
		if o.Default {
			state = "on"
		}
		fmt.Fprintf(w, "  %-16s %-4s %s\n", o.ID, state, o.Description)
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Settings (use --set name=value):")
	fmt.Fprintln(w)
	for _, c := range installer.Choices() {
		var values []string
		for _, v := range c.Values {
			values = append(values, v.Value)
		}
		fmt.Fprintf(w, "  %-16s %-4s %s\n", c.ID, c.Default, strings.Join(values, " | "))
	}
}

func usage(w io.Writer) {
	fmt.Fprint(w, `Illogical-mango installer

Usage:
  ilmango-installer [flags]

Flags:
  --repo PATH        Path to the Illogical-mango checkout (default: search upwards from here)
  -y, --yes          Run without the interactive interface
  --verbose          Stream all phase output; implies --yes
  --uninstall        Remove a previous installation, keeping files you edited
  --update           Pull the checkout forward, then reinstall from it
  --rollback         Restore the files the last run replaced
  --changes          List installed files you have edited since
  --dry-run          Log every command and write nothing
  --root DIR         Write files under DIR instead of the real home (testing)
  --set NAME=VALUE   Select a value, e.g. --set aur-helper=paru; repeatable
  --enable LIST      Comma-separated options to switch on
  --disable LIST     Comma-separated options to switch off
  --without LIST     Packages to leave out, e.g. --without cava,mpv; repeatable
  --list-options     List the available options and exit
  --list-packages    List the packages an install would install, and exit
  -h, --help         Show this help

Examples:
  ilmango-installer                              Guided installation
  ilmango-installer -y                           Unattended, defaults
  ilmango-installer -y --disable deps,fonts      Unattended, skip packages and fonts
  ilmango-installer -y --set aur-helper=paru     Unattended, require paru for AUR packages
  ilmango-installer --dry-run                    Inspect the plan, change nothing
  ilmango-installer --update                     Pull and reinstall
  ilmango-installer --changes                    What have I edited?
  ilmango-installer --uninstall                  Remove what a previous run installed

Exit status:
  0 success   1 a step failed   2 bad usage   3 preflight failed   130 cancelled
`)
}

// listPackages prints what an install would install, grouped the way the
// options are, so that --without has something to name.
func listPackages(w io.Writer) {
	family := string(system.DetectDistro().Family)
	if !pkg.KnownFamily(family) {
		fmt.Fprintf(w, "No package list for %s; dependencies are installed by hand on this distribution.\n", family)
		return
	}

	fmt.Fprintf(w, "Packages for %s (leave one out with --without NAME):\n\n", family)
	for _, group := range pkg.AllGroups() {
		names := pkg.Packages(family, group)
		if len(names) == 0 {
			continue
		}
		fmt.Fprintf(w, "  %s\n", group)
		for _, name := range names {
			marker := " "
			if pkg.IsCritical(name) {
				marker = "*"
			}
			fmt.Fprintf(w, "    %s %s\n", marker, name)
		}
		fmt.Fprintln(w)
	}
	fmt.Fprintln(w, "  * the shell visibly breaks without it")
}
