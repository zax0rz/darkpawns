package session

import (
	"strings"
	"testing"
)

// Exercise registered buy dispatch, preserving C's leading-count syntax and
// affordable-prefix state (src/shop.c:489-589). Gold and objects are proof.
func TestShopCountedBuyStateThroughDispatch(t *testing.T) {
	for _, tc := range []struct {
		command                string
		gold, count, remaining int
		text                   string
	}{
		{"buy 2 apple", 100, 2, 80, "You now have a red apple (x 2).\n\r"},
		{"buy apple 3", 100, 1, 90, "You now have a red apple.\n\r"},
		{"buy 5 apple", 25, 2, 5, "You can only afford 2."},
		{"buy 0 apple", 100, 0, 100, "What do you want to buy??"},
		{"buy -1 apple", 100, 0, 100, "No such item."},
		{"buy apple", 0, 0, 0, "You're too poor!"},
	} {
		t.Run(tc.command+tc.text, func(t *testing.T) {
			_, manager, keeper, _ := shopOutputFixture(t, 1)
			buyer := addShopOutputSession(t, manager, "Countbuyer")
			peer := addShopOutputSession(t, manager, "Countpeer")
			setShopOutputTestAttributes(buyer.player)
			buyer.player.SetLevel(10)
			buyer.player.Inventory.SetCapacity(10, 0, 10, 10)
			buyer.player.SetGold(tc.gold)
			shopDispatch(t, buyer, tc.command)
			if got := buyer.player.GetGold(); got != tc.remaining {
				t.Errorf("gold=%d want %d", got, tc.remaining)
			}
			if got := buyer.player.Inventory.GetItemCount(); got != tc.count {
				t.Errorf("inventory=%d want %d", got, tc.count)
			}
			if got := keeper.GetGold(); got != 1000+tc.gold-tc.remaining {
				t.Errorf("keeper gold=%d", got)
			}
			if got := renderedOutput(buyer); !strings.Contains(got, tc.text) {
				t.Errorf("output=%q want %q", got, tc.text)
			}
			if tc.count > 0 && !strings.Contains(renderedOutput(peer), "Countbuyer buys a red apple") {
				t.Errorf("peer=%q", renderedOutput(peer))
			}
			if tc.count == 0 && strings.Contains(renderedOutput(peer), "buys") {
				t.Errorf("refused purchase reached peer=%q", renderedOutput(peer))
			}
		})
	}
}

func TestShopCountedBuyStopsAtEachCapacity(t *testing.T) {
	for _, weight := range []int{0, 100} {
		_, manager, keeper, _ := shopOutputFixture(t, weight)
		buyer := addShopOutputSession(t, manager, "Capacitybuyer")
		setShopOutputTestAttributes(buyer.player)
		buyer.player.SetLevel(10)
		buyer.player.Inventory.SetCapacity(10, 0, 10, 10)
		buyer.player.SetGold(10000)
		before := buyer.player.GetGold()
		shopDispatch(t, buyer, "buy 1000 apple")
		count := buyer.player.Inventory.GetItemCount()
		if count < 1 {
			t.Fatal("did not buy affordable prefix")
		}
		if count > buyer.player.MaxCarryItems() || buyer.player.CarriedWeight() > buyer.player.MaxCarryWeight() {
			t.Fatal("purchase exceeded C capacity")
		}
		if buyer.player.GetGold() != before-10*count || keeper.GetGold() != 1000+10*count {
			t.Fatal("coins do not match transferred objects")
		}
		expected := "You can only hold"
		if weight > 0 {
			expected = "You can only carry"
		}
		if !strings.Contains(renderedOutput(buyer), expected) {
			t.Errorf("missing partial refusal %q: %q", expected, renderedOutput(buyer))
		}
	}
}

func shopDispatch(t *testing.T, s *Session, line string) {
	t.Helper()
	fields := strings.Fields(line)
	if err := ExecuteCommand(s, fields[0], fields[1:]); err != nil {
		t.Fatal(err)
	}
}
