package game

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/scripting"
)

func callbackBodyFixture(t *testing.T) (*World, *MobInstance, *MobInstance) {
	t.Helper()
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001}}, Mobs: []parser.Mob{
		{VNum: 300, Keywords: "guard", ShortDesc: "Guard", Level: 30, Race: 1, Sex: 1, ScriptName: "first.lua", LuaFunctions: 32 | 256, ActionFlags: []string{"MEMORY"}},
		{VNum: 301, Keywords: "guard", ShortDesc: "Guard", Level: 20, Race: 2, Sex: 2, ScriptName: "second.lua", LuaFunctions: 32 | 256},
	}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	first, err := w.SpawnMobQuiet(300, 1001)
	if err != nil {
		t.Fatal(err)
	}
	second, err := w.SpawnMobQuiet(301, 1001)
	if err != nil {
		t.Fatal(err)
	}
	first.SetHealth(111)
	second.SetHealth(222)
	first.SetAlignment(-777)
	second.SetAlignment(555)
	first.RaceHates = [5]int{1, 2, 3, 4, 5}
	second.RaceHates = [5]int{5, 4, 3, 2, 1}
	return w, first, second
}

func TestCombatBodyCallbackIdentity(t *testing.T) {
	w, first, second := callbackBodyFixture(t)
	cb := w.WireCombatCallbacks()
	cases := []struct {
		name     string
		read     func(combat.Combatant) int
		one, two int
	}{
		{"race", cb.GetRace, 1, 2},
		{"alignment", cb.GetAlignment, -777, 555},
		{"sex", cb.GetSex, 0, 1},
		{"hp", cb.GetHP, 111, 222},
		{"level", cb.GetLevel, 30, 20},
		{"race-hate", func(b combat.Combatant) int { return cb.GetRaceHate(b, 0) }, 1, 5},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for i := 0; i < 40; i++ {
				if cbval := tc.read(first); cbval != tc.one {
					t.Fatalf("first=%d want %d", cbval, tc.one)
				}
				if cbval := tc.read(second); cbval != tc.two {
					t.Fatalf("second=%d want %d", cbval, tc.two)
				}
			}
		})
	}
	if !cb.IsNPC(first) || !cb.IsNPC(second) {
		t.Fatal("mobile identity lost")
	}
	if cb.GetRaceHate(first, -1) != -1 || cb.GetRaceHate(second, 5) != -1 {
		t.Fatal("race-hate invalid-index defaults changed")
	}
}

func TestCombatBodyCallbackAffectsFlags(t *testing.T) {
	w, first, second := callbackBodyFixture(t)
	cb := w.WireCombatCallbacks()
	first.SetAffected(affSanctuary)
	first.SetAffected(affGroup)
	for i := 0; i < 40; i++ {
		if !cb.HasAffect(first, affSanctuary) || cb.HasAffect(second, affSanctuary) {
			t.Fatal("numeric affect selected another duplicate")
		}
		if !cb.HasAffectStr(first, "AFF_GROUP") || cb.HasAffectStr(second, "AFF_GROUP") {
			t.Fatal("string affect selected another duplicate")
		}
		if !cb.HasMobFlag(first, "MOB_MEMORY") || cb.HasMobFlag(second, "MOB_MEMORY") {
			t.Fatal("mobile flag selected another duplicate")
		}
		if !cb.HasMobVNum(first, 300) || cb.HasMobVNum(second, 300) {
			t.Fatal("prototype selected another duplicate")
		}
	}
	cb.RemoveAffect(second, affSanctuary)
	if !first.IsAffected(affSanctuary) {
		t.Fatal("clearing a duplicate cleared the other body")
	}
	cb.RemoveAffect(first, affSanctuary)
	if first.IsAffected(affSanctuary) {
		t.Fatal("actual body affect not cleared")
	}
	if cb.HasScriptFlag(first, "MS_UNKNOWN") {
		t.Fatal("unrecognized script-flag default changed")
	}
	if !cb.HasScriptFlag(first, "MS_FIGHT") || !cb.HasScriptFlag(second, "MS_FIGHT") {
		t.Fatal("script flags lost")
	}
}

func TestCombatBodyProtectionIsNotNameSelf(t *testing.T) {
	w, first, second := callbackBodyFixture(t)
	room, _ := w.GetRoom(1001)
	room.Flags = append(room.Flags, "PEACEFUL")
	if !w.DamageRefused(first, second) {
		t.Fatal("different same-description bodies were treated as self in a peaceful room")
	}
	if w.DamageRefused(first, first) {
		t.Fatal("true self-damage was refused")
	}
}

