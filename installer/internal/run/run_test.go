package run

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestZeroRunnerPlansRatherThanExecutes(t *testing.T) {
	var logged []string
	r := Runner{Log: func(line string) { logged = append(logged, line) }}

	// A command that would be catastrophic if it ever actually ran.
	err := r.Run(context.Background(), Command{Name: "rm", Args: []string{"-rf", "/"}})
	if err != nil {
		t.Fatalf("plan mode returned an error: %v", err)
	}

	if len(logged) != 1 || !strings.HasPrefix(logged[0], "would run:") {
		t.Fatalf("plan mode logged %v, want a single \"would run\" line", logged)
	}
	if !strings.Contains(logged[0], "rm -rf /") {
		t.Errorf("the plan must show the exact command: %q", logged[0])
	}
}

func TestCommandStringShowsEscalation(t *testing.T) {
	plain := Command{Name: "systemctl", Args: []string{"--user", "daemon-reload"}}
	if got, want := plain.String(), "systemctl --user daemon-reload"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}

	privileged := Command{Name: "usermod", Args: []string{"-aG", "video", "test"}, Privileged: true}
	if got := privileged.String(); !strings.HasPrefix(got, "sudo ") {
		t.Errorf("a privileged command must read as escalated: %q", got)
	}
}

func TestPrivilegedCommandsNeverPrompt(t *testing.T) {
	// A prompt raised while the TUI owns the terminal is invisible and hangs
	// the install, so escalation must always be told not to ask.
	for tool, want := range map[string]string{"sudo": "-n", "doas": "-n"} {
		if got := noPromptFlag(tool); len(got) != 1 || got[0] != want {
			t.Errorf("noPromptFlag(%q) = %v, want [%q]", tool, got, want)
		}
	}
	if got := noPromptFlag("run0"); got != nil {
		t.Errorf("run0 authenticates through polkit and takes no flag, got %v", got)
	}
}

func TestResolveRejectsNamelessCommand(t *testing.T) {
	r := Runner{}
	if _, _, err := r.resolve(Command{}); err == nil {
		t.Error("a command with no name must be rejected")
	}
}

func TestResolveLeavesUnprivilegedCommandsAlone(t *testing.T) {
	r := Runner{}
	name, args, err := r.resolve(Command{Name: "git", Args: []string{"status"}})
	if err != nil {
		t.Fatal(err)
	}
	if name != "git" || len(args) != 1 || args[0] != "status" {
		t.Errorf("resolve rewrote an unprivileged command: %s %v", name, args)
	}
}

func TestApplyModeRunsAndStreamsOutput(t *testing.T) {
	var logged []string
	r := Runner{Mode: ModeApply, Log: func(line string) { logged = append(logged, line) }}

	err := r.Run(context.Background(), Command{Name: "echo", Args: []string{"hello from the step"}})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	var found bool
	for _, line := range logged {
		if strings.Contains(line, "hello from the step") {
			found = true
		}
	}
	if !found {
		t.Errorf("command output was not streamed to the log: %v", logged)
	}
}

func TestApplyModeReportsExitStatus(t *testing.T) {
	r := Runner{Mode: ModeApply}

	err := r.Run(context.Background(), Command{Name: "false"})
	if err == nil {
		t.Fatal("a non-zero exit must be reported as an error")
	}
	if !strings.Contains(err.Error(), "status") {
		t.Errorf("error should name the exit status, got %q", err)
	}
}

func TestRunHonoursCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	r := Runner{Mode: ModeApply}
	if err := r.Run(ctx, Command{Name: "sleep", Args: []string{"5"}}); err == nil {
		t.Error("a cancelled context must abort the command")
	}
}

func TestStallLimitDefaultsWhenUnset(t *testing.T) {
	var r Runner
	if got := r.stallLimit(); got != DefaultStall {
		t.Fatalf("zero Stall = %s, want the default %s", got, DefaultStall)
	}
	r.Stall = time.Minute
	if got := r.stallLimit(); got != time.Minute {
		t.Fatalf("explicit Stall = %s, want 1m", got)
	}
}

func TestRunKillsASilentCommand(t *testing.T) {
	r := Runner{Mode: ModeApply, Stall: 200 * time.Millisecond}

	start := time.Now()
	err := r.Run(context.Background(), Command{Name: "sleep", Args: []string{"60"}})
	if !errors.Is(err, ErrStalled) {
		t.Fatalf("Run() error = %v, want ErrStalled", err)
	}
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Fatalf("watchdog took %s to fire; the command was not killed promptly", elapsed)
	}
}

