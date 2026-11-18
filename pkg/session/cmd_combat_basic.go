package session

import (
	"fmt"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func cmdAssist(s *Session, args []string) error {
	actor := s.combatBody()
	if actor == nil {
		return nil
	}
	actActor, ok := actor.(game.Actor)
	if !ok || actActor == nil {
		return nil
	}

	// 1. Actor must not already be fighting
	if s.manager.combatEngine.IsFighting(actor) {
		s.Send("You're already fighting!  How can you assist someone else?\r\n")
		return nil
	}

	// 2. If mounted, must dismount first
	if mounter, ok := actor.(interface{ IsMounted() bool }); ok && mounter.IsMounted() {
		s.Send("Dismount first!\r\n")
		return nil
	}

	if len(args) == 0 {
		s.Send("Whom do you wish to assist?\r\n")
		return nil
	}

	targetName := strings.ToLower(args[0])

	// 3. Find the target character in the room. C do_assist resolves the
	// helpee with get_char_room_vis (src/act.offensive.c:74 →
	// src/handler.c:1277-1300), then tests helpee == ch for the self-refusal.
	target, ok := s.manager.world.ResolveCharInRoom(actor, targetName)
	if !ok {
		// C get_char_room_vis failure prints NOPERSON (config.c:93).
		s.Send("No-one by that name here.\r\n")
		return nil
	}
	var helpee combat.Combatant
	helpeeName := ""
	if target.Player != nil {
		helpee, helpeeName = target.Player, target.Player.Name
	} else {
		helpee, helpeeName = target.Mob, target.Mob.GetShortDesc()
	}
	if targetActor(target) == actActor {
		s.Send("You can't help yourself any more than this!\r\n")
		return nil
	}

	// Find who is fighting the helpee
	opponent, fighting := s.manager.combatEngine.GetCombatTarget(helpee)
	if !fighting {
		// C uses $M (objective pronoun): "But nobody is fighting him!"
		if helpeeActor, ok := helpee.(game.Actor); ok {
			game.Act(nil, false, actActor, helpeeActor, nil, nil,
				"But nobody is fighting $M!", "", game.ToChar)
		} else {
			s.Send(fmt.Sprintf("But nobody is fighting %s!\r\n", helpeeName))
		}
		return nil
	}
	// C's do_assist refuses to join when the opponent fighting the helpee is
	// invisible to the assisting actor (act.offensive.c:87-89). The helpee is
	// the $M target of the message; no combat state or audience output changes
	// on this rejection.
	if helpeeActor, ok := helpee.(game.Actor); ok {
		if opponentActor, ok := opponent.(game.Actor); ok && !game.CanSee(actActor, opponentActor) {
			game.Act(nil, false, actActor, helpeeActor, nil, nil,
				"You can't see who is fighting $M!", "", game.ToChar)
			return nil
		}
	}

	// 4. Actor joins the fight. C's do_assist swings immediately via hit()
	// (act.offensive.c:95) and — unlike do_hit — sets NO wait state.
	s.Send("You join the fight!\r\n")
	// Notify the helpee (covers both players and switched mobs).
	if helpeeActor, ok := helpee.(game.Actor); ok {
		helpeeActor.SendMessage(fmt.Sprintf("%s assists you!\r\n", actor.GetName()))
	}
	// C act("$n assists $N.", ..., TO_NOTVICT) excludes both the actor and the
	// helpee — the helper must not hear their own assist line.
	if helpeeActor, ok := helpee.(game.Actor); ok {
		game.Act(s.manager.world, false, actActor, helpeeActor, nil, nil,
			"$n assists $N.", "", game.ToNotVict)
	}

	// do_assist calls hit() after its own audience messages
	// (act.offensive.c:95): the swing consumes hit()'s to-hit and damage
	// draws and only then reaches damage()'s protection block
	// (fight.c:1318-1368) against the pre-enrolment state. The helper enrols
	// at the damage point only when the damage is not refused.
	if err := s.manager.combatEngine.PerformDeferredInitialAttack(actor, opponent); err != nil {
		return err
	}
	s.markDirty(VarFighting)
	return nil
}

// cmdKick — kick a target.
// Ported from do_kick() in src/act.offensive.c lines 587-633.
// Can target by name or default to current fight opponent.
// Skill check: percent=((7-(AC/10))<<1)+rand(1,101), prob=skill level.
// On hit: damage = level>>1, improve_skill. On miss: 0 damage.
// WAIT_STATE: PULSE_VIOLENCE + 2 = 2 ticks.
