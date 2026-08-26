// Package run is the installer's single gateway to executing external
// commands.
//
// Nothing else in the installer may call os/exec directly. Routing every
// command through here buys three properties that matter for something that
// modifies a working machine:
//
//   - Dry run. ModePlan records what would happen and executes nothing, so the
//     whole installer can be exercised without side effects.
//   - Auditability. Every command is logged as it would be typed, before it
//     runs, including whether it escalates privileges.
//   - No hidden destruction. Commands are built from explicit argument slices,
//     never from interpolated shell strings, so there is no shell to
//     reinterpret them.
package run

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Mode decides whether commands actually execute.
type Mode int

const (
	// ModePlan logs commands and executes nothing.
	ModePlan Mode = iota
	// ModeApply executes commands for real.
	ModeApply
)

// ErrNoPrivilegeTool is returned when a privileged command is requested but no
// escalation tool exists on the machine.
var ErrNoPrivilegeTool = errors.New("no privilege escalation tool found (sudo, doas, run0)")

// ErrStalled is returned when a command produced no output for longer than the
// runner's stall limit and was killed.
var ErrStalled = errors.New("command stalled")

// ErrTimedOut is returned when a command outlived its Timeout and was killed.
var ErrTimedOut = errors.New("command timed out")

// DefaultStall is how long a command may stay completely silent before the
// runner assumes it is wedged.
//
// It is deliberately generous: package managers do go quiet while linking a
// large binary or waiting on a slow mirror. What it catches is the failure this
// installer actually hits — an AUR source download against a host that accepts
// the connection and then never sends another byte, which curl will wait on
// forever because makepkg gives it no timeout.
const DefaultStall = 15 * time.Minute

// Command is one external invocation.
type Command struct {
	// Name is the executable, resolved through PATH.
	Name string
	// Args are passed verbatim. They are never word-split or glob-expanded.
	Args []string
	// Dir is the working directory; empty means the process's own.
	Dir string
	// Env are extra KEY=VALUE entries layered over the process environment.
	Env []string
	// Privileged routes the command through sudo, doas or run0.
	Privileged bool
	// Timeout caps the command's total wall time. Zero means no cap.
	//
	// It exists alongside the runner's stall watchdog because the two catch
	// different failures. The watchdog catches a process that has gone silent;
	// this catches one that is still talking but getting nowhere — a download
	// that dribbles a few kilobytes per retry, say, which no amount of silence
	// detection will ever notice.
	Timeout time.Duration
}

// String renders the command the way a user would type it, for logs.
func (c Command) String() string {
	parts := make([]string, 0, len(c.Args)+2)
	if c.Privileged {
		parts = append(parts, "sudo")
	}
	parts = append(parts, c.Name)
	parts = append(parts, c.Args...)
	return strings.Join(parts, " ")
}

// Runner executes commands under a mode.
//
// The zero Runner is in ModePlan: forgetting to configure a Runner cannot
// accidentally modify the machine.
type Runner struct {
	Mode Mode
	// Log receives the command line and every line of its output. It may be
	// nil, in which case output is discarded.
	Log func(string)

	// Stall is how long Run tolerates silence from a command before killing
	// it. Zero means DefaultStall, so a runner cannot be left unprotected by
	// forgetting to set it; a negative value disables the watchdog for callers
	// that genuinely expect an unbounded quiet command.
	Stall time.Duration

	// privilegeOnce caches the escalation tool lookup.
	privilegeOnce sync.Once
	privilegeName string
}

// Run executes cmd, streaming its combined output to the runner's log.
func (r *Runner) Run(ctx context.Context, cmd Command) error {
	name, args, err := r.resolve(cmd)
	if err != nil {
		return err
	}

	if r.Mode == ModePlan {
		r.log("would run: " + cmd.String())
		return nil
	}
	r.log("$ " + cmd.String())

	parent := ctx
	if cmd.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cmd.Timeout)
		defer cancel()
	}

	proc := exec.CommandContext(ctx, name, args...)
	proc.Dir = cmd.Dir
	if len(cmd.Env) > 0 {
		proc.Env = append(os.Environ(), cmd.Env...)
	}
	// The child leads its own process group so that a wedged build can be
	// killed along with everything it spawned. Package managers work through
	// helpers — makepkg, curl, git — and killing only the process we started
	// would leave the one that is actually stuck holding the pipe open.
	proc.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	proc.Cancel = func() error { return r.terminate(proc, cmd.Privileged) }
	// Bound how long Wait may block on output pipes that a leaked grandchild
	// is still holding open after the process itself is gone.
	proc.WaitDelay = 10 * time.Second

	// Both streams go to one pipe: package managers interleave progress on
	// stdout with warnings on stderr, and the transcript should preserve that
	// order rather than reassemble it.
	pipeReader, pipeWriter := io.Pipe()
	proc.Stdout = pipeWriter
	proc.Stderr = pipeWriter

	if err := proc.Start(); err != nil {
		pipeWriter.Close()
		return fmt.Errorf("starting %s: %w", cmd.Name, err)
	}

	// The heartbeat wraps the reader rather than the line scanner because a
	// download meter redraws itself with carriage returns and may not emit a
	// newline for minutes. Counting bytes measures whether the command is
	// alive; counting lines would sometimes measure whether it is verbose.
	activity := newHeartbeat()
	var streamed sync.WaitGroup
	streamed.Add(1)
	go func() {
		defer streamed.Done()
		r.stream(activity.wrap(pipeReader))
	}()

	stopWatch, stalled := r.watch(proc, cmd, activity)

	waitErr := proc.Wait()
	stopWatch()
	// Closing the writer ends the scanner, which must finish before we return
	// so no output arrives after the step is reported as complete.
	pipeWriter.Close()
	streamed.Wait()

	if err := waitErr; err != nil {
		if *stalled {
			return fmt.Errorf("%s produced no output for %s: %w", cmd.Name, r.stallLimit(), ErrStalled)
		}
		// A deadline the caller set is a verdict on the command, not a
		// cancellation of the run, so it must not read like the user pressed
		// Ctrl+C.
		if cmd.Timeout > 0 && errors.Is(ctx.Err(), context.DeadlineExceeded) && !errors.Is(parent.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("%s did not finish within %s: %w", cmd.Name, cmd.Timeout, ErrTimedOut)
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return fmt.Errorf("%s exited with status %d", cmd.Name, exitErr.ExitCode())
		}
		return fmt.Errorf("%s failed: %w", cmd.Name, err)
	}
	return nil
}

