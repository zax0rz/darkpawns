package command

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/common"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestSetActorHandlerUsesExistingEntryAndKeepsPlayerHandler(t *testing.T) {
	registry := NewRegistry()
	playerCalls := 0
	registry.Register("stand", func(_ common.CommandSession, _ string, _ []string) error {
		playerCalls++
		return nil
	}, "stand", 0, 5, "sta")
	entry, _ := registry.Lookup("stand")
	actorHandler := ActorHandler(func(game.CharacterCommandContext, string, []string) game.CharacterCommandResult {
		return game.CharacterCommandHandled
	})
	if err := entry.Handler(nil, "stand", nil); err != nil {
		t.Fatal(err)
	}
	if err := registry.SetActorHandler("stand", actorHandler); err != nil {
		t.Fatal(err)
	}
	alias, _ := registry.Lookup("sta")
	if alias != entry || alias.ActorHandler == nil || entry.ActorHandler == nil {
		t.Fatal("alias did not retain the same entry and actor callback")
	}
	if err := entry.Handler(nil, "stand", nil); err != nil {
		t.Fatal(err)
	}
	if playerCalls != 2 {
		t.Fatalf("player handler calls = %d, want 2", playerCalls)
	}
}
