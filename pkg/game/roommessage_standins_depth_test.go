package game

// roommessage_standins_depth_test.go — fidelity cover for the three
// former roomMessage stand-ins (specUndeadKnight, specNoMoveDown,
// specFieldObject). None of the three is reachable through the live
// dispatch in either engine: black/red undead knights are declared at
// src/spec_assign.c:108-109 but never ASSIGNMOB'd, no_move_down is
// declared at src/spec_assign.c:151 with the same status, obj specials
// only receive a non-empty cmd from the command lane
// (src/interpreter.c:1440-1476), and the real field objects carry VALs
// that are all zero (lib/world/obj/0.obj #50/#52). No census scenario
// can reach these bodies on the C oracle at all, so — as with
// no_move_down's unit-test precedent — direct tests against C's
// behaviour carry the proof (R5h: each test fails when the old body is
// restored).

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

// seedFirstNumber finds a stream seed whose first number(from, to) is want.
func seedFirstNumber(t *testing.T, from, to, want int) uint32 {
	t.Helper()
	for seed := uint32(1); seed < 5000; seed++ {
		dprng.ResetStream(seed)
		if dprng.Number(from, to) == want {
			return seed
		}
	}
	t.Fatalf("no seed in 1..4999 draws %d first", want)
	return 0
}

// seedFirstDice finds a stream seed whose first Dice(num, size) is want.
func seedFirstDice(t *testing.T, num, size, want int) uint32 {
	t.Helper()
	for seed := uint32(1); seed < 5000; seed++ {
		dprng.ResetStream(seed)
		if dprng.Dice(num, size) == want {
			return seed
		}
	}
	t.Fatalf("no seed in 1..4999 draws %d first", want)
	return 0
}

func roomMessageWorld(t *testing.T, extra parser.World) (*World, map[string]string) {
	t.Helper()
	parsed := parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Voice Room", Zone: 1}},
		Mobs:  extra.Mobs,
		Objs:  extra.Objs,
	}
	w, err := NewWorld(&parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	w.StopAITicker()
	t.Cleanup(w.StopAITicker)
	out := map[string]string{}
	w.MessageSink = func(name string, msg []byte) { out[name] += string(msg) }
	return w, out
}

var testPlayerID int

func addRoomPlayer(t *testing.T, w *World, name string) *Player {
	t.Helper()
	testPlayerID++
	p := NewPlayer(testPlayerID, name, 1001)
	if err := w.AddPlayer(p); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	return p
}

func knightMobProto(vnum int, shortDesc string) parser.Mob {
	return parser.Mob{
		VNum: vnum, Keywords: "knight", ShortDesc: shortDesc, Sex: 0,
		Race: 1, Level: 20, Position: 8, DefaultPos: 8,
		HP: parser.DiceRoll{Plus: 100},
	}
}