// stallLimit resolves the configured watchdog window.
func (r *Runner) stallLimit() time.Duration {
	if r.Stall == 0 {
		return DefaultStall
	}
	return r.Stall
}

// watch kills proc once it has been silent for longer than the stall limit.
//
// It returns a stop function, which must be called once the process has been
// waited for, and a pointer that reports whether the watchdog fired. The
// pointer is only read after stop, so no synchronisation is needed around it.
func (r *Runner) watch(proc *exec.Cmd, cmd Command, activity *heartbeat) (stop func(), stalled *bool) {
	fired := new(bool)
	limit := r.stallLimit()
	if limit <= 0 {
		return func() {}, fired
	}

	done := make(chan struct{})
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		// Polling at a fraction of the limit keeps the check cheap while
		// bounding how long past the limit a stall can go unnoticed.
		tick := time.NewTicker(limit / 4)
		defer tick.Stop()
		warned := false
		for {
			select {
			case <-done:
				return
			case <-tick.C:
				silence := time.Since(activity.last())
				if silence < limit {
					// Say something before the kill, so a long quiet stretch
					// reads as a diagnosis in progress rather than a freeze.
					switch {
					case silence < limit/2:
						warned = false
					case !warned:
						warned = true
						r.log(fmt.Sprintf("! %s has been silent for %s; giving it until %s", cmd.Name, silence.Round(time.Second), limit))
					}
					continue
				}
				*fired = true
				r.log(fmt.Sprintf("! no output from %s for %s — killing it", cmd.Name, limit))
				if err := r.terminate(proc, cmd.Privileged); err != nil {
					r.log("! could not kill the stalled command: " + err.Error())
				}
				return
			}
		}
	}()
	return func() { close(done); <-finished }, fired
}

// terminate kills a command's whole process group.
//
// A privileged command runs as root, so an unprivileged signal bounces off it;
// in that case the kill is re-issued through the same escalation tool that
// started it, whose credentials are still cached.
func (r *Runner) terminate(proc *exec.Cmd, privileged bool) error {
	if proc.Process == nil {
		return nil
	}
	pgid := proc.Process.Pid
	if err := syscall.Kill(-pgid, syscall.SIGKILL); err == nil || !privileged {
		return err
	}

	tool := r.privilegeTool()
	if tool == "" {
		return ErrNoPrivilegeTool
	}
	args := append(noPromptFlag(tool), "kill", "-KILL", fmt.Sprintf("-%d", pgid))
	return exec.Command(tool, args...).Run()
}

// heartbeat records when a command last produced output.
type heartbeat struct {
	mu sync.Mutex
	at time.Time
}

func newHeartbeat() *heartbeat { return &heartbeat{at: time.Now()} }

func (h *heartbeat) beat() {
	h.mu.Lock()
	h.at = time.Now()
	h.mu.Unlock()
}

func (h *heartbeat) last() time.Time {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.at
}

// wrap returns a reader that records a beat for every read that yields data.
func (h *heartbeat) wrap(r io.Reader) io.Reader { return &beatReader{r: r, h: h} }

type beatReader struct {
	r io.Reader
	h *heartbeat
}

func (b *beatReader) Read(p []byte) (int, error) {
	n, err := b.r.Read(p)
	if n > 0 {
		b.h.beat()
	}
	return n, err
}

