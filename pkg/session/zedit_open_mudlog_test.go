package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/olc"
)

func TestZeditOpenParentMudlog(t *testing.T) {
	testOLCOpenParentMudlog(t, "zedit", "zon",
		"SYSERR: OLC: zedit_save_to_disk:  Can't write zone 30.",
		"Saving all zone information.\r\n", olc.KindZone)
}
