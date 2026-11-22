package game

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// weightProbe stages the BRF (1) at LVL_IMMORT (31), file TRUE contract of C's
// object_activity weight arm (src/comm.c:773-787): a LOG1 observer at the
// threshold sees it, one level below does not, and an observer with no syslog
// flags stays silent, which pins the type to exactly BRF.
func weightProbe(t *testing.T) (*milestoneProvider, *bytes.Buffer, *Player, *Player, *Player) {
	t.Helper()
	watch := NewPlayer(701, "Wtwatch", 100)
	watch.SetLevel(LVL_IMMORT)
	watch.SetPlrFlag(PrfLog1, true)
	below := NewPlayer(702, "Wtbelow", 100)
	below.SetLevel(LVL_IMMORT - 1)
	below.SetPlrFlag(PrfLog1, true)
	off := NewPlayer(703, "Wtoff", 100)
	off.SetLevel(40)
	provider := &milestoneProvider{
		players: []*Player{watch, below, off},
		lines:   map[string]string{},
	}
	oldProvider := getImmortalSessionProvider()
	SetImmortalSessionProvider(provider)
	t.Cleanup(func() { SetImmortalSessionProvider(oldProvider) })
	file := &bytes.Buffer{}
	oldWriter := getLogWriter()
	SetLogWriter(file)
	t.Cleanup(func() { SetLogWriter(oldWriter) })
	return provider, file, watch, below, off
}

