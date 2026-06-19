package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestBucketBPart1CommandsRegistered guards the port-completion wiring for
// Bucket B Part 1 (see docs/port-reachability-map.md): real C top-level
// command names, plus their already-correct Go handlers, that were either
// missing, registered under the wrong name, or implemented but never called.
func TestBucketBPart1CommandsRegistered(t *testing.T) {
	wired := []string{
		"abilities", "glance", "mount", "reallyquit",
		"sip", "taste",
		"think", "insult", "dream",
		"credits", "news", "policy", "handbook", "future", "whoami", "version",
		"reroll", "unaffect",
	}
	for _, name := range wired {
		if _, ok := cmdRegistry.Lookup(name); !ok {
			t.Errorf("command %q is not registered (port-completion regression)", name)
		}
	}
}

// registerInWorld adds a player to the World and registers its session
// under the player's exact name, so Player.SendMessage (routed through
// World.MessageSink) can find a live session to deliver to.
func registerInWorld(t *testing.T, s *Session) {
	t.Helper()
	if err := s.manager.world.AddPlayer(s.player); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	s.manager.mu.Lock()
	s.manager.sessions[s.player.Name] = s
	s.manager.mu.Unlock()
}

// makeFoodItem builds an ITEM_FOOD object with the given bite count and
// poison flag, matching the parser.Obj Values layout used by act.item.c:
// Values[0] = bites left, Values[3] = poison flag.
func makeFoodItem(bites int, poisoned bool) *game.ObjectInstance {
	poison := 0
	if poisoned {
		poison = 1
	}
	return &game.ObjectInstance{
		VNum: 100,
		Prototype: &parser.Obj{
			VNum:      100,
			TypeFlag:  game.ITEM_FOOD,
			Keywords:  "bread",
			ShortDesc: "a loaf of bread",
			Values:    [4]int{bites, 0, 0, poison},
		},
	}
}

// makeDrinkItem builds an ITEM_DRINKCON object (type 17) with the given
// liquid amount and poison flag — Values[1] = amount left, Values[3] = poison.
func makeDrinkItem(amount int, poisoned bool) *game.ObjectInstance {
	poison := 0
	if poisoned {
		poison = 1
	}
	return &game.ObjectInstance{
		VNum: 101,
		Prototype: &parser.Obj{
			VNum:      101,
			TypeFlag:  17, // ITEM_DRINKCON
			Keywords:  "waterskin",
			ShortDesc: "a waterskin",
			Values:    [4]int{0, amount, 0, poison},
		},
	}
}

// ---------------------------------------------------------------------------
// sip vs drink
// ---------------------------------------------------------------------------

func TestCmdSipDoesNotConsumeOrPoison(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	item := makeDrinkItem(5, true)
	if err := s.player.Inventory.AddItem(item); err != nil {
		t.Fatalf("AddItem: %v", err)
	}

	if err := cmdSip(s, []string{"waterskin"}); err != nil {
		t.Fatalf("cmdSip: %v", err)
	}
	if msg := readSessionText(t, s); !strings.Contains(msg, "tastes like") {
		t.Errorf("sip message: got %q", msg)
	}
	if item.Prototype.Values[1] != 5 {
		t.Errorf("sip should not deplete the container, got Values[1]=%d", item.Prototype.Values[1])
	}
	if len(s.player.ActiveAffects) != 0 {
		t.Errorf("sip should not apply a poison affect even on a poisoned drink, got %d affects", len(s.player.ActiveAffects))
	}
}

func TestCmdDrinkConsumesAndAppliesPoison(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	item := makeDrinkItem(5, true)
	if err := s.player.Inventory.AddItem(item); err != nil {
		t.Fatalf("AddItem: %v", err)
	}

	if err := cmdDrink(s, []string{"waterskin"}); err != nil {
		t.Fatalf("cmdDrink: %v", err)
	}
	if msg := readSessionText(t, s); !strings.Contains(msg, "You drink the") {
		t.Errorf("drink message: got %q", msg)
	}
	if item.Prototype.Values[1] != 4 {
		t.Errorf("drink should deplete the container by 1, got Values[1]=%d", item.Prototype.Values[1])
	}
	if len(s.player.ActiveAffects) != 1 {
		t.Errorf("drink should apply a poison affect on a poisoned drink, got %d affects", len(s.player.ActiveAffects))
	}
}

