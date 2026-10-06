package session

import (
	"os"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// Tests hit the actual session command/editor path and the live MudLog
// provider. Invalid-mode cases are defensive state injections, not oracle
// or valid-play reachability claims.
func testLocalMudlog(t *testing.T, kind string) {
	t.Helper()
	t.Chdir(t.TempDir())
	w := makeSeditTestWorld(t)
	m := newTestManager(t, w, nil)
	a := makeCommandTestSession(t, m, "Logactor", 40, 3000)
	a.player.SetInvisLevel(40) // these C producers have no invis term
	minimum, typ, toFile := game.LVL_IMMORT, game.MudlogComplete, true
	if kind == "help" {
		typ = game.MudlogNormal
	}
	if kind == "withdraw" || kind == "deposit" || kind == "oedit" || kind == "sedit" {
		typ = game.MudlogBrief
	}
	watch := makeCommandTestSession(t, m, "Logwatch", minimum, 3001)
	watch.player.SetPlrFlag(game.PrfLog1, typ&1 != 0)
	watch.player.SetPlrFlag(game.PrfLog2, typ&2 != 0)
	below := makeCommandTestSession(t, m, "Logbelow", minimum-1, 3001)
	below.player.SetPlrFlag(game.PrfLog1, true)
	below.player.SetPlrFlag(game.PrfLog2, true)
	lowerType := makeCommandTestSession(t, m, "Logless", 40, 3001)
	lowerType.player.SetPlrFlag(game.PrfLog1, (typ-1)&1 != 0)
	lowerType.player.SetPlrFlag(game.PrfLog2, (typ-1)&2 != 0)
	for _, s := range []*Session{a, watch, below, lowerType} {
		registerTestSession(t, m, s, s.player.GetName())
	}
	run := func(cmd string, args ...string) {
		t.Helper()
		if err := ExecuteCommand(a, cmd, args); err != nil {
			t.Fatal(err)
		}
	}
	payload, ack := "", ""
	var invoke func()
	var atLog func()
	switch kind {
	case "oedit":
		run("oedit", "3003")
		a.textEditMu.Lock()
		a.oedit.mode = oeditMode(250)
		a.textEditMu.Unlock()
		payload = "SYSERR: OLC: Reached default case in oedit_parse()!"
		invoke = func() { a.handleOeditInput("x") }
		atLog = func() {
			if a.oedit == nil || a.oedit.olcVal != 0 || len(a.send) != 0 {
				t.Error("oedit log must precede dirty flag and menu")
			}
		}
	case "sedit":
		run("sedit", "3001")
		a.textEditMu.Lock()
		a.sedit.mode = seditMode(999)
		a.textEditMu.Unlock()
		payload = "SYSERR: OLC: sedit_parse(): Reached default case!"
		invoke = func() { a.handleSeditInput("x") }
		atLog = func() {
			if a.sedit != nil || a.player.GetFlags()&(1<<uint(game.PlrWriting)) != 0 {
				t.Error("sedit log must follow cleanup")
			}
		}
	case "help":
		w.HelpTable = []game.HelpEntry{{Keyword: "known", Entry: "known help\r\n"}}
		payload = "HELP: Logactor attempted to get help on no  entry"
		ack = "There is no help on: no  entry\r\n"
		invoke = func() { run("help", "no", "", "entry") }
		atLog = func() {
			if got := strings.Join(drainSessionText(t, a), ""); got != ack {
				t.Errorf("help miss must precede log: %q", got)
			}
		}
	case "withdraw", "deposit":
		w.Clans = game.NewClanManager()
		w.Clans.AddClan(&game.Clan{ID: 1, Name: "Proof", Treasure: 50})
		a.player.SetGold(20)
		a.player.SetSex(game.SexFemale)
		if kind == "withdraw" {
			payload = "Logactor withdraws 10 coins from her clan account."
			ack = "You withdraw from the clan's treasure.\r\n"
		} else {
			payload = "Logactor adds 10 coins to her clan account."
			ack = "You add to the clan's treasure.\r\n"
		}
		invoke = func() { run("clan", kind, "10", "Proof") }
		atLog = func() {
			if got := strings.Join(drainSessionText(t, a), ""); got != ack {
				t.Errorf("clan acknowledgement must precede log: %q", got)
			}
			c := w.Clans.GetClanByIndex(0)
			gold, treasure := 20, int64(60)
			if kind == "withdraw" {
				gold, treasure = 30, 40
			}
			if a.player.GetGold() != gold || c.Treasure != treasure {
				t.Error("bank mutation must precede log")
			}
			if _, err := os.Stat(game.ClanFilePath()); !os.IsNotExist(err) {
				t.Error("bank log must precede clan save")
			}
		}
	default:
		t.Fatal("unknown test case")
	}
	for _, s := range []*Session{a, watch, below, lowerType} {
		drainSessionText(t, s)
	}
	file := captureMudlogFile(t)
	sink := w.MessageSink
	called := false
	w.MessageSink = func(name string, b []byte) {
		if name == watch.player.GetName() {
			called = true
			atLog()
		}
		sink(name, b)
	}
	invoke()
	if !called {
		t.Fatalf("missing %s mudlog producer", kind)
	}
	if got := strings.Join(drainSessionText(t, watch), ""); got != "[ "+payload+" ]\r\n" {
		t.Fatalf("producer bytes=%q want %q", got, payload)
	}
	if strings.Contains(file.String(), payload) != toFile {
		t.Fatalf("wrong file flag: %q", file.String())
	}
	if len(below.send) != 0 || len(lowerType.send) != 0 {
		t.Fatal("producer minimum/type filter leak")
	}
}
