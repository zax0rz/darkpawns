package game

import "testing"

func TestRaceSayUsesCPlayerRaceLanguageTables(t *testing.T) {
	tests := []struct {
		name        string
		race        int
		translation string
	}{
		{name: "human", race: RaceHuman, translation: "keen ar yoi?"},
		{name: "elf", race: RaceElf, translation: "quad est yoi?"},
		{name: "dwarf", race: RaceDwarf, translation: "var icht yoi?"},
		{name: "kender", race: RaceKender, translation: "angti ese yoi?"},
		{name: "minotaur", race: RaceMinotaur, translation: "hi'fen era yoi?"},
		{name: "rakshasa", race: RaceRakshasa, translation: "ciss nec yoii?"},
		{name: "ssaur", race: RaceSsaur, translation: "hi'fen era yoi?"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, actor, target, output := newDirectedSpeechWorld(t)
			actor.Race = tt.race
			target.Race = (tt.race + 1) % 7

			w.doRaceSay(actor, nil, "rsay", "how are you?")

			want := "Actor asks, '" + tt.translation + "'\r\n"
			if got := directedOutput(output, target.Name); got != want {
				t.Fatalf("translated output = %q, want %q", got, want)
			}
		})
	}
}

func TestRaceSayUsesCControlBytesAndGateOrder(t *testing.T) {
	t.Run("empty uses LFCR", func(t *testing.T) {
		w, actor, _, output := newDirectedSpeechWorld(t)
		w.doRaceSay(actor, nil, "rsay", "")
		if got, want := directedOutput(output, actor.Name), "Yes, but WHAT do you want to say?\n\r"; got != want {
			t.Fatalf("empty output = %q, want %q", got, want)
		}
	})

	t.Run("stupid precedes empty", func(t *testing.T) {
		w, actor, _, output := newDirectedSpeechWorld(t)
		actor.Stats.Int = 0
		w.doRaceSay(actor, nil, "rsay", "")
		if got, want := directedOutput(output, actor.Name), "You are too stupid to communicate with language!\r\n"; got != want {
			t.Fatalf("stupid output = %q, want %q", got, want)
		}
	})

	t.Run("noshout precedes empty", func(t *testing.T) {
		w, actor, _, output := newDirectedSpeechWorld(t)
		actor.SetPlrFlag(PlrNoshout, true)
		w.doRaceSay(actor, nil, "rsay", "")
		if got, want := directedOutput(output, actor.Name), "You cannot race-say!\r\n"; got != want {
			t.Fatalf("noshout output = %q, want %q", got, want)
		}
	})

	t.Run("norepeat uses LFCR confirmation", func(t *testing.T) {
		w, actor, target, output := newDirectedSpeechWorld(t)
		target.Race = RaceElf
		actor.SetPlrFlag(PrfNoRepeat, true)
		w.doRaceSay(actor, nil, "rsay", "how are you?")
		if got, want := directedOutput(output, actor.Name), "Ok.\n\r"; got != want {
			t.Fatalf("norepeat output = %q, want %q", got, want)
		}
		if got, want := directedOutput(output, target.Name), "Actor asks, 'keen ar yoi?'\r\n"; got != want {
			t.Fatalf("norepeat target output = %q, want %q", got, want)
		}
	})
}
