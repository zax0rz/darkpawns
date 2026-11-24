package session

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/spells"
	"github.com/zax0rz/darkpawns/pkg/testutil"
)

func readSessionText(t *testing.T, s *Session) string {
	t.Helper()
	select {
	case msg := <-s.send:
		var sm ServerMessage
		if err := json.Unmarshal(msg, &sm); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		switch sm.Type {
		case MsgText:
			var td TextData
			data, _ := json.Marshal(sm.Data)
			_ = json.Unmarshal(data, &td)
			return td.Text
		case MsgEvent:
			var ed EventData
			data, _ := json.Marshal(sm.Data)
			_ = json.Unmarshal(data, &ed)
			// "raw" carries the same text bytes, untouched (literal sends).
			if ed.Type == "text" || ed.Type == "raw" {
				return ed.Text
			}
		}
		t.Fatalf("unexpected message type %q", sm.Type)
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for session message")
	}
	panic("unreachable")
}

func TestAlignmentText(t *testing.T) {
	tests := []struct {
		val  int
		want string
	}{
		{1000, "Epitome of Righteousness"},
		{900, "angels jealous"},
		{600, "path of right"},
		{0, "boring"},
		{-600, "evil it hurts"},
		{-1000, "Epitome of Evil"},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := alignmentText(tt.val)
			if !strings.Contains(got, tt.want) {
				t.Errorf("alignmentText(%d) = %q, expected substring %q", tt.val, got, tt.want)
			}
		})
	}
}

// TestAcTextEveryBand pins every AC band against src/act.informative.c:1232-1257
// (DP-1551). Only an exact 100 is "naked" (gear and APPLY_AC affects can push a
// character above 100, and C still calls that "lightly clothed"); the -45 band
// has no final period; the very-well armored arm at :1244 is dead (ac > -20
// implies ac > -40) and is not probed by value.
func TestAcTextEveryBand(t *testing.T) {
	for _, tt := range []struct {
		ac   int
		want string
	}{
		{ac: 110, want: "You are lightly clothed."},
		{ac: 101, want: "You are lightly clothed."},
		{ac: 100, want: "You are naked, have you no shame?"},
		{ac: 99, want: "You are lightly clothed."},
		{ac: 71, want: "You are lightly clothed."},
		{ac: 70, want: "You are pretty well clothed."},
		{ac: 41, want: "You are pretty well clothed."},
		{ac: 40, want: "You are lightly armored."},
		{ac: 11, want: "You are lightly armored."},
		{ac: 10, want: "You are well armored."},
		{ac: -9, want: "You are well armored."},
		{ac: -10, want: "You are getting pretty sweaty with all that armor on."},
		{ac: -39, want: "You are getting pretty sweaty with all that armor on."},
		{ac: -40, want: "You are extremely well armored"},
		{ac: -45, want: "You are extremely well armored"},
		{ac: -49, want: "You are extremely well armored"},
		{ac: -50, want: "You are decked out in full battle armor."},
		{ac: -74, want: "You are decked out in full battle armor."},
		{ac: -75, want: "You are armored like a wyvern!"},
		{ac: -124, want: "You are armored like a wyvern!"},
		{ac: -125, want: "You are armored like a dragon!"},
		{ac: -149, want: "You are armored like a dragon!"},
		{ac: -150, want: "You could walk through the gates of Hell in all that armor!"},
		{ac: -174, want: "You could walk through the gates of Hell in all that armor!"},
		{ac: -175, want: "You are armored like a god!"},
		{ac: -200, want: "You are armored like a god!"},
	} {
		if got := acText(tt.ac); got != tt.want {
			t.Errorf("acText(%d) = %q, want %q", tt.ac, got, tt.want)
		}
		if tt.ac == -45 && acText(tt.ac) != "You are extremely well armored" {
			t.Errorf("AC -45 must have no final period, got %q", acText(tt.ac))
		}
	}
}

