package game

// room_activity_depth_test.go — R5h proofs for the three room_activity arms
// ported in DP-1426: the CON<=0 croak (comm.c:748-754), loud_mobs
// (spec_procs2.c:1532-1550) and flow_room (new_cmds2.c:885-955, gated at
// comm.c:741-742). Every test fails when its arm is reverted: the croak tests
// assert the death bytes and the HP-1 re-entry state, the loud_mobs tests
// assert the number(0,20) draw itself, and the flow tests assert the
// number(0,1) draw and the drift bytes.

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// drawRecorder is a Roller that scripts Number results and records every
// draw's bounds, so tests can assert both the value fed to a gate and that a
// gate burned exactly the draws C's code path burns.
type drawRecorder struct {
	values []int
	calls  [][2]int
	idx    int
}

func (r *drawRecorder) Number(from, to int) int {
	r.calls = append(r.calls, [2]int{from, to})
	v := 0
	if r.idx < len(r.values) {
		v = r.values[r.idx]
		r.idx++
	}
	return v
}

func (r *drawRecorder) Dice(num, size int) int {
	sum := 0
	for i := 0; i < num; i++ {
		sum += r.Number(1, size)
	}
	return sum
}

func (r *drawRecorder) IntN(n int) int { return r.Number(0, n-1) }

func (r *drawRecorder) count() int { return len(r.calls) }

func (r *drawRecorder) lastBounds() [2]int {
	if len(r.calls) == 0 {
		return [2]int{}
	}
	return r.calls[len(r.calls)-1]
}

// installDrawRecorder swaps the global roller for a scripted recorder and
// returns it for draw assertions.
func installDrawRecorder(t *testing.T, values ...int) *drawRecorder {
	t.Helper()
	r := &drawRecorder{values: values}
	old := combat.GetRoller()
	combat.SetRoller(r)
	t.Cleanup(func() { combat.SetRoller(old) })
	return r
}

// croakTestWorld builds a two-room world (1001 north of 1002) with wired
// combat callbacks and a message sink capturing per-player output.
func croakTestWorld(t *testing.T) (*World, map[string]string) {
	t.Helper()
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{
		{VNum: 1001, Name: "The shrine", Exits: map[string]parser.Exit{
			"north": {Direction: "north", ToRoom: 1002},
		}},
		{VNum: 1002, Name: "The approach", Exits: map[string]parser.Exit{
			"south": {Direction: "south", ToRoom: 1001},
		}},
	}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	w.StopAITicker()
	t.Cleanup(w.StopAITicker)
	out := map[string]string{}
	w.MessageSink = func(name string, msg []byte) { out[name] += string(msg) }
	old := combat.GetCallbacks()
	t.Cleanup(func() { combat.SetCallbacks(old) })
	cb := w.WireCombatCallbacks()
	cb.Broadcast = w.WoundBroadcast
	combat.SetCallbacks(cb)
	return w, out
}

func croakPlayer(t *testing.T, w *World, name string, room int) *Player {
	t.Helper()
	p := NewPlayer(len(name), name, room)
	p.Stats.Con = 11
	p.CopyBaseAttributes()
	p.SetGold(50)
	if err := w.AddPlayer(p); err != nil {
		t.Fatalf("AddPlayer(%s): %v", name, err)
	}
	return p
}

