package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

type roachTestWorld struct {
	w       *World
	actor   *Player
	peer    *Player
	mob     *MobInstance
	message map[string]string
}

func newRoachTestWorld(t *testing.T, peerRoom int) roachTestWorld {
	t.Helper()
	w, actor, _ := newSpecProcTestWorld(t)

	proto := &parser.Mob{
		VNum:        23,
		Keywords:    "roach",
		ShortDesc:   "a large roach",
		LongDesc:    "A large roach scurries here.",
		Level:       1,
		HP:          parser.DiceRoll{Num: 1, Sides: 1, Plus: 100},
		Damage:      parser.DiceRoll{Num: 1, Sides: 4, Plus: 1},
		ActionFlags: []string{"SPEC"},
	}
	w.mu.Lock()
	w.mobs[proto.VNum] = proto
	w.mu.Unlock()
	mob, err := w.SpawnMob(proto.VNum, actor.GetRoomVNum())
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}
	mob.SetPosition(combat.PosStanding)

	peer := NewPlayer(2, "RoachPeer", peerRoom)
	peer.SetPosition(combat.PosStanding)
	if err := w.AddPlayer(peer); err != nil {
		t.Fatalf("AddPlayer peer: %v", err)
	}

	message := make(map[string]string)
	w.MessageSink = func(name string, msg []byte) { message[name] += string(msg) }
	return roachTestWorld{w: w, actor: actor, peer: peer, mob: mob, message: message}
}

func (r roachTestWorld) clearMessages() {
	for name := range r.message {
		r.message[name] = ""
	}
}

func addRoachFood(t *testing.T, r roachTestWorld, vnum, cost int) *ObjectInstance {
	t.Helper()
	proto := &parser.Obj{
		VNum:      vnum,
		Keywords:  "bread loaf",
		ShortDesc: "a loaf of bread",
		LongDesc:  "A loaf of bread lies here.",
		WearFlags: [4]int{1},
		Cost:      cost,
	}
	r.w.mu.Lock()
	r.w.objs[vnum] = proto
	r.w.mu.Unlock()
	obj := r.w.newObjectInstance(proto, r.actor.GetRoomVNum())
	r.w.AddItemToRoom(obj, r.actor.GetRoomVNum())
	return obj
}

func setRoachDraws(t *testing.T, draws ...int) {
	t.Helper()
	old := roachNumber
	index := 0
	roachNumber = func(_, _ int) int {
		if index >= len(draws) {
			t.Fatalf("roach RNG exhausted at draw %d", index)
		}
		value := draws[index]
		index++
		return value
	}
	t.Cleanup(func() { roachNumber = old })
}

func TestSpecRoachEntryGates(t *testing.T) {
	r := newRoachTestWorld(t, 1001)
	setRoachDraws(t, 1, 5)
	if got := specRoach(r.w, r.actor, r.mob, "look", ""); got {
		t.Fatal("command invocation was handled")
	}
	if got := specRoach(r.w, nil, r.mob, "", ""); got {
		t.Fatal("sleep-free commandless invocation unexpectedly handled")
	}
	r.mob.SetPosition(combat.PosSleeping)
	if got := specRoach(r.w, nil, r.mob, "", ""); got {
		t.Fatal("sleeping roach was handled")
	}
	if got := strings.TrimSpace(r.message[r.actor.Name]); got != "" {
		t.Fatalf("entry gate emitted %q", got)
	}
}

func TestSpecRoachFoodSuccessGrowthAndExtraction(t *testing.T) {
	r := newRoachTestWorld(t, 1001)
	obj := addRoachFood(t, r, 4001, 100)
	r.mob.SetMaxHP(100)
	r.mob.SetHealth(100)
	r.clearMessages()

	// The first two draws are starvation checks, followed by food success,
	// damnodice increment, damsizedice increment, and stretching choice.
	setRoachDraws(t, 0, 1, 0, 0, 0, 0)
	if !specRoach(r.w, nil, r.mob, "", "") {
		t.Fatal("food success was not handled")
	}
	want := "A large roach feeds on a loaf of bread.\r\nYou hear some stretching noises.\r\n"
	if r.message[r.actor.Name] != want || r.message[r.peer.Name] != want {
		t.Fatalf("food success audience: actor=%q peer=%q want=%q", r.message[r.actor.Name], r.message[r.peer.Name], want)
	}
	if got := r.mob.GetMaxHealth(); got != 150 {
		t.Fatalf("grown max HP = %d, want 150", got)
	}
	if got := r.mob.GetDamageRoll(); got != (combat.DiceRoll{Num: 2, Sides: 5, Plus: 1}) {
		t.Fatalf("grown damage roll = %+v, want 2d5+1", got)
	}
	if got := obj.Location; got != LocNowhere() {
		t.Fatalf("food location = %+v, want nowhere", got)
	}
	if got := len(r.w.GetItemsInRoom(r.actor.GetRoomVNum())); got != 0 {
		t.Fatalf("room food count = %d, want 0", got)
	}
}

