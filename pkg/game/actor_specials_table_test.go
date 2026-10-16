package game

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// specInterceptGolden is every cSpecialInterceptsOnly entry's command set,
// transcribed by reading the named C function (DP-1448). cite is the SPECIAL
// body's line range in src/; the test re-reads that range and requires every
// command to appear as CMD_IS("<cmd>") inside it, so a typo in a command name
// fails.
var specInterceptGolden = map[string]struct {
	cite string
	cmds []string
}{
	"stableboy":               {cite: "src/spec_procs2.c:315-461", cmds: []string{"buy", "collect", "list", "stable"}},
	"assassin":                {cite: "src/spec_procs2.c:845-924", cmds: []string{"hire", "list"}},
	"bank":                    {cite: "src/spec_procs.c:2345-2399", cmds: []string{"balance", "deposit", "withdraw"}},
	"pet_shops":               {cite: "src/spec_procs.c:1844-1908", cmds: []string{"buy", "list"}},
	"recharger":               {cite: "src/new_cmds2.c:415-486", cmds: []string{"help", "list", "recharge"}},
	"clerk":                   {cite: "src/spec_procs3.c:75-142", cmds: []string{"buy", "list"}},
	"con_seller":              {cite: "src/spec_procs3.c:239-314", cmds: []string{"buy", "list"}},
	"prostitute":              {cite: "src/spec_procs3.c:670-705", cmds: []string{"buy", "list"}},
	"tattoo1":                 {cite: "src/spec_procs2.c:945-1008", cmds: []string{"buy", "list"}},
	"tattoo2":                 {cite: "src/spec_procs2.c:1010-1072", cmds: []string{"buy", "list"}},
	"tattoo3":                 {cite: "src/spec_procs2.c:1075-1137", cmds: []string{"buy", "list"}},
	"tattoo4":                 {cite: "src/spec_procs2.c:1282-1340", cmds: []string{"buy", "list"}},
	"identifier":              {cite: "src/spec_procs2.c:1193-1280", cmds: []string{"give", "list", "value"}},
	"mortician":               {cite: "src/spec_procs3.c:807-856", cmds: []string{"list", "retrieve"}},
	"horn":                    {cite: "src/spec_procs.c:2401-2420", cmds: []string{"use"}},
	"moon_gate":               {cite: "src/gate.c:242-326", cmds: []string{"enter", "look"}},
	"beholder":                {cite: "src/new_cmds2.c:328-398", cmds: []string{"cast", "recite"}},
	"cuchi":                   {cite: "src/spec_procs.c:1034-1071", cmds: []string{"pat"}},
	"dracula":                 {cite: "src/spec_procs.c:1798-1834", cmds: []string{"look"}},
	"medusa":                  {cite: "src/spec_procs2.c:1552-1581", cmds: []string{"examine", "look"}},
	"no_get":                  {cite: "src/new_cmds2.c:552-574", cmds: []string{"get", "palm", "take"}},
	"pray_for_items":          {cite: "src/spec_procs.c:2068-2168", cmds: []string{"pray"}},
	"portal_to_temple":        {cite: "src/spec_procs3.c:515-540", cmds: []string{"'", "say"}},
	"recruiter":               {cite: "src/spec_procs3.c:905-930", cmds: []string{"cast", "hit", "kill", "will"}},
	"suck_in":                 {cite: "src/spec_procs.c:2279-2299", cmds: []string{"look"}},
	"elements_load_cylinders": {cite: "src/spec_procs3.c:1058-1135", cmds: []string{"drop", "get"}},
	"oro_quarters_room":       {cite: "src/spec_procs.c:2302-2318"},
	"oro_study_room":          {cite: "src/spec_procs.c:2321-2337"},
	"werewolf":                {cite: "src/spec_procs3.c:427-454"},
}