// TestPositionTextEveryPosition pins every do_score position arm against
// src/act.informative.c:1324-1355 (DP-1553). Floating is the default arm.
func TestPositionTextEveryPosition(t *testing.T) {
	for _, tt := range []struct {
		pos  int
		want string
	}{
		{pos: 0, want: "You are DEAD!(Tell a god)"},
		{pos: 1, want: "You are mortally wounded!  You should seek help!"},
		{pos: 2, want: "You are incapacitated, slowly fading away..."},
		{pos: 3, want: "You are stunned!  You can't move!"},
		{pos: 4, want: "You are sleeping."},
		{pos: 5, want: "You are resting."},
		{pos: 6, want: "You are sitting."},
		{pos: 8, want: "You are standing."},
		{pos: 9, want: "You are floating."},
		{pos: 14, want: "You are floating."},
		{pos: -1, want: "You are floating."},
	} {
		if got := positionText(tt.pos); got != tt.want {
			t.Errorf("positionText(%d) = %q, want %q", tt.pos, got, tt.want)
		}
	}
	// POS_FIGHTING without a target is handled by scorePositionText, not
	// positionText; the bare text stays "You are fighting." for the
	// fallback when a target name is unavailable.
	if got := positionText(7); got != "You are fighting." {
		t.Errorf("positionText(7) = %q, want %q", got, "You are fighting.")
	}
}

func TestArticleFor(t *testing.T) {
	tests := []struct {
		val  string
		want string
	}{
		{"Elf", "an"},
		{"Dwarf", "a"},
		{"Kender", "a"},
		{"Human", "a"},
		{"", "a"},
	}
	for _, tt := range tests {
		t.Run(tt.val, func(t *testing.T) {
			got := articleFor(tt.val)
			if got != tt.want {
				t.Errorf("articleFor(%q) = %q, want %q", tt.val, got, tt.want)
			}
		})
	}
}

func TestCmdLevels(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	s.wantsStructuredData = true

	err := cmdLevels(s)
	if err != nil {
		t.Fatalf("cmdLevels failed: %v", err)
	}

	got := readSessionText(t, s)
	if !strings.Contains(got, "[ 1]") {
		t.Errorf("expected level info output, got %q", got)
	}
	const cLVLImmort = 31 // src/structs.h:620; independent C expectation
	if rows := strings.Count(got, "\r\n"); rows != cLVLImmort-1 {
		t.Errorf("levels output has %d rows, want %d", rows, cLVLImmort-1)
	}
	if !strings.Contains(got, "[30]") || strings.Contains(got, "[31]") {
		t.Errorf("levels output crossed the C immortal boundary: %q", got)
	}
}

func TestCmdAbils(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	s.player.Stats.Str = 10
	s.player.Stats.Dex = 10
	s.player.Stats.Int = 10
	s.player.Stats.Wis = 10
	s.player.Stats.Con = 10
	s.player.Stats.Cha = 10
	s.player.CopyBaseAttributes()
	s.player.AddAffect(engine.NewAffectDirect(0, game.ApplyStr, 6, 2, 0, "strength"))
	s.player.AddAffect(engine.NewAffectDirect(0, game.ApplyDex, 6, -1, 0, "clumsy"))
	s.player.AddAffect(engine.NewAffectDirect(0, game.ApplyInt, 6, 1, 0, "intellect"))
	s.player.AddAffect(engine.NewAffectDirect(0, game.ApplyWis, 6, 1, 0, "wisdom"))
	s.player.AddAffect(engine.NewAffectDirect(0, game.ApplyCon, 6, 3, 0, "constitution"))
	s.player.AddAffect(engine.NewAffectDirect(0, game.ApplyCha, 6, -2, 0, "charisma"))

	err := cmdAbils(s)
	if err != nil {
		t.Fatalf("cmdAbils failed: %v", err)
	}

	var got strings.Builder
	for range 7 {
		got.WriteString(readSessionText(t, s))
	}
	want := "Your current ability scores: \r\n" +
		"Strength:      (decent)\r\n" +
		"Dexterity:     (average)\r\n" +
		"Intelligence:  (decent)\r\n" +
		"Wisdom:        (decent)\r\n" +
		"Constitution:  (good)\r\n" +
		"Charisma:      (below average)\r\n"
	if got.String() != want {
		t.Errorf("unexpected abils output:\n got %q\nwant %q", got.String(), want)
	}
}

