package game

import (
	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
)

type huntedItem struct {
	objectVNum int
	hunters    [3]int
	chance     int
}

// hunteds is the C table in src/new_cmds2.c:748-753, in table order.
var hunteds = [...]huntedItem{
	{objectVNum: 4209, hunters: [3]int{4207}, chance: 1},
	{objectVNum: 4210, hunters: [3]int{4207}, chance: 1},
	{objectVNum: 14223, hunters: [3]int{14203, 14205, 14207}, chance: 5},
}

// huntItemsNumber is a seam for proving CMWC draw order. Production uses the
// process-wide stream, just like number() in the C routine.
var huntItemsNumber = dprng.Number

// HuntItems scans the active descriptor bodies' equipment once per MUD hour.
// It preserves C's descriptor, wear-slot, table, and character_list order
// because every matching object consumes a draw and changes which hunter is
// available to later objects (src/new_cmds2.c:756-805).
func (w *World) HuntItems() {
	descriptors := w.huntItemsDescriptorBodies()
	for _, wearer := range descriptors {
		for _, item := range wornItems(wearer) {
			if item == nil {
				continue
			}
			for _, hunted := range hunteds {
				if item.GetVNum() != hunted.objectVNum {
					continue
				}
				// #nosec G404 — gameplay RNG, matching C number(0, 100).
				if huntItemsNumber(0, 100) >= hunted.chance {
					break
				}
				hunter := w.availableItemHunter(hunted.hunters)
				if hunter == nil {
					break
				}
				// C passes a message addressed to the hunter itself to do_tell
				// (new_cmds2.c:800-804). Its own NPC body has no descriptor, so
				// the tell is rejected silently; don't invent a player-facing line.
				w.npcTell(hunter, hunter.GetName()+" I must have "+item.GetShortDesc()+"!")
				w.SetHuntingTarget(hunter, wearer)
				break
			}
		}
	}
}

func (w *World) huntItemsDescriptorBodies() []Actor {
	if w.PlayingDescriptorBodies != nil {
		return w.PlayingDescriptorBodies()
	}
	// Game-layer fixtures have no session manager. Their player bodies stand
	// in for playing descriptors, in the same stable C character order.
	characters := w.cOrderedCharacters()
	bodies := make([]Actor, 0, len(characters))
	for _, character := range characters {
		if character.player != nil {
			bodies = append(bodies, character.player)
		}
	}
	return bodies
}

func wornItems(wearer Actor) []*ObjectInstance {
	switch body := wearer.(type) {
	case *Player:
		return COrderedWorn(body)
	case *MobInstance:
		return COrderedMobWorn(body)
	default:
		return nil
	}
}

func (w *World) availableItemHunter(vnums [3]int) *MobInstance {
	for _, character := range w.cOrderedCharacters() {
		hunter := character.mob
		if hunter == nil || !matchesHuntedMob(hunter.GetVNum(), vnums) ||
			hunter.GetFightingBody() != nil || hunter.GetPosition() <= combat.PosSleeping ||
			hunter.IsAffected(affCharm) || w.mobHasActiveHunt(hunter) {
			continue
		}
		return hunter
	}
	return nil
}

func matchesHuntedMob(vnum int, hunters [3]int) bool {
	for _, candidate := range hunters {
		if candidate != 0 && candidate == vnum {
			return true
		}
	}
	return false
}

// mobHasActiveHunt models HUNTING(mob), which is a non-NULL mobile pointer or
// a player ID that currently resolves through a playing descriptor.
func (w *World) mobHasActiveHunt(mob *MobInstance) bool {
	if mob == nil {
		return false
	}
	mob.mu.RLock()
	name, playerID, preyMobID := mob.Hunting, mob.HuntingID, mob.HuntingMobID
	mob.mu.RUnlock()
	if preyMobID != 0 {
		return true // C retains the pointer even after the body leaves character_list.
	}
	if playerID != "" {
		id, err := parseHuntingPlayerID(playerID)
		if err != nil {
			return false
		}
		return w.playerHasPlayingDescriptor(w.GetPlayerByID(id))
	}
	if name == "" {
		return false
	}
	for _, character := range w.cOrderedCharacters() {
		if character.mob != nil && character.mob.GetName() == name {
			return true
		}
		if character.player != nil && character.player.GetName() == name {
			return w.playerHasPlayingDescriptor(character.player)
		}
	}
	return false
}
