package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// src/comm.c:1285-1286,1376-1378: saved character flags govern entry output.
func TestEntryMOTDRawSavedColor(t *testing.T) {
	for _, flag := range []int{-1, int(game.PrfColor1), int(game.PrfColor2)} {
		s := makeCharSession(t, makeTestManager(t))
		s.player = game.NewPlayer(1, "Motdraw", 8004)
		if flag >= 0 {
			s.player.SetPlrFlag(flag, true)
		}
		s.sendCharCreatePrompt("motd", "&cMOTD&n\r\n\n*** PRESS RETURN: ", nil)
		_, got := unmarshalCharCreate(t, drainMsg(t, s))
		want := "&cMOTD&n\r\n\n*** PRESS RETURN: "
		if flag >= 0 {
			want = "\x1b[0;36mMOTD\x1b[0m\r\n\n*** PRESS RETURN: "
		}
		if got.Prompt != want {
			t.Fatalf("flag %d: prompt %q, want %q", flag, got.Prompt, want)
		}
	}
}
