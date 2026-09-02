package session

import (
	"fmt"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

// makeSkillsetTestSession creates a wizard-actor session (LVL_GRGOD) registered
// in the manager so it can resolve targets, plus a separate target player
// session. Returns (wizard, target).
func makeSkillsetTestSession(t *testing.T) (*Session, *Session) {
	t.Helper()
	m := makeTestManager(t)
	wiz := makeTestSession(t, m, "God", 1001, true)
	wiz.player.Level = LVL_GRGOD

	target := makeTestSession(t, m, "Hero", 1001, true)
	m.mu.Lock()
	m.sessions["god"] = wiz
	m.sessions["hero"] = target
	m.mu.Unlock()
	return wiz, target
}

// readOneText drains and returns the first MsgText the session produced. Many
// cmdSkillset paths emit a single message; this asserts its exact bytes.
func readOneText(t *testing.T, s *Session) string {
	t.Helper()
	return readSessionText(t, s)
}

// headOf returns the first n bytes of s (or all of s if shorter), for readable
// error messages. Safe against short strings.
func headOf(s string, n int) string {
	if len(s) < n {
		return s
	}
	return s[:n]
}

// tailOf returns the last n bytes of s (or all of s if shorter).
func tailOf(s string, n int) string {
	if len(s) < n {
		return s
	}
	return s[len(s)-n:]
}

// --- Step 1: no-argument syntax + skill list ---

func TestCmdSkillset_NoArg_SyntaxAndSkillList(t *testing.T) {
	wiz, _ := makeSkillsetTestSession(t)

	if err := cmdSkillset(wiz, nil); err != nil {
		t.Fatalf("cmdSkillset: %v", err)
	}
	// First message: the exact syntax line (\r\n, not \n\r).
	got := readOneText(t, wiz)
	if want := "Syntax: skillset <name> '<skill>' <value>\r\n"; got != want {
		t.Errorf("syntax line = %q, want %q", got, want)
	}

	// Second message: the skill list. The reference is hand-traced from C
	// (modify.c:266-279): completed lines (i%4==3) flush with \r\n; the partial
	// final line is sent AS-IS (no \r\n), then a single trailing \n\r. The last
	// printed skill is index 206 ("lightning breath", 206%4==2 → partial), so C
	// ends …lightning breath\n\r — never …lightning breath\r\n\n\r.
	list := readOneText(t, wiz)
	var wantB strings.Builder
	wantB.WriteString("Skill being one of the following:\n\r")
	size := spells.SkillCatalogSize()
	for i := 0; i < size; i++ {
		raw := spells.SpellRawName(i)
		if raw == "" || raw[0] == '\n' {
			break
		}
		if raw[0] == '!' {
			continue
		}
		fmt.Fprintf(&wantB, "%18s", raw)
		if i%4 == 3 {
			wantB.WriteString("\r\n")
		}
	}
	wantB.WriteString("\n\r")
	if list != wantB.String() {
		t.Errorf("skill list mismatch:\n--- got ---\n%q\n--- want ---\n%q", list, wantB.String())
	}

	// Sharp guards tracing C exactly — these FAIL on the buggy \r\n-on-partial
	// version and PASS on the fix, independent of the reference builder above.
	// (1) The double terminator "\r\n\n\r" never appears: a partial line has no
	//     trailing \r\n, so only the final \n\r follows the last entry.
	if strings.Contains(list, "\r\n\n\r") {
		t.Errorf("skill list has a spurious \\r\\n before the trailing \\n\\r (partial line must have no \\r\\n): %q", list)
	}
	// (2) The list ends with the last printed skill ("lightning breath", index
	//     206) followed by a single \n\r — no \r\n between them. The last skill
	//     on a partial line (206 % 4 == 2) is sent verbatim, then \n\r.
	if want := "lightning breath\n\r"; !strings.HasSuffix(list, want) {
		t.Errorf("skill list tail = %q, want suffix ending in %q (partial final line, no \\r\\n)", tailOf(list, 60), want)
	}

	// Spot-check the leading content: index 0 (!RESERVED!) is skipped; the list
	// begins with "holy ward","shift reality","bless" then \r\n (i=3 break).
	if !strings.HasPrefix(list, "Skill being one of the following:\n\r") {
		t.Errorf("skill list should start with the header, got prefix %q", headOf(list, 50))
	}
	// The first line holds indices 1,2,3 (holy ward / shift reality / bless),
	// each %18s right-justified, then \r\n.
	firstLine := strings.Split(list, "\r\n")[0]
	if !strings.Contains(firstLine, "holy ward") || !strings.Contains(firstLine, "bless") {
		t.Errorf("first list line should contain holy ward and bless, got %q", firstLine)
	}
}

// --- Step 2: NOPERSON on unknown target (exact C bytes) ---

func TestCmdSkillset_UnknownTarget_NOPERSON(t *testing.T) {
	wiz, _ := makeSkillsetTestSession(t)

	if err := cmdSkillset(wiz, []string{"nobody", "'backstab'", "75"}); err != nil {
		t.Fatalf("cmdSkillset: %v", err)
	}
	if got, want := readOneText(t, wiz), "No-one by that name here.\r\n"; got != want {
		t.Errorf("unknown target = %q, want exact C NOPERSON %q", got, want)
	}
}

// --- Step 3: skill-name-expected (no quoted skill) ---

func TestCmdSkillset_SkillNameExpected(t *testing.T) {
	wiz, _ := makeSkillsetTestSession(t)

	if err := cmdSkillset(wiz, []string{"hero"}); err != nil {
		t.Fatalf("cmdSkillset: %v", err)
	}
	if got, want := readOneText(t, wiz), "Skill name expected.\n\r"; got != want {
		t.Errorf("empty skill = %q, want %q", got, want)
	}
}

// --- Step 4: skill must be quoted (no opening quote) ---

func TestCmdSkillset_NotQuoted(t *testing.T) {
	wiz, _ := makeSkillsetTestSession(t)

	if err := cmdSkillset(wiz, []string{"hero", "backstab", "75"}); err != nil {
		t.Fatalf("cmdSkillset: %v", err)
	}
	if got, want := readOneText(t, wiz), "Skill must be enclosed in: ''\n\r"; got != want {
		t.Errorf("unquoted skill = %q, want %q", got, want)
	}
}

// --- Step 5: unterminated quote ---

func TestCmdSkillset_UnterminatedQuote(t *testing.T) {
	wiz, _ := makeSkillsetTestSession(t)

	if err := cmdSkillset(wiz, []string{"hero", "'backstab", "75"}); err != nil {
		t.Fatalf("cmdSkillset: %v", err)
	}
	if got, want := readOneText(t, wiz), "Skill must be enclosed in: ''\n\r"; got != want {
		t.Errorf("unterminated quote = %q, want %q", got, want)
	}
}

// --- Step 6: unrecognized skill ---

func TestCmdSkillset_UnrecognizedSkill(t *testing.T) {
	wiz, _ := makeSkillsetTestSession(t)

	// Properly-closed quotes around a name that is not a real skill.
	if err := cmdSkillset(wiz, []string{"hero", "'zzznotaskill'", "75"}); err != nil {
		t.Fatalf("cmdSkillset: %v", err)
	}
	if got, want := readOneText(t, wiz), "Unrecognized skill.\n\r"; got != want {
		t.Errorf("unrecognized skill = %q, want %q", got, want)
	}
}

// --- Step 7: value-expected ---

func TestCmdSkillset_ValueExpected(t *testing.T) {
	wiz, _ := makeSkillsetTestSession(t)

	if err := cmdSkillset(wiz, []string{"hero", "'backstab'"}); err != nil {
		t.Fatalf("cmdSkillset: %v", err)
	}
	if got, want := readOneText(t, wiz), "Learned value expected.\n\r"; got != want {
		t.Errorf("empty value = %q, want %q", got, want)
	}
}

// --- Step 7: value < 0 ---

func TestCmdSkillset_ValueBelowMin(t *testing.T) {
	wiz, _ := makeSkillsetTestSession(t)

	if err := cmdSkillset(wiz, []string{"hero", "'backstab'", "-1"}); err != nil {
		t.Fatalf("cmdSkillset: %v", err)
	}
	if got, want := readOneText(t, wiz), "Minimum value for learned is 0.\n\r"; got != want {
		t.Errorf("value < 0 = %q, want %q", got, want)
	}
}

// --- Step 7: value > 100 ---

func TestCmdSkillset_ValueAboveMax(t *testing.T) {
	wiz, _ := makeSkillsetTestSession(t)

	if err := cmdSkillset(wiz, []string{"hero", "'backstab'", "101"}); err != nil {
		t.Fatalf("cmdSkillset: %v", err)
	}
	if got, want := readOneText(t, wiz), "Max value for learned is 100.\n\r"; got != want {
		t.Errorf("value > 100 = %q, want %q", got, want)
	}
}

// --- Step 11: success — sets the skill and emits the byte-exact confirmation ---

func TestCmdSkillset_Success_SetsSkillAndConfirms(t *testing.T) {
	wiz, target := makeSkillsetTestSession(t)

	if err := cmdSkillset(wiz, []string{"hero", "'backstab'", "75"}); err != nil {
		t.Fatalf("cmdSkillset: %v", err)
	}
	got := readOneText(t, wiz)
	if want := "You change Hero's backstab to 75.\n\r"; got != want {
		t.Errorf("confirmation = %q, want %q", got, want)
	}
	// The skill was actually set (keyed by the canonical lowercased name).
	if lvl := target.player.GetSkill("backstab"); lvl != 75 {
		t.Errorf("target backstab = %d, want 75", lvl)
	}
}

// --- Multiword skill name round-trips through the quotes (e.g. 'cure light') ---

func TestCmdSkillset_Success_MultiwordSkill(t *testing.T) {
	wiz, target := makeSkillsetTestSession(t)

	// 'cure light' splits across args after Fields; the handler rejoins them.
	if err := cmdSkillset(wiz, []string{"hero", "'cure", "light'", "80"}); err != nil {
		t.Fatalf("cmdSkillset: %v", err)
	}
	got := readOneText(t, wiz)
	if want := "You change Hero's cure light to 80.\n\r"; got != want {
		t.Errorf("multiword confirmation = %q, want %q", got, want)
	}
	if lvl := target.player.GetSkill("cure light"); lvl != 80 {
		t.Errorf("target cure light = %d, want 80", lvl)
	}
}

func TestCmdSkillset_PickLockUsesDoorSkillKey(t *testing.T) {
	wiz, target := makeSkillsetTestSession(t)

	if err := cmdSkillset(wiz, []string{"hero", "'pick", "lock'", "100"}); err != nil {
		t.Fatalf("cmdSkillset: %v", err)
	}
	if got, want := readOneText(t, wiz), "You change Hero's pick lock to 100.\n\r"; got != want {
		t.Fatalf("confirmation = %q, want %q", got, want)
	}
	if got := target.player.GetSkill(game.SkillPickLock); got != 100 {
		t.Fatalf("pick-lock skill = %d, want 100", got)
	}
}

func TestCmdSkillset_SerpentKickUsesCommandSkillKey(t *testing.T) {
	wiz, target := makeSkillsetTestSession(t)

	if err := cmdSkillset(wiz, []string{"hero", "'serpent", "kick'", "75"}); err != nil {
		t.Fatalf("cmdSkillset: %v", err)
	}
	if got, want := readOneText(t, wiz), "You change Hero's serpent kick to 75.\n\r"; got != want {
		t.Fatalf("confirmation = %q, want %q", got, want)
	}
	if got := target.player.GetSkill(game.SkillSerpentKick); got != 75 {
		t.Fatalf("serpent-kick skill = %d, want 75", got)
	}
}

// --- Level gate: a mortal (< LVL_GRGOD) is rejected ---

func TestCmdSkillset_MortalRejected(t *testing.T) {
	m := makeTestManager(t)
	mortal := makeTestSession(t, m, "Mort", 1001, true)
	mortal.player.Level = 10 // well below LVL_GRGOD (38)

	if err := cmdSkillset(mortal, nil); err != nil {
		t.Fatalf("cmdSkillset: %v", err)
	}
	// The handler's own checkLevel gate fires ("Huh!?!" — matching cmdSet/advance).
	if got := readOneText(t, mortal); !strings.HasPrefix(got, "Huh") {
		t.Errorf("mortal should be gated, got %q", got)
	}
}