// ---------------------------------------------------------------------------
// taste vs eat
// ---------------------------------------------------------------------------

func TestCmdTasteDecrementsBitesWithoutFullyConsuming(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	item := makeFoodItem(3, false)
	if err := s.player.Inventory.AddItem(item); err != nil {
		t.Fatalf("AddItem: %v", err)
	}

	if err := cmdTaste(s, []string{"bread"}); err != nil {
		t.Fatalf("cmdTaste: %v", err)
	}
	if msg := readSessionText(t, s); !strings.Contains(msg, "nibble") {
		t.Errorf("taste message: got %q", msg)
	}
	if item.Prototype.Values[0] != 2 {
		t.Errorf("taste should decrement bites by 1, got Values[0]=%d", item.Prototype.Values[0])
	}
	if _, found := s.player.Inventory.FindItem("bread"); !found {
		t.Errorf("taste should not remove the item while bites remain")
	}
}

func TestCmdTasteRemovesItemOnceBitesRunOut(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	item := makeFoodItem(1, false)
	if err := s.player.Inventory.AddItem(item); err != nil {
		t.Fatalf("AddItem: %v", err)
	}

	if err := cmdTaste(s, []string{"bread"}); err != nil {
		t.Fatalf("cmdTaste: %v", err)
	}
	readSessionText(t, s) // "You nibble..."
	if msg := readSessionText(t, s); !strings.Contains(msg, "nothing left") {
		t.Errorf("expected 'nothing left' message, got %q", msg)
	}
	if _, found := s.player.Inventory.FindItem("bread"); found {
		t.Errorf("taste should remove the item once bites reach 0")
	}
}

func TestCmdEatRemovesItemImmediately(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	item := makeFoodItem(3, true)
	if err := s.player.Inventory.AddItem(item); err != nil {
		t.Fatalf("AddItem: %v", err)
	}

	if err := cmdEat(s, []string{"bread"}); err != nil {
		t.Fatalf("cmdEat: %v", err)
	}
	if msg := readSessionText(t, s); !strings.Contains(msg, "You eat") {
		t.Errorf("eat message: got %q", msg)
	}
	if _, found := s.player.Inventory.FindItem("bread"); found {
		t.Errorf("eat should remove the item regardless of remaining bites")
	}
	if len(s.player.ActiveAffects) != 1 {
		t.Errorf("eating poisoned food should apply a poison affect, got %d affects", len(s.player.ActiveAffects))
	}
}

// ---------------------------------------------------------------------------
// think
// ---------------------------------------------------------------------------

func TestCmdThinkNoArgs(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)

	if err := cmdThink(s, nil); err != nil {
		t.Fatalf("cmdThink: %v", err)
	}
	if msg := readSessionText(t, s); !strings.Contains(msg, "life, the universe, and everything") {
		t.Errorf("no-arg think message: got %q", msg)
	}
}

func TestCmdThinkBlockedByNoshout(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	s.player.Flags |= 1 << game.PlrNoshout

	if err := cmdThink(s, []string{"hello"}); err != nil {
		t.Fatalf("cmdThink: %v", err)
	}
	if msg := readSessionText(t, s); !strings.Contains(msg, "cannot") {
		t.Errorf("noshout-blocked think message: got %q", msg)
	}
}

func TestCmdThinkBlockedByZeroInt(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	// NewPlayer leaves Stats zero-valued, so Int == 0 here already —
	// matches C's GET_INT(ch) == 0 check.

	if err := cmdThink(s, []string{"hello"}); err != nil {
		t.Fatalf("cmdThink: %v", err)
	}
	if msg := readSessionText(t, s); !strings.Contains(msg, "cannot") {
		t.Errorf("zero-INT-blocked think message: got %q", msg)
	}
}

func TestCmdThinkWithArgsSendsPrivateThought(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	s.player.Stats.Int = 13

	if err := cmdThink(s, []string{"about", "tulips"}); err != nil {
		t.Fatalf("cmdThink: %v", err)
	}
	if msg := readSessionText(t, s); !strings.Contains(msg, "about tulips") {
		t.Errorf("think-with-args message: got %q", msg)
	}
}

