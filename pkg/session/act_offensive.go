package session

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
)

// broadcastCombatMsg encodes and broadcasts a combat event message to a room.
func broadcastCombatMsg(s *Session, roomVNum int, eventType, text string) {
	msg, err := json.Marshal(ServerMessage{
		Type: MsgEvent,
		Data: EventData{
			Type: eventType,
			From: s.player.Name,
			Text: text,
		},
	})
	if err != nil {
		slog.Error("broadcastCombatMsg marshal", "error", err)
		return
	}
	s.manager.BroadcastToRoom(roomVNum, msg, s.player.Name)
}

// findMobInRoom finds a mob in the player's current room by partial name match.
// Returns nil if not found.
func findMobInRoom(s *Session) func(name string) interface {
	GetShortDesc() string
	GetName() string
} {
	return nil // see inline usage below
}

// cmdAssist — assist a target in their combat (LVL_IMMORT).
func cmdAssist(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) == 0 {
		s.Send("Assist whom?")
		return nil
	}

	targetName := strings.ToLower(args[0])
	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	// Find the player being assisted
	for _, p := range s.manager.world.GetPlayersInRoom(room.VNum) {
		if p.Name == s.player.Name {
			continue
		}
		if !strings.Contains(strings.ToLower(p.Name), targetName) {
			continue
		}
		// Find who they're fighting
		opponent, fighting := s.manager.combatEngine.GetCombatTarget(p.Name)
		if !fighting {
			s.Send(fmt.Sprintf("%s is not in combat.", p.Name))
			return nil
		}
		if s.manager.combatEngine.IsFighting(s.player.Name) {
			s.Send("You're already fighting someone!")
			return nil
		}
		if err := s.manager.combatEngine.StartCombat(s.player, opponent); err != nil {
			s.Send(err.Error())
			return nil
		}
		s.Send(fmt.Sprintf("You jump to the aid of %s!", p.Name))
		broadcastCombatMsg(s, room.VNum, "assist",
			fmt.Sprintf("%s jumps to the aid of %s!", s.player.Name, p.Name))
		s.markDirty(VarFighting)
		return nil
	}

	s.Send("They aren't here.")
	return nil
}

// cmdKill — attack a non-player (mob) target.
func cmdKill(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Kill who?")
		return nil
	}

	targetName := strings.ToLower(args[0])
	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	if s.manager.combatEngine.IsFighting(s.player.Name) {
		s.Send("You're already fighting!")
		return nil
	}

	mobs := s.manager.world.GetMobsInRoom(room.VNum)
	for _, mob := range mobs {
		if strings.Contains(strings.ToLower(mob.GetShortDesc()), targetName) ||
			strings.Contains(strings.ToLower(mob.GetName()), targetName) {
			if err := s.manager.combatEngine.StartCombat(s.player, mob); err != nil {
				s.Send(err.Error())
				return nil
			}
			s.Send(fmt.Sprintf("You lunge at %s!", mob.GetShortDesc()))
			broadcastCombatMsg(s, room.VNum, "combat",
				fmt.Sprintf("%s attacks %s!", s.player.Name, mob.GetShortDesc()))
			s.markDirty(VarFighting)
			return nil
		}
	}

	s.Send("They aren't here.")
	return nil
}

// cmdBackstab — backstab a target (requires piercing weapon, sneak/hide).
func cmdBackstab(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Backstab who?")
		return nil
	}

	targetName := strings.ToLower(args[0])
	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	if s.manager.combatEngine.IsFighting(s.player.Name) {
		s.Send("You can't backstab while you're fighting!")
		return nil
	}

	mobs := s.manager.world.GetMobsInRoom(room.VNum)
	for _, mob := range mobs {
		if strings.Contains(strings.ToLower(mob.GetShortDesc()), targetName) ||
			strings.Contains(strings.ToLower(mob.GetName()), targetName) {
			if err := s.manager.combatEngine.StartCombat(s.player, mob); err != nil {
				s.Send(err.Error())
				return nil
			}
			s.Send(fmt.Sprintf("You plunge your blade into the back of %s!", mob.GetShortDesc()))
			broadcastCombatMsg(s, room.VNum, "backstab",
				fmt.Sprintf("%s backstabs %s!", s.player.Name, mob.GetShortDesc()))
			s.markDirty(VarFighting)
			return nil
		}
	}

	s.Send("They aren't here.")
	return nil
}

