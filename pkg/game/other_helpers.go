package game

import "github.com/zax0rz/darkpawns/pkg/parser"

// ---------------------------------------------------------------------------
// PLR flag bit positions (from structs.h PLR_* constants)
// ---------------------------------------------------------------------------

const (
	PlrOutlaw   = 0
	PlrNODELETE = 13
	PlrCRYO     = 15
	PlrWerewolf = 16
	PlrVampire  = 17
)

// ---------------------------------------------------------------------------
// PRF flag bit positions: C's PRF_* number (structs.h) plus prfBase.
//
// C keeps PLR flags (char_specials.saved.act) and PRF flags (player_specials
// .saved.pref) in two separate arrays; the port keeps both in Player.Flags.
// PLR bits run 0..21 (PLR_EXTRACT) plus the mob-program "calibrate" bit 22,
// and PRF bits run 0..31, so PRF lives in the upper 32 bits. An earlier base
// of 20 put PRF_BRIEF on PLR_REMORT and PRF_COMPACT on PLR_EXTRACT.
// ---------------------------------------------------------------------------

// prfBase is where PRF bit 0 sits in Player.Flags.
const prfBase = 32

const (
	PrfBrief      = prfBase + 0
	PrfCompact    = prfBase + 1
	PrfDeaf       = prfBase + 2
	PrfNotell     = prfBase + 3
	PrfDisphp     = prfBase + 4
	PrfDispmmana  = prfBase + 5
	PrfDispmove   = prfBase + 6
	PrfAutoexit   = prfBase + 7
	PrfNohassle   = prfBase + 8
	PrfQuest      = prfBase + 9
	PrfSummonable = prfBase + 10
	PrfNoRepeat   = prfBase + 11
	PrfHolyLight  = prfBase + 12
	PrfColor1     = prfBase + 13
	PrfColor2     = prfBase + 14
	PrfNowiz      = prfBase + 15
	PrfLog1       = prfBase + 16
	PrfLog2       = prfBase + 17
	PrfNoAuctions = prfBase + 18
	PrfNoGossip   = prfBase + 19
	PrfNoGratz    = prfBase + 20
	PrfRoomFlags  = prfBase + 21
	PrfAFK        = prfBase + 22
	PrfAutoLoot   = prfBase + 23
	PrfAutoGold   = prfBase + 24
	PrfAutoSplit  = prfBase + 25
	PrfDispTank   = prfBase + 26
	PrfDispTarget = prfBase + 27
	PrfNoNewbie   = prfBase + 28
	PrfInactive   = prfBase + 29
	PrfNoCTell    = prfBase + 30
	PrfNoBroad    = prfBase + 31
)

// ---------------------------------------------------------------------------
// AFF flag bits used in this file (other bits defined in act_movement.go /
// act_offensive.go)
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// isPlayerNPC returns true if the character is a mob (me != nil).
func isPlayerNPC(ch *Player, me *MobInstance) bool {
	return me != nil
}

// actToRoom is a legacy preformatted-message adapter to canonical Act.
func actToRoom(w *World, roomVNum int, format string, excludeName string) {
	format = legacyActFormat(format)
	if excludeName != "" {
		for _, player := range w.GetPlayersInRoom(roomVNum) {
			if player.Name == excludeName {
				Act(w, false, player, nil, nil, nil, format, "", ToRoom)
				return
			}
		}
	}
	anchor := &ObjectInstance{RoomVNum: roomVNum}
	Act(w, false, nil, nil, anchor, nil, format, "", ToRoom)
}

// hasRoomFlag checks if a room has the named flag (e.g. "indoors", "death", "tunnel").
func hasRoomFlag(room *parser.Room, flag string) bool {
	return roomHasNamedFlag(room, flag)
}

// isOutdoors returns true if the room is outdoors.
func isOutdoors(room *parser.Room) bool {
	return !hasRoomFlag(room, "indoors")
}
