package session

import (
	"fmt"
	"strings"
	"github.com/zax0rz/darkpawns/pkg/combat"
)

func cmdAssist(s *Session, args []string) error {
	// 1. Player must not already be fighting
	if s.manager.combatEngine.IsFighting(s.player.Name) {
		s.Send("You're already fighting! How can you assist someone else?\r\n")
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
		if p.Name == s.player.Name {
			continue
		}
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
		s.Send("They don't seem to be here.\r\n")
		return nil
	}

	// Find who is fighting the helpee
	opponent, fighting := s.manager.combatEngine.GetCombatTarget(helpeeName)
	if !fighting {
		s.Send(fmt.Sprintf("But nobody is fighting %s!\r\n", helpeeName))
		return nil
	}

	// 4. Player joins the fight
	if err := s.manager.combatEngine.StartCombat(s.player, opponent); err != nil {
		s.Send(err.Error())
		return nil
	}
	s.Send("You join the fight!\r\n")
	// Notify the helpee
	if !helpee.IsNPC() {
		if helpeeSess, ok := s.manager.GetSession(helpeeName); ok {
			helpeeSess.Send(fmt.Sprintf("%s assists you!\r\n", s.player.Name))
		}
	}
	broadcastCombatMsg(s, room.VNum, "assist",
		fmt.Sprintf("%s assists %s.", s.player.Name, helpeeName))
	s.markDirty(VarFighting)
	return nil
}

// cmdKick — kick a target.
// Ported from do_kick() in src/act.offensive.c lines 587-633.
// Can target by name or default to current fight opponent.
// Skill check: percent=((7-(AC/10))<<1)+rand(1,101), prob=skill level.
// On hit: damage = level>>1, improve_skill. On miss: 0 damage.
// WAIT_STATE: PULSE_VIOLENCE + 2 = 2 ticks.
