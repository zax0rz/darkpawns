package combat

import (
	"strings"
	"testing"
)

// testCombatBody supplies standalone message fixtures. Production never uses it.
func testCombatBody(name string) Combatant { return &mockCombatant{name: name} }

func testBodyNames(bodies []Combatant) string {
	var names []string
	for _, b := range bodies {
		names = append(names, b.GetName())
	}
	return strings.Join(names, " ")
}

func TestCombatBodyDefenseAudience(t *testing.T) {
	fighter := &msgMockCombatant{mockCombatant: mockCombatant{name: "Guard", position: PosFighting}}
	opponent := &msgMockCombatant{mockCombatant: mockCombatant{name: "Guard", position: PosFighting}}
	peer := &msgMockCombatant{mockCombatant: mockCombatant{name: "Guard", position: PosStanding}}
	sleeper := &msgMockCombatant{mockCombatant: mockCombatant{name: "Sleeper", position: PosSleeping}}
	old := GetCallbacks()
	t.Cleanup(func() { SetCallbacks(old) })
	SetCallbacks(&GameCallbacks{GetRoomCombatants: func(int) []Combatant { return []Combatant{fighter, opponent, peer, sleeper} }})
	NewCombatEngine().sendDefenseObserverMessage(fighter, opponent, "defense")
	if len(fighter.messages) != 0 || len(opponent.messages) != 0 || len(sleeper.messages) != 0 || len(peer.messages) != 1 {
		t.Fatalf("audience=%v/%v/%v/%v", fighter.messages, opponent.messages, peer.messages, sleeper.messages)
	}
}

func TestCombatBodyDirectText(t *testing.T) {
	one, two := testCombatBody("Guard"), testCombatBody("Guard")
	old := GetCallbacks()
	t.Cleanup(func() { SetCallbacks(old) })
	var recipient Combatant
	var text string
	SetCallbacks(&GameCallbacks{SendText: func(body Combatant, msg string) { recipient = body; text = msg }})
	sendCombatMessage(two, "exact\r\n")
	if recipient != two || recipient == one || text != "exact\r\n" {
		t.Fatalf("recipient=%v text=%q", recipient, text)
	}
}
