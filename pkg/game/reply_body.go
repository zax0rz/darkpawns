package game

import (
	"fmt"
	"strings"
)

// CharacterReply ports do_reply for a mobile body (src/act.comm.c:934-973).
// C reads GET_LAST_TELL(ch) from ch->player_specials; every mobile shares the
// zeroed dummy_mob record (src/db.c:104,1281), so a switched NPC starts at 0,
// which is not NOBODY (-1). Bare reply therefore asks for text, and reply
// with text walks character_list for GET_IDNUM(tch) == 0. Every mobile
// carries idnum 0, so the match is the newest mobile; if it has no descriptor
// C prints the no-longer-playing line without looking further.
func CharacterReply(ctx CharacterCommandContext, _ string, args []string) CharacterCommandResult {
	body, ok := ctx.Actor.(*MobInstance)
	if !ok || ctx.World == nil {
		return CharacterCommandUnsupported
	}
	argument := ctx.RawArgs
	if argument == "" && len(args) > 0 {
		argument = strings.Join(args, " ")
	}
	ctx.World.doMobReply(body, argument)
	return CharacterCommandHandled
}

// doMobReply mirrors do_reply's mobile arm in C's order (src/act.comm.c:934-
// 973). The shared dummy_mob record is zeroed, so a switched NPC's last_tell
// is 0 (not NOBODY): bare reply asks for text, and reply with text walks
// character_list for the first body whose GET_IDNUM equals that value. Every
// mobile carries idnum 0, so the match is the newest mobile in C's
// newest-first list; if that mobile has no descriptor C prints the
// no-longer-playing line without looking further.
func (w *World) doMobReply(ch *MobInstance, argument string) {
	argument = strings.TrimLeft(argument, " \t\r\n\v\f")
	last := w.npcDummyLastTell.Load()
	if last == -1 { // NOBODY (src/structs.h:36); only a player record uses it
		ch.SendMessage("You have no-one to reply to!\r\n")
		return
	}
	if argument == "" {
		ch.SendMessage("What is your reply?\r\n")
		return
	}
	target := w.firstCharacterWithIDNum(last)
	mob, isMob := target.(*MobInstance)
	if target == nil || (isMob && (w.MobileDescriptorOwner == nil || w.MobileDescriptorOwner(mob) == nil)) {
		ch.SendMessage("They are no longer playing.\r\n")
		return
	}
	if isMob {
		// PLR_FLAGGED(tch, PLR_WRITING) and PLR_FLAGGED(ch, PLR_NOSHOUT) are
		// NPC-guarded (src/utils.h:234), so both are false for a mobile.
		// PRF_FLAGGED(tch, PRF_NOTELL) is not guarded (src/utils.h:236) and
		// reads the shared zeroed dummy_mob preferences, so it is false too.
		if w.RoomHasFlag(ch.GetRoom(), "soundproof") {
			ch.SendMessage("The walls seem to absorb your words.\r\n")
			return
		}
		if w.RoomHasFlag(mob.GetRoom(), "soundproof") {
			Act(w, false, ch, mob, nil, nil, "$E can't hear you.", "", ToChar|ToSleep)
			return
		}
		w.mobPerformTell(ch, mob, argument)
		return
	}
	// A nonzero shared value can only come from a tell-to-mobile path, which is
	// not ported; without C player-idnum parity, keep C's miss.
	ch.SendMessage("They are no longer playing.\r\n")
}

// firstCharacterWithIDNum walks C's newest-first character_list. Only mobiles
// carry idnum 0 in the port's shared-record path; players have nonzero ids.
func (w *World) firstCharacterWithIDNum(idnum int64) any {
	for _, body := range w.GetAllCharsInCharacterListOrder() {
		if _, ok := body.(*MobInstance); ok && idnum == 0 {
			return body
		}
	}
	return nil
}

// mobPerformTell is perform_tell for a mobile source and victim
// (src/act.comm.c:873-895): victim line, actor echo, then
// GET_LAST_TELL(vict) = GET_IDNUM(ch), which is 0 for every mobile.
func (w *World) mobPerformTell(ch, vict *MobInstance, message string) {
	message = deleteANSIControls(message)
	Act(w, false, ch, vict, nil, nil, fmt.Sprintf("$n tells you, '%s'", message), "", ToVict|ToSleep)
	Act(w, false, ch, vict, nil, nil, fmt.Sprintf("You tell $N, '%s'", message), "", ToChar|ToSleep)
	w.npcDummyLastTell.Store(0)
}
