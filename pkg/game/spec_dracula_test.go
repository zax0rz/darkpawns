package game

import (
	"strings"
	"testing"
)

func TestSpecDracula_EntryGatesAndDelegation(t *testing.T) {
	w, player, lastMsg := newSpecProcTestWorld(t)
	mob := newSpecProcTestMob(t, w, player.GetRoomVNum(), 10)
	mob.Prototype.Keywords = "Lothar Vampire Lord"
	mob.Prototype.ShortDesc = "Lothar the Vampire Lord"
	player.Stats.Int = 10
	player.Stats.Wis = 10
	lastMsg() // discard the spawn announcement

	tests := []struct {
		name string
		ch   *Player
		cmd  string
		arg  string
		want bool
	}{
		{name: "missing player", ch: nil, cmd: "look", arg: "lothar", want: false},
		{name: "non-look command", ch: player, cmd: "examine", arg: "lothar", want: false},
		{name: "wrong target", ch: player, cmd: "look", arg: "someone", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := specDracula(w, tt.ch, mob, tt.cmd, tt.arg); got != tt.want {
				t.Fatalf("handled = %v, want %v", got, tt.want)
			}
			if got := lastMsg(); got != "" {
				t.Fatalf("output = %q, want empty", got)
			}
		})
	}

	player.SetPlrFlag(PrfNohassle, true)
	if specDracula(w, player, mob, "look", "lothar") {
		t.Fatal("nohassle player should not trigger Dracula")
	}
	if got := lastMsg(); got != "" {
		t.Fatalf("nohassle output = %q, want empty", got)
	}
	player.SetPlrFlag(PrfNohassle, false)

	mob.SetPosition(posFighting)
	mob.SetFighting(player.GetName())
	player.SetFighting(mob.GetName())
	if !specDracula(w, nil, mob, "", "") {
		t.Fatal("fighting commandless Dracula should delegate to magic_user")
	}
	if got := lastMsg(); got != "" {
		t.Fatalf("delegated magic_user output = %q, want empty", got)
	}
}

func TestSpecDracula_LookBiteAudienceAndVampireState(t *testing.T) {
	w, player, lastMsg := newSpecProcTestWorld(t)
	observer := NewPlayer(2, "Observer", player.GetRoomVNum())
	observer.Stats.Int = 10
	observer.Stats.Wis = 10
	if err := w.AddPlayer(observer); err != nil {
		t.Fatalf("AddPlayer observer: %v", err)
	}
	mob := newSpecProcTestMob(t, w, player.GetRoomVNum(), 10)
	mob.Prototype.Keywords = "Lothar Vampire Lord"
	mob.Prototype.ShortDesc = "Lothar the Vampire Lord"
	player.Stats.Int = 10
	player.Stats.Wis = 10
	lastMsg() // discard the spawn announcement

	transcript := make(map[string]string)
	w.MessageSink = func(playerName string, msg []byte) {
		transcript[playerName] += string(msg)
	}

	if !specDracula(w, player, mob, "look", "lothar") {
		t.Fatal("keyword abbreviation should trigger Dracula")
	}
	if got, want := transcript[player.GetName()], "You feel mesmerized... your will weakens.\r\n"+
		"Lothar the Vampire Lord sinks his fangs into your neck!\r\n"+
		"You exclaim 'Now I know... The blood is the life!'\r\n"+
		"Your blood boils with a stinging fire...\r\n"; got != want {
		t.Errorf("actor output = %q, want %q", got, want)
	}
	if got, want := transcript[observer.GetName()],
		"Tester looks at Lothar the Vampire Lord.\r\n\r\n"+
			"Lothar the Vampire Lord gazes intently at Tester.\r\n\r\n"+
			"Lothar the Vampire Lord sinks his fangs into Tester!\r\n\r\n"+
			"Tester exclaims, 'Now I know... The blood is the life!'\r\n"; got != want {
		t.Errorf("observer output = %q, want %q", got, want)
	}
	if got := player.GetFlags(); got&(1<<uint(PlrVampire)) == 0 {
		t.Fatal("eligible human player was not made a vampire")
	}
}

func TestSpecDracula_ExistingVampireOrWerewolfSkipsTransformation(t *testing.T) {
	for _, flag := range []int{PlrVampire, PlrWerewolf} {
		t.Run(map[int]string{PlrVampire: "vampire", PlrWerewolf: "werewolf"}[flag], func(t *testing.T) {
			w, player, lastMsg := newSpecProcTestWorld(t)
			mob := newSpecProcTestMob(t, w, player.GetRoomVNum(), 10)
			mob.Prototype.Keywords = "Lothar Vampire Lord"
			mob.Prototype.ShortDesc = "Lothar the Vampire Lord"
			lastMsg()
			player.Stats.Int = 10
			player.Stats.Wis = 10
			player.SetPlrFlag(flag, true)

			if !specDracula(w, player, mob, "look", "lothar") {
				t.Fatal("eligible player should trigger Dracula")
			}
			if got := lastMsg(); got == "" || containsDraculaTransformation(got) {
				t.Fatalf("existing %s transformation output = %q", map[int]string{PlrVampire: "vampire", PlrWerewolf: "werewolf"}[flag], got)
			}
			if got := player.GetFlags(); flag == PlrWerewolf && got&(1<<uint(PlrVampire)) != 0 {
				t.Fatal("werewolf was incorrectly changed into a vampire")
			}
		})
	}
}

func containsDraculaTransformation(output string) bool {
	return strings.Contains(output, "Your blood boils with a stinging fire...")
}
