package game

import (
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newDoorHandlerTestWorld(t *testing.T) (*World, *Player, map[string][]string) {
	t.Helper()
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 100, Name: "Near", Exits: map[string]parser.Exit{
				"north": {Direction: "north", ToRoom: 101, DoorState: 1, ExitInfo: parser.ExitIsDoor | parser.ExitClosed | parser.ExitLocked, Key: 900, Keywords: "iron door"},
			}},
			{VNum: 101, Name: "Far", Exits: map[string]parser.Exit{
				"south": {Direction: "south", ToRoom: 100, DoorState: 1, ExitInfo: parser.ExitIsDoor | parser.ExitClosed | parser.ExitLocked, Key: 900, Keywords: "iron door"},
			}},
		},
		Objs: []parser.Obj{
			{VNum: 700, TypeFlag: ITEM_CONTAINER, Keywords: "test chest", ShortDesc: "a test chest", Values: [4]int{100, contCloseable | contClosed | contLocked, 900, 0}, WearFlags: [4]int{1}},
			{VNum: 701, TypeFlag: ITEM_CONTAINER, Keywords: "keyless box", ShortDesc: "a keyless box", Values: [4]int{100, contCloseable | contClosed | contLocked, -1, 0}, WearFlags: [4]int{1}},
			{VNum: 702, Keywords: "plain rock", ShortDesc: "a plain rock", WearFlags: [4]int{1}},
			{VNum: 900, Keywords: "iron key", ShortDesc: "an iron key", WearFlags: [4]int{1}},
			{VNum: 8027, Keywords: "lockpicks picks", ShortDesc: "a set of lockpicks", WearFlags: [4]int{1 << 14}},
			{VNum: 8028, Keywords: "broken lockpicks picks", ShortDesc: "a set of broken lockpicks", WearFlags: [4]int{1 << 14}},
		},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })
	p := NewPlayer(1, "DoorTester", 100)
	p.SetLevel(10)
	if err := w.AddPlayer(p); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	messages := make(map[string][]string)
	w.MessageSink = func(name string, msg []byte) {
		messages[name] = append(messages[name], string(msg))
	}
	return w, p, messages
}

func giveDoorTestObject(t *testing.T, w *World, p *Player, vnum int) *ObjectInstance {
	t.Helper()
	obj, err := w.SpawnObject(vnum, -1)
	if err != nil {
		t.Fatalf("SpawnObject(%d): %v", vnum, err)
	}
	if err := w.MoveObjectToPlayerInventory(obj, p); err != nil {
		t.Fatalf("MoveObjectToPlayerInventory(%d): %v", vnum, err)
	}
	return obj
}

func joinedDoorMessages(messages map[string][]string, name string) string {
	return strings.Join(messages[name], "")
}

func TestDoGenDoorContainerLadderAndInstanceIsolation(t *testing.T) {
	w, p, messages := newDoorHandlerTestWorld(t)
	first := giveDoorTestObject(t, w, p, 700)
	second := giveDoorTestObject(t, w, p, 700)
	rock := giveDoorTestObject(t, w, p, 702)
	first.SetValue(contFlags, contCloseable|contClosed)
	second.SetValue(contFlags, contCloseable|contClosed)

	w.DoOpen(p, "test")
	if first.GetValue(contFlags)&contClosed != 0 {
		t.Fatal("opening the first matching container left it closed")
	}
	if second.GetValue(contFlags)&contClosed == 0 {
		t.Fatal("opening one instance mutated the second instance")
	}
	if first.Prototype.Values[contFlags] != contCloseable|contClosed|contLocked {
		t.Fatalf("prototype flags mutated: got %d", first.Prototype.Values[contFlags])
	}

	messages[p.Name] = nil
	w.DoOpen(p, "test")
	if got := joinedDoorMessages(messages, p.Name); !strings.Contains(got, "But it's currently open!") {
		t.Fatalf("second open message = %q", got)
	}

	messages[p.Name] = nil
	w.DoClose(p, "test")
	w.DoClose(p, "test")
	if got := joinedDoorMessages(messages, p.Name); !strings.Contains(got, "But it's already closed!") {
		t.Fatalf("second close message = %q", got)
	}

	messages[p.Name] = nil
	w.DoOpen(p, "plain")
	if got := joinedDoorMessages(messages, p.Name); !strings.Contains(got, "You can't open that!") {
		t.Fatalf("non-container message = %q", got)
	}
	if rock.GetValue(contFlags) != 0 {
		t.Fatal("non-container operation mutated object values")
	}
}

