package game

// room_activity.go — port of C room_activity() (src/comm.c:690-756), invoked
// by the game loop every PULSE_MOBILE immediately after mobile_activity
// (heartbeat order: mobile_activity, room_activity, object_activity).
//
// Ported arms, in C's per-character order:
//   - AFF_FLAMING without PRF_NOHASSLE → damage(ch, ch, 15, SPELL_FLAMESTRIKE)
//   - SECT_UNDERWATER without AFF_WATERBREATHE (no-hassle exempt) →
//     damage(ch, ch, 25, SPELL_DROWNING)
//   - SECT_WATER_NOSWIM without AFF_WATERWALK/AFF_FLY/boat (no-hassle exempt)
//     → damage(ch, ch, 25, SPELL_DROWNING)
//   - PC-only: the room's special procedure on pulse; a TRUE return aborts
//     the whole pass (C returns from room_activity)
//   - PC-only: SECT_FLYING without AFF_FLY → fall message pair, move down if
//     possible, otherwise relocate to real_room(5) and abort the pass
//
// Deliberately NOT ported here (pre-existing gaps, documented in the round
// handoff): DG room scripts with RS_ONPULSE, flow_room (no room in the stock
// world carries a ROOM_FLOW_* flag, so C's gated number(0,1) draw never
// fires), loud_mobs, and the CON<=0 croak. The fixed damage calls ride the
// shared fight.c damage() machinery: modifiers, position update, the M-103 /
// M-96 skill_message blocks from lib/misc/messages, the wounded/stunned/dead
// position bytes, and HandleDeath for the corpse's SPELL_DROWNING wording.

