package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/olc"
)

func TestReditOpenParentMudlog(t *testing.T) {
	testOLCOpenParentMudlog(t, "redit", "wld", "SYSERR: OLC: Cannot open room file!", "Saving all rooms in zone.\r\n", olc.KindRoom)
}