// TestObjectActivityWeightMudlog proves src/comm.c:773-787 at every location
// wording C can reach and at its three exemptions (src/comm.c:770-771 and
// :774). The wordings are C's truthiness reading of carried_by/in_room/worn_by,
// not a one-to-one map of the port's location union: see repairObjectWeight. A
// repaired object must also come back to its prototype's weight.
func TestObjectActivityWeightMudlog(t *testing.T) {
	run := func(t *testing.T, prepare func(t *testing.T, w *World) *ObjectInstance, wantLog bool, wantLocation string) {
		t.Helper()
		w, _ := newZoneResetTestSpawner(t)
		provider, file, watch, below, off := weightProbe(t)
		obj := prepare(t, w)
		before := obj.GetWeight()
		w.ObjectActivity()
		if !wantLog {
			if got := strings.Count(file.String(), "weight incorrect"); got != 0 {
				t.Fatalf("exempt object logged %d weight lines: %q", got, file.String())
			}
			if obj.GetWeight() != before {
				t.Fatalf("exempt object weight changed: %d -> %d", before, obj.GetWeight())
			}
			return
		}
		payload := fmt.Sprintf("SYSERR: Object '%s' weight incorrect, location '%s'", obj.GetShortDesc(), wantLocation)
		if got := strings.Count(file.String(), payload); got != 1 {
			t.Fatalf("file payload count=%d want 1; log=%q", got, file.String())
		}
		if want := "[ " + payload + " ]\r\n"; !strings.Contains(provider.lines[watch.Name], want) {
			t.Fatalf("observer bytes=%q want %q", provider.lines[watch.Name], want)
		}
		if strings.Contains(provider.lines[below.Name], payload) || strings.Contains(provider.lines[off.Name], payload) {
			t.Fatal("level or type filter leak")
		}
		if obj.GetWeight() != obj.Prototype.Weight {
			t.Fatalf("weight not repaired: %d want %d", obj.GetWeight(), obj.Prototype.Weight)
		}
	}

	weighted := func(obj *ObjectInstance) *ObjectInstance {
		obj.SetWeight(obj.Prototype.Weight + 5)
		return obj
	}
	spawn := func(t *testing.T, w *World, vnum, room int) *ObjectInstance {
		t.Helper()
		obj, err := w.SpawnObject(vnum, room)
		if err != nil {
			t.Fatalf("spawn %d: %v", vnum, err)
		}
		return obj
	}

	t.Run("in room", func(t *testing.T) {
		run(t, func(t *testing.T, w *World) *ObjectInstance {
			return weighted(spawn(t, w, 200, 100))
		}, true, "in room")
	})
	t.Run("carried", func(t *testing.T) {
		run(t, func(t *testing.T, w *World) *ObjectInstance {
			p := NewPlayer(704, "Wtcarrier", 100)
			if err := w.AddPlayer(p); err != nil {
				t.Fatalf("AddPlayer: %v", err)
			}
			obj := spawn(t, w, 200, 100)
			if err := w.MoveObjectToPlayerInventory(obj, p); err != nil {
				t.Fatalf("move to inventory: %v", err)
			}
			return weighted(obj)
		}, true, "carried")
	})
	t.Run("worn object prints in room", func(t *testing.T) {
		// C's "worn by" arm is dead code: equip_char leaves in_room at NOWHERE
		// (src/handler.c:565,731) and -1 is truthy (src/structs.h:34).
		run(t, func(t *testing.T, w *World) *ObjectInstance {
			p := NewPlayer(705, "Wtwearer", 100)
			if err := w.AddPlayer(p); err != nil {
				t.Fatalf("AddPlayer: %v", err)
			}
			obj := spawn(t, w, 200, 100)
			obj.Prototype.WearFlags = [4]int{1 << 4}
			if err := w.MoveObjectToPlayerInventory(obj, p); err != nil {
				t.Fatalf("move to inventory: %v", err)
			}
			if err := w.EquipItem(p, obj, eqWearHead); err != nil {
				t.Fatalf("equip object: %v", err)
			}
			return weighted(obj)
		}, true, "in room")
	})
	t.Run("object inside a container prints in room", func(t *testing.T) {
		run(t, func(t *testing.T, w *World) *ObjectInstance {
			container := spawn(t, w, 204, 100)
			container.Prototype.TypeFlag = ITEM_CONTAINER
			inner := spawn(t, w, 200, 100)
			if err := w.MoveObjectToContainer(inner, container); err != nil {
				t.Fatalf("move to container: %v", err)
			}
			return weighted(inner)
		}, true, "in room")
	})
	t.Run("drink container is exempt", func(t *testing.T) {
		run(t, func(t *testing.T, w *World) *ObjectInstance {
			obj := spawn(t, w, 200, 100)
			obj.Prototype.TypeFlag = ITEM_DRINKCON
			return weighted(obj)
		}, false, "")
	})
	t.Run("corpse is exempt", func(t *testing.T) {
		// C's IS_CORPSE is GET_OBJ_TYPE == ITEM_CONTAINER && GET_OBJ_VAL(obj,3)
		// == 1 (src/utils.h:490-491, and the port sets exactly that on a real corpse:
		// pkg/game/death.go:1090-1101), so the predicate is exercised here
		// without the corpse builder's world prerequisites.
		run(t, func(t *testing.T, w *World) *ObjectInstance {
			obj := spawn(t, w, 204, 100)
			obj.Prototype.TypeFlag = ITEM_CONTAINER
			obj.SetValue(3, 1)
			return weighted(obj)
		}, false, "")
	})
	t.Run("container holding things is exempt", func(t *testing.T) {
		run(t, func(t *testing.T, w *World) *ObjectInstance {
			container := spawn(t, w, 204, 100)
			container.Prototype.TypeFlag = ITEM_CONTAINER
			inner := spawn(t, w, 200, 100)
			if err := w.MoveObjectToContainer(inner, container); err != nil {
				t.Fatalf("move to container: %v", err)
			}
			return weighted(container)
		}, false, "")
	})
	t.Run("shop stock prints carried", func(t *testing.T) {
		// A shop's stock sits in the keeper's inventory in C.
		run(t, func(t *testing.T, w *World) *ObjectInstance {
			obj := spawn(t, w, 200, 100)
			if err := w.MoveObject(obj, LocShop(1)); err != nil {
				t.Fatalf("move to shop stock: %v", err)
			}
			return weighted(obj)
		}, true, "carried")
	})
	t.Run("room zero after extraction prints unknown", func(t *testing.T) {
		w, err := NewWorld(&parser.World{
			Rooms: []parser.Room{{VNum: 0, Name: "The Void", Zone: 0}},
			Objs:  []parser.Obj{{VNum: 200, Keywords: "weighted", ShortDesc: "a weighted thing", Weight: 1}},
		})
		if err != nil {
			t.Fatalf("NewWorld: %v", err)
		}
		t.Cleanup(w.StopAITicker)
		provider, file, watch, below, off := weightProbe(t)
		carrier := NewPlayer(706, "Wtvoid", 0)
		if err := w.AddPlayer(carrier); err != nil {
			t.Fatalf("AddPlayer: %v", err)
		}
		obj, err := w.SpawnObject(200, -1)
		if err != nil {
			t.Fatalf("SpawnObject: %v", err)
		}
		if err := w.MoveObjectToPlayerInventory(obj, carrier); err != nil {
			t.Fatalf("move to inventory: %v", err)
		}
		obj.SetWeight(obj.Prototype.Weight + 5)

		// C extract_char drops carried objects into the character's current room
		// (src/handler.c:1132-1137). The Go extraction pass likewise inserts the
		// item into room 0 (pkg/game/char_mgmt.go:249-290), where C's falsy
		// in_room value makes object_activity call it "unknown" (src/comm.c:775-782).
		w.QueuePlayerExtraction(carrier)
		if got := w.ExtractPendingPlayers(); len(got) != 1 || got[0] != carrier {
			t.Fatalf("extracted players = %v, want Wtvoid", got)
		}
		if got := obj.Location; got != LocRoom(0) {
			t.Fatalf("object after extraction = %+v, want room 0", got)
		}
		w.ObjectActivity()

		payload := fmt.Sprintf("SYSERR: Object '%s' weight incorrect, location 'unknown'", obj.GetShortDesc())
		if got := strings.Count(file.String(), payload); got != 1 {
			t.Fatalf("file payload count=%d want 1; log=%q", got, file.String())
		}
		if want := "[ " + payload + " ]\r\n"; !strings.Contains(provider.lines[watch.Name], want) {
			t.Fatalf("observer bytes=%q want %q", provider.lines[watch.Name], want)
		}
		if strings.Contains(provider.lines[below.Name], payload) || strings.Contains(provider.lines[off.Name], payload) {
			t.Fatal("level or type filter leak")
		}
		if obj.GetWeight() != obj.Prototype.Weight {
			t.Fatalf("weight not repaired: %d want %d", obj.GetWeight(), obj.Prototype.Weight)
		}
	})
}

