package session

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// zeditLogNewZoneOpenFailure covers C's five fopen failures only. C does not
// check fprintf/fclose; a Go write/close failure must not invent that broadcast.
func zeditLogNewZoneOpenFailure(err error) {
	var failure *os.PathError
	if !errors.As(err, &failure) || failure.Op != "open" {
		return
	}
	switch filepath.Ext(failure.Path) {
	case ".zon":
		game.MudLog("SYSERR: OLC: Can't write new zone file", game.MudlogBrief, game.LVL_IMPL, true)
	case ".wld":
		game.MudLog("SYSERR: OLC: Can't write new world file", game.MudlogBrief, game.LVL_IMPL, true)
	}
}
