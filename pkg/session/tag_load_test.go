package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestUntagItIfPlayingPresent(t *testing.T) {
	f := newConPlayingFixture(t)

	// Non-CON_PLAYING descriptors holding PLR_IT (editing, menu, linkless)
	// must NOT clear PLR_IT from a loading player (src/db.c:2514).
	f.editing.player.SetPlrFlag(game.PlrIt, true)
	f.menu.player.SetPlrFlag(game.PlrIt, true)
	f.linkless.player.SetPlrFlag(game.PlrIt, true)

	loading := &game.Player{Name: "Tagstale"}
	loading.SetPlrFlag(game.PlrIt, true)
	f.manager.untagItIfPlayingPresent(loading)
	if !loading.HasPLRFlag(game.PlrIt) {
		t.Fatal("untagItIfPlayingPresent cleared PLR_IT when only non-CON_PLAYING sessions held it")
	}

	// A CON_PLAYING non-NPC player holding PLR_IT clears PLR_IT on load.
	f.playing.player.SetPlrFlag(game.PlrIt, true)
	f.manager.untagItIfPlayingPresent(loading)
	if loading.HasPLRFlag(game.PlrIt) {
		t.Fatal("untagItIfPlayingPresent kept PLR_IT despite a CON_PLAYING IT holder (db.c:2513-2519)")
	}
}
