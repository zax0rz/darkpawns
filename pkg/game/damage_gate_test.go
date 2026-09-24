package game

import (
	"strings"
	"testing"
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
	mob.SetFighting(ch.Name)
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
