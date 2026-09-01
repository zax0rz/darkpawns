package command

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// TestCmdKickUsesCOneArgument proves the command-layer target parser follows
// do_kick's one_argument call: fill words are skipped and trailing words are
// ignored, so the first real target token remains authoritative (R1/R2/R5e).
func TestCmdKickUsesCOneArgument(t *testing.T) {
	ktw := newKillTestWorld(t, 100, 0, 0, 1, "rat")
	p := ktw.addPlayer(t, 1, "Kicker", 10, game.ClassWarrior, false)
	p.SetPosition(combat.PosFighting)
	p.SetSkill(game.SkillKick, 1)
	sess := &killPayoutSession{player: p, world: ktw.world}

	if err := CmdKick(sess, []string{"at", "rat", "trailing", "words"}); err != nil {
		t.Fatalf("CmdKick: %v", err)
	}
	if sess.hasMessage("Kick who?") {
		t.Fatalf("C one_argument target was not resolved: messages=%v", sess.getMessages())
	}
}

func TestCmdKickTargetGates(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		fighting   bool
		wantPrompt bool
	}{
		{name: "no argument", wantPrompt: true},
		{name: "missing target", args: []string{"missing"}, wantPrompt: true},
		{name: "fighting fallback", fighting: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ktw := newKillTestWorld(t, 100, 0, 0, 1, "rat")
			p := ktw.addPlayer(t, 1, "Kicker", 10, game.ClassWarrior, false)
			p.SetPosition(combat.PosFighting)
			p.SetSkill(game.SkillKick, 1)
			if tt.fighting {
				p.SetFighting(ktw.mob.GetName())
			}
			sess := &killPayoutSession{player: p, world: ktw.world}

			if err := CmdKick(sess, tt.args); err != nil {
				t.Fatalf("CmdKick: %v", err)
			}
			gotPrompt := sess.hasMessage("Kick who?")
			if gotPrompt != tt.wantPrompt {
				t.Fatalf("Kick who? = %v, want %v; messages=%v", gotPrompt, tt.wantPrompt, sess.getMessages())
			}
		})
	}
}
