package session

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func cmdQuit(s *Session) error {
	return execQuit(s, false)
}

func cmdReallyQuit(s *Session) error {
	return execQuit(s, true)
}

// cmdQuiStub mirrors C do_quit's subcmd guard (src/act.other.c:114-116): the
// table's "qui" entry (interpreter.c:629) forces mortals to type quit in full
// — the exact refusal is player-facing surface. C's guard is level-gated, so
// immortals typing "qui" fall through to a real quit.
func cmdQuiStub(s *Session) error {
	if getEffectiveLevel(s) < LVL_IMMORT {
		s.Send("You have to type quit--no less, to quit!")
		return nil
	}
	return cmdQuit(s)
}

// cmdShutdowStub mirrors C do_shutdown's subcmd guard (src/act.wizard.c:
// 1079-1080) for the "shutdow" table entry (interpreter.c:698). Reachable only
// at LVL_IMPL-1+ via command_gates.tsv; below that, resolution skips it (law 3)
// and exact typing is gate-rejected, matching C's scan behavior.
func cmdShutdowStub(s *Session) error {
	s.Send("If you want to shut something down, say so!")
	return nil
}

// execQuit is the session-teardown half of C do_quit (src/act.other.c:72-181).
// The game layer (World.DoQuit) owns the safe-room gate, the refuse messages,
// and the equipment kept-vs-lost fork; this half performs the descriptor-side
// logout sequence only when the game layer approves the logout. reallyQuit
// mirrors C's SCMD_REALLY_QUIT subcmd.
func execQuit(s *Session, reallyQuit bool) error {
	outcome := s.manager.world.DoQuit(s.player, reallyQuit)
	if outcome != game.QuitLogoutKeepEQ && outcome != game.QuitLogoutLoseEQ {
		return nil
	}

	room := s.player.GetRoom()
	// C: act("$n has left the game.", TRUE, ch, 0, 0, TO_ROOM) iff !GET_INVIS_LEV(ch).
	s.leaveBroadcastHandled = true
	if s.player.GetFlags()&game.PLR_INVISIBLE == 0 {
		msg, err := json.Marshal(ServerMessage{
			Type: MsgEvent,
			Data: EventData{
				Type: "leave",
				From: s.player.Name,
				Text: fmt.Sprintf("%s has left the game.", s.player.Name),
			},
		})
		if err != nil {
			slog.Error("json.Marshal error", "error", err)
		} else {
			s.manager.BroadcastToRoom(room, msg, s.player.Name)
		}
	}

	// C: InfoBarOff(ch) if the infobar is on.
	if s.infobarMode == InfobarOn {
		s.infobarMode = InfobarOff
	}

	// C: close_socket on every other descriptor sharing this character's
	// IDNUM (anti-dupe). Their saves run before ours so the quitting save —
	// with the equipment fork applied — is the final state.
	s.manager.closeDuplicateSessions(s)

	// Unregister → cleanupSession performs the final save and world removal.
	// Closing the send channel lets each transport flush the queued goodbye
	// before its writer closes the underlying connection.
	s.manager.Unregister(s.player.Name)

	return nil
}

// cmdInventory shows the player's inventory.
// Delegates to the game-layer DoInventory for C-faithful formatting.
func cmdInventory(s *Session, args []string) error {
	s.manager.world.DoInventory(s.player)
	return nil
}

// cmdEquipment shows the player's equipped items.
// Delegates to the game-layer DoEquipment for C-faithful formatting.
func cmdEquipment(s *Session, args []string) error {
	s.manager.world.DoEquipment(s.player)
	return nil
}

// cmdWear equips an item from inventory.
func cmdWear(s *Session, args []string) error {
	s.manager.world.DoWear(s.player, strings.Join(args, " "))
	s.markDirty(VarInventory, VarEquipment)
	return nil
}

// cmdRemove unequips an item.
func cmdRemove(s *Session, args []string) error {
	s.manager.world.DoRemove(s.player, strings.Join(args, " "))
	s.markDirty(VarInventory, VarEquipment)
	return nil
}

// cmdWield equips a weapon.
func cmdWield(s *Session, args []string) error {
	s.manager.world.DoWield(s.player, strings.Join(args, " "))
	s.markDirty(VarInventory, VarEquipment)
	return nil
}

// cmdHold holds an item.
func cmdHold(s *Session, args []string) error {
	s.manager.world.DoGrab(s.player, strings.Join(args, " "))
	s.markDirty(VarInventory, VarEquipment)
	return nil
}

// cmdGet picks up an item from the room, container, or corpse.
// Delegates to game-layer doGet which handles:
//
//	get <item>, get all, get all.<item>, get <item> <container>
func cmdGet(s *Session, args []string) error {
	arg := strings.Join(args, " ")
	s.manager.world.DoGet(s.player, arg)
	s.markDirty(VarInventory, VarRoomItems)
	return nil
}

// cmdGive gives an item or gold to another character.
// Delegates to game-layer doGive which handles:
//
//	give <item> <player>, give <N> coins <player>, give <N> <player>
func cmdGive(s *Session, args []string) error {
	arg := strings.Join(args, " ")
	s.manager.world.DoGive(s.player, arg)
	s.markDirty(VarInventory)
	return nil
}

// cmdPut puts an item into a container.
// Delegates to game-layer doPut which handles:
//
//	put <item> <container>, put all <container>, put all.<item> <container>
func cmdPut(s *Session, args []string) error {
	arg := strings.Join(args, " ")
	s.manager.world.DoPut(s.player, arg)
	s.markDirty(VarInventory, VarRoomItems)
	return nil
}

// cmdDrop drops an item from inventory.
func cmdDrop(s *Session, args []string) error {
	s.manager.world.DoDrop(s.player, strings.Join(args, " "))
	s.markDirty(VarInventory, VarRoomItems)
	return nil
}

// cmdJunk destroys carried item(s) or gold for a small experience reward.
// Delegates to game-layer DoJunk which handles:
//
//	junk <item>, junk all.<item>, junk <N> coins
func cmdJunk(s *Session, args []string) error {
	arg := strings.Join(args, " ")
	s.manager.world.DoJunk(s.player, arg)
	s.markDirty(VarInventory, VarRoomItems)
	return nil
}

// cmdDonate donates carried item(s) or gold to the donation room.
// Delegates to game-layer DoDonate which handles:
//
//	donate <item>, donate all.<item>, donate <N> coins
func cmdDonate(s *Session, args []string) error {
	arg := strings.Join(args, " ")
	s.manager.world.DoDonate(s.player, arg)
	s.markDirty(VarInventory, VarRoomItems)
	return nil
}

// cmdFollow sets the player to follow another player.
// Source: act.movement.c do_follow() lines 883–951
