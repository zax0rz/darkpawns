// Package game — act_comm_bridge.go: Exported wrappers for player communication
// command functions in act_comm.go, following the bridge pattern established by
// act_other_bridge.go.
//
// Each exported ExecXxx method delegates to the corresponding unexported doXxx,
// passing nil for the mob instance.
package game

// ---------------------------------------------------------------------------
// Race-Say bridge
// ---------------------------------------------------------------------------

// ExecRaceSay executes the race-specific language say command.
func (w *World) ExecRaceSay(ch *Player, arg string) { w.doRaceSay(ch, nil, "race_say", arg) }

// ---------------------------------------------------------------------------
// CTell bridge (clan tell)
// ---------------------------------------------------------------------------

// ExecCTell executes the clan tell command.
func (w *World) ExecCTell(ch *Player, arg string) { w.doCTell(ch, nil, "ctell", arg) }
