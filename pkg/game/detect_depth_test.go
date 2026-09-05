package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newDetectDepthWorld(t *testing.T, exits map[string]parser.Exit) (*World, *Player) {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{
			VNum:  8105,
			Name:  "Detect Room",
			Zone:  1,
			Flags: []string{"0", "0", "0", "0"},
			Exits: exits,
		}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	ch := NewPlayer(1, "Detector", 8105)
	ch.SetPosition(combat.PosStanding)
	return w, ch
}

func TestDoDetectDepthGatesAndOutput(t *testing.T) {
	t.Run("ordinary player without skill", func(t *testing.T) {
		w, ch := newDetectDepthWorld(t, nil)
		dprng.ResetStream(1)
		result := DoDetect(ch, w)
		if result.Success || result.MessageToCh != "Yeah, right.\r\n" || result.WaitChPulses != 0 {
			t.Fatalf("no-skill result = %#v", result)
		}
		gotNext := dprng.Number(1, 101)
		dprng.ResetStream(1)
		wantNext := dprng.Number(1, 101)
		if gotNext != wantNext {
			t.Fatalf("no-skill gate consumed a roll: next=%d want=%d", gotNext, wantNext)
		}
	})

	t.Run("elf bypasses skill gate but still rolls", func(t *testing.T) {
		w, ch := newDetectDepthWorld(t, nil)
		ch.Race = RaceElf
		ch.Class = ClassWarrior
		dprng.ResetStream(1)
		result := DoDetect(ch, w)
		if result.Success || result.MessageToCh != "You carefully check the room...\r\nYou can't seem to find anything.\r\n" {
			t.Fatalf("elf result = %#v", result)
		}
		if result.WaitChPulses != engine.PULSE_VIOLENCE+1 {
			t.Fatalf("elf wait pulses = %d, want %d", result.WaitChPulses, engine.PULSE_VIOLENCE+1)
		}
	})

	t.Run("blind gate precedes room check and roll", func(t *testing.T) {
		w, ch := newDetectDepthWorld(t, nil)
		ch.SetSkill(SkillDetect, 100)
		ch.SetAffect(affBlind, true)
		dprng.ResetStream(1)
		result := DoDetect(ch, w)
		if result.Success || result.MessageToCh != "You're fucking blind, you can't find anything!!\r\n" || result.WaitChPulses != 0 {
			t.Fatalf("blind result = %#v", result)
		}
		gotNext := dprng.Number(1, 101)
		dprng.ResetStream(1)
		wantNext := dprng.Number(1, 101)
		if gotNext != wantNext {
			t.Fatalf("blind gate consumed a roll: next=%d want=%d", gotNext, wantNext)
		}
	})
}

func TestDoDetectDepthSuccessOrderAndSecretMark(t *testing.T) {
	w, ch := newDetectDepthWorld(t, map[string]parser.Exit{
		"north": {Direction: "north", ToRoom: 8161, Keywords: "secret"},
		"east":  {Direction: "east", ToRoom: 8161, Keywords: "ordinary"},
		"up":    {Direction: "up", ToRoom: 8161, Keywords: "secret"},
		"down":  {Direction: "down", ToRoom: 8161, Keywords: "secret"},
	})
	ch.SetSkill(SkillDetect, 100)
	dprng.ResetStream(1)
	result := DoDetect(ch, w)
	want := "You carefully check the room...\r\n" +
		"You notice something funny about the north wall.\r\n" +
		"You notice something funny about the ceiling.\r\n" +
		"You notice something funny about the floor.\r\n"
	if !result.Success || result.MessageToCh != want {
		t.Fatalf("success result = %#v, want message %q", result, want)
	}
	if result.WaitChPulses != 0 {
		t.Fatalf("successful detect wait pulses = %d, want 0", result.WaitChPulses)
	}
	room := w.GetRoomInWorld(8105)
	if !movementRoomHasFlag(room, roomFlagSecretMark, "secret_mark") {
		t.Fatal("successful detect did not set ROOM_SECRET_MARK")
	}
}

func TestDoDetectDepthFailureMessageWaitAndRollRange(t *testing.T) {
	w, ch := newDetectDepthWorld(t, nil)
	ch.SetSkill(SkillDetect, 1)
	dprng.ResetStream(1)
	result := DoDetect(ch, w)
	if result.Success || result.MessageToCh != "You carefully check the room...\r\nYou can't seem to find anything.\r\n" {
		t.Fatalf("failure result = %#v", result)
	}
	if result.WaitChPulses != engine.PULSE_VIOLENCE+1 {
		t.Fatalf("failure wait pulses = %d, want %d", result.WaitChPulses, engine.PULSE_VIOLENCE+1)
	}

	var boundarySeed uint32
	for candidate := uint32(1); candidate <= 100000; candidate++ {
		dprng.ResetStream(candidate)
		cRoll := dprng.Number(1, 101)
		dprng.ResetStream(candidate)
		oldGoRoll := dprng.Number(1, 100)
		if cRoll == 100 && oldGoRoll == 99 {
			boundarySeed = candidate
			break
		}
	}
	if boundarySeed == 0 {
		t.Fatal("did not find a seed that distinguishes C number(1,101) from number(1,100)")
	}
	ch.SetSkill(SkillDetect, 100)
	dprng.ResetStream(boundarySeed)
	result = DoDetect(ch, w)
	if result.Success || result.MessageToCh != "You carefully check the room...\r\nYou can't seem to find anything.\r\n" {
		t.Fatalf("upper-bound result = %#v at seed %d", result, boundarySeed)
	}
}
