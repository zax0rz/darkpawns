package session

import (
	"encoding/json"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/scripting"
)

func TestConPlayingSessionFanoutMatchesCStates(t *testing.T) {
	f := newConPlayingFixture(t)
	for _, message := range []struct{ kind, text string }{
		{"broadcast", "Manager broadcast bytes.\r\n"},
		{"outdoor", "The outside weather changes.\r\n"},
	} {
		if message.kind == "broadcast" {
			f.manager.SendToAll(message.text)
		} else {
			f.manager.SendToOutdoor(message.text)
		}
		for _, s := range []*Session{f.actor, f.playing, f.mailing, f.noteWrite} {
			if got := drainSessionEventText(t, s); !equalStrings(got, []string{message.text}) {
				t.Errorf("%s %s event bytes = %q, want %q", s.playerName, message.kind, got, message.text)
			}
		}
		for name, s := range map[string]*Session{
			"menu": f.menu, "character creation": f.creating, "OLC editor": f.editing, "TEDIT": f.textEditing, "linkless": f.linkless,
		} {
			if got := drainSessionEventText(t, s); len(got) != 0 {
				t.Errorf("%s descriptor received %s event: %q", name, message.kind, got)
			}
		}
	}
}

func TestManagerEachSessionOnlyVisitsConPlayingDescriptors(t *testing.T) {
	f := newConPlayingFixture(t)
	var got []string
	f.manager.EachSession(func(player interface{}, _ func(string)) {
		got = append(got, player.(*game.Player).Name)
	})
	gotSet := make(map[string]bool, len(got))
	for _, name := range got {
		gotSet[name] = true
	}
	for _, want := range []string{"Actor", "Playing", "Mailing", "NoteWrite"} {
		if !gotSet[want] {
			t.Errorf("mudlog provider omitted CON_PLAYING session %q: %v", want, got)
		}
	}
	for _, excluded := range []string{"Menu", "Creating", "Editing", "TextEditing", "Linkless"} {
		if gotSet[excluded] {
			t.Errorf("mudlog provider included non-playing %q: %v", excluded, got)
		}
	}
	if len(gotSet) != 4 {
		t.Fatalf("mudlog provider sessions = %v, want exactly the four CON_PLAYING descriptors", got)
	}
}

func TestMudLogCombinesDescriptorAndWritingGates(t *testing.T) {
	f := newConPlayingFixture(t)
	for _, s := range []*Session{f.actor, f.playing, f.mailing, f.noteWrite, f.menu, f.creating, f.editing, f.textEditing, f.linkless} {
		s.player.SetLevel(game.LVL_IMPL)
		s.player.SetPlrFlag(game.PrfLog2, true)
	}
	game.MudLog("descriptor probe", game.MudlogNormal, game.LVL_GOD, false)
	for _, s := range []*Session{f.actor, f.playing} {
		if got, want := drainSessionText(t, s), []string{"[ descriptor probe ]\r\n"}; !equalStrings(got, want) {
			t.Errorf("%s mudlog output = %q, want %q", s.playerName, got, want)
		}
	}
	for name, s := range map[string]*Session{
		"mail writer": f.mailing, "note writer": f.noteWrite, "menu": f.menu,
		"character creation": f.creating, "OLC editor": f.editing, "TEDIT": f.textEditing,
		"linkless": f.linkless,
	} {
		if got := drainSessionText(t, s); len(got) != 0 {
			t.Errorf("%s received mudlog: %q", name, got)
		}
	}
}

func TestScriptableOutdoorAndGlobalEchoOnlyVisitConPlayingDescriptors(t *testing.T) {
	f := newConPlayingFixture(t)
	adapter := game.NewWorldScriptableAdapter(f.manager.world)
	const outdoor = "The weather turns.\r\n"
	adapter.Echo("outdoor", nil, nil, outdoor)
	for _, s := range []*Session{f.actor, f.playing, f.mailing, f.noteWrite} {
		if got := drainSessionText(t, s); !equalStrings(got, []string{outdoor}) {
			t.Errorf("%s script outdoor bytes = %q, want %q", s.playerName, got, outdoor)
		}
	}
	for name, s := range map[string]*Session{
		"menu": f.menu, "character creation": f.creating, "OLC editor": f.editing, "TEDIT": f.textEditing, "linkless": f.linkless,
	} {
		if got := drainSessionText(t, s); len(got) != 0 {
			t.Errorf("%s descriptor received script outdoor echo: %q", name, got)
		}
	}

	const global = "A script global echo.\r\n"
	actorRef := scripting.CharRef{ID: f.actor.player.GetID()}
	adapter.Echo("global", nil, &actorRef, "A script global echo.")
	for _, s := range []*Session{f.actor, f.playing, f.mailing, f.noteWrite} {
		if got := drainSessionText(t, s); !equalStrings(got, []string{global}) {
			t.Errorf("%s script global bytes = %q, want %q", s.playerName, got, global)
		}
	}
	for name, s := range map[string]*Session{
		"menu": f.menu, "character creation": f.creating, "OLC editor": f.editing, "TEDIT": f.textEditing, "linkless": f.linkless,
	} {
		if got := drainSessionText(t, s); len(got) != 0 {
			t.Errorf("%s descriptor received script global echo: %q", name, got)
		}
	}
}

func drainSessionEventText(t *testing.T, s *Session) []string {
	t.Helper()
	var out []string
	for {
		select {
		case msg := <-s.send:
			var serverMessage ServerMessage
			if err := json.Unmarshal(msg, &serverMessage); err != nil {
				t.Fatalf("session message: %v", err)
			}
			if serverMessage.Type != MsgEvent {
				t.Fatalf("session message type = %q, want event", serverMessage.Type)
			}
			data, _ := json.Marshal(serverMessage.Data)
			var event EventData
			if err := json.Unmarshal(data, &event); err != nil {
				t.Fatalf("event data: %v", err)
			}
			out = append(out, event.Text)
		default:
			return out
		}
	}
}
