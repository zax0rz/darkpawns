package olc

import (
	"testing"
)

type testOwner struct {
	id       string
	name     string
	frontend Frontend
}

func (o testOwner) Identity() string    { return o.id }
func (o testOwner) DisplayName() string { return o.name }
func (o testOwner) Frontend() Frontend  { return o.frontend }

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
