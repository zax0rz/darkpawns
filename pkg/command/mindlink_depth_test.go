package command

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestSendSkillResultMindlinkFailureOrder pins the NPC failure state and
// audience path that is otherwise silent in a telnet transcript. C emits the
// room act, then improves the skill, then sends the final actor line, and only
// then stuns the actor (new_cmds2.c:309-325).
func TestSendSkillResultMindlinkFailureOrder(t *testing.T) {
	world, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Test Room", Zone: 1}},
		Mobs:  []parser.Mob{{VNum: 2004, Keywords: "dummy", ShortDesc: "a full-mana dummy", Position: combat.PosStanding}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	messages := make(map[string]string)
	world.MessageSink = func(name string, message []byte) { messages[name] += string(message) }

	actor := game.NewPlayer(1, "Mindlinker", 1001)
	actor.SetSkill(game.SkillMindlink, 100)
	actor.SetHP(200)
	actor.SetPosition(combat.PosStanding)
	if err := world.AddPlayer(actor); err != nil {
		t.Fatalf("AddPlayer actor: %v", err)
	}
	observer := game.NewPlayer(2, "Observer", 1001)
	if err := world.AddPlayer(observer); err != nil {
		t.Fatalf("AddPlayer observer: %v", err)
	}
	mob, err := world.SpawnMobQuiet(2004, 1001)
	if err != nil {
		t.Fatalf("SpawnMobQuiet: %v", err)
	}
	mob.SetMana(100)

	result := game.DoMindlink(actor, mob)
	session := &skillCommandSession{player: actor, world: world}
	if err := sendSkillResult(session, actor, mob, result); err != nil {
		t.Fatalf("sendSkillResult: %v", err)
	}

	if got := actor.GetHP(); got != 100 {
		t.Fatalf("actor HP = %d, want 100", got)
	}
	if got := actor.GetPosition(); got != combat.PosStunned {
		t.Fatalf("actor position = %d, want stunned %d", got, combat.PosStunned)
	}
	if got := messages[observer.Name]; !strings.Contains(got, "Mindlinker stares at a full-mana dummy for a while and then falls flat on his face.") {
		t.Fatalf("observer output = %q, want room failure act", got)
	}
	if got := strings.Join(session.messages, ""); !strings.Contains(got, "You feel a little drained...") {
		t.Fatalf("actor output = %q, want final drain line", got)
	}
}