func TestCmdAbilsClampsCEffectiveStatCeilings(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	s.player.Stats = game.CharStats{Str: 18, Dex: 18, Int: 18, Wis: 18, Con: 18, Cha: 18}
	s.player.CopyBaseAttributes()
	s.player.AddAffect(engine.NewAffectDirect(0, game.ApplyStr, 6, 2, 0, "strength"))
	s.player.AddAffect(engine.NewAffectDirect(0, game.ApplyDex, 6, 2, 0, "dexterity"))
	s.player.AddAffect(engine.NewAffectDirect(0, game.ApplyInt, 6, 2, 0, "intelligence"))
	s.player.AddAffect(engine.NewAffectDirect(0, game.ApplyWis, 6, 2, 0, "wisdom"))
	s.player.AddAffect(engine.NewAffectDirect(0, game.ApplyCon, 6, 2, 0, "constitution"))
	s.player.AddAffect(engine.NewAffectDirect(0, game.ApplyCha, 6, 2, 0, "charisma"))

	if err := cmdAbils(s); err != nil {
		t.Fatalf("cmdAbils failed: %v", err)
	}

	var got strings.Builder
	for range 7 {
		got.WriteString(readSessionText(t, s))
	}
	want := "Your current ability scores: \r\n" +
		"Strength:      (excellent)\r\n" +
		"Dexterity:     (excellent)\r\n" +
		"Intelligence:  (excellent)\r\n" +
		"Wisdom:        (excellent)\r\n" +
		"Constitution:  (excellent)\r\n" +
		"Charisma:      (godlike)\r\n"
	if got.String() != want {
		t.Errorf("unexpected capped abils output:\n got %q\nwant %q", got.String(), want)
	}
}

func TestCmdCoins(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	s.player.Gold = 100
	s.player.BankGold = 500

	err := cmdCoins(s)
	if err != nil {
		t.Fatalf("cmdCoins failed: %v", err)
	}

	got := readSessionText(t, s)
	if !strings.Contains(got, "carrying 100 coins") {
		t.Errorf("expected coins output, got %q", got)
	}
}

func TestCmdScore(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	s.player.Level = 5
	s.player.Exp = 1000

	err := cmdScore(s)
	if err != nil {
		t.Fatalf("cmdScore failed: %v", err)
	}

	got := readSessionText(t, s)
	if !strings.Contains(got, "Alice") || !strings.Contains(got, "Hit points") {
		t.Errorf("expected score output, got %q", got)
	}
	// DP-913: armor line must not double the subject ("You You are well armored.").
	if strings.Contains(got, "You You") {
		t.Errorf("armor line doubles the subject, got %q", got)
	}
	if !strings.Contains(got, "You are ") {
		t.Errorf("expected single-subject armor line, got %q", got)
	}
}

func TestScoreManaLabelMatchesCClassBranch(t *testing.T) {
	for class := game.ClassMageUser; class <= game.ClassMystic; class++ {
		want := "Mana"
		if class == game.ClassPsionic || class == game.ClassMystic {
			want = "Mind/Psi"
		}
		if got := scoreManaLabel(class); got != want {
			t.Errorf("class %d mana label = %q, want %q", class, got, want)
		}
	}
}