// The croak kills a CON<=0 mortal in place: the victim line, the room act,
// the death cry (own and adjacent rooms), an in-place default-arm corpse
// carrying everything, HP pinned to 1, and extraction queued for the next
// heartbeat.
func TestRoomActivityConCroakKillsMortalInPlace(t *testing.T) {
	w, out := croakTestWorld(t)
	victim := croakPlayer(t, w, "Croaker", 1001)
	croakPlayer(t, w, "Watcher", 1001)
	croakPlayer(t, w, "Hearer", 1002)
	vest := NewObjectInstance(&parser.Obj{VNum: 14425, Keywords: "vest", WearFlags: [4]int{9}, Weight: 1, Affects: []parser.ObjAffect{{Location: ApplyCon, Modifier: -12}}}, -1)
	if err := victim.Equipment.Equip(vest, victim.Inventory); err != nil {
		t.Fatalf("equip vest: %v", err)
	}
	if got := victim.GetCon(); got != 0 {
		t.Fatalf("effective CON = %d, want 0 (11 base - 12 gear, clamped at the attribute floor)", got)
	}

	r := installDrawRecorder(t)
	w.RoomActivity()

	if got, want := out["Croaker"], "You are too unhealthy to live!\r\n"; !strings.Contains(got, want) {
		t.Fatalf("victim bytes = %q, want %q among them", got, want)
	}
	if strings.Contains(out["Croaker"], "You are dead!") {
		t.Fatal("direct raw_kill must not emit damage()'s You are dead! line")
	}
	if strings.Contains(out["Croaker"], "death cry") {
		t.Fatal("death cry is TO_ROOM; the victim must not hear their own cry")
	}
	if got, want := out["Watcher"], "Suddenly, Croaker croaks before your very eyes!\r\nYour blood freezes as you hear Croaker's death cry.\r\n"; !strings.Contains(got, want) {
		t.Fatalf("room bytes = %q, want %q in order (croak act then death cry)", got, want)
	}
	if got, want := out["Hearer"], "Your blood freezes as you hear someone's death cry.\r\n"; !strings.Contains(got, want) {
		t.Fatalf("adjacent room bytes = %q, want %q", got, want)
	}

	if got := victim.GetHP(); got != 1 {
		t.Fatalf("HP after croak = %d, want 1 (GET_HIT=1 before raw_kill)", got)
	}
	if victim.GetFlags()&(1<<uint(plrExtractBit)) == 0 {
		t.Fatal("croak did not queue extraction for the next heartbeat")
	}
	if len(victim.Equipment.GetEquippedItems()) != 0 {
		t.Fatal("equipment not stripped into the corpse")
	}
	items := w.GetItemsInRoom(1001)
	if len(items) != 1 || !items[0].IsCorpse {
		t.Fatalf("room holds %#v, want exactly the corpse", items)
	}
	corpse := items[0]
	if got, want := corpse.Runtime.LongDesc, "The corpse of Croaker is lying here."; got != want {
		t.Fatalf("corpse description = %q, want default-arm %q (attacktype 0)", got, want)
	}
	if !strings.Contains(corpse.Runtime.Keywords, "Croaker corpse") {
		t.Fatalf("corpse keywords = %q", corpse.Runtime.Keywords)
	}
	foundVest, foundGold := false, false
	for _, c := range corpse.Contains {
		if c == vest {
			foundVest = true
		}
		if c.CustomData["is_money"] == true {
			foundGold = true
		}
	}
	if !foundVest || !foundGold {
		t.Fatalf("corpse contents missing gear/gold: vest=%t gold=%t", foundVest, foundGold)
	}
	if got := victim.GetGold(); got != 0 {
		t.Fatalf("gold after croak = %d, want 0 (moved to corpse)", got)
	}
	if r.count() != 0 {
		t.Fatalf("croak pulse drew %v, want zero draws (raw_kill path is draw-free)", r.calls)
	}
}

// GET_CON is affect_total's result: a healthy base constitution with enough
// -CON gear still croaks, and CON at exactly 1 does not.
func TestRoomActivityConCroakUsesEffectiveCon(t *testing.T) {
	t.Run("gear drives con to zero", func(t *testing.T) {
		w, out := croakTestWorld(t)
		p := croakPlayer(t, w, "Stacker", 1001)
		p.Stats.Con = 3
		p.CopyBaseAttributes()
		vest := NewObjectInstance(&parser.Obj{VNum: 14426, Keywords: "rags", WearFlags: [4]int{9}, Weight: 1, Affects: []parser.ObjAffect{{Location: ApplyCon, Modifier: -3}}}, -1)
		if err := p.Equipment.Equip(vest, p.Inventory); err != nil {
			t.Fatal(err)
		}
		if got := p.GetCon(); got != 0 {
			t.Fatalf("effective CON before the pulse = %d, want 0", got)
		}
		w.RoomActivity()
		if !strings.Contains(out["Stacker"], "too unhealthy") {
			t.Fatal("CON 0 (the boundary) must croak: GET_CON <= 0")
		}
		if p.GetFlags()&(1<<uint(plrExtractBit)) == 0 {
			t.Fatal("boundary croak did not queue extraction")
		}
	})
	t.Run("con one survives", func(t *testing.T) {
		w, out := croakTestWorld(t)
		p := croakPlayer(t, w, "Hale", 1001)
		p.Stats.Con = 1
		p.CopyBaseAttributes()
		w.RoomActivity()
		if out["Hale"] != "" {
			t.Fatalf("CON 1 player saw %q, want nothing", out["Hale"])
		}
		if p.GetFlags()&(1<<uint(plrExtractBit)) != 0 {
			t.Fatal("CON 1 player was extracted")
		}
	})
}

