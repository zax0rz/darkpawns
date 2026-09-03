package session

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func cmdFollow(s *Session, args []string) error {
	s.manager.world.DoFollow(s.player, strings.Join(args, " "), false)
	return nil
}

// cmdShadow — "shadow <target>" quiet follow (act.movement.c do_follow/SCMD shadow,
// subcmd=TRUE). The quiet path suppresses the normal follow announcement; the deeper
// SKILL_SHADOW success roll + AFF_DODGE affect remains a game-layer TODO (DoFollow line 261).
func cmdShadow(s *Session, args []string) error {
	s.manager.world.DoFollow(s.player, strings.Join(args, " "), true)
	return nil
}

// cmdGroup adds/removes players from a group, or prints group status.
// Source: act.other.c do_group() lines 685–740 and perform_group() lines 624–635
func cmdGroup(s *Session, args []string) error {
	// do_group begins with one_argument(), which skips leading fill words and
	// discards the remainder. Reconstruct the raw argument text lost by the
	// command wrapper before applying that C parser.
	argument, _ := game.OneArgument(strings.Join(args, " "))
	if argument == "" {
		return printGroup(s)
	}

	// Must have no master to enroll others — act.other.c line 699
	if s.player.GetFollowing() != "" {
		game.Act(nil, false, s.player, nil, nil, nil,
			"You can not enroll group members without being head of a group.", "", game.ToChar)
		return nil
	}

	// "group all" — act.other.c lines 706–717
	if strings.EqualFold(argument, "all") {
		performGroup(s, s.player)
		found := 0
		for _, follower := range s.manager.world.GetFollowerActorsInRoom(s.player.Name, s.player.GetRoom()) {
			if isShadowing(follower) {
				continue
			}
			found += performGroup(s, follower)
		}
		if found == 0 {
			s.player.SendMessage("Everyone following you here is already in your group.\r\n")
		}
		return nil
	}

	// Single target — act.other.c lines 719–738
	target, ok := s.manager.world.ResolveCharInRoom(s.player, argument)
	if !ok {
		s.player.SendMessage("No-one by that name here.\r\n")
		return nil
	}
	victim := groupActor(target)
	if victim == nil {
		return nil
	}

	// Target must be following us — act.other.c line 721: vict->master != ch
	// Agent exception: agents auto-follow and auto-accept the invite.
	if !follows(victim, s.player) && victim != s.player {
		targetPlayer, isPlayer := victim.(*game.Player)
		targetSess, hasSess := s.manager.GetSession(victim.GetName())
		if isPlayer && hasSess && targetSess.isAgent {
			// Agent auto-follow — mirrors BRENDA accepting an invite
			targetPlayer.SetFollowing(s.player.Name)
			setGrouped(targetPlayer, false)
			targetPlayer.SendMessage(fmt.Sprintf("You start following %s.\r\n", s.player.Name))
			s.player.SendMessage(fmt.Sprintf("%s starts following you.\r\n", targetPlayer.Name))
		} else {
			game.Act(nil, false, s.player, victim, nil, nil,
				"$N must follow you to enter your group.", "", game.ToChar)
			return nil
		}
	}

	// Toggle membership — perform_group() / kick-out path (act.other.c lines 726–738)
	if !isGrouped(victim) {
		performGroup(s, victim)
	} else {
		// Kick out — REMOVE_BIT AFF_GROUP (act.other.c line 737)
		if victim != s.player {
			game.Act(nil, false, s.player, victim, nil, nil,
				"$N is no longer a member of your group.", "", game.ToChar)
		}
		game.Act(nil, false, s.player, victim, nil, nil,
			"You have been kicked out of $n's group!", "", game.ToVict)
		game.Act(s.manager.world, false, s.player, victim, nil, nil,
			"$N has been kicked out of $n's group!", "", game.ToNotVict)
		setGrouped(victim, false)
	}
	return nil
}

// groupActor extracts the concrete Actor needed by the canonical act() and
// visibility paths from ResolveCharInRoom's typed result.
func groupActor(target game.CharTarget) game.Actor {
	if target.Player != nil {
		return target.Player
	}
	if target.Mob != nil {
		return target.Mob
	}
	return nil
}

func follows(victim game.Actor, leader game.Actor) bool {
	if victim == nil || leader == nil {
		return false
	}
	var following string
	switch ch := victim.(type) {
	case *game.Player:
		following = ch.GetFollowing()
	case *game.MobInstance:
		following = ch.GetFollowing()
	default:
		return false
	}
	return strings.EqualFold(following, leader.GetName())
}

func isGrouped(victim game.Actor) bool {
	switch ch := victim.(type) {
	case *game.Player:
		return ch.IsAffected(game.AffGroup) || ch.IsInGroup()
	case *game.MobInstance:
		return ch.IsAffected(game.AffGroup)
	default:
		return false
	}
}

func setGrouped(victim game.Actor, grouped bool) {
	switch ch := victim.(type) {
	case *game.Player:
		ch.SetInGroup(grouped)
		ch.SetAffect(game.AffGroup, grouped)
	case *game.MobInstance:
		if grouped {
			ch.SetAffected(game.AffGroup)
		} else {
			ch.RemoveAffected(game.AffGroup)
		}
	}
}

func isShadowing(victim game.Actor) bool {
	player, ok := victim.(*game.Player)
	return ok && player.IsAffected(game.AffDodge)
}