// TestUndeadKnightHateDepth pins black/red undead knights' hate arms
// against C (src/spec_procs.c:1192-1202, :1250-1260): COrderedRoomMobs
// people order, number(0,2) — not (0,3) — "$n sees the knight dressed
// in red/black ..." through Act TO_ROOM, and the mobHit path for hit().
// The old body drew number(0,3), pre-rendered the target's own name into
// a roomMessage, and only pointed a target without swinging.
func TestUndeadKnightHateDepth(t *testing.T) {
	setup := func(t *testing.T) (*World, map[string]string, *MobInstance, *MobInstance) {
		w, out := roomMessageWorld(t, parser.World{Mobs: []parser.Mob{
			knightMobProto(18401, "the Black Undead Knight"),
			knightMobProto(18402, "the Red Undead Knight"),
		}})
		addRoomPlayer(t, w, "Watcher")
		black, err := w.SpawnMob(18401, 1001)
		if err != nil {
			t.Fatalf("SpawnMob(18401): %v", err)
		}
		red, err := w.SpawnMob(18402, 1001)
		if err != nil {
			t.Fatalf("SpawnMob(18402): %v", err)
		}
		black.SetPosition(combat.PosStanding)
		red.SetPosition(combat.PosStanding)
		return w, out, black, red
	}
	const redLine = "The Black Undead Knight sees the knight dressed in red and gives a battle cry!\r\n"
	const blackLine = "The Red Undead Knight sees the knight dressed in black and gives a battle cry!\r\n"

	t.Run("black knight cries at a red knight", func(t *testing.T) {
		w, out, black, _ := setup(t)
		dprng.ResetStream(seedFirstNumber(t, 0, 2, 0))
		if !specBlackUndeadKnight(w, nil, black, "", "") {
			t.Fatal("specBlackUndeadKnight = false, want true on a passing roll")
		}
		if got := out["Watcher"]; !strings.Contains(got, redLine) {
			t.Fatalf("Watcher = %q, want to contain %q", got, redLine)
		}
	})

	t.Run("red knight cries at a black knight", func(t *testing.T) {
		w, out, _, red := setup(t)
		dprng.ResetStream(seedFirstNumber(t, 0, 2, 0))
		if !specRedUndeadKnight(w, nil, red, "", "") {
			t.Fatal("specRedUndeadKnight = false, want true on a passing roll")
		}
		if got := out["Watcher"]; !strings.Contains(got, blackLine) {
			t.Fatalf("Watcher = %q, want to contain %q", got, blackLine)
		}
	})

	t.Run("failing roll is silent", func(t *testing.T) {
		w, out, black, _ := setup(t)
		dprng.ResetStream(seedFirstNumber(t, 0, 2, 1))
		if specBlackUndeadKnight(w, nil, black, "", "") {
			t.Fatal("specBlackUndeadKnight = true, want false on a failing roll")
		}
		if got := out["Watcher"]; got != "" {
			t.Fatalf("Watcher = %q, want silence", got)
		}
	})

	t.Run("sleeping knight never acts", func(t *testing.T) {
		w, out, black, _ := setup(t)
		black.SetPosition(combat.PosSleeping)
		dprng.ResetStream(seedFirstNumber(t, 0, 2, 0))
		if specBlackUndeadKnight(w, nil, black, "", "") {
			t.Fatal("sleeping knight acted")
		}
		if got := out["Watcher"]; got != "" {
			t.Fatalf("Watcher = %q, want silence for a sleeping knight", got)
		}
	})
}

// TestNoMoveDownDepth pins SPECIAL(no_move_down) (src/spec_procs3.c:
// 316-335): the immort gate, the pulse gate, the sleeping gate, and the
// three act() lines in C's order (TO_NOTVICT, TO_VICT, TO_ROOM). The old
// body sent one invented roomMessage line to everyone.
func TestNoMoveDownDepth(t *testing.T) {
	setup := func(t *testing.T) (*World, map[string]string, *Player, *MobInstance) {
		w, out := roomMessageWorld(t, parser.World{Mobs: []parser.Mob{
			{
				VNum: 9802, Keywords: "blocker", ShortDesc: "the blocker", Sex: 0,
				Race: 1, Level: 20, Position: 8, DefaultPos: 8, HP: parser.DiceRoll{Plus: 100},
			},
		}})
		mover := addRoomPlayer(t, w, "Mover")
		mover.SetLevel(10)
		addRoomPlayer(t, w, "Watcher")
		blocker, err := w.SpawnMob(9802, 1001)
		if err != nil {
			t.Fatalf("SpawnMob(9802): %v", err)
		}
		blocker.SetPosition(combat.PosStanding)
		return w, out, mover, blocker
	}
	const toVict = "The blocker blocks your way.\r\n"
	const toNotVict = "The blocker blocks Mover's way.\r\n"
	const toRoom = "The blocker says 'Thou shalt not pass.'\r\n"

	t.Run("mortal moving down is blocked with C's three lines", func(t *testing.T) {
		w, out, mover, blocker := setup(t)
		if !specNoMoveDown(w, mover, blocker, "down", "") {
			t.Fatal("specNoMoveDown = false, want true")
		}
		if got := out["Mover"]; !strings.Contains(got, toVict) || !strings.Contains(got, toRoom) {
			t.Fatalf("Mover = %q, want %q and %q", got, toVict, toRoom)
		}
		if strings.Contains(out["Mover"], toNotVict) {
			t.Fatalf("Mover = %q, TO_NOTVICT line must skip the victim", out["Mover"])
		}
		if got := out["Watcher"]; !strings.Contains(got, toNotVict) || !strings.Contains(got, toRoom) {
			t.Fatalf("Watcher = %q, want %q and %q", got, toNotVict, toRoom)
		}
		if strings.Contains(out["Watcher"], toVict) {
			t.Fatalf("Watcher = %q, TO_VICT line must be private", out["Watcher"])
		}
	})

	t.Run("immort walks down", func(t *testing.T) {
		w, out, mover, blocker := setup(t)
		mover.SetLevel(combat.LVL_IMMORT)
		if specNoMoveDown(w, mover, blocker, "down", "") {
			t.Fatal("specNoMoveDown blocked an immort")
		}
		if got := out["Mover"] + out["Watcher"]; got != "" {
			t.Fatalf("immort move emitted %q, want silence", got)
		}
	})

	t.Run("other moves and the pulse pass", func(t *testing.T) {
		w, _, mover, blocker := setup(t)
		if specNoMoveDown(w, mover, blocker, "north", "") {
			t.Fatal("specNoMoveDown blocked a north move")
		}
		if specNoMoveDown(w, mover, blocker, "", "") {
			t.Fatal("specNoMoveDown acted on the pulse")
		}
	})

	t.Run("sleeping guard lets the mover through", func(t *testing.T) {
		w, out, mover, blocker := setup(t)
		blocker.SetPosition(combat.PosSleeping)
		if specNoMoveDown(w, mover, blocker, "down", "") {
			t.Fatal("specNoMoveDown blocked while the guard slept")
		}
		if got := out["Mover"] + out["Watcher"]; got != "" {
			t.Fatalf("sleeping guard emitted %q, want silence", got)
		}
	})
}