// cmdBash — bash a target.
func cmdBash(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Bash who?")
		return nil
	}

	targetName := strings.ToLower(args[0])
	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	mobs := s.manager.world.GetMobsInRoom(room.VNum)
	for _, mob := range mobs {
		if strings.Contains(strings.ToLower(mob.GetShortDesc()), targetName) ||
			strings.Contains(strings.ToLower(mob.GetName()), targetName) {
			if !s.manager.combatEngine.IsFighting(s.player.Name) {
				_ = s.manager.combatEngine.StartCombat(s.player, mob)
			}
			s.Send(fmt.Sprintf("You bash %s with all your might!", mob.GetShortDesc()))
			broadcastCombatMsg(s, room.VNum, "bash",
				fmt.Sprintf("%s bashes %s!", s.player.Name, mob.GetShortDesc()))
			s.markDirty(VarFighting)
			return nil
		}
	}

	s.Send("They aren't here.")
	return nil
}

// cmdDisembowel — disembowel a target (graphic combat action).
func cmdDisembowel(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Disembowel who?")
		return nil
	}

	targetName := strings.ToLower(args[0])
	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	mobs := s.manager.world.GetMobsInRoom(room.VNum)
	for _, mob := range mobs {
		if strings.Contains(strings.ToLower(mob.GetShortDesc()), targetName) ||
			strings.Contains(strings.ToLower(mob.GetName()), targetName) {
			if !s.manager.combatEngine.IsFighting(s.player.Name) {
				_ = s.manager.combatEngine.StartCombat(s.player, mob)
			}
			s.Send(fmt.Sprintf("You drive your blade deep into %s's gut, spilling entrails everywhere!", mob.GetShortDesc()))
			broadcastCombatMsg(s, room.VNum, "disembowel",
				fmt.Sprintf("%s disembowels %s in a shower of gore!", s.player.Name, mob.GetShortDesc()))
			s.markDirty(VarFighting)
			return nil
		}
	}

	s.Send("They aren't here.")
	return nil
}

// cmdRescue — rescue another player from combat.
func cmdRescue(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Rescue who?")
		return nil
	}

	targetName := strings.ToLower(args[0])
	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	for _, p := range s.manager.world.GetPlayersInRoom(room.VNum) {
		if p.Name == s.player.Name {
			continue
		}
		if !strings.Contains(strings.ToLower(p.Name), targetName) {
			continue
		}
		// Take their combat opponent
		opponent, fighting := s.manager.combatEngine.GetCombatTarget(p.Name)
		if !fighting {
			s.Send(fmt.Sprintf("%s doesn't need rescuing!", p.Name))
			return nil
		}
		s.manager.combatEngine.StopCombat(p.Name)
		if err := s.manager.combatEngine.StartCombat(s.player, opponent); err != nil {
			s.Send(err.Error())
			return nil
		}
		s.Send(fmt.Sprintf("You valiantly rescue %s!", p.Name))
		if target, ok := s.manager.GetSession(p.Name); ok {
			target.Send(fmt.Sprintf("%s rescues you!", s.player.Name))
		}
		broadcastCombatMsg(s, room.VNum, "rescue",
			fmt.Sprintf("%s rescues %s!", s.player.Name, p.Name))
		s.markDirty(VarFighting)
		return nil
	}

	s.Send("They aren't here.")
	return nil
}

// cmdKick — kick a target.
func cmdKick(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Kick who?")
		return nil
	}

	targetName := strings.ToLower(args[0])
	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	mobs := s.manager.world.GetMobsInRoom(room.VNum)
	for _, mob := range mobs {
		if strings.Contains(strings.ToLower(mob.GetShortDesc()), targetName) ||
			strings.Contains(strings.ToLower(mob.GetName()), targetName) {
			if !s.manager.combatEngine.IsFighting(s.player.Name) {
				_ = s.manager.combatEngine.StartCombat(s.player, mob)
			}
			s.Send(fmt.Sprintf("You deliver a powerful kick to %s!", mob.GetShortDesc()))
			broadcastCombatMsg(s, room.VNum, "kick",
				fmt.Sprintf("%s kicks %s!", s.player.Name, mob.GetShortDesc()))
			s.markDirty(VarFighting)
			return nil
		}
	}

	s.Send("They aren't here.")
	return nil
}

