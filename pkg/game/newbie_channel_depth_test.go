package game

import "testing"

func TestDoNewbieChannelDepthGatesAndFanout(t *testing.T) {
	t.Run("level one exception and fanout", func(t *testing.T) {
		w, actor, local, remote, output := newChannelWorld(t)
		actor.SetLevel(1)
		w.DoChannel(actor, "hello", "newbie")
		if got := channelOutput(output, actor.Name); got != "You newbie, 'hello'\r\n" {
			t.Fatalf("actor output = %q", got)
		}
		for _, target := range []*Player{local, remote} {
			if got := channelOutput(output, target.Name); got != "Actor newbies, 'hello'\r\n" {
				t.Fatalf("%s output = %q", target.Name, got)
			}
		}
	})

	t.Run("empty argument", func(t *testing.T) {
		w, actor, _, _, output := newChannelWorld(t)
		w.DoChannel(actor, " \r\n", "newbie")
		if got := channelOutput(output, actor.Name); got != "Yes, newbie, fine, newbie we must, but WHAT???\r\n" {
			t.Fatalf("output = %q", got)
		}
	})

	t.Run("sender gates", func(t *testing.T) {
		for _, test := range []struct {
			name  string
			setup func(*Player)
			want  string
		}{
			{name: "noshout", setup: func(p *Player) { p.SetPlrFlag(PlrNoshout, true) }, want: "You cannot newbie!\r\n"},
			{name: "channel off", setup: func(p *Player) { p.SetPlrFlag(PrfNoNewbie, true) }, want: "You aren't even on the channel!\r\n"},
			{name: "soundproof", setup: func(p *Player) { p.SetRoom(1002) }, want: "The walls seem to absorb your words.\r\n"},
			{name: "stupid", setup: func(p *Player) { p.Stats.Int = 0 }, want: "You are too stupid to communicate with language!\r\n"},
		} {
			t.Run(test.name, func(t *testing.T) {
				w, actor, _, _, output := newChannelWorld(t)
				test.setup(actor)
				w.DoChannel(actor, "hello", "newbie")
				if got := channelOutput(output, actor.Name); got != test.want {
					t.Fatalf("output = %q, want %q", got, test.want)
				}
			})
		}
	})

	t.Run("recipient channel off", func(t *testing.T) {
		w, actor, local, _, output := newChannelWorld(t)
		local.SetPlrFlag(PrfNoNewbie, true)
		w.DoChannel(actor, "hello", "newbie")
		if got := channelOutput(output, actor.Name); got != "You newbie, 'hello'\r\n" {
			t.Fatalf("actor output = %q", got)
		}
		if got := channelOutput(output, local.Name); got != "" {
			t.Fatalf("muted recipient output = %q", got)
		}
	})

	t.Run("recipient writing", func(t *testing.T) {
		w, actor, local, _, output := newChannelWorld(t)
		local.SetPlrFlag(PlrWriting, true)
		w.DoChannel(actor, "hello", "newbie")
		if got := channelOutput(output, local.Name); got != "" {
			t.Fatalf("writing recipient output = %q", got)
		}
	})
}
