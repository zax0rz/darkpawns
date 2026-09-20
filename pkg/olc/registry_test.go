package olc

import (
	"testing"
	"time"
)

type testOwner struct {
	id       string
	name     string
	frontend Frontend
}

func (o testOwner) Identity() string    { return o.id }
func (o testOwner) DisplayName() string { return o.name }
func (o testOwner) Frontend() Frontend  { return o.frontend }

func TestRegistryUsesTypedKeysAndStableOwners(t *testing.T) {
	r := NewRegistry()
	first := testOwner{id: "first", name: "First", frontend: FrontendTelnet}
	reconnected := testOwner{id: "first", name: "First", frontend: FrontendTelnet}
	second := testOwner{id: "second", name: "Second", frontend: Frontend("web")}

	if _, ok := r.Claim(KindRoom, 3001, first); !ok {
		t.Fatal("first owner was refused")
	}
	if _, ok := r.Claim(KindRoom, 3001, reconnected); !ok {
		t.Fatal("same stable owner was refused")
	}
	holder, ok := r.Claim(KindRoom, 3001, second)
	if ok || holder != first {
		t.Fatalf("second owner claim = (%v, %t), want (first, false)", holder, ok)
	}
	if _, ok := r.Claim(KindMob, 3001, second); !ok {
		t.Fatal("different OLC kind collided with room claim")
	}

	r.Release(KindRoom, 3001, second)
	if _, ok := r.Holder(KindRoom, 3001); !ok {
		t.Fatal("stale owner release dropped another owner's claim")
	}
	r.Release(KindRoom, 3001, reconnected)
	if _, ok := r.Holder(KindRoom, 3001); ok {
		t.Fatal("stable owner release did not drop claim")
	}
	if got := HolderDescription(second); got != "Second (web)" {
		t.Fatalf("holder description = %q, want %q", got, "Second (web)")
	}
}

func TestSaveListIsTypedByKindAndZone(t *testing.T) {
	s := NewSaveList()
	s.Mark(KindObject, 30)
	if !s.Dirty(KindObject, 30) {
		t.Fatal("object zone was not marked dirty")
	}
	if s.Dirty(KindMob, 30) || s.Dirty(KindObject, 31) {
		t.Fatal("dirty marker leaked across kind or zone")
	}
	s.Remove(KindObject, 30)
	if s.Dirty(KindObject, 30) {
		t.Fatal("object zone marker was not removed")
	}
}

func TestRegistryListSnapshotsClaimsAndAcquisitionTime(t *testing.T) {
	r := NewRegistry()
	owner := &testOwner{id: "builder-1", name: "Builder", frontend: FrontendTelnet}
	other := testOwner{id: "builder-2", name: "Other", frontend: Frontend("web")}

	before := time.Now()
	if _, ok := r.Claim(KindRoom, 3001, owner); !ok {
		t.Fatal("room claim was refused")
	}
	after := time.Now()
	if _, ok := r.Claim(KindMob, 1200, other); !ok {
		t.Fatal("mob claim was refused")
	}

	entries := r.List()
	if len(entries) != 2 {
		t.Fatalf("claim list length = %d, want 2", len(entries))
	}
	if entries[0].Kind != KindMob || entries[1].Kind != KindRoom {
		t.Fatalf("claim list order = %#v, want mob then room", entries)
	}
	room := entries[1]
	if room.Number != 3001 || room.OwnerIdentity != "builder-1" ||
		room.OwnerDisplayName != "Builder" || room.OwnerFrontend != FrontendTelnet {
		t.Fatalf("room claim snapshot = %#v", room)
	}
	if room.ClaimedAt.Before(before) || room.ClaimedAt.After(after) {
		t.Fatalf("claim time %v is outside [%v, %v]", room.ClaimedAt, before, after)
	}

	owner.name = "Renamed"
	if _, ok := r.Claim(KindRoom, 3001, *owner); !ok {
		t.Fatal("same stable owner was refused")
	}
	again := r.List()[1]
	if !again.ClaimedAt.Equal(room.ClaimedAt) {
		t.Fatalf("repeated claim changed acquisition time from %v to %v", room.ClaimedAt, again.ClaimedAt)
	}
	if room.OwnerDisplayName != "Builder" {
		t.Fatalf("previous list entry changed through owner mutation: %#v", room)
	}
}

func TestSaveListListReturnsSortedValueSnapshot(t *testing.T) {
	s := NewSaveList()
	s.Mark(KindObject, 31)
	s.Mark(KindMob, 30)
	s.Mark(KindObject, 30)

	entries := s.List()
	want := []DirtyEntry{
		{Kind: KindMob, Zone: 30},
		{Kind: KindObject, Zone: 30},
		{Kind: KindObject, Zone: 31},
	}
	if len(entries) != len(want) {
		t.Fatalf("dirty list length = %d, want %d", len(entries), len(want))
	}
	for i := range want {
		if entries[i] != want[i] {
			t.Fatalf("dirty entry %d = %#v, want %#v", i, entries[i], want[i])
		}
	}
	entries[0].Zone = 999
	if got := s.List()[0]; got != want[0] {
		t.Fatalf("mutating returned entry changed list: %#v", got)
	}
}
