package game

import (
	"strconv"
	"testing"
)

func TestMilestoneMudlogAdvance(t *testing.T) {
	for _, invis := range []int{0, 34, 40} {
		t.Run(strconv.Itoa(invis), func(t *testing.T) { testMilestoneMudlog(t, "advance", invis) })
	}
}
