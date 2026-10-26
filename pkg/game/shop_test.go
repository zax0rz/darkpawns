package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestShopPriceUsesCIntegerOrder(t *testing.T) {
	shop := &Shop{ProfitBuy: 1.5, ProfitSell: 0.5}

	// C truncates the markup before applying charisma, then truncates the
	// compound assignment. Applying charisma to the float markup would return
	// 4 for this case instead of C's 3.
	if got := shop.BuyPrice(3, 10); got != 3 {
		t.Fatalf("BuyPrice(3, 10) = %d, want 3", got)
	}
	if got := shop.SellPrice(3, 10); got != 1 {
		t.Fatalf("SellPrice(3, 10) = %d, want 1", got)
	}
}

// C ok_shop_room compares configured room entries to real_room(1), even
// though the ordinary comparison uses the current room's virtual number.
func TestShopRoomWildcardUsesCRealRoomIndex(t *testing.T) {
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 0}, {VNum: 1}, {VNum: 1001}}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	for _, tc := range []struct {
		rooms []int
		room  int
		want  bool
	}{
		{[]int{1001}, 1001, true}, {[]int{1001}, 0, false}, {[]int{1}, 1001, true}, {nil, 1001, false},
	} {
		if got := w.ShopAllowsRoom(&Shop{Rooms: tc.rooms}, tc.room); got != tc.want {
			t.Errorf("rooms=%v room=%d got=%t", tc.rooms, tc.room, got)
		}
	}
}
