package game

import (
	"strings"
	"testing"
)

func TestSprintbitArraySignExtendsBit31(t *testing.T) {
	names := []string{"ZERO", "ONE"}
	got := SprintbitArray([]uint32{1 << 31}, names, 1)
	want := strings.Repeat("UNDEFINED ", 33)
	if got != want {
		t.Fatalf("SprintbitArray = %q, want %q", got, want)
	}
}

func TestSprintbitArrayWithoutBit31DoesNotExtend(t *testing.T) {
	got := SprintbitArray([]uint32{1}, []string{"ZERO", "ONE"}, 1)
	if got != "ZERO " {
		t.Fatalf("SprintbitArray = %q, want %q", got, "ZERO ")
	}
}

func TestSprintbitArraySignExtensionStillAppendsFollowingWords(t *testing.T) {
	names := make([]string, 36)
	for i := range names {
		names[i] = "NAME"
	}
	names[31], names[32], names[33] = "THIRTYONE", "THIRTYTWO", "THIRTYTHREE"
	got := SprintbitArray([]uint32{1 << 31, 1}, names, 2)
	want := "THIRTYONE THIRTYTWO THIRTYTHREE NAME NAME " + strings.Repeat("UNDEFINED ", 28) + "THIRTYTWO "
	if got != want {
		t.Fatalf("SprintbitArray = %q, want %q", got, want)
	}
}
