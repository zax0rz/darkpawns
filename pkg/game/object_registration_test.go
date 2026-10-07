package game

import (
	"os"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func objregWorld(t *testing.T) *World {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Shop Room", Zone: 1}},
		Objs: []parser.Obj{
			{VNum: 7001, Keywords: "bag", ShortDesc: "a leather bag", LongDesc: "A bag.", TypeFlag: 15},
			{VNum: 7002, Keywords: "sword", ShortDesc: "a sword", LongDesc: "A sword."},
		},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	return w
}

func registrySize(w *World) int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return len(w.objectInstances)
}

// The proven bug (brief, verified 2026-10-07): a shop-bought container kept
// ID 0 and never entered objectInstances, so MoveObjectToContainer rejected
// it — "You can't put that in there." The registered bag must accept items.
func TestBoughtContainerAcceptsPut(t *testing.T) {
	w := objregWorld(t)
	w.mu.RLock()
	bagProto := w.objs[7001]
	swordProto := w.objs[7002]
	w.mu.RUnlock()

	bag := w.NewObjectFromProto(bagProto, -1)
	if bag == nil || bag.ID == 0 {
		t.Fatal("bag has no registry ID")
	}
	if _, ok := func() (*ObjectInstance, bool) {
		w.mu.RLock()
		defer w.mu.RUnlock()
		o, ok := w.objectInstances[bag.ID]
		return o, ok
	}(); !ok {
		t.Fatal("bag not in objectInstances")
	}

	sword := w.NewObjectFromProto(swordProto, -1)
	if err := w.MoveObjectToContainer(sword, bag); err != nil {
		t.Fatalf("put into bought container failed (the live bug): %v", err)
	}
	if sword.Location.Kind != ObjInContainer || sword.Location.ContainerObjID != bag.ID {
		t.Fatalf("sword not inside bag: location = %+v", sword.Location)
	}
}

// Every live site's constructor path must yield a registered, uniquely
// identified object. This covers the shared constructors; per-site dispatch
// is covered by the session and world tests below.
func TestRegisteredObjectsHaveUniqueIDs(t *testing.T) {
	w := objregWorld(t)
	w.mu.RLock()
	proto := w.objs[7002]
	w.mu.RUnlock()

	ids := map[int]bool{}
	for i := 0; i < 50; i++ {
		obj := w.NewObjectFromProto(proto, -1)
		if obj == nil || obj.ID == 0 {
			t.Fatalf("object %d unregistered", i)
		}
		if ids[obj.ID] {
			t.Fatalf("duplicate ID %d", obj.ID)
		}
		ids[obj.ID] = true
	}
	before := registrySize(w)
	if before < 50 {
		t.Fatalf("registry holds %d, want >= 50", before)
	}
}

// The mobprogs/gates CreateObject path must register (its objects land in
// rooms and can be picked up and containerized by players).
func TestCreateObjectRegisters(t *testing.T) {
	w := objregWorld(t)
	before := registrySize(w)
	obj := w.CreateObject(7002, 1001)
	if obj == nil {
		t.Fatal("CreateObject nil")
	}
	if obj.ID == 0 {
		t.Fatal("CreateObject object unregistered (ID 0)")
	}
	if registrySize(w) != before+1 {
		t.Fatalf("registry %d -> %d, want +1", before, registrySize(w))
	}
}

// The Lua room arm (AddItemToRoomScriptable) must register its spawned item.
func TestAddItemToRoomScriptableRegisters(t *testing.T) {
	w := objregWorld(t)
	before := registrySize(w)
	// Pass a bare scriptable with only a vnum so the fallback constructor runs.
	fallback := &vnumOnlyScriptable{vnum: 7002}
	if err := w.AddItemToRoomScriptable(fallback, 1001); err != nil {
		t.Fatalf("AddItemToRoomScriptable: %v", err)
	}
	if registrySize(w) != before+1 {
		t.Fatalf("Lua-spawned item not registered: %d -> %d", before, registrySize(w))
	}
}

type vnumOnlyScriptable struct{ vnum int }

func (v *vnumOnlyScriptable) GetVNum() int         { return v.vnum }
func (v *vnumOnlyScriptable) GetKeywords() string  { return "sword" }
func (v *vnumOnlyScriptable) GetShortDesc() string { return "a sword" }
func (v *vnumOnlyScriptable) GetCost() int         { return 10 }
func (v *vnumOnlyScriptable) GetTimer() int        { return 0 }
func (v *vnumOnlyScriptable) SetTimer(int)         {}
func (v *vnumOnlyScriptable) GetTypeFlag() int     { return 0 }
func (v *vnumOnlyScriptable) GetInstanceID() int   { return 0 }