// fieldObjStandIn builds the SpecFunc-lane stand-in mobile the dispatch
// never delivers in production (the wrapper passes me == nil there): it
// carries the field object's vnum and room, exactly as C's `me` carries
// the object (src/spec_procs3.c:460). It is not a room occupant, so the
// victim walk never treats it as a body.
func fieldObjStandIn(vnum, roomVNum int) *MobInstance {
	return NewMob(&parser.Mob{
		VNum: vnum, Keywords: "field", ShortDesc: "a field", Sex: 0,
		Race: 1, Level: 20, Position: 8, DefaultPos: 8,
		HP: parser.DiceRoll{Plus: 100},
	}, roomVNum)
}

func fieldObjProto(vnum int, values [4]int, actionDesc string) parser.Obj {
	return parser.Obj{
		VNum: vnum, Keywords: "field", ShortDesc: "a field", LongDesc: "a field",
		ActionDesc: actionDesc, Values: values,
	}
}

// TestFieldObjectDamageDepth pins the FO_DAMAGE arm of
// SPECIAL(field_object) (src/spec_procs3.c:456-513): one dice(VAL0,
// VAL1) roll applied to every occupant of the room (players and mobiles
// alike), the action description through Act TO_VICT (else stc, to `ch`
// only), and the kill at HP <= 0 with the TO_NOTVICT death line. The old
// body invented damage from the stand-in's level, sent the stc line to
// the victim, ran update_pos's wounded band, killed only at POS_DEAD,
// and pre-rendered the death line to the whole room.
func TestFieldObjectDamageDepth(t *testing.T) {
	const obj50 = 50
	const descLine = "The wall sears your flesh!\r\n"
	const stcLine = "An incredible force hits you!\r\n"
	const fallenLine = "Victim falls to the ground, screaming in agony!\r\n"

	t.Run("one dice roll hits every occupant", func(t *testing.T) {
		w, out := roomMessageWorld(t, parser.World{
			Mobs: []parser.Mob{
				{
					VNum: 9803, Keywords: "goblin", ShortDesc: "a goblin", Sex: 0,
					Race: 1, Level: 20, Position: 8, DefaultPos: 8, HP: parser.DiceRoll{Plus: 50},
				},
			},
			Objs: []parser.Obj{fieldObjProto(obj50, [4]int{2, 6, 0, 0}, "The wall sears your flesh!")},
		})
		victim := addRoomPlayer(t, w, "Victim")
		victim.SetHP(50)
		issuer := addRoomPlayer(t, w, "Issuer")
		issuer.SetHP(50)
		goblin, err := w.SpawnMob(9803, 1001)
		if err != nil {
			t.Fatalf("SpawnMob(9803): %v", err)
		}
		floorObj, err := w.SpawnObject(obj50, 1001)
		if err != nil {
			t.Fatalf("SpawnObject(50): %v", err)
		}
		w.AddItemToRoom(floorObj, 1001)
		me := fieldObjStandIn(obj50, 1001)
		dprng.ResetStream(seedFirstDice(t, 2, 6, 7))
		if !specFieldObject(w, issuer, me, "", "") {
			t.Fatal("specFieldObject = false, want true")
		}
		if got := out["Victim"]; !strings.Contains(got, descLine) {
			t.Fatalf("Victim = %q, want action description %q", got, descLine)
		}
		if victim.GetHP() != 43 {
			t.Fatalf("victim HP = %d, want 43 (one 2d6 roll of 7)", victim.GetHP())
		}
		// The issuer stands in the same room and takes the same roll, in
		// C's world[room].people (src/spec_procs3.c:484).
		if issuer.GetHP() != 43 {
			t.Fatalf("issuer HP = %d, want 43 (same roll, same room)", issuer.GetHP())
		}
		if got := goblin.GetHP(); got != 43 {
			t.Fatalf("mob HP = %d, want 43: mobiles are people too", got)
		}
		if strings.Contains(out["Issuer"], stcLine) {
			t.Fatalf("Issuer = %q, stc must not fire when the object has a description", out["Issuer"])
		}
	})

	t.Run("no description means stc to the issuer only", func(t *testing.T) {
		w, out := roomMessageWorld(t, parser.World{Objs: []parser.Obj{
			fieldObjProto(obj50, [4]int{2, 6, 0, 0}, ""),
		}})
		victim := addRoomPlayer(t, w, "Victim")
		victim.SetHP(50)
		issuer := addRoomPlayer(t, w, "Issuer")
		issuer.SetHP(50)
		floorObj, err := w.SpawnObject(obj50, 1001)
		if err != nil {
			t.Fatalf("SpawnObject(50): %v", err)
		}
		w.AddItemToRoom(floorObj, 1001)
		me := fieldObjStandIn(obj50, 1001)
		dprng.ResetStream(seedFirstDice(t, 2, 6, 7))
		if !specFieldObject(w, issuer, me, "", "") {
			t.Fatal("specFieldObject = false, want true")
		}
		if got := out["Issuer"]; !strings.Contains(got, stcLine) {
			t.Fatalf("Issuer = %q, want stc line %q on C's own target (ch)", got, stcLine)
		}
		if strings.Contains(out["Victim"], stcLine) {
			t.Fatalf("Victim = %q, stc must not go to the victim", out["Victim"])
		}
		if victim.GetHP() != 43 {
			t.Fatalf("victim HP = %d, want 43", victim.GetHP())
		}
	})

	t.Run("zero HP kills with TO_NOTVICT line", func(t *testing.T) {
		w, out := roomMessageWorld(t, parser.World{Objs: []parser.Obj{
			fieldObjProto(obj50, [4]int{2, 6, 0, 0}, ""),
		}})
		victim := addRoomPlayer(t, w, "Victim")
		victim.SetHP(3)
		issuer := addRoomPlayer(t, w, "Issuer")
		addRoomPlayer(t, w, "Watcher")
		floorObj, err := w.SpawnObject(obj50, 1001)
		if err != nil {
			t.Fatalf("SpawnObject(50): %v", err)
		}
		w.AddItemToRoom(floorObj, 1001)
		me := fieldObjStandIn(obj50, 1001)
		dprng.ResetStream(seedFirstDice(t, 2, 6, 7))
		if !specFieldObject(w, issuer, me, "", "") {
			t.Fatal("specFieldObject = false, want true")
		}
		if victim.GetHP() > 0 {
			t.Fatalf("victim HP = %d, want <= 0 after a 7-damage hit on 3", victim.GetHP())
		}
		if got := out["Watcher"]; !strings.Contains(got, fallenLine) {
			t.Fatalf("Watcher = %q, want TO_NOTVICT line %q", got, fallenLine)
		}
		if strings.Contains(out["Victim"], fallenLine) {
			t.Fatalf("Victim = %q, death line must skip the victim", out["Victim"])
		}
		if got := out["Issuer"]; !strings.Contains(got, stcLine) {
			t.Fatalf("Issuer = %q, want stc line %q", got, stcLine)
		}
	})

	t.Run("fire wall in production is inert and draws nothing", func(t *testing.T) {
		// The real obj 50 carries VALs of 0 0 0 0 (lib/world/obj/0.obj),
		// so dice(0,0) is a zero-consumption return (src/utils.c:66-77)
		// and the arm is inert even if reached.
		w, _ := roomMessageWorld(t, parser.World{Objs: []parser.Obj{
			fieldObjProto(obj50, [4]int{0, 0, 0, 0}, ""),
		}})
		victim := addRoomPlayer(t, w, "Victim")
		issuer := addRoomPlayer(t, w, "Issuer")
		floorObj, err := w.SpawnObject(obj50, 1001)
		if err != nil {
			t.Fatalf("SpawnObject(50): %v", err)
		}
		w.AddItemToRoom(floorObj, 1001)
		me := fieldObjStandIn(obj50, 1001)
		dprng.ResetStream(7)
		want := dprng.Number(0, 100)
		dprng.ResetStream(7)
		if specFieldObject(w, issuer, me, "", "") {
			t.Fatal("zero-val field object reported damage")
		}
		if victim.GetHP() != 100 {
			t.Fatalf("victim HP = %d, want 100", victim.GetHP())
		}
		if got := dprng.Number(0, 100); got != want {
			t.Fatalf("stream next draw = %d, want %d: dice(0,0) must consume nothing", got, want)
		}
	})
}

