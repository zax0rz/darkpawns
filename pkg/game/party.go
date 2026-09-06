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
	// The Go runtime does not retain C's linked-list insertion order. Use a
	// stable order so group messages and listings remain deterministic.
	sort.SliceStable(followers, func(i, j int) bool {
		return strings.ToLower(followers[i].GetName()) < strings.ToLower(followers[j].GetName())
	})
	return followers
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

// AwardMobKillXP distributes experience to the killer and all grouped members in the same room.
// Solo kills use the C calc_level_diff() path with two levels of slack before
// higher-level penalties apply.
// If victimGold > 0 and the killer has AutoGold enabled, gold is looted/split
// according to the autosplit preference.
// Source: fight.c group_gain() lines 708–745, called at die_with_killer() line 1638
func (w *World) AwardMobKillXP(killer combat.Combatant, victimExp int, victimGold int, victimLevel int, victimAlign int) {
	killerName := killer.GetName()
	killerRoom := killer.GetRoom()

	// Look up the killer as a Player for preference flags
	kp, isPlayer := w.GetPlayer(killerName)

	// --- Gold handling (fight.c group_gain() lines 747+) ---
	// Source: fight.c lines 747-830
	if isPlayer && victimGold > 0 && kp.AutoGold {
		// Base gold for distribution — fight.c: gold_looted = GET_GOLD(victim); GET_GOLD(victim) = 0
		goldLooted := victimGold

		if kp.AutoSplit {
			// Autosplit path — fight.c lines 756-830
			// Announce the loot to killer
			kp.SendMessage(fmt.Sprintf("You loot %d coins from the corpse of %s.\r\n",
				goldLooted, killerName))

			// Get group members in the room
			members := w.GetGroupMembers(killerName)
			var inRoom []*Player
			for _, m := range members {
				if m.GetRoom() == killerRoom {
					inRoom = append(inRoom, m)
				}
			}
			totMembers := len(inRoom)

			// If we have group members, distribute
			if totMembers > 1 {
				goldPerMember := goldLooted / totMembers
				remaining := goldLooted

				for _, m := range inRoom {
					if m == kp {
						continue // handle leader last
					}
					if goldPerMember > 0 {
						m.SendMessage(fmt.Sprintf("%s splits some gold with you, you get %d.\r\n",
							kp.Name, goldPerMember))
						kp.SendMessage(fmt.Sprintf("You share %d gold with %s.\r\n",
							goldPerMember, m.Name))
						// Lock across GetGold+SetGold to prevent read-modify-write race (DP-347)
						m.mu.Lock()
						m.Gold += goldPerMember
						m.mu.Unlock()
						remaining -= goldPerMember
					} else {
						kp.SendMessage(fmt.Sprintf("You would share gold with %s, but there was none to split!\r\n",
							m.Name))
						m.SendMessage(fmt.Sprintf("%s would have shared some gold with you but there was none to split!\r\n",
							kp.Name))
					}
				}

				// Killer keeps the remainder
				if remaining > 0 {
					kp.SendMessage(fmt.Sprintf("You split the gold and keep %d for yourself.\r\n", remaining))
					// Lock across GetGold+SetGold to prevent read-modify-write race (DP-347)
					kp.mu.Lock()
					kp.Gold += remaining
					kp.mu.Unlock()
				} else {
					kp.SendMessage("When you split no gold, you got none.\r\n")
				}
			} else {
				// Solo kill with autogold+autosplit but no group — just take all
				// Lock across GetGold+SetGold to prevent read-modify-write race (DP-347)
				kp.mu.Lock()
				kp.Gold += goldLooted
				kp.mu.Unlock()
				kp.SendMessage(fmt.Sprintf("You loot %d gold from the corpse.\r\n", goldLooted))
			}
		} else {
			// AutoGold without AutoSplit — just loot
			// Lock across GetGold+SetGold to prevent read-modify-write race (DP-347)
			kp.mu.Lock()
			kp.Gold += goldLooted
			kp.mu.Unlock()
			kp.SendMessage(fmt.Sprintf("You loot %d gold from the corpse.\r\n", goldLooted))
		}
	}

	// --- Experience handling ---
	members := w.GetGroupMembers(killerName)

	// Solo kill (not in any group) — fight.c group_gain() totMembers==1 path
	if len(members) == 0 {
		p, ok := w.GetPlayer(killerName)
		if !ok {
			return
		}
		// Level-difference XP penalty — fight.c calc_level_diff(), solo path
		// gets the C two-level slack before higher-level penalties apply.
		xp := combat.CalcXPShare(killer.GetLevel(), victimLevel, victimExp, false, maxExpGain)
		w.GainExp(p, xp)
		if xp > 1 {
			p.SendMessage(fmt.Sprintf("You receive %d experience points.\r\n", xp))
		} else {
			p.SendMessage("You receive one lousy experience point.\r\n")
		}
		// Alignment shift for the killer — fight.c:1667 change_alignment(ch, victim).
		changeAlignment(p, victimAlign)
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
		base -= int(float64(base) * 0.01)
	}
	if base < 1 {
		base = 1
	}

	// perform_group_gain() for each member — fight.c lines 688–705
	for _, m := range inRoom {
		xp := combat.CalcXPShare(m.GetLevel(), victimLevel, base, true, maxExpGain)
		w.GainExp(m, xp)
		if xp > 1 {
			m.SendMessage(fmt.Sprintf("You receive your share of experience -- %d points.\r\n", xp))
		} else {
			m.SendMessage("You receive your share of experience -- one measly little point!\r\n")
		}
		// Alignment shift for every group member — fight.c:704 change_alignment(ch, victim).
		changeAlignment(m, victimAlign)
	}
}