// specPulseOnlyGolden is every cSpecialDeclinesEveryCommand entry mapped to the
// C function that proves it: C's first statement there rejects a bare `cmd`
// operand and returns FALSE/0 before touching anything. The test requires both
// halves of that shape in the cited range.
var specPulseOnlyGolden = map[string]string{
	"backstabber":         "src/spec_procs2.c:2003-2017",
	"black_undead_knight": "src/spec_procs.c:1147-1205",
	"brain_eater":         "src/spec_procs3.c:198-223",
	"breed_killer":        "src/spec_procs2.c:1679-1724",
	"butler":              "src/spec_procs3.c:144-196",
	"citizen":             "src/spec_procs.c:986-1031",
	"cityguard":           "src/spec_procs.c:771-820",
	"cleric":              "src/spec_procs.c:1425-1608",
	"conductor":           "src/spec_procs.c:1612-1738",
	"dragon_breath":       "src/spec_procs.c:926-983",
	"elemental_room":      "src/spec_procs.c:2021-2065",
	"eq_thief":            "src/spec_procs2.c:1613-1646",
	"fearface":            "src/spec_procs.c:2171-2201",
	"fido":                "src/spec_procs.c:724-747",
	"field_object":        "src/spec_procs3.c:456-513",
	"fighter":             "src/spec_procs.c:509-535",
	"ira":                 "src/spec_procs2.c:1405-1425",
	"jail":                "src/spec_procs2.c:1470-1493",
	"janitor":             "src/spec_procs.c:750-768",
	"little_boy":          "src/spec_procs2.c:1391-1403",
	"magic_user":          "src/spec_procs.c:409-503",
	"mallory":             "src/spec_procs.c:1374-1422",
	"mickey":              "src/spec_procs.c:1269-1371",
	"mindflayer":          "src/spec_procs2.c:1972-2001",
	"never_die":           "src/spec_procs2.c:2075-2090",
	"normal_checker":      "src/spec_procs2.c:162-187",
	"paladin":             "src/spec_procs.c:537-567",
	"pissedalchemist":     "src/spec_procs2.c:546-590",
	"puff":                "src/spec_procs.c:611-721",
	"red_undead_knight":   "src/spec_procs.c:1208-1263",
	"rescuer":             "src/spec_procs2.c:523-543",
	"roach":               "src/spec_procs3.c:707-805",
	"snake":               "src/spec_procs.c:330-346",
	"take_to_jail":        "src/spec_procs2.c:1427-1468",
	"teleport_victim":     "src/spec_procs3.c:225-237",
	"teleporter":          "src/spec_procs2.c:2019-2031",
	"thief":               "src/spec_procs.c:389-406",
	"tipster":             "src/spec_procs2.c:464-520",
	"troll":               "src/spec_procs3.c:347-370",
	"zen_master":          "src/new_cmds2.c:658-687",
}

// cSpecialBody returns the cited src/ line range as text.
func cSpecialBody(t *testing.T, cite string) string {
	t.Helper()
	file, span, ok := strings.Cut(cite, ":")
	if !ok {
		t.Fatalf("citation %q is not <file>:<start>-<end>", cite)
	}
	startText, endText, ok := strings.Cut(span, "-")
	if !ok {
		t.Fatalf("citation %q has no line range", cite)
	}
	start, startErr := strconv.Atoi(startText)
	end, endErr := strconv.Atoi(endText)
	if startErr != nil || endErr != nil || start <= 0 || end < start {
		t.Fatalf("citation %q has an unusable line range", cite)
	}
	path, err := filepath.Abs(filepath.Join("..", "..", file))
	if err != nil {
		t.Fatalf("resolve %s: %v", file, err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", cite, err)
	}
	lines := strings.Split(string(data), "\n")
	if end > len(lines) {
		t.Fatalf("citation %q runs past the end of %s (%d lines)", cite, file, len(lines))
	}
	return strings.Join(lines[start-1:end], "\n")
}

func sortedMapKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// TestSpecialInterceptTableMatchesRecordedCommandSets pins every
// cSpecialInterceptsOnly entry to its recorded command set: an added, dropped or
// renamed command fails here, and a command that does not literally appear as
// CMD_IS("<cmd>") inside the cited C function fails too (R1/R5g/R5h).
func TestSpecialInterceptTableMatchesRecordedCommandSets(t *testing.T) {
	for _, name := range sortedMapKeys(cSpecialInterceptsOnly) {
		if _, recorded := specInterceptGolden[name]; !recorded {
			t.Errorf("cSpecialInterceptsOnly[%q] has no recorded C command set; add it to specInterceptGolden with its src/ citation", name)
		}
	}
	for _, name := range sortedMapKeys(specInterceptGolden) {
		golden := specInterceptGolden[name]
		entry, listed := cSpecialInterceptsOnly[name]
		if !listed {
			t.Errorf("%q is recorded in specInterceptGolden but missing from cSpecialInterceptsOnly", name)
			continue
		}
		got := sortedMapKeys(entry)
		want := append([]string(nil), golden.cmds...)
		sort.Strings(want)
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("%q command set = %v, want %v (%s)", name, got, want, golden.cite)
		}
		body := cSpecialBody(t, golden.cite)
		for _, cmd := range want {
			if !strings.Contains(body, `CMD_IS("`+cmd+`")`) {
				t.Errorf("%q lists %q but %s has no CMD_IS(%q)", name, cmd, golden.cite, cmd)
			}
		}
	}
}