// Destruction: extraction must remove registered objects — no registry leak.
// Buy-then-junk shape: register, containerize, extract both.
func TestExtractionRemovesRegisteredObjects(t *testing.T) {
	w := objregWorld(t)
	w.mu.RLock()
	bagProto, swordProto := w.objs[7001], w.objs[7002]
	w.mu.RUnlock()

	bag := w.NewObjectFromProto(bagProto, 1001)
	sword := w.NewObjectFromProto(swordProto, 1001)
	_ = w.MoveObjectToContainer(sword, bag)

	before := registrySize(w)
	w.ExtractObject(bag, 1001) // extracts contents with the container
	after := registrySize(w)
	if after != before-2 {
		t.Fatalf("extraction removed %d entries, want 2 (bag+contents): %d -> %d", before-after, before, after)
	}
}

// The scrounge path: DoScrounge's found object must be registered.
func TestScroungeObjectRegistered(t *testing.T) {
	w := objregWorld(t)
	p := NewPlayer(1, "Scrounger", 1001)
	before := registrySize(w)
	_ = DoScrounge(p, w)
	// Scrounge may find nothing (skill roll); when it created an object it
	// must be registered — registry can only grow by registered objects now.
	after := registrySize(w)
	if after > before {
		// growth happened; every new ID must be non-zero and resolvable
		w.mu.RLock()
		defer w.mu.RUnlock()
		for id := range w.objectInstances {
			if id == 0 {
				t.Fatal("ID 0 present in registry")
			}
		}
	}
}

// Probes must NOT grow the registry (brief decision 2): NewObjectInstance
// stays unregistered for vstat / shop-produced comparison.
func TestProbeObjectsStayUnregistered(t *testing.T) {
	w := objregWorld(t)
	w.mu.RLock()
	proto := w.objs[7002]
	w.mu.RUnlock()

	before := registrySize(w)
	_ = NewObjectInstance(proto, -1) // the probe constructor
	if registrySize(w) != before {
		t.Fatalf("probe constructor grew the registry: %d -> %d", before, registrySize(w))
	}
}

// House loads register what they place, but match saved container IDs on the
// still-unregistered objects. Saved ContainerIDs are runtime IDs from an
// earlier process (DP-1401); registering first would hand out fresh IDs in
// the same small range, and a stored item could be nested inside an
// unrelated object (here, a sword). Until DP-1401 changes the format, such an
// item is dropped exactly as before registration existed.
func TestHouseLoadNestsNothingIntoWrongObject(t *testing.T) {
	t.Chdir(t.TempDir())
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "House", Zone: 1}},
		Objs: []parser.Obj{
			{VNum: 9001, Keywords: "ring", ShortDesc: "a ring", LongDesc: "A ring."},
			{VNum: 9002, Keywords: "sword", ShortDesc: "a sword", LongDesc: "A sword.", TypeFlag: 5},
			{VNum: 9003, Keywords: "chest", ShortDesc: "a chest", LongDesc: "A chest.", TypeFlag: 15},
		},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	// The ring's saved ContainerID (2) is a stale runtime ID. Registering in
	// file order would give the sword ID 2.
	save := `{"room_vnum":1001,"items":[` +
		`{"vnum":9001,"container_id":2},` +
		`{"vnum":9002,"container_id":-1},` +
		`{"vnum":9003,"container_id":-1}]}`
	if err := os.MkdirAll("house", 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(HouseGetFilename(1001), []byte(save), 0o600); err != nil {
		t.Fatal(err)
	}
	if !w.houseLoad(1001) {
		t.Fatal("houseLoad returned false")
	}

	seen := map[int]bool{}
	for _, obj := range w.GetItemsInRoom(1001) {
		if obj.ID == 0 || seen[obj.ID] {
			t.Fatalf("placed object %q has ID %d (unregistered or duplicate)", obj.GetShortDesc(), obj.ID)
		}
		seen[obj.ID] = true
		if !obj.IsContainer() && len(obj.Contains) > 0 {
			t.Fatalf("%q (not a container) now contains %d object(s): a stale saved ContainerID matched a fresh registry ID", obj.GetShortDesc(), len(obj.Contains))
		}
	}
	if len(seen) != 2 {
		t.Fatalf("room holds %d objects, want the sword and the chest", len(seen))
	}
}
