package session

import "testing"

func TestClanRankRefusalOutputGates(t *testing.T) {
	_, _, _, players := clanOutputGateFixture(t)
	checks := []struct {
		name  string
		actor string
		args  []string
		want  string
	}{
		{"clanless", "Clanless", []string{"set", "ranks", "3"}, "You don't belong to any clan!\r\n"},
		{"low immortal", "LowGod", []string{"set", "ranks", "3", "Wolves"}, "You do not have clan privileges.\r\n"},
		{"unknown clan", "ClanGod", []string{"set", "ranks", "3", "Missing"}, "Unknown clan.\r\n"},
		{"insufficient influence", "Applicant", []string{"set", "ranks", "3"}, "You're not influent enough in the clan to do that!\r\n"},
		{"nonnumeric", "ClanGod", []string{"set", "ranks", "nope", "Wolves"}, "Set the ranks to what?\r\n"},
		{"same rank count", "ClanGod", []string{"set", "ranks", "3", "Wolves"}, "The clan already has this number of ranks.\r\n"},
		{"below range", "ClanGod", []string{"set", "ranks", "1", "Wolves"}, "Clans must have from 2 to 20 ranks.\r\n"},
		{"above range", "ClanGod", []string{"set", "ranks", "21", "Wolves"}, "Clans must have from 2 to 20 ranks.\r\n"},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			if got := runClanOutputCommand(t, players[check.actor], check.args...); got != check.want {
				t.Fatalf("output = %q, want %q", got, check.want)
			}
		})
	}

	leader := players["Leader"]
	leader.player.SetGold(0)
	if got := runClanOutputCommand(t, leader, "set", "ranks", "2"); got != "Changing the clan hierarchy requires 5,000 coins!\r\n" {
		t.Fatalf("insufficient gold output = %q", got)
	}
}
