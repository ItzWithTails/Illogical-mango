package ui

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// journal writes the full installation transcript to disk. The screen keeps
// only a short tail in memory, so this file is what a user attaches to a bug
// report.
//
// A journal that fails to open is inert rather than fatal: losing the log is
// not a reason to refuse to install.
type journal struct {
	file   *os.File
	writer *bufio.Writer
}

// openJournal creates a timestamped transcript under the state directory.
func openJournal() *journal {
	dir, err := journalDir()
	if err != nil {
		return &journal{}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return &journal{}
	}

	name := fmt.Sprintf("install-%s.log", time.Now().Format("20060102-150405"))
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		return &journal{}
	}

	j := &journal{file: f, writer: bufio.NewWriter(f)}
	j.line(fmt.Sprintf("Illogical-mango installation started %s", time.Now().Format(time.RFC3339)))
	return j
}

func journalDir() (string, error) {
	if state := os.Getenv("XDG_STATE_HOME"); state != "" {
		return filepath.Join(state, "ilmango"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "state", "ilmango"), nil
}

// section writes a heading for a step.
func (j *journal) section(title string) {
	if j.writer == nil {
		return
	}
	fmt.Fprintf(j.writer, "\n=== %s ===\n", title)
}

// line appends one output line.
func (j *journal) line(text string) {
	if j.writer == nil {
		return
	}
	fmt.Fprintln(j.writer, text)
}

// path returns the transcript's location, or an empty string if there is none.
func (j *journal) path() string {
	if j.file == nil {
		return ""
	}
	return j.file.Name()
}

// close flushes and releases the file. It is safe to call on an inert journal.
func (j *journal) close() {
	if j.writer != nil {
		j.writer.Flush()
	}
	if j.file != nil {
		j.file.Close()
	}
}