// Start launches a command and returns without waiting for it.
//
// It is for the one thing Run cannot do: starting something meant to outlive
// the installer. The child is put in its own session, so it survives the
// installer exiting and never inherits the terminal — a shell daemon must not
// be killed by the hangup that ends this process.
func (r *Runner) Start(ctx context.Context, cmd Command) error {
	name, args, err := r.resolve(cmd)
	if err != nil {
		return err
	}

	if r.Mode == ModePlan {
		r.log("would start: " + cmd.String())
		return nil
	}
	r.log("$ " + cmd.String() + " &")

	// Deliberately not CommandContext: cancelling the installer must not kill
	// a shell the user asked to keep running.
	proc := exec.Command(name, args...)
	proc.Dir = cmd.Dir
	if len(cmd.Env) > 0 {
		proc.Env = append(os.Environ(), cmd.Env...)
	}
	proc.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	proc.Stdin, proc.Stdout, proc.Stderr = nil, nil, nil

	if err := proc.Start(); err != nil {
		return fmt.Errorf("starting %s: %w", cmd.Name, err)
	}
	// Nothing waits on the child, so hand it to init rather than leaving a
	// zombie behind when the installer exits.
	go func() { _ = proc.Wait() }()
	return nil
}

// Output runs a command and returns its stdout. It is for probing the system —
// querying installed packages and the like — and so is permitted in ModePlan:
// reading state has no side effects.
func (r *Runner) Output(ctx context.Context, cmd Command) (string, error) {
	name, args, err := r.resolve(cmd)
	if err != nil {
		return "", err
	}

	proc := exec.CommandContext(ctx, name, args...)
	proc.Dir = cmd.Dir
	out, err := proc.Output()
	return strings.TrimSpace(string(out)), err
}

// Exists reports whether an executable is on PATH.
func Exists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// resolve applies privilege escalation and checks the executable exists.
func (r *Runner) resolve(cmd Command) (name string, args []string, err error) {
	if cmd.Name == "" {
		return "", nil, errors.New("run: command has no name")
	}

	if !cmd.Privileged {
		return cmd.Name, cmd.Args, nil
	}

	tool := r.privilegeTool()
	if tool == "" {
		return "", nil, ErrNoPrivilegeTool
	}
	// Never let an escalation tool prompt: while a plan runs, the TUI owns the
	// terminal, so a password prompt would be invisible and the install would
	// hang on input nobody can see. AcquirePrivileges takes the password up
	// front, outside the interface; here we only ever use that credential.
	return tool, append(append([]string{}, noPromptFlag(tool)...), append([]string{cmd.Name}, cmd.Args...)...), nil
}

// noPromptFlag is the flag that makes an escalation tool fail rather than ask.
func noPromptFlag(tool string) []string {
	switch tool {
	case "sudo":
		return []string{"-n"}
	case "doas":
		return []string{"-n"}
	default:
		// run0 has no equivalent; it authenticates through polkit, which
		// presents its own agent rather than reading the terminal.
		return nil
	}
}

// NeedsPrivileges reports whether this runner has an escalation tool at all.
func (r *Runner) NeedsPrivileges() bool { return r.privilegeTool() != "" }

// PrivilegeTool names the escalation tool in use, for messages.
func (r *Runner) PrivilegeTool() string { return r.privilegeTool() }

// HasPrivileges reports whether a privileged command would succeed right now
// without prompting.
func (r *Runner) HasPrivileges(ctx context.Context) bool {
	tool := r.privilegeTool()
	if tool == "" {
		return false
	}
	if tool != "sudo" && tool != "doas" {
		return true // polkit-based tools decide per call
	}
	_, err := r.Output(ctx, Command{Name: tool, Args: []string{"-n", "true"}})
	return err == nil
}

// AcquireCommand returns the command that obtains a credential interactively.
// The caller must run it with the terminal handed back to the user — under
// Bubble Tea that means tea.ExecProcess.
func (r *Runner) AcquireCommand() (name string, args []string, ok bool) {
	switch tool := r.privilegeTool(); tool {
	case "sudo":
		return tool, []string{"-v"}, true
	case "doas":
		return tool, []string{"true"}, true
	default:
		return "", nil, false
	}
}

// KeepAlive refreshes the credential until ctx is cancelled, so a long package
// install cannot outlive the timestamp it was authorised under.
func (r *Runner) KeepAlive(ctx context.Context) {
	tool := r.privilegeTool()
	if tool != "sudo" {
		return
	}

	// sudo's default timestamp lasts five minutes; refresh well inside that.
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, _ = r.Output(ctx, Command{Name: tool, Args: []string{"-n", "-v"}})
		}
	}
}

// privilegeTool finds the escalation tool once per runner.
func (r *Runner) privilegeTool() string {
	r.privilegeOnce.Do(func() {
		for _, tool := range []string{"sudo", "doas", "run0"} {
			if Exists(tool) {
				r.privilegeName = tool
				return
			}
		}
	})
	return r.privilegeName
}

// stream forwards output line by line so the UI updates as work happens.
func (r *Runner) stream(out io.Reader) {
	scanner := bufio.NewScanner(out)
	// Package managers print long dependency lists on one line.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		if line := strings.TrimRight(scanner.Text(), " \t\r"); strings.TrimSpace(line) != "" {
			r.log(line)
		}
	}
}

func (r *Runner) log(line string) {
	if r.Log != nil {
		r.Log(line)
	}
}
