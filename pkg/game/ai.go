package game

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// CombatEngine interface for AI to initiate combat
type CombatEngine interface {
	StartCombat(attacker, defender combat.Combatant) error
	IsFighting(body combat.Combatant) bool
	GetCombatTarget(body combat.Combatant) (combat.Combatant, bool)
}

// AIBehavior defines mob AI behavior
type AIBehavior int

const (
	AIWandering AIBehavior = iota
	AIAggressive
	AISentinel
)

// CRIT-006: aiCombatEngine moved to World.combatEngine.
// SetAICombatEngine is replaced by World.SetCombatEngine.

// runMobAI runs AI for a single mob.
//
// DP-590: this used to hold mob.mu for the whole AI cycle (CRIT-004), but the
// dispatch below runs entirely through self-locking accessors (GetName,
// GetFighting, SetStatus, wanderMobWithDoor → GetRoom/SetRoom, …). Holding the write
// lock made the first accessor self-deadlock, so the mob AI never ran. The
// accessors provide their own synchronization; we no longer hold mob.mu here.
func (w *World) runMobAI(mob *MobInstance) {
	mob.mu.RLock()
	ready := mob.Proto() != nil
	mob.mu.RUnlock()
	if !ready {
		return
	}

	// Don't act if already fighting
	if w.combatEngine != nil && w.combatEngine.IsFighting(mob) {
		return
	}

	// MED-009: call per-mob activity instead of full MobileActivity()
	// This fixes the O(N²) bug where runMobAI called MobileActivity()
	// which re-iterated ALL mobs — making every mob get processed N times
	// per tick. MobileActivityForMob processes a single mob.
	//
	// Wandering happens inside MobileActivityForMob (mobileActivityForMob →
	// wanderMobWithDoor), exactly once per tick — matching C mobile_activity, which
	// runs the movement block once per mob. DP-908 removed a redundant second
	// wanderMobWithDoor call here plus a non-C 25% gate (number(0,99) < 25) that had
	// no equivalent in mobact.c and dropped the effective wander rate to ~8%.
	w.MobileActivityForMob(mob)
}

