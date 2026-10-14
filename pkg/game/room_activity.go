package game

// room_activity.go — port of C room_activity() (src/comm.c:691-756), invoked
// by the game loop every PULSE_MOBILE immediately after mobile_activity
// (heartbeat order: mobile_activity, room_activity, object_activity).
//
// Ported arms, in C's per-character order:
//   - PC-only, room flagged RS_ONPULSE → the room's onpulse Lua script
//     (comm.c:706-708), which can draw number(); its first-in-body position
//     is draw-order-sensitive (R3)
//   - AFF_FLAMING without PRF_NOHASSLE → damage(ch, ch, 15, SPELL_FLAMESTRIKE)
//   - SECT_UNDERWATER without AFF_WATERBREATHE (no-hassle exempt) →
//     damage(ch, ch, 25, SPELL_DROWNING)
//   - SECT_WATER_NOSWIM without AFF_WATERWALK/AFF_FLY/boat (no-hassle exempt)
//     → damage(ch, ch, 25, SPELL_DROWNING)
//   - PC-only: the room's special procedure on pulse; a TRUE return aborts
//     the whole pass (C returns from room_activity)
//   - PC-only: SECT_FLYING without AFF_FLY → fall message pair, move down if
//     possible, otherwise relocate to real_room(5) and abort the pass
//   - PC-only: a ROOM_FLOW_* room draws number(0,1) every pulse and drifts
//     the player downstream on a 0 (comm.c:741-742 → flowRoom)
//   - NPC-only: loud_mobs (comm.c:743-744 → loudMobs) — number(0,20) is
//     drawn for every noise-carrying mob every pulse (R3d gated draw)
//   - PC-only, every iteration that was not aborted: the CON<=0 croak
//     (comm.c:748-754) — "too unhealthy to live", HP pinned to 1, direct
//     raw_kill with C's effective attacktype 0
//
// The stock world DOES carry ROOM_FLOW_* flags (73 rooms, e.g. #9700 "Under a
// Bridge", flow-east); the earlier handoff note claiming no room carried one
// was wrong. Room scripts with RS_ONPULSE (comm.c:706-708) run through
// RunRoomPulseScript in room_obj_scripts.go. The fixed damage calls ride the
// shared fight.c damage() machinery: modifiers, position update, the M-103 /
// M-96 skill_message blocks from lib/misc/messages, the wounded/stunned/dead
// position bytes, and HandleDeath for the corpse's SPELL_DROWNING wording.

