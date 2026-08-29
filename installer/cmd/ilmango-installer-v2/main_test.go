package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	v2 "ilmango/internal/v2"
)

func TestParseUsesExclusiveOperationSubcommand(t *testing.T) {
	var stderr bytes.Buffer
	_, _, err := parse([]string{"install", "uninstall"}, &stderr)
	if err == nil || !strings.Contains(err.Error(), "unexpected arguments") {
		t.Fatalf("parse contradictory operations error = %v", err)
	}
	cfg, _, err := parse([]string{"update", "--no-packages", "--language", "ru"}, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Operation != v2.Update || cfg.Packages || cfg.Language != "ru" {
		t.Fatalf("parsed config = %+v", cfg)
	}
}

func TestHeadlessRootLifecycleHasNoANSIAndNoHostPackages(t *testing.T) {
	repo := commandFixture(t)
	root := t.TempDir()
	args := []string{"install", "--repo", repo, "--home", "/home/tester", "--root", root, "--yes"}
	var stdout, stderr bytes.Buffer
	if code := start(args, &stdout, &stderr); code != exitOK {
		t.Fatalf("start() = %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	if strings.Contains(stdout.String(), "\x1b[") {
		t.Fatal("headless output contains ANSI escapes")
	}
	if strings.Contains(stdout.String(), "[packages]") {
		t.Fatal("--root attempted package work")
	}
	if lines := strings.Count(stdout.String(), "\n"); lines > 45 {
		t.Fatalf("concise output has %d lines", lines)
	}
	launcher := filepath.Join(root, "home", "tester", ".local", "bin", "ilmango")
	if _, err := os.Stat(launcher); err != nil {
		t.Fatal(err)
	}
	mainConfig := filepath.Join(root, "home", "tester", ".config", "mango", "config.conf")
	if _, err := os.Stat(mainConfig); err != nil {
		t.Fatalf("seeded Mango config was not created: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	if code := start([]string{"uninstall", "--home", "/home/tester", "--root", root, "--yes"}, &stdout, &stderr); code != exitOK {
		t.Fatalf("uninstall = %d: %s", code, stderr.String())
	}
	if _, err := os.Stat(launcher); !os.IsNotExist(err) {
		t.Fatalf("launcher survived uninstall: %v", err)
	}
	if _, err := os.Stat(mainConfig); !os.IsNotExist(err) {
		t.Fatalf("unchanged installer-seeded Mango config survived uninstall: %v", err)
	}
}

func TestNonTerminalRequiresExplicitYes(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := start([]string{"install", "--repo", commandFixture(t), "--no-packages"}, &stdout, &stderr)
	if code != exitUsage || !strings.Contains(stderr.String(), "use --yes") {
		t.Fatalf("start() = %d, stderr = %q", code, stderr.String())
	}
}

func commandFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write := func(rel, body string, mode os.FileMode) {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), mode); err != nil {
			t.Fatal(err)
		}
	}
	write("src/shell.qml", "// shell\n", 0o644)
	write("src/qmldir", "module shell\n", 0o644)
	write("src/VERSION", "1.0.0\n", 0o644)
	write("src/scripts/ilmango", "#!/bin/sh\n", 0o755)
	write("src/defaults/mango/config.conf", "exec-once=ilmango run --daemon\n", 0o644)
	if err := os.MkdirAll(filepath.Join(root, "src", "dots"), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}