// TestNpcSpecialDeclineNeedsANonEmptyCommand pins the pulse boundary: an empty
// command is C's commandless call (src/mobact.c's func(ch, ch, 0, "")), not a
// typed command, so it is never declined from these tables.
func TestNpcSpecialDeclineNeedsANonEmptyCommand(t *testing.T) {
	for _, name := range sortedMapKeys(cSpecialInterceptsOnly) {
		if npcSpecialDeclinesCommand(name, "") {
			t.Errorf("npcSpecialDeclinesCommand(%q, \"\") = true, want false (pulse calls stay unsupported)", name)
		}
	}
	for _, tc := range []struct {
		name string
		cmd  string
		want bool
	}{
		{"backstabber", "", false},
		{"backstabber", "look", true},
		{"bank", "emote", true},
		{"bank", "deposit", false},
		{"werewolf", "", false},
		{"werewolf", "emote", true},
		{"roach", "emote", true},
		{"roach", "", false},
		{"conjured", "emote", false},
		{"moon_gate", "look", false},
		{"moon_gate", "emote", true},
	} {
		if got := npcSpecialDeclinesCommand(tc.name, tc.cmd); got != tc.want {
			t.Errorf("npcSpecialDeclinesCommand(%q, %q) = %v, want %v", tc.name, tc.cmd, got, tc.want)
		}
	}
}

// TestObjectLaneConsultsDeclineTables proves the object host honours the same
// tables: a descriptor-less pet beside the KD ATM (obj 8034, SPECIAL(bank)) or
// the Silent Tower horn (obj 14415, SPECIAL(horn)) moves and obeys freely, while
// each object's own commands stay unsupported for an NPC actor.
func TestObjectLaneConsultsDeclineTables(t *testing.T) {
	w, _, mob, _ := newNPCCommandWorld(t)
	ctx := CharacterSpecialContext{Actor: mob}
	host := &ObjectInstance{}

	if got := DispatchCharacterObjectSpecial("bank", w, ctx, host, "north", ""); got != SpecialDeclined {
		t.Fatalf("bank ATM NPC \"north\" = %v, want declined", got)
	}
	if got := DispatchCharacterObjectSpecial("bank", w, ctx, host, "deposit", ""); got != SpecialUnsupported {
		t.Fatalf("bank ATM NPC \"deposit\" = %v, want unsupported", got)
	}
	if got := DispatchCharacterObjectSpecial("horn", w, ctx, host, "emote", ""); got != SpecialDeclined {
		t.Fatalf("horn object NPC \"emote\" = %v, want declined", got)
	}
	if got := DispatchCharacterObjectSpecial("horn", w, ctx, host, "use", ""); got != SpecialUnsupported {
		t.Fatalf("horn object NPC \"use\" = %v, want unsupported", got)
	}
}

// TestGenBoardActorDeclinesDescriptorlessMobiles ports C's `if (!ch->desc)
// return 0;` (src/boards.c:186-187) and keeps C's CMD_WRITE/CMD_LOOK/
// CMD_EXAMINE/CMD_READ/CMD_REMOVE gate for a switched body that has a descriptor.
func TestGenBoardActorDeclinesDescriptorlessMobiles(t *testing.T) {
	w, _, mob, _ := newNPCCommandWorld(t)
	board := &ObjectInstance{}
	ctx := CharacterSpecialContext{Actor: mob}

	for _, cmd := range []string{"north", "emote", "say", "read", "write"} {
		if got := DispatchCharacterObjectSpecial("gen_board", w, ctx, board, cmd, ""); got != SpecialDeclined {
			t.Errorf("descriptor-less board NPC %q = %v, want declined", cmd, got)
		}
	}

	ctx.PrivateOutput = func(string) {}
	if got := DispatchCharacterObjectSpecial("gen_board", w, ctx, board, "read", ""); got != SpecialUnsupported {
		t.Errorf("switched board NPC \"read\" = %v, want unsupported", got)
	}
	if got := DispatchCharacterObjectSpecial("gen_board", w, ctx, board, "emote", ""); got != SpecialDeclined {
		t.Errorf("switched board NPC \"emote\" = %v, want declined", got)
	}
}

