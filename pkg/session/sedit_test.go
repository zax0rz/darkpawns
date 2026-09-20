package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func makeSeditTestWorld(t *testing.T) *game.World {
	t.Helper()
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 3000, Name: "Sedit Test Room", Zone: 30},
			{VNum: 3001, Name: "Second Sedit Room", Zone: 30},
		},
		Mobs: []parser.Mob{
			{VNum: 3002, ShortDesc: "a shopkeeper", Keywords: "shopkeeper", ActionFlags: []string{"ISNPC"}},
		},
		Objs: []parser.Obj{
			{VNum: 3003, ShortDesc: "a shop product", Keywords: "product"},
		},
		Zones: []parser.Zone{{Number: 30, Name: "Sedit Test Zone", TopRoom: 3099}},
		Shops: []parser.ShopProto{{
			VNum:       3001,
			Products:   []int{3003},
			BuyProfit:  1.25,
			SellProfit: 0.75,
			BuyTypes:   []int{10},
			BuyWords:   []string{"wheat"},
			Messages:   [7]string{"no1", "no2", "nobuy", "cash1", "cash2", "buy", "sell"},
			KeeperVNum: 3002,
			Rooms:      []int{3000},
			CloseHour1: 24,
		}},
		SourceDir: t.TempDir(),
	}
	w, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })
	return w
}

func makeSeditTestSession(t *testing.T, m *Manager, name string, level int) *Session {
	t.Helper()
	s := makeCommandTestSession(t, m, name, level, 3000)
	s.olcZone = 30
	return s
}

func TestSeditRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["sedit"]
	if !ok {
		t.Fatal("sedit command has no C gate")
	}
	// src/interpreter.c:684: { "sedit", POS_DEAD, do_olc, LVL_BUILDER, SCMD_OLC_SEDIT }.
	if gate.MinLevel != 31 || gate.MinPosition != 0 {
		t.Fatalf("sedit gate = (%d,%d), want (31,0)", gate.MinLevel, gate.MinPosition)
	}
	if _, ok := cmdRegistry.Lookup("sedit"); !ok {
		t.Fatal("sedit command is not registered")
	}
}

func TestSeditEntryGatesAndNewDefaults(t *testing.T) {
	w := makeSeditTestWorld(t)
	m := newTestManager(t, w, nil)
	builder := makeSeditTestSession(t, m, "Seditbuilder", 35)

	if err := cmdSedit(builder, nil); err != nil {
		t.Fatal(err)
	}
	if got := readMsgText(t, builder); got != "Specify a shop VNUM to edit.\r\n" {
		t.Fatalf("no-arg = %q", got)
	}
	if err := cmdSedit(builder, []string{"abc"}); err != nil {
		t.Fatal(err)
	}
	if got := readMsgText(t, builder); got != "Yikes!  Stop that, someone will get hurt!\r\n" {
		t.Fatalf("nonnumeric = %q", got)
	}
	if err := cmdSedit(builder, []string{"99999"}); err != nil {
		t.Fatal(err)
	}
	if got := readMsgText(t, builder); got != "Sorry, there is no zone for that number!\r\n" {
		t.Fatalf("unknown zone = %q", got)
	}

	confined := makeSeditTestSession(t, m, "Confined", 31)
	confined.olcZone = 31
	if err := cmdSedit(confined, []string{"3005"}); err != nil {
		t.Fatal(err)
	}
	if got := readMsgText(t, confined); got != "You do not have permission to edit this zone.\r\n" {
		t.Fatalf("permission = %q", got)
	}

	if err := cmdSedit(builder, []string{"3005"}); err != nil {
		t.Fatal(err)
	}
	got := readMsgText(t, builder)
	if !strings.HasPrefix(got, "\r\n-- Shop Number : [3005]\r\n0) Keeper      : [-1] None\r\n") {
		t.Fatalf("new menu = %q", got)
	}
	builder.textEditMu.Lock()
	state := builder.sedit
	if state.shop.KeeperVNum != -1 || state.shop.CloseHour1 != 28 || state.shop.BuyProfit != 1.0 || state.shop.SellProfit != 1.0 {
		t.Fatalf("new defaults = %+v", state.shop)
	}
	builder.textEditMu.Unlock()
	builder.handleSeditInput("q")
	if got := readMsgTextOrEmpty(t, builder); got != "" {
		t.Fatalf("clean quit = %q", got)
	}
}

func TestSeditEmptyNumericInputAndMessages(t *testing.T) {
	w := makeSeditTestWorld(t)
	m := newTestManager(t, w, nil)
	s := makeSeditTestSession(t, m, "Seditnumeric", 35)
	if err := cmdSedit(s, []string{"3001"}); err != nil {
		t.Fatal(err)
	}
	_ = readMsgText(t, s)
	s.handleSeditInput("1")
	_ = readMsgText(t, s)
	s.handleSeditInput("")
	menu := readMsgText(t, s)
	if strings.Contains(menu, "Field must be numerical") || !strings.Contains(menu, "1) Open 1      :    0") {
		t.Fatalf("empty numeric input = %q", menu)
	}
	s.handleSeditInput("7")
	_ = readMsgText(t, s)
	s.handleSeditInput("hello")
	_ = readMsgText(t, s)
	s.textEditMu.Lock()
	message := s.sedit.shop.Messages[0]
	s.textEditMu.Unlock()
	if message != "%s hello" {
		t.Fatalf("message prepend = %q, want %%s hello", message)
	}
	s.handleSeditInput("q")
	_ = readMsgText(t, s)
	s.handleSeditInput("n")
}

