package v2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ilmango/internal/pkg"
	"ilmango/internal/run"
	"ilmango/internal/system"
)

type PackageInstaller interface {
	Install(context.Context, Config, []string, func(Event)) error
}

type ArchPackages struct {
	Runner *run.Runner
}

func (a ArchPackages) Install(ctx context.Context, cfg Config, names []string, emit func(Event)) error {
	if len(names) == 0 {
		return nil
	}
	if cfg.Root != "" {
		return fmt.Errorf("internal safety check: package execution is forbidden with --root")
	}
	distro := system.DetectDistro()
	if distro.Family != system.FamilyArch {
		return nil
	}
	manager, err := pkg.FindManager(string(distro.Family))
	if err != nil {
		return err
	}
	runner := a.Runner
	if runner == nil {
		runner = &run.Runner{Mode: run.ModeApply}
	}
	if cfg.SystemUpgrade {
		emit(Event{Step: "packages", Detail: "upgrading the Arch system before dependency installation"})
		if err := manager.Refresh(ctx, runner, true); err != nil {
			return fmt.Errorf("system upgrade failed: %w", err)
		}
	}
	failed, err := manager.Install(ctx, runner, names, func(progress pkg.Progress) {
		emit(Event{Step: "packages", Detail: progress.Action + " " + progress.Name,
			Done: progress.Done, Total: progress.Total})
	})
	if err != nil {
		return err
	}
	if len(failed) > 0 {
		return fmt.Errorf("required dependencies were not installed: %s", strings.Join(failed, ", "))
	}

	// A successful exit is not enough: helpers and hooks can return zero while
	// leaving a requested package unavailable. Verify the resulting state.
	present, err := manager.InstalledSet(ctx, runner)
	if err != nil {
		return fmt.Errorf("could not verify installed dependencies: %w", err)
	}
	var missing []string
	for _, name := range names {
		if !present[name] {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("package manager finished, but these dependencies are still missing: %s", strings.Join(missing, ", "))
	}
	return nil
}

type Engine struct {
	Packages PackageInstaller
	Emit     func(Event)
}

func (e Engine) event(ev Event) {
	if e.Emit != nil {
		e.Emit(ev)
	}
}

func (e Engine) Run(ctx context.Context, plan *Plan) (result Result) {
	started := time.Now()
	result = Result{Operation: plan.Config.Operation, Warnings: append([]string{}, plan.Impact.Warnings...)}
	defer func() { result.Duration = time.Since(started) }()

	if err := ctx.Err(); err != nil {
		result.Err = err
		return result
	}
	if plan.Config.Operation == Status {
		result.Success = true
		result.Kept = plan.Impact.KeepModified
		return result
	}
	if plan.Config.DryRun {
		result.Success = true
		result.Changed = plan.Impact.Mutations()
		result.Kept = plan.Impact.KeepModified
		return result
	}
	transcript, logicalLog, err := openTranscript(plan.Config)
	if err != nil {
		result.Err = fmt.Errorf("opening persistent transcript: %w", err)
		return result
	}
	defer transcript.Close()
	result.LogPath = logicalLog
	writeLog := func(line string) {
		fmt.Fprintf(transcript, "%s %s\n", time.Now().Format("15:04:05.000"), line)
	}
	originalEmit := e.Emit
	e.Emit = func(ev Event) {
		writeLog(fmt.Sprintf("[%s] %s", ev.Step, ev.Detail))
		if originalEmit != nil {
			originalEmit(ev)
		}
	}
	if e.Packages == nil {
		e.Packages = ArchPackages{}
	}
	if arch, ok := e.Packages.(ArchPackages); ok {
		if arch.Runner == nil {
			arch.Runner = &run.Runner{Mode: run.ModeApply}
		}
		previousLog := arch.Runner.Log
		arch.Runner.Log = func(line string) {
			writeLog(line)
			if previousLog != nil {
				previousLog(line)
			}
		}
		e.Packages = arch
	}
	writeLog(fmt.Sprintf("operation=%s actions=%d packages=%d", plan.Config.Operation, len(plan.Actions), len(plan.Impact.Packages)))
	if plan.Config.Operation == Rollback {
		count, err := RollbackLast(plan.Config)
		result.Changed, result.Err = count, err
		result.Success = err == nil
		return result
	}

	recovered, err := RecoverActive(plan.Config)
	if err != nil {
		result.Err = fmt.Errorf("recovering an interrupted run: %w", err)
		return result
	}
	if recovered {
		result.Err = errors.New("an interrupted transaction was recovered after review; destination state changed, so review a freshly computed plan")
		return result
	}

	if len(plan.Impact.Packages) > 0 {
		installer := e.Packages
		e.event(Event{Step: "packages", Detail: fmt.Sprintf("installing %d verified dependencies", len(plan.Impact.Packages))})
		if err := installer.Install(ctx, plan.Config, plan.Impact.Packages, e.event); err != nil {
			result.Err = fmt.Errorf("dependency installation failed before files were changed: %w", err)
			return result
		}
	}

	tx, err := Begin(plan.Config)
	if err != nil {
		result.Err = err
		return result
	}
	fail := func(err error) Result {
		if rollbackErr := tx.Rollback(err); rollbackErr != nil {
			result.Err = fmt.Errorf("%w; %v", err, rollbackErr)
		} else {
			result.Err = fmt.Errorf("%w (all filesystem changes were rolled back)", err)
		}
		return result
	}

	total := len(plan.Actions) + 1
	for i, action := range plan.Actions {
		if err := ctx.Err(); err != nil {
			return fail(err)
		}
		e.event(Event{Step: string(action.Kind), Detail: action.Path, Done: i, Total: total})
		switch action.Kind {
		case Write:
			err = tx.WriteFile(action.Path, action.Desired.Data, action.Desired.Mode)
		case Link:
			err = tx.Symlink(action.Path, action.Desired.Link)
		case Remove:
			err = tx.Remove(action.Path)
		default:
			err = fmt.Errorf("unknown action %q", action.Kind)
		}
		if err != nil {
			return fail(fmt.Errorf("%s %s: %w", action.Kind, action.Path, err))
		}
		result.Changed++
	}

	manifestPath := ManifestPath(plan.Config)
	if plan.Config.Operation == Uninstall {
		err = tx.Remove(manifestPath)
	} else {
		data, marshalErr := marshalManifest(plan.Manifest)
		if marshalErr != nil {
			return fail(marshalErr)
		}
		err = tx.WriteFile(manifestPath, data, 0o600)
	}
	if err != nil {
		return fail(fmt.Errorf("updating installation record: %w", err))
	}
	if plan.LegacyPath != "" {
		if err := tx.Remove(plan.LegacyPath); err != nil {
			return fail(fmt.Errorf("migrating legacy installation record: %w", err))
		}
		result.Changed++
	}
	if err := tx.Commit(); err != nil {
		return fail(fmt.Errorf("committing transaction: %w", err))
	}
	result.Success = true
	result.Kept = plan.Impact.KeepModified
	e.event(Event{Step: "done", Detail: "transaction committed", Done: total, Total: total})
	return result
}

func openTranscript(cfg Config) (*os.File, string, error) {
	logicalDir := filepath.Join(cfg.StateHome(), "ilmango-v2", "logs")
	actualDir, err := cfg.Resolve(logicalDir)
	if err != nil {
		return nil, "", err
	}
	if err := os.MkdirAll(actualDir, 0o700); err != nil {
		return nil, "", err
	}
	name := time.Now().UTC().Format("20060102T150405.000000000Z") + ".log"
	file, err := os.OpenFile(filepath.Join(actualDir, name), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, "", err
	}
	return file, filepath.Join(logicalDir, name), nil
}

func marshalManifest(m *Manifest) ([]byte, error) {
	if m == nil {
		return nil, errors.New("internal error: missing target manifest")
	}
	data, err := jsonMarshalIndent(m)
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

// Kept tiny and replaceable in tests without exposing the atomic writer.
var jsonMarshalIndent = func(v any) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}
