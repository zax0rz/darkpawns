package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// captureOutput records every byte the world sends, keyed by player name.
func captureOutput(w *World) map[string]*strings.Builder {
	out := map[string]*strings.Builder{}
	w.MessageSink = func(name string, msg []byte) {
		if out[name] == nil {
			out[name] = &strings.Builder{}
		}
		out[name].Write(msg)
	}
	return out
}

func outputOf(out map[string]*strings.Builder, name string) string {
	if b := out[name]; b != nil {
		return b.String()
	}
	return ""
}

// TestDamageRefusedProtections pins the refusal bytes of damage()'s
// protection block (fight.c:1318-1368), the gate every damage seam shares
// (DP-1327).
func TestDamageRefusedProtections(t *testing.T) {
	cases := []struct {
		name        string
		attackerLvl int
		victimLvl   int
		outlaw      bool
		peaceful    bool
		wantRefused bool
		wantOutput  string
	}{
		{
			name: "newbie attacker", attackerLvl: 10, victimLvl: 30, wantRefused: true,
			wantOutput: "You are not experienced enough to attack Victim!",
		},
		{
			name: "newbie victim", attackerLvl: 30, victimLvl: 10, wantRefused: true,
			wantOutput: "Ancient forces protect Victim from your wrath!",
		},
		{name: "outlaw newbie victim", attackerLvl: 30, victimLvl: 10, outlaw: true},
		{
			name: "peaceful room", attackerLvl: 30, victimLvl: 30, peaceful: true, wantRefused: true,
			wantOutput: "This room just has such a peaceful, easy feeling...\r\n",
		},
		{name: "peaceful room outlaw", attackerLvl: 30, victimLvl: 30, outlaw: true, peaceful: true},
		{name: "open pk", attackerLvl: 30, victimLvl: 30},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, ch := newCombatTestWorld(t)
			ch.Level = tc.attackerLvl
			victim := NewPlayer(2, "Victim", ch.GetRoom())
			victim.Level = tc.victimLvl
			if tc.outlaw {
				victim.SetPlrFlag(PlrOutlaw, true)
			}
			if err := w.AddPlayer(victim); err != nil {
				t.Fatalf("AddPlayer: %v", err)
			}
			if tc.peaceful {
				w.GetRoomInWorld(ch.GetRoom()).Flags = []string{"peaceful"}
			}
			out := captureOutput(w)

			if got := w.DamageRefused(ch, victim); got != tc.wantRefused {
				t.Fatalf("DamageRefused = %v, want %v", got, tc.wantRefused)
			}
			got := outputOf(out, ch.Name)
			if tc.wantOutput == "" && got != "" {
				t.Fatalf("attacker output = %q, want none", got)
			}
			if !strings.Contains(got, tc.wantOutput) {
				t.Fatalf("attacker output = %q, want %q", got, tc.wantOutput)
			}
			if v := outputOf(out, victim.Name); v != "" {
				t.Fatalf("victim output = %q, want none", v)
			}
		})
	}
}

// C exempts a victim already fighting the attacker from peaceful-room
// protection, so a fight that started elsewhere can finish.
func TestDamageRefusedPeacefulRetaliation(t *testing.T) {
	w, ch := newCombatTestWorld(t)
	mob := spawnTargetMob(t, w)
	w.GetRoomInWorld(ch.GetRoom()).Flags = []string{"peaceful"}
	mob.SetFightingBody(ch)
	if w.DamageRefused(ch, mob) {
		t.Fatal("peaceful room refused a victim already fighting the attacker")
	}
}

// An NPC attacker has no descriptor: C's refusal writes nothing.
func TestDamageRefusedNPCAttackerIsSilent(t *testing.T) {
	w, ch := newCombatTestWorld(t)
	mob := spawnTargetMob(t, w)
	w.GetRoomInWorld(ch.GetRoom()).Flags = []string{"peaceful"}
	out := captureOutput(w)
	if !w.DamageRefused(mob, ch) {
		t.Fatal("peaceful room let an NPC hurt a player")
	}
	if got := outputOf(out, ch.Name); got != "" {
		t.Fatalf("victim output = %q, want none", got)
	}
}

// is_shopkeeper covers the guild, guild_guard, butler and clerk specials as
// well as shop_keeper (mobprog.c:473-507).
func TestDamageRefusedShopkeeperSpecials(t *testing.T) {
	for _, spec := range []string{"shop_keeper", "guild", "guild_guard", "butler", "clerk"} {
		t.Run(spec, func(t *testing.T) {
			w, ch := newCombatTestWorld(t)
			mob := spawnTargetMob(t, w)
			prev, had := MobSpecAssign[mob.GetVNum()]
			MobSpecAssign[mob.GetVNum()] = spec
			t.Cleanup(func() {
				if had {
					MobSpecAssign[mob.GetVNum()] = prev
				} else {
					delete(MobSpecAssign, mob.GetVNum())
				}
			})
			out := captureOutput(w)
			if !w.DamageRefused(ch, mob) {
				t.Fatal("shopkeeper was not protected")
			}
			if got := outputOf(out, ch.Name); got != "Ha ha... Don't think so.\r\n" {
				t.Fatalf("attacker output = %q", got)
			}
		})
	}
}

// TestMobRescueVictimParsesLikeCCommandBodies pins mobRescueVictim to C's
// parsing: the argument goes through one_argument (fill words skipped —
// src/interpreter.c:1267) and the result through get_char_room_vis
// (src/handler.c:1277-1300). "the guard" therefore resolves "guard", while
// an article-leading short description ("a guard trainee") keeps looking for
// the keyword "a". Red on the pre-fix first-word split, which resolved "the"
// and "a" alike and scanned players before mobs by ID.
func TestMobRescueVictimParsesLikeCCommandBodies(t *testing.T) {
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Guard Post", Zone: 1}},
		Mobs: []parser.Mob{
			{VNum: 3001, Keywords: "keeper shopkeeper", ShortDesc: "the shop keeper", Level: 10, Str: 10, Dex: 10},
			{VNum: 3002, Keywords: "guard trainee", ShortDesc: "a guard trainee", Level: 5, Str: 10, Dex: 10},
			{VNum: 3003, Keywords: "guard trainee", ShortDesc: "a guard trainee", Level: 5, Str: 10, Dex: 10},
		},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	keeper, err := w.SpawnMob(3001, 1001)
	if err != nil {
		t.Fatalf("SpawnMob keeper: %v", err)
	}
	if _, err := w.SpawnMob(3002, 1001); err != nil {
		t.Fatalf("SpawnMob guard: %v", err)
	}
	if _, err := w.SpawnMob(3003, 1001); err != nil {
		t.Fatalf("SpawnMob second guard: %v", err)
	}

	if got := mobRescueVictim(w, keeper, "the guard"); got == nil || got == combat.Combatant(keeper) {
		t.Fatalf("the guard = %v, want the fill-word-skipped resolution to a trainee", got)
	}
	if got := mobRescueVictim(w, keeper, "a guard trainee"); got != nil {
		t.Fatalf("article-leading short desc resolved %v; C looks for the keyword \"a\"", got)
	}
	if got := mobRescueVictim(w, keeper, "2.trainee"); got == nil {
		t.Fatal("2.trainee = nil; want the ordinal to count both visible trainees")
	}
	if got := mobRescueVictim(w, keeper, "3.trainee"); got != nil {
		t.Fatalf("3.trainee = %v; two trainees exist, the third ordinal must fail", got)
	}
}
