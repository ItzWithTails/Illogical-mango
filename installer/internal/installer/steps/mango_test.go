package steps

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShippedMangoConfigsUseOnlyKeysMangoParses(t *testing.T) {
	// The env= lines this project shipped for years were never parsed. Nothing
	// failed; the settings simply did not happen. A directive that silently
	// does nothing is worth a test.
	known := map[string]bool{
		"bind": true, "bindl": true, "bindr": true, "bindp": true, "bindc": true,
		"binds": true, "mousebind": true, "axisbind": true, "switchbind": true,
		"gesturebind": true, "source": true, "source-optional": true,
		"exec": true, "exec-once": true,
		"cursor_theme": true, "cursor_size": true,
	}

	dir := filepath.Join("..", "..", "..", "..", "src", "defaults", "mango")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("templates not readable from here: %v", err)
	}

	for _, entry := range entries {
		body, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatalf("reading %s: %v", entry.Name(), err)
		}
		for n, line := range strings.Split(string(body), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			key, _, found := strings.Cut(line, "=")
			if !found {
				t.Errorf("%s:%d is not a key=value line: %q", entry.Name(), n+1, line)
				continue
			}
			if !known[key] {
				t.Errorf("%s:%d uses %q, which mango does not parse", entry.Name(), n+1, key)
			}
		}
	}
}