func TestSeditKeeperValidationKeepsPrompt(t *testing.T) {
	w := makeSeditTestWorld(t)
	m := newTestManager(t, w, nil)
	s := makeSeditTestSession(t, m, "Seditkeeper", 35)
	if err := cmdSedit(s, []string{"3001"}); err != nil {
		t.Fatal(err)
	}
	_ = readMsgText(t, s)
	s.handleSeditInput("0")
	if got := readMsgText(t, s); got != "Enter virtual number of shop keeper : " {
		t.Fatalf("keeper prompt = %q", got)
	}
	s.handleSeditInput("9999")
	if got := readMsgText(t, s); got != "That mobile does not exist, try again : " {
		t.Fatalf("invalid keeper = %q", got)
	}
	s.handleSeditInput("-1")
	if got := readMsgText(t, s); !strings.HasPrefix(got, "\r\n-- Shop Number : [") {
		t.Fatalf("valid keeper redraw = %q", got)
	}
	s.handleSeditInput("q")
	_ = readMsgTextOrEmpty(t, s)
}

func TestSeditMenusPreserveCLayout(t *testing.T) {
	w := makeSeditTestWorld(t)
	m := newTestManager(t, w, nil)
	s := makeSeditTestSession(t, m, "Seditmenus", 35)
	if err := cmdSedit(s, []string{"3001"}); err != nil {
		t.Fatal(err)
	}
	_ = readMsgText(t, s)
	s.handleSeditInput("r")
	long := readMsgText(t, s)
	if !strings.HasPrefix(long, "\x1b[H\x1b[J##     VNUM     Room\r\n\r\n") {
		t.Fatalf("long rooms prefix = %q", long)
	}
	s.handleSeditInput("c")
	compact := readMsgText(t, s)
	if strings.Contains(compact, "\x1b[H\x1b[J") {
		t.Fatalf("compact rooms cleared screen: %q", compact)
	}
	if !strings.Contains(compact, " 0 - [ 3000]  | ") {
		t.Fatalf("compact rooms layout = %q", compact)
	}
	s.handleSeditInput("q")
	_ = readMsgTextOrEmpty(t, s)
}

func TestSeditMenuColorsMatchGetCharCols(t *testing.T) {
	for _, tc := range []struct {
		name  string
		level int
	}{
		{name: "color-off", level: 0},
		{name: "color-normal", level: 2},
		{name: "color-complete", level: 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := makeSeditTestWorld(t)
			m := newTestManager(t, w, nil)
			s := makeSeditTestSession(t, m, "Seditcolors", 35)
			s.player.SetPlrFlag(game.PrfColor1, tc.level >= 3)
			s.player.SetPlrFlag(game.PrfColor2, tc.level >= 2)
			if err := cmdSedit(s, []string{"3001"}); err != nil {
				t.Fatal(err)
			}
			menu := readMsgText(t, s)
			if tc.level == 0 && strings.Contains(menu, "\x1b[") {
				t.Fatalf("color-off menu contains ANSI bytes: %q", menu)
			}
			if tc.level >= 2 && (!strings.Contains(menu, "\x1b[32m") || !strings.Contains(menu, "\x1b[36m")) {
				t.Fatalf("color-%d menu is missing expected ANSI bytes: %q", tc.level, menu)
			}
			s.handleSeditInput("q")
			_ = readMsgTextOrEmpty(t, s)
		})
	}
}

func TestSeditSaveRoundTripAndWriter(t *testing.T) {
	w := makeSeditTestWorld(t)
	m := newTestManager(t, w, nil)
	s := makeSeditTestSession(t, m, "Seditwriter", 35)
	if err := cmdSedit(s, []string{"3001"}); err != nil {
		t.Fatal(err)
	}
	_ = readMsgText(t, s)
	s.handleSeditInput("q")
	if got := readMsgTextOrEmpty(t, s); got != "" {
		t.Fatalf("clean existing quit = %q", got)
	}
	if err := cmdSedit(s, []string{"3001"}); err != nil {
		t.Fatal(err)
	}
	_ = readMsgText(t, s)
	s.handleSeditInput("7")
	_ = readMsgText(t, s)
	s.handleSeditInput("%s changed")
	_ = readMsgText(t, s)
	s.handleSeditInput("q")
	if got := readMsgText(t, s); got != "Do you wish to save the changes to the shop? (y/n) : " {
		t.Fatalf("save prompt = %q", got)
	}
	s.handleSeditInput("y")
	if got := readMsgText(t, s); got != "Saving shop to memory.\r\n" {
		t.Fatalf("save output = %q", got)
	}
	shop, ok := w.GetShopByKeeper(3002)
	if !ok || shop.Messages[0] != "%s changed" {
		t.Fatalf("live shop = %+v, found=%v", shop, ok)
	}

	if err := cmdSedit(s, []string{"save", "30"}); err != nil {
		t.Fatal(err)
	}
	if got := readMsgText(t, s); got != "Saving all shops in zone.\r\n" {
		t.Fatalf("zone save output = %q", got)
	}
	data, err := os.ReadFile(filepath.Join(w.GetParsedWorld().SourceDir, "shp", "30.shp"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, fragment := range []string{"CircleMUD v3.0 Shop File~\n", "#3001~\n", "3003\n-1\n1.25\n0.75\n", "%s changed~\n", "3002\n0\n3000\n-1\n", "$~\n"} {
		if !strings.Contains(text, fragment) {
			t.Fatalf("shop file missing %q in %q", fragment, text)
		}
	}
}
