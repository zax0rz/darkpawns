package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestShowShopsBootOrderAndColumns(t *testing.T) {
	parsed := &parser.World{Rooms: []parser.Room{{VNum: 1001}}, Mobs: []parser.Mob{{VNum: 5, ShortDesc: "keeper"}}}
	for i := range 20 {
		parsed.Shops = append(parsed.Shops, parser.ShopProto{VNum: 900 - i, Rooms: []int{1001}, KeeperVNum: 5, BuyProfit: 1.234, SellProfit: 0.567, WithWho: 3})
	}
	w, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	s := makeCommandTestSession(t, newTestManager(t, w, nil), "Showgod", 40, 1001)
	s.wantsStructuredData = true
	if err := cmdShow(s, []string{"shops"}); err != nil {
		t.Fatal(err)
	}
	got := renderedOutput(s)
	const header = " ##   Virtual   Where    Keeper    Buy   Sell   Customers\n\r---------------------------------------------------------\n\r"
	if !strings.HasPrefix(got, "\n\r"+header+"  1      900     1001         5   0.57   1.23    __NMCTW\n\r") {
		t.Fatalf("first row/columns/order: %q", got)
	}
	if strings.Count(got, header) != 2 {
		t.Fatalf("19-row header cadence: %q", got)
	}
	if !strings.Contains(got, " 19      882     1001         5   0.57   1.23    __NMCTW\n\r"+header+" 20      881") {
		t.Fatalf("header boundary/order: %q", got)
	}
	if !strings.HasSuffix(got, " 20      881     1001         5   0.57   1.23    __NMCTW\n\r") {
		t.Fatalf("last row: %q", got)
	}
}

func TestShowShopsEmptyAndMissingKeeper(t *testing.T) {
	for _, shops := range [][]parser.ShopProto{nil, {{VNum: 1, KeeperVNum: -1, BuyProfit: 2, SellProfit: 0.5, WithWho: 127}}} {
		w, err := game.NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001}}, Shops: shops})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(w.StopAITicker)
		s := makeCommandTestSession(t, newTestManager(t, w, nil), "Showgod", 40, 1001)
		if err := cmdShow(s, []string{"shops"}); err != nil {
			t.Fatal(err)
		}
		got := renderedOutput(s)
		if len(shops) == 0 {
			if got != "\n\r" {
				t.Fatalf("empty %q", got)
			}
		} else if !strings.Contains(got, "  1        1       -1    <NONE>   0.50   2.00    _______\n\r") {
			t.Fatalf("missing keeper/room %q", got)
		}
	}
}

// Literal paging opts in narrowly: normal paging retains its existing bytes.
func TestShopLiteralPagerDeliveryAndSnoop(t *testing.T) {
	for _, heartbeat := range []bool{false, true} {
		m := makeTestManager(t)
		actor := makeCommandTestSession(t, m, "Actor", 40, 1001)
		spy := makeCommandTestSession(t, m, "Spy", 40, 1001)
		actor.snoopBy = spy
		if heartbeat {
			m.BeginHeartbeatOutput()
		}
		pageLiteralString(actor, "first\n\rsecond\n\r")
		if heartbeat {
			if len(actor.send) != 0 || len(spy.send) != 0 {
				t.Fatal("literal page escaped heartbeat")
			}
			m.EndHeartbeatOutput()
		}
		if got := renderedOutput(actor); got != "first\n\rsecond\n\r" {
			t.Fatalf("literal actor heartbeat=%v: %q", heartbeat, got)
		}
		if got := renderedOutput(spy); got != "% first\n\rsecond\n\r%%" {
			t.Fatalf("literal snoop heartbeat=%v: %q", heartbeat, got)
		}
		if actor.pagerLiteral || actor.IsPaging() {
			t.Fatal("single page retained literal mode")
		}
		PageString(actor, "normal\n\r")
		if got := renderedOutput(actor); got != "normal\r\n" {
			t.Fatalf("ordinary paging changed: %q", got)
		}
		_ = renderedOutput(spy)
		pageLiteralString(actor, strings.Repeat("row\n\r", 30))
		if !actor.IsPaging() || !actor.pagerLiteral {
			t.Fatal("literal multipage inactive")
		}
		if got := renderedOutput(actor); !strings.Contains(got, "row\n\r") || strings.Contains(got, "row\r\n") {
			t.Fatalf("literal page boundaries: %q", got)
		}
		_ = renderedOutput(spy)
		actor.navigatePager("q")
		if actor.pagerLiteral || actor.IsPaging() {
			t.Fatal("quit retained literal mode")
		}
		PageString(actor, "ordinary\n\r")
		if got := renderedOutput(actor); got != "ordinary\r\n" {
			t.Fatalf("after quit: %q", got)
		}
	}
}

func TestShopLiteralPagerMixedHeartbeatSnoop(t *testing.T) {
	m := makeTestManager(t)
	actor := makeCommandTestSession(t, m, "Actor", 40, 1001)
	spy := makeCommandTestSession(t, m, "Spy", 40, 1001)
	actor.snoopBy = spy
	m.BeginHeartbeatOutput()
	actor.Send("normal\n")
	pageLiteralString(actor, "literal\n\r")
	m.EndHeartbeatOutput()
	if got := renderedOutput(actor); got != "normal\r\nliteral\n\r" {
		t.Fatalf("mixed actor: %q", got)
	}
	if got := renderedOutput(spy); got != "% normal\r\nliteral\n\r%%" {
		t.Fatalf("mixed snoop: %q", got)
	}
}
