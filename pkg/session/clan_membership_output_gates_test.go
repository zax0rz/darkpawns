package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func clanOutputGateFixture(t *testing.T) (*Manager, *game.Clan, *game.Clan, map[string]*Session) {
	t.Helper()
	m, w := mudlogTestWorld(t)
	w.Clans = game.NewClanManager()
	wolves := &game.Clan{Name: "Wolves", Ranks: 3}
	tigers := &game.Clan{Name: "Tigers", Ranks: 3}
	for _, clan := range []*game.Clan{wolves, tigers} {
		clan.Privilege[game.CPEnroll] = 1
		clan.Privilege[game.CPExpel] = 1
		clan.Privilege[game.CPPromote] = 1
		clan.Privilege[game.CPDemote] = 1
		w.Clans.AddClan(clan)
	}

	players := map[string]*Session{}
	add := func(name string, level int, clan *game.Clan, rank int) {
		s := makeTestSession(t, m, name, 1001, true)
		s.player.SetLevel(level)
		if clan != nil {
			s.player.ClanID = clan.ID
			s.player.ClanRank = rank
		}
		registerTestSession(t, m, s, strings.ToLower(name))
		drainSessionText(t, s)
		players[name] = s
	}

	add("ClanGod", game.LVL_GOD, nil, 0)
	add("LowGod", game.LVL_IMMORT+1, nil, 0)
	add("Clanless", 1, nil, 0)
	add("Leader", 1, wolves, 3)
	add("Applicant", 1, wolves, 1)
	add("Peer", 1, wolves, 3)
	add("RankOne", 1, wolves, 1)
	add("RankTwo", 1, wolves, 2)
	add("Unenrolled", 1, wolves, 0)
	add("Other", 1, tigers, 1)
	return m, wolves, tigers, players
}

func runClanOutputCommand(t *testing.T, actor *Session, args ...string) string {
	t.Helper()
	if err := ExecuteCommand(actor, "clan", args); err != nil {
		t.Fatalf("clan %s: %v", strings.Join(args, " "), err)
	}
	return strings.Join(drainSessionText(t, actor), "")
}

func TestClanExpelOutputGates(t *testing.T) {
	_, wolves, _, players := clanOutputGateFixture(t)
	checks := []struct {
		name  string
		actor string
		args  []string
		want  string
	}{
		{"no clan", "Clanless", []string{"expel", "Applicant"}, "You don't belong to any clan!\r\n"},
		{"low immortal", "LowGod", []string{"expel", "Applicant", "Wolves"}, "You do not have clan privileges.\r\n"},
		{"unknown clan", "ClanGod", []string{"expel", "Nobody", "Missing"}, "Unknown clan.\r\n"},
		{"missing target", "ClanGod", []string{"expel", "Nobody", "Wolves"}, "Er, Who ??\r\n"},
		{"other clan", "Leader", []string{"expel", "Other"}, "They're not in your clan.\r\n"},
		{"equal rank", "Leader", []string{"expel", "Peer"}, "You cannot kick out that person.\r\n"},
	}
	wolves.Privilege[game.CPExpel] = 2
	checks = append(checks, struct {
		name  string
		actor string
		args  []string
		want  string
	}{"insufficient influence", "Applicant", []string{"expel", "Nobody"}, "You're not influent enough in the clan to do that!\r\n"})

	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			if got := runClanOutputCommand(t, players[check.actor], check.args...); got != check.want {
				t.Fatalf("output = %q, want %q", got, check.want)
			}
		})
	}
	wolves.Privilege[game.CPExpel] = 1
}

func TestClanDemoteOutputGates(t *testing.T) {
	_, wolves, _, players := clanOutputGateFixture(t)
	checks := []struct {
		name  string
		actor string
		args  []string
		want  string
	}{
		{"no clan", "Clanless", []string{"demote", "Applicant"}, "You don't belong to any clan!\r\n"},
		{"low immortal", "LowGod", []string{"demote", "Applicant", "Wolves"}, "You do not have clan privileges.\r\n"},
		{"unknown clan", "ClanGod", []string{"demote", "Nobody", "Missing"}, "Unknown clan.\r\n"},
		{"missing target", "ClanGod", []string{"demote", "Nobody", "Wolves"}, "Er, Who ??\r\n"},
		{"other clan", "Leader", []string{"demote", "Other"}, "They're not in your clan.\r\n"},
		{"rank one", "ClanGod", []string{"demote", "RankOne", "Wolves"}, "They can't be demoted any further, use expel now.\r\n"},
		{"equal rank", "Leader", []string{"demote", "Peer"}, "You cannot demote a person of this rank!\r\n"},
	}
	wolves.Privilege[game.CPDemote] = 2
	checks = append(checks, struct {
		name  string
		actor string
		args  []string
		want  string
	}{"insufficient influence", "Applicant", []string{"demote", "Nobody"}, "You're not influent enough in the clan to do that!\r\n"})

	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			if got := runClanOutputCommand(t, players[check.actor], check.args...); got != check.want {
				t.Fatalf("output = %q, want %q", got, check.want)
			}
		})
	}
	wolves.Privilege[game.CPDemote] = 1

	god, target := players["ClanGod"], players["RankTwo"]
	if got := runClanOutputCommand(t, god, "demote", "RankTwo", "Wolves"); got != "Done.\r\n" {
		t.Fatalf("successful demote actor output = %q", got)
	}
	if got := strings.Join(drainSessionText(t, target), ""); got != "You've been demoted within your clan!\r\n" {
		t.Fatalf("successful demote target output = %q", got)
	}
}

func TestClanPromoteOutputGates(t *testing.T) {
	_, wolves, _, players := clanOutputGateFixture(t)
	checks := []struct {
		name  string
		actor string
		args  []string
		want  string
	}{
		{"no clan", "Clanless", []string{"promote", "Applicant"}, "You don't belong to any clan!\r\n"},
		{"unknown clan", "ClanGod", []string{"promote", "Nobody", "Missing"}, "Unknown clan.\r\n"},
		{"missing target", "ClanGod", []string{"promote", "Nobody", "Wolves"}, "Er, Who ??\r\n"},
		{"other clan", "Leader", []string{"promote", "Other"}, "They're not in your clan.\r\n"},
		{"not enrolled", "ClanGod", []string{"promote", "Unenrolled", "Wolves"}, "They're not enrolled yet.\r\n"},
		{"top rank", "ClanGod", []string{"promote", "Peer", "Wolves"}, "You cannot promote someone over the top rank!\r\n"},
	}
	wolves.Privilege[game.CPPromote] = 2
	checks = append(checks, struct {
		name  string
		actor string
		args  []string
		want  string
	}{"insufficient influence", "Applicant", []string{"promote", "Nobody"}, "You're not influent enough in the clan to do that!\r\n"})

	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			if got := runClanOutputCommand(t, players[check.actor], check.args...); got != check.want {
				t.Fatalf("output = %q, want %q", got, check.want)
			}
		})
	}
	wolves.Privilege[game.CPPromote] = 1
	if got := runClanOutputCommand(t, players["Applicant"], "promote", "RankTwo"); got != "You cannot promote that person over your rank!\r\n" {
		t.Fatalf("over-actor-rank output = %q", got)
	}

	god, target := players["ClanGod"], players["RankOne"]
	if got := runClanOutputCommand(t, god, "promote", "RankOne", "Wolves"); got != "Done.\r\n" {
		t.Fatalf("successful promote actor output = %q", got)
	}
	if got := strings.Join(drainSessionText(t, target), ""); got != "You've been promoted within your clan!\r\n" {
		t.Fatalf("successful promote target output = %q", got)
	}
}
