package scripting

import (
	"os"
	"path/filepath"
	"testing"
)

// legacy_char_table_test.go — DP-1433's non-bridge arm. C's char_to_table
// (src/scripts.c:1823) has one shape for both character kinds; the legacy
// builder now emits exactly that shape instead of its own (a partial
// player-only field set, an invented skills table, an empty mobile wear table).
//
// These tests drive the real RunScript path with a ScriptableWorld that is not
// a Bridge, which is the only way to reach the legacy builder.

// legacyTableChar is one character of the legacy builder's fixture: it
// implements ScriptablePlayer, ScriptableMob and ScriptableChar so a single
// value can stand in for either kind.
type legacyTableChar struct {
	player  bool
	name    string
	align   int
	gold    int
	level   int
	hp      int
	maxhp   int
	pos     int
	alias   string
	carry   []ScriptableObject
	worn    []ScriptableObject
	leader  string
	id      int
	exp     int
	bank    int
	mana    int
	move    int
	cha     int
	jail    int
	tattoo  int
	follows int
	vnum    int
	timer   int
}

func (c *legacyTableChar) AsPlayer() ScriptablePlayer {
	if c.player {
		return c
	}
	return nil
}

func (c *legacyTableChar) AsMob() ScriptableMob {
	if c.player {
		return nil
	}
	return c
}

func (c *legacyTableChar) GetID() int                            { return c.id }
func (c *legacyTableChar) GetName() string                       { return c.name }
func (c *legacyTableChar) GetLevel() int                         { return c.level }
func (c *legacyTableChar) GetHealth() int                        { return c.hp }
func (c *legacyTableChar) SetHealth(v int)                       { c.hp = v }
func (c *legacyTableChar) GetMaxHealth() int                     { return c.maxhp }
func (c *legacyTableChar) GetGold() int                          { return c.gold }
func (c *legacyTableChar) SetGold(v int)                         { c.gold = v }
func (c *legacyTableChar) GetRace() int                          { return 1 }
func (c *legacyTableChar) GetClass() int                         { return 0 }
func (c *legacyTableChar) GetAlignment() int                     { return c.align }
func (c *legacyTableChar) GetRoomVNum() int                      { return 1001 }
func (c *legacyTableChar) SendMessage(string)                    {}
func (c *legacyTableChar) GetFlags() uint64                      { return 0 }
func (c *legacyTableChar) GetInventoryItems() []ScriptableObject { return c.carry }
func (c *legacyTableChar) GetCarriedObjects() []ScriptableObject { return c.carry }
func (c *legacyTableChar) GetWornObjects() []ScriptableObject    { return c.worn }
func (c *legacyTableChar) GetPosition() int                      { return c.pos }
func (c *legacyTableChar) GetAlias() string                      { return c.alias }
func (c *legacyTableChar) GetLeaderName() string                 { return c.leader }
func (c *legacyTableChar) GetFollowerCount() int                 { return c.follows }
func (c *legacyTableChar) GetExp() int                           { return c.exp }
func (c *legacyTableChar) GetBankGold() int                      { return c.bank }
func (c *legacyTableChar) GetMana() int                          { return c.mana }
func (c *legacyTableChar) GetMove() int                          { return c.move }
func (c *legacyTableChar) GetCha() int                           { return c.cha }
func (c *legacyTableChar) GetJailTimer() int                     { return c.jail }
func (c *legacyTableChar) GetTattoo() int                        { return c.tattoo }
func (c *legacyTableChar) GetVNum() int                          { return c.vnum }
func (c *legacyTableChar) GetWaitState() int                     { return c.timer }
func (c *legacyTableChar) GetFighting() string                   { return "" }
func (c *legacyTableChar) SetHunting(string)                     {}
func (c *legacyTableChar) IsHunting() bool                       { return false }

// runLegacyTableScript runs the fixture script with ch = pc and me = mob
// through the non-bridge path and returns the string it stored in
// "table_seen".
func runLegacyTableScript(t *testing.T, pc, mob, leader *legacyTableChar) string {
	t.Helper()
	return runLegacyScript(t, pc, mob, leader, `function oncmd()
  table_seen = "pc|" .. ch.name .. "|" .. ch.align .. "|" .. ch.pos .. "|" .. ch.alias ..
    "|" .. ch.evil .. "|" .. ch.leader.name .. "|" .. ch.id .. "|" .. ch.exp ..
    "|" .. ch.bank .. "|" .. ch.mana .. "|" .. ch.move .. "|" .. ch.cha ..
    "|" .. ch.jail .. "|" .. ch.tattoo .. "|" .. ch.followers ..
    "|" .. ch.objs[1].name .. "|" .. ch.wear[1].name ..
    "|" .. tostring(ch.skills) .. "|" .. tostring(ch.vnum) ..
    "||" .. me.name .. "|" .. me.align .. "|" .. me.pos .. "|" .. me.alias ..
    "|" .. me.evil .. "|" .. me.leader.name .. "|" .. me.vnum .. "|" .. me.timer ..
    "|" .. me.objs[1].name .. "|" .. me.wear[1].name ..
    "|" .. tostring(me.skills) .. "|" .. tostring(me.id)
  return TRUE
end
`)
}