// The croak's gates: immortals and NPCs are exempt even at CON<=0.
func TestRoomActivityConCroakControls(t *testing.T) {
	t.Run("immortal exempt", func(t *testing.T) {
		w, out := croakTestWorld(t)
		p := croakPlayer(t, w, "Godly", 1001)
		p.SetLevel(LVL_IMMORT)
		p.Stats.Con = 0
		p.CopyBaseAttributes()
		w.RoomActivity()
		if out["Godly"] != "" || p.GetFlags()&(1<<uint(plrExtractBit)) != 0 {
			t.Fatalf("immortal croaked: out=%q extracted=%t", out["Godly"], p.GetFlags()&(1<<uint(plrExtractBit)) != 0)
		}
	})
	t.Run("npc exempt", func(t *testing.T) {
		w, err := NewWorld(&parser.World{
			Rooms: []parser.Room{{VNum: 1001, Name: "Den"}},
			Mobs:  []parser.Mob{{VNum: 1, Keywords: "frail wraith", ShortDesc: "a frail wraith", Con: -2}},
		})
		if err != nil {
			t.Fatalf("NewWorld: %v", err)
		}
		w.StopAITicker()
		t.Cleanup(w.StopAITicker)
		out := map[string]string{}
		w.MessageSink = func(name string, msg []byte) { out[name] += string(msg) }
		m, err := w.SpawnMob(1, 1001)
		if err != nil {
			t.Fatalf("SpawnMob: %v", err)
		}
		w.RoomActivity()
		if !m.IsAlive() {
			t.Fatal("NPC with CON<=0 died; the croak is !IS_NPC")
		}
		for _, v := range out {
			if strings.Contains(v, "croaks") || strings.Contains(v, "unhealthy") {
				t.Fatalf("NPC croak leaked bytes: %q", v)
			}
		}
	})
}

