package game

import "github.com/zax0rz/darkpawns/pkg/combat"

// specShopKeeperActor offers SPECIAL(shop_keeper) (src/shop.c:942-1002) to a
// mobile command actor.
//
// The special is not in any assignment macro: assign_the_shopkeepers
// (src/shop.c:1226-1243) writes it onto every keeper loaded from .shp data at
// boot, so the static MobSpecAssign cannot know it. The session layer recovers
// that assignment from the world's shop index, the same way it already does for
// the player lane (pkg/session/commands.go), and then dispatches here.
//
// C order: secondary shop function (SHOP_FUNC, the keeper's pre-boot special —
// a separate entry in the Go table, so it is not repeated here), the
// switched-keeper arm, the shop-room and awake gates, then steal/buy/sell/
// value/list. Every other command returns FALSE, which is what lets an ordered
// pet act in a shop.
func specShopKeeperActor(w *World, ctx CharacterSpecialContext, me *MobInstance, cmd, arg string) SpecialResult {
	actor := specialActor(ctx)
	if actor == nil || me == nil {
		return SpecialUnsupported
	}
	if _, ok := w.GetShopByKeeper(me.GetVNum()); !ok {
		return SpecialDeclined // C: the keeper matches no shop_nr
	}
	// C's `keeper == ch` arm is a switched keeper acting through itself; the
	// player lane covers it (pkg/session/commands.go ResetShopStockSort).
	if Actor(me) == actor {
		return SpecialDeclined
	}
	shop, _ := w.GetShopByKeeper(me.GetVNum())
	inShopRoom := false
	for _, room := range shop.Rooms {
		if room == actor.GetRoom() {
			inShopRoom = true
			break
		}
	}
	if !inShopRoom {
		return SpecialDeclined
	}
	if me.GetPosition() <= combat.PosSleeping {
		return SpecialDeclined
	}

	if cmd == "steal" {
		// GET_RACE for a mobile is mob_specials.race (src/utils.h:347).
		if me.Proto() != nil && me.Proto().Race == RaceKender {
			w.mobShoutGenComm(me, "Stinkin' Kender scum!")
		}
		w.npcSocial(me, "slap", actor.GetName())
		// C's act(..., TO_CHAR) names the actor and is the keeper's shout; a
		// descriptor-less pet receives nothing, a switched body receives it.
		Act(w, false, actor, me, nil, nil, "$N shouts '"+msgNoStealHere+"'", "", ToChar)
		return SpecialHandled
	}
	switch cmd {
	case "buy", "sell", "value":
		// The shopping_* bodies talk to the actor through do_tell and its own
		// inventory; they are not ported for an NPC body yet.
		return SpecialUnsupported
	case "list":
		// shopping_list ends in page_string(ch->desc, …) (src/shop.c:925);
		// a pet has no descriptor, so C dereferences NULL there. That reads
		// undefined state (R1a) and stays a recorded gap, never a copied
		// crash.
		return SpecialUnsupported
	}
	return SpecialDeclined
}

func init() {
	RegisterActorSpec("shop_keeper", specShopKeeperActor)
}