// TestPulseOnlyTableHasCCommandGate pins every cSpecialDeclinesEveryCommand
// entry to the C shape that justifies it: an `if` condition carrying a bare
// `cmd` operand whose statement returns FALSE/0, with no write, draw or state
// change before it. A body that gained a side effect ahead of its gate, or lost
// the gate, fails here (R1/R5g/R5h).
func TestPulseOnlyTableHasCCommandGate(t *testing.T) {
	for _, name := range sortedMapKeys(cSpecialDeclinesEveryCommand) {
		cite, recorded := specPulseOnlyGolden[name]
		if !recorded {
			t.Errorf("cSpecialDeclinesEveryCommand[%q] has no recorded C citation; add it to specPulseOnlyGolden", name)
			continue
		}
		body := cSpecialBody(t, cite)
		guard := pulseGateIndex(body)
		if guard < 0 {
			t.Errorf("%q: %s has no `if (... cmd ...)` guard whose statement returns FALSE/0", name, cite)
			continue
		}
		if token := firstSideEffectToken(body[:guard]); token != "" {
			t.Errorf("%q: %s performs %q before its command gate, so an unnamed command is not side-effect free", name, cite, token)
		}
	}
	for _, name := range sortedMapKeys(specPulseOnlyGolden) {
		if !cSpecialDeclinesEveryCommand[name] {
			t.Errorf("%q is recorded in specPulseOnlyGolden but missing from cSpecialDeclinesEveryCommand", name)
		}
	}
}

// sideEffectTokens are the C constructs that make a special's unnamed-command
// path observable. Declarations, buffer setup, skip_spaces, one_argument and
// get_obj_in_list_vis are deliberately absent.
var sideEffectTokens = []string{
	"act(", "act (", "number(", "number (", "send_to_char", "stc(", "hit(",
	"hit (", "extract_char", "extract_obj", "gain_exp", "damage(", "char_to_room",
	"char_from_room", "do_say(", "do_tell(", "do_gen_comm(", "mudlog", "read_mobile",
	"obj_from_room", "obj_to_room", "set_hunting",
}

func firstSideEffectToken(text string) string {
	for _, token := range sideEffectTokens {
		if strings.Contains(text, token) {
			return token
		}
	}
	return ""
}

// pulseGateIndex returns the offset of the first `if` whose condition contains a
// bare `cmd` operand and whose statement returns FALSE or 0, or -1. The scanner
// walks parentheses so a nested call in the condition does not cut it short.
func pulseGateIndex(body string) int {
	for offset := 0; offset < len(body); {
		relative := strings.Index(body[offset:], "if")
		if relative < 0 {
			return -1
		}
		start := offset + relative
		offset = start + 2
		if start > 0 && isIdentByte(body[start-1]) {
			continue
		}
		cursor := start + 2
		for cursor < len(body) && (body[cursor] == ' ' || body[cursor] == '\t') {
			cursor++
		}
		if cursor >= len(body) || body[cursor] != '(' {
			continue
		}
		depth := 0
		end := -1
		for i := cursor; i < len(body) && end < 0; i++ {
			switch body[i] {
			case '(':
				depth++
			case ')':
				depth--
				if depth == 0 {
					end = i
				}
			}
		}
		if end < 0 {
			return -1
		}
		if !regexp.MustCompile(`\bcmd\b`).MatchString(body[cursor+1 : end]) {
			continue
		}
		statement := body[end+1:]
		semicolon := strings.Index(statement, ";")
		if semicolon < 0 {
			continue
		}
		statement = statement[:semicolon]
		if !strings.Contains(statement, "return") {
			continue
		}
		if !strings.Contains(statement, "FALSE") && !strings.Contains(statement, "0") {
			continue
		}
		return start
	}
	return -1
}

func isIdentByte(b byte) bool {
	return b == '_' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9'
}

// TestConjuredActorDeclinesOnlyForCharmedBodies pins the one branch of
// SPECIAL(conjured) (src/spec_procs3.c:859-892) this lane ports: a charmed
// conjured creature declines every command, and the uncharmed branch stays an
// explicit gap instead of a silent decline.
func TestConjuredActorDeclinesOnlyForCharmedBodies(t *testing.T) {
	w, _, actor, _ := newNPCCommandWorld(t)
	conjured := newSpecProcTestMob(t, w, actor.GetRoom(), 10)
	ctx := CharacterSpecialContext{Actor: actor}

	conjured.RemoveAffected(affCharm)
	if got := DispatchCharacterSpecial("conjured", w, ctx, conjured, "emote", ""); got != SpecialUnsupported {
		t.Errorf("uncharmed conjured NPC \"emote\" = %v, want unsupported (the leave-the-world branch is unported)", got)
	}
	conjured.SetAffected(affCharm)
	if got := DispatchCharacterSpecial("conjured", w, ctx, conjured, "emote", ""); got != SpecialDeclined {
		t.Errorf("charmed conjured NPC \"emote\" = %v, want declined", got)
	}
	if got := DispatchCharacterSpecial("conjured", w, ctx, conjured, "north", ""); got != SpecialDeclined {
		t.Errorf("charmed conjured NPC \"north\" = %v, want declined", got)
	}
}