func TestCombatBodyCallbackPlayerState(t *testing.T) {
	w, first, _ := callbackBodyFixture(t)
	retired := NewPlayer(40, "Retained", 1001)
	current := NewPlayer(41, "Retained", 1001)
	if err := w.AddPlayer(retired); err != nil {
		t.Fatal(err)
	}
	w.RemovePlayer(retired.GetName())
	if err := w.AddPlayer(current); err != nil {
		t.Fatal(err)
	}
	retired.SetSkill(SkillParry, 93)
	current.SetSkill(SkillParry, 7)
	retired.SetCondition(CondDrunk, 13)
	current.SetCondition(CondDrunk, 1)
	retired.SetGold(81)
	current.SetGold(2)
	retired.Kills = 3
	current.Kills = 50
	retired.Deaths = 4
	current.Deaths = 60
	retired.PKs = 5
	current.PKs = 70
	cb := w.WireCombatCallbacks()
	if cb.GetSkill(retired, combat.SKILL_PARRY) != 93 || cb.GetDrunk(retired) != 13 || cb.GetGold(retired) != 81 {
		t.Fatal("condition/skill/owner selected the replacement by name")
	}
	if cb.GetKills(retired) != 3 || cb.GetDeaths(retired) != 4 || cb.GetPks(retired) != 5 {
		t.Fatal("counter selected replacement")
	}
	cb.SetKills(retired, 9)
	cb.SetDeaths(retired, 10)
	cb.SetPks(retired, 11)
	cb.SetGold(retired, 12)
	cb.SetLastDeath(retired, 123)
	cb.SetAlignment(retired, -600)
	cb.SetPlrFlag(retired)
	if retired.Kills != 9 || retired.Deaths != 10 || retired.PKs != 11 || retired.GetGold() != 12 || retired.GetLastDeath() != 123 || retired.GetAlignment() != -600 || !cb.HasPlrFlag(retired, "outlaw") {
		t.Fatal("mutation did not reach retained body")
	}
	if current.Kills != 50 || current.Deaths != 60 || current.PKs != 70 || current.GetGold() != 2 || current.GetLastDeath() != 0 || cb.HasPlrFlag(current, "outlaw") {
		t.Fatal("mutation touched replacement")
	}
	// Explicit defaults for previously player-only and unwired hooks.
	cb.SetAlignment(first, 0)
	if first.GetAlignment() != -777 || cb.GetDrunk(first) != 0 || cb.GetSkill(first, combat.SKILL_PARRY) != 0 || cb.GetGold(first) != 0 || cb.GetWeaponDescription(first) != "" {
		t.Fatal("player-only mobile defaults changed")
	}
	if cb.Broadcast != nil || cb.SendToChar != nil || cb.SendText != nil || cb.SendRaw != nil || cb.BroadChat != nil || cb.DoFlee != nil || cb.DoRetreat != nil {
		t.Fatal("world wiring invented session hooks")
	}
}

type bodyScriptRecorder struct {
	bodies []combat.Combatant
	actors []*Player
}

func (*bodyScriptRecorder) ForgetFailures() {}
func (r *bodyScriptRecorder) RunScript(ctx *ScriptContext, _ string, _ string) (bool, error) {
	r.bodies = append(r.bodies, ctx.Me.(combat.Combatant))
	p, _ := ctx.Ch.(*Player)
	r.actors = append(r.actors, p)
	return true, nil
}

func TestCombatBodyScriptDispatch(t *testing.T) {
	w, first, second := callbackBodyFixture(t)
	one := NewPlayer(1, "One", 1001)
	two := NewPlayer(2, "Two", 1001)
	if err := w.AddPlayer(one); err != nil {
		t.Fatal(err)
	}
	if err := w.AddPlayer(two); err != nil {
		t.Fatal(err)
	}
	old := ScriptEngine
	r := &bodyScriptRecorder{}
	ScriptEngine = r
	t.Cleanup(func() { ScriptEngine = old })
	for i := 0; i < 40; i++ {
		w.FireMobFightScript(first, one, 1001)
		w.FireMobDeathScript(second, two, 1001)
	}
	for i, b := range r.bodies {
		want := combat.Combatant(first)
		actor := one
		if i%2 == 1 {
			want = second
			actor = two
		}
		if b != want || r.actors[i] != actor {
			t.Fatalf("script %d body=%p actor=%p, want %p/%p", i, b, r.actors[i], want, actor)
		}
	}
}

func TestCombatBodyFollowingAndRoomOrder(t *testing.T) {
	w, first, second := callbackBodyFixture(t)
	p := NewPlayer(3, "Follower", 1001)
	if err := w.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	p.SetFollowingBody(second)
	cb := w.WireCombatCallbacks()
	if cb.GetFollowing(p) != second {
		t.Fatal("mobile master rediscovered by description")
	}
	p.SetFollowing("")
	if cb.GetFollowing(p) != nil {
		t.Fatal("detach retained master")
	}
	cb.StopFollowerOfMaster(first, p)
	chars := cb.GetRoomCombatants(1001)
	if len(chars) != 3 || chars[0] != p || chars[1] != second || chars[2] != first {
		t.Fatalf("room prepend order=%v", chars)
	}
}

