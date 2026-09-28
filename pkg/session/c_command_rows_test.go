package session

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// unportedCCommands are C cmd_info rows with no port yet. Each entry cites
// the issue that tracks it; remove the entry when the command lands.
var unportedCCommands = map[string]string{
	"olc":   "DP-1361",
	"track": "DP-1361",
}

// TestEveryCCommandRowIsRegistered guards R2: since DP-1336 a special runs only
// after the typed word resolves to a registered command, so a C cmd_info row
// the port does not register is "Huh?!?" everywhere, including where C lets a
// special handle it (retreat at the guild guard, src/spec_procs.c:577). Socials
// (do_action, do_insult) resolve through the socials table instead.
func TestEveryCCommandRowIsRegistered(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("..", "..", "src", "interpreter.c"))
	if err != nil {
		t.Fatalf("read C command table: %v", err)
	}
	text := string(src)
	start := strings.Index(text, "cmd_info[] =")
	if start < 0 {
		t.Fatal("cmd_info[] not found in src/interpreter.c")
	}
	end := strings.Index(text[start:], `"\n"`)
	if end < 0 {
		t.Fatal("cmd_info[] terminator not found")
	}
	row := regexp.MustCompile(`\{\s*"([^"]+)"\s*,\s*POS_\w+\s*,\s*(\w+)`)
	rows := row.FindAllStringSubmatch(text[start:start+end], -1)
	if len(rows) < 400 {
		t.Fatalf("parsed only %d cmd_info rows; the table format changed", len(rows))
	}
	for _, m := range rows {
		// The C table spells one row "whod  " with trailing spaces; a typed
		// "whod" still prefix-matches it, and the port registers "whod".
		name, handler := strings.TrimSpace(m[1]), m[2]
		if handler == "do_action" || handler == "do_insult" {
			continue
		}
		if _, ok := unportedCCommands[name]; ok {
			if _, registered := cmdRegistry.Lookup(name); registered {
				t.Errorf("%q is registered now; remove it from unportedCCommands (%s)", name, unportedCCommands[name])
			}
			continue
		}
		if _, ok := cmdRegistry.Lookup(name); !ok {
			t.Errorf("C command row %q (%s) is not registered (R2)", name, handler)
		}
	}
}
