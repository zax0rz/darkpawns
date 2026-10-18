package game

import (
	"strings"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// newHorseWorld builds the stable room, an outdoor corridor, a horse and a
// stableboy. Horse 8021 is the C stableboy's HORSE_VNUM
// (src/spec_procs2.c:315).
func newHorseWorld(t *testing.T, corridor int) (*World, *Player, *MobInstance) {
	t.Helper()
	rooms := make([]parser.Room, 0, corridor+1)
	rooms = append(rooms, parser.Room{
		VNum: 1000, Name: "Stable", Description: "A stable.\n",
		Exits: map[string]parser.Exit{"north": {ToRoom: 1001}},
	})
	for i := 1; i <= corridor; i++ {
		room := parser.Room{
			VNum: 1000 + i, Name: "Field", Description: "A field.\n",
			Exits:  map[string]parser.Exit{},
			Sector: SECT_FIELD,
		}
		if i < corridor {
			room.Exits["north"] = parser.Exit{ToRoom: 1000 + i + 1}
		}
		if i > 1 {
			room.Exits["south"] = parser.Exit{ToRoom: 1000 + i - 1}
		}
		rooms = append(rooms, room)
	}
	w, err := NewWorld(&parser.World{
		Rooms: rooms,
		Mobs: []parser.Mob{
			{
				VNum: horseVnum, Keywords: "horse", ShortDesc: "a horse",
				LongDesc: "A horse is here, grazing.\n", Level: 5,
				Position: combat.PosStanding, DefaultPos: combat.PosStanding,
				ActionFlags: []string{"mountable"},
			},
			{
				VNum: 8100, Keywords: "stableboy", ShortDesc: "the stableboy",
				LongDesc: "A stableboy stands here.\n", Level: 10,
				Position: combat.PosStanding, DefaultPos: combat.PosStanding,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	w.StopAITicker()
	t.Cleanup(w.StopAITicker)

	p := NewPlayer(1, "Egon", 1000)
	p.SetPosition(combat.PosStanding)
	p.SetMove(100)
	p.SetStat("CHA", 20)
	p.CopyBaseAttributes()
	if err := w.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	boy, err := w.SpawnMob(8100, 1000)
	if err != nil {
		t.Fatal(err)
	}
	boy.SetPosition(combat.PosStanding)
	return w, p, boy
}

func horseInRoom(t *testing.T, w *World, room int) *MobInstance {
	t.Helper()
	for _, m := range w.GetMobsInRoom(room) {
		if m.VNum == horseVnum {
			return m
		}
	}
	t.Fatalf("no horse in room %d", room)
	return nil
}

// TestMountedTravelSpendsTheMountPool reproduces the reported route class with
// C's arithmetic: need = avg(sector loss of source and destination)
// (src/act.movement.c:137-141), the gate reads GET_MOVE(get_mount(ch))
// (:163), and the spend hits GET_MOVE(mount) (:202). A 230-point stable horse
// therefore covers 76 field steps (3 points each), not the 16 a read_mobile
// 50-point pool allows.
func TestMountedTravelSpendsTheMountPool(t *testing.T) {
	const steps = 200
	w, rider, _ := newHorseWorld(t, steps)
	w.GetRoomInWorld(1000).Sector = SECT_FIELD
	horse, err := w.SpawnMob(horseVnum, 1000)
	if err != nil {
		t.Fatal(err)
	}
	horse.SetPosition(combat.PosStanding)
	applyStableHorseState(horse)
	if !w.doRide(rider, nil, "ride", "horse") {
		t.Fatal("ride was not handled")
	}
	if !rider.IsMounted() || !horse.IsAffected(affMounted) {
		t.Fatal("ride did not establish the mounted pair")
	}

	var out []string
	w.MessageSink = func(name string, message []byte) {
		if name == rider.GetName() {
			out = append(out, string(message))
		}
	}

	moved := 0
	for i := 0; i < steps; i++ {
		if !w.DoMove(rider, "north").Success {
			break
		}
		moved++
	}
	if moved != 76 {
		t.Fatalf("mounted field steps = %d, want 76 (230/3)", moved)
	}
	if got, want := horse.GetMove(), 2; got != want {
		t.Errorf("mount pool after 76 steps = %d, want %d", got, want)
	}
	if got, want := rider.GetMove(), 100; got != want {
		t.Errorf("rider pool = %d, want %d (the mount pays)", got, want)
	}
	if !w.DoMove(rider, "north").Success {
		// expected exhaustion
	} else {
		t.Fatal("a 77th step succeeded with 2 mount move points left")
	}
	want := "Your mount is too exhausted to carry you further.\r\n"
	if len(out) == 0 || out[len(out)-1] != want {
		t.Fatalf("exhaustion line = %q, want %q", out, want)
	}
}

// TestHorseAppearsInRoomLook pins C list_char_to_char's default-position branch
// (src/act.informative.c:511-526): a standing horse with a long description
// prints it in look.
func TestHorseAppearsInRoomLook(t *testing.T) {
	w, rider, _ := newHorseWorld(t, 1)
	if _, err := w.SpawnMob(horseVnum, 1000); err != nil {
		t.Fatal(err)
	}
	assertLookContains(t, w, rider, "A horse is here, grazing.")
}

// TestHorseReappearsInLookAfterDismount pins the DP-1453 room-description
// defect: C suppresses a mount only while get_rider() is non-nil, and
// get_rider() is IS_MOUNTED(mount) (src/utils.c:387-392;
// src/act.informative.c:635). Go tested the cached MountRider name instead,
// which survives unmount(), so a horse that had been ridden vanished from look
// for the rest of its life.
func TestHorseReappearsInLookAfterDismount(t *testing.T) {
	w, rider, _ := newHorseWorld(t, 1)
	w.GetRoomInWorld(1000).Sector = SECT_FIELD
	horse, err := w.SpawnMob(horseVnum, 1000)
	if err != nil {
		t.Fatal(err)
	}
	horse.SetPosition(combat.PosStanding)
	if !w.doRide(rider, nil, "ride", "horse") {
		t.Fatal("ride was not handled")
	}
	w.doDismount(rider, nil, "dismount", "")
	if rider.IsMounted() {
		t.Fatal("rider is still mounted after dismount")
	}
	if horse.IsAffected(affMounted) {
		t.Fatal("horse still has AFF_MOUNT after dismount")
	}
	if got := horse.GetMountRider(); got != rider.GetName() {
		t.Fatalf("unmount cleared the master/rider link: %q", got)
	}
	assertLookContains(t, w, rider, "A horse is here, grazing.")
}

// TestHorseFollowsPlayerAndStaysVisible covers the other bookkeeping path the
// brief names: a horse moved by the follow loop must remain registered to the
// room the look renderer reads.
func TestHorseFollowsPlayerAndStaysVisible(t *testing.T) {
	w, rider, _ := newHorseWorld(t, 1)
	w.GetRoomInWorld(1000).Sector = SECT_FIELD
	horse, err := w.SpawnMob(horseVnum, 1000)
	if err != nil {
		t.Fatal(err)
	}
	horse.SetPosition(combat.PosStanding)
	horse.SetAffected(affCharm)
	AddFollowerMob(w, horse, rider)
	if !w.DoMove(rider, "north").Success {
		t.Fatal("rider could not move north")
	}
	if got, want := horse.GetRoom(), 1001; got != want {
		t.Fatalf("horse room = %d, want %d", got, want)
	}
	assertLookContains(t, w, rider, "A horse is here, grazing.")
}

// TestHorseVisibleToInfravisionAfterDismount covers the second get_rider
// suppression site, appendDarkOccupants (src/act.informative.c:651-658).
func TestHorseVisibleToInfravisionAfterDismount(t *testing.T) {
	w, rider, _ := newHorseWorld(t, 1)
	room := w.GetRoomInWorld(1000)
	room.Sector = SECT_FIELD
	room.Flags = []string{"1"} // ROOM_DARK
	horse, err := w.SpawnMob(horseVnum, 1000)
	if err != nil {
		t.Fatal(err)
	}
	horse.SetPosition(combat.PosStanding)
	if !w.doRide(rider, nil, "ride", "horse") {
		t.Fatal("ride was not handled")
	}
	w.doDismount(rider, nil, "dismount", "")
	// C's arm needs a dark room for the viewer and AFF_INFRAVISION on the mob
	// (src/act.informative.c:651-658); the rider stays without infravision.
	horse.SetAffected(affInfravision)
	assertLookContains(t, w, rider, "You see a pair of glowing red eyes.")
}

func assertLookContains(t *testing.T, w *World, viewer *Player, want string) {
	t.Helper()
	result := w.DoLookRoom(viewer, true)
	var got string
	for _, message := range result.Messages {
		got += message.Format
	}
	if !strings.Contains(got, want) {
		t.Fatalf("look bytes =\n%q\nwant substring %q", got, want)
	}
}

// (src/spec_procs2.c:344-360 buy, 403-419 collect). Go previously filed 230
// only in an unread Runtime.Horse record and left points.move at read_mobile's
// 50 (src/db.c:1073,1758), which is the reported "spent almost immediately".
func TestStableHorseGetsCMovementPool(t *testing.T) {
	t.Run("buy", func(t *testing.T) {
		w, p, boy := newHorseWorld(t, 1)
		p.SetGold(1000)
		if !specStableboy(w, p, boy, "buy", "horse") {
			t.Fatal("stableboy buy was not handled")
		}
		horse := horseInRoom(t, w, 1000)
		if got := horse.GetMove(); got != 230 {
			t.Errorf("bought horse GET_MOVE = %d, want 230", got)
		}
		if got := horse.GetMaxMove(); got != 230 {
			t.Errorf("bought horse GET_MAX_MOVE = %d, want 230", got)
		}
		if got := horse.GetExp(); got != 0 {
			t.Errorf("bought horse GET_EXP = %d, want 0", got)
		}
		if got := mobCarriedWeight(horse); got != 1000 {
			t.Errorf("bought horse IS_CARRYING_W = %d, want 1000", got)
		}
		if got := mobCarriedItemCount(horse); got != 100 {
			t.Errorf("bought horse IS_CARRYING_N = %d, want 100", got)
		}
	})

	t.Run("collect", func(t *testing.T) {
		w, p, boy := newHorseWorld(t, 1)
		p.SetGold(1000)
		p.MountVNum = horseVnum
		p.MountRentTime = time.Now().Unix()
		p.MountCostDay = 5
		if !specStableboy(w, p, boy, "collect", "") {
			t.Fatal("stableboy collect was not handled")
		}
		horse := horseInRoom(t, w, 1000)
		if got := horse.GetMove(); got != 230 {
			t.Errorf("collected horse GET_MOVE = %d, want 230", got)
		}
		if got := horse.GetMaxMove(); got != 230 {
			t.Errorf("collected horse GET_MAX_MOVE = %d, want 230", got)
		}
	})
}

// TestMobCreationPathsInheritReadMobileMove is the R5b class audit for the
// "creation path starts at zero/partial MV" hypothesis: every MobInstance in
// pkg/ is built by NewMob, and every world allocator delegates to World.spawnMob,
// which copies C's read_mobile move = max_move (src/db.c:1758).
func TestMobCreationPathsInheritReadMobileMove(t *testing.T) {
	w, _, _ := newHorseWorld(t, 1)
	paths := map[string]func() (*MobInstance, error){
		"World.SpawnMob":      func() (*MobInstance, error) { return w.SpawnMob(horseVnum, 1000) },
		"World.spawnMobQuiet": func() (*MobInstance, error) { return w.spawnMobQuiet(horseVnum, 1000) },
		"World.SpawnMobQuiet": func() (*MobInstance, error) { return w.SpawnMobQuiet(horseVnum, 1000) },
		"Spawner.SpawnMob":    func() (*MobInstance, error) { return w.GetSpawner().SpawnMob(horseVnum, 1000) },
	}
	for name, spawn := range paths {
		horse, err := spawn()
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got, want := horse.GetMove(), 50; got != want {
			t.Errorf("%s GET_MOVE = %d, want %d", name, got, want)
		}
		if got, want := horse.GetMaxMove(), 50; got != want {
			t.Errorf("%s GET_MAX_MOVE = %d, want %d", name, got, want)
		}
	}
}
