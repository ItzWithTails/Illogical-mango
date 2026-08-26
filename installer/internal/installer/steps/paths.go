package steps

import (
	"os"
	"path/filepath"
)

// The installer writes only under the XDG user directories. Resolving them in
// one place keeps every step honest about where it is allowed to put things.

func home() string {
	dir, err := os.UserHomeDir()
	if err != nil {
		return os.Getenv("HOME")
	}
	return dir
}

func configHome() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return dir
	}
	return filepath.Join(home(), ".config")
}

func dataHome() string {
	if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
		return dir
	}
	return filepath.Join(home(), ".local", "share")
}

func stateHome() string {
	if dir := os.Getenv("XDG_STATE_HOME"); dir != "" {
		return dir
	}
	return filepath.Join(home(), ".local", "state")
}

// shellDir is where Quickshell looks for the Illogical-mango configuration.
func shellDir() string {
	return filepath.Join(configHome(), "quickshell", "ilmango")
}
