package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestConPlayingDescriptorFanoutMatchesCStates(t *testing.T) {
	f := newConPlayingFixture(t)
	const worldMessage = "The weather shifts.\r\n"
	f.manager.world.SendToAll(worldMessage)
	for _, s := range []*Session{f.actor, f.playing, f.mailing, f.noteWrite} {
		if got := drainSessionText(t, s); !equalStrings(got, []string{worldMessage}) {
			t.Errorf("%s world broadcast = %q, want %q", s.playerName, got, worldMessage)
		}
	}
	for name, s := range map[string]*Session{
		"menu": f.menu, "character creation": f.creating, "OLC editor": f.editing, "TEDIT": f.textEditing, "linkless": f.linkless,
	} {
		if got := drainSessionText(t, s); len(got) != 0 {
			t.Errorf("%s descriptor received world broadcast: %q", name, got)
		}
	}

	for _, s := range []*Session{f.playing, f.mailing, f.noteWrite, f.menu, f.creating, f.editing, f.textEditing, f.linkless} {
		s.player.SetRoom(1002)
	}
	f.playing.player.SetPosition(combat.PosSleeping)
	const zoneMessage = "A distant horn sounds.\r\n"
	f.manager.world.SendToZone(1001, zoneMessage)
	if got := drainSessionText(t, f.actor); len(got) != 0 {
		t.Errorf("same-room actor received send_to_zone: %q", got)
	}
	if got := drainSessionText(t, f.playing); len(got) != 0 {
		t.Errorf("sleeping player received send_to_zone: %q", got)
	}
	for _, s := range []*Session{f.mailing, f.noteWrite} {
		if got := drainSessionText(t, s); !equalStrings(got, []string{zoneMessage}) {
			t.Errorf("%s send_to_zone bytes = %q, want %q", s.playerName, got, zoneMessage)
		}
	}
	for name, s := range map[string]*Session{
		"menu": f.menu, "character creation": f.creating, "OLC editor": f.editing, "TEDIT": f.textEditing, "linkless": f.linkless,
	} {
		if got := drainSessionText(t, s); len(got) != 0 {
			t.Errorf("%s descriptor received send_to_zone: %q", name, got)
		}
	}
}