func TestCombatBodyLuaTarget(t *testing.T) {
	w, first, second := callbackBodyFixture(t)
	one, two := NewPlayer(1, "One", 100), NewPlayer(2, "Two", 100)
	for _, p := range []*Player{one, two} {
		if err := w.AddPlayer(p); err != nil {
			t.Fatal(err)
		}
	}
	first.SetFightingBody(one)
	second.SetFightingBody(two)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "body.lua"), []byte(`function fight() local target=isfighting(me); assert(target ~= nil and target.name == argument, "wrong fighting body"); return true end`), 0o600); err != nil {
		t.Fatal(err)
	}
	adapter := NewWorldScriptableAdapter(w)
	e := scripting.NewEngine(dir, adapter)
	defer e.Close()
	for _, pair := range []struct {
		body *MobInstance
		name string
	}{{first, "One"}, {second, "Two"}} {
		_, err := e.RunScript(&scripting.ScriptContext{MeRef: charRefFor(pair.body), RoomVNum: 100, Argument: pair.name}, "body.lua", "fight")
		if err != nil {
			t.Fatal(err)
		}
	}
	one.SetFollowingBody(second)
	master := adapter.masterRef(one, one.GetFollowing())
	if master == nil || !master.NPC || master.ID != second.GetID() {
		t.Fatalf("master=%+v", master)
	}
}

func TestCombatBodyLookTarget(t *testing.T) {
	_, first, _ := callbackBodyFixture(t)
	viewer := NewPlayer(1, first.GetName(), 100)
	if got := fightingPresence(first, viewer); got == " is here, fighting YOU!" {
		t.Fatal("same-name body became YOU")
	}
	if got := fightingPresence(viewer, viewer); got != " is here, fighting YOU!" {
		t.Fatal(got)
	}
	first.SetAffected(affInvisible)
	if got := fightingPresence(first, viewer); got != " is here, fighting someone!" {
		t.Fatal(got)
	}
}

func TestCombatBodyGroupRecipients(t *testing.T) {
	w, _, victim := callbackBodyFixture(t)
	leader, member := NewPlayer(1, "Leader", 1001), NewPlayer(2, "Member", 1001)
	for _, p := range []*Player{leader, member} {
		if err := w.AddPlayer(p); err != nil {
			t.Fatal(err)
		}
		p.SetInGroup(true)
		p.SetAffect(affGroup, true)
	}
	leader.Level = 5
	member.Level = 40
	member.SetFollowingBody(leader)
	cb := w.WireCombatCallbacks()
	old := combat.GetCallbacks()
	t.Cleanup(func() { combat.SetCallbacks(old) })
	var recipients []combat.Combatant
	var amounts []int
	cb.GetExp = func(combat.Combatant) int { return 200 }
	cb.GainExp = func(body combat.Combatant, n int) {
		recipients = append(recipients, body)
		amounts = append(amounts, n)
	}
	combat.SetCallbacks(cb)
	combat.GroupGain(leader, victim)
	if len(recipients) != 2 || recipients[0] != leader || recipients[1] != member {
		t.Fatalf("recipients=%v", recipients)
	}
	if amounts[0] == amounts[1] {
		t.Fatalf("member levels lost: shares=%v", amounts)
	}
}

func TestCombatBodyNPCFollowing(t *testing.T) {
	w, first, second := callbackBodyFixture(t)
	w.mobs[302] = &parser.Mob{VNum: 302, Keywords: "follower", ShortDesc: "Follower", Level: 10}
	follower, err := w.SpawnMobQuiet(302, 1001)
	if err != nil {
		t.Fatal(err)
	}
	cb := w.WireCombatCallbacks()
	for _, leader := range []*MobInstance{first, second} {
		follower.SetFollowingBody(leader)
		if got := cb.GetFollowing(follower); got != leader {
			t.Fatalf("NPC master=%p want %p", got, leader)
		}
		ref := NewWorldScriptableAdapter(w).masterRef(follower, follower.GetFollowing())
		if ref == nil || !ref.NPC || ref.ID != leader.GetID() {
			t.Fatalf("master handle=%+v", ref)
		}
		// These group hooks were player-only before the migration and stay so.
		if cb.GetMasterInRoom(follower, 1001) || cb.GetFellowFollowersInRoom(follower, 1001) || cb.CountGroupMembers(follower, 1001) != 0 {
			t.Fatal("expanded unsupported NPC group hooks")
		}
		cb.StopFollowerOfMaster(follower, leader)
		if cb.GetFollowing(follower) != nil || follower.GetFollowing() != "" {
			t.Fatal("NPC master retained after detach")
		}
	}
	// Live name-taking API captures its selected NPC leader once.
	if err := w.SetFollower("Follower", "Guard", true); err != nil {
		t.Fatal(err)
	}
	held := cb.GetFollowing(follower)
	if held != first && held != second {
		t.Fatal("name-taking follow did not retain an actual NPC")
	}
	w.executeMobCommand(302, "follow Guard")
	if got := cb.GetFollowing(follower); got != first && got != second {
		t.Fatal("mob follow command did not retain actual NPC")
	}
}