func TestCmdScoreFixedFixtureGolden(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Scoretest", 1001, true)
	p := s.player
	p.Class = game.ClassWarrior
	p.Race = game.RaceHuman
	p.Stats.Str = 10
	p.Stats.Dex = 10
	p.CopyBaseAttributes()
	p.Title = "the Warrior"
	p.Hometown = 1
	p.Level = 1
	p.Exp = 1
	p.Health = 22
	p.MaxHealth = 22
	p.Mana = 100
	p.MaxMana = 100
	p.Move = 85
	p.MaxMove = 85
	p.Alignment = 0
	p.AC = 100
	p.Gold = 0
	p.BankGold = 0
	p.Kills = 0
	p.PKs = 0
	p.Deaths = 0
	p.Position = game.PosStanding
	p.Hunger = 36
	p.Thirst = 36
	p.Drunk = 0
	p.Birth = time.Now().Unix()
	p.ConnectedAt = time.Now()
	p.PlayedDuration = 0

	if err := cmdScore(s); err != nil {
		t.Fatal(err)
	}
	const want = "Scoretest                           Age: 17 years (It's your birthday today.)\r\n" +
		"Hit points: 22(22)  Mana points: 100(100)  Movement points: 85(85)\r\n" +
		"You are neutral, how boring.\r\n" +
		"You are naked, have you no shame?\r\n" +
		"Experience:    1 points\r\n" +
		"Coins carried: 0 gold coins    Coins in bank: 0 gold coins\r\n" +
		"Kills: 0  Pks: 0  Deaths: 0\r\n" +
		"You need 1499 exp to reach your next level.\r\n" +
		"You have been playing for 0 days and 0 hours.\r\n" +
		"You are a citizen of Kir Drax'in.\r\n" +
		"This ranks you as Scoretest the Warrior (level 1).\r\n" +
		"You are a Human Warrior.\n\r" +
		"Your pack is empty.\r\n" +
		"You are standing.\r\n"
	if got := readSessionText(t, s); got != want {
		t.Fatalf("score output mismatch\n--- got ---\n%q\n--- want ---\n%q", got, want)
	}
}

func TestCmdScorePreservesNegativeExperienceToNextLevel(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Scoreoverlevel", 1001, true)
	s.player.Class = game.ClassMageUser
	s.player.Level = 1
	s.player.Exp = 7_000_000

	if err := cmdScore(s); err != nil {
		t.Fatal(err)
	}
	got := readSessionText(t, s)
	if !strings.Contains(got, "You need -6998500 exp to reach your next level.\r\n") {
		t.Fatalf("score omitted C's negative XP remainder: %q", got)
	}
}

func TestCmdScoreUsesPlayerRaceAndClassNamesAndCLineEnding(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Racecheck", 1001, true)
	s.player.Race = game.RaceElf
	s.player.Class = game.ClassMageUser

	if err := cmdScore(s); err != nil {
		t.Fatal(err)
	}
	got := readSessionText(t, s)
	if !strings.Contains(got, "You are an Elven Magic User.\n\r") {
		t.Fatalf("score did not use C player race/class names and LF-CR ending: %q", got)
	}
	if strings.Contains(got, "You are an Elf Mage.") {
		t.Fatalf("score used generic race/class display names: %q", got)
	}
}

func TestCmdScoreStateVariants(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Scorestate", 1001, true)
	p := s.player
	p.Level = 40
	p.SetPLRFlag(game.PlrChosen)
	p.SetPlrFlag(game.PrfSummonable, true)
	p.SetAffect(game.AffBlind, true)
	p.SetAffect(game.AffWerewolf, true)
	p.SetAffect(game.AffVampire, true)
	p.SetAffect(game.AffMount, true)
	p.SetAffect(game.AffFleshAlter, true)

	if err := cmdScore(s); err != nil {
		t.Fatal(err)
	}
	got := readSessionText(t, s)
	for _, want := range []string{
		"You are a chosen of the gods.(BadMuthaFucker)\r\n",
		"You are summonable by other players.\r\n",
		"You have been blinded!\r\n",
		"You're a lycanthrope!\r\n",
		"You're a vampire!\r\n",
		"You're mounted.\r\n",
		"Your hand is a gleaming scythe!\r\n",
		"\r\nEquipment spells affecting you:\r\nblind\r\nflesh alter\r\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("score output missing %q:\n%q", want, got)
		}
	}
}

