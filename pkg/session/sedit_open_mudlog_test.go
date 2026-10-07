package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/olc"
)

func TestSeditOpenParentMudlog(t *testing.T) {
	testOLCOpenParentMudlog(t, "sedit", "shp", "SYSERR: OLC: Cannot open shop file!", "Saving all shops in zone.\r\n", olc.KindShop)
}
