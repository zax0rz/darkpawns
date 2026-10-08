package game

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
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

// TestObjectActivityWeightMudlog proves comm.c:773-787 at its four location
// wordings and its three exemptions (src/comm.c:770-771 and :774). A repaired
// object must also come back to its prototype's weight.
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
	t.Run("worn by", func(t *testing.T) {
		run(t, func(t *testing.T, w *World) *ObjectInstance {
			p := NewPlayer(705, "Wtwearer", 100)
			if err := w.AddPlayer(p); err != nil {
				t.Fatalf("AddPlayer: %v", err)
			}
			obj := spawn(t, w, 200, 100)
			obj.Location = LocEquippedPlayer(p.GetName(), EquipmentSlot(0))
			return weighted(obj)
		}, true, "worn by")
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
		// == 1 (src/utils.h, and the port sets exactly that on a real corpse:
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
}