func TestCmdScoreFightingTargetUsesCPersAndThinAirFallback(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Scorefighter", 1001, true)
	s.player.Level = 1
	s.player.SetPosition(game.PosFighting)
	target := game.NewPlayer(2002, "Scoretarget", 1001)
	s.player.SetFightingBody(target)

	if err := cmdScore(s); err != nil {
		t.Fatal(err)
	}
	if got := readSessionText(t, s); !strings.Contains(got, "You are fighting Scoretarget.\r\n") {
		t.Fatalf("score omitted visible fighting target: %q", got)
	}

	target.SetInvisLevel(game.LVL_IMMORT)
	if err := cmdScore(s); err != nil {
		t.Fatal(err)
	}
	if got := readSessionText(t, s); !strings.Contains(got, "You are fighting someone.\r\n") || strings.Contains(got, "You are fighting Scoretarget.") {
		t.Fatalf("score did not apply C PERS visibility to fighting target: %q", got)
	}

	s.player.SetFightingBody(nil)
	if err := cmdScore(s); err != nil {
		t.Fatal(err)
	}
	if got := readSessionText(t, s); !strings.Contains(got, "You are fighting thin air.\r\n") {
		t.Fatalf("score omitted C thin-air fallback: %q", got)
	}
}

func TestCmdScoreSpellAffectsUseCNewestFirstTwoColumnLayout(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Scoreaffect", 1001, true)
	s.player.AddAffect(engine.NewAffectDirect(spells.SpellStrength, game.ApplyStr, 6, 2, engine.AFFNothing, "strength"))
	s.player.AddAffect(engine.NewAffectDirect(spells.SpellCurse, game.ApplyHitroll, 6, -3, engine.AFFCurse, "curse"))
	s.player.AddAffect(engine.NewAffectDirect(spells.SpellCurse, game.ApplyDamroll, 6, -3, engine.AFFCurse, "curse"))
	s.player.AddAffect(engine.NewAffectDirect(spells.SpellPoison, game.ApplyStr, 6, -2, engine.AFFPoison, "poison"))
	s.player.AddAffect(engine.NewAffectDirect(spells.SpellPoison, game.ApplyHitroll, 6, -2, engine.AFFPoison, "poison"))

	if err := cmdScore(s); err != nil {
		t.Fatal(err)
	}
	msg := <-s.send
	var sm ServerMessage
	if err := json.Unmarshal(msg, &sm); err != nil {
		t.Fatalf("unmarshal score event: %v", err)
	}
	var event EventData
	data, err := json.Marshal(sm.Data)
	if err != nil {
		t.Fatalf("marshal score event data: %v", err)
	}
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("unmarshal score event data: %v", err)
	}
	if event.Type != "raw" {
		t.Fatalf("score event type = %q, want raw so prompt framing supplies the single trailing CRLF", event.Type)
	}
	frame, ok := RenderTerminalFrame(msg)
	if !ok {
		t.Fatal("score event was not rendered as a terminal frame")
	}
	got := frame.Text
	spellCell := func(name string) string {
		return "     " + fmt.Sprintf("%-24.24s", name)
	}
	want := "Spells affecting you:\r\n" + spellCell("poison") + spellCell("poison") + "\r\n" + spellCell("curse") + spellCell("curse") + "\r\n" + spellCell("strength")
	if !strings.HasSuffix(got, want) {
		t.Fatalf("C spell affect list suffix = %q, want %q (full output %q)", strings.TrimSuffix(got, want), want, got)
	}
	if strings.HasSuffix(got, "\r") || strings.HasSuffix(got, "\n") {
		t.Fatalf("score buffer has a terminal line ending; C's prompt flush owns it: %q", got[len(got)-32:])
	}
}

