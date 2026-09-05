package parser

import (
	"os"
	"path/filepath"
	"testing"
)

// TestParseAllShopFiles_FieldOrder pins boot_the_shops' field order
// (shop.c:1145-1219) against a synthetic shop file: producing list, two
// profits, trade-type list (with the sscanf("%d%s") "10wheat" buy-word
// shape), seven message strings, temper, BITVECTOR, KEEPER, trade-with,
// room list, and the four open/close hours.
func TestParseAllShopFiles_FieldOrder(t *testing.T) {
	dir := t.TempDir()
	record := "" +
		"#99001~\n" + // shop header vnum
		"99001\n99002\n99003\n-1\n" + // producing list
		"1.40\n" + // buy profit
		"0.60\n" + // sell profit
		"10wheat\n-1\n" + // trade types (buy word attached)
		"%s No such item one~\n" +
		"%s No such item two~\n" +
		"%s Do not buy~\n" +
		"%s Missing cash one~\n" +
		"%s Missing cash two~\n" +
		"%s Buy message~\n" +
		"%s Sell message~\n" +
		"0\n" + // temper
		"0\n" + // bitvector (no WILL_START_FIGHT)
		"99100\n" + // keeper vnum
		"0\n" + // trade with
		"99105\n-1\n" + // in-room list
		"0\n24\n-1\n-1\n" // open1/close1/open2/close2
	if err := os.WriteFile(filepath.Join(dir, "99.shp"), []byte(record), 0o600); err != nil {
		t.Fatalf("write shop file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index"), []byte("99.shp\n$\n"), 0o600); err != nil {
		t.Fatalf("write index: %v", err)
	}

	shops, err := ParseAllShopFiles(dir)
	if err != nil {
		t.Fatalf("ParseAllShopFiles: %v", err)
	}
	if len(shops) != 1 {
		t.Fatalf("parsed %d shops, want 1", len(shops))
	}
	shop := shops[0]
	if shop.VNum != 99001 {
		t.Errorf("shop vnum = %d, want 99001", shop.VNum)
	}
	if len(shop.Products) != 3 || shop.Products[0] != 99001 || shop.Products[2] != 99003 {
		t.Errorf("products = %v, want [99001 99002 99003]", shop.Products)
	}
	if shop.BuyProfit != 1.40 || shop.SellProfit != 0.60 {
		t.Errorf("profits = %.2f/%.2f, want 1.40/0.60", shop.BuyProfit, shop.SellProfit)
	}
	if len(shop.BuyTypes) != 1 || shop.BuyTypes[0] != 10 {
		t.Errorf("buy types = %v, want [10]", shop.BuyTypes)
	}
	if shop.Messages[0] != "%s No such item one" || shop.Messages[6] != "%s Sell message" {
		t.Errorf("messages = %#v, want parsed seven shop messages", shop.Messages)
	}
	if shop.Temper != 0 {
		t.Errorf("temper = %d, want 0", shop.Temper)
	}
	if shop.KeeperVNum != 99100 {
		t.Errorf("keeper vnum = %d, want 99100", shop.KeeperVNum)
	}
	if shop.Bitvector != 0 {
		t.Errorf("bitvector = %d, want 0", shop.Bitvector)
	}
	if shop.WithWho != 0 || len(shop.Rooms) != 1 || shop.Rooms[0] != 99105 {
		t.Errorf("with-who/rooms = %d/%v, want 0/[99105]", shop.WithWho, shop.Rooms)
	}
	if shop.OpenHour1 != 0 || shop.CloseHour1 != 24 || shop.OpenHour2 != -1 || shop.CloseHour2 != -1 {
		t.Errorf("hours = %d %d %d %d, want 0 24 -1 -1", shop.OpenHour1, shop.CloseHour1, shop.OpenHour2, shop.CloseHour2)
	}
}

// TestParseAllShopFiles_MissingIndex — a world tree without a shop index has
// no shops; parsing is a no-op, not an error.
func TestParseAllShopFiles_MissingIndex(t *testing.T) {
	shops, err := ParseAllShopFiles(t.TempDir())
	if err != nil {
		t.Fatalf("ParseAllShopFiles on empty dir: %v", err)
	}
	if len(shops) != 0 {
		t.Fatalf("parsed %d shops from empty dir, want 0", len(shops))
	}
}