func TestObjectActivityLogsContainerWeightFloorAfterGet(t *testing.T) {
	w, _ := newZoneResetTestSpawner(t)
	provider, file, watch, below, off := weightProbe(t)

	container, err := w.SpawnObject(204, -1)
	if err != nil {
		t.Fatalf("spawn container: %v", err)
	}
	container.Prototype.TypeFlag = ITEM_CONTAINER
	container.Prototype.Weight = 0
	container.Prototype.ShortDesc = "an empty zero-weight container"
	if err := w.MoveObjectToRoom(container, 100); err != nil {
		t.Fatalf("place container: %v", err)
	}

	item, err := w.SpawnObject(200, -1)
	if err != nil {
		t.Fatalf("spawn item: %v", err)
	}
	item.Prototype.Weight = 12
	if err := w.MoveObjectToContainer(item, container); err != nil {
		t.Fatalf("put item in container: %v", err)
	}

	carrier := NewPlayer(704, "Wtcarrier", 100)
	if err := w.AddPlayer(carrier); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	if err := w.MoveObjectToPlayerInventory(item, carrier); err != nil {
		t.Fatalf("get item from container: %v", err)
	}
	if got := container.GetWeight(); got != 0 || !container.weightFloorPending.Load() {
		t.Fatalf("empty zero-weight container weight = %d, pending floor = %v; want derived weight 0 with C floor pending", got, container.weightFloorPending.Load())
	}

	// C get_from_container calls obj_from_obj before the next mobile pulse
	// reaches object_activity (src/act.item.c:197-209; src/handler.c:963-990;
	// src/comm.c:815-820). The empty container's diagnostic is visible to the
	// BRF/LVL_IMMORT observer, then the pulse clears the pending marker.
	w.ObjectActivity()
	payload := "SYSERR: Object 'an empty zero-weight container' weight incorrect, location 'in room'"
	if got := strings.Count(file.String(), payload); got != 1 {
		t.Fatalf("file payload count=%d want 1; log=%q", got, file.String())
	}
	if want := "[ " + payload + " ]\r\n"; !strings.Contains(provider.lines[watch.Name], want) {
		t.Fatalf("observer bytes=%q want %q", provider.lines[watch.Name], want)
	}
	if strings.Contains(provider.lines[below.Name], payload) || strings.Contains(provider.lines[off.Name], payload) {
		t.Fatal("level or type filter leak")
	}
	if got := container.GetWeight(); got != container.Prototype.Weight || container.weightFloorPending.Load() {
		t.Fatalf("container after object activity: weight=%d pending floor=%v, want prototype weight %d and no pending floor", got, container.weightFloorPending.Load(), container.Prototype.Weight)
	}
}