func TestCmdScoreActiveAffectFilterUsesCBitPositions(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Scoreaffectbits", 1001, true)
	s.player.AddAffect(engine.NewAffectDirect(spells.SpellProtFromEvil, game.ApplyNone, 24, 0, engine.AFFProtectionEvil, "protection from evil"))
	s.player.AddAffect(engine.NewAffectDirect(138, game.ApplyNone, 6, 0, engine.AFFSneak, "sneak"))

	if err := cmdScore(s); err != nil {
		t.Fatal(err)
	}
	got := readSessionText(t, s)
	if !strings.Contains(got, "     protection from evil") {
		t.Fatalf("score omitted active protection-from-evil affect whose engine bit aliases C AFF_SNEAK: %q", got)
	}
	if strings.Contains(got, "     sneak") {
		t.Fatalf("score included active sneak affect that C excludes from the spell list: %q", got)
	}
}

func TestFleshAlterWeaponMatchesCLevelBands(t *testing.T) {
	for _, tc := range []struct {
		level int
		want  string
	}{
		{1, "studded wooden club"},
		{3, "studded wooden club"},
		{4, "razor-sharp dagger"},
		{18, "steel-shafted battle axe"},
		{19, "double-headed battle axe"},
		{27, "gleaming broad sword"},
		{28, "gleaming long sword"},
		{29, "gleaming long sword"},
		{30, "gleaming scythe"},
		{40, "gleaming scythe"},
	} {
		if got := game.FleshAlterWeapon(tc.level); got != tc.want {
			t.Errorf("game.FleshAlterWeapon(%d) = %q, want %q", tc.level, got, tc.want)
		}
	}
}

func TestCmdSaySelfEcho(t *testing.T) {
	// DP-913: self-echo must conjugate in the second person ("You say"),
	// not third ("You says"). The room echo keeps third person.
	cases := []struct {
		name, text, wantSelf string
	}{
		{"plain", "hello", "You say 'hello'"},
		{"exclaim", "hi!", "You exclaim 'hi!'"},
		{"question", "what?", "You ask 'what?'"},
		{"statement", "ok.", "You state 'ok.'"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := makeTestManager(t)
			s := makeTestSession(t, m, "Alice", 1001, true)
			s.player.Stats.Int = 10
			s.player.Stats.Wis = 10
			s.player.CopyBaseAttributes()
			registerInWorld(t, s)

			if err := cmdSay(s, []string{tc.text}); err != nil {
				t.Fatalf("cmdSay failed: %v", err)
			}
			got := readSessionText(t, s)
			if !strings.Contains(got, tc.wantSelf) {
				t.Errorf("self-echo = %q, want substring %q", got, tc.wantSelf)
			}
			if strings.Contains(got, "You says") {
				t.Errorf("self-echo still uses third-person verb: %q", got)
			}
		})
	}
}

func TestCmdHelp(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)

	// With an empty help table, a topic lookup reports the C "No help
	// available" line. The invented overview/registry fallback is gone (R4).
	m.world.HelpTable = nil
	m.world.HelpScreen = ""
	err := cmdHelp(s, []string{"commands"})
	if err != nil {
		t.Fatalf("cmdHelp failed: %v", err)
	}
	got := readSessionText(t, s)
	if !strings.Contains(got, "No help available") {
		t.Errorf("empty-table help = %q; want \"No help available\" (invented overview removed)", got)
	}
}

func TestCmdWhere(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	m.mu.Lock()
	m.sessions["alice"] = s
	m.mu.Unlock()

	err := cmdWhere(s, nil)
	if err != nil {
		t.Fatalf("cmdWhere failed: %v", err)
	}

	got := readSessionText(t, s)
	if !strings.Contains(got, "Alice") || !strings.Contains(got, "Room A") {
		t.Errorf("expected where output, got %q", got)
	}
}

