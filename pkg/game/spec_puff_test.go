package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/dprng"
)

func TestSpecPuff_CommandAndDeathGates(t *testing.T) {
	w, _, lastMsg := newSpecProcTestWorld(t)
	mob := newSpecProcTestMob(t, w, 1001, 10)
	lastMsg() // discard the spawn announcement

	dprng.ResetStream(1)
	if specPuff(w, nil, mob, "look", "") {
		t.Fatal("command-triggered puff should fall through")
	}
	if got := lastMsg(); got != "" {
		t.Fatalf("command-triggered puff output = %q, want empty", got)
	}

	mob.CurrentHP = -1
	if !specPuff(w, nil, mob, "", "") {
		t.Fatal("dead puff should be handled")
	}
	if got := lastMsg(); got != "A test mob states, 'Shit, I'm dead.'\r\n" {
		t.Fatalf("dead puff output = %q, want C do_say output", got)
	}
}

func TestSpecPuff_CaseOutputsAndReturnContract(t *testing.T) {
	tests := []struct {
		name       string
		roll       int
		wantHandle bool
		wantOutput string
	}{
		{
			name:       "exclaims",
			roll:       0,
			wantHandle: true,
			wantOutput: "A test mob exclaims, 'My god!  It's full of stars!'\r\n",
		},
		{
			name:       "states",
			roll:       3,
			wantHandle: true,
			wantOutput: "A test mob states, 'I've got this peaceful, easy feeling.'\r\n",
		},
		{
			name:       "room-emote",
			roll:       13,
			wantHandle: true,
			wantOutput: "A test mob looks at you and then breaks out in a fit of laughter!\r\n",
		},
		{
			name:       "silent-handled",
			roll:       4,
			wantHandle: true,
		},
		{
			name:       "default-fallthrough",
			roll:       90,
			wantOutput: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, _, lastMsg := newSpecProcTestWorld(t)
			mob := newSpecProcTestMob(t, w, 1001, 10)
			lastMsg() // discard the spawn announcement
			seed := seedForPuffRoll(t, tt.roll)
			dprng.ResetStream(seed)

			if got := specPuff(w, nil, mob, "", ""); got != tt.wantHandle {
				t.Fatalf("handled = %v, want %v", got, tt.wantHandle)
			}
			if got := lastMsg(); got != tt.wantOutput {
				t.Fatalf("output = %q, want %q", got, tt.wantOutput)
			}
		})
	}
}

func TestSpecPuff_CompleteCOutcomePartition(t *testing.T) {
	wantOutput := map[int]string{
		0:  "A test mob exclaims, 'My god!  It's full of stars!'\r\n",
		1:  "A test mob asks, 'How'd all those fish get up here?'\r\n",
		2:  "A test mob states, 'I'm a very female dragon.'\r\n",
		3:  "A test mob states, 'I've got this peaceful, easy feeling.'\r\n",
		7:  "A test mob exclaims, 'Goddamn, what a trip! Listen to those colors!'\r\n",
		8:  "A test mob exclaims, 'Bring out your dead!'\r\n",
		9:  "A test mob states, 'Rule number 6...there is NO rule number 6.'\r\n",
		10: "A test mob exclaims, 'To be rich is no longer a sin...its a MIRACLE!'\r\n",
		13: "A test mob looks at you and then breaks out in a fit of laughter!\r\n",
		15: "A test mob asks, 'What is the sound of down?'\r\n",
		17: "A test mob wonders where she left that darn wand.\r\n",
		20: "A test mob asks, 'Do you want to stroke my tail?'\r\n",
		21: "A test mob asks, 'Do you want to stroke my tail?'\r\n",
		23: "A test mob does female stuff.\r\n",
		24: "A test mob does female stuff.\r\n",
		26: "A test mob contemplates the meaning of life.\r\n",
		27: "A test mob exclaims, 'NIH!'\r\n",
		31: "A test mob rocks out to some funky beats.\r\n",
		32: "A test mob rocks out to some funky beats.\r\n",
		37: "A test mob exclaims, 'I'm gonna kick your ASS!'\r\n",
		38: "A test mob exclaims, 'I'm gonna kick your ASS!'\r\n",
		39: "A test mob exclaims, 'I'm gonna kick your ASS!'\r\n",
	}

	for roll := 0; roll <= 90; roll++ {
		w, _, lastMsg := newSpecProcTestWorld(t)
		mob := newSpecProcTestMob(t, w, 1001, 10)
		lastMsg() // discard the spawn announcement
		seed := seedForPuffRoll(t, roll)
		dprng.ResetStream(seed)

		handled := specPuff(w, nil, mob, "", "")
		output := lastMsg()
		if roll <= 42 && !handled {
			t.Fatalf("roll %d: handled = false, want true", roll)
		}
		if roll >= 43 && handled {
			t.Fatalf("roll %d: handled = true, want C default false", roll)
		}
		if want := wantOutput[roll]; output != want {
			t.Fatalf("roll %d: output = %q, want %q", roll, output, want)
		}
	}
}

func TestSpecPuff_EmoteHidesInvisibleMob(t *testing.T) {
	w, _, lastMsg := newSpecProcTestWorld(t)
	mob := newSpecProcTestMob(t, w, 1001, 10)
	lastMsg() // discard the spawn announcement
	mob.SetAffected(affInvisible)
	dprng.ResetStream(seedForPuffRoll(t, 13))

	if !specPuff(w, nil, mob, "", "") {
		t.Fatal("invisible emote case should be handled")
	}
	if got := lastMsg(); got != "" {
		t.Fatalf("invisible mob emote output = %q, want hidden from observer", got)
	}
}

func seedForPuffRoll(t *testing.T, want int) uint32 {
	t.Helper()
	for seed := uint32(1); seed < 10000; seed++ {
		if dprng.New(seed).Number(0, 90) == want {
			return seed
		}
	}
	t.Fatalf("could not find a seed for puff roll %d", want)
	return 0
}
