package game

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/scripting"
)

// Real Lua engine -> bridge -> world adapter -> numbered skill/affect_total.
// C authority: src/scripts.c:1365-1383; src/utils.h:344;
// src/spells.h:179,181; src/handler.c:350-373.
func luaSetSkillFixture(t *testing.T) (*World, *Player, *MobInstance, *scripting.Engine, *scripting.ScriptContext, string) {
	t.Helper()
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001, Name: "Skill room"}}, Mobs: []parser.Mob{{VNum: 3000, Keywords: "probe", ShortDesc: "a probe", Str: 26, Dex: 26, Int: 26, Wis: 26, Con: 26, Cha: 30}}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	p := NewPlayer(1, "Skillprobe", 1001)
	p.Level = LVL_IMPL
	p.Stats = CharStats{Str: 25, StrAdd: 37, Dex: 25, Int: 25, Wis: 25, Con: 25, Cha: 30}
	p.CopyBaseAttributes()
	p.SetSkill("kick", 11)
	p.SetSkill("bash", 23)
	if err := w.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	m, err := w.SpawnMob(3000, 1001)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "globals.lua"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	a := NewWorldScriptableAdapter(w)
	e := scripting.NewEngine(dir, a)
	t.Cleanup(e.Close)
	ch := scripting.CharRef{ID: p.ID}
	me := scripting.CharRef{ID: m.ID, NPC: true}
	ctx := &scripting.ScriptContext{World: a, ChRef: &ch, MeRef: &me, RoomVNum: 1001}
	return w, p, m, e, ctx, dir
}

func runSetSkillScript(t *testing.T, e *scripting.Engine, ctx *scripting.ScriptContext, dir, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "skill.lua"), []byte("function oncmd()\n"+body+"\nreturn TRUE\nend"), 0o600); err != nil {
		t.Fatal(err)
	}
	handled, err := e.RunScript(ctx, "skill.lua", "oncmd")
	if err != nil || !handled {
		t.Fatalf("Lua contract assertion: handled=%t err=%v", handled, err)
	}
}

func TestLuaSetSkillDepthValid(t *testing.T) {
	for _, tc := range []struct {
		name, call string
		kick, bash int
	}{
		{"zero", "set_skill(ch,134,0)", 0, 23},
		{"one", "set_skill(ch,134,1)", 1, 23},
		{"kick", "set_skill(ch,134,37)", 37, 23},
		{"bash", "set_skill(ch,132,83)", 11, 83},
		{"hundred", "set_skill(ch,134,100)", 100, 23},
		{"numeric_strings", "set_skill(ch,\"134\",\"37\")", 37, 23},
		{"fractional", "set_skill(ch,134.75,37.75)", 37, 23},
		{"extra_argument", "set_skill(ch,134,37,\"ignored\")", 37, 23},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, p, _, e, ctx, dir := luaSetSkillFixture(t)
			runSetSkillScript(t, e, ctx, dir, "local r = "+tc.call+"\nassert(type(r) == 'userdata', 'valid return type')\nassert(r == ch.struct, 'valid return identity')")
			if got := p.GetSkill("kick"); got != tc.kick {
				t.Errorf("kick=%d want %d", got, tc.kick)
			}
			if got := p.GetSkill("bash"); got != tc.bash {
				t.Errorf("bash=%d want %d", got, tc.bash)
			}
			if got := attrSnapshot(p); got != (CharStats{Str: 18, StrAdd: 100, Dex: 18, Int: 18, Wis: 18, Con: 18, Cha: 30}) {
				t.Errorf("effective=%+v", got)
			}
			if p.Stats.Str != 25 || p.Stats.Dex != 25 || p.Stats.StrAdd != 37 {
				t.Fatal("base attributes changed")
			}
		})
	}
}

func TestLuaSetSkillDepthInvalid(t *testing.T) {
	for _, tc := range []struct{ name, call, want string }{
		{"character", "set_skill('not a table',134,37)", "r == 37"},
		{"skill", "set_skill(ch,'not a number',37)", "r == 37"},
		{"level", "set_skill(ch,134,'not a number')", "r == 'not a number'"},
		{"nil_character", "set_skill(nil,134,37)", "r == 37"},
		{"nil_skill", "set_skill(ch,nil,37)", "r == 37"},
		{"nil_level", "set_skill(ch,134,nil)", "r == nil"},
		{"extra_top", "set_skill(ch,'bad',37,'top')", "r == 'top'"},
		{"table_top", "set_skill(ch,134,ch)", "r == ch"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, p, m, e, ctx, dir := luaSetSkillFixture(t)
			runSetSkillScript(t, e, ctx, dir, "local r = "+tc.call+"\nassert("+tc.want+", 'invalid stack top')")
			if p.GetSkill("kick") != 11 || p.GetSkill("bash") != 23 {
				t.Fatal("invalid call wrote skills")
			}
			if p.GetDex() != 25 || p.GetStr() != 25 || m.GetDex() != 26 {
				t.Fatal("invalid call ran affect_total")
			}
		})
	}
}

func TestLuaSetSkillDepthModifiers(t *testing.T) {
	_, p, m, e, ctx, dir := luaSetSkillFixture(t)
	// The direct tattoo boundary leaves DEX unbounded until set_skill totals.
	p.Stats.Dex = 18
	p.CopyBaseAttributes()
	p.Tattoo = TattooSpider
	TattooAf(p, true)
	if p.GetDex() != 21 {
		t.Fatal("tattoo fixture didn't preserve unbounded direct addition")
	}
	runSetSkillScript(t, e, ctx, dir, "set_skill(ch,134,37)")
	if p.GetDex() != 18 || p.Stats.Dex != 18 {
		t.Fatal("set_skill didn't rebuild current tattoo")
	}
	// Explicit unbounded copy models restore while modifiers remain present.
	obj := NewObjectInstance(&parser.Obj{VNum: 14425, Keywords: "vest", ShortDesc: "a spider silk vest", WearFlags: [4]int{9}, Affects: []parser.ObjAffect{{Location: ApplyDex, Modifier: 2}}}, 1)
	if err := p.Equipment.SetSlot(SlotBody, obj); err != nil {
		t.Fatal(err)
	}
	p.AddAffect(engine.NewAffectDirect(1, ApplyWis, 1, -30, 0, "probe"))
	p.CopyBaseAttributes()
	runSetSkillScript(t, e, ctx, dir, "set_skill(ch,132,83)")
	if p.GetDex() != 18 || p.GetWis() != 0 {
		t.Fatalf("equipment/affect total dex=%d wis=%d", p.GetDex(), p.GetWis())
	}
	// NPCs use 25, not the PC's 18, and return their own handle.
	runSetSkillScript(t, e, ctx, dir, "local r=set_skill(me,134,37)\nassert(type(r)=='userdata' and r==me.struct, 'NPC return')")
	if m.GetDex() != 25 || m.GetStr() != 25 || m.GetCha() != 30 || m.Dex != 26 {
		t.Fatalf("NPC total dex=%d str=%d cha=%d", m.GetDex(), m.GetStr(), m.GetCha())
	}
	if p.GetSkill("kick") != 37 || p.GetSkill("bash") != 83 {
		t.Fatal("NPC call altered PC skills")
	}
}
