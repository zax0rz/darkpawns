package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/scripting"
)

func TestGossipObserverMatchesMortalAndLeavesGameBytes(t *testing.T) {
	for _, hidden := range []string{"visible", "wizard", "invisible", "hidden"} {
		t.Run(hidden, func(t *testing.T) {
			w, actor, mortal, _, output := newChannelWorld(t)
			mortal.Level = 1
			switch hidden {
			case "wizard":
				actor.SetInvisLevel(2)
			case "invisible":
				actor.SetAffectBit(affInvisible, true)
			case "hidden":
				actor.SetAffectBit(affHide, true)
			}
			var mirrored []string
			w.SetGossipObserver(func(line string) { mirrored = append(mirrored, line) })
			w.DoChannel(actor, "hello $$ $n", "gossip")
			got := channelOutput(output, mortal.Name)
			if len(mirrored) != 1 || mirrored[0] != got {
				t.Fatalf("mirror %q != mortal %q", mirrored, got)
			}
			if hidden == "wizard" || hidden == "invisible" {
				if got != "Someone gossips, 'hello $ someone'\r\n" {
					t.Fatalf("private identity leaked: %q", got)
				}
			} else if got != "Actor gossips, 'hello $ Actor'\r\n" {
				t.Fatalf("visible line %q", got)
			}
			history := w.ReviewGossip(mortal)
			output[mortal.Name].Reset()
			w.SetGossipObserver(nil)
			w.DoChannel(actor, "hello $$ $n", "gossip")
			if channelOutput(output, mortal.Name) != got {
				t.Fatal("observer changed game bytes")
			}
			if history == "" {
				t.Fatal("history lost")
			}
		})
	}
}

func TestGossipObserverOnlyPublicGossipAndIncludesNPC(t *testing.T) {
	w, actor, _, _, _ := newChannelWorld(t)
	var mirrored []string
	w.SetGossipObserver(func(line string) { mirrored = append(mirrored, line) })
	actor.SetPlrFlag(PlrNoshout, true)
	w.DoChannel(actor, "private refusal", "gossip")
	actor.SetPlrFlag(PlrNoshout, false)
	w.DoChannel(actor, "", "gossip")
	w.DoChannel(actor, "unrelated", "auction")
	actor.SetPlrFlag(PrfNoGossip, true)
	w.DoChannel(actor, "channel off", "gossip")
	actor.SetPlrFlag(PrfNoGossip, false)
	w.MovePlayerToRoom(actor, 1002)
	w.DoChannel(actor, "soundproof", "gossip")
	if len(mirrored) != 0 {
		t.Fatalf("non-public text mirrored: %q", mirrored)
	}
	mob := NewMob(&parser.Mob{VNum: 100, ShortDesc: "a town crier", Level: 10}, 1001)
	w.NPCGossipCommand(mob, "public mob line")
	mob.SetAffected(affInvisible)
	w.NPCGossipCommand(mob, "hidden mob line")
	if len(mirrored) != 2 || mirrored[0] != "A town crier gossips, 'public mob line'\r\n" || mirrored[1] != "Someone gossips, 'hidden mob line'\r\n" {
		t.Fatalf("NPC projection = %q", mirrored)
	}
	// Grapevine can reconnect without clearing the independent ops observer.
	w.SetOnGossip(nil)
	w.NPCGossipCommand(mob, "still public")
	if len(mirrored) != 3 {
		t.Fatal("Grapevine replaced the observer")
	}
}

func TestGossipObserverReceivesLuaBridge(t *testing.T) {
	w, _, mob, _ := newNPCCommandWorld(t)
	var line string
	w.SetGossipObserver(func(value string) { line = value })
	NewWorldScriptableAdapter(w).Gossip(scripting.CharRef{NPC: true, ID: mob.GetID()}, "scripted gossip")
	if line == "" || !strings.Contains(line, "gossips, 'scripted gossip'") {
		t.Fatalf("Lua bridge skipped observer: %q", line)
	}
}
