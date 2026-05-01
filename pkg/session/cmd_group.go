package session

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
)

func cmdFollow(s *Session, args []string) error {
	if len(args) == 0 {
		s.sendText("Whom do you wish to follow?")
		return nil
	}

	targetName := args[0]

	// follow self = stop following (act.movement.c line 912–917)
	if strings.EqualFold(targetName, s.player.Name) {
		if s.player.Following == "" {
			s.sendText("You are already following yourself.")
			return nil
		}
		oldLeader := s.player.Following
		s.player.Following = ""
		s.player.InGroup = false // REMOVE_BIT AFF_GROUP — act.movement.c line 926
		s.sendText(fmt.Sprintf("You stop following %s.", oldLeader))
		if leader, ok := s.manager.world.GetPlayer(oldLeader); ok {
			leader.SendMessage(fmt.Sprintf("%s stops following you.\r\n", s.player.Name))
		}
		return nil
	}

	// Find target — get_char_room_vis (act.movement.c line 895)
	target, ok := s.manager.world.GetPlayer(targetName)
	if !ok {
		s.sendText("There is no one by that name here.")
		return nil
	}
	if target.GetRoom() != s.player.GetRoom() {
		s.sendText("They are not here.")
		return nil
	}

	// Already following? (act.movement.c line 904)
	if s.player.Following == target.Name {
		s.sendText(fmt.Sprintf("You are already following %s.", target.Name))
		return nil
	}

	// Stop following previous leader (act.movement.c line 924–925 stop_follower)
	if s.player.Following != "" {
		oldLeader := s.player.Following
		if leader, ok := s.manager.world.GetPlayer(oldLeader); ok {
			leader.SendMessage(fmt.Sprintf("%s stops following you.\r\n", s.player.Name))
		}
	}

	// REMOVE_BIT AFF_GROUP — act.movement.c line 926 (leaving old group when changing leader)
	s.player.Following = target.Name
	s.player.InGroup = false

	// add_follower() — act.movement.c line 948
	s.sendText(fmt.Sprintf("You now follow %s.", target.Name))
	target.SendMessage(fmt.Sprintf("%s now follows you.\r\n", s.player.Name))
	return nil
}

// cmdGroup adds/removes players from a group, or prints group status.
// Source: act.other.c do_group() lines 685–740 and perform_group() lines 624–635
func cmdGroup(s *Session, args []string) error {
	// No args: print group — act.other.c do_group() line 693
	if len(args) == 0 {
		return printGroup(s)
	}

	// Must have no master to enroll others — act.other.c line 699
	if s.player.Following != "" {
		s.sendText("You cannot enroll group members without being head of a group.")
		return nil
	}

	targetName := strings.Join(args, " ")

	// "group all" — act.other.c lines 706–717
	if strings.EqualFold(targetName, "all") {
		s.player.InGroup = true
		found := 0
		for _, f := range s.manager.world.GetFollowersInRoom(s.player.Name, s.player.GetRoom()) {
			if !f.InGroup {
				f.InGroup = true
				s.sendText(fmt.Sprintf("%s is now a member of your group.", f.Name))
				f.SendMessage(fmt.Sprintf("You are now a member of %s's group.\r\n", s.player.Name))
				found++
			}
		}
		if found == 0 {
			s.sendText("Everyone following you here is already in your group.")
		}
		return nil
	}

	// Single target — act.other.c lines 719–738
	target, ok := s.manager.world.GetPlayer(targetName)
	if !ok {
		s.sendText("There is no one by that name here.")
		return nil
	}

	// Target must be following us — act.other.c line 721: vict->master != ch
	// Agent exception: agents auto-follow and auto-accept the invite.
	if target.Following != s.player.Name {
		targetSess, hasSess := s.manager.GetSession(target.Name)
		if hasSess && targetSess.isAgent {
			// Agent auto-follow — mirrors BRENDA accepting an invite
			target.Following = s.player.Name
			target.InGroup = false
			target.SendMessage(fmt.Sprintf("You start following %s.\r\n", s.player.Name))
			s.sendText(fmt.Sprintf("%s starts following you.", target.Name))
		} else {
			s.sendText(fmt.Sprintf("%s must follow you to enter your group.", target.Name))
			return nil
		}
	}

	// Toggle membership — perform_group() / kick-out path (act.other.c lines 726–738)
	if !target.InGroup {
		// perform_group(): SET_BIT AFF_GROUP
		target.InGroup = true
		s.player.InGroup = true // leader is also in the group
		if target.Name != s.player.Name {
			s.sendText(fmt.Sprintf("%s is now a member of your group.", target.Name))
		}
		target.SendMessage(fmt.Sprintf("You are now a member of %s's group.\r\n", s.player.Name))
	} else {
		// Kick out — REMOVE_BIT AFF_GROUP (act.other.c line 737)
		target.InGroup = false
		s.sendText(fmt.Sprintf("%s is no longer a member of your group.", target.Name))
		target.SendMessage(fmt.Sprintf("You have been kicked out of %s's group!\r\n", s.player.Name))
	}
	return nil
}

