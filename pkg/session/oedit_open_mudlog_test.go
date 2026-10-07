package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/olc"
)

func TestOeditOpenParentMudlog(t *testing.T) {
	testOLCOpenParentMudlog(t, "oedit", "obj", "SYSERR: OLC: Cannot open objects file!", "Saving all objects in zone.\r\n", olc.KindObject)
}
