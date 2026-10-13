package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestConPlayingGossipOnlyReachesPlayingDescriptors(t *testing.T) {
	f := newConPlayingFixture(t)
	if err := ExecuteCommand(f.actor, "gossip", []string{"hello", "world"}); err != nil {
		t.Fatalf("dispatch gossip: %v", err)
	}
	if got, want := drainSessionText(t, f.actor), []string{"You gossip, 'hello world'\r\n"}; !equalStrings(got, want) {
		t.Fatalf("actor output = %q, want %q", got, want)
	}
	if got, want := drainSessionText(t, f.playing), []string{"Actor gossips, 'hello world'\r\n"}; !equalStrings(got, want) {
		t.Errorf("playing recipient output = %q, want %q", got, want)
	}
	for _, s := range []*Session{f.mailing, f.noteWrite} {
		if got := drainSessionText(t, s); len(got) != 0 {
			t.Errorf("%s writer received gossip despite C's PLR_WRITING gate: %q", s.playerName, got)
		}
	}
	for name, s := range map[string]*Session{
		"menu": f.menu, "character creation": f.creating, "OLC editor": f.editing, "TEDIT": f.textEditing, "linkless": f.linkless,
	} {
		if got := drainSessionText(t, s); len(got) != 0 {
			t.Errorf("%s descriptor received gossip: %q", name, got)
		}
	}
}

func TestClanTellOnlyBroadcastsToConPlayingDescriptors(t *testing.T) {
	f := newConPlayingFixture(t)
	f.actor.player.SetLevel(10)
	for _, s := range []*Session{f.actor, f.playing, f.mailing, f.noteWrite, f.menu, f.creating, f.editing, f.textEditing, f.linkless} {
		s.player.ClanID, s.player.ClanRank = 1, 1
	}
	if err := ExecuteCommand(f.actor, "ctell", []string{"proof"}); err != nil {
		t.Fatalf("dispatch ctell: %v", err)
	}
	if got, want := drainSessionText(t, f.actor), []string{"You tell your clan, 'proof'\r\n"}; !equalStrings(got, want) {
		t.Fatalf("actor ctell bytes = %q, want %q", got, want)
	}
	for _, s := range []*Session{f.playing, f.mailing, f.noteWrite} {
		if got, want := drainSessionText(t, s), []string{"Actor tells your clan, 'proof'\r\n"}; !equalStrings(got, want) {
			t.Errorf("%s ctell bytes = %q, want %q", s.playerName, got, want)
		}
	}
	for name, s := range map[string]*Session{
		"menu": f.menu, "character creation": f.creating, "OLC editor": f.editing, "TEDIT": f.textEditing, "linkless": f.linkless,
	} {
		if got := drainSessionText(t, s); len(got) != 0 {
			t.Errorf("%s descriptor received ctell: %q", name, got)
		}
	}
}

func TestQuestBroadcastOnlyVisitsConPlayingDescriptors(t *testing.T) {
	f := newConPlayingFixture(t)
	for _, s := range []*Session{f.actor, f.playing, f.mailing, f.noteWrite, f.menu, f.creating, f.editing, f.textEditing, f.linkless} {
		s.player.SetPlrFlag(game.PrfQuest, true)
	}
	if err := ExecuteCommand(f.actor, "qsay", []string{"proof"}); err != nil {
		t.Fatalf("dispatch qsay: %v", err)
	}
	if got, want := drainSessionText(t, f.actor), []string{"&WYou quest-say, 'proof'&n"}; !equalStrings(got, want) {
		t.Fatalf("actor qsay = %q, want %q", got, want)
	}
	if got, want := drainSessionText(t, f.playing), []string{"&WActor quest-says, 'proof'&n"}; !equalStrings(got, want) {
		t.Errorf("playing qsay recipient = %q, want %q", got, want)
	}
	for _, s := range []*Session{f.mailing, f.noteWrite} {
		if got := drainSessionText(t, s); len(got) != 0 {
			t.Errorf("%s writer received qsay despite C's PLR_WRITING gate: %q", s.playerName, got)
		}
	}
	for name, s := range map[string]*Session{
		"menu": f.menu, "character creation": f.creating, "OLC editor": f.editing, "TEDIT": f.textEditing, "linkless": f.linkless,
	} {
		if got := drainSessionText(t, s); len(got) != 0 {
			t.Errorf("%s descriptor received qsay: %q", name, got)
		}
	}
}

func TestPageAllOnlyPagesConPlayingDescriptors(t *testing.T) {
	f := newConPlayingFixture(t)
	f.actor.player.SetLevel(game.LVL_IMPL)
	if err := ExecuteCommand(f.actor, "page", []string{"all", "proof"}); err != nil {
		t.Fatalf("dispatch page all: %v", err)
	}
	want := []string{"\a\a*Actor* proof\r\n\r\n"}
	for _, s := range []*Session{f.actor, f.playing, f.mailing, f.noteWrite} {
		if got := drainSessionText(t, s); !equalStrings(got, want) {
			t.Errorf("%s page-all bytes = %q, want %q", s.playerName, got, want)
		}
	}
	for name, s := range map[string]*Session{
		"menu": f.menu, "character creation": f.creating, "OLC editor": f.editing, "TEDIT": f.textEditing, "linkless": f.linkless,
	} {
		if got := drainSessionText(t, s); len(got) != 0 {
			t.Errorf("%s descriptor received page-all: %q", name, got)
		}
	}
}
