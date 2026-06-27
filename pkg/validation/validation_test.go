package validation

import (
	"strings"
	"testing"
)

func TestIsValidPlayerName_ValidNames(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"simple name", "Hero"},
		{"alphanumeric", "Fighter123"},
		{"with underscore", "a_b"},
		{"min length (2 chars)", "ab"},
		{"max length (32 chars)", strings.Repeat("a", 32)},
		{"mixed case", "DarkMage"},
		{"all digits", "12345"},
		{"all underscores", "___"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidPlayerName(tt.input); !got {
				t.Errorf("IsValidPlayerName(%q) = %v, want true", tt.input, got)
			}
		})
	}
}

func TestIsValidPlayerName_TooShort(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"single char", "a"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidPlayerName(tt.input); got {
				t.Errorf("IsValidPlayerName(%q) = %v, want false", tt.input, got)
			}
		})
	}
}

func TestIsValidPlayerName_TooLong(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"33 chars", strings.Repeat("a", 33)},
		{"50 chars", strings.Repeat("b", 50)},
		{"100 chars", strings.Repeat("c", 100)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidPlayerName(tt.input); got {
				t.Errorf("IsValidPlayerName(%q) = %v, want false", tt.input, got)
			}
		})
	}
}

func TestIsValidPlayerName_InvalidChars(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"exclamation", "Hero!"},
		{"space", "player name"},
		{"accented char", "h\u00e9ro"},
		{"unicode snowman", "\u2603"},
		{"html tag", "admin<script>"},
		{"dot", "player.name"},
		{"dash", "player-name"},
		{"parens", "Hero(1)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidPlayerName(tt.input); got {
				t.Errorf("IsValidPlayerName(%q) = %v, want false", tt.input, got)
			}
		})
	}
}

func TestIsValidPlayerName_ReservedNames(t *testing.T) {
	reserved := []string{
		"admin", "system", "root", "server", "null", "undefined",
		"gm", "moderator", "god", "implementor", "imp", "staff",
		"dev", "bot", "agent", "zax0rz",
	}
	for _, r := range reserved {
		t.Run(r, func(t *testing.T) {
			if got := IsValidPlayerName(r); got {
				t.Errorf("IsValidPlayerName(%q) = %v, want false", r, got)
			}
		})
	}
}

func TestIsValidPlayerName_CaseInsensitiveReserved(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"capitalized Admin", "Admin"},
		{"uppercase SYSTEM", "SYSTEM"},
		{"mixed Zax0Rz", "Zax0Rz"},
		{"uppercase ROOT", "ROOT"},
		{"title case Null", "Null"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidPlayerName(tt.input); got {
				t.Errorf("IsValidPlayerName(%q) = %v, want false", tt.input, got)
			}
		})
	}
}

func TestValidateInput_XSSCaseVariants(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"uppercase script tag", "<SCRIPT>alert(1)</SCRIPT>"},
		{"mixed case script tag", "<ScRiPt>alert(1)</ScRiPt>"},
		{"uppercase javascript protocol", "JaVaScRiPt:alert(1)"},
		{"uppercase event handler", "<img ONERROR=alert(1)>"},
		{"mixed case event handler", "<body OnLoad=alert(1)>"},
		{"multiline script tag", "<script>\nalert(1)\n</script>"},
		{"data uri", "data:text/html,<script>alert(1)</script>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if valid, _ := ValidateInput(tt.input); valid {
				t.Errorf("ValidateInput(%q) = true, want false", tt.input)
			}
		})
	}
}

func TestValidateCommand_XSSCaseVariants(t *testing.T) {
	if valid, _ := ValidateCommand("say", []string{"<SCRIPT>alert(1)</SCRIPT>"}); valid {
		t.Error("ValidateCommand should reject mixed-case script argument")
	}
	if valid, _ := ValidateCommand("JaVaScRiPt:alert(1)", nil); valid {
		t.Error("ValidateCommand should reject mixed-case javascript: command")
	}
}

func TestIsValidPlayerName_Boundary(t *testing.T) {
	// Exactly 2 chars (min valid)
	if got := IsValidPlayerName("ab"); !got {
		t.Errorf("IsValidPlayerName(%q) = %v, want true", "ab", got)
	}
	// Exactly 32 chars (max valid)
	maxName := strings.Repeat("x", 32)
	if got := IsValidPlayerName(maxName); !got {
		t.Errorf("IsValidPlayerName(%q) = %v, want true", maxName, got)
	}
	// 33 chars (just over max)
	tooLong := strings.Repeat("y", 33)
	if got := IsValidPlayerName(tooLong); got {
		t.Errorf("IsValidPlayerName(%q) = %v, want false", tooLong, got)
	}
}

func TestSanitizePlayerName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"valid name unchanged", "Hero", "Hero"},
		{"strips invalid chars", "He@ro!", "Hero"},
		{"strips leading invalid chars", "!Hero", "Hero"},
		{"strips trailing invalid chars", "Hero!", "Hero"},
		{"strips all invalid chars - returns empty", "!", ""},
		{"strips all invalid chars - returns empty", "!@#", ""},
		{"DP-570: single char becomes empty", "a!", ""},
		{"DP-570: single valid char returns empty (below min)", "!b", ""},
		{"DP-570: all invalid", "!", ""},
		{"DP-570: valid two chars stay", "ab", "ab"},
		{"truncates at max length", strings.Repeat("a", 33), strings.Repeat("a", 32)},
		{"empty input stays empty", "", ""},
		{"underscores preserved", "a_b", "a_b"},
		{"digits preserved", "123", "123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizePlayerName(tt.input)
			if got != tt.expected {
				t.Errorf("SanitizePlayerName(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSanitizeInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"normal text passes through", "hello world", "hello world"},
		{"HTML entities escaped", "a & b", "a &amp; b"},
		{"tags escaped", "<b>bold</b>", "&lt;b&gt;bold&lt;/b&gt;"},
		{"quotes escaped", `he said "hi"`, "he said &quot;hi&quot;"},
		{"apostrophe escaped", "it's", "it&#39;s"},
		{"control chars removed", "a\x00b", "ab"},
		{"tab preserved", "a\tb", "a\tb"},
		{"newline preserved", "a\nb", "a\nb"},
		{"CR preserved", "a\rb", "a\rb"},
		{"empty input", "", ""},
		{"DP-569: ampersands don't inflate length", strings.Repeat("&", 999), strings.Repeat("&amp;", 999)},
		{"truncation at 1000", strings.Repeat("a", 1001), strings.Repeat("a", 1000)},
		{"mixed entities within limit", "hello & <world>", "hello &amp; &lt;world&gt;"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeInput(tt.input)
			if got != tt.expected {
				t.Errorf("SanitizeInput(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestValidateCommand(t *testing.T) {
	tests := []struct {
		name    string
		command string
		args    []string
		want    bool
	}{
		{"valid command", "look", []string{}, true},
		{"valid command with args", "say", []string{"hello"}, true},
		{"XSS in command", "<script>alert(1)</script>", []string{}, false},
		{"XSS in args", "say", []string{"<script>alert(1)</script>"}, false},
		{"SQL injection in command", "'; DROP TABLE users; --", []string{}, false},
		{"empty command", "", []string{}, true},
		{"multiple valid args", "give", []string{"sword", "fighter"}, true},
		{"multiple args one invalid", "give", []string{"sword", "<script>xss</script>"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := ValidateCommand(tt.command, tt.args)
			if got != tt.want {
				t.Errorf("ValidateCommand(%q, %v) = %v, want %v", tt.command, tt.args, got, tt.want)
			}
		})
	}
}