// loud_mobs: the number(0,20) draw happens for every noise-carrying mob on
// every pulse; a 0 sends the noise line to each adjacent room (never the
// mob's own room), any other value sends nothing but still burns the draw.
func TestRoomActivityLoudMobs(t *testing.T) {
	build := func(t *testing.T, noise string) (*World, map[string]string) {
		t.Helper()
		w, err := NewWorld(&parser.World{
			Rooms: []parser.Room{
				{VNum: 1001, Name: "Source", Exits: map[string]parser.Exit{
					"north": {Direction: "north", ToRoom: 1002},
					"east":  {Direction: "east", ToRoom: 1003},
				}},
				{VNum: 1002, Name: "North side", Exits: map[string]parser.Exit{"south": {Direction: "south", ToRoom: 1001}}},
				{VNum: 1003, Name: "East side", Exits: map[string]parser.Exit{"west": {Direction: "west", ToRoom: 1001}}},
			},
			Mobs: []parser.Mob{{VNum: 1, Keywords: "noisy sprite", ShortDesc: "a noisy sprite", Noise: noise, Con: 11}},
		})
		if err != nil {
			t.Fatalf("NewWorld: %v", err)
		}
		w.StopAITicker()
		t.Cleanup(w.StopAITicker)
		out := map[string]string{}
		w.MessageSink = func(name string, msg []byte) { out[name] += string(msg) }
		return w, out
	}

	t.Run("a closed door keeps the noise in", func(t *testing.T) {
		// CAN_GO (src/utils.h:565-567) refuses an EX_CLOSED exit.
		w, out := build(t, "You hear a sprite humming.")
		if _, err := w.SpawnMob(1, 1001); err != nil {
			t.Fatal(err)
		}
		room := w.GetRoomInWorld(1001)
		ext := room.Exits["east"]
		ext.ExitInfo |= parser.ExitIsDoor | parser.ExitClosed
		room.Exits["east"] = ext
		north := NewPlayer(1, "Northwatch", 1002)
		east := NewPlayer(3, "Eastwatch", 1003)
		for _, p := range []*Player{north, east} {
			p.Stats.Con = 11
			p.CopyBaseAttributes()
			if err := w.AddPlayer(p); err != nil {
				t.Fatal(err)
			}
		}
		installDrawRecorder(t, 0)
		w.RoomActivity()
		if got, want := out["Northwatch"], "You hear a sprite humming.\r\n"; got != want {
			t.Fatalf("open exit heard %q, want %q", got, want)
		}
		if got := out["Eastwatch"]; got != "" {
			t.Fatalf("noise crossed a closed door: %q", got)
		}
	})

	t.Run("draw fires and noise reaches adjacent rooms only", func(t *testing.T) {
		w, out := build(t, "You hear a sprite humming.")
		if _, err := w.SpawnMob(1, 1001); err != nil {
			t.Fatal(err)
		}
		north := NewPlayer(1, "Northwatch", 1002)
		same := NewPlayer(2, "Sameroom", 1001)
		east := NewPlayer(3, "Eastwatch", 1003)
		for _, p := range []*Player{north, same, east} {
			p.Stats.Con = 11
			p.CopyBaseAttributes()
			if err := w.AddPlayer(p); err != nil {
				t.Fatal(err)
			}
		}
		r := installDrawRecorder(t, 0)
		w.RoomActivity()
		if r.count() != 1 || r.lastBounds() != [2]int{0, 20} {
			t.Fatalf("draws = %v, want exactly one number(0,20)", r.calls)
		}
		want := "You hear a sprite humming.\r\n"
		if got := out["Northwatch"]; got != want {
			t.Fatalf("north room heard %q, want %q", got, want)
		}
		if got := out["Eastwatch"]; got != want {
			t.Fatalf("east room heard %q, want %q", got, want)
		}
		if got := out["Sameroom"]; got != "" {
			t.Fatalf("mob's own room heard %q, want nothing (act redirects to adjacent rooms)", got)
		}
	})
	t.Run("nonzero roll stays silent but burns the draw", func(t *testing.T) {
		w, out := build(t, "You hear a sprite humming.")
		if _, err := w.SpawnMob(1, 1001); err != nil {
			t.Fatal(err)
		}
		r := installDrawRecorder(t, 7)
		w.RoomActivity()
		if r.count() != 1 || r.lastBounds() != [2]int{0, 20} {
			t.Fatalf("draws = %v, want exactly one number(0,20)", r.calls)
		}
		for name, v := range out {
			if v != "" {
				t.Fatalf("%s heard %q on a nonzero roll", name, v)
			}
		}
	})
	t.Run("silent mob draws nothing", func(t *testing.T) {
		w, _ := build(t, "")
		if _, err := w.SpawnMob(1, 1001); err != nil {
			t.Fatal(err)
		}
		r := installDrawRecorder(t)
		w.RoomActivity()
		if r.count() != 0 {
			t.Fatalf("noiseless mob drew %v, want none", r.calls)
		}
	})
}

