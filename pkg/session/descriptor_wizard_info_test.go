package session

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestConPlayingGechoDispatchMatchesCStates(t *testing.T) {
	f := newConPlayingFixture(t)
	if err := ExecuteCommand(f.actor, "gecho", []string{"Attention!"}); err != nil {
		t.Fatalf("gecho: %v", err)
	}
	for _, s := range []*Session{f.actor, f.playing, f.mailing, f.noteWrite} {
		if got := drainSessionText(t, s); !equalStrings(got, []string{"Attention!"}) {
			t.Errorf("%s gecho = %q, want exact playing-descriptor bytes", s.playerName, got)
		}
	}
	for name, s := range map[string]*Session{
		"menu": f.menu, "character creation": f.creating, "OLC editor": f.editing, "TEDIT": f.textEditing, "linkless": f.linkless,
	} {
		if got := drainSessionText(t, s); len(got) != 0 {
			t.Errorf("%s descriptor received gecho: %q", name, got)
		}
	}
}

func TestClanWhoOnlyListsConPlayingDescriptors(t *testing.T) {
	f := newConPlayingFixture(t)
	f.manager.world.Clans = game.NewClanManager()
	clan := &game.Clan{Ranks: 1}
	clan.RankName[0] = "Member"
	f.manager.world.Clans.AddClan(clan)
	for _, s := range []*Session{f.actor, f.playing, f.mailing, f.noteWrite, f.menu, f.creating, f.editing, f.textEditing, f.linkless} {
		s.player.ClanID, s.player.ClanRank = clan.ID, 1
	}
	if err := ExecuteCommand(f.actor, "clan", []string{"who"}); err != nil {
		t.Fatalf("dispatch clan who: %v", err)
	}
	got := strings.Join(drainSessionText(t, f.actor), "")
	if !strings.HasPrefix(got, "\r\nClan members online\r\n-------------------------\r\n") {
		t.Fatalf("clan who header bytes = %q", got)
	}
	for _, name := range []string{"Actor", "Playing", "Mailing", "NoteWrite"} {
		if !strings.Contains(got, "Member "+name+"\r\n") {
			t.Errorf("clan who bytes omit exact playing row %q: %q", "Member "+name+"\\r\\n", got)
		}
	}
	for _, name := range []string{"Menu", "Creating", "Editing", "TextEditing", "Linkless"} {
		if strings.Contains(got, "Member "+name+"\r\n") {
			t.Errorf("clan who bytes include excluded row for %s: %q", name, got)
		}
	}
}

func TestForceAllOnlyTargetsConPlayingDescriptors(t *testing.T) {
	f := newConPlayingFixture(t)
	f.actor.player.SetLevel(game.LVL_GRGOD)
	for _, s := range []*Session{f.editing, f.textEditing} {
		s.player.SetLevel(10)
	}
	targets := []*Session{f.playing, f.mailing, f.noteWrite}
	baseline := make(map[*Session]string, len(targets))
	for _, s := range targets {
		if err := ExecuteCommand(s, "score", nil); err != nil {
			t.Fatalf("capture %s score output: %v", s.playerName, err)
		}
		baseline[s] = strings.Join(drainSessionText(t, s), "")
	}
	if err := ExecuteCommand(f.actor, "force", []string{"all", "score"}); err != nil {
		t.Fatalf("dispatch force all: %v", err)
	}
	for _, s := range targets {
		want := baseline[s]
		if s == f.playing {
			want = "Actor has forced you to 'score'.\r\n" + want
		}
		if got := strings.Join(drainSessionText(t, s), ""); got != want {
			t.Errorf("%s force-all bytes = %q, want exact bytes %q", s.playerName, got, want)
		}
	}
	for name, s := range map[string]*Session{
		"menu": f.menu, "character creation": f.creating, "OLC editor": f.editing, "TEDIT": f.textEditing, "linkless": f.linkless,
	} {
		if got := drainSessionText(t, s); len(got) != 0 {
			t.Errorf("%s descriptor received forced command bytes: %q", name, got)
		}
	}
}

func TestTransferAllOnlyMovesConPlayingDescriptors(t *testing.T) {
	f := newConPlayingFixture(t)
	f.actor.player.SetLevel(game.LVL_IMPL)
	for _, s := range []*Session{f.playing, f.mailing, f.noteWrite, f.menu, f.creating, f.editing, f.textEditing, f.linkless} {
		if err := f.manager.world.PlayerTransfer(s.player, 1002); err != nil {
			t.Fatalf("move %s to room 1002: %v", s.playerName, err)
		}
	}
	if err := ExecuteCommand(f.actor, "transfer", []string{"all"}); err != nil {
		t.Fatalf("dispatch transfer all: %v", err)
	}
	for _, s := range []*Session{f.playing, f.mailing, f.noteWrite} {
		if got := s.player.GetRoomVNum(); got != 1001 {
			t.Errorf("%s room = %d, want playing descriptor transferred to 1001", s.playerName, got)
		}
	}
	for name, s := range map[string]*Session{
		"menu": f.menu, "character creation": f.creating, "OLC editor": f.editing, "TEDIT": f.textEditing, "linkless": f.linkless,
	} {
		if got := s.player.GetRoomVNum(); got != 1002 {
			t.Errorf("%s room = %d, want excluded character to remain in 1002", name, got)
		}
	}
	got := drainSessionText(t, f.actor)
	want := []string{"Mailing arrives from a puff of smoke.\r\n", "NoteWrite arrives from a puff of smoke.\r\n", "Playing arrives from a puff of smoke.\r\n", "Okay.\r\n"}
	sort.Strings(got)
	sort.Strings(want)
	if !equalStrings(got, want) {
		t.Errorf("transfer-all actor bytes = %q, want exact output multiset %q", got, want)
	}
}