// wanderMobWithDoor moves a mob to the adjacent room selected by door, faithful
// to C mobact.c's movement block (src/mobact.c:119-143). The caller performs
// the unconditional number(0,18) draw before checking SENTINEL or position;
// this helper never draws PRNG. Caller must NOT hold mob.mu — the accessors lock
// individually (DP-590).
//
// C semantics reproduced here (DP-908):
//   - door = number(0, 18): a SINGLE random draw in [0,18].
//   - door < NUM_OF_DIRS (6): the same draw both gates movement (~6/19 ≈ 31.6%
//     chance the door is even a real cardinal direction) AND picks the exit.
//     Previously Go matched the *probability* but drew the direction separately,
//     which changes the per-direction distribution. We now reuse the gate draw
//     as the direction index.
//   - CAN_GO(ch, door): the exit exists in that cardinal direction and is not
//     EX_CLOSED (utils.h:565-567), read from the live room.
//   - ROOM_DEATH / ROOM_NOMOB: skip.
//   - MOB_STAY_ZONE: skip exits that leave the mob's home zone.
//   - Water sectors require CAN_SWIM; the flying sector requires IS_FLYING
//     (mobact.c:130-138; utils.h:148-153). A failed sector gate continues the
//     C mobile_activity loop, skipping this mob's remaining pulse behavior.
//
// Note: SENTINEL and exact POS_STANDING are checked by the caller only after it
// has consumed the unconditional door draw.
// The return value is true only for the C sector-gate continue described above.
func (w *World) wanderMobWithDoor(mob *MobInstance, door int) bool {
	// door < NUM_OF_DIRS (len(dirs) == 6): if the draw isn't a real direction,
	// the mob doesn't move this tick. This is both the gate and the direction.
	if door < 0 || door >= len(dirs) {
		return false
	}

	snap := w.snapshots.Snapshot()
	room, ok := snap.Rooms[mob.GetRoom()] // getter — mutex-protected
	if !ok {
		return false
	}

	direction := dirs[door]
	exit, ok := room.Exits[direction] // CAN_GO(ch, door)
	if !ok {
		return false
	}
	// CAN_GO also refuses an EX_CLOSED exit (src/utils.h:565-567), so a
	// wandering mob never passes a closed door (src/mobact.c:124). Door state
	// changes at runtime, so read it from the live room the way player
	// movement does (getExit), not from the snapshot.
	if live := w.GetRoomInWorld(mob.GetRoom()); live != nil {
		if liveExit, ok := live.Exits[direction]; ok && liveExit.ExitInfo&parser.ExitClosed != 0 {
			return false
		}
	}
	targetRoom, ok := snap.Rooms[exit.ToRoom]
	if !ok {
		return false
	}

	// MOB_STAY_ZONE: skip exits that lead to a different zone (mobact.c:127-128).
	if hasMobFlag(mob, "stay_zone") && targetRoom.Zone != room.Zone {
		return false
	}

	// C checks ROOM_DEATH and ROOM_NOMOB before either sector gate.
	// ROOM_DEATH also subsumes C's NPC-DT death path (act.movement.c:288-301).
	if roomHasFlag(targetRoom, "death") || roomHasFlag(targetRoom, "no_mob") {
		return false
	}

	// C's CAN_SWIM includes waterwalk, fly, no-hassle, and has_boat
	// (utils.h:150-153; act.movement.c:58-78). The existing room activity
	// helper models has_boat for mobiles, including innate affects and boats.
	if targetRoom.Sector == SECT_WATER_SWIM || targetRoom.Sector == SECT_WATER_NOSWIM {
		if !roomActivityHasBoat(mob) {
			return true // mobact.c:131-135 continues the outer mobile loop
		}
	}
	// IS_FLYING is AFF_FLY or PRF_NOHASSLE (utils.h:148-149). NPC preference
	// flags are the zeroed dummy_mob record (db.c:1281), so only AFF_FLY applies.
	if targetRoom.Sector == SECT_FLYING && !mob.IsAffected(affFly) {
		return true // mobact.c:136-138 continues the outer mobile loop
	}

	// Move mob. GetRoom/SetRoom lock individually; the caller no longer holds
	// mob.mu (DP-590), so there is no lock to release or re-acquire here.
	oldRoom := mob.GetRoom()

	slog.Debug(
		"mob wandered",
		"mob_vnum", mob.GetVNum(),
		"name", mob.GetName(),
		"from", oldRoom,
		"to", targetRoom.VNum,
		"dir", direction,
	)

	// src/act.movement.c:214-220: perform_move() sends the leave act unless
	// the mobile is sneaking. Act preserves C's visibility and capitalization.
	if !mob.IsAffected(affSneak) {
		Act(w, true, mob, nil, nil, nil, fmt.Sprintf("$n leaves %s.", direction), "", ToRoom)
	}

	// C char_from_room/char_to_room and arrival act: act.movement.c:222-280.
	mob.SetRoom(targetRoom.VNum)
	if !mob.IsAffected(affSneak) {
		var arrival string
		switch door {
		case 0, 1, 2, 3:
			direct := dirs[revDir[door]]
			switch {
			case mob.IsAffected(affFly):
				arrival = fmt.Sprintf("$n flies in from the %s.", direct)
			case targetRoom.Sector == SECT_UNDERWATER:
				arrival = fmt.Sprintf("$n swims in from the %s.", direct)
			default:
				arrival = fmt.Sprintf("$n arrives from the %s.", direct)
			}
		case 4:
			switch {
			case mob.IsAffected(affFly):
				arrival = "$n flies in from below."
			case targetRoom.Sector == SECT_UNDERWATER:
				arrival = "$n swims in from below."
			default:
				arrival = "$n climbs in from below."
			}
		case 5:
			switch {
			case mob.IsAffected(affFly):
				arrival = "$n flies in from above."
			case targetRoom.Sector == SECT_UNDERWATER:
				arrival = "$n swims in from above."
			default:
				arrival = "$n climbs in from above."
			}
		}
		Act(w, true, mob, nil, nil, nil, arrival, "", ToRoom)
	}

	// MobProg entry trigger — fires when mob enters a room
	w.EntryProg(mob, targetRoom.VNum)
	if mob.GetRoom() < 0 {
		return false // C: perform_move returns if entry_prog extracted the mover.
	}
	if !mob.IsAffected(affSneak) {
		// act.movement.c:273-282 runs mp_greet and then each room mob's greet
		// script after entry_prog, in the destination room's character order.
		w.MpGreetActor(mob, mob.GetRoom())
		if mob.GetRoom() < 0 {
			return false
		}
		w.runMobGreetScripts(mob, mob.GetRoom())
		if mob.GetRoom() < 0 {
			return false
		}
	}
	return false
}

// StartEventQueue starts the event processing loop used by MobProg delayed
// events and other scheduled callbacks. The loop runs at 10 pulses per second
// (100ms), matching the original C code: OPT_USEC = 100000, PASSES_PER_SEC = 10.
// Source: comm.c game_loop() — heartbeat(++pulse) calls event_process().
//
// Mob AI is NOT driven here. It is driven solely by the game loop's
// OnMobileActivity callback (PULSE_MOBILE = 4s) → World.MobileActivity(),
// faithful to C's mobile_activity() called from heartbeat() (DP-1035).
func (w *World) StartEventQueue() {
	if w.EventQueue != nil {
		ctx := context.Background()
		w.EventQueue.Start(ctx)
	}
}