func TestDoGenDoorExactPreconditionMessages(t *testing.T) {
	tests := []struct {
		name      string
		flags     int
		command   func(*World, *Player, string)
		argument  string
		want      string
		container int
	}{
		{name: "empty", command: (*World).DoOpen, want: "Open what?", container: 700},
		{name: "wasnt locked", flags: contCloseable | contClosed, command: (*World).DoPick, argument: "test", want: "Oh.. it wasn't locked, after all..", container: 700},
		{name: "locked blocks open", flags: contCloseable | contClosed | contLocked, command: (*World).DoOpen, argument: "test", want: "It seems to be locked.", container: 700},
		{name: "missing key", flags: contCloseable | contClosed | contLocked, command: (*World).DoUnlock, argument: "test", want: "You don't seem to have the proper key.", container: 700},
		{name: "odd keyhole", flags: contCloseable | contClosed | contLocked, command: (*World).DoPick, argument: "keyless", want: "Odd - you can't seem to find a keyhole.", container: 701},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, p, messages := newDoorHandlerTestWorld(t)
			obj := giveDoorTestObject(t, w, p, tt.container)
			if tt.flags != 0 {
				obj.SetValue(contFlags, tt.flags)
			}
			tt.command(w, p, tt.argument)
			if got := joinedDoorMessages(messages, p.Name); !strings.Contains(got, tt.want) {
				t.Fatalf("message = %q, want substring %q", got, tt.want)
			}
		})
	}
}

func TestDoorExitMutationIsReciprocal(t *testing.T) {
	w, p, _ := newDoorHandlerTestWorld(t)
	giveDoorTestObject(t, w, p, 900)

	w.DoUnlock(p, "door north")
	w.DoOpen(p, "door north")
	near := w.GetRoomInWorld(100).Exits["north"]
	far := w.GetRoomInWorld(101).Exits["south"]
	if near.ExitInfo&(parser.ExitClosed|parser.ExitLocked) != 0 || far.ExitInfo&(parser.ExitClosed|parser.ExitLocked) != 0 {
		t.Fatalf("open did not clear both sides: near=%d far=%d", near.ExitInfo, far.ExitInfo)
	}
	if near.DoorState != 1 || far.DoorState != 1 {
		t.Fatalf("runtime mutation changed .wld capability: near=%d far=%d", near.DoorState, far.DoorState)
	}
}

func TestOkPickTier2RollAndBreakage(t *testing.T) {
	t.Run("success roll", func(t *testing.T) {
		w, p, messages := newDoorHandlerTestWorld(t)
		chest := giveDoorTestObject(t, w, p, 700)
		chest.SetValue(contFlags, contCloseable|contClosed|contLocked)
		picks := giveDoorTestObject(t, w, p, 8027)
		if err := w.MoveObject(picks, LocEquippedPlayer(p.Name, SlotHold)); err != nil {
			t.Fatalf("hold picks: %v", err)
		}
		p.SetSkill(SkillPickLock, 101)
		oldNumber := doorNumber
		rng := rand.New(rand.NewPCG(1, 2))
		doorNumber = func(from, to int) int { return rng.IntN(to-from+1) + from }
		t.Cleanup(func() { doorNumber = oldNumber })

		// Tier-2: Go's RNG is injected; the C oracle is not seed-matched.
		w.DoPick(p, "test")
		if chest.GetValue(contFlags)&contLocked != 0 {
			t.Fatal("successful skill roll left container locked")
		}
		if got := joinedDoorMessages(messages, p.Name); !strings.Contains(got, "The lock quickly yields to your skills.") {
			t.Fatalf("success message = %q", got)
		}
	})

	t.Run("failed roll breaks held picks", func(t *testing.T) {
		w, p, messages := newDoorHandlerTestWorld(t)
		chest := giveDoorTestObject(t, w, p, 700)
		chest.SetValue(contFlags, contCloseable|contClosed|contLocked)
		picks := giveDoorTestObject(t, w, p, 8027)
		if err := w.MoveObject(picks, LocEquippedPlayer(p.Name, SlotHold)); err != nil {
			t.Fatalf("hold picks: %v", err)
		}
		p.SetLevel(0)
		p.SetSkill(SkillPickLock, 0)
		oldNumber := doorNumber
		rng := rand.New(rand.NewPCG(3, 4))
		doorNumber = func(from, to int) int { return rng.IntN(to-from+1) + from }
		t.Cleanup(func() { doorNumber = oldNumber })

		// Tier-2: the failed skill roll and breakage roll are deterministic here.
		w.DoPick(p, "test")
		held, ok := p.Equipment.GetItemInSlot(SlotHold)
		if !ok || held.VNum != 8028 {
			t.Fatalf("held item after breakage = %#v, want vnum 8028", held)
		}
		if chest.GetValue(contFlags)&contLocked == 0 {
			t.Fatal("failed pick unexpectedly unlocked container")
		}
		got := joinedDoorMessages(messages, p.Name)
		if !strings.Contains(got, "You failed to pick the lock.") || !strings.Contains(got, "You ruin your lockpicks in the process.") {
			t.Fatalf("breakage messages = %q", got)
		}
	})
}

