package pkg

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
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

	diagnosis := newDiagnosis()
	watch := combine(newProgressWatcher(wanted, report), diagnosis.observe)
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

	// Everything failing usually points at something the manager said out
	// loud. Repeating its own diagnosis beats guessing at network trouble,
	// which is what this used to claim whatever the cause.
	if len(failed) == len(wanted) {
		if why := diagnosis.explain(); why != "" {
			return failed, errors.New(why)
		}
		return failed, fmt.Errorf("could not install any of the %d packages; check your package manager and network", len(wanted))
	}
	return failed, nil
}

// Conflict is one package that cannot be installed while another is present.
type Conflict struct{ Wanted, Installed string }

// diagnosis reads a package manager's own complaints as they stream past.
//
// A failed transaction says why it failed, in words meant for a person, and
// then the exit status throws that away. Keeping the sentences means the
// installer can repeat the manager's reason instead of inventing one.
type diagnosis struct {
	conflicts []Conflict
	missing   []string
}

func newDiagnosis() *diagnosis { return &diagnosis{} }

// conflictLine matches pacman's "A and B are in conflict".
var conflictLine = regexp.MustCompile(`([^\s:]+) and ([^\s]+) are in conflict`)

// missingLine matches a name no repository or the AUR can supply.
var missingLine = regexp.MustCompile(`target not found: ([^\s]+)|could not find all required packages:\s*([^\s]+)`)

func (d *diagnosis) observe(line string) {
	line = strings.TrimSpace(line)

	if m := conflictLine.FindStringSubmatch(line); m != nil {
		// pacman prints the version with the name; the name is what a person
		// can act on.
		c := Conflict{Wanted: stripVersion(m[1]), Installed: stripVersion(m[2])}
		for _, seen := range d.conflicts {
			if seen == c {
				return
			}
		}
		d.conflicts = append(d.conflicts, c)
		return
	}

	if m := missingLine.FindStringSubmatch(line); m != nil {
		name := m[1]
		if name == "" {
			name = m[2]
		}
		if name != "" && !slices.Contains(d.missing, name) {
			d.missing = append(d.missing, name)
		}
	}
}

// Conflicts returns what the manager refused to install over.
func (d *diagnosis) Conflicts() []Conflict { return d.conflicts }

// explain turns what was overheard into one sentence, or nothing.
func (d *diagnosis) explain() string {
	if len(d.conflicts) > 0 {
		c := d.conflicts[0]
		return fmt.Sprintf(
			"%s cannot be installed while %s is present — they provide the same files. "+
				"Remove %s yourself if you want the newer one, or leave it out with --without %s",
			c.Wanted, c.Installed, c.Installed, c.Wanted)
	}
	if len(d.missing) > 0 {
		return fmt.Sprintf("no repository or the AUR could supply %s", strings.Join(d.missing, ", "))
	}
	return ""
}

// stripVersion turns "foo-1.2-3" into "foo".
//
// pacman names a package with its version in these messages, and a version is
// not something the reader can type back at it.
func stripVersion(s string) string {
	parts := strings.Split(s, "-")
	for i := len(parts) - 1; i > 0; i-- {
		// A version part starts with a digit; the name never does at this
		// position for the packages this installer deals with.
		if len(parts[i]) > 0 && parts[i][0] >= '0' && parts[i][0] <= '9' {
			continue
		}
		return strings.Join(parts[:i+1], "-")
	}
	return s
}

// combine feeds a line to several observers.
func combine(observers ...func(string)) func(string) {
	return func(line string) {
		for _, observe := range observers {
			if observe != nil {
				observe(line)
			}
		}
	}
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

// ConflictsWith reports installed packages that would block installing names.
//
// A conflict is only discovered at the end of a transaction, which on Arch can
// be an hour of compiling away: the friend's install built eleven font
// packages before pacman refused the last one. Asking beforehand costs one
// query per candidate and turns that into something said at the start.
//
// The question is asked of the candidate rather than of everything installed:
// "what does this replace" is one lookup, "what conflicts with this" would be
// a scan of the whole system.
func (m Manager) ConflictsWith(ctx context.Context, r *run.Runner, names []string) []Conflict {
	if m.Name != "pacman" && !m.IsAURHelper() {
		return nil // only pacman's query syntax is known here
	}

	installed, err := m.InstalledSet(ctx, r)
	if err != nil {
		return nil
	}

	var found []Conflict
	for _, name := range names {
		if installed[name] {
			continue // already there, so nothing to conflict with
		}
		for _, other := range m.declaredConflicts(ctx, r, name) {
			if installed[other] {
				found = append(found, Conflict{Wanted: name, Installed: other})
			}
		}
	}
	return found
}

// declaredConflicts asks what a package says it conflicts with or replaces.
func (m Manager) declaredConflicts(ctx context.Context, r *run.Runner, name string) []string {
	out, err := r.Output(ctx, run.Command{Name: m.Name, Args: []string{"-Si", name}})
	if err != nil {
		return nil // not in the repositories, or the AUR helper cannot say
	}

	var names []string
	for _, line := range strings.Split(out, "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		switch strings.TrimSpace(key) {
		case "Conflicts With", "Replaces":
		default:
			continue
		}
		for _, field := range strings.Fields(value) {
			if field == "None" {
				continue
			}
			// Entries can carry a version constraint: "foo<2.0".
			if cut := strings.IndexAny(field, "<>="); cut > 0 {
				field = field[:cut]
			}
			names = append(names, field)
		}
	}
	return names
}
