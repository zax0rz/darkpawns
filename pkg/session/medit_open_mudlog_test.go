package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/olc"
)

func TestMeditOpenParentMudlog(t *testing.T) {
	testOLCOpenParentMudlog(t, "medit", "mob", "SYSERR: OLC: Cannot open mob file!", "Saving all mobiles in zone.\r\n", olc.KindMob)
}
