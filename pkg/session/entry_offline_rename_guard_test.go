package session

import (
	"reflect"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// DP-1381: offline name edits cannot disturb online/entering identities.
func TestEntryOfflineRenameGuard(t *testing.T) {
	for _, state := range []string{"source-online", "source-linkdead", "source-case-only", "source-live-renamed", "destination-online", "destination-case-only", "destination-entering", "source-entering", "offline-success", "offline-with-guests"} {
		t.Run(state, func(t *testing.T) {
			database := entryDatabase(t)
			source := entrySeed(t, database, "Returner")
			m := entryWorldManager(t, database)
			wizard := makeCommandTestSession(t, m, "Wizard", game.LVL_IMPL, 1001)
			name := "Unused"
			var held *Session
			switch state {
			case "source-online", "source-linkdead", "source-case-only", "source-live-renamed":
				held = makeCharSession(t, m)
				if err := held.handleLogin(loginMsg("Returner", "oraclepass")); err != nil {
					t.Fatal(err)
				}
				renderedOutput(held)
				sendMenuInput(t, held, "")
				renderedOutput(held)
				sendMenuInput(t, held, "1")
				renderedOutput(held)
				if state == "source-linkdead" {
					held.player.SetLinkless(true)
					held.transportDone = make(chan struct{})
					held.DetachTransport()
				}
				if state == "source-case-only" {
					name = "returner"
				}
				if state == "source-live-renamed" {
					runString(t, wizard, "mob Returner name Alias")
					renderedOutput(wizard)
				}
			case "destination-online", "destination-case-only":
				entrySeed(t, database, "Target")
				held = makeCharSession(t, m)
				if err := held.handleLogin(loginMsg("Target", "oraclepass")); err != nil {
					t.Fatal(err)
				}
				renderedOutput(held)
				sendMenuInput(t, held, "")
				renderedOutput(held)
				sendMenuInput(t, held, "1")
				renderedOutput(held)
				name = "Target"
				if state == "destination-case-only" {
					name = "tArGeT"
				}
			case "destination-entering":
				held = makeCharSession(t, m)
				if err := held.handleLogin(loginMsg("Unused", "")); err != nil {
					t.Fatal(err)
				}
				renderedOutput(held)
			case "offline-with-guests":
				for _, guestName := range []string{"GuestOne", "GuestTwo"} {
					guest := makeCharSession(t, m)
					if err := guest.handleLogin(loginMsg(guestName, "")); err != nil {
						t.Fatal(err)
					}
					if !guest.isGuest {
						t.Fatal("actual guest entry did not occur")
					}
				}
				collision := false
				for _, p := range m.world.GetAllPlayers() {
					if p.GetID() == source.ID {
						collision = true
					}
				}
				if !collision {
					t.Fatal("fixture did not produce guest/store numeric overlap")
				}
			case "source-entering":
				held = makeCharSession(t, m)
				if err := held.handleLogin(loginMsg("Returner", "")); err != nil {
					t.Fatal(err)
				}
				renderedOutput(held)
			}
			before, err := database.GetPlayer("Returner")
			if err != nil {
				t.Fatal(err)
			}
			if err := cmdSet(wizard, []string{"file", "Returner", "name", name}); err != nil {
				t.Fatal(err)
			}
			got := renderedOutput(wizard)
			if state == "offline-success" || state == "offline-with-guests" {
				if got != "Okay.\r\nSaved in file.\r\n" {
					t.Fatal("allowed offline rename", got)
				}
				after, err := database.GetPlayer(name)
				if err != nil || after == nil || after.ID != source.ID {
					t.Fatal("allowed rename did not retain ID")
				}
				if rec, err := database.GetPlayer("Returner"); err != nil || rec != nil {
					t.Fatal("old name retained")
				}
				return
			}
			if got != "Sorry, you can't do that.\r\n" {
				t.Fatalf("guard did not refuse %s: %q", state, got)
			}
			after, err := database.GetPlayer("Returner")
			if err != nil || !reflect.DeepEqual(after, before) {
				t.Fatal("refused rename changed durable record", after, before, err)
			}
			if held == nil || held.SendClosed() {
				t.Fatal("refusal disconnected held descriptor")
			}
		})
	}
}

// Run the previously reachable topology-building commands; no manual map or
// ID edits. The guard must preserve both identities and normal takeover.
func TestEntryDuplicateRenameTopologyExclusions(t *testing.T) {
	for _, mode := range []string{"different-ID", "folded-key"} {
		t.Run(mode, func(t *testing.T) {
			database := entryDatabase(t)
			source := entrySeed(t, database, "Returner")
			other := entrySeed(t, database, "Other")
			m := entryWorldManager(t, database)
			old := makeCharSession(t, m)
			if err := old.handleLogin(loginMsg("Returner", "oraclepass")); err != nil {
				t.Fatal(err)
			}
			renderedOutput(old)
			sendMenuInput(t, old, "")
			renderedOutput(old)
			sendMenuInput(t, old, "1")
			renderedOutput(old)
			wizard := makeCommandTestSession(t, m, "Wizard", game.LVL_IMPL, 1001)
			commands := [][]string{{"file", "Returner", "name", "Archived"}, {"file", "Other", "name", "Returner"}}
			if mode == "folded-key" {
				commands = [][]string{{"file", "Returner", "name", "returner"}}
			}
			for _, args := range commands {
				if err := cmdSet(wizard, args); err != nil {
					t.Fatal(err)
				}
				renderedOutput(wizard)
			}
			rec, err := database.GetPlayer("returner")
			if err != nil || rec == nil || rec.ID != source.ID || rec.Name != "Returner" {
				t.Fatal("topology guard failed to preserve original stored identity", rec, err)
			}
			otherRec, err := database.GetPlayer("Other")
			if err != nil || otherRec == nil || otherRec.ID != other.ID {
				t.Fatal("guard lost other identity")
			}
			fresh := makeCharSession(t, m)
			if err := fresh.handleLogin(loginMsg("RETURNER", "oraclepass")); err != nil {
				t.Fatal(err)
			}
			if fresh.player != old.player || fresh.player.ID != source.ID || fresh.menuActive {
				t.Fatal("guard failed to preserve ordinary duplicate adoption")
			}
		})
	}
}

// Live name edits do not change the store's identity column: SavePlayer omits
// name, and ApplyCharacterData does not restore its redundant JSON name.
func TestEntryLiveRenameKeepsStoredIdentity(t *testing.T) {
	database := entryDatabase(t)
	source := entrySeed(t, database, "Returner")
	m := entryWorldManager(t, database)
	old := makeCharSession(t, m)
	old.transportDone = make(chan struct{})
	if err := old.handleLogin(loginMsg("Returner", "oraclepass")); err != nil {
		t.Fatal(err)
	}
	renderedOutput(old)
	sendMenuInput(t, old, "")
	renderedOutput(old)
	sendMenuInput(t, old, "1")
	renderedOutput(old)
	wizard := makeCommandTestSession(t, m, "Wizard", game.LVL_IMPL, 1001)
	if err := cmdSet(wizard, []string{"player", "Returner", "name", "returner"}); err != nil {
		t.Fatal(err)
	}
	if old.player.Name != "returner" {
		t.Fatal("live rename did not happen", renderedOutput(wizard))
	}
	if !m.HandleTransportDisconnect(old) {
		t.Fatal("EOF failed to retain playing body")
	}
	rec, err := database.GetPlayer("returner")
	if err != nil || rec == nil || rec.Name != "Returner" || rec.ID != source.ID {
		t.Fatal("live save changed indexed identity")
	}
	fresh := makeCharSession(t, m)
	if err := fresh.handleLogin(loginMsg("RETURNER", "oraclepass")); err != nil {
		t.Fatal(err)
	}
	if fresh.player != old.player || fresh.player.ID != source.ID || fresh.menuActive {
		t.Fatal("live edit created folded-key adoption miss")
	}
}