// runLegacyScript runs body with ch = pc and me = mob through the non-bridge
// path and returns the string it stored in "table_seen".
func runLegacyScript(t *testing.T, pc, mob, leader *legacyTableChar, body string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "globals.lua"), nil, 0o600); err != nil {
		t.Fatalf("globals.lua: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "table.lua"), []byte(body), 0o600); err != nil {
		t.Fatalf("table.lua: %v", err)
	}
	world := &mockWorldForTest{}
	if leader != nil {
		world.chars = map[string]ScriptableChar{leader.name: leader}
	}
	engine := NewEngine(dir, world)
	t.Cleanup(engine.Close)
	ctx := &ScriptContext{Ch: pc, Me: mob, RoomVNum: 1001}
	if handled, err := engine.RunScript(ctx, "table.lua", "oncmd"); err != nil || !handled {
		t.Fatalf("RunScript: handled=%t err=%v", handled, err)
	}
	return engine.l.GetGlobal("table_seen").String()
}

// TestLegacyCharTableMatchesCFieldSet pins the legacy builder to C's field set:
// every player-only field, the mobile-only pair, pos/alias/evil, the leader
// chain, carried and worn objects for both kinds, and the absent invented
// skills table (R1/R4/R5e).
func TestLegacyCharTableMatchesCFieldSet(t *testing.T) {
	leader := &legacyTableChar{player: true, name: "Chainleader", pos: 8, alias: "chainleader"}
	pc := &legacyTableChar{
		player: true, name: "Tabletester", align: -400, pos: 8, alias: "tabletester",
		id: 7, exp: 555, bank: 44, mana: 66, move: 77, cha: 15, jail: 3, tattoo: 9,
		follows: 2, leader: "Chainleader",
		carry: []ScriptableObject{&testObj{vnum: 3001, shortDesc: "a pc trinket"}},
		worn:  []ScriptableObject{&testObj{vnum: 3002, shortDesc: "a pc cuirass"}},
	}
	mob := &legacyTableChar{
		name: "the healer", align: -400, pos: 8, alias: "healer probe",
		vnum: 9001, timer: 4, leader: "Chainleader",
		carry: []ScriptableObject{&testObj{vnum: 3003, shortDesc: "an npc bauble"}},
		worn:  []ScriptableObject{&testObj{vnum: 3004, shortDesc: "an npc helm"}},
	}

	want := "pc|Tabletester|-400|8|tabletester|1|Chainleader|7|555|44|66|77|15|3|9|2" +
		"|a pc trinket|a pc cuirass|nil|nil" +
		"||the healer|-400|8|healer probe|1|Chainleader|9001|4" +
		"|an npc bauble|an npc helm|nil|nil"
	if got := runLegacyTableScript(t, pc, mob, leader); got != want {
		t.Fatalf("legacy char tables = %q, want %q", got, want)
	}
}

// TestLegacyCharTableEvilThreshold pins C's IS_EVIL boundary at -350: -349 is
// not evil, -350 is (src/scripts.c:1865-1868). The script reads only the evil
// field, so no other value can satisfy the assertion.
func TestLegacyCharTableEvilThreshold(t *testing.T) {
	const body = `function oncmd()
  table_seen = "evil=" .. ch.evil
  return TRUE
end
`
	pc := &legacyTableChar{player: true, name: "Borderline", align: -349}
	at349 := runLegacyScript(t, pc, &legacyTableChar{name: "x"}, nil, body)
	pc.align = -350
	at350 := runLegacyScript(t, pc, &legacyTableChar{name: "x"}, nil, body)
	if at349 != "evil=0" {
		t.Fatalf("IS_EVIL at -349 = %q, want evil=0", at349)
	}
	if at350 != "evil=1" {
		t.Fatalf("IS_EVIL at -350 = %q, want evil=1", at350)
	}
}

func (c *legacyTableChar) SetFollowing(string) {}

func (c *legacyTableChar) GetPrototype() ScriptableMobPrototype {
	return &testMobPrototype{shortDesc: c.name}
}

var (
	_ ScriptablePlayer = (*legacyTableChar)(nil)
	_ ScriptableMob    = (*legacyTableChar)(nil)
	_ ScriptableChar   = (*legacyTableChar)(nil)
)
