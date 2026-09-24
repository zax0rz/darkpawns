package game

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// DamageRefused ports the protection block at the top of C damage()
// (fight.c:1318-1368): the corpse and different-room guards, peaceful rooms,
// the level-10 player-killing protections, and shopkeeper protection. It
// returns true when C's damage() returns FALSE before touching the victim,
// after emitting exactly the bytes C emits on that branch.
//
// Every Go stand-in for damage() must ask this first: the complete port
// (combat.TakeDamage via its callback), DoSpellDamage, the mob-special
// seams, spell damage, and the skill command tail. Before DP-1327 each seam
// carried its own subset, most of them silent, so a skill could hurt a
// level-1 player that C protects (R5c).
//
// The jail-guard branch (fight.c:1370-1400) is not here: it has state
// effects of its own and is owned by the mob-redirect seam.
func (w *World) DamageRefused(ch, victim combat.Combatant) bool {
	if ch == nil || victim == nil {
		return true
	}
	if victim.GetPosition() <= combat.PosDead {
		return true
	}

	if ch.GetRoom() != victim.GetRoom() && ch.GetLevel() < LVL_IMMORT {
		slog.Warn("Attempt to assign damage when ch and vict are in different rooms.",
			"ch", ch.GetName(), "victim", victim.GetName())
		return true
	}

	self := ch.GetName() == victim.GetName()
	victimOutlaw := false
	if p, ok := victim.(*Player); ok {
		victimOutlaw = p.GetFlags()&(1<<uint(PlrOutlaw)) != 0
	}

	// C IS_OUTLAW is !IS_NPC && PLR_OUTLAW; the room is the attacker's.
	if !victimOutlaw && victim.GetFighting() != ch.GetName() && !self &&
		w.RoomHasFlag(ch.GetRoom(), "peaceful") {
		ch.SendMessage("This room just has such a peaceful, easy feeling...\r\n")
		return true
	}

	if !self && !ch.IsNPC() && !victim.IsNPC() {
		if ch.GetLevel() <= 10 {
			w.damageGateAct(ch, victim, "You are not experienced enough to attack $N!")
			return true
		}
		if victim.GetLevel() <= 10 && !victimOutlaw {
			w.damageGateAct(ch, victim, "Ancient forces protect $N from your wrath!")
			return true
		}
	}

	// !ok_damage_shopkeeper(ch, victim) || is_shopkeeper(victim): the OR
	// short-circuits, so the keeper's slap-and-tell prelude runs whenever the
	// victim keeps a shop that does not WILL_START_FIGHT.
	mob, _ := victim.(*MobInstance)
	if (mob != nil && !w.okDamageShopkeeper(ch, mob)) || (mob != nil && IsShopkeeperMob(w, mob)) {
		ch.SendMessage("Ha ha... Don't think so.\r\n")
		w.damageGateStopFighting(ch)
		w.damageGateStopFighting(victim)
		return true
	}
	return false
}

// damageGateAct delivers a TO_CHAR act() line; an NPC attacker has no
// descriptor, so C's act() writes nothing for it.
func (w *World) damageGateAct(ch, victim combat.Combatant, format string) {
	chActor, ok := ch.(Actor)
	if !ok {
		return
	}
	victActor, _ := victim.(Actor)
	Act(w, false, chActor, victActor, nil, nil, format, "", ToChar)
}

// damageGateStopFighting is C's `if (FIGHTING(x)) stop_fighting(x)`.
func (w *World) damageGateStopFighting(c combat.Combatant) {
	if c.GetFighting() == "" {
		return
	}
	if stopper, ok := w.combatEngine.(interface{ StopCombat(string) }); ok {
		stopper.StopCombat(c.GetName())
		return
	}
	c.StopFighting()
}

