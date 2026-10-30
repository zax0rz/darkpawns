package game

// party.go — group/follow world-level management and XP distribution.
//
// Original sources:
//   act.other.c  — do_group(), do_ungroup(), perform_group(), print_group()
//   act.movement.c — do_follow(), add_follower(), stop_follower()
//   fight.c      — group_gain(), perform_group_gain(), called from die_with_killer() line 1638

import (
	"fmt"
	"sort"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// GetFollowers returns all players currently following leaderName, in any room.
// Source: act.other.c do_group() iterates ch->followers linked list.
func (w *World) GetFollowers(leaderName string) []*Player {
	w.mu.RLock()
	defer w.mu.RUnlock()

	var followers []*Player
	for _, p := range w.players {
		if p.GetFollowing() == leaderName {
			followers = append(followers, p)
		}
	}
	sort.SliceStable(followers, func(i, j int) bool {
		if followers[i].GetFollowingSequence() != followers[j].GetFollowingSequence() {
			return followers[i].GetFollowingSequence() > followers[j].GetFollowingSequence()
		}
		return strings.ToLower(followers[i].GetName()) < strings.ToLower(followers[j].GetName())
	})
	return followers
}

// GetFollowersInRoom returns all players following leaderName who are in roomVNum.
// Source: act.other.c do_group() line 710 — f->follower->in_room == ch->in_room
func (w *World) GetFollowersInRoom(leaderName string, roomVNum int) []*Player {
	w.mu.RLock()
	defer w.mu.RUnlock()

	var followers []*Player
	for _, p := range w.players {
		if p.GetFollowing() == leaderName && p.GetRoom() == roomVNum {
			followers = append(followers, p)
		}
	}
	return followers
}

// GetFollowerActors returns every character following leaderName, including
// charmed mobs. The C do_group path walks ch->followers, not just the player
// table; keep that distinction at the world boundary for group commands.
//
// Prefer FollowersOf when the leader's body is known: this name form cannot tell
// two same-named NPCs apart (R3).
func (w *World) GetFollowerActors(leaderName string) []Actor {
	var followers []Actor
	for _, player := range w.GetAllPlayers() {
		if strings.EqualFold(player.GetFollowing(), leaderName) {
			followers = append(followers, player)
		}
	}
	for _, mob := range w.GetAllMobs() {
		if strings.EqualFold(mob.GetFollowing(), leaderName) {
			followers = append(followers, mob)
		}
	}
	sortFollowerOrder(followers)
	return followers
}

// FollowersOf returns every character following the given leader body, in C's
// add_follower prepend order. The match is by body identity, not name: C walks
// k->followers, so two same-named NPCs keep separate lists and a name match would
// merge them (R3).
func (w *World) FollowersOf(leader Actor) []Actor {
	if leader == nil {
		return nil
	}
	var followers []Actor
	for _, player := range w.GetAllPlayers() {
		if w.FollowingActor(player) == leader {
			followers = append(followers, player)
		}
	}
	for _, mob := range w.GetAllMobs() {
		if w.FollowingActor(mob) == leader {
			followers = append(followers, mob)
		}
	}
	sortFollowerOrder(followers)
	return followers
}

// sortFollowerOrder puts a follower slice into C's list order: C prepends each
// new follower to the leader's linked list (utils.c:add_follower), and
// SetFollowing records that insertion sequence so this traversal matches.
func sortFollowerOrder(followers []Actor) {
	sort.SliceStable(followers, func(i, j int) bool {
		left, right := actorFollowingSequence(followers[i]), actorFollowingSequence(followers[j])
		if left != right {
			return left > right
		}
		return strings.ToLower(followers[i].GetName()) < strings.ToLower(followers[j].GetName())
	})
}

func actorFollowingSequence(actor Actor) uint64 {
	switch follower := actor.(type) {
	case *Player:
		return follower.GetFollowingSequence()
	case *MobInstance:
		return follower.GetFollowingSequence()
	default:
		return 0
	}
}

// GetFollowerActorsInRoom is the room-scoped form used by do_group "all".
func (w *World) GetFollowerActorsInRoom(leaderName string, roomVNum int) []Actor {
	followers := w.GetFollowerActors(leaderName)
	filtered := followers[:0]
	for _, follower := range followers {
		if follower.GetRoom() == roomVNum {
			filtered = append(filtered, follower)
		}
	}
	return filtered
}

// GetGroupMembers returns all players in the same group as playerName (InGroup==true),
// including the leader. Order is leader first, then followers.
// Source: fight.c group_gain() lines 716–727 — k = ch->master ?? ch, then iterate k->followers
func (w *World) GetGroupMembers(playerName string) []*Player {
	w.mu.RLock()
	defer w.mu.RUnlock()

	p, ok := w.players[playerName]
	if !ok {
		return nil
	}

	// Find the leader (ch->master ?? ch)
	leaderName := p.Name
	if p.GetFollowing() != "" {
		leaderName = p.GetFollowing()
	}
	leader, ok := w.players[leaderName]
	if !ok {
		return nil
	}

	var members []*Player
	if leader.InGroup {
		members = append(members, leader)
	}
	for _, follower := range w.players {
		if follower.GetFollowing() == leaderName && follower.InGroup {
			members = append(members, follower)
		}
	}
	return members
}

// combatIsInGroup mirrors fight.c:is_in_group() (src/fight.c:637-657) for any
// character body, player or mobile. A follow relation alone is not enough: the
// actor itself must carry AFF_GROUP, and another grouped member of that
// leader's group — the master, or any fellow follower in the C follower list —
// must be in the actor's room. An ungrouped killer following a grouped leader
// therefore stays on the solo path (retained #2215 regression).
func (w *World) combatIsInGroup(body combat.Combatant) bool {
	if !combatGroupedBody(body) {
		return false
	}

	room := body.GetRoom()
	leader := body
	if master := w.combatFollowingBody(body); master != nil {
		leader = master
		if combatGroupedBody(leader) && leader.GetRoom() == room {
			return true
		}
	}

	for _, follower := range w.combatFollowersInRoom(leader, room) {
		if follower != body && combatGroupedBody(follower) {
			return true
		}
	}
	return false
}

// AwardMobKillXP distributes experience to the killer and all grouped members in the same room.
// Solo kills use the C calc_level_diff() path with two levels of slack before
// higher-level penalties apply.
// If victimGold > 0 and the killer has AutoGold enabled, gold is looted/split
// according to the autosplit preference.
// Source: fight.c group_gain() lines 708–848, called at die_with_killer() line 1638
func (w *World) AwardMobKillXP(killer combat.Combatant, victim Actor, victimExp int, victimGold int, victimLevel int, victimAlign int) {
	killerRoom := killer.GetRoom()
	victimName := ""
	if victim != nil {
		victimName = victim.GetName()
	}

	// Look up the killer as a Player for preference flags. C reads the
	// killer's own PRF bits (src/fight.c:747, 1654); a mobile killer has none,
	// so it loots nothing and its group's gold stays on the corpse.
	kp, isPlayer := combatPlayer(killer)

	// awardGroupGold ports the autogold/autosplit tail of group_gain()
	// (src/fight.c:747-846). The divisor and every gold transfer include the
	// in-room group members C counts — mobiles included.
	awardGroupGold := func(inRoom []combat.Combatant, totMembers int, leader combat.Combatant) {
		if !isPlayer || victimGold <= 0 || !kp.AutoGold {
			return
		}
		goldLooted := victimGold

		if kp.AutoSplit {
			// Autosplit path — fight.c lines 756-830
			// Announce the loot to killer
			kp.SendMessage(fmt.Sprintf("You loot %d coins from the corpse of %s.\r\n",
				goldLooted, victimName))
			if victim != nil {
				Act(w, false, kp, nil, nil, nil,
					fmt.Sprintf("$n loots some gold from the corpse of %s.", victimName), "", ToRoom)
			}

			// If we have group members, distribute
			if totMembers > 1 {
				goldPerMember := goldLooted / totMembers

				// The leader share has no AFF_GROUP check in C: k receives
				// gold_per_member whenever k != ch and shares the room
				// (src/fight.c:768-781), even though tot_members did not
				// count an ungrouped leader and the XP loop skips them.
				// `group <follower>` leaves the leader ungrouped
				// (src/act.other.c:726-727), so this is the ordinary case.
				if leader != killer && goldPerMember > 0 && leader.GetRoom() == killerRoom {
					combatSendMessage(leader, fmt.Sprintf("%s splits some gold with you, you get %d.\r\n",
						kp.Name, goldPerMember))
					kp.SendMessage(fmt.Sprintf("You share %d gold with %s.\r\n",
						goldPerMember, leader.GetName()))
					combatAddGold(leader, goldPerMember)
					goldLooted -= goldPerMember
				}

				for _, m := range inRoom {
					if m == killer || m == leader {
						continue // killer's share below (fight.c:814-823); the leader was paid above
					}
					if goldPerMember > 0 {
						combatSendMessage(m, fmt.Sprintf("%s splits some gold with you, you get %d.\r\n",
							kp.Name, goldPerMember))
						kp.SendMessage(fmt.Sprintf("You share %d gold with %s.\r\n",
							goldPerMember, m.GetName()))
						combatAddGold(m, goldPerMember)
						goldLooted -= goldPerMember
					} else {
						// The none-to-split notice is a follower-loop arm
						// (fight.c:801-811); the leader block has no else.
						kp.SendMessage(fmt.Sprintf("You would share gold with %s, but there was none to split!\r\n",
							m.GetName()))
						combatSendMessage(m, fmt.Sprintf("%s would have shared some gold with you but there was none to split!\r\n",
							kp.Name))
					}
				}

				if goldPerMember > 0 {
					kp.SendMessage(fmt.Sprintf("You split the gold and keep %d for yourself.\r\n", goldPerMember))
					combatAddGold(kp, goldPerMember)
					goldLooted -= goldPerMember
				} else {
					kp.SendMessage("When you split no gold, you got none.\r\n")
				}
				if goldLooted > 0 {
					kp.SendMessage(fmt.Sprintf("You keep %d gold leftover.\r\n", goldLooted))
					combatAddGold(kp, goldLooted)
				}
			}
		} else {
			// AutoGold without AutoSplit — fight.c:836-845 also notifies the room.
			kp.SendMessage(fmt.Sprintf("You loot %d coins from the corpse of %s.\r\n",
				goldLooted, victimName))
			if victim != nil {
				Act(w, false, kp, nil, nil, nil,
					fmt.Sprintf("$n loots some gold from the corpse of %s.", victimName), "", ToRoom)
			}
			combatAddGold(kp, goldLooted)
		}
	}

	// --- Experience handling ---
	grouped := w.combatIsInGroup(killer)

	// Solo kill (not in any group) — fight.c damage() lines 1639-1667. A
	// mobile killer on this path produces nothing observable: C's
	// send_to_char no-ops, gain_exp skips NPCs, its PRF bits are clear, and
	// change_alignment returns for NPCs (src/fight.c:486-489).
	if !grouped {
		p, ok := combatPlayer(killer)
		if !ok {
			return
		}
		// Level-difference XP penalty — fight.c calc_level_diff(), solo path
		// gets the C two-level slack before higher-level penalties apply.
		xp := combat.CalcXPShare(killer.GetLevel(), victimLevel, victimExp, false, maxExpGain)
		if xp > 1 {
			p.SendMessage(fmt.Sprintf("You receive %d experience points.\r\n", xp))
		} else {
			p.SendMessage("You receive one lousy experience point.\r\n")
		}
		w.GainExp(p, xp)
		// Solo damage(): XP, then PRF_AUTOGOLD (src/fight.c:1644-1667).
		if victimGold != 0 && p.AutoGold {
			p.SendMessage(fmt.Sprintf("You loot %d gold from the corpse.\r\n", victimGold))
			p.mu.Lock()
			p.Gold += victimGold
			p.mu.Unlock()
		}
		// Alignment shift for the killer — fight.c:1667 change_alignment(ch, victim).
		changeAlignment(p, victimAlign)
		return
	}

	// The leader group_gain() splits around (src/fight.c:716-717), including
	// mobiles in both the divisor and the recipient list.
	leader := killer
	if master := w.combatFollowingBody(killer); master != nil {
		leader = master
	}
	members := w.combatGroupMembers(killer)

	// Count members in killer's room — fight.c group_gain() lines 719–727
	var inRoom []combat.Combatant
	for _, m := range members {
		if m.GetRoom() == killerRoom {
			inRoom = append(inRoom, m)
		}
	}
	totMembers := len(inRoom)
	if totMembers == 0 {
		totMembers = 1
	}

	// Per-member share — fight.c group_gain() lines 730–736
	// base = GET_EXP(victim) / tot_members
	// if base > 100: base -= base*.01   (1% group penalty)
	// base = MAX(1, base)
	base := victimExp / totMembers
	if base > 100 {
		// C subtracts in floating point and truncates once on assignment:
		// base = (int)(base - base*0.01), so 150 becomes 148, not 149
		// (src/fight.c:731-732).
		base = int(float64(base) - float64(base)*0.01)
	}
	if base < 1 {
		base = 1
	}

	// perform_group_gain() for each member — fight.c lines 688-705 and
	// 739-745. Mobile members receive the C no-op message send, skip
	// gain_exp, and skip change_alignment (which itself returns for NPCs,
	// src/fight.c:486-489). Every counted member shares a room with another
	// grouped member of the same leader, so is_in_group(member) is true for
	// each and calc_level_diff applies its group slack.
	for _, m := range inRoom {
		share := combat.CalcXPShare(m.GetLevel(), victimLevel, base, true, maxExpGain)
		if share > 1 {
			combatSendMessage(m, fmt.Sprintf("You receive your share of experience -- %d points.\r\n", share))
		} else {
			combatSendMessage(m, "You receive your share of experience -- one measly little point!\r\n")
		}
		if p, ok := m.(*Player); ok {
			// C perform_group_gain emits the share before gain_exp can announce a
			// level transition (src/fight.c:688-705).
			w.GainExp(p, share)
			changeAlignment(p, victimAlign)
		}
	}
	awardGroupGold(inRoom, totMembers, leader) // src/fight.c:747-846: XP shares precede autogold.
}

// combatSendMessage delivers a C send_to_char to any character body. Mobiles
// have no descriptor, so their copy is the C no-op.
func combatSendMessage(body combat.Combatant, msg string) {
	switch b := body.(type) {
	case *Player:
		b.SendMessage(msg)
	case *MobInstance:
		b.SendMessage(msg)
	}
}

// combatAddGold applies a C GET_GOLD(ch) += n to any character body; mobile
// group members really receive their autosplit share (src/fight.c:779, 798).
func combatAddGold(body combat.Combatant, amount int) {
	switch b := body.(type) {
	case *Player:
		// Lock across GetGold+SetGold to prevent read-modify-write race (DP-347)
		b.mu.Lock()
		b.Gold += amount
		b.mu.Unlock()
	case *MobInstance:
		b.updateGold(func(gold int) (int, bool) { return gold + amount, true })
	}
}