// performGroup ports act.other.c:624-635. It returns one only when the
// victim newly receives AFF_GROUP, which is the C do_group "found" count.
func performGroup(s *Session, victim game.Actor) int {
	if victim == nil || isGrouped(victim) || !game.CanSee(s.player, victim) {
		return 0
	}
	setGrouped(victim, true)
	if victim != s.player {
		game.Act(nil, false, s.player, victim, nil, nil,
			"$N is now a member of your group.", "", game.ToChar)
	}
	game.Act(nil, false, s.player, victim, nil, nil,
		"You are now a member of $n's group.", "", game.ToVict)
	game.Act(s.manager.world, false, s.player, victim, nil, nil,
		"$N is now a member of $n's group.", "", game.ToNotVict)
	return 1
}

// printGroup displays the current group composition.
// Source: act.other.c print_group() lines 638–681
func printGroup(s *Session) error {
	if !isGrouped(s.player) {
		s.player.SendMessage("But you are not the member of a group!\r\n")
		return nil
	}

	leaderName := s.player.Name
	if s.player.GetFollowing() != "" {
		leaderName = s.player.GetFollowing()
	}

	leader, ok := s.manager.world.GetPlayer(leaderName)
	if !ok {
		s.player.SendMessage("Your group leader is not online.\r\n")
		return nil
	}

	s.player.SendMessage("Your group consists of:\r\n")
	if isGrouped(leader) {
		sendGroupMemberLine(s, leader, true)
	}
	for _, follower := range s.manager.world.GetFollowerActors(leaderName) {
		if isGrouped(follower) {
			sendGroupMemberLine(s, follower, false)
		}
	}
	return nil
}

func sendGroupMemberLine(s *Session, member game.Actor, head bool) {
	if mob, ok := member.(*game.MobInstance); ok {
		game.Act(s.manager.world, false, s.player, mob, nil, nil,
			"     [---H ---M ---V] [-- --] $N", "", game.ToChar)
		return
	}
	player, ok := member.(*game.Player)
	if !ok {
		return
	}
	class := "??"
	if n := player.GetClass(); n >= 0 && n < len(game.ClassAbbrevs) {
		class = game.ClassAbbrevs[n]
	}
	format := fmt.Sprintf("     [%3dH %3dM %3dV] [%2d %s] $N",
		player.GetHealth(), player.GetMana(), player.GetMove(), player.GetLevel(), class)
	if head {
		format += " (Head of group)"
	}
	game.Act(s.manager.world, false, s.player, player, nil, nil, format, "", game.ToChar)
}

// cmdUngroup removes a player from the group or disbands the entire group.
// Source: act.other.c do_ungroup() lines 744–794
func cmdUngroup(s *Session, args []string) error {
	// No args: disband if leader — act.other.c lines 752–770
	if len(args) == 0 {
		if s.player.GetFollowing() != "" || !s.player.InGroup {
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
	if target.GetFollowing() != s.player.Name {
		s.sendText("That person is not following you!")
		return nil
	}
	if !target.InGroup {
		s.sendText("That person isn't in your group.")
		return nil
	}

	target.InGroup = false
	target.SetFollowing("") // stop_follower — act.other.c line 793
	s.sendText(fmt.Sprintf("%s is no longer a member of your group.", target.Name))
	target.SendMessage(fmt.Sprintf("You have been kicked out of %s's group!\r\n", s.player.Name))
	return nil
}

// cmdGtell sends a message to all group members.
// Source: act.comm.c do_gsay() lines 824–870 (registered as "gtell" in interpreter.c line 484)
func cmdGtell(s *Session, args []string) error {
	return cmdGtellText(s, strings.Join(args, " "))
}

// cmdGtellText is the raw-argument form of do_gsay. C's skip_spaces preserves
// internal and trailing whitespace, and delete_ansi_controls runs over the
// complete formatted message before it is sent to any audience.
func cmdGtellText(s *Session, text string) error {
	if !s.player.InGroup {
		s.sendText("But you are not the member of a group!")
		return nil
	}
	if text == "" {
		s.sendText("Yes, but WHAT do you want to group-say?")
		return nil
	}

	// Word filter + spam check
	filtered, block := filterCommMessage(s, text)
	if block {
		s.sendText("Your message was blocked.")
		return nil
	}
	text = game.DeleteANSIControls(filtered)

	broadcastMsg := fmt.Sprintf("%s tells the group, '%s'\r\n", s.player.Name, text)

	// Find leader — act.comm.c do_gsay() line 838–841
	leaderName := s.player.Name
	if s.player.GetFollowing() != "" {
		leaderName = s.player.GetFollowing()
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

	// Confirm to sender — act.comm.c lines 862–865. The C OK macro is
	// "Okay.", not the shorter command-layer "Ok." used by other paths.
	if s.player.GetFlags()&(1<<uint(game.PrfNoRepeat)) != 0 {
		s.sendText("Okay.")
	} else {
		s.sendText(fmt.Sprintf("You tell the group, '%s'", text))
	}
	return nil
}

// sendText sends a simple text message to the player.
func (s *Session) sendText(text string) {
	s.forwardSnoopOutput(text)
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
		slog.Warn("session sendText channel full — dropping message", "player", s.playerName)
	}
}

// cmdScore shows the player's stats.
// Source: act.informative.c do_score() lines 1168-1451