func TestTagDescriptorSearchAndLinklessGateMatchC(t *testing.T) {
	f := newConPlayingFixture(t)
	f.editing.player.SetPlrFlag(game.PlrIt, true)
	if err := ExecuteCommand(f.actor, "tag", nil); err != nil {
		t.Fatalf("dispatch tag without argument: %v", err)
	}
	if got, want := strings.Join(drainSessionText(t, f.actor), ""), "Tag who?\r\n"; got != want {
		t.Fatalf("empty tag output = %q, want %q", got, want)
	}
	if err := ExecuteCommand(f.actor, "tag", []string{"NoSuchPlayer"}); err != nil {
		t.Fatalf("dispatch tag with missing target: %v", err)
	}
	if got, want := strings.Join(drainSessionText(t, f.actor), ""), "They don't seem to be here.\r\n"; got != want {
		t.Fatalf("missing tag target output = %q, want %q", got, want)
	}

	if err := ExecuteCommand(f.actor, "tag", []string{"Playing"}); err != nil {
		t.Fatalf("dispatch tag to playing target: %v", err)
	}
	if got, want := strings.Join(drainSessionText(t, f.actor), ""), "You tap Playing and scream, 'TAG! You're it!'\r\n"; got != want {
		t.Fatalf("actor tag output = %q, want %q", got, want)
	}
	if got, want := strings.Join(drainSessionText(t, f.playing), ""), "Actor taps you and screams, 'TAG! You're it!'\r\n"; got != want {
		t.Fatalf("tag target output = %q, want %q", got, want)
	}
	roomTag := "Actor taps Playing and screams, 'TAG! You're it!'\r\n"
	for _, observer := range []*Session{f.mailing, f.noteWrite} {
		if got := strings.Join(drainSessionText(t, observer), ""); got != roomTag {
			t.Errorf("%s room tag output = %q, want %q", observer.playerName, got, roomTag)
		}
	}
	if got := strings.Join(drainSessionText(t, f.editing), ""); got != roomTag {
		t.Errorf("OLC room tag output = %q, want %q", got, roomTag)
	}
	for _, observer := range []*Session{f.menu, f.creating, f.textEditing, f.linkless} {
		drainSessionText(t, observer)
	}
	if !f.playing.player.HasPLRFlag(game.PlrIt) || f.actor.player.HasPLRFlag(game.PlrIt) || !f.editing.player.HasPLRFlag(game.PlrIt) {
		t.Fatal("tag should mark the playing target, leave the actor clear, and ignore stale OLC PLR_IT")
	}
	if got, want := f.playing.player.GetWaitState(), 2*engine.PULSE_VIOLENCE; got != want {
		t.Fatalf("tag target wait = %d, want %d pulses", got, want)
	}

	if err := ExecuteCommand(f.actor, "tag", []string{"Playing"}); err != nil {
		t.Fatalf("dispatch tag as non-it actor: %v", err)
	}
	if got, want := strings.Join(drainSessionText(t, f.actor), ""), "But you are not it!\r\n"; got != want {
		t.Fatalf("non-it actor output = %q, want %q", got, want)
	}
	if err := ExecuteCommand(f.playing, "tag", []string{"Playing"}); err != nil {
		t.Fatalf("dispatch tag on current it: %v", err)
	}
	if got, want := strings.Join(drainSessionText(t, f.playing), ""), "You're it already!\r\n"; got != want {
		t.Fatalf("already-it output = %q, want %q", got, want)
	}

	if err := ExecuteCommand(f.playing, "tag", []string{"Editing"}); err != nil {
		t.Fatalf("dispatch tag to OLC descriptor: %v", err)
	}
	if got, want := strings.Join(drainSessionText(t, f.playing), ""), "You tap Editing and scream, 'TAG! You're it!'\r\n"; got != want {
		t.Fatalf("actor tag-to-editor output = %q, want %q", got, want)
	}
	if got, want := strings.Join(drainSessionText(t, f.editing), ""), "Playing taps you and screams, 'TAG! You're it!'\r\n"; got != want {
		t.Fatalf("editor target output = %q, want %q", got, want)
	}
	if got, want := strings.Join(drainSessionText(t, f.actor), ""), "Playing taps Editing and screams, 'TAG! You're it!'\r\n"; got != want {
		t.Fatalf("tag room output = %q, want %q", got, want)
	}
	if f.playing.player.HasPLRFlag(game.PlrIt) || !f.editing.player.HasPLRFlag(game.PlrIt) {
		t.Fatal("tagging from the current it should move PLR_IT to the selected OLC player")
	}
	if got, want := f.editing.player.GetWaitState(), 2*engine.PULSE_VIOLENCE; got != want {
		t.Fatalf("OLC target wait = %d, want %d pulses", got, want)
	}
	for _, s := range []*Session{f.actor, f.playing, f.mailing, f.noteWrite, f.menu, f.creating, f.editing, f.textEditing, f.linkless} {
		drainSessionText(t, s)
	}
	if err := ExecuteCommand(f.actor, "tag", []string{"Linkless"}); err != nil {
		t.Fatalf("dispatch tag to linkless target: %v", err)
	}
	if got, want := strings.Join(drainSessionText(t, f.actor), ""), "No way! They're linkless!\r\n"; got != want {
		t.Fatalf("linkless target output = %q, want %q", got, want)
	}
	if err := ExecuteCommand(f.actor, "tag", []string{"Actor"}); err != nil {
		t.Fatalf("dispatch self-tag: %v", err)
	}
	if got, want := strings.Join(drainSessionText(t, f.actor), ""), "Let the game begin!\r\n"; got != want {
		t.Fatalf("self-tag output = %q, want %q", got, want)
	}
	if !f.actor.player.HasPLRFlag(game.PlrIt) {
		t.Fatal("self-tag should set PLR_IT after the game has no playing target")
	}
	if got := f.actor.player.GetWaitState(); got != 0 {
		t.Fatalf("self-tag wait state = %d, want 0", got)
	}
}