// printGroup displays the current group composition.
// Source: act.other.c print_group() lines 638–681
func printGroup(s *Session) error {
	if !s.player.InGroup {
		s.sendText("But you are not the member of a group!")
		return nil
	}

	leaderName := s.player.Name
	if s.player.Following != "" {
		leaderName = s.player.Following
	}

	leader, ok := s.manager.world.GetPlayer(leaderName)
	if !ok {
		s.sendText("Your group leader is not online.")
		return nil
	}

	var sb strings.Builder
	sb.WriteString("Your group consists of:\r\n")
	if leader.InGroup {
		sb.WriteString(fmt.Sprintf("     [%3dH %3dM] [%2d] %s (Head of group)\r\n",
			leader.Health, leader.Mana, leader.Level, leader.Name))
	}
	for _, m := range s.manager.world.GetGroupMembers(leaderName) {
		if m.Name == leaderName {
			continue // already printed above
		}
		sb.WriteString(fmt.Sprintf("     [%3dH %3dM] [%2d] %s\r\n",
			m.Health, m.Mana, m.Level, m.Name))
	}
	s.sendText(sb.String())
	return nil
}

// cmdUngroup removes a player from the group or disbands the entire group.
// Source: act.other.c do_ungroup() lines 744–794
func cmdUngroup(s *Session, args []string) error {
	// No args: disband if leader — act.other.c lines 752–770
	if len(args) == 0 {
		if s.player.Following != "" || !s.player.InGroup {
			s.sendText("But you lead no group!")
			return nil
		}
		disbandMsg := fmt.Sprintf("%s has disbanded the group.\r\n", s.player.Name)
		for _, m := range s.manager.world.GetGroupMembers(s.player.Name) {
			if m.Name == s.player.Name {
				continue
			}
			m.InGroup = false
			m.Following = "" // stop_follower — act.other.c line 764
			m.SendMessage(disbandMsg)
		}
		s.player.InGroup = false
		s.sendText("You disband the group.")
		return nil
	}

	// Remove specific member — act.other.c lines 772–793
	targetName := strings.Join(args, " ")
	target, ok := s.manager.world.GetPlayer(targetName)
	if !ok {
		s.sendText("There is no such person!")
		return nil
	}
	if target.Following != s.player.Name {
		s.sendText("That person is not following you!")
		return nil
	}
	if !target.InGroup {
		s.sendText("That person isn't in your group.")
		return nil
	}

	target.InGroup = false
	target.Following = "" // stop_follower — act.other.c line 793
	s.sendText(fmt.Sprintf("%s is no longer a member of your group.", target.Name))
	target.SendMessage(fmt.Sprintf("You have been kicked out of %s's group!\r\n", s.player.Name))
	return nil
}

// cmdGtell sends a message to all group members.
// Source: act.comm.c do_gsay() lines 824–870 (registered as "gtell" in interpreter.c line 484)
func cmdGtell(s *Session, args []string) error {
	if !s.player.InGroup {
		s.sendText("But you are not the member of a group!")
		return nil
	}
	if len(args) == 0 {
		s.sendText("Yes, but WHAT do you want to group-say?")
		return nil
	}

	text := strings.Join(args, " ")
	broadcastMsg := fmt.Sprintf("%s tells the group, '%s'\r\n", s.player.Name, text)

	// Find leader — act.comm.c do_gsay() line 838–841
	leaderName := s.player.Name
	if s.player.Following != "" {
		leaderName = s.player.Following
	}

	// Send to leader if not self (act.comm.c lines 846–851)
	if leaderName != s.player.Name {
		if leader, ok := s.manager.world.GetPlayer(leaderName); ok && leader.InGroup {
			leader.SendMessage(broadcastMsg)
		}
	}

	// Send to all group followers excluding self (act.comm.c lines 852–858)
	for _, f := range s.manager.world.GetFollowers(leaderName) {
		if f.InGroup && f.Name != s.player.Name {
			f.SendMessage(broadcastMsg)
		}
	}

	// Confirm to sender — act.comm.c line 862–865
	s.sendText(fmt.Sprintf("You tell the group, '%s'", text))
	return nil
}

// sendText sends a simple text message to the player.
func (s *Session) sendText(text string) {
	msg, err := json.Marshal(ServerMessage{
		Type: MsgText,
		Data: TextData{Text: text},
	})
	if err != nil {
		slog.Error("json.Marshal error", "error", err)
		return
	}
	select {
	case s.send <- msg:
	default:
	}
}

// cmdScore shows the player's stats.
// Source: act.informative.c do_score() lines 1168-1451
