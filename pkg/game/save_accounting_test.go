package game

import (
	"testing"
	"time"
)

func TestCharacterSaveAccountingBoundaries(t *testing.T) {
	old := characterNow
	t.Cleanup(func() { characterNow = old })
	characterNow = func() time.Time { return time.Unix(100000, 0) }
	p := NewPlayer(1, "Saver", 1)
	p.PlayedDuration = 3600
	p.ConnectedAt = time.Unix(95000, 0)
	characterNow = func() time.Time { return time.Unix(100125, 0) }
	p.AccountCharacterSave()
	if p.PlayedDuration != 3725 || p.LastLogon != 100125 {
		t.Fatalf("first save: played=%d last=%d", p.PlayedDuration, p.LastLogon)
	}
	characterNow = func() time.Time { return time.Unix(100185, 0) }
	p.AccountCharacterSave()
	if p.PlayedDuration != 3785 || p.LastLogon != 100185 {
		t.Fatalf("second save double-counted: played=%d last=%d", p.PlayedDuration, p.LastLogon)
	}
	if !p.ConnectedAt.Equal(time.Unix(95000, 0)) {
		t.Fatal("save changed connection age")
	}
	characterNow = func() time.Time { return time.Unix(103785, 0) }
	if got := p.PlayingTime(); got.Hours != 2 {
		t.Fatalf("live reader double-counted: %+v", got)
	}
}
