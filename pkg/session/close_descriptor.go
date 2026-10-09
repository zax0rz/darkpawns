package session

import (
	"fmt"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// close_socket's non-playing producers (src/comm.c:2134-2143):
//
//	if (d->character) { ... if (d->connected == CON_PLAYING) { "Closing link
//	  to: %s." } else { "Losing player: %s." (or "<null>" when the character has
//	  no name) } } else { "Losing descriptor without char." }
//
// All three are CMP / LVL_IMMORT / file TRUE. The CON_PLAYING arm is ported
// already where the linkdead transition happens (manager.go,
// idle_close.go, limits_misc.go); loseDescriptor covers the other two and is
// idempotent, because Go closes a descriptor from both its transport and the
// manager while C's close_socket runs exactly once per descriptor.

// markDescriptorBound records C's `d->character` existing. C creates the
// character on the first input at the name prompt, before it even checks for an
// empty line (src/interpreter.c:1743-1752), so a bound descriptor is one the
// name prompt has heard from.
func (s *Session) markDescriptorBound() {
	s.descriptorBound = true
}

// LoseDescriptor emits close_socket's producer for a descriptor that never
// played, choosing the arm from the descriptor's C state. It is a no-op the
// second time (and for a playing descriptor).
func (s *Session) LoseDescriptor() {
	s.descriptorLossOnce.Do(func() {
		if !s.descriptorBound {
			// C has no character on this descriptor: the client dropped
			// before its first line at the name prompt.
			game.MudLog("Losing descriptor without char.", game.MudlogComplete, game.LVL_IMMORT, true)
			return
		}
		game.MudLog(fmt.Sprintf("Losing player: %s.", s.descriptorCharacterName()), game.MudlogComplete, game.LVL_IMMORT, true)
	})
}

// LoseDescriptorWithoutChar is the arm a descriptor takes when C's
// perform_dupe_check has already cleared d->character (src/interpreter.c:1551-1576)
// before the descriptor is closed: close_socket then finds no character at all.
func (s *Session) LoseDescriptorWithoutChar() {
	s.descriptorLossOnce.Do(func() {
		game.MudLog("Losing descriptor without char.", game.MudlogComplete, game.LVL_IMMORT, true)
	})
}

// descriptorCharacterName is C's GET_NAME(d->character) at close time, with
// C's "<null>" for a character that has no name yet. The name lives on the
// loaded player once one is attached, and on the entry name before that.
func (s *Session) descriptorCharacterName() string {
	if s.player != nil && s.player.Name != "" {
		return s.player.Name
	}
	if s.charName != "" {
		return s.charName
	}
	return "<null>"
}