func TestRunLetsAChattyCommandOutliveTheStallWindow(t *testing.T) {
	// The command stays under the window only because it keeps talking, which
	// is the distinction the watchdog exists to make.
	r := Runner{Mode: ModeApply, Stall: 400 * time.Millisecond}

	script := "for i in 1 2 3 4 5 6; do printf .; sleep 0.15; done; echo"
	if err := r.Run(context.Background(), Command{Name: "sh", Args: []string{"-c", script}}); err != nil {
		t.Fatalf("Run() error = %v, want success", err)
	}
}

func TestStallWatchdogKillsTheWholeProcessGroup(t *testing.T) {
	// A wedged build is usually wedged in a child. Killing only the process we
	// started would leave that child running and holding the output pipe.
	r := Runner{Mode: ModeApply, Stall: 300 * time.Millisecond}

	marker := filepath.Join(t.TempDir(), "child-alive")
	script := "sh -c 'sleep 30; : > " + marker + "' & wait"
	err := r.Run(context.Background(), Command{Name: "sh", Args: []string{"-c", script}})
	if !errors.Is(err, ErrStalled) {
		t.Fatalf("Run() error = %v, want ErrStalled", err)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("the grandchild outlived the kill and wrote its marker")
	}
}

func TestNegativeStallDisablesTheWatchdog(t *testing.T) {
	r := Runner{Mode: ModeApply, Stall: -1}
	if err := r.Run(context.Background(), Command{Name: "sleep", Args: []string{"0.5"}}); err != nil {
		t.Fatalf("Run() error = %v, want success", err)
	}
}

func TestTimeoutKillsACommandThatKeepsTalking(t *testing.T) {
	// The case the stall watchdog cannot see: steady output, no progress.
	r := Runner{Mode: ModeApply, Stall: time.Hour}

	script := "while :; do printf 'downloading...\\n'; sleep 0.05; done"
	err := r.Run(context.Background(), Command{
		Name:    "sh",
		Args:    []string{"-c", script},
		Timeout: 300 * time.Millisecond,
	})
	if !errors.Is(err, ErrTimedOut) {
		t.Fatalf("Run() error = %v, want ErrTimedOut", err)
	}
}

func TestTimeoutLeavesAPromptCommandAlone(t *testing.T) {
	r := Runner{Mode: ModeApply}
	if err := r.Run(context.Background(), Command{Name: "true", Timeout: time.Minute}); err != nil {
		t.Fatalf("Run() error = %v, want success", err)
	}
}

func TestCancellationIsNotReportedAsATimeout(t *testing.T) {
	// A user pressing Ctrl+C and a command overrunning its budget must not
	// produce the same message.
	r := Runner{Mode: ModeApply}
	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(150 * time.Millisecond); cancel() }()

	err := r.Run(ctx, Command{Name: "sleep", Args: []string{"30"}, Timeout: time.Hour})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() error = %v, want context.Canceled", err)
	}
	if errors.Is(err, ErrTimedOut) {
		t.Fatal("cancellation was reported as a timeout")
	}
}

func TestSlowCommandGetsAProxyHintLongBeforeItIsKilled(t *testing.T) {
	// A minute of silence is not a stall, but it is worth saying something
	// about — the hint has to arrive while waiting is still the plan.
	var mu sync.Mutex
	var logged []string
	r := Runner{
		Mode:  ModeApply,
		Hint:  200 * time.Millisecond,
		Stall: 3 * time.Second,
		Log:   func(line string) { mu.Lock(); logged = append(logged, line); mu.Unlock() },
	}

	err := r.Run(context.Background(), Command{Name: "sleep", Args: []string{"30"}})
	if !errors.Is(err, ErrStalled) {
		t.Fatalf("Run() error = %v, want ErrStalled", err)
	}

	mu.Lock()
	defer mu.Unlock()
	hintAt, killAt := -1, -1
	for i, line := range logged {
		if strings.Contains(line, "a proxy usually fixes it") && hintAt < 0 {
			hintAt = i
		}
		if strings.Contains(line, "killing it") && killAt < 0 {
			killAt = i
		}
	}
	if hintAt < 0 {
		t.Fatalf("no proxy hint was logged; log was %v", logged)
	}
	if killAt < 0 || hintAt > killAt {
		t.Fatalf("the hint arrived at %d and the kill at %d; the hint must come first", hintAt, killAt)
	}
}

func TestProxyHintCanBeSilenced(t *testing.T) {
	var mu sync.Mutex
	var logged []string
	r := Runner{
		Mode:  ModeApply,
		Hint:  -1,
		Stall: time.Second,
		Log:   func(line string) { mu.Lock(); logged = append(logged, line); mu.Unlock() },
	}

	_ = r.Run(context.Background(), Command{Name: "sleep", Args: []string{"30"}})

	mu.Lock()
	defer mu.Unlock()
	for _, line := range logged {
		if strings.Contains(line, "a proxy usually fixes it") {
			t.Fatalf("the hint was logged despite being silenced: %q", line)
		}
	}
}
