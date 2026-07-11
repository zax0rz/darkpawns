package dreaming

import "testing"

// TestMakeEventID_DistinctAtSameNanosecond verifies that two events of the
// same kind for the same agent that arrive within the same nanosecond get
// distinct IDs. Before the index suffix was added, the IDs collided and
// AddOrReinforceNode silently merged the two events, dropping one.
func TestMakeEventID_DistinctAtSameNanosecond(t *testing.T) {
	const agentID = "brenda"
	const kind = "combat"
	const nanoTs = 1752200000000000000

	id0 := makeEventID(agentID, kind, nanoTs, 0)
	id1 := makeEventID(agentID, kind, nanoTs, 1)

	if id0 == id1 {
		t.Fatalf("events at the same nanosecond got the same ID: %q", id0)
	}
}