// cmdDragonKick — dragon-style kick attack.
func cmdDragonKick(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Dragon kick whom?")
		return nil
	}

	targetName := strings.ToLower(args[0])
	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	mobs := s.manager.world.GetMobsInRoom(room.VNum)
	for _, mob := range mobs {
		if strings.Contains(strings.ToLower(mob.GetShortDesc()), targetName) ||
			strings.Contains(strings.ToLower(mob.GetName()), targetName) {
			if !s.manager.combatEngine.IsFighting(s.player.Name) {
				_ = s.manager.combatEngine.StartCombat(s.player, mob)
			}
			s.Send(fmt.Sprintf("You unleash a devastating dragon kick against %s!", mob.GetShortDesc()))
			broadcastCombatMsg(s, room.VNum, "dragon_kick",
				fmt.Sprintf("%s dragon kicks %s!", s.player.Name, mob.GetShortDesc()))
			s.markDirty(VarFighting)
			return nil
		}
	}

	s.Send("They aren't here.")
	return nil
}

// cmdTigerPunch — tiger-style punch attack.
func cmdTigerPunch(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Tiger punch whom?")
		return nil
	}

	targetName := strings.ToLower(args[0])
	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	mobs := s.manager.world.GetMobsInRoom(room.VNum)
	for _, mob := range mobs {
		if strings.Contains(strings.ToLower(mob.GetShortDesc()), targetName) ||
			strings.Contains(strings.ToLower(mob.GetName()), targetName) {
			if !s.manager.combatEngine.IsFighting(s.player.Name) {
				_ = s.manager.combatEngine.StartCombat(s.player, mob)
			}
			s.Send(fmt.Sprintf("You snap a lightning-fast tiger punch into %s!", mob.GetShortDesc()))
			broadcastCombatMsg(s, room.VNum, "tiger_punch",
				fmt.Sprintf("%s tiger punches %s!", s.player.Name, mob.GetShortDesc()))
			s.markDirty(VarFighting)
			return nil
		}
	}

	s.Send("They aren't here.")
	return nil
}

// cmdShoot — ranged attack with bow or gun.
func cmdShoot(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Shoot who?")
		return nil
	}

	targetName := strings.ToLower(args[0])
	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	mobs := s.manager.world.GetMobsInRoom(room.VNum)
	for _, mob := range mobs {
		if strings.Contains(strings.ToLower(mob.GetShortDesc()), targetName) ||
			strings.Contains(strings.ToLower(mob.GetName()), targetName) {
			if !s.manager.combatEngine.IsFighting(s.player.Name) {
				_ = s.manager.combatEngine.StartCombat(s.player, mob)
			}
			s.Send(fmt.Sprintf("You fire at %s!", mob.GetShortDesc()))
			broadcastCombatMsg(s, room.VNum, "shoot",
				fmt.Sprintf("%s shoots at %s!", s.player.Name, mob.GetShortDesc()))
			s.markDirty(VarFighting)
			return nil
		}
	}

	s.Send("They aren't here.")
	return nil
}

// cmdSubdue — non-lethal subduing attack.
func cmdSubdue(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Subdue who?")
		return nil
	}

	targetName := strings.ToLower(args[0])
	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	mobs := s.manager.world.GetMobsInRoom(room.VNum)
	for _, mob := range mobs {
		if strings.Contains(strings.ToLower(mob.GetShortDesc()), targetName) ||
			strings.Contains(strings.ToLower(mob.GetName()), targetName) {
			if !s.manager.combatEngine.IsFighting(s.player.Name) {
				_ = s.manager.combatEngine.StartCombat(s.player, mob)
			}
			s.Send(fmt.Sprintf("You attempt to subdue %s!", mob.GetShortDesc()))
			broadcastCombatMsg(s, room.VNum, "subdue",
				fmt.Sprintf("%s tries to subdue %s!", s.player.Name, mob.GetShortDesc()))
			s.markDirty(VarFighting)
			return nil
		}
	}

	s.Send("They aren't here.")
	return nil
}

