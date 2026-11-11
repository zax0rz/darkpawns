package command

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// C uses the victim as act()'s actor, so the attacker and other room members
// receive this line; perform_act appends after the embedded CRLF
// (src/new_cmds.c:2641-2644).
func TestSendSkillResultGroinripActUsesVictimRoomAudience(t *testing.T) {
	session := newSkillCommandSession(t)
	victim := game.NewPlayer(2, "Victim", 1001)
	observer := game.NewPlayer(3, "Observer", 1001)
	for _, player := range []*game.Player{victim, observer} {
		if err := session.world.AddPlayer(player); err != nil {
			t.Fatalf("AddPlayer %s: %v", player.Name, err)
		}
	}
	output := make(map[string]string)
	session.world.MessageSink = func(name string, msg []byte) { output[name] += string(msg) }
	result := game.SkillResult{
		DamageSkill:   game.SkillGroinrip,
		MessageToRoom: "Victim falls to his knees, clutching his groin and throwing up\r\neverywhere!",
	}
	if err := sendSkillResult(session, session.player, victim, result); err != nil {
		t.Fatalf("sendSkillResult: %v", err)
	}
	const want = "Victim falls to his knees, clutching his groin and throwing up\r\neverywhere!\r\n"
	for _, name := range []string{session.player.Name, observer.Name} {
		if got := output[name]; got != want {
			t.Errorf("%s output bytes = %q, want %q", name, got, want)
		}
	}
	if got := output[victim.Name]; got != "" {
		t.Fatalf("victim received its own TO_ROOM act: %q", got)
	}
	if session.player.GetPosition() != combat.PosStanding {
		t.Fatalf("attacker position = %d, want unchanged", session.player.GetPosition())
	}
}