// TestFieldObjectAffectDepth pins the FO_AFFECT arm (src/spec_procs3.c:
// 503-510): the action description (if any) through Act TO_VICT, then
// call_magic(vict, vict, 0, affect, level, CAST_SPELL). The old body
// cast a hardcoded poison at the stand-in's level.
func TestFieldObjectAffectDepth(t *testing.T) {
	const obj52 = 52
	w, out := roomMessageWorld(t, parser.World{Objs: []parser.Obj{
		fieldObjProto(obj52, [4]int{spells.SpellPoison, 30, 0, 0}, "The gas burns your eyes!"),
	}})
	victim := addRoomPlayer(t, w, "Victim")
	issuer := addRoomPlayer(t, w, "Issuer")
	floorObj, err := w.SpawnObject(obj52, 1001)
	if err != nil {
		t.Fatalf("SpawnObject(52): %v", err)
	}
	w.AddItemToRoom(floorObj, 1001)
	me := fieldObjStandIn(obj52, 1001)
	if !specFieldObject(w, issuer, me, "", "") {
		t.Fatal("specFieldObject = false, want true")
	}
	const line = "The gas burns your eyes!\r\n"
	if got := out["Victim"]; !strings.Contains(got, line) {
		t.Fatalf("Victim = %q, want action description %q", got, line)
	}
	if !victim.HasSpellAffect(spells.SpellPoison) {
		t.Fatal("victim lacks the poison call_magic cast (VAL0 spell, VAL1 level)")
	}
	// The affect arm has no stc branch (src/spec_procs3.c:503-510): the
	// only line on a no-description object is the one line the damage arm
	// owns, so it must never appear here.
	if strings.Contains(out["Issuer"], "An incredible force hits you!") {
		t.Fatalf("Issuer = %q, stc is damage-arm only", out["Issuer"])
	}
	if !issuer.HasSpellAffect(spells.SpellPoison) {
		t.Fatal("issuer lacks poison: every room occupant is walked, issuer included")
	}
	if specFieldObject(w, issuer, me, "look", "") {
		t.Fatal("field_object acted on a typed command")
	}
}
