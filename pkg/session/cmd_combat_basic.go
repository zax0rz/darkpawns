package session

import (
	"fmt"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func cmdAssist(s *Session, args []string) error {
	// 1. Player must not already be fighting
	if s.manager.combatEngine.IsFighting(s.player.Name) {
		s.Send("You're already fighting!  How can you assist someone else?\r\n")
		return nil
	}

	// 2. If mounted, must dismount first
	if s.player.IsMounted() {
		s.Send("Dismount first!\r\n")
		return nil
	}

	if len(args) == 0 {
		s.Send("Whom do you wish to assist?\r\n")
		return nil
	}

	targetName := strings.ToLower(args[0])
	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	// 3. Find the target character in the room (players and mobs)
	var helpee combat.Combatant
	helpeeName := ""

	for _, p := range s.manager.world.GetPlayersInRoom(room.VNum) {
		if !strings.Contains(strings.ToLower(p.Name), targetName) {
			continue
		}
		helpee = p
		helpeeName = p.Name
		break
	}
	if helpee == nil {
		for _, m := range s.manager.world.GetMobsInRoom(room.VNum) {
			if !strings.Contains(strings.ToLower(m.GetShortDesc()), targetName) &&
				!strings.Contains(strings.ToLower(m.GetName()), targetName) {
				continue
			}
			helpee = m
			helpeeName = m.GetShortDesc()
			break
		}
	}
	if helpee == nil {
		// C get_char_room_vis failure prints NOPERSON (config.c:93).
		s.Send("No-one by that name here.\r\n")
		return nil
	}
	if !helpee.IsNPC() && helpeeName == s.player.Name {
		s.Send("You can't help yourself any more than this!\r\n")
		return nil
	}

	// Find who is fighting the helpee
	opponent, fighting := s.manager.combatEngine.GetCombatTarget(helpeeName)
	if !fighting {
		// C uses $M (objective pronoun): "But nobody is fighting him!"
		if helpeeActor, ok := helpee.(game.Actor); ok {
			game.Act(nil, false, s.player, helpeeActor, nil, nil,
				"But nobody is fighting $M!", "", game.ToChar)
		} else {
			s.Send(fmt.Sprintf("But nobody is fighting %s!\r\n", helpeeName))
		}
		return nil
	}
	// C's do_assist refuses to join when the opponent fighting the helpee is
	// invisible to the assisting player (act.offensive.c:87-89). The helpee is
	// the $M target of the message; no combat state or audience output changes
	// on this rejection.
	if helpeeActor, ok := helpee.(game.Actor); ok {
		if opponentActor, ok := opponent.(game.Actor); ok && !game.CanSee(s.player, opponentActor) {
			game.Act(nil, false, s.player, helpeeActor, nil, nil,
				"You can't see who is fighting $M!", "", game.ToChar)
			return nil
		}
	}

	// 4. Player joins the fight. C's do_assist swings immediately via hit()
	// (act.offensive.c:95) and — unlike do_hit — sets NO wait state.
	s.Send("You join the fight!\r\n")
	// Notify the helpee
	if !helpee.IsNPC() {
		if helpeeSess, ok := s.manager.GetSession(helpeeName); ok {
			helpeeSess.Send(fmt.Sprintf("%s assists you!\r\n", s.player.Name))
		}
	}
	// C act("$n assists $N.", ..., TO_NOTVICT) excludes both the actor and the
	// helpee — the helper must not hear their own assist line.
	if helpeeActor, ok := helpee.(game.Actor); ok {
		game.Act(s.manager.world, false, s.player, helpeeActor, nil, nil,
			"$n assists $N.", "", game.ToNotVict)
	}

	// do_assist calls hit() after its own audience messages. Preserve hit()'s
	// PC-vs-PC protection gates here before enrolling the helper, so the
	// immediate assist swing cannot bypass fight.c:1336-1357 when the opponent
	// is a low-level player. The mob-helpee path is observable when its opponent
	// is a player, even though the helpee itself receives no direct message.
	if !opponent.IsNPC() {
		if s.player.GetLevel() <= 10 {
			game.Act(nil, false, s.player, opponent.(game.Actor), nil, nil,
				"You are not experienced enough to attack $N!", "", game.ToChar)
			return nil
		}
		victimOutlaw := false
		if victim, ok := opponent.(*game.Player); ok {
			victimOutlaw = victim.GetFlags()&(1<<uint(game.PlrOutlaw)) != 0
		}
		if opponent.GetLevel() <= 10 && !victimOutlaw {
			game.Act(nil, false, s.player, opponent.(game.Actor), nil, nil,
				"Ancient forces protect $N from your wrath!", "", game.ToChar)
			return nil
		}
	}
	if err := s.manager.combatEngine.StartCombat(s.player, opponent); err != nil {
		s.Send(err.Error())
		return nil
	}
	if err := s.manager.combatEngine.PerformInitialAttack(s.player, opponent); err != nil {
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
