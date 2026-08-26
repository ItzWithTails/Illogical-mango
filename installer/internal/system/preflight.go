package system

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// CheckStatus is the outcome of a preflight check.
type CheckStatus int

const (
	// CheckPass means the condition is satisfied.
	CheckPass CheckStatus = iota
	// CheckWarn means installation can proceed but something is off.
	CheckWarn
	// CheckFail means installation cannot proceed.
	CheckFail
)

// CheckResult is a completed preflight check.
type CheckResult struct {
	ID     string
	Title  string
	Status CheckStatus
	Detail string
}

// Check is one precondition verified before any step runs. Add a check by
// appending to the slice returned by Checks.
type Check struct {
	ID    string
	Title string
	Run   func(ctx context.Context, repo Repo) CheckResult
}

// Checks returns the preflight suite in execution order.
func Checks() []Check {
	return []Check{
		{ID: "repo", Title: "Illogical-mango checkout", Run: checkRepo},
		{ID: "bash", Title: "Bash available", Run: checkBash},
		{ID: "user", Title: "Running as a normal user", Run: checkNotRoot},
		{ID: "privilege", Title: "Privilege escalation", Run: checkPrivilege},
		{ID: "disk", Title: "Free disk space", Run: checkDiskSpace},
		{ID: "session", Title: "Wayland session", Run: checkSession},
	}
}

// RunChecks executes the suite and returns every result.
func RunChecks(ctx context.Context, repo Repo) []CheckResult {
	checks := Checks()
	results := make([]CheckResult, 0, len(checks))
	for _, c := range checks {
		if err := ctx.Err(); err != nil {
			break
		}
		results = append(results, c.Run(ctx, repo))
	}
	return results
}

// Blocking reports whether any result prevents installation.
func Blocking(results []CheckResult) bool {
	for _, r := range results {
		if r.Status == CheckFail {
			return true
		}
	}
	return false
}

func checkRepo(_ context.Context, repo Repo) CheckResult {
	res := CheckResult{ID: "repo", Title: "Illogical-mango checkout"}

	// The QML entry point is what Quickshell loads; without it there is
	// nothing to install, whatever else the directory contains.
	if _, err := os.Stat(filepath.Join(repo.Root, "shell.qml")); err != nil {
		res.Status = CheckFail
		res.Detail = "shell.qml missing — this is not an Illogical-mango checkout"
		return res
	}
	if _, err := os.Stat(filepath.Join(repo.Root, "dots")); err != nil {
		res.Status = CheckWarn
		res.Detail = "no dots/ directory — dotfiles will not be installed"
		return res
	}

	res.Status = CheckPass
	res.Detail = fmt.Sprintf("v%s at %s", repo.Version, shortenHome(repo.Root))
	return res
}

func checkBash(_ context.Context, _ Repo) CheckResult {
	res := CheckResult{ID: "bash", Title: "Bash available"}
	path, err := exec.LookPath("bash")
	if err != nil {
		res.Status = CheckFail
		res.Detail = "bash is required to run the install phases"
		return res
	}
	res.Status = CheckPass
	res.Detail = path
	return res
}

func checkNotRoot(_ context.Context, _ Repo) CheckResult {
	res := CheckResult{ID: "user", Title: "Running as a normal user"}
	if os.Geteuid() == 0 {
		res.Status = CheckFail
		res.Detail = "run as your own user; the installer escalates only where needed"
		return res
	}
	res.Status = CheckPass
	res.Detail = currentUserName()
	return res
}

func checkPrivilege(ctx context.Context, _ Repo) CheckResult {
	res := CheckResult{ID: "privilege", Title: "Privilege escalation"}
	for _, tool := range []string{"sudo", "doas", "run0"} {
		if path, err := exec.LookPath(tool); err == nil {
			res.Status = CheckPass
			res.Detail = path
			return res
		}
	}
	res.Status = CheckWarn
	res.Detail = "no sudo/doas/run0 found — package installation will fail"
	return res
}

// minFreeBytes is a deliberately generous floor: dependency installation pulls
// in Qt and a full font set.
const minFreeBytes = 4 << 30 // 4 GiB

func checkDiskSpace(_ context.Context, _ Repo) CheckResult {
	res := CheckResult{ID: "disk", Title: "Free disk space"}
	home, err := os.UserHomeDir()
	if err != nil {
		res.Status = CheckWarn
		res.Detail = "could not resolve home directory"
		return res
	}

	var stat syscall.Statfs_t
	if err := syscall.Statfs(home, &stat); err != nil {
		res.Status = CheckWarn
		res.Detail = "could not read filesystem statistics"
		return res
	}

	free := stat.Bavail * uint64(stat.Bsize)
	res.Detail = fmt.Sprintf("%s available on %s", humanBytes(free), shortenHome(home))
	if free < minFreeBytes {
		res.Status = CheckWarn
		res.Detail = fmt.Sprintf("%s — dependencies may need more", res.Detail)
		return res
	}
	res.Status = CheckPass
	return res
}

func checkSession(_ context.Context, _ Repo) CheckResult {
	res := CheckResult{ID: "session", Title: "Wayland session"}
	switch {
	case os.Getenv("WAYLAND_DISPLAY") != "":
		res.Status = CheckPass
		res.Detail = "Wayland session detected"
	case os.Getenv("DISPLAY") != "":
		res.Status = CheckWarn
		res.Detail = "X11 session — Illogical-mango needs Wayland, log into Mango after installing"
	default:
		res.Status = CheckWarn
		res.Detail = "no graphical session — installation works, log in afterwards"
	}
	return res
}

func currentUserName() string {
	if name := os.Getenv("USER"); name != "" {
		return name
	}
	return fmt.Sprintf("uid %d", os.Geteuid())
}

func humanBytes(n uint64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := uint64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

// shortenHome renders paths under $HOME with a tilde, for compact display.
func shortenHome(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	if path == home {
		return "~"
	}
	if len(path) > len(home) && path[:len(home)] == home && path[len(home)] == filepath.Separator {
		return "~" + path[len(home):]
	}
	return path
}