import (
	"sort"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
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

// roomActCroakAttackType is the attacktype C's croak raw_kill call effectively
// passes. comm.c:697 declares raw_kill(ch, killer, attacktype) with three
// parameters, but fight.c:534 defines raw_kill(struct char_data *ch, int
// attacktype) with two, so the NULL the call site passes as its second
// argument (comm.c:753) lands in attacktype on both 32-bit cdecl and x86-64
// SysV — 0, not the TYPE_UNDEFINED (-1, spells.h:60) the source text
// suggests. At 0, make_corpse takes the default arm (fight.c:369-371: "The
// corpse of %s is lying here.", byte-identical to the TYPE_UNDEFINED arm at
// fight.c:284-286), make_corpse's SPELL_DISINTEGRATE gate (fight.c:266-270)
// is false, and make_dust's SPELL_DISINTEGRATE test (fight.c:470) is false,
// so race alone picks dust over corpse — the same outcomes TYPE_UNDEFINED
// would produce.
const roomActCroakAttackType = 0

// ROOM_FLOW_* bit positions (src/structs.h:77-82) for roomHasFlagBit, which
// takes plain positions — the RoomFlow* constants in room_flags.go carry the
// equivalent masks.
const (
	roomFlowNorthBit = 21
	roomFlowSouthBit = 22
	roomFlowEastBit  = 23
	roomFlowWestBit  = 24
	roomFlowUpBit    = 25
	roomFlowDownBit  = 26
)

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
		// C comm.c:743-744 — the NPC else-arm. loud_mobs draws number(0,20)
		// for every noise-carrying mob on every mobile pulse, after that
		// mob's damage arms and before the croak check (R3d).
		w.loudMobs(m)
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

	// C comm.c:706-708 — the room's RS_ONPULSE script runs FIRST in the
	// per-character body, before the damage arms; pattern_dmg.lua draws
	// number() here, so this position is draw-order-sensitive (R3).
	w.RunRoomPulseScript(p)

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
	// C comm.c:741-742 — the flow gate evaluates the room the fall (if any)
	// left the player in, and ROOM_FLOWS is tested before number() draws, so
	// every player in a ROOM_FLOW_* room burns exactly one draw per pulse.
	if roomAfterFall := w.GetRoomInWorld(p.GetRoom()); roomAfterFall != nil && roomFlows(roomAfterFall) {
		if combat.GetRoller().Number(0, 1) == 0 {
			w.flowRoom(p)
		}
	}

	// C comm.c:748-754 — the CON<=0 croak, at the end of the per-character
	// body. GET_CON is the affect_total result (gear included), and a direct
	// raw_kill: no "You are dead!" line, no mudlog, no die_with_killer
	// experience or CON loss. GET_HIT is pinned to 1 before the kill so the
	// corpse-less re-entry starts at 1 HP. Extraction is deferred to the next
	// heartbeat, exactly as for any death.
	if p.GetCon() <= 0 && p.GetLevel() < LVL_IMMORT {
		p.SendMessage("You are too unhealthy to live!\r\n")
		Act(w, true, p, nil, nil, nil, "Suddenly, $n croaks before your very eyes!", "", ToRoom)
		p.SetHP(1)
		w.RawKillCombatant(p, roomActCroakAttackType)
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
		w.selfDamage(ch, 15, roomActSpellFlamestrike)
	}
	if room.Sector == SECT_UNDERWATER && !ch.IsAffected(affWaterBreathe) && !nohassle {
		w.selfDamage(ch, 25, roomActSpellDrowning)
	}
	if room.Sector == SECT_WATER_NOSWIM {
		if !ch.IsAffected(affWaterWalk) && !ch.IsAffected(affFly) && !nohassle {
			if !roomActivityHasBoat(ch) {
				w.selfDamage(ch, 25, roomActSpellDrowning)
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
		for _, obj := range c.EquipmentSnapshot() {
			if obj != nil && obj.Prototype != nil && obj.Prototype.TypeFlag == ITEM_BOAT {
				return true
			}
		}
	}
	return false
}

// selfDamage ports the ch == victim arm of fight.c damage() used
// by room activity, tick damage and equip_char: the modifier funnel, HP/position
// update, skill_message (M-96/M-103 blocks from lib/misc/messages), the
// position/pain/scream/bleeding bytes, and the death pipeline. Branches that
// C cannot reach with ch == victim (peaceful-room notice, novice protection,
// jail redirects, wimpy flee) are absent by construction.
func (w *World) selfDamage(vict combat.Combatant, dam int, attackType int) bool {
	if vict == nil || vict.GetPosition() <= combat.PosDead {
		return false // C: "Attempt to damage a corpse."
	}

	if w.DamageRefused(vict, vict) {
		return false
	}

	// fight.c:1460-1464 — self-damage still strips AFF_HIDE inline.
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
	if w.DamageBeforeMessage(vict, vict, dam) {
		return false
	}
	newPos := vict.GetPosition()

	// fight.c:1534-1545 — a spell attacktype is never IS_WEAPON, so C always
	// takes skill_message here, before the position bytes.
	combat.EmitSkillMessage(dam, vict, vict, attackType, vict.GetRoom())
	// fight.c:1549-1558 applies to self damage too, including both draws.
	if p, ok := vict.(*Player); ok && p.IsMounted() && dam > 0 && combat.GetRoller().Number(0, 99) < 10 {
		Act(w, false, p, nil, nil, nil, "The hit knocks you off of your mount!\r\n", "", ToChar)
		Act(w, false, p, nil, nil, nil, "$n gets knocked off of his mount.", "", ToRoom)
		w.doDismount(p, nil, "dismount", "")
		if float64(combat.GetRoller().Number(0, 99)) < float64(p.GetDex())*1.5 {
			p.SendMessage("You land on your feet!\r\n")
		} else {
			p.SendMessage("You hit the ground with a thud.\r\n")
			p.SetPosition(combat.PosSitting)
		}
		newPos = p.GetPosition()
	}
	w.emitMobSkillSurvival(vict, vict, dam, newPos)

	if newPos == combat.PosDead {
		w.HandleDeath(vict, vict, attackType)
	}
	return dam > 0
}

// roomFlows is C's ROOM_FLOWS macro (src/utils.h:402-407): true when the room
// carries any of the six ROOM_FLOW_* bits.
func roomFlows(room *parser.Room) bool {
	if room == nil {
		return false
	}
	for bit := roomFlowNorthBit; bit <= roomFlowDownBit; bit++ {
		if roomHasFlagBit(room.Flags, bit) {
			return true
		}
	}
	return false
}

// flowRoom ports flow_room (src/new_cmds2.c:885-955): the current sweeps the
// player (and their mount) one room downstream. C's caller already excludes
// NPCs, so the IS_NPC(ch) guard inside the original is unreachable here.
func (w *World) flowRoom(p *Player) {
	wasIn := p.GetRoom()
	room := w.GetRoomInWorld(wasIn)
	if room == nil {
		return
	}

	// C's if/else chain picks the first flow flag in NORTH, SOUTH, EAST,
	// WEST, UP, DOWN order. "direct" names where the victim is seen to come
	// from: flowing north leaves you arriving from the south, and so on.
	dir := -1
	direct := ""
	switch {
	case roomHasFlagBit(room.Flags, roomFlowNorthBit):
		dir, direct = 0, "the south"
	case roomHasFlagBit(room.Flags, roomFlowSouthBit):
		dir, direct = 2, "the north"
	case roomHasFlagBit(room.Flags, roomFlowEastBit):
		dir, direct = 1, "the west"
	case roomHasFlagBit(room.Flags, roomFlowWestBit):
		dir, direct = 3, "the east"
	case roomHasFlagBit(room.Flags, roomFlowUpBit):
		dir, direct = 4, "below"
	case roomHasFlagBit(room.Flags, roomFlowDownBit):
		dir, direct = 5, "above"
	}

	// C's guard order: missing direction flag, no such exit, the
	// immortal+nohassle exemption, and a NOWHERE destination.
	if dir < 0 {
		return
	}
	ext, hasExit := room.Exits[dirs[dir]]
	if !hasExit || ext.ToRoom <= 0 {
		return
	}
	if p.GetLevel() >= LVL_IMMORT && p.GetFlags()&(1<<uint(PrfNohassle)) != 0 {
		return
	}
	toRoom := ext.ToRoom

	// The mount drifts with its rider; both rooms hear the pair arrive and
	// leave. $N is the rider, $n the mount (C: act(msg, TRUE, mount, 0,
	// ch, TO_NOTVICT) — ch=mount, vict_obj=rider).
	var mount *MobInstance
	if p.IsMounted() {
		mount = w.riddenMount(p)
	}
	if mount != nil {
		Act(w, true, mount, p, nil, nil, "$N drifts "+dirs[dir]+" on $n.", "", ToNotVict)
		mount.SetRoom(toRoom)
		Act(w, true, mount, p, nil, nil, "$N drifts in from "+direct+" on $n.", "", ToNotVict)
	} else {
		Act(w, true, p, nil, nil, nil, "$n drifts "+dirs[dir]+".", "", ToRoom)
	}
	p.SendMessage("You drift " + dirs[dir] + ".\r\n")
	p.SetRoom(toRoom)
	w.LookAtRoom(p, false)
	Act(w, true, p, nil, nil, nil, "$n drifts in from "+direct+".", "", ToNotVict)
}

// loudMobs ports loud_mobs (src/spec_procs2.c:1532-1550): a noise-carrying
// mob is heard in every room it can reach, one pulse in twenty-one. The
// number(0,20) draw happens for every noise-carrying mob on every mobile
// pulse — including the twenty pulses in twenty-one that say nothing — so the
// shared stream stays aligned with C (R3d).
func (w *World) loudMobs(m *MobInstance) {
	noise := ""
	if proto := m.Proto(); proto != nil {
		noise = proto.Noise
	}
	if noise == "" || m.GetRoom() <= 0 {
		return
	}
	if combat.GetRoller().Number(0, 20) > 0 {
		return
	}
	room := w.GetRoomInWorld(m.GetRoom())
	if room == nil {
		return
	}
	for door := 0; door < len(dirs); door++ {
		ext, ok := room.Exits[dirs[door]]
		if !ok || ext.ToRoom <= 0 {
			continue
		}
		w.actInRoom(m, ext.ToRoom, noise)
	}
}

// actInRoom is act()'s TO_ROOM delivery with the audience room overridden.
// C's loud_mobs repoints ch->in_room at the adjacent room before calling
// act() (src/spec_procs2.c:1544-1546), which redirects both act()'s audience
// and its ROOM_ARENA broadcast hook to that room (src/comm.c:2519-2534).
// hide_invisible is loud_mobs' FALSE, so invisible actors are not filtered.
func (w *World) actInRoom(ch Actor, roomVNum int, format string) {
	if format == "" {
		return
	}
	if w.RoomHasFlag(roomVNum, "arena") {
		actArenaBroadcast(w, false, ch, nil, nil, nil, format, "", sendActLine)
	}
	for _, to := range w.actChar(roomVNum) {
		if !sendOk(to, false) {
			continue
		}
		if to == ch {
			continue
		}
		msg := performAct(format, ch, nil, nil, nil, "", "", to)
		sendActLine(to, cap(msg)+"\r\n")
	}
}
