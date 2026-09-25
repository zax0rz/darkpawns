package game

import "testing"

type testSessions []*Player

func (s testSessions) EachSession(fn func(player interface{}, send func(msg string))) {
	for _, p := range s {
		p := p
		fn(p, p.SendMessage)
	}
}

// mudlog's syslog level is PRF_LOG1 (1) + PRF_LOG2 (2); a message reaches an
// immortal at or above its level whose syslog level is at least its type,
// unless they are writing (utils.c:258-270).
func TestMudLogSyslogLevels(t *testing.T) {
	w, _, _, _, output := newChannelWorld(t)
	mk := func(name string, level int, log1, log2, writing bool) *Player {
		p := channelPlayer(t, w, len(w.GetAllPlayers())+10, name, 1001)
		p.Level = level
		p.SetPlrFlag(PrfLog1, log1)
		p.SetPlrFlag(PrfLog2, log2)
		p.SetPlrFlag(PlrWriting, writing)
		return p
	}
	complete := mk("Complete", 34, true, true, false)
	normal := mk("Normal", 34, false, true, false)
	brief := mk("Brief", 34, true, false, false)
	writer := mk("Writer", 34, true, true, true)
	mortal := mk("Mortal", 10, true, true, false)

	prev := getImmortalSessionProvider()
	SetImmortalSessionProvider(testSessions{complete, normal, brief, writer, mortal})
	defer SetImmortalSessionProvider(prev)

	MudLog("cmp line", 3, lvlImmort, false)
	MudLog("nrm line", MudlogNormal, lvlImmort, false)
	MudLog("god line", MudlogBrief, 35, false)

	want := map[string]string{
		"Complete": "[ cmp line ]\r\n[ nrm line ]\r\n",
		"Normal":   "[ nrm line ]\r\n",
		"Brief":    "",
		"Writer":   "",
		"Mortal":   "",
	}
	for name, w := range want {
		if got := channelOutput(output, name); got != w {
			t.Errorf("%s saw %q, want %q", name, got, w)
		}
	}
}