// flow_room: a ROOM_FLOW_* room burns one number(0,1) draw per pulse for
// every player in it; a 0 drifts the player downstream with C's byte
// sequence, and the immortal+nohassle exemption lives inside flow_room, after
// the draw.
func TestRoomActivityFlowRoom(t *testing.T) {
	build := func(t *testing.T) *World {
		t.Helper()
		w, err := NewWorld(&parser.World{Rooms: []parser.Room{
			{
				VNum: 1001, Name: "River", Flags: []string{"2097152", "0", "0", "0"},
				Exits: map[string]parser.Exit{"north": {Direction: "north", ToRoom: 1002}},
			},
			{VNum: 1002, Name: "Downstream", Exits: map[string]parser.Exit{"south": {Direction: "south", ToRoom: 1001}}},
		}})
		if err != nil {
			t.Fatalf("NewWorld: %v", err)
		}
		w.StopAITicker()
		t.Cleanup(w.StopAITicker)
		return w
	}

	t.Run("zero roll drifts downstream", func(t *testing.T) {
		w := build(t)
		out := map[string]string{}
		w.MessageSink = func(name string, msg []byte) { out[name] += string(msg) }
		drifted := NewPlayer(1, "Swimmer", 1001)
		banker := NewPlayer(2, "Banker", 1001)
		downstream := NewPlayer(3, "Fisher", 1002)
		for _, p := range []*Player{drifted, banker, downstream} {
			p.Stats.Con = 11
			p.CopyBaseAttributes()
			if err := w.AddPlayer(p); err != nil {
				t.Fatal(err)
			}
		}
		// Players walk in name order: Banker's roll (1) leaves it on the
		// bank; Swimmer's roll (0) drifts it downstream.
		r := installDrawRecorder(t, 1, 0)
		w.RoomActivity()
		if r.count() != 2 {
			t.Fatalf("draws = %v, want one number(0,1) per player in the flow room", r.calls)
		}
		for _, c := range r.calls {
			if c != [2]int{0, 1} {
				t.Fatalf("draw bounds = %v, want number(0,1)", r.calls)
			}
		}
		if got := drifted.GetRoom(); got != 1002 {
			t.Fatalf("drifted player in room %d, want 1002", got)
		}
		if !strings.HasPrefix(out["Swimmer"], "You drift north.\r\n") {
			t.Fatalf("drifter bytes = %q, want the drift line before the room look", out["Swimmer"])
		}
		if !strings.Contains(out["Swimmer"], "Downstream") {
			t.Fatalf("drifter did not look at the new room: %q", out["Swimmer"])
		}
		if got, want := out["Banker"], "Swimmer drifts north.\r\n"; !strings.Contains(got, want) {
			t.Fatalf("old room bytes = %q, want %q", got, want)
		}
		if got, want := out["Fisher"], "Swimmer drifts in from the south.\r\n"; !strings.Contains(got, want) {
			t.Fatalf("new room bytes = %q, want %q", got, want)
		}
		if strings.Contains(out["Swimmer"], "drifts in from") {
			t.Fatal("TO_NOTVICT drift-in line leaked to the drifter")
		}
	})
	t.Run("one roll stays put", func(t *testing.T) {
		w := build(t)
		out := map[string]string{}
		w.MessageSink = func(name string, msg []byte) { out[name] += string(msg) }
		p := NewPlayer(1, "Swimmer", 1001)
		p.Stats.Con = 11
		p.CopyBaseAttributes()
		if err := w.AddPlayer(p); err != nil {
			t.Fatal(err)
		}
		r := installDrawRecorder(t, 1)
		w.RoomActivity()
		if r.count() != 1 || r.lastBounds() != [2]int{0, 1} {
			t.Fatalf("draws = %v, want exactly one number(0,1)", r.calls)
		}
		if p.GetRoom() != 1001 || out["Swimmer"] != "" {
			t.Fatalf("player moved/heard bytes on a nonzero roll: room=%d out=%q", p.GetRoom(), out["Swimmer"])
		}
	})
	t.Run("plain room draws nothing", func(t *testing.T) {
		w, err := NewWorld(&parser.World{Rooms: []parser.Room{
			{VNum: 1001, Name: "Dry", Flags: []string{"0", "0", "0", "0"}},
		}})
		if err != nil {
			t.Fatal(err)
		}
		w.StopAITicker()
		t.Cleanup(w.StopAITicker)
		p := NewPlayer(1, "Wanderer", 1001)
		p.Stats.Con = 11
		p.CopyBaseAttributes()
		if err := w.AddPlayer(p); err != nil {
			t.Fatal(err)
		}
		r := installDrawRecorder(t)
		w.RoomActivity()
		if r.count() != 0 {
			t.Fatalf("non-flow room drew %v, want none", r.calls)
		}
	})
	t.Run("immortal nohassle exempt after the draw", func(t *testing.T) {
		w := build(t)
		p := NewPlayer(1, "Deity", 1001)
		p.SetLevel(LVL_IMMORT)
		p.Stats.Con = 11
		p.CopyBaseAttributes()
		p.SetPlrFlag(PrfNohassle, true)
		if err := w.AddPlayer(p); err != nil {
			t.Fatal(err)
		}
		r := installDrawRecorder(t, 0)
		w.RoomActivity()
		if r.count() != 1 {
			t.Fatalf("draws = %v, want the draw even for the exempt immortal", r.calls)
		}
		if p.GetRoom() != 1001 {
			t.Fatal("nohassle immortal drifted")
		}
	})
}