func TestCmdThinkNoRepeatSendsOk(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	s.player.Stats.Int = 13
	s.player.Flags |= 1 << game.PrfNoRepeat

	if err := cmdThink(s, []string{"about", "tulips"}); err != nil {
		t.Fatalf("cmdThink: %v", err)
	}
	if msg := readSessionText(t, s); msg != "Ok." {
		t.Errorf("norepeat think message: got %q, want %q", msg, "Ok.")
	}
}

// ---------------------------------------------------------------------------
// insult
// ---------------------------------------------------------------------------

func TestCmdInsultNoTarget(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	registerInWorld(t, s)

	if err := cmdInsult(s, nil); err != nil {
		t.Fatalf("cmdInsult: %v", err)
	}
	if msg := readSessionText(t, s); !strings.Contains(msg, "everybody") {
		t.Errorf("no-target insult message: got %q", msg)
	}
}

func TestCmdInsultTargetNotFound(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	registerInWorld(t, s)

	if err := cmdInsult(s, []string{"bob"}); err != nil {
		t.Fatalf("cmdInsult: %v", err)
	}
	if msg := readSessionText(t, s); !strings.Contains(msg, "Can't hear you") {
		t.Errorf("target-not-found insult message: got %q", msg)
	}
}

func TestCmdInsultSelf(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	registerInWorld(t, s)

	if err := cmdInsult(s, []string{"alice"}); err != nil {
		t.Fatalf("cmdInsult: %v", err)
	}
	if msg := readSessionText(t, s); !strings.Contains(msg, "insulted") {
		t.Errorf("self-insult message: got %q", msg)
	}
}

func TestCmdInsultDeliversToTarget(t *testing.T) {
	m := makeTestManager(t)
	alice := makeTestSession(t, m, "Alice", 1001, true)
	bob := makeTestSession(t, m, "Bob", 1001, true)
	registerInWorld(t, alice)
	registerInWorld(t, bob)

	if err := cmdInsult(alice, []string{"bob"}); err != nil {
		t.Fatalf("cmdInsult: %v", err)
	}
	if msg := readSessionText(t, alice); !strings.Contains(msg, "You insult Bob") {
		t.Errorf("sender confirmation: got %q", msg)
	}
	if msg := readSessionText(t, bob); !strings.Contains(msg, "Alice") {
		t.Errorf("target should receive an insult naming the sender, got %q", msg)
	}
}

// ---------------------------------------------------------------------------
// dream
// ---------------------------------------------------------------------------

func TestCmdDreamWhileAwake(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	registerInWorld(t, s)
	// NewPlayer defaults to POS_STANDING.

	if err := cmdDream(s, nil); err != nil {
		t.Fatalf("cmdDream: %v", err)
	}
	if msg := readSessionText(t, s); !strings.Contains(msg, "daydream") {
		t.Errorf("awake dream message: got %q", msg)
	}
}

func TestCmdDreamWhileAsleep(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	registerInWorld(t, s)
	s.player.SetPosition(combat.PosSleeping)

	if err := cmdDream(s, nil); err != nil {
		t.Fatalf("cmdDream: %v", err)
	}
	if msg := readSessionText(t, s); !strings.Contains(msg, "tulips") {
		t.Errorf("asleep dream message: got %q", msg)
	}
}

// ---------------------------------------------------------------------------
// reroll / unaffect — level gating
// ---------------------------------------------------------------------------

func TestCmdRerollBlockedBelowGrgod(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	s.player.Level = LVL_GOD // one level short of LVL_GRGOD

	if err := cmdReroll(s, []string{"bob"}); err != nil {
		t.Fatalf("cmdReroll: %v", err)
	}
	if msg := readSessionText(t, s); msg != "Huh?!?" {
		t.Errorf("below-LVL_GRGOD reroll message: got %q", msg)
	}
}

