package olc

import "testing"

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