func TestPickRequiresHeldPicksAndHonorsPickproof(t *testing.T) {
	w, p, messages := newDoorHandlerTestWorld(t)
	chest := giveDoorTestObject(t, w, p, 700)

	oldNumber := doorNumber
	doorNumber = func(from, _ int) int { return from }
	t.Cleanup(func() { doorNumber = oldNumber })

	w.DoPick(p, "test")
	if got := joinedDoorMessages(messages, p.Name); !strings.Contains(got, "You'll need to hold a set of lockpicks") {
		t.Fatalf("missing-picks message = %q", got)
	}

	messages[p.Name] = nil
	picks := giveDoorTestObject(t, w, p, 8027)
	if err := w.MoveObject(picks, LocEquippedPlayer(p.Name, SlotHold)); err != nil {
		t.Fatalf("hold picks: %v", err)
	}
	chest.SetValue(contFlags, contCloseable|contPickproofBit|contClosed|contLocked)
	w.DoPick(p, "test")
	if got := joinedDoorMessages(messages, p.Name); !strings.Contains(got, "It resists your attempts to pick it.") {
		t.Fatalf("pickproof message = %q", got)
	}
}

func TestZoneDoorResetUsesRuntimeBits(t *testing.T) {
	w, _, _ := newDoorHandlerTestWorld(t)
	spawner := NewSpawner(w)
	zone := &parser.Zone{Commands: []parser.ZoneCommand{{Command: "D", Arg1: 100, Arg2: 0, Arg3: 0}}}
	if err := spawner.ExecuteZoneReset(zone); err != nil {
		t.Fatalf("open reset: %v", err)
	}
	ext := w.GetRoomInWorld(100).Exits["north"]
	if ext.ExitInfo&parser.ExitClosed != 0 || ext.ExitInfo&parser.ExitIsDoor == 0 {
		t.Fatalf("open reset exit info = %d", ext.ExitInfo)
	}
	zone.Commands[0].Arg3 = 2
	if err := spawner.ExecuteZoneReset(zone); err != nil {
		t.Fatalf("locked reset: %v", err)
	}
	ext = w.GetRoomInWorld(100).Exits["north"]
	if ext.ExitInfo&(parser.ExitClosed|parser.ExitLocked) != parser.ExitClosed|parser.ExitLocked {
		t.Fatalf("locked reset exit info = %d", ext.ExitInfo)
	}
	if ext.DoorState != 1 {
		t.Fatalf("reset changed .wld capability code to %d", ext.DoorState)
	}
}

func TestKeylessDoorRejectsCoinPileKey(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 200, Name: "Keyless near", Exits: map[string]parser.Exit{
				"east": {Direction: "east", ToRoom: 201, DoorState: 1, ExitInfo: parser.ExitIsDoor | parser.ExitClosed | parser.ExitLocked, Key: -1, Keywords: "wooden door"},
			}},
			{VNum: 201, Name: "Keyless far", Exits: map[string]parser.Exit{
				"west": {Direction: "west", ToRoom: 200, DoorState: 1, ExitInfo: parser.ExitIsDoor | parser.ExitClosed | parser.ExitLocked, Key: -1, Keywords: "wooden door"},
			}},
		},
		Objs: []parser.Obj{
			{VNum: -1, Keywords: "coins gold", ShortDesc: "a pile of coins"},
		},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	p := NewPlayer(1, "KeylessTester", 200)
	p.SetLevel(10)
	if err := w.AddPlayer(p); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}

	coins, err := w.SpawnObject(-1, -1)
	if err != nil {
		t.Fatalf("SpawnObject(-1): %v", err)
	}
	if err := w.MoveObjectToPlayerInventory(coins, p); err != nil {
		t.Fatalf("MoveObjectToPlayerInventory: %v", err)
	}

	messages := make(map[string][]string)
	w.MessageSink = func(name string, msg []byte) {
		messages[name] = append(messages[name], string(msg))
	}

	w.DoUnlock(p, "door east")
	if got := joinedDoorMessages(messages, p.Name); !strings.Contains(got, "You don't seem to have the proper key.") {
		t.Fatalf("unlock message = %q, want proper-key rejection", got)
	}

	// For locking, the door must be closed and unlocked to reach the key check.
	nearExit := w.GetRoomInWorld(200).Exits["east"]
	nearExit.ExitInfo &^= parser.ExitLocked
	w.GetRoomInWorld(200).Exits["east"] = nearExit
	farExit := w.GetRoomInWorld(201).Exits["west"]
	farExit.ExitInfo &^= parser.ExitLocked
	w.GetRoomInWorld(201).Exits["west"] = farExit
	messages[p.Name] = nil
	w.DoLock(p, "door east")
	if got := joinedDoorMessages(messages, p.Name); !strings.Contains(got, "You don't seem to have the proper key.") {
		t.Fatalf("lock message = %q, want proper-key rejection", got)
	}
}