// okDamageShopkeeper ports ok_damage_shopkeeper (shop.c:1006-1024). It
// returns false, after the keeper slaps the attacker, tells them off, and
// shouts at kender, when the victim keeps a shop that is not
// WILL_START_FIGHT.
func (w *World) okDamageShopkeeper(ch combat.Combatant, keeper *MobInstance) bool {
	if keeper == nil || !isShopKeeperInWorld(w, keeper) {
		return true
	}
	bits, ok := w.ShopBitvectorForKeeper(keeper.GetVNum())
	if !ok || bits&1 != 0 { // not a .shp keeper, or WILL_START_FIGHT
		return true
	}
	// do_action(victim, GET_NAME(ch), cmd_slap, 0) and do_tell(victim,
	// "<name> Get out of here...") parse GET_NAME(ch) as an argument: the
	// keeper resolves its first word as a keyword it can see. A player's name
	// finds the player; a mob's short description ("a Kir-Oshi guard") looks
	// for "a" and usually finds nobody, so C prints nothing. do_tell's
	// get_char_vis would search past the room on a miss; the room is where a
	// real attacker stands, so the room lookup is the reachable case.
	if target, ok := mobRescueVictim(w, keeper, ch.GetName()).(Actor); ok {
		// The slap social's victim and room lines (lib/misc/socials "slap").
		Act(w, false, keeper, target, nil, nil, "$n slaps $N.", "", ToNotVict)
		Act(nil, false, keeper, target, nil, nil, "You are slapped by $n.", "", ToVict)
		Act(nil, false, keeper, target, nil, nil,
			"$n tells you, 'Get out of here before I call the guards!'", "", ToVict)
	}
	if p, ok := ch.(*Player); ok && p.GetRace() == RaceKender {
		w.mobShoutGenComm(keeper, "Stinkin' Kender scum!")
	}
	return false
}

// IsShopkeeperMob ports is_shopkeeper (mobprog.c:473-507): the shop,
// guild, guild_guard, butler and clerk specials, plus the hardcoded
// protector vnums. Keepers of .shp shops carry the shop special from
// assign_the_shopkeepers at boot, which World records separately.
func IsShopkeeperMob(w *World, mob *MobInstance) bool {
	if mob == nil {
		return false
	}
	switch MobSpecAssign[mob.GetVNum()] {
	case "shop_keeper", "guild", "guild_guard", "butler", "clerk":
		return true
	}
	switch mob.GetVNum() {
	case 8003, 8004, 8005, 8006, 8007, 8008, 8009, 8010, 8011, 8078:
		return true
	}
	if w == nil {
		return false
	}
	if _, ok := w.ShopBitvectorForKeeper(mob.GetVNum()); ok {
		return true
	}
	manager := w.GetShopManager()
	if manager == nil {
		return false
	}
	_, ok := manager.GetShopByNPC(mob.GetVNum())
	return ok
}

// mobShoutGenComm is do_gen_comm(mob, arg, 0, SCMD_SHOUT) (act.comm.c:1146):
// an NPC has no descriptor for its own echo, and the zone, position, deaf,
// writing and soundproof filters are the player channel's.
func (w *World) mobShoutGenComm(me *MobInstance, argument string) {
	if me == nil || w.communicationRoomSoundproof(me.GetRoom()) || me.GetLevel() < levelCanShout {
		return
	}
	argument = strings.TrimLeft(argument, " \t\r\n\v\f")
	if argument == "" {
		return
	}
	spec := communicationChannels["shout"]
	format := deleteANSIControls(fmt.Sprintf("$n %ss, '%s'", spec.verb, argument))
	senderRoom := w.GetRoomInWorld(me.GetRoom())
	for _, target := range w.GetAllPlayers() {
		if target.GetFlags()&(1<<uint(spec.recipientOffFlag)) != 0 ||
			target.GetFlags()&(1<<uint(PlrWriting)) != 0 ||
			w.communicationRoomSoundproof(target.GetRoom()) {
			continue
		}
		targetRoom := w.GetRoomInWorld(target.GetRoom())
		if senderRoom == nil || targetRoom == nil || senderRoom.Zone != targetRoom.Zone ||
			target.GetPosition() < spec.minimumHearer {
			continue
		}
		Act(w, false, me, target, nil, nil, format, "", ToVict|ToSleep)
	}
}