func TestCmdWho(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	m.mu.Lock()
	m.sessions["alice"] = s
	m.mu.Unlock()

	err := cmdWho(s, nil)
	if err != nil {
		t.Fatalf("cmdWho failed: %v", err)
	}

	got := readSessionText(t, s)
	if !strings.Contains(got, "Alice") {
		t.Errorf("expected who output, got %q", got)
	}
}

func TestCmdUsersCFormat(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "InformativeResidual", 1001, true)
	s.player.SetLevel(game.LVL_IMPL)
	s.player.Class = game.ClassWarrior
	s.connectionNumber = 1
	s.connectedAt = time.Date(2026, time.September, 4, 22, 58, 48, 0, time.UTC)
	s.request = nil
	s.SetRemoteIP("127.0.0.1")
	m.mu.Lock()
	m.sessions[s.player.Name] = s
	m.mu.Unlock()

	if err := cmdUsers(s, nil); err != nil {
		t.Fatalf("cmdUsers: %v", err)
	}
	want := "Num Class   Name         State          Idl Login@   Site\r\n" +
		"--- ------- ------------ -------------- --- -------- ------------------------\r\n" +
		"  1 [40 Wa] InformativeResidual Playing          0 22:58:48 [127.000.000.001]\r\n" +
		"\r\n1 visible sockets connected.\r\n"
	if got := readSessionText(t, s); got != want {
		t.Fatalf("users output = %q, want %q", got, want)
	}
}

func TestCmdUsersPreLoginAndStateLabels(t *testing.T) {
	m := makeTestManager(t)
	imm := makeTestSession(t, m, "Watcher", 1001, true)
	imm.player.SetLevel(game.LVL_IMPL)
	imm.connectionNumber = 1
	m.mu.Lock()
	m.sessions[imm.player.Name] = imm
	m.mu.Unlock()

	preLogin := m.NewSession()
	preLogin.SetRemoteIP("127.0.0.1")

	creationRoll := m.NewSession()
	creationRoll.SetRemoteIP("127.0.0.1")
	creationRoll.charCreating = true
	creationRoll.charStage = "stats_roll"
	creationRoll.charName = "Roller"

	creationMotd := m.NewSession()
	creationMotd.SetRemoteIP("127.0.0.1")
	creationMotd.charCreating = true
	creationMotd.charStage = "motd"
	creationMotd.charName = "MotdPeer"

	menuDesc := makeTestSession(t, m, "DescPeer", 1001, true)
	menuDesc.authenticated = true
	menuDesc.menuActive = true
	menuDesc.menuStage = "description"
	menuDesc.textEdit = &textEditState{}
	m.mu.Lock()
	m.sessions[menuDesc.player.Name] = menuDesc
	m.mu.Unlock()

	del1 := makeTestSession(t, m, "DelOne", 1001, true)
	del1.authenticated = true
	del1.menuActive = true
	del1.menuStage = "delete_password"
	m.mu.Lock()
	m.sessions[del1.player.Name] = del1
	m.mu.Unlock()

	del2 := makeTestSession(t, m, "DelTwo", 1001, true)
	del2.authenticated = true
	del2.menuActive = true
	del2.menuStage = "delete_confirm"
	m.mu.Lock()
	m.sessions[del2.player.Name] = del2
	m.mu.Unlock()

	if err := cmdUsers(imm, nil); err != nil {
		t.Fatalf("cmdUsers: %v", err)
	}
	got := readSessionText(t, imm)
	for _, want := range []string{
		"UNDEFINED    Get name",
		"Roller       Rolling abilities 2",
		"MotdPeer     Reading MOTD",
		"DescPeer     Get descript.",
		"DelOne       Self-Delete 1",
		"DelTwo       Self-Delete 2",
		"\r\n7 visible sockets connected.\r\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("users output missing %q:\n%s", want, got)
		}
	}

	preLogin.LoseDescriptor()
	if err := cmdUsers(imm, nil); err != nil {
		t.Fatalf("cmdUsers after LoseDescriptor: %v", err)
	}
	gotAfter := readSessionText(t, imm)
	if strings.Contains(gotAfter, "UNDEFINED") || !strings.Contains(gotAfter, "\r\n6 visible sockets connected.\r\n") {
		t.Fatalf("users output did not drop closed pre-login descriptor:\n%s", gotAfter)
	}
}

