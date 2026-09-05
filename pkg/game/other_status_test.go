package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestDoAFKToggleState(t *testing.T) {
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001}}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	player := NewCharacter(1, "Afkstate", ClassWarrior, RaceHuman)
	player.SetRoom(1001)
	if err := w.AddPlayer(player); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}

	w.ExecAFK(player, "ignored argument")
	if !player.GetAFK() || player.GetFlags()&(1<<PrfAFK) == 0 {
		t.Fatal("enabling AFK did not set both the typed state and C preference bit")
	}
	if player.GetAFKMessage() != "" {
		t.Fatalf("AFK message = %q, want empty C-compatible message", player.GetAFKMessage())
	}

	w.ExecAFK(player, "ignored argument")
	if player.GetAFK() || player.GetFlags()&(1<<PrfAFK) != 0 {
		t.Fatal("disabling AFK did not clear both the typed state and C preference bit")
	}
}

func setTransformClimate(t *testing.T, sunlight, day, moon int) {
	t.Helper()
	weatherMu.Lock()
	oldTime, oldWeather := timeInfo, weatherInfo
	timeInfo.Day = day
	timeInfo.Moon = moon
	weatherInfo.Sunlight = sunlight
	weatherMu.Unlock()
	t.Cleanup(func() {
		weatherMu.Lock()
		timeInfo, weatherInfo = oldTime, oldWeather
		weatherMu.Unlock()
	})
}

func newTransformTestPlayer(t *testing.T) (*World, *Player, *strings.Builder) {
	t.Helper()
	w := &World{}
	var out strings.Builder
	w.MessageSink = func(_ string, msg []byte) { out.Write(msg) }
	p := NewPlayer(1, "Transform", 1001)
	p.worldRef = w
	return w, p, &out
}

func TestDoTransformMatchesCStateMachines(t *testing.T) {
	t.Run("ordinary player rejected", func(t *testing.T) {
		w, p, out := newTransformTestPlayer(t)
		w.doTransform(p, nil, "transform", "ignored")
		if got, want := out.String(), "You aren't transformable!\n\r"; got != want {
			t.Fatalf("rejection = %q, want %q", got, want)
		}
	})

	t.Run("werewolf no moon gate", func(t *testing.T) {
		setTransformClimate(t, SunDark, 3, MoonFull)
		w, p, out := newTransformTestPlayer(t)
		p.SetPlrFlag(PlrWerewolf, true)
		w.doTransform(p, nil, "transform", "ignored")
		if got, want := out.String(), "You can't transform when there's no moon in the sky!\r\n"; got != want {
			t.Fatalf("no-moon gate = %q, want %q", got, want)
		}
		if p.IsAffected(affWerewolf) || p.GetHP() != 100 {
			t.Fatalf("no-moon state = affected:%v hp:%d", p.IsAffected(affWerewolf), p.GetHP())
		}
	})

	t.Run("werewolf full moon transform and daytime revert", func(t *testing.T) {
		setTransformClimate(t, SunDark, 17, MoonFull)
		w, p, out := newTransformTestPlayer(t)
		p.SetPlrFlag(PlrWerewolf, true)
		w.doTransform(p, nil, "transform", "ignored")
		if got, want := out.String(), "Your nails grow into talons, and hair sprouts from every pore.\n\r"; got != want {
			t.Fatalf("transform = %q, want %q", got, want)
		}
		if !p.IsAffected(affWerewolf) || p.GetHP() != 150 {
			t.Fatalf("transform state = affected:%v hp:%d, want true/150", p.IsAffected(affWerewolf), p.GetHP())
		}

		weatherMu.Lock()
		weatherInfo.Sunlight = SunLight
		weatherMu.Unlock()
		out.Reset()
		w.doTransform(p, nil, "transform", "ignored")
		if got, want := out.String(), "Your hair and nails shorten and you revert to your normal shape.\n\r"; got != want {
			t.Fatalf("revert = %q, want %q", got, want)
		}
		if p.IsAffected(affWerewolf) || p.GetHP() != 100 {
			t.Fatalf("revert state = affected:%v hp:%d, want false/100", p.IsAffected(affWerewolf), p.GetHP())
		}
	})

	t.Run("vampire full moon transform and daytime revert", func(t *testing.T) {
		setTransformClimate(t, SunDark, 17, MoonFull)
		w, p, out := newTransformTestPlayer(t)
		p.SetPlrFlag(PlrVampire, true)
		w.doTransform(p, nil, "transform", "ignored")
		if got, want := out.String(), "Your nails grow transluscent and fangs sprout from your incisors!\n\r"; got != want {
			t.Fatalf("transform = %q, want %q", got, want)
		}
		if !p.IsAffected(affVampire) || p.GetMana() != 150 {
			t.Fatalf("transform state = affected:%v mana:%d, want true/150", p.IsAffected(affVampire), p.GetMana())
		}

		weatherMu.Lock()
		weatherInfo.Sunlight = SunLight
		weatherMu.Unlock()
		out.Reset()
		w.doTransform(p, nil, "transform", "ignored")
		if got, want := out.String(), "Your fangs recess, and you revert to your normal shape.\n\r"; got != want {
			t.Fatalf("revert = %q, want %q", got, want)
		}
		if p.IsAffected(affVampire) || p.GetMana() != 100 {
			t.Fatalf("revert state = affected:%v mana:%d, want false/100", p.IsAffected(affVampire), p.GetMana())
		}
	})
}
