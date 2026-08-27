package pkg

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ilmango/internal/run"
)

// Manager drives one distribution's package tooling.
type Manager struct {
	// Name is the executable, e.g. "pacman".
	Name string
	// Install is the argument prefix for a non-interactive install. Package
	// names are appended.
	InstallArgs []string
	// QueryArgs asks whether a package is installed; the name is appended.
	// A zero exit status means installed.
	QueryArgs []string
	// ListInstalledArgs prints every installed package, one per line, with the
	// name in the first field. Probing hundreds of packages individually is
	// far too slow, so this is the path Install actually uses.
	ListInstalledArgs []string
	// RefreshArgs updates the package databases without upgrading anything.
	// Empty means the manager has no safe way to do that — pacman does not:
	// syncing its databases and then installing against them, without also
	// upgrading, is the partial upgrade that Arch documents as unsupported.
	RefreshArgs []string
	// UpgradeArgs refreshes the databases and upgrades the system in one
	// transaction. Empty means the manager offers no such combined step.
	UpgradeArgs []string
	// Privileged marks managers that must run through sudo. AUR helpers must
	// not: they refuse to run as root.
	Privileged bool
}

// AURHelper names a preferred helper, matching the values of the aur-helper
// choice. The empty string means no preference.
type AURHelper = string

// managers are tried in order per family; the first one present wins. AUR
// helpers precede pacman because several dependencies live only in the AUR.
//
// The AUR helpers carry more than --noconfirm. That flag answers pacman's
// questions but not the helper's own menus, so a rebuilt package can still stop
// the install on a "show diffs?" prompt with nothing to read it — the installer
// gives its children no stdin, precisely so that a prompt can never sit there
// invisibly behind the interface.
var managers = map[string][]Manager{
	"arch": {
		{Name: "paru", InstallArgs: []string{"-S", "--needed", "--noconfirm", "--skipreview"}, QueryArgs: []string{"-Q"}, ListInstalledArgs: []string{"-Qq"}, UpgradeArgs: []string{"-Syu", "--noconfirm"}},
		{Name: "yay", InstallArgs: []string{"-S", "--needed", "--noconfirm", "--answerdiff=None", "--answeredit=None", "--answerclean=None", "--answerupgrade=None"}, QueryArgs: []string{"-Q"}, ListInstalledArgs: []string{"-Qq"}, UpgradeArgs: []string{"-Syu", "--noconfirm", "--answerdiff=None", "--answeredit=None", "--answerclean=None", "--answerupgrade=None"}},
		{Name: "pacman", InstallArgs: []string{"-S", "--needed", "--noconfirm"}, QueryArgs: []string{"-Q"}, ListInstalledArgs: []string{"-Qq"}, UpgradeArgs: []string{"-Syu", "--noconfirm"}, Privileged: true},
	},
	"fedora": {
		{Name: "dnf", InstallArgs: []string{"install", "-y"}, QueryArgs: []string{"list", "--installed"}, ListInstalledArgs: []string{"repoquery", "--installed", "--qf", "%{name}"}, Privileged: true},
	},
	"debian": {
		{Name: "apt-get", InstallArgs: []string{"install", "-y"}, RefreshArgs: []string{"update"}, Privileged: true},
	},
	"ubuntu": {
		{Name: "apt-get", InstallArgs: []string{"install", "-y"}, RefreshArgs: []string{"update"}, Privileged: true},
	},
}

// ErrNoManager is returned when no supported package manager is present.
var ErrNoManager = fmt.Errorf("no supported package manager found")

// Time budgets for install transactions.
//
// They are deliberately far larger than any healthy install needs — a first run
// on Arch compiles several AUR packages and can honestly take half an hour. What
// they bound is the dishonest case: an AUR source whose server accepts the
// connection and then feeds a few kilobytes per retry forever. That never
// triggers the runner's stall watchdog, because the download keeps talking, and
// without a ceiling one broken upstream can hold the whole install open
// indefinitely. Hitting a budget is not fatal: the package is reported as one
// that could not be installed, which is what the step does with any other
// failure.
const (
	// batchOverhead is the batch's fixed allowance: database refresh, download
	// setup and the transaction itself, independent of how much is being
	// installed.
	batchOverhead = 10 * time.Minute
	// perPackage is added to the batch budget for each package in it.
	perPackage = time.Minute
	// batchCeiling caps the batch budget however long the list gets.
	batchCeiling = 90 * time.Minute
	// PackageBudget covers one package during the retry pass. A single AUR
	// package compiled from source can honestly take this long.
	PackageBudget = 20 * time.Minute
	// refreshBudget covers a database sync, which downloads a few megabytes
	// and is the first thing to hang on an unreachable mirror.
	refreshBudget = 10 * time.Minute
	// upgradeBudget covers a full system upgrade, which on a machine left
	// alone for months is the longest single thing the installer does.
	upgradeBudget = 90 * time.Minute
)

