// Command ilmango-installer-v2 is the transactional Bubble Tea installer.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	tea "charm.land/bubbletea/v2"
	"github.com/mattn/go-isatty"

	"ilmango/internal/run"
	v2 "ilmango/internal/v2"
)

const (
	exitOK        = 0
	exitFailed    = 1
	exitUsage     = 2
	exitPreflight = 3
	exitCancelled = 130
)

func main() { os.Exit(start(os.Args[1:], os.Stdout, os.Stderr)) }

func start(args []string, out, errOut io.Writer) int {
	cfg, help, err := parse(args, errOut)
	if help {
		usage(out)
		return exitOK
	}
	if err != nil {
		fmt.Fprintln(errOut, "error:", err)
		usage(errOut)
		return exitUsage
	}
	if cfg.Root != "" {
		// --root is a filesystem sandbox, not a chroot. Disabling every external
		// action here makes the safe behavior the convenient behavior.
		cfg.Packages, cfg.SystemUpgrade = false, false
	}
	if err := cfg.Validate(); err != nil {
		fmt.Fprintln(errOut, "error:", err)
		return exitUsage
	}

	mode := run.ModeApply
	if cfg.DryRun {
		mode = run.ModePlan
	}
	runner := &run.Runner{Mode: mode}
	if cfg.Verbose {
		runner.Log = func(line string) { fmt.Fprintln(errOut, line) }
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	_ = ctx // Bubble Tea owns interactive signals; headless uses it below.

	if cfg.Yes {
		plan, err := v2.PreparePlan(cfg)
		if err != nil {
			fmt.Fprintln(errOut, "error:", err)
			return exitPreflight
		}
		printPlan(out, plan, cfg.Verbose)
		eventCount := 0
		result := (v2.Engine{Packages: v2.ArchPackages{Runner: runner}, Emit: func(ev v2.Event) {
			eventCount++
			if ev.Detail != "" && (cfg.Verbose || ev.Step == "packages" || ev.Step == "done" || eventCount%100 == 0) {
				fmt.Fprintf(out, "[%s] %s\n", ev.Step, ev.Detail)
			}
		}}).Run(ctx, plan)
		printResult(out, errOut, result)
		return resultExit(result)
	}

	if file, ok := out.(*os.File); !ok || (!isatty.IsTerminal(file.Fd()) && !isatty.IsCygwinTerminal(file.Fd())) {
		fmt.Fprintln(errOut, "error: interactive UI needs a terminal; use --yes for scripts and CI")
		return exitUsage
	}
	app := v2.NewUI(cfg, runner)
	program := tea.NewProgram(app)
	if _, err := program.Run(); err != nil {
		fmt.Fprintln(errOut, "error:", err)
		return exitFailed
	}
	if app.Aborted() {
		fmt.Fprintln(errOut, "cancelled; any active filesystem transaction was rolled back")
		return exitCancelled
	}
	printResult(out, errOut, app.Result)
	return resultExit(app.Result)
}

func parse(args []string, errOut io.Writer) (v2.Config, bool, error) {
	cfg := v2.DefaultConfig()
	op := "install"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		op, args = args[0], args[1:]
	}
	cfg.Operation = v2.Operation(op)
	fs := flag.NewFlagSet("ilmango-installer-v2", flag.ContinueOnError)
	fs.SetOutput(errOut)
	repo := fs.String("repo", "", "path to the source checkout")
	home := fs.String("home", cfg.Home, "logical user home")
	root := fs.String("root", "", "redirect every filesystem access below this test root")
	preset := fs.String("preset", string(cfg.Preset), "minimal, recommended or full")
	conflict := fs.String("conflict", string(cfg.ConflictPolicy), "preserve or replace existing files")
	layout := fs.String("layout", cfg.KeyboardLayout, "keyboard layout, e.g. system or us,de")
	language := fs.String("language", cfg.Language, "auto, en or ru")
	packages := fs.Bool("packages", cfg.Packages, "install verified dependencies on Arch")
	noPackages := fs.Bool("no-packages", false, "do not invoke a package manager")
	mango := fs.Bool("mango", cfg.MangoHook, "install a marked, reversible Mango include")
	upgrade := fs.Bool("system-upgrade", false, "explicitly allow a full Arch upgrade")
	dry := fs.Bool("dry-run", false, "show the plan and execute nothing")
	yes := fs.Bool("yes", false, "non-interactive mode")
	fs.BoolVar(yes, "y", false, "non-interactive mode")
	verbose := fs.Bool("verbose", false, "print external command output")
	noColor := fs.Bool("no-color", false, "disable ANSI colors")
	help := fs.Bool("help", false, "show help")
	fs.BoolVar(help, "h", false, "show help")
	if err := fs.Parse(args); err != nil {
		return cfg, false, err
	}
	if fs.NArg() != 0 {
		return cfg, false, fmt.Errorf("unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}
	cfg.Repo, cfg.Home, cfg.Root = *repo, *home, *root
	cfg.Preset, cfg.ConflictPolicy = v2.Preset(*preset), v2.ConflictPolicy(*conflict)
	cfg.KeyboardLayout, cfg.Language = *layout, *language
	cfg.Packages, cfg.MangoHook, cfg.SystemUpgrade = *packages, *mango, *upgrade
	if *noPackages {
		cfg.Packages = false
	}
	cfg.DryRun, cfg.Yes, cfg.Verbose, cfg.NoColor = *dry, *yes, *verbose, *noColor
	return cfg, *help, nil
}

func usage(w io.Writer) {
	fmt.Fprintln(w, `Usage: ilmango-installer-v2 [install|update|uninstall|rollback|status] [options]

The operation is one positional subcommand, so contradictory operation flags
cannot be combined. Install/update use the current checkout exactly as it is;
the installer never performs a hidden git pull.

Core options:
  --preset minimal|recommended|full
  --conflict preserve|replace
  --layout system|us|us,de       (interactive entry is also supported)
  --packages / --no-packages     verified Arch recipe only
  --system-upgrade               explicit, never the default
  --root PATH                    filesystem-only test sandbox
  --dry-run --yes                reproducible non-interactive preview
  --language auto|en|ru
  --no-color

Uninstall never removes packages and keeps every modified owned file.
Rollback restores the complete previous filesystem state of the last run.`)
}

func printPlan(out io.Writer, plan *v2.Plan, verbose bool) {
	i := plan.Impact
	if plan.Config.Operation == v2.Rollback {
		fmt.Fprintf(out, "Plan: restore previous %d, remove newly-created %d\n", i.RestorePrevious, i.RemoveCreated)
	} else {
		fmt.Fprintf(out, "Plan: create %d, replace %d, remove stale %d, keep modified %d, unchanged %d\n",
			i.Create, i.Replace, i.RemoveStale, i.KeepModified, i.Unchanged)
	}
	if len(i.Packages) > 0 {
		fmt.Fprintf(out, "Packages (%d): %s\n", len(i.Packages), strings.Join(i.Packages, " "))
	}
	for _, warning := range i.Warnings {
		fmt.Fprintln(out, "Warning:", warning)
	}
	details := i.Details
	if !verbose && len(details) > 30 {
		details = details[:30]
	}
	for _, detail := range details {
		fmt.Fprintln(out, "Detail:", detail)
	}
	if len(details) != len(i.Details) {
		fmt.Fprintf(out, "Detail: … %d more (use --verbose to list all)\n", len(i.Details)-len(details))
	}
	actions := plan.Actions
	if !verbose && len(actions) > 30 {
		actions = actions[:30]
	}
	for _, action := range actions {
		fmt.Fprintf(out, "  %-7s %s\n", action.Kind, action.Path)
	}
	if len(actions) != len(plan.Actions) {
		fmt.Fprintf(out, "  … %d more actions (use --verbose to list all)\n", len(plan.Actions)-len(actions))
	}
}

func printResult(out, errOut io.Writer, result v2.Result) {
	if result.Err != nil {
		fmt.Fprintln(errOut, "failed:", result.Err)
		if result.LogPath != "" {
			fmt.Fprintln(errOut, "Transcript:", result.LogPath)
		}
		return
	}
	if !result.Success {
		return
	}
	fmt.Fprintf(out, "%s complete: %d paths changed, %d modified paths kept (%s)\n",
		result.Operation, result.Changed, result.Kept, result.Duration.Round(1e6))
	if result.LogPath != "" {
		fmt.Fprintln(out, "Transcript:", result.LogPath)
	}
}

func resultExit(result v2.Result) int {
	if errors.Is(result.Err, context.Canceled) {
		return exitCancelled
	}
	if result.Err != nil || !result.Success {
		return exitFailed
	}
	return exitOK
}
