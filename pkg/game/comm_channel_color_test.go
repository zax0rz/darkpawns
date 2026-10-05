package game

import "testing"

// completeColor gives a player COLOR_LEV C_CMP (3): both color bits
// (src/screen.h:47-49).
func completeColor(p *Player) {
	p.SetPlrFlag(PrfColor1, true)
	p.SetPlrFlag(PrfColor2, true)
}

// normalColor gives a player COLOR_LEV C_NRM (2): PRF_COLOR_2 only.
func normalColor(p *Player) {
	p.SetPlrFlag(PrfColor1, false)
	p.SetPlrFlag(PrfColor2, true)
}

// TestGenCommChannelColorMatchesC proves DP-1384. C's do_gen_comm copies the
// channel color out of com_msgs[subcmd][3] (src/act.comm.c:1185-1259) and
// wraps the speaker's echo when COLOR_LEV(ch) >= C_CMP, with KNRM folded into
// the sprintf before act appends the line ending, and wraps each listener when
// COLOR_LEV(listener) >= C_NRM, with KNRM a separate write after act's whole
// line (src/act.comm.c:1262-1295). Before the port carried a color per channel
// none of the six was ever colored.
func TestGenCommChannelColorMatchesC(t *testing.T) {
	const (
		yel = "\x1b[33m" // KYEL
		grn = "\x1b[32m" // KGRN
		mag = "\x1b[35m" // KMAG
		rst = "\x1b[0m"  // KNRM
	)
	// com_msgs[subcmd][3]: holler/shout/gossip/newbie KYEL, auction KMAG,
	// grats KGRN.
	channels := []struct{ command, verb, color string }{
		{"newbie", "newbie", yel},
		{"holler", "holler", yel},
		{"gossip", "gossip", yel},
		{"grats", "congrat", grn},
		{"auction", "auction", mag},
		{"shout", "shout", yel},
	}

	t.Run("complete colors the sender echo and the listener", func(t *testing.T) {
		for _, channel := range channels {
			t.Run(channel.command, func(t *testing.T) {
				w, actor, local, _, output := newChannelWorld(t)
				completeColor(actor)
				completeColor(local)
				actor.SetMove(100)
				w.DoChannel(actor, "hi", channel.command)

				// Speaker echo: sprintf("%sYou %s, '%s'%s"), so KNRM lands
				// before the newline act appends.
				wantEcho := channel.color + "You " + channel.verb + ", 'hi'" + rst + "\r\n"
				if got := channelOutput(output, actor.Name); got != wantEcho {
					t.Fatalf("echo = %q, want %q", got, wantEcho)
				}
				// Listener: send_to_char(color_on) / act() / send_to_char(KNRM),
				// so KNRM follows act's whole line.
				wantLine := channel.color + "Actor " + channel.verb + "s, 'hi'\r\n" + rst
				if got := channelOutput(output, local.Name); got != wantLine {
					t.Fatalf("listener = %q, want %q", got, wantLine)
				}
			})
		}
	})

	t.Run("normal colors the listener but not the sender echo", func(t *testing.T) {
		w, actor, local, _, output := newChannelWorld(t)
		normalColor(actor)
		normalColor(local)
		w.DoChannel(actor, "hi", "gossip")
		if got := channelOutput(output, actor.Name); got != "You gossip, 'hi'\r\n" {
			t.Fatalf("echo = %q; C_CMP is needed for the echo, so it stays plain", got)
		}
		if want := yel + "Actor gossips, 'hi'\r\n" + rst; channelOutput(output, local.Name) != want {
			t.Fatalf("listener = %q, want %q", channelOutput(output, local.Name), want)
		}
	})

	t.Run("color off leaves both plain", func(t *testing.T) {
		w, actor, local, _, output := newChannelWorld(t)
		w.DoChannel(actor, "hi", "auction")
		if got := channelOutput(output, actor.Name); got != "You auction, 'hi'\r\n" {
			t.Fatalf("echo = %q", got)
		}
		if got := channelOutput(output, local.Name); got != "Actor auctions, 'hi'\r\n" {
			t.Fatalf("listener = %q", got)
		}
	})
}

// TestGenCommMobGossipColorMatchesC pins the NPC-authored do_gen_comm
// (SCMD_GOSSIP) path. An NPC has no descriptor, so there is no self echo, but
// each listener is still wrapped at COLOR_LEV >= C_NRM exactly as for a player
// author (src/act.comm.c:1288-1295).
func TestGenCommMobGossipColorMatchesC(t *testing.T) {
	w, mob, actor, peer, messages := newQuanLoTestWorld(t)
	completeColor(actor)
	completeColor(peer)
	w.mobGlobalGossip(mob, "colored native message.")
	want := "\x1b[33mQuan Lo gossips, 'colored native message.'\r\n\x1b[0m"
	for _, player := range []*Player{actor, peer} {
		if got := messages[player.Name]; got != want {
			t.Errorf("%s gossip = %q, want %q", player.Name, got, want)
		}
	}
}
