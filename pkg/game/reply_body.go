package game

import (
	"fmt"
	"strings"
)

// CharacterReply ports do_reply for a mobile body (src/act.comm.c:934-973).
// C reads GET_LAST_TELL(ch) from ch->player_specials; every mobile shares the
// zeroed dummy_mob record (src/db.c:104,1281), so a switched NPC starts with
// no teller. Bare reply asks for text, and reply with text walks
// character_list for the first body whose GET_IDNUM equals the record. Every
// mobile carries idnum 0, so a zero record matches the newest mobile.
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

// SetNPCDummyLastTell writes C's shared dummy_mob last_tell. The port stores
// the teller's name because players carry no C idnum; a tell to any mobile
// therefore becomes the reply target of every mobile until overwritten.
func (w *World) SetNPCDummyLastTell(name string) {
	w.npcDummyMu.Lock()
	w.npcDummyLastTell = name
	w.npcDummyMu.Unlock()
}

// NPCDummyLastTell reads the shared dummy_mob last_tell ("" is C's 0).
func (w *World) NPCDummyLastTell() string {
	w.npcDummyMu.RLock()
	defer w.npcDummyMu.RUnlock()
	return w.npcDummyLastTell
}

// doMobReply mirrors do_reply's mobile arm in C's order (src/act.comm.c:934-
// 973). The shared record is 0 for mobiles, so the empty-argument gate comes
// first; a zero record then searches character_list for the newest mobile.
// A nonzero record (set by a tell to a mobile) searches for that player.
func (w *World) doMobReply(ch *MobInstance, argument string) {
	argument = strings.TrimLeft(argument, " \t\r\n\v\f")
	if argument == "" {
		ch.SendMessage("What is your reply?\r\n")
		return
	}
	last := w.NPCDummyLastTell()
	if last != "" {
		w.mobReplyToTeller(ch, last, argument)
		return
	}
	// A zero record matches the first idnum-0 body in C's newest-first
	// character_list, which is the newest mobile; if it has no descriptor C
	// prints the no-longer-playing line without looking further
	// (src/act.comm.c:952-958).
	mob := w.newestMobile()
	if mob == nil || w.MobileDescriptorOwner == nil || w.MobileDescriptorOwner(mob) == nil {
		ch.SendMessage("They are no longer playing.\r\n")
		return
	}
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
}

// newestMobile returns the first mobile in C's newest-first character_list.
// Every mobile carries idnum 0 and every player a nonzero one.
func (w *World) newestMobile() *MobInstance {
	for _, body := range w.GetAllCharsInCharacterListOrder() {
		if mob, ok := body.(*MobInstance); ok {
			return mob
		}
	}
	return nil
}

// mobReplyToTeller is do_reply's player-target arm for a mobile actor, in C's
// check order (src/act.comm.c:956-971). PLR_FLAGGED(ch) is NPC-guarded, so the
// actor's PLR_NOSHOUT is false; every other check applies to the player.
func (w *World) mobReplyToTeller(ch *MobInstance, name, argument string) {
	player := w.firstPlayerInCharacterList(name)
	if player == nil {
		ch.SendMessage("They are no longer playing.\r\n")
		return
	}
	if player.GetFlags()&(1<<PlrWriting) != 0 {
		ch.SendMessage("They are writing now, try later.\r\n")
		return
	}
	if player.GetFlags()&(1<<PrfNotell) != 0 {
		ch.SendMessage("They can't hear you.\r\n")
		return
	}
	if w.RoomHasFlag(ch.GetRoom(), "soundproof") {
		ch.SendMessage("The walls seem to absorb your words.\r\n")
		return
	}
	if w.PlayerHasDescriptor != nil && !w.PlayerHasDescriptor(player) {
		Act(nil, false, ch, player, nil, nil, "$E's linkless at the moment.", "", ToChar|ToSleep)
		return
	}
	if player.GetFlags()&(1<<PrfNotell) != 0 || w.RoomHasFlag(player.GetRoom(), "soundproof") {
		Act(nil, false, ch, player, nil, nil, "$E can't hear you.", "", ToChar|ToSleep)
		return
	}
	w.mobTellPlayer(ch, player, argument)
}

// firstPlayerInCharacterList walks C's newest-first list for the named player.
func (w *World) firstPlayerInCharacterList(name string) *Player {
	for _, body := range w.GetAllCharsInCharacterListOrder() {
		if player, ok := body.(*Player); ok && strings.EqualFold(player.GetName(), name) {
			return player
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
	w.SetNPCDummyLastTell("")
}

// mobTellPlayer is perform_tell from a mobile to a player. C writes
// GET_LAST_TELL(vict) = GET_IDNUM(mob) = 0, so the player's own record is
// cleared (the port's empty teller).
func (w *World) mobTellPlayer(ch *MobInstance, player *Player, message string) {
	message = deleteANSIControls(message)
	Act(w, false, ch, player, nil, nil, fmt.Sprintf("$n tells you, '%s'", message), "", ToVict|ToSleep)
	Act(w, false, ch, player, nil, nil, fmt.Sprintf("You tell $N, '%s'", message), "", ToChar|ToSleep)
	player.SetLastTeller("")
}

// playerTellMob is perform_tell from a player to a mobile (src/act.comm.c:873-
// 895). The mobile's descriptor owner sees the victim line; the player sees the
// echo; the shared dummy_mob record receives the teller's name (C idnum).
func (w *World) playerTellMob(ch Actor, mob *MobInstance, message string) {
	message = deleteANSIControls(message)
	Act(w, false, ch, mob, nil, nil, fmt.Sprintf("$n tells you, '%s'", message), "", ToVict|ToSleep)
	Act(w, false, ch, mob, nil, nil, fmt.Sprintf("You tell $N, '%s'", message), "", ToChar|ToSleep)
	w.SetNPCDummyLastTell(ch.GetName())
}
