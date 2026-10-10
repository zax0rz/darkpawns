package game

import (
	"strconv"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestZoneResetLoopIterationMatrix(t *testing.T) {
	for _, iterations := range []int{-1, 0, 1, 2, 5} {
		t.Run(strconv.Itoa(iterations), func(t *testing.T) {
			w, s := newZoneResetTestSpawner(t)
			calls := installZoneObjectOrderHooks(t, true)
			s.world.zoneResetMu.Lock()
			err := s.executeZoneResetLocked(&parser.Zone{Commands: []parser.ZoneCommand{
				{Command: "L", Arg1: 100, Arg3: iterations},
				{Command: "O", IfFlag: 1, Arg1: 200, Arg2: 10, Arg3: 100},
				{Command: "L", IfFlag: 1, Arg1: 100, Arg2: 1},
				{Command: "O", IfFlag: 1, Arg1: 201, Arg2: 1, Arg3: 100},
			}})
			s.world.zoneResetMu.Unlock()
			if err != nil {
				t.Fatal(err)
			}
			want := max(1, iterations)
			if w.countObjectInstances(200) != want || w.countObjectInstances(201) != 1 || len(*calls) != want+1 {
				t.Fatalf("L iterations=%d produced body=%d successor=%d draws=%d", iterations, w.countObjectInstances(200), w.countObjectInstances(201), len(*calls))
			}
		})
	}
}

func TestZoneResetLoopUsesSingleCounter(t *testing.T) {
	w, s := newZoneResetTestSpawner(t)
	s.world.zoneResetMu.Lock()
	err := s.executeZoneResetLocked(&parser.Zone{Commands: []parser.ZoneCommand{
		{Command: "L", Arg1: 100, Arg3: 3},
		{Command: "O", Arg1: 200, Arg2: 10, Arg3: 100},
		{Command: "L", Arg1: 100, Arg3: 2},
		{Command: "O", Arg1: 201, Arg2: 10, Arg3: 100},
		{Command: "L", IfFlag: 1, Arg1: 100, Arg2: 1},
		{Command: "O", Arg1: 204, Arg2: 10, Arg3: 100},
		{Command: "L", IfFlag: 1, Arg1: 100, Arg2: 1},
	}})
	s.world.zoneResetMu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	if w.countObjectInstances(200) != 1 || w.countObjectInstances(201) != 2 || w.countObjectInstances(204) != 1 {
		t.Fatal("nested L did not replace C's single saved loop/counter")
	}
}

func TestZoneResetConditionalStateMatrix(t *testing.T) {
	for _, tc := range []struct {
		name   string
		middle parser.ZoneCommand
		want   bool
	}{
		{"failed-conditional", parser.ZoneCommand{Command: "O", IfFlag: 1, Arg1: 200, Arg2: 1, Arg3: 100}, true},
		{"failed-unconditional", parser.ZoneCommand{Command: "O", Arg1: 200, Arg2: 1, Arg3: 100}, false},
		{"conditional-comment", parser.ZoneCommand{Command: "*", IfFlag: 1}, true},
		{"unconditional-comment", parser.ZoneCommand{Command: "*"}, false},
		{"conditional-loop-end", parser.ZoneCommand{Command: "L", IfFlag: 1, Arg1: 100, Arg2: 1}, true},
		{"unconditional-loop-end", parser.ZoneCommand{Command: "L", Arg1: 100, Arg2: 1}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, s := newZoneResetTestSpawner(t)
			s.world.zoneResetMu.Lock()
			err := s.executeZoneResetLocked(&parser.Zone{Commands: []parser.ZoneCommand{
				{Command: "O", Arg1: 200, Arg2: 1, Arg3: 100},
				tc.middle,
				{Command: "O", IfFlag: 1, Arg1: 201, Arg2: 1, Arg3: 100},
			}})
			s.world.zoneResetMu.Unlock()
			if err != nil {
				t.Fatal(err)
			}
			if got := w.GetObjNum(201) != nil; got != tc.want {
				t.Fatalf("last_cmd for %s: successor=%v want %v", tc.name, got, tc.want)
			}
		})
	}
}

func TestZoneResetSkippedLoopDoesNotStart(t *testing.T) {
	w, s := newZoneResetTestSpawner(t)
	s.world.zoneResetMu.Lock()
	err := s.executeZoneResetLocked(&parser.Zone{Commands: []parser.ZoneCommand{
		{Command: "L", IfFlag: 1, Arg1: 100, Arg3: 5},
		{Command: "O", IfFlag: 1, Arg1: 201, Arg2: 1, Arg3: 100},
		{Command: "O", Arg1: 200, Arg2: 10, Arg3: 100},
		{Command: "L", IfFlag: 1, Arg1: 100, Arg2: 1},
	}})
	s.world.zoneResetMu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	if w.countObjectInstances(200) != 1 || w.GetObjNum(201) != nil {
		t.Fatal("skipped conditional L changed loop or last_cmd")
	}
}