import (
	"sort"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// Attack-type numbers from C spells.h — used as damage()/skill_message
// attacktype values exactly as comm.c passes them.
const (
	roomActSpellFlamestrike = 96  // SPELL_FLAMESTRIKE (spells.h:161)
	roomActSpellDrowning    = 103 // SPELL_DROWNING (spells.h:168)
)

// fallFromSkyDTRoom is C's hardcoded real_room(5) death-trap room for flying
// falls with no down exit (comm.c:735).
const fallFromSkyDTRoom = 5

// RoomActivity runs C room_activity() for every character in the world.
//
// C walks the global character_list, which is prepend-ordered (most recently
// created first; db.c:1716-1746). Go has no equivalent single list, so this
// port processes players (sorted by name) before mobs (sorted by VNum) — a
// deterministic order that matches C for the single-observer cases the
// oracle corpus exercises. Multi-occupant water rooms may interleave
// differently than C's list order; that approximation is recorded in the
// round handoff.
func (w *World) RoomActivity() {
	w.mu.RLock()
	players := make([]*Player, 0, len(w.players))
	for _, p := range w.players {
		players = append(players, p)
	}
	mobs := make([]*MobInstance, 0, len(w.activeMobs))
	for _, m := range w.activeMobs {
		mobs = append(mobs, m)
	}
	w.mu.RUnlock()

	sort.Slice(players, func(i, j int) bool { return players[i].Name < players[j].Name })
	sort.Slice(mobs, func(i, j int) bool { return mobs[i].VNum < mobs[j].VNum })

	for _, p := range players {
		if w.roomActivityForPlayer(p) {
			// C returns from room_activity when a room spec reports TRUE or a
			// flying fall lands in the down-less death trap.
			return
		}
	}
	for _, m := range mobs {
		if m == nil || !m.IsAlive() {
			continue
		}
		w.roomActivityDamageArms(m)
	}
}

// roomActivityForPlayer runs the per-character damage arms plus the PC-only
// room-spec dispatch and flying fall. Returns true when C would return from
// room_activity entirely.
func (w *World) roomActivityForPlayer(p *Player) bool {
	if p == nil {
		return false
	}
	roomVNum := p.GetRoom()
	if roomVNum <= 0 {
		return false // C: if (!ch->in_room) continue
	}
	room := w.GetRoomInWorld(roomVNum)
	if room == nil {
		return false
	}

	w.roomActivityDamageArms(p)

	// C: GET_ROOM_SPEC(ch->in_room)(ch, world + ch->in_room, 0, 0) — cmd 0 /
	// no argument, so "" mirrors the pulse-time call.
	if spec := GetRoomSpec(roomVNum); spec != nil {
		if spec(w, p, nil, "", "") {
			return true
		}
	}

	if room.Sector == SECT_FLYING && !p.IsAffected(affFly) {
		p.SendMessage("You fall from the sky...\r\n")
		Act(w, true, p, nil, nil, nil, "$n falls from the sky...", "", ToRoom)
		if ext, ok := room.Exits["down"]; ok && ext.ToRoom > 0 {
			p.SetRoom(ext.ToRoom)
			w.LookAtRoom(p, false)
		} else {
			p.SetRoom(fallFromSkyDTRoom)
			return true
		}
	}
	return false
}

// roomActivityChar is the shared surface of Player and MobInstance that the
// damage arms need: full combat.Combatant (for damage()) plus affect checks.
type roomActivityChar interface {
	combat.Combatant
	IsAffected(int) bool
}

// roomActivityDamageArms ports the three fixed self-damage checks that C
// applies to every character (players and NPCs alike) before the PC-only
// spec/fall arms (comm.c:710-720).
func (w *World) roomActivityDamageArms(ch roomActivityChar) {
	// The no-hassle exemption only exists for players; C NPCs never carry
	// PRF flags.
	nohassle := false
	if p, ok := ch.(*Player); ok {
		nohassle = p.GetFlags()&(1<<uint(PrfNohassle)) != 0
	}

	roomVNum := ch.GetRoom()
	if roomVNum <= 0 {
		return
	}
	room := w.GetRoomInWorld(roomVNum)
	if room == nil {
		return
	}

	if !nohassle && ch.IsAffected(affFlaming) {
		w.roomActivitySelfDamage(ch, 15, roomActSpellFlamestrike)
	}
	if room.Sector == SECT_UNDERWATER && !ch.IsAffected(affWaterBreathe) && !nohassle {
		w.roomActivitySelfDamage(ch, 25, roomActSpellDrowning)
	}
	if room.Sector == SECT_WATER_NOSWIM {
		if !ch.IsAffected(affWaterWalk) && !ch.IsAffected(affFly) && !nohassle {
			if !roomActivityHasBoat(ch) {
				w.roomActivitySelfDamage(ch, 25, roomActSpellDrowning)
			}
		}
	}
}

// roomActivityHasBoat mirrors C has_boat (act.movement.c) for both character
// kinds: immortals, AFF_WATERWALK, AFF_FLY, or an ITEM_BOAT in inventory or
// equipment.
func roomActivityHasBoat(ch roomActivityChar) bool {
	if ch.GetLevel() >= LVL_IMMORT {
		return true
	}
	if ch.IsAffected(affWaterWalk) || ch.IsAffected(affFly) {
		return true
	}
	switch c := ch.(type) {
	case *Player:
		if c.Inventory != nil {
			for _, obj := range c.Inventory.Items {
				if obj != nil && obj.Prototype != nil && obj.Prototype.TypeFlag == ITEM_BOAT {
					return true
				}
			}
		}
		for _, obj := range c.Equipment.Slots {
			if obj != nil && obj.Prototype != nil && obj.Prototype.TypeFlag == ITEM_BOAT {
				return true
			}
		}
	case *MobInstance:
		for _, obj := range c.Inventory {
			if obj != nil && obj.Prototype != nil && obj.Prototype.TypeFlag == ITEM_BOAT {
				return true
			}
		}
		for _, obj := range c.Equipment {
			if obj != nil && obj.Prototype != nil && obj.Prototype.TypeFlag == ITEM_BOAT {
				return true
			}
		}
	}
	return false
}

// roomActivitySelfDamage ports the ch == victim arm of fight.c damage() used
// by room_activity's fixed damage calls: the modifier funnel, HP/position
// update, skill_message (M-96/M-103 blocks from lib/misc/messages), the
// position/pain/scream/bleeding bytes, and the death pipeline. Branches that
// C cannot reach with ch == victim (peaceful-room notice, novice protection,
// jail redirects, wimpy flee, group exp) are absent by construction.
func (w *World) roomActivitySelfDamage(vict combat.Combatant, dam int, attackType int) bool {
	if vict == nil || vict.GetPosition() <= combat.PosDead {
		return false // C: "Attempt to damage a corpse."
	}

	// fight.c:1449-1453 — self-damage still strips AFF_HIDE inline.
	hidden := false
	switch c := vict.(type) {
	case *Player:
		hidden = c.IsAffected(affHide)
		if hidden {
			c.RemoveAffectBit(affHide)
		}
	case *MobInstance:
		hidden = c.IsAffected(affHide)
		if hidden {
			c.RemoveAffected(affHide)
		}
	}
	if hidden {
		Act(w, false, vict, nil, nil, nil, "$n slowly fades into existence.", "", ToRoom)
	}

	dam = combat.ApplyDamageModifiers(vict, vict, dam)
	if dam > 0 {
		vict.TakeDamage(dam)
	}
	newPos := combat.GetPositionFromHP(vict.GetHP(), vict.GetPosition())
	vict.SetPosition(newPos)

	// fight.c:1534-1545 — a spell attacktype is never IS_WEAPON, so C always
	// takes skill_message here, before the position bytes.
	combat.EmitSkillMessage(dam, vict.GetName(), vict.GetName(), attackType, vict.GetRoom())
	w.emitMobSkillSurvival(vict, vict, dam, newPos)

	if newPos == combat.PosDead {
		w.HandleDeath(vict, vict, attackType)
	}
	return dam > 0
}
