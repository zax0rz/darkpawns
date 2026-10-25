package game

import (
	"reflect"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/scripting"
)

func TestWorldBridgeMobileActionsDispatchTheCCommandAndArgument(t *testing.T) {
	w, player, mob, _ := newNPCCommandWorld(t)
	adapter := NewWorldScriptableAdapter(w)
	mobRef := scripting.CharRef{NPC: true, ID: mob.GetID()}
	playerRef := scripting.CharRef{ID: player.ID}
	var got []struct {
		name string
		arg  string
	}
	w.CharacterCommandDirectExecFunc = func(actor combat.Combatant, name, arg string) CharacterCommandResult {
		if actor != mob {
			t.Fatalf("direct command actor = %v, want mobile %v", actor, mob)
		}
		got = append(got, struct {
			name string
			arg  string
		}{name, arg})
		return CharacterCommandHandled
	}

	adapter.Say(mobRef, "  hello")
	adapter.Emote(mobRef, " smiles.")
	adapter.Tell(mobRef, "Zach hello ")
	adapter.Gossip(mobRef, "world message")
	adapter.Echo("local", nil, &mobRef, "local message")
	adapter.Echo("global", nil, &mobRef, "global message")
	social := ""
	for name := range Socials {
		if _, ok := socialMinPosition[name]; ok {
			social = name
			break
		}
	}
	if social == "" {
		t.Fatal("no registered social is available for the direct command check")
	}
	if !adapter.Social(mobRef, playerRef, social) {
		t.Fatal("mobile social was rejected")
	}
	adapter.Follow(mobRef, playerRef, true)
	adapter.Mount(mobRef, &playerRef, "ride")
	adapter.Mount(mobRef, nil, "dismount")

	want := []struct {
		name string
		arg  string
	}{
		{"say", "  hello"},
		{"emote", " smiles."},
		{"tell", "Zach hello "},
		{"gossip", "world message"},
		{"echo", "local message"},
		{"gecho", "global message"},
		{social, player.GetName()},
		{"follow", charKeywords(player)},
		{"ride", player.GetName()},
		{"dismount", ""},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("direct mobile calls = %#v, want %#v", got, want)
	}
}