func TestSpecRoachFoodFailureStillExtractsObject(t *testing.T) {
	r := newRoachTestWorld(t, 1001)
	obj := addRoachFood(t, r, 4002, 100)
	r.clearMessages()
	setRoachDraws(t, 0, 1, 1)
	if !specRoach(r.w, nil, r.mob, "", "") {
		t.Fatal("food failure was not handled")
	}
	want := "A large roach feeds on a loaf of bread.\r\nYou hear a large roach burp.\r\n"
	if got := r.message[r.actor.Name]; got != want {
		t.Fatalf("food failure output = %q, want %q", got, want)
	}
	if got := obj.Location; got != LocNowhere() {
		t.Fatalf("failed food location = %+v, want nowhere", got)
	}
	if got := r.mob.GetMaxHealth(); got != 101 {
		t.Fatalf("failed food changed max HP to %d", got)
	}
}

func TestSpecRoachSplitResetsOriginalAndSpawnsQuietRoach(t *testing.T) {
	r := newRoachTestWorld(t, 1001)
	addRoachFood(t, r, 4003, 800)
	r.mob.SetMaxHP(100)
	r.mob.SetHealth(100)
	r.clearMessages()
	setRoachDraws(t, 0, 1, 0, 0, 0)
	if !specRoach(r.w, nil, r.mob, "", "") {
		t.Fatal("split was not handled")
	}
	want := "A large roach feeds on a loaf of bread.\r\nA large roach splits in half forming a new roach!\r\n"
	if got := r.message[r.actor.Name]; got != want {
		t.Fatalf("split output = %q, want %q", got, want)
	}
	if got := r.mob.GetMaxHealth(); got != 10 {
		t.Fatalf("original max HP = %d, want 10", got)
	}
	if got := r.mob.GetHP(); got != 10 {
		t.Fatalf("original HP = %d, want 10", got)
	}
	if got := r.mob.GetDamageRoll(); got != (combat.DiceRoll{Num: 2, Sides: 4, Plus: 1}) {
		t.Fatalf("original damage roll = %+v, want 2d4+1", got)
	}
	mobs := r.w.GetMobsInRoom(r.actor.GetRoomVNum())
	if len(mobs) != 2 {
		t.Fatalf("roach count after split = %d, want 2", len(mobs))
	}
}

func TestSpecRoachTeleportSubgateAndAudience(t *testing.T) {
	r := newRoachTestWorld(t, 1002)
	r.w.mu.Lock()
	r.w.rooms[1002] = &parser.Room{VNum: 1002, Name: "Destination", Zone: 1}
	r.w.roomOrder = append(r.w.roomOrder, 1002)
	r.w.mu.Unlock()
	r.clearMessages()

	// Case 4 followed by a nonzero 1-in-6 gate must be a silent fallthrough.
	setRoachDraws(t, 1, 4, 1)
	if specRoach(r.w, nil, r.mob, "", "") {
		t.Fatal("failed teleport subgate was handled")
	}
	if got := r.mob.GetRoomVNum(); got != 1001 {
		t.Fatalf("failed teleport moved roach to %d", got)
	}
	if got := r.message[r.actor.Name] + r.message[r.peer.Name]; got != "" {
		t.Fatalf("failed teleport emitted %q", got)
	}

	r.clearMessages()
	setRoachDraws(t, 1, 4, 0, 1)
	if !specRoach(r.w, nil, r.mob, "", "") {
		t.Fatal("successful teleport was not handled")
	}
	if got := r.mob.GetRoomVNum(); got != 1002 {
		t.Fatalf("successful teleport room = %d, want 1002", got)
	}
	if got, want := r.message[r.actor.Name], "A large roach fades out slowly with a soft swoosh.\r\n"; got != want {
		t.Fatalf("origin audience = %q, want %q", got, want)
	}
	if got, want := r.message[r.peer.Name], "A large roach fades in slowly, looking a bit disoriented.\r\n"; got != want {
		t.Fatalf("destination audience = %q, want %q", got, want)
	}
}

func TestSpecRoachStarvationExtractsWithoutCorpse(t *testing.T) {
	r := newRoachTestWorld(t, 1001)
	r.mob.SetMaxHP(10)
	r.clearMessages()
	setRoachDraws(t, 0, 0)
	if !specRoach(r.w, nil, r.mob, "", "") {
		t.Fatal("starvation was not handled")
	}
	want := "A large roach seems to starve to death and simply fades out of existence.\r\n"
	if got := r.message[r.actor.Name]; got != want {
		t.Fatalf("starvation output = %q, want %q", got, want)
	}
	if r.mob.IsAlive() {
		t.Fatal("starved roach remained alive")
	}
	if got, ok := r.w.GetMobByID(r.mob.GetID()); ok || got != nil {
		t.Fatal("starved roach remained in active world")
	}
	if got := len(r.w.GetItemsInRoom(r.actor.GetRoomVNum())); got != 0 {
		t.Fatalf("starvation created room objects: %d", got)
	}
}
