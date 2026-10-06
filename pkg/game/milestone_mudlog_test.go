package game

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

type milestoneProvider struct {
	players []*Player
	lines   map[string]string
}

func (p *milestoneProvider) EachSession(fn func(interface{}, func(string))) {
	for _, ch := range p.players {
		fn(ch, func(s string) { p.lines[ch.Name] += s })
	}
}

type milestoneWriter struct {
	bytes.Buffer
	payload string
	at      func()
	called  bool
}

func (w *milestoneWriter) Write(b []byte) (int, error) {
	if strings.Contains(string(b), w.payload) {
		w.called = true
		w.at()
	}
	return w.Buffer.Write(b)
}

// Registered spec handlers and the actual world/player boundaries. The provider
// isolates the producer's type/level contract; live delivery is separately
// proved by the session/oracle vehicles, not by this adapter.
func testMilestoneMudlog(t *testing.T, kind string, invis ...int) {
	t.Helper()
	w, a, roster, last := newAssassinTestWorld(t)
	w.StopAITicker()
	a.SetInvisLevel(40)
	minimum, typ := LVL_IMMORT, MudlogBrief
	if kind == "advance" {
		level := 34
		if len(invis) > 0 {
			level = invis[0]
		}
		a.SetInvisLevel(level)
		minimum = max(LVL_IMMORT, level)
	}
	if kind == "stable" {
		minimum = LVL_GRGOD
	}
	if kind == "loot" {
		typ = MudlogComplete
	}
	watch := NewPlayer(101, "Milwatch", 1001)
	watch.SetLevel(minimum)
	below := NewPlayer(102, "Milbelow", 1001)
	below.SetLevel(minimum - 1)
	lower := NewPlayer(103, "Millower", 1001)
	lower.SetLevel(40)
	for _, p := range []*Player{watch, below} {
		p.SetPlrFlag(PrfLog1, true)
		p.SetPlrFlag(PrfLog2, true)
	}
	lower.SetPlrFlag(PrfLog1, (typ-1)&1 != 0)
	lower.SetPlrFlag(PrfLog2, (typ-1)&2 != 0)
	provider := &milestoneProvider{players: []*Player{watch, below, lower}, lines: map[string]string{}}
	old := getImmortalSessionProvider()
	SetImmortalSessionProvider(provider)
	t.Cleanup(func() { SetImmortalSessionProvider(old) })
	oldWriter := getLogWriter()
	t.Cleanup(func() { SetLogWriter(oldWriter) })
	payload := ""
	var invoke, at func()
	switch kind {
	case "advance":
		a.SetLevel(2)
		a.SetClass(ClassWarrior)
		saved := false
		w.PlayerSaver = func(p *Player, why string, room int) SaveResult {
			if p == a && why == "advance level" && room == LoadRoomNowhere {
				saved = true
			}
			return SaveSucceeded
		}
		payload = "AssassinActor advanced to level 2"
		at = func() {
			if !saved || a.GetMaxHP() <= 10 {
				t.Error("level log must follow gains and save")
			}
		}
		invoke = func() { a.AdvanceLevel() }
	case "stable":
		a.MountVNum = 99999
		a.MountCostDay = 5
		a.MountRentTime = time.Now().Unix()
		a.SetGold(100)
		payload = "Mount not loaded in stable"
		at = func() {
			if !strings.Contains(last(), "Sorry, we are unable to gather your mount") || a.MountVNum != 99999 || a.GetGold() != 100 {
				t.Error("stable log must follow refusal and retain mount/gold")
			}
		}
		invoke = func() { SpecRegistry["stableboy"](w, a, roster, "collect", "") }
	case "remort":
		a.SetLevel(30)
		a.SetGold(60000)
		a.SetClass(ClassWarrior)
		a.Race = RaceHuman
		saved := false
		w.PlayerSaver = func(*Player, string, int) SaveResult { saved = true; return SaveSucceeded }
		payload = "Due to remorting:"
		at = func() {
			if saved || a.GetLevel() != 1 || a.GetGold() != 0 || a.GetClass() != ClassPaladin || a.GetPractices() != 10 || a.GetMaxHP() > 40 {
				t.Error("remort producer must precede AdvanceLevel and follow remort mutation")
			}
		}
		invoke = func() { SpecRegistry["remorter"](w, a, roster, "remort", "") }
	case "assassinpc":
		pc := NewPlayer(104, "Rosterpc", 1002)
		if err := w.AddPlayer(pc); err != nil {
			t.Fatal(err)
		}
		payload = "Rosterpc is in the assassin store room."
		at = func() {
			out := last()
			if !strings.Contains(out, "GET THE HELL OUT OF THAT ROOM, NOW !!!") || strings.Contains(out, "You can't hire players") {
				t.Error("roster log must be between victim warning and caller refusal")
			}
		}
		invoke = func() { SpecRegistry["assassin"](w, a, nil, "hire", "Rosterpc") }
	case "assassinhire":
		victim := NewPlayer(104, "VictimTarget", 1001)
		victim.SetLevel(5)
		if err := w.AddPlayer(victim); err != nil {
			t.Fatal(err)
		}
		a.SetGold(1000)
		payload = "AssassinActor hires a street urchin to kill VictimTarget.\r\n"
		at = func() {
			out := last()
			if !strings.Contains(out, "security, you know.") || !strings.Contains(out, "hires a street urchin for a job.") || a.GetGold() != 0 {
				t.Error("hire log must follow charge, acknowledgement and room act")
			}
			found := false
			for _, mob := range w.GetMobsInRoom(1001) {
				if mob != roster && mob.GetHunting() == victim.GetName() {
					found = true
				}
			}
			if !found {
				t.Error("hired body not established before log")
			}
		}
		invoke = func() { SpecRegistry["assassin"](w, a, nil, "hire", "street VictimTarget") }
	case "medusa":
		a.SetLevel(1)
		a.SetClass(ClassWarrior)
		a.SetExp(100)
		mob := medusaTestMob(t, w, a)
		payload = "AssassinActor killed by Medusa special at Assassin Room"
		at = func() {
			if !strings.Contains(last(), "your body slowly turns to stone!") || a.Deaths != 0 || a.GetExp() != 100 || a.GetPosition() == combat.PosDead {
				t.Error("Medusa log must follow act and precede death/XP/raw kill")
			}
		}
		invoke = func() { dprng.ResetStream(medusaSeed(t, false)); SpecRegistry["medusa"](w, a, mob, "look", "medusa") }
	case "loot":
		roster, err := w.SpawnMobQuiet(8070, 1001)
		if err != nil {
			t.Fatal(err)
		}
		w.objs[200] = &parser.Obj{VNum: 200, Keywords: "ward armor", ShortDesc: "a ward", TypeFlag: ITEM_ARMOR, Cost: 200, WearFlags: [4]int{1 | 1<<3}}
		obj, err := w.SpawnObject(200, -1)
		if err != nil {
			t.Fatal(err)
		}
		corpse := w.makeCorpse(a.GetName(), a.GetSex(), []*ObjectInstance{obj}, nil, 1001, combat.TYPE_UNDEFINED, 0, false)
		if err := w.MoveObjectToRoomFront(corpse, 1001); err != nil {
			t.Fatal(err)
		}
		payload = "(LOOT) a street urchin attitude looted AssassinActor."
		at = func() {
			if len(corpse.Contains) != 0 || len(roster.Equipment) != 1 {
				t.Errorf("loot log must follow get/junk/wear passes: corpse=%d equipment=%v inventory=%v cost=%d location=%v", len(corpse.Contains), roster.Equipment, roster.Inventory, obj.GetCost(), obj.Location)
			}
		}
		invoke = func() { w.attitudeLootMob(roster, a) }
	default:
		t.Fatal("unknown case")
	}
	last()
	file := &milestoneWriter{payload: payload, at: at}
	SetLogWriter(file)
	invoke()
	if !file.called {
		t.Fatalf("missing %s producer", kind)
	}
	want := "[ " + payload + " ]\r\n"
	if !strings.Contains(provider.lines[watch.Name], want) {
		t.Fatalf("observer bytes=%q want %q", provider.lines[watch.Name], want)
	}
	if strings.Contains(provider.lines[below.Name], payload) || strings.Contains(provider.lines[lower.Name], payload) {
		t.Fatal("level/type filter leak")
	}
	if !strings.Contains(file.String(), payload) {
		t.Fatal("missing file payload")
	}
	if kind == "loot" {
		file.Reset()
		provider.lines = map[string]string{}
		w.attitudeLootMob(roster, roster)
		if strings.Contains(file.String(), "(LOOT)") || strings.Contains(provider.lines[watch.Name], "(LOOT)") {
			t.Fatal("NPC-victim loot must not log")
		}
	}
}
