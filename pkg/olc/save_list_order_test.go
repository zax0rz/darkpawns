package olc

import (
	"reflect"
	"sync"
	"testing"
)

func TestSaveListCInsertionOrder(t *testing.T) {
	s := NewSaveList()
	first := DirtyEntry{KindObject, 31}
	second := DirtyEntry{KindRoom, 30}
	third := DirtyEntry{KindMob, 31}
	s.Mark(first.Kind, first.Zone)
	s.Mark(second.Kind, second.Zone)
	s.Mark(third.Kind, third.Zone)
	s.Mark(first.Kind, first.Zone)
	want := []DirtyEntry{third, second, first}
	if got := s.Ordered(); !reflect.DeepEqual(got, want) {
		t.Fatalf("C insertion order = %v, want %v", got, want)
	}
	snapshot := s.Ordered()
	snapshot[0].Zone = 999
	if got := s.Ordered(); !reflect.DeepEqual(got, want) {
		t.Fatalf("snapshot changed registry: %v", got)
	}
	s.Remove(second.Kind, second.Zone)
	s.Remove(second.Kind, second.Zone)
	s.Mark(second.Kind, second.Zone)
	want = []DirtyEntry{second, third, first}
	if got := s.Ordered(); !reflect.DeepEqual(got, want) {
		t.Fatalf("remove/re-mark = %v, want %v", got, want)
	}
	for _, entry := range want {
		s.Remove(entry.Kind, entry.Zone)
	}
	if len(s.Ordered()) != 0 || len(s.List()) != 0 {
		t.Fatal("empty list retains entries")
	}
}

func TestSaveListOrderedConcurrentSnapshots(t *testing.T) {
	s := NewSaveList()
	var wg sync.WaitGroup
	for zone := 1; zone <= 8; zone++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 100 {
				s.Mark(KindRoom, zone)
				s.Ordered()
				s.List()
				s.Remove(KindRoom, zone)
			}
			s.Mark(KindRoom, zone)
		}()
	}
	wg.Wait()
	if len(s.Ordered()) != 8 || len(s.List()) != 8 {
		t.Fatal("concurrent marks lost entries")
	}
	for _, entry := range s.Ordered() {
		if !s.Dirty(entry.Kind, entry.Zone) {
			t.Fatal("ordered entry has no dirty marker")
		}
	}
}