// BatchBudget scales the batch's time budget with the amount of work in it, so
// that reinstalling a handful of missing packages fails fast while a first run
// on a bare machine still gets the time it genuinely needs.
func BatchBudget(packages int) time.Duration {
	budget := batchOverhead + time.Duration(packages)*perPackage
	return min(budget, batchCeiling)
}

// FindManager returns the package manager to use on this machine.
func FindManager(family string) (Manager, error) { return FindManagerPreferring(family, "") }

// FindManagerPreferring honours an explicit AUR helper preference.
//
//   - "" or "auto" keeps the default order: paru, then yay, then pacman.
//   - "none" refuses every AUR helper, so only the official repositories are
//     used and AUR-only packages are reported as missing rather than built.
//   - a helper's name requires that helper, and fails clearly if it is absent
//     rather than quietly falling back to something the user did not choose.
func FindManagerPreferring(family string, preferred AURHelper) (Manager, error) {
	candidates, ok := managers[family]
	if !ok {
		// Debian derivatives that report their own family still use apt.
		candidates = managers["debian"]
	}

	switch preferred {
	case "", "auto":
	case "none":
		candidates = withoutAURHelpers(candidates)
	default:
		named := byName(candidates, preferred)
		if len(named) == 0 {
			return Manager{}, fmt.Errorf("%s is not a package manager for %s", preferred, family)
		}
		if !run.Exists(preferred) {
			return Manager{}, fmt.Errorf("%s was chosen as the AUR helper but is not installed — install it, or pick another with --set aur-helper=auto", preferred)
		}
		candidates = named
	}

	for _, m := range candidates {
		if run.Exists(m.Name) {
			return m, nil
		}
	}
	return Manager{}, fmt.Errorf("%w for %s", ErrNoManager, family)
}

// IsAURHelper reports whether a manager builds from the AUR.
func (m Manager) IsAURHelper() bool { return !m.Privileged && m.Name != "pacman" }

func withoutAURHelpers(candidates []Manager) []Manager {
	var out []Manager
	for _, m := range candidates {
		if !m.IsAURHelper() {
			out = append(out, m)
		}
	}
	return out
}

func byName(candidates []Manager, name string) []Manager {
	for _, m := range candidates {
		if m.Name == name {
			return []Manager{m}
		}
	}
	return nil
}

// Refresh brings the package databases up to date before installing.
//
// upgrade decides how, and on Arch the distinction is not cosmetic. Syncing
// pacman's databases without upgrading leaves the system half in the old world
// and half in the new: the next install pulls a fresh library against packages
// that still want the old one, and the transaction fails — or worse, succeeds
// and breaks something already installed. That is why the Arch managers carry
// no RefreshArgs at all. Either the system is upgraded along with the
// databases, or neither is touched and packages are installed from whatever the
// local databases already know.
func (m Manager) Refresh(ctx context.Context, r *run.Runner, upgrade bool) error {
	args := m.RefreshArgs
	if upgrade && len(m.UpgradeArgs) > 0 {
		args = m.UpgradeArgs
	}
	if len(args) == 0 {
		return nil
	}
	// A full upgrade downloads and installs a great deal more than a database
	// sync, so it gets an install-sized budget rather than a refresh-sized one.
	budget := refreshBudget
	if upgrade && len(m.UpgradeArgs) > 0 {
		budget = upgradeBudget
	}
	return r.Run(ctx, run.Command{Name: m.Name, Args: args, Privileged: m.Privileged, Timeout: budget})
}

// CanUpgrade reports whether this manager can upgrade the system.
func (m Manager) CanUpgrade() bool { return len(m.UpgradeArgs) > 0 }

// Install installs every package in names that is not already present.
//
// It tries the whole set in one transaction, which is what a package manager
// is good at. If that fails — usually because one name does not exist on this
// distribution — it retries them individually so that one bad name cannot cost
// the user every other dependency. The names that still fail are returned
// rather than treated as fatal: most are optional tools, and abandoning the
// install over one of them helps nobody.
func (m Manager) Install(ctx context.Context, r *run.Runner, names []string, report func(Progress)) (failed []string, err error) {
	present, err := m.InstalledSet(ctx, r)
	if err != nil {
		// Without the set we simply try everything; the manager skips what is
		// already there anyway.
		present = map[string]bool{}
	}

	var wanted []string
	for _, name := range names {
		if !present[name] {
			wanted = append(wanted, name)
		}
	}
	if len(wanted) == 0 {
		return nil, nil
	}

	watch := newProgressWatcher(wanted, report)
	if err := m.install(ctx, r, wanted, BatchBudget(len(wanted)), watch); err == nil {
		return nil, nil
	}

	// A failed batch is rarely a total loss: package managers install what they
	// can and give up on the one that broke. Re-reading what is present keeps
	// the retry loop from rebuilding everything that already succeeded.
	if after, err := m.InstalledSet(ctx, r); err == nil {
		present = after
	}

	for _, name := range wanted {
		if err := ctx.Err(); err != nil {
			return failed, err
		}
		if present[name] {
			continue
		}
		if err := m.install(ctx, r, []string{name}, PackageBudget, watch); err != nil {
			failed = append(failed, name)
		}
	}

	// Everything failing points at a broken package manager or no network,
	// not at a list of bad names.
	if len(failed) == len(wanted) {
		return failed, fmt.Errorf("could not install any of the %d packages; check your package manager and network", len(wanted))
	}
	return failed, nil
}

