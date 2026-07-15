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
// Shout bridge
// ---------------------------------------------------------------------------

// ExecShout executes the shout command.
func (w *World) ExecShout(ch *Player, arg string) { w.doShout(ch, nil, arg) }

// ---------------------------------------------------------------------------
// QComm bridge
// ---------------------------------------------------------------------------

// ExecQcomm executes the team/quiz communication command.
func (w *World) ExecQcomm(ch *Player, arg string) { w.doQcomm(ch, nil, "qcomm", arg) }

// ---------------------------------------------------------------------------
// Think bridge
// ---------------------------------------------------------------------------

// ExecThink executes the think command.
func (w *World) ExecThink(ch *Player, arg string) { w.doThink(ch, nil, "think", arg) }

// ---------------------------------------------------------------------------
// GenComm bridge (gossip, chat, auction, gratz, newbie)
// ---------------------------------------------------------------------------

// ExecGenComm executes a generic channel command (gossip, chat, auction, etc.).
func (w *World) ExecGenComm(ch *Player, cmd, arg string) { w.doGenComm(ch, nil, cmd, arg) }

// ---------------------------------------------------------------------------
// CTell bridge (clan tell)
// ---------------------------------------------------------------------------

// ExecCTell executes the clan tell command.
func (w *World) ExecCTell(ch *Player, arg string) { w.doCTell(ch, nil, "ctell", arg) }
