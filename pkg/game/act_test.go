package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestSanaUsesRuntimeKeywords verifies sana() consults GetKeywords() so
// synthetic objects (corpses, money) without a Prototype get the right
// article instead of always defaulting to "a".
func TestSanaUsesRuntimeKeywords(t *testing.T) {
	if got := sana(nil); got != "a" {
		t.Errorf("nil object: got %q, want %q", got, "a")
	}

	synthetic := &ObjectInstance{Runtime: ObjectRuntimeState{Keywords: "apple"}}
	if got := sana(synthetic); got != "an" {
		t.Errorf("synthetic object starting with a vowel: got %q, want %q", got, "an")
	}

	// GetKeywords() falls back to "object generic" for a totally bare object
	// (no Runtime.Keywords, no Prototype) — "object" starts with a vowel.
	bare := &ObjectInstance{}
	if got := sana(bare); got != "an" {
		t.Errorf("object with no keywords at all: got %q, want %q", got, "an")
	}
}

// TestObjNameUsesRuntimeKeywords verifies objName() falls back to
// GetKeywords() for synthetic, Prototype-less objects instead of always
// returning "something".
func TestObjNameUsesRuntimeKeywords(t *testing.T) {
	observer := NewPlayer(1, "Observer", 1001)

	synthetic := &ObjectInstance{Runtime: ObjectRuntimeState{Keywords: "corpse goblin"}}
	if got := objName(synthetic, observer); got != "corpse" {
		t.Errorf("got %q, want first keyword %q", got, "corpse")
	}

	if got := objName(nil, observer); got != "something" {
		t.Errorf("nil object: got %q, want %q", got, "something")
	}
}

func TestDoAction_PositionCheckExpandsTargetName(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Test Room", Zone: 1}},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	defer w.StopAITicker()

	actor := NewPlayer(1, "Hero", 1001)
	target := NewPlayer(2, "SleepyTarget", 1001)
	target.SetPosition(combat.PosSleeping)

	if err := w.AddPlayer(actor); err != nil {
		t.Fatalf("AddPlayer(actor) failed: %v", err)
	}
	if err := w.AddPlayer(target); err != nil {
		t.Fatalf("AddPlayer(target) failed: %v", err)
	}

	var captured []string
	w.MessageSink = func(playerName string, msg []byte) {
		captured = append(captured, string(msg))
	}

	// Inject a test social that requires the target to be standing.
	Socials["testsocial"] = &Social{
		Name:              "testsocial",
		MinVictimPosition: combat.PosStanding,
		Messages:          []string{"You test.", "$n tests.", "You test $M.", "$n tests $N.", "$n tests you.", "Test who?", "You test yourself.", "$n tests $mself."},
	}
	defer delete(Socials, "testsocial")

	DoAction(w, actor, "testsocial", "sleepy")

	found := false
	for _, msg := range captured {
		if strings.Contains(msg, "SleepyTarget is not in a proper position for that") {
			found = true
		}
		if strings.Contains(msg, "$N is not in a proper position for that") {
			t.Errorf("message contained raw $N token: %q", msg)
		}
	}
	if !found {
		t.Errorf("expected actor to see target name in position-failure message, got: %v", captured)
	}
}
