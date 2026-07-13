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

func TestPerformActSubstitutesEveryCode(t *testing.T) {
	actor := NewPlayer(1, "Hero", 1001)
	actor.Sex = 0
	victim := NewPlayer(2, "Target", 1001)
	victim.Sex = 1
	primary := NewObjectInstance(&parser.Obj{Keywords: "apple fruit", ShortDesc: "an apple"}, -1)
	secondary := NewObjectInstance(&parser.Obj{Keywords: "yellow orb", ShortDesc: "a yellow orb"}, -1)

	tests := []struct {
		name   string
		format string
		want   string
	}{
		{"actor name", "$n", "Hero"},
		{"victim name", "$N", "Target"},
		{"actor objective", "$m", "him"},
		{"victim objective", "$M", "her"},
		{"actor possessive", "$s", "his"},
		{"victim possessive", "$S", "her"},
		{"actor subjective", "$e", "he"},
		{"victim subjective", "$E", "she"},
		{"primary object name", "$o", "apple"},
		{"secondary object name", "$O", "yellow"},
		{"primary object description", "$p", "an apple"},
		{"secondary object description", "$P", "a yellow orb"},
		{"primary article", "$a", "an"},
		{"secondary article", "$A", "an"},
		{"first argument", "$t", "mixed Words"},
		{"second argument", "$T", "Second Reply"},
		{"capitalized first argument", "$r", "Mixed Words"},
		{"capitalized second argument", "$R", "Second Reply"},
		{"lowercase first argument", "$q", "mixed words"},
		{"lowercase second argument", "$Q", "second reply"},
		{"first word of second argument", "$F", "Second"},
		{"literal dollar", "$$", "$"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := performAct(tt.format, actor, victim, primary, secondary, "mixed Words", "Second Reply", actor)
			if got != tt.want {
				t.Fatalf("performAct(%q) = %q, want %q", tt.format, got, tt.want)
			}
		})
	}
}

func TestActPronounsUseGoSexEncoding(t *testing.T) {
	actor := NewPlayer(1, "Hero", 1001)
	tests := []struct {
		sex  int
		want string
	}{
		{sex: 0, want: "he/him/his"},
		{sex: 1, want: "she/her/her"},
		{sex: 2, want: "it/it/its"},
	}
	for _, tt := range tests {
		actor.Sex = tt.sex
		if got := performAct("$e/$m/$s", actor, nil, nil, nil, "", "", actor); got != tt.want {
			t.Errorf("sex %d pronouns = %q, want %q", tt.sex, got, tt.want)
		}
	}
}

func TestActAudienceRouting(t *testing.T) {
	w, actor, victim, observer, outsider := newActTestWorld(t)
	received := make(map[string][]string)
	w.MessageSink = func(name string, msg []byte) {
		received[name] = append(received[name], string(msg))
	}

	tests := []struct {
		name    string
		actType int
		want    []string
	}{
		{name: "char", actType: ToChar, want: []string{actor.Name}},
		{name: "victim", actType: ToVict, want: []string{victim.Name}},
		{name: "room", actType: ToRoom, want: []string{observer.Name, victim.Name}},
		{name: "not victim", actType: ToNotVict, want: []string{observer.Name}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clear(received)
			Act(w, false, actor, victim, nil, nil, "$n nods.", "", tt.actType)
			for _, player := range []*Player{actor, victim, observer, outsider} {
				want := containsName(tt.want, player.Name)
				if got := len(received[player.Name]) > 0; got != want {
					t.Errorf("delivery to %s = %t, want %t; received=%v", player.Name, got, want, received)
				}
			}
		})
	}

	clear(received)
	victim.SetPosition(combat.PosSleeping)
	Act(w, false, actor, victim, nil, nil, "$n wakes you.", "", ToVict)
	if len(received[victim.Name]) != 0 {
		t.Fatalf("sleeping victim received without ToSleep: %v", received)
	}
	Act(w, false, actor, victim, nil, nil, "$n wakes you.", "", ToVict|ToSleep)
	if got := received[victim.Name]; len(got) != 1 || got[0] != "Someone wakes you.\r\n" {
		t.Fatalf("ToSleep delivery = %v, want one substituted message", got)
	}
}

func TestActVisibilityIsRecipientSpecific(t *testing.T) {
	w, actor, _, observer, _ := newActTestWorld(t)
	observer.SetAffect(affBlind, true)
	obj := NewObjectInstance(&parser.Obj{Keywords: "apple fruit", ShortDesc: "an apple"}, -1)
	received := make(map[string][]string)
	w.MessageSink = func(name string, msg []byte) {
		received[name] = append(received[name], string(msg))
	}

	Act(w, false, actor, nil, obj, nil, "$n offers $p.", "", ToRoom)
	if got := received[observer.Name]; len(got) != 1 || got[0] != "Someone offers something.\r\n" {
		t.Fatalf("blind observer message = %v, want visibility-safe substitutions", got)
	}

	clear(received)
	Act(w, true, actor, nil, obj, nil, "$n offers $p.", "", ToRoom)
	if len(received[observer.Name]) != 0 {
		t.Fatalf("hideInvisible delivered to blind observer: %v", received[observer.Name])
	}
}

func newActTestWorld(t *testing.T) (*World, *Player, *Player, *Player, *Player) {
	t.Helper()
	parsed := &parser.World{Rooms: []parser.Room{
		{VNum: 1001, Name: "Room A", Zone: 1},
		{VNum: 1002, Name: "Room B", Zone: 1},
	}}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	actor := NewPlayer(1, "Hero", 1001)
	victim := NewPlayer(2, "Target", 1001)
	observer := NewPlayer(3, "Observer", 1001)
	outsider := NewPlayer(4, "Outsider", 1002)
	for _, player := range []*Player{actor, victim, observer, outsider} {
		if err := w.AddPlayer(player); err != nil {
			t.Fatalf("AddPlayer(%s): %v", player.Name, err)
		}
	}
	return w, actor, victim, observer, outsider
}

func containsName(names []string, name string) bool {
	for _, candidate := range names {
		if candidate == name {
			return true
		}
	}
	return false
}
