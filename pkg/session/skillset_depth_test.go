package session

import (
	"testing"
)

func TestCmdSkillsetUsesCVisibleTargetResolver(t *testing.T) {
	wiz, target := makeSkillsetTestSession(t)

	if err := cmdSkillset(wiz, []string{"me", "'backstab'", "50"}); err != nil {
		t.Fatalf("cmdSkillset self alias: %v", err)
	}
	if got, want := readOneText(t, wiz), "You change God's backstab to 50.\n\r"; got != want {
		t.Fatalf("self alias response = %q, want %q", got, want)
	}
	if got := wiz.player.GetSkill("backstab"); got != 50 {
		t.Fatalf("self alias skill = %d, want 50", got)
	}

	if err := cmdSkillset(wiz, []string{"Hero", "'backstab'", "51"}); err != nil {
		t.Fatalf("cmdSkillset room player: %v", err)
	}
	if got, want := readOneText(t, wiz), "You change Hero's backstab to 51.\n\r"; got != want {
		t.Fatalf("room player response = %q, want %q", got, want)
	}
	if got := target.player.GetSkill("backstab"); got != 51 {
		t.Fatalf("room player skill = %d, want 51", got)
	}
}
