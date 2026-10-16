package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestStatRoomIncludesExtraDescriptionKeywordsAtCColorLevel(t *testing.T) {
	parsed := &parser.World{Rooms: []parser.Room{{
		VNum:        1001,
		Name:        "Map Room",
		Description: "A map room.",
		ExtraDescs: []parser.ExtraDesc{{
			Keywords: "map charts",
		}},
	}}}
	world, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(func() { world.StopAITicker() })
	manager := newTestManager(t, world, nil)
	session := makeTestSession(t, manager, "Statgod", 1001, true)
	if err := world.AddPlayer(session.player); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}

	for _, test := range []struct {
		name  string
		color bool
		want  string
	}{
		{name: "color off", want: "Extra descs: map charts\r\n"},
		{name: "normal color", color: true, want: "Extra descs:\x1b[36m map charts\x1b[0m\r\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.color {
				session.player.SetPlrFlag(game.PrfColor2, true)
			}
			session.sendStatRoom()
			var output strings.Builder
			for len(session.send) > 0 {
				output.WriteString(readMsgText(t, session))
			}
			got := output.String()
			if !strings.Contains(got, test.want) {
				t.Fatalf("stat room output lacks %q: %q", test.want, got)
			}
		})
	}
}
