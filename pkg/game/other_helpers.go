//nolint:unused // Game logic port — not yet wired to command registry.
package game

import (
	"strings"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// ---------------------------------------------------------------------------
// PLR flag bit positions (from structs.h PLR_* constants)
// ---------------------------------------------------------------------------

const (
	PlrOutlaw    = 0
	PlrNODELETE  = 13
	PlrCRYO      = 15
	PlrWerewolf  = 16
	PlrVampire   = 17
)

// ---------------------------------------------------------------------------
// PRF flag bit positions (from structs.h, shifted to avoid PLR collision)
// ---------------------------------------------------------------------------

const (
	PrfBrief      = 20
	PrfCompact    = 21
	PrfDeaf       = 22
	PrfNotell     = 23
	PrfDisphp     = 24
	PrfDispmmana  = 25
	PrfDispmove   = 26
	PrfAutoexit   = 27
	PrfNohassle   = 28
	PrfHolyLight  = 29
	PrfNoRepeat   = 30
	PrfColor1     = 31
	PrfColor2     = 32
	PrfNowiz      = 33
	PrfLog1       = 34
	PrfLog2       = 35
	PrfNoAuctions = 36
	PrfNoGossip   = 37
	PrfNoGratz    = 38
	PrfRoomFlags  = 39
	PrfAFK        = 40
	PrfAutoLoot   = 41
	PrfAutoGold   = 42
	PrfAutoSplit  = 43
	PrfDispTank   = 44
	PrfDispTarget = 45
	PrfNoNewbie   = 46
	PrfInactive   = 47
	PrfSummonable = 48
	PrfQuest      = 49
	PrfNoCTell    = 50
	PrfNoBroad    = 51
)

// ---------------------------------------------------------------------------
// AFF flag bits used in this file (other bits defined in act_movement.go /
// act_offensive.go)
// ---------------------------------------------------------------------------

const (
	affInvisible = 2
	affWerewolf  = 32
	affVampire   = 33
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// isPlayerNPC returns true if the character is a mob (me != nil).
func isPlayerNPC(ch *Player, me *MobInstance) bool {
	return me != nil
}

// actToRoom broadcasts a formatted message to all players in the given room,
// optionally excluding one player by name.
func actToRoom(w *World, roomVNum int, format string, excludeName string) {
	players := w.GetPlayersInRoom(roomVNum)
	for _, p := range players {
		if p.IsNPC() {
			continue
		}
		if excludeName != "" && p.Name == excludeName {
			continue
		}
		p.SendMessage(format)
	}
}

// getPlayerByName finds a player by name in a slice.
func getPlayerByName(players []*Player, name string) *Player {
	for _, p := range players {
		if strings.EqualFold(p.Name, name) {
			return p
		}
	}
	return nil
}

// strCompare returns true if strings differ case-insensitively (matching C str_cmp).
func strCompare(a, b string) bool {
	return !strings.EqualFold(a, b)
}

// hasRoomFlag checks if a room has the named flag (e.g. "indoors", "death", "tunnel").
func hasRoomFlag(room *parser.Room, flag string) bool {
	for _, f := range room.Flags {
		if strings.EqualFold(f, flag) {
			return true
		}
	}
	return false
}

// isDark returns true if the room is dark (no light source, not outdoors with sun).
func isDark(room *parser.Room) bool {
	return hasRoomFlag(room, "dark")
}

// isOutdoors returns true if the room is outdoors.
func isOutdoors(room *parser.Room) bool {
	return !hasRoomFlag(room, "indoors")
}

// getMount finds the mount mob for a rider.
func getMount(ch *Player, w *World) *MobInstance {
	if !ch.IsAffected(affMounted) {
		return nil
	}
	mobs := w.GetMobsInRoom(ch.GetRoomVNum())
	for _, m := range mobs {
		if m.HasFlag("mount") {
			// Check if this mob is mounted with this player as rider
			// We check if the mob is also marked as mounted
			return m
		}
	}
	return nil
}
