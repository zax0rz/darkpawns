package game

// TestDeathCryBoundedCallers is the bounded caller proof for fight.c:506-516,
// the "death_cry() in fight.c called with ch->in_room = NOWHERE" diagnostic.
// Go never ported that arm because Go's callers cannot hand death_cry a
// NOWHERE room: the death-trap path is gated on a resolved ROOM_DEATH room
// (act_movement.go:427-430) and re-resolves the mount in the rider's room
// instead of reusing a pre-entry pointer the way C's do_move does
// (src/act.movement.c:195-202, :296-300). This test pins the second half: a
// rider whose mount has already left the world produces no mount cry at all,
// so there is no NOWHERE mount to cry for. Removing the re-resolution fails
// the assertion.