// install runs one install transaction under a time budget.
func (m Manager) install(ctx context.Context, r *run.Runner, names []string, budget time.Duration, watch func(string)) error {
	args := append(append([]string{}, m.InstallArgs...), names...)
	return r.Run(ctx, run.Command{
		Name: m.Name, Args: args, Privileged: m.Privileged, Timeout: budget, OnLine: watch,
	})
}

// Progress reports what a transaction is doing.
//
// Counting installed packages alone leaves the longest part of an Arch install
// — fetching and compiling from the AUR — with nothing to show. That phase can
// run for a quarter of an hour before the first package is installed, and a
// step that says nothing for that long is indistinguishable from one that has
// hung. So building and downloading are reported too.
type Progress struct {
	Done, Total int
	Name        string
	// Action is what is happening to Name: the word the package manager used
	// when installing, or "building" or "downloading" while it works up to
	// that.
	Action string
	// Counted is true when Name is one of the packages that were asked for, so
	// Done and Total mean something. Builds and downloads are worth showing
	// but not worth counting: one build can pull in several packages.
	Counted bool
}

// progressVerbs are the words pacman prints when it starts on a package. The
// post-transaction hooks it numbers "(1/2)" are not packages, which is why
// this matches the verb and the name rather than the counter.
var progressVerbs = []string{"installing", "reinstalling", "upgrading", "downgrading"}

// newProgressWatcher returns a line handler that counts packages as the
// manager reports them.
//
// It counts only names the caller asked for. That makes the total honest —
// dependencies pulled in along the way are not in it — and it is what keeps a
// hook or a stray log line from being mistaken for a package.
func newProgressWatcher(wanted []string, report func(Progress)) func(string) {
	if report == nil {
		return nil
	}
	remaining := make(map[string]bool, len(wanted))
	for _, name := range wanted {
		remaining[name] = true
	}

	total := len(wanted)
	done := 0
	return func(line string) {
		line = strings.TrimSpace(line)

		// makepkg announces each package it builds and each source it fetches.
		// Neither is a package from the list — one build can pull in several —
		// but both say the install is alive and what it is working on.
		if rest, found := strings.CutPrefix(line, "==> Making package: "); found {
			if name, _, _ := strings.Cut(rest, " "); name != "" {
				report(Progress{Done: done, Total: total, Name: name, Action: "building"})
			}
			return
		}
		if rest, found := strings.CutPrefix(line, "-> Downloading "); found {
			if name := strings.TrimSuffix(strings.TrimSpace(rest), "..."); name != "" {
				report(Progress{Done: done, Total: total, Name: name, Action: "downloading"})
			}
			return
		}

		for _, verb := range progressVerbs {
			rest, found := strings.CutPrefix(line, verb+" ")
			if !found {
				continue
			}
			name := strings.TrimSuffix(strings.TrimSpace(rest), "...")
			if !remaining[name] {
				return
			}
			delete(remaining, name)
			done++
			report(Progress{Done: done, Total: total, Name: name, Action: verb, Counted: true})
			return
		}
	}
}

// InstalledSet lists everything installed, in one call.
func (m Manager) InstalledSet(ctx context.Context, r *run.Runner) (map[string]bool, error) {
	if len(m.ListInstalledArgs) == 0 {
		return nil, fmt.Errorf("%s cannot list installed packages", m.Name)
	}

	out, err := r.Output(ctx, run.Command{Name: m.Name, Args: m.ListInstalledArgs})
	if err != nil {
		return nil, err
	}

	set := map[string]bool{}
	for _, line := range strings.Split(out, "\n") {
		if name, _, _ := strings.Cut(strings.TrimSpace(line), " "); name != "" {
			set[name] = true
		}
	}
	return set, nil
}

// IsInstalled reports whether a package is present.
func (m Manager) IsInstalled(ctx context.Context, r *run.Runner, name string) bool {
	if len(m.QueryArgs) == 0 {
		return false
	}
	args := append(append([]string{}, m.QueryArgs...), name)
	_, err := r.Output(ctx, run.Command{Name: m.Name, Args: args})
	return err == nil
}
