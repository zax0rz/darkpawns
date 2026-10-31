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
// Source: act.other.c do_group() lines 685-740 and perform_group() lines 624-635
//
// The body lives in pkg/game so that a mobile forced or ordered to run `group`
// reaches the same code C reaches through command_interpreter (F9). Only the
// empty-argument listing is session-owned, because it renders through the
// viewer's own report path.
func cmdGroup(s *Session, args []string) error {
	if !s.manager.world.GroupCommand(s.player, strings.Join(args, " ")) {
		return printGroup(s)
	}
	return nil
}

// printGroup displays the current group composition.
// Source: act.other.c print_group() lines 638–681
func printGroup(s *Session) error {
	if !game.GroupedActor(s.player) {
		s.player.SendMessage("But you are not the member of a group!\r\n")
		return nil
	}

	// C: k = (ch->master ? ch->master : ch) — the head is whoever the viewer
	// follows, and the viewer otherwise. The followed body is resolved by
	// identity, never by name: two NPCs can share a name, and a name lookup
	// (GetMobByName matches a substring over a map-ordered slice) could pick a
	// different duplicate from run to run (R3). Nothing in C requires ch->master
	// to be a player either — do_follow reaches mobiles through get_char_room_vis
	// with no IS_NPC gate (src/act.movement.c:895) and perform_group sets
	// AFF_GROUP with no gate (src/act.other.c:624-635) — so the head may be a
	// mobile, and a viewer following a name that no longer resolves falls back to
	// itself, which is C's null-master arm rather than an invented message.
	leader := s.manager.world.FollowingActor(s.player)
	if leader == nil {
		leader = s.player
	}

	s.player.SendMessage("Your group consists of:\r\n")
	if game.GroupedActor(leader) {
		sendGroupMemberLine(s, leader, true)
	}
	// C walks k->followers, so the list belongs to the leader *body*: two
	// same-named NPCs keep separate follower lists and a name match would merge
	// them (R3).
	for _, follower := range s.manager.world.FollowersOf(leader) {
		if game.GroupedActor(follower) {
			sendGroupMemberLine(s, follower, false)
		}
	}
	return nil
}

func sendGroupMemberLine(s *Session, member game.Actor, head bool) {
	// C print_group wraps each stat at C_CMP against the *viewer* (src/act.other.c:653-676),
	// closing every field with CCNRM before the next, so the H/M/V letters sit
	// after the reset. The class abbreviation is the one asymmetry: the head row
	// uses CCCYN (line 659) and a follower's row uses CCGRN (line 676).
	complete := hasCompleteColor(s.player.GetFlags())
	classColor := ansiGreen
	if head {
		classColor = ansiCyan
	}

	var hp, mana, move, level int
	class := "??"
	switch actor := member.(type) {
	case *game.Player:
		hp, mana, move, level = actor.GetHealth(), actor.GetMana(), actor.GetMove(), actor.GetLevel()
		if n := actor.GetClass(); n >= 0 && n < len(game.ClassAbbrevs) {
			class = game.ClassAbbrevs[n]
		}
	case *game.MobInstance:
		if !head {
			// C prints the dashes row for a grouped mobile *follower*
			// (src/act.other.c:668-669): checking out mob stats via group is bad.
			game.Act(s.manager.world, false, s.player, actor, nil, nil,
				"     [---H ---M ---V] [-- --] $N", "", game.ToChar)
			return
		}
		// The head arm has no IS_NPC check, so a mobile leader reports its real
		// stats; CLASS_ABBR is the literal "--" for an NPC (src/utils.h:570).
		hp, mana, move, level = actor.GetHealth(), actor.GetMana(), actor.GetMove(), actor.GetLevel()
		class = "--"
	default:
		return
	}

	format := fmt.Sprintf("     [%sH %sM %sV] [%2d %s] $N",
		colorWrap(fmt.Sprintf("%3d", hp), ansiGreen, complete),
		colorWrap(fmt.Sprintf("%3d", mana), ansiGreen, complete),
		colorWrap(fmt.Sprintf("%3d", move), ansiGreen, complete),
		level,
		colorWrap(class, classColor, complete))
	if head {
		format += " (Head of group)"
	}
	game.Act(s.manager.world, false, s.player, member, nil, nil, format, "", game.ToChar)
}

// cmdUngroup removes a player from the group or disbands the entire group.
// Source: act.other.c do_ungroup() lines 744–794
func cmdUngroup(s *Session, args []string) error {
	argument, _ := game.OneArgument(strings.Join(args, " "))

	// No args: disband if leader — act.other.c lines 752–770
	if argument == "" {
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
			m.SetFollowing("") // stop_follower — act.other.c line 764
			m.SendMessage(disbandMsg)
		}
		s.player.InGroup = false
		s.sendText("You disband the group.")
		return nil
	}

	// Remove specific member — act.other.c lines 772–793. C's one_argument
	// selects one visible room token and ignores the remainder.
	targetName := argument
	target, ok := s.manager.world.ResolveCharInRoom(s.player, targetName)
	if !ok {
		s.sendText("There is no such person!")
		return nil
	}
	victim := game.AsActor(target.Combatant)
	if victim == nil {
		s.sendText("There is no such person!")
		return nil
	}
	if !s.manager.world.FollowsBody(victim, s.player) {
		s.sendText("That person is not following you!")
		return nil
	}
	if !game.GroupedActor(victim) {
		s.sendText("That person isn't in your group.")
		return nil
	}

	game.SetGroupedActor(victim, false)
	game.Act(nil, false, s.player, victim, nil, nil,
		"$N is no longer a member of your group.", "", game.ToChar)
	game.Act(nil, false, s.player, victim, nil, nil,
		"You have been kicked out of $n's group!", "", game.ToVict)
	game.Act(s.manager.world, false, s.player, victim, nil, nil,
		"$N has been kicked out of $n's group!", "", game.ToNotVict)
	if mob, ok := victim.(*game.MobInstance); ok {
		game.StopFollowerMob(s.manager.world, mob)
	} else if player, ok := victim.(*game.Player); ok {
		game.StopFollower(s.manager.world, player)
	}
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
	if !s.player.IsAffected(game.AffGroup) || text == "" {
		s.manager.world.GroupSay(s.player, text)
		return nil
	}

	// Word filter + spam check
	filtered, block := filterCommMessage(s, text)
	if block {
		s.sendText("Your message was blocked.")
		return nil
	}
	s.manager.world.GroupSay(s.player, filtered)
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
	if s.stageHeartbeat(msg, text, false) {
		return
	}
	s.forwardSnoopOutput(text)
	// C's process_output frames the next prompt whenever output was flushed
	// (comm.c:1631); a sendText-only response owed a frame it never got
	// (PF-063 cluster 5: do_ungroup's reply lost its blank line).
	s.notePlayerOutput()
	select {
	case s.send <- msg:
	default:
		slog.Warn("session sendText channel full — dropping message", "player", s.playerName)
	}
}

// cmdScore shows the player's stats.
// Source: act.informative.c do_score() lines 1168-1451