func TestCmdTagNPCAcknowledgment(t *testing.T) {
	m := makeTestManagerWithMobs(t)
	actor := makeCommandTestSession(t, m, "Tagger", game.LVL_GOD, 1001)
	registerInWorld(t, actor)
	target := registerMob(t, m, 2001, 1001)
	actor.player.SetPlrFlag(game.PlrIt, true)

	if err := ExecuteCommand(actor, "tag", []string{"goblin"}); err != nil {
		t.Fatalf("ExecuteCommand tag NPC: %v", err)
	}
	if got, want := strings.Join(drainSessionText(t, actor), ""), "You tag A goblin guard, but It tags you right back!\r\n"; got != want {
		t.Fatalf("NPC tag output = %q, want %q", got, want)
	}
	if !actor.player.HasPLRFlag(game.PlrIt) {
		t.Fatal("tagging an NPC should not clear the current player's PLR_IT marker")
	}
	if got := target.GetWaitState(); got != 0 {
		t.Fatalf("NPC tag wait state = %d, want 0", got)
	}
}

func TestUsersListsDescriptorsButOmitsLinklessCharacters(t *testing.T) {
	f := newConPlayingFixture(t)
	f.menu.player.SetInvisLevel(game.LVL_IMPL)
	if err := ExecuteCommand(f.actor, "users", nil); err != nil {
		t.Fatalf("dispatch users: %v", err)
	}
	got := strings.Join(drainSessionText(t, f.actor), "")
	for _, row := range [][2]string{
		{"Actor", "Playing"},
		{"Playing", "Playing"},
		{"Menu", "Main Menu"},
		{"Creating", "Confirm name"},
		{"Editing", "Room edit"},
		{"TextEditing", "Text edit"},
	} {
		want := fmt.Sprintf("%-12s %-14s", row[0], row[1])
		if !strings.Contains(got, want) {
			t.Errorf("users output missing exact row bytes for %s/%s: %q", row[0], row[1], got)
		}
	}
	if !strings.Contains(got, "\r\n8 visible sockets connected.\r\n") {
		t.Errorf("users footer bytes do not count the eight descriptors: %q", got)
	}
	if strings.Contains(got, "Linkless") {
		t.Fatalf("users listed a character with no descriptor: %q", got)
	}
}

func TestWhereNoArgumentOnlyListsConPlayingDescriptors(t *testing.T) {
	f := newConPlayingFixture(t)
	if err := ExecuteCommand(f.actor, "where", nil); err != nil {
		t.Fatalf("dispatch where: %v", err)
	}
	got := strings.Join(drainSessionText(t, f.actor), "")
	for _, want := range []string{"Actor", "Playing", "Mailing", "NoteWrite"} {
		if !strings.Contains(got, want) {
			t.Errorf("where output missing playing name %q: %q", want, got)
		}
	}
	for _, excluded := range []string{"Menu", "Creating", "Editing", "TextEditing", "Linkless"} {
		if strings.Contains(got, excluded) {
			t.Errorf("where output included non-playing %q: %q", excluded, got)
		}
	}
}
