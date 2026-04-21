package game

// party.go — group/follow world-level management and XP distribution.
//
// Original sources:
//   act.other.c  — do_group(), do_ungroup(), perform_group(), print_group()
//   act.movement.c — do_follow(), add_follower(), stop_follower()
//   fight.c      — group_gain(), perform_group_gain(), called from die_with_killer() line 1638

import (
	"fmt"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// GetFollowers returns all players currently following leaderName, in any room.
// Source: act.other.c do_group() iterates ch->followers linked list.
func (w *World) GetFollowers(leaderName string) []*Player {
	w.mu.RLock()
	defer w.mu.RUnlock()

	var followers []*Player
	for _, p := range w.players {
		if p.Following == leaderName {
			followers = append(followers, p)
		}
	}
	return followers
}

// GetFollowersInRoom returns all players following leaderName who are in roomVNum.
// Source: act.other.c do_group() line 710 — f->follower->in_room == ch->in_room
func (w *World) GetFollowersInRoom(leaderName string, roomVNum int) []*Player {
	w.mu.RLock()
	defer w.mu.RUnlock()

	var followers []*Player
	for _, p := range w.players {
		if p.Following == leaderName && p.RoomVNum == roomVNum {
			followers = append(followers, p)
		}
	}
	return followers
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
	if p.Following != "" {
		leaderName = p.Following
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
		if follower.Following == leaderName && follower.InGroup {
			members = append(members, follower)
		}
	}
	return members
}

// AwardMobKillXP distributes experience to the killer and all grouped members in the same room.
// For solo kills (no group), the killer gets the full victimExp.
// Source: fight.c group_gain() lines 708–745, called at die_with_killer() line 1638
func (w *World) AwardMobKillXP(killer combat.Combatant, victimExp int) {
	if victimExp <= 0 {
		return
	}

	killerName := killer.GetName()
	killerRoom := killer.GetRoom()

	members := w.GetGroupMembers(killerName)

	// Solo kill (not in any group) — fight.c group_gain() totMembers==1 path
	if len(members) == 0 {
		p, ok := w.GetPlayer(killerName)
		if !ok {
			return
		}
		p.mu.Lock()
		p.Exp += victimExp
		p.mu.Unlock()
		if victimExp > 1 {
			p.SendMessage(fmt.Sprintf("You receive %d experience points.\r\n", victimExp))
		} else {
			p.SendMessage("You receive one measly little experience point!\r\n")
		}
		return
	}

	// Count members in killer's room — fight.c group_gain() lines 719–727
	var inRoom []*Player
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
		base -= base / 100
	}
	if base < 1 {
		base = 1
	}

	// perform_group_gain() for each member — fight.c lines 688–705
	for _, m := range inRoom {
		m.mu.Lock()
		m.Exp += base
		m.mu.Unlock()
		if base > 1 {
			m.SendMessage(fmt.Sprintf("You receive your share of experience -- %d points.\r\n", base))
		} else {
			m.SendMessage("You receive your share of experience -- one measly little point!\r\n")
		}
	}
}