func TestCmdRerollAtGrgod(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	s.player.Level = LVL_GRGOD
	target := makeTestSession(t, m, "Bob", 1001, true)
	m.mu.Lock()
	m.sessions["bob"] = target
	m.mu.Unlock()

	if err := cmdReroll(s, []string{"bob"}); err != nil {
		t.Fatalf("cmdReroll: %v", err)
	}
	if msg := readSessionText(t, s); !strings.Contains(msg, "Rerolled") {
		t.Errorf("at-LVL_GRGOD reroll message: got %q", msg)
	}
}

func TestCmdUnaffectBlockedBelowGod(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	s.player.Level = LVL_IMMORT // one level short of LVL_GOD

	if err := cmdUnaffect(s, []string{"bob"}); err != nil {
		t.Fatalf("cmdUnaffect: %v", err)
	}
	if msg := readSessionText(t, s); msg != "Huh?!?" {
		t.Errorf("below-LVL_GOD unaffect message: got %q", msg)
	}
}

func TestCmdUnaffectAtGodClearsAffects(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	s.player.Level = LVL_GOD
	target := makeTestSession(t, m, "Bob", 1001, true)
	target.player.ActiveAffects = []*engine.Affect{
		engine.NewAffectDirect(0, engine.ApplyNone, 10, 0, engine.AFFPoison, "test"),
	}
	m.mu.Lock()
	m.sessions["bob"] = target
	m.mu.Unlock()

	if err := cmdUnaffect(s, []string{"bob"}); err != nil {
		t.Fatalf("cmdUnaffect: %v", err)
	}
	if msg := readSessionText(t, s); !strings.Contains(msg, "All spells removed") {
		t.Errorf("unaffect-clears-affects message: got %q", msg)
	}
	if target.player.ActiveAffects != nil {
		t.Errorf("unaffect should clear the target's ActiveAffects, got %v", target.player.ActiveAffects)
	}
}

func TestCmdUnaffectNoAffects(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	s.player.Level = LVL_GOD
	target := makeTestSession(t, m, "Bob", 1001, true)
	m.mu.Lock()
	m.sessions["bob"] = target
	m.mu.Unlock()

	if err := cmdUnaffect(s, []string{"bob"}); err != nil {
		t.Fatalf("cmdUnaffect: %v", err)
	}
	if msg := readSessionText(t, s); !strings.Contains(msg, "affections") {
		t.Errorf("unaffect-with-no-affects message: got %q", msg)
	}
}

// ---------------------------------------------------------------------------
// static info-text commands
// ---------------------------------------------------------------------------

func TestCmdWhoami(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)

	if err := cmdWhoami(s, nil); err != nil {
		t.Fatalf("cmdWhoami: %v", err)
	}
	if msg := readSessionText(t, s); msg != "Alice" {
		t.Errorf("whoami message: got %q, want %q", msg, "Alice")
	}
}

func TestCmdVersionHidesBuildInfoForMortals(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)

	if err := cmdVersion(s, nil); err != nil {
		t.Fatalf("cmdVersion: %v", err)
	}
	if msg := readSessionText(t, s); !strings.Contains(msg, "Dark Pawns") {
		t.Errorf("version message: got %q", msg)
	}
}

func TestCmdCreditsMissingFileFailsGracefully(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)

	// go test's working directory is the package directory, not the repo
	// root, so lib/world/text/credits won't resolve here — exercises the
	// graceful-failure branch of sendTextFile.
	if err := cmdCredits(s, nil); err != nil {
		t.Fatalf("cmdCredits: %v", err)
	}
	if msg := readSessionText(t, s); !strings.Contains(msg, "not available") {
		t.Errorf("missing-file credits message: got %q", msg)
	}
}

func TestCmdCreditsReadsRealFile(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)

	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(filepath.Join(orig, "..", "..")); err != nil {
		t.Fatalf("Chdir to repo root: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	want, err := os.ReadFile("lib/world/text/credits")
	if err != nil {
		t.Fatalf("reading lib/world/text/credits directly: %v", err)
	}

	if err := cmdCredits(s, nil); err != nil {
		t.Fatalf("cmdCredits: %v", err)
	}
	if msg := readSessionText(t, s); msg != string(want) {
		t.Errorf("credits content mismatch: got %d bytes, want %d bytes", len(msg), len(want))
	}
}