func TestCmdUsersSwitchedIdleUsesBodyLevel(t *testing.T) {
	m := makeTestManagerWithMobs(t)
	viewer := makeTestSession(t, m, "Viewer", 1001, true)
	viewer.player.SetLevel(game.LVL_GRGOD) // 39
	viewer.connectionNumber = 2

	owner := makeTestSession(t, m, "Owner", 1001, true)
	owner.player.SetLevel(game.LVL_IMPL) // 40
	owner.connectionNumber = 1
	owner.connectedAt = time.Date(2026, time.September, 4, 22, 58, 48, 0, time.UTC)
	owner.SetRemoteIP("127.0.0.1")

	mob := registerMob(t, m, 2001, 1001)
	mob.SetLevel(9)
	owner.isSwitched = true
	owner.switchedOriginal = owner.player
	owner.switchedMob = mob

	m.mu.Lock()
	m.sessions[viewer.player.Name] = viewer
	m.sessions[owner.player.Name] = owner
	m.mu.Unlock()

	if err := cmdUsers(viewer, nil); err != nil {
		t.Fatalf("cmdUsers: %v", err)
	}
	got := readSessionText(t, viewer)
	wantRow := "  1 [40 Mu] Owner        Switched         0 22:58:48 [Hostname unknown]"
	if !strings.Contains(got, wantRow) {
		t.Fatalf("users switched idle row = %q, want %q", got, wantRow)
	}
}

func TestCmdWhoisOffline(t *testing.T) {
	database := testutil.NewMockDatabase()
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Room A", Zone: 1},
		},
		Mobs: []parser.Mob{},
		Objs: []parser.Obj{},
	}
	w, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	m := newTestManager(t, w, database)
	s := makeTestSession(t, m, "OnlinePlayer", 1001, true)

	// Create offline player in DB
	err = database.CreatePlayer(&db.PlayerRecord{
		Name:     "OfflineHero",
		Level:    25,
		Class:    game.ClassWarrior,
		Title:    "the Veteran of the North",
		RoomVNum: 1001,
	})
	if err != nil {
		t.Fatalf("CreatePlayer failed: %v", err)
	}

	// 1. No args
	if err := cmdWhois(s, nil); err != nil {
		t.Fatalf("cmdWhois nil args: %v", err)
	}
	if got := readSessionText(t, s); got != "For whom do you wish to search?\r\n" {
		t.Errorf("no-arg got %q, want %q", got, "For whom do you wish to search?\r\n")
	}

	// 2. Offline player lookup (persisted title)
	if err := cmdWhois(s, []string{"OfflineHero"}); err != nil {
		t.Fatalf("cmdWhois: %v", err)
	}
	wantOffline := fmt.Sprintf("[%2d %s] %s %s\r\n", 25, "Wa", "OfflineHero", "the Veteran of the North")
	if got := readSessionText(t, s); got != wantOffline {
		t.Errorf("offline whois: got %q, want %q", got, wantOffline)
	}

	// 3. Unknown player lookup
	if err := cmdWhois(s, []string{"NoSuchHero"}); err != nil {
		t.Fatalf("cmdWhois: %v", err)
	}
	if got := readSessionText(t, s); got != "There is no such player.\r\n" {
		t.Errorf("unknown whois: got %q, want %q", got, "There is no such player.\r\n")
	}
}