// cmdSleeper — put target to sleep.
func cmdSleeper(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Use a sleeper hold on who?")
		return nil
	}

	targetName := strings.ToLower(args[0])
	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	mobs := s.manager.world.GetMobsInRoom(room.VNum)
	for _, mob := range mobs {
		if strings.Contains(strings.ToLower(mob.GetShortDesc()), targetName) ||
			strings.Contains(strings.ToLower(mob.GetName()), targetName) {
			if !s.manager.combatEngine.IsFighting(s.player.Name) {
				_ = s.manager.combatEngine.StartCombat(s.player, mob)
			}
			s.Send(fmt.Sprintf("You apply a sleeper hold to %s!", mob.GetShortDesc()))
			broadcastCombatMsg(s, room.VNum, "sleeper",
				fmt.Sprintf("%s applies a sleeper hold to %s!", s.player.Name, mob.GetShortDesc()))
			s.markDirty(VarFighting)
			return nil
		}
	}

	s.Send("They aren't here.")
	return nil
}

// cmdNeckbreak — lethal stealth attack.
func cmdNeckbreak(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Neckbreak who?")
		return nil
	}

	targetName := strings.ToLower(args[0])
	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	mobs := s.manager.world.GetMobsInRoom(room.VNum)
	for _, mob := range mobs {
		if strings.Contains(strings.ToLower(mob.GetShortDesc()), targetName) ||
			strings.Contains(strings.ToLower(mob.GetName()), targetName) {
			if !s.manager.combatEngine.IsFighting(s.player.Name) {
				_ = s.manager.combatEngine.StartCombat(s.player, mob)
			}
			s.Send(fmt.Sprintf("You snap %s's neck with a sickening crack!", mob.GetShortDesc()))
			broadcastCombatMsg(s, room.VNum, "neckbreak",
				fmt.Sprintf("%s breaks %s's neck!", s.player.Name, mob.GetShortDesc()))
			s.markDirty(VarFighting)
			return nil
		}
	}

	s.Send("They aren't here.")
	return nil
}

// cmdAmbush — ambush attack from hiding.
func cmdAmbush(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Ambush who?")
		return nil
	}

	targetName := strings.ToLower(args[0])
	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	if s.manager.combatEngine.IsFighting(s.player.Name) {
		s.Send("You can't ambush while you're already fighting!")
		return nil
	}

	mobs := s.manager.world.GetMobsInRoom(room.VNum)
	for _, mob := range mobs {
		if strings.Contains(strings.ToLower(mob.GetShortDesc()), targetName) ||
			strings.Contains(strings.ToLower(mob.GetName()), targetName) {
			if err := s.manager.combatEngine.StartCombat(s.player, mob); err != nil {
				s.Send(err.Error())
				return nil
			}
			s.Send(fmt.Sprintf("You spring from the shadows and ambush %s!", mob.GetShortDesc()))
			broadcastCombatMsg(s, room.VNum, "ambush",
				fmt.Sprintf("%s leaps from the shadows to ambush %s!", s.player.Name, mob.GetShortDesc()))
			s.markDirty(VarFighting)
			return nil
		}
	}

	s.Send("They aren't here.")
	return nil
}

// cmdOrder — order a pet or follower to perform a command (LVL_IMMORT).
func cmdOrder(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) < 2 {
		s.Send("Order whom to do what?")
		return nil
	}

	targetName := strings.ToLower(args[0])
	orderCmd := strings.Join(args[1:], " ")

	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	for _, mob := range s.manager.world.GetMobsInRoom(room.VNum) {
		if strings.Contains(strings.ToLower(mob.GetName()), targetName) ||
			strings.Contains(strings.ToLower(mob.GetShortDesc()), targetName) {
			s.Send(fmt.Sprintf("%s obeys your order: %s", mob.GetShortDesc(), orderCmd))
			broadcastCombatMsg(s, room.VNum, "order",
				fmt.Sprintf("%s orders %s to '%s'.", s.player.Name, mob.GetShortDesc(), orderCmd))
			return nil
		}
	}

	s.Send("No follower by that name here.")
	return nil
}
