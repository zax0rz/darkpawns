package game

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"unicode"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/scripting"
)

// NPCCommand is command_interpreter (interpreter.c:883-949) run by a
// mobile, as lua_action does for a script's action(me, "...") (scripts.c:
// 122-140). A mobile has no descriptor, so every refusal C would send it
// ("Huh?!?", the position messages, a command's own error lines) reaches no
// one; what matters is which commands run and what they show the room.
//
// The session-installed CharacterCommandExecFunc sends this real body through
// the shared command interpreter and its ordered special() stages. When that
// executor is absent, this legacy fallback retains the previously ported
// say, tell, emote, give, stand and social behaviors. It does not claim generic
// command parity.
func (w *World) NPCCommand(me *MobInstance, line string) {
	if me == nil {
		return
	}
	if w.CharacterCommandExecFunc != nil {
		if result := w.CharacterCommandExecFunc(me, line); result == CharacterCommandUnsupported {
			slog.Warn("NPC command dispatch unsupported", "mob_vnum", me.GetVNum(), "line", line)
		}
		return
	}
	// if (!number(0,3)) REMOVE_BIT_AR(AFF_FLAGS(ch), AFF_HIDE): drawn for
	// every command, before anything else (R3).
	// #nosec G404 — game RNG, not cryptographic
	if dprng.Number(0, 3) == 0 {
		me.RemoveAffected(affHide)
	}
	line = strings.TrimLeft(line, " ")
	if line == "" {
		return
	}
	var word, rest string
	if c := line[0]; !isASCIIAlpha(c) { // isalpha(*line)
		word, rest = line[:1], line[1:]
	} else {
		word, rest = anyOneArg(line)
	}
	word = strings.ToLower(word)

	position := me.GetPosition()
	switch word {
	case "say", "'":
		if position >= combat.PosResting {
			w.npcSay(me, rest)
		}
	case "tell":
		if position >= combat.PosMortally {
			w.npcTell(me, rest)
		}
	case "emote", ":":
		if position >= combat.PosResting && me.GetLevel() >= 1 {
			w.npcEmote(me, rest)
		}
	case "gossip":
		w.mobGenComm(me, strings.TrimPrefix(rest, " "), "gossip")
	case "shout":
		w.mobGenComm(me, strings.TrimPrefix(rest, " "), "shout")
	case "auction":
		w.mobGenComm(me, strings.TrimPrefix(rest, " "), "auction")
	case "give":
		if position >= combat.PosResting {
			w.npcGive(me, rest)
		}
	case "stand":
		if position >= combat.PosResting {
			w.npcStand(me)
		}
	default:
		minimum, isSocial := socialMinPosition[word]
		if !isSocial {
			slog.Warn("NPC command not ported for mobiles; ignored", "mob_vnum", me.GetVNum(), "line", line)
			return
		}
		if position >= minimum {
			w.npcSocial(me, word, rest)
		}
	}
}

// NPCCommandDirect ports a direct C ACMD call such as
// do_tell(mobile, argument, find_command("tell"), 0). The caller supplies the
// known command body separately from its argument, so the argument reaches the
// command parser unchanged. This skips command_interpreter's hide draw,
// position/level/wait gates, and special() pass, all of which C skips for a
// direct ACMD call. Callers in C commonly ignore the ACMD result, so they may
// ignore the returned status too.
func (w *World) NPCCommandDirect(me *MobInstance, commandName, argument string) CharacterCommandResult {
	if w == nil || me == nil {
		return CharacterCommandUnsupported
	}
	commandName = strings.ToLower(strings.TrimSpace(commandName))
	if w.CharacterCommandDirectExecFunc != nil {
		if result := w.CharacterCommandDirectExecFunc(me, commandName, argument); result == CharacterCommandHandled {
			return result
		}
	}
	return w.npcCommandDirectFallback(me, commandName, argument)
}

// npcCommandDirectFallback supplies the same actor command bodies when the
// session-owned registry is not installed (focused game tests and standalone
// worlds). It deliberately has no interpreter prelude.
func (w *World) npcCommandDirectFallback(me *MobInstance, commandName, argument string) CharacterCommandResult {
	switch commandName {
	case "say", "'":
		w.NPCSayCommand(me, argument)
	case "tell":
		w.NPCTellCommand(me, argument)
	case "emote", ":":
		w.NPCEmoteCommand(me, argument)
	case "echo":
		w.npcEcho(me, argument)
	case "gecho":
		w.npcGlobalEcho(me, argument)
	case "gossip":
		w.NPCGossipCommand(me, argument)
	case "shout":
		w.NPCShoutCommand(me, argument)
	case "auction":
		w.NPCAuctionCommand(me, argument)
	case "stand":
		w.doPositionCommand(me, positionCommandSpecs[0], nil)
	case "sit":
		w.doPositionCommand(me, positionCommandSpecs[1], nil)
	case "rest":
		w.doPositionCommand(me, positionCommandSpecs[2], nil)
	case "sleep":
		w.doPositionCommand(me, positionCommandSpecs[3], nil)
	case "wake":
		wakeBody(w, me, argument, nil)
	case "flee":
		w.DoFlee(me)
	case "follow":
		arg := extractArg(argument)
		if target, found := w.resolveCharInRoomAt(me, me.GetRoomVNum(), arg); found && target.Combatant != nil {
			followMobBody(w, me, target.Combatant)
		}
	case "give":
		w.NPCGiveCommand(me, argument)
	case "get":
		return w.npcGetDirect(me, argument)
	case "wear":
		return w.npcWearDirect(me, argument)
	case "rescue":
		return w.npcRescueDirect(me, argument)
	case "assist":
		w.mobAssist(me, argument)
	case "look":
		if strings.TrimSpace(argument) != "" {
			return CharacterCommandUnsupported
		}
		// Direct do_look(mobile, "", ...) has only descriptor output. The
		// start_room special runs it after moving a descriptor-less mobile.
	case "behead":
		return w.npcBeheadDirect(me, argument)
	case "ride":
		w.doRideMob(me, extractArg(argument))
	case "dismount":
		w.doDismountActor(me)
	case "headbutt", "parry", "bash", "berserk", "charge", "disarm", "backstab":
		return w.npcFighterCommand(me, commandName, argument)
	default:
		if _, social := socialMinPosition[commandName]; social {
			w.NPCSocialCommand(me, commandName, argument)
		} else {
			return CharacterCommandUnsupported
		}
	}
	return CharacterCommandHandled
}

// NPCSayCommand, NPCTellCommand, NPCEmoteCommand, NPCGiveCommand, and
// NPCSocialCommand adapt the existing mobile behavior to the shared command
// registry. Each is registered against its C command entry by pkg/session.
func (w *World) NPCSayCommand(me *MobInstance, arg string)         { w.npcSay(me, arg) }
func (w *World) NPCTellCommand(me *MobInstance, arg string)        { w.npcTell(me, arg) }
func (w *World) NPCEmoteCommand(me *MobInstance, arg string)       { w.npcEmote(me, arg) }
func (w *World) NPCGiveCommand(me *MobInstance, arg string)        { w.npcGive(me, arg) }
func (w *World) NPCSocialCommand(me *MobInstance, cmd, arg string) { w.npcSocial(me, cmd, arg) }

// NPCGossipCommand, NPCShoutCommand, and NPCAuctionCommand adapt the NPC
// do_gen_comm delivery (src/act.comm.c:1146-1298) to the shared command
// registry.
func (w *World) NPCGossipCommand(me *MobInstance, arg string)  { w.mobGenComm(me, arg, "gossip") }
func (w *World) NPCShoutCommand(me *MobInstance, arg string)   { w.mobGenComm(me, arg, "shout") }
func (w *World) NPCAuctionCommand(me *MobInstance, arg string) { w.mobGenComm(me, arg, "auction") }

func (w *World) npcEcho(me *MobInstance, argument string) {
	text := strings.TrimLeft(argument, " \t\r\n\v\f")
	if text != "" {
		Act(w, false, me, nil, nil, nil, text, "", ToRoom)
	}
}

func (w *World) npcGlobalEcho(me *MobInstance, argument string) {
	text := strings.TrimLeft(argument, " \t\r\n\v\f")
	if text == "" {
		return
	}
	line := text + "\r\n"
	for _, player := range w.playingPlayers() {
		player.SendMessage(line)
	}
}

func isASCIIAlpha(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// anyOneArg is any_one_arg: the first space-delimited word, and the rest of
// the line after it (not lowercased, fill words kept).
func anyOneArg(line string) (string, string) {
	line = strings.TrimLeft(line, " ")
	end := strings.IndexByte(line, ' ')
	if end < 0 {
		return line, ""
	}
	return line[:end], line[end:]
}

// npcStand is do_stand (act.movement.c:696-731) for a mobile: only the
// room's lines are visible. A mounted mobile's dismount is not ported.
func (w *World) npcStand(me *MobInstance) {
	switch me.GetPosition() {
	case combat.PosStanding, combat.PosSleeping, combat.PosFighting:
		return // TO_CHAR only (a standing mobile's dismount is not ported)
	case combat.PosSitting:
		Act(w, true, me, nil, nil, nil, "$n clambers to $s feet.", "", ToRoom)
	case combat.PosResting:
		Act(w, true, me, nil, nil, nil, "$n stops resting, and clambers on $s feet.", "", ToRoom)
	default:
		Act(w, true, me, nil, nil, nil, "$n stops floating around, and puts $s feet on the ground.", "", ToRoom)
	}
	me.SetPosition(combat.PosStanding)
}

// npcSay is do_say (act.comm.c) for a mobile: the room hears
// "$n says, '...'" (or exclaims/asks/states by the line's last character).
func (w *World) npcSay(me *MobInstance, argument string) {
	text := strings.TrimLeft(argument, " ")
	if text == "" {
		return // "Yes, but WHAT do you want to say?" to a descriptor-less mobile
	}
	verb := "says"
	switch argument[len(argument)-1] {
	case '!':
		verb = "exclaims"
	case '?':
		verb = "asks"
	case '.':
		verb = "states"
	}
	w.channelAct("say", false, me, nil, deleteANSIControls(fmt.Sprintf("$n %s, '%s'", verb, text)), ToRoom)
}

// npcEmote is do_echo with SCMD_EMOTE (act.wizard.c:130-155) for a mobile.
func (w *World) npcEmote(me *MobInstance, argument string) {
	text := strings.TrimLeft(argument, " ")
	if text == "" {
		return
	}
	Act(w, false, me, nil, nil, nil, "$n "+text, "", ToRoom)
}

// npcTell is do_tell (act.comm.c) for a mobile: half_chop, get_char_vis,
// C's delivery gates, then perform_tell. A mobile's GET_IDNUM is not a
// player's, so the recipient's reply target is not set to it.
func (w *World) npcTell(me *MobInstance, argument string) {
	targetName, message := splitMobileTellArgument(argument)
	if targetName == "" || message == "" {
		return
	}
	target := w.npcVisiblePlayer(me, targetName)
	if target == nil {
		return // NOPERSON; a mobile target has no descriptor either
	}
	if w.communicationRoomSoundproof(me.GetRoom()) {
		return
	}
	if target.IsLinkless() {
		return
	}
	if target.GetFlags()&(1<<uint(PlrWriting)) != 0 {
		return
	}
	if (target.GetFlags()&(1<<uint(PrfNotell)) != 0 || w.communicationRoomSoundproof(target.GetRoom())) &&
		me.GetLevel() < lvlImmort {
		return
	}
	// The player path's own perform_tell carries the color sandwich, the AFK
	// notice (TO_CHAR, so a mobile's is dropped) and the reply target.
	w.performTell(me, target, message)
}

// splitMobileTellArgument is half_chop (interpreter.c:1372-1380): lowercase
// the first token, skip whitespace before the message, and preserve the
// message's trailing bytes for perform_tell.
func splitMobileTellArgument(argument string) (string, string) {
	argument = strings.TrimLeftFunc(argument, unicode.IsSpace)
	end := strings.IndexFunc(argument, unicode.IsSpace)
	if end < 0 {
		return strings.ToLower(argument), ""
	}
	return strings.ToLower(argument[:end]), strings.TrimLeftFunc(argument[end:], unicode.IsSpace)
}

// npcVisiblePlayer is get_char_vis for a mobile looking for a player to
// tell: the room first, then the world, visible to the mobile.
func (w *World) npcVisiblePlayer(me *MobInstance, name string) *Player {
	// C get_char_vis() starts with get_char_room_vis(), which scans the
	// combined room.people chain newest-first and lets the first visible
	// abbreviation match win (handler.c:1276-1325). A matching mobile is
	// still the winner; do_tell then rejects it when it has no descriptor.
	if target, found := w.resolveCharInRoomAt(me, me.GetRoom(), name); found {
		return target.Player
	}
	players := w.GetAllPlayers()
	sort.Slice(players, func(i, j int) bool { return players[i].GetName() < players[j].GetName() })
	for _, p := range players {
		if isnameWithAbbrevs(name, p.GetName()) && canSee(me, p) {
			return p
		}
	}
	return nil
}

// npcGive is do_give's object branch (act.item.c:767-830) for a mobile:
// give_find_vict, get_obj_in_list_vis over what the mobile carries, and
// perform_give. Coins and "all" are not used by C's scripts and are not
// ported for mobiles.
func (w *World) npcGive(me *MobInstance, argument string) {
	objName, rest := oneArgument(argument)
	if objName == "" || isNumber(objName) || strings.HasPrefix(strings.ToLower(objName), "all") {
		if objName != "" {
			slog.Warn("NPC give form not ported for mobiles; ignored", "mob_vnum", me.GetVNum(), "argument", argument)
		}
		return
	}
	victName, _ := oneArgument(rest)
	if victName == "" {
		return
	}
	vict := mobRescueVictim(w, me, victName)
	if vict == nil || vict.GetName() == me.GetName() {
		return
	}
	obj := w.npcCarriedVis(me, objName)
	if obj == nil {
		return
	}
	w.npcPerformGive(me, vict, obj)
}

// npcCarriedVis is get_obj_in_list_vis(me, arg, me->carrying).
func (w *World) npcCarriedVis(me *MobInstance, arg string) *ObjectInstance {
	name := strings.TrimSpace(arg)
	number := GetNumber(&name)
	if number <= 0 || name == "" {
		return nil
	}
	carried := me.InventorySnapshot()
	found := 0
	for _, obj := range carried {
		if !isnameWithAbbrevs(name, obj.GetKeywords()) {
			continue
		}
		if !canSeeObject(me, obj) && obj.GetTypeFlag() != ITEM_LIGHT {
			continue
		}
		found++
		if found == number {
			return obj
		}
	}
	return nil
}

// npcPerformGive is perform_give (act.item.c) with a mobile giver. Its
// refusals ("You can't let go of $p!!", "$N seems to have $S hands full.",
// "$E can't carry that much weight.") are TO_CHAR lines to the mobile.
func (w *World) npcPerformGive(me *MobInstance, vict interface{}, obj *ObjectInstance) {
	if obj.HasExtraFlag(0, extraFlagNoDrop) && me.GetLevel() < lvlImmort {
		return
	}
	switch v := vict.(type) {
	case *Player:
		if v.Inventory.IsFull() {
			return
		}
		if obj.GetWeight()+v.Inventory.GetWeight() > v.Inventory.GetCapacity()*10 {
			return
		}
		if err := w.MoveObjectToPlayerInventory(obj, v); err != nil {
			slog.Error("NPC give to player failed", "mob_vnum", me.GetVNum(), "obj_vnum", obj.VNum, "error", err)
			return
		}
		// MoveObject's attach is obj_to_char and flags the player recipient's
		// crash save there (src/handler.c:569-571; movement.go).
		Act(nil, false, me, v, obj, nil, "$n gives you $p.", "", ToVict)
		Act(w, true, me, v, obj, nil, "$n gives $p to $N.", "", ToNotVict)
	case *MobInstance:
		if !v.HasFlag("OKGIVE") && me.GetLevel() < lvlImmort {
			return
		}
		if obj.GetWeight()+mobCarriedWeight(v) > mobMaxCarryWeight(v) {
			return
		}
		if err := w.MoveObjectToMobInventoryFront(obj, v); err != nil {
			slog.Error("NPC give to mobile failed", "mob_vnum", me.GetVNum(), "obj_vnum", obj.VNum, "error", err)
			return
		}
		Act(w, true, me, v, obj, nil, "$n gives $p to $N.", "", ToNotVict)
		// C perform_give (src/act.item.c:700-701) runs mp_give for the NPC
		// victim here, before the ongive script.
		w.MpGiveObject(me, v, obj)
		// perform_give runs the receiving mobile's ongive with the giver as ch.
		if ScriptEngine != nil && v.HasScript("ongive") {
			ctx := v.CreateScriptContext(nil, obj, "")
			ctx.ChRef = &scripting.CharRef{NPC: true, ID: me.GetID()}
			ctx.RoomVNum = me.GetRoom()
			if _, err := v.RunScript("ongive", ctx); err != nil {
				slog.Warn("ongive script error", "mob_vnum", v.GetVNum(), "error", err)
			}
		}
	}
}

// npcSocial is do_action (act.social.c) for a mobile.
func (w *World) npcSocial(me *MobInstance, cmd, argument string) {
	social, ok := Socials[cmd]
	if !ok {
		return
	}
	if _, ok := socialMessage(social, socCharFound); !ok {
		if message, ok := socialMessage(social, socOthersNoArg); ok {
			Act(w, social.hidesInvisibleActor(), me, nil, nil, nil, message, "", ToRoom)
		}
		return
	}
	targetName := extractArg(argument)
	if targetName == "" {
		if message, ok := socialMessage(social, socOthersNoArg); ok {
			Act(w, social.hidesInvisibleActor(), me, nil, nil, nil, message, "", ToRoom)
		}
		return
	}
	// C do_action uses get_char_room_vis(ch, arg), which searches visible
	// players and mobiles in char_to_room order. A bare prefix can therefore
	// resolve to a newer character whose name starts with the command actor's
	// name.
	target, ok := w.resolveCharInRoomAt(me, me.GetRoomVNum(), targetName)
	if !ok {
		return // action->not_found to the mobile
	}
	targetActor, _ := target.Combatant.(Actor)
	if target.Combatant == me {
		if message, ok := socialMessage(social, socOthersAuto); ok {
			Act(w, social.hidesInvisibleActor(), me, nil, nil, nil, message, "", ToRoom)
		}
		return
	}
	if minimum := social.minimumVictimPosition(); minimum > 0 && targetActor.GetPosition() < minimum {
		return
	}
	if message, ok := socialMessage(social, socOthersFound); ok {
		Act(w, social.hidesInvisibleActor(), me, targetActor, nil, nil, message, "", ToNotVict)
	}
	if message, ok := socialMessage(social, socVictFound); ok {
		Act(nil, social.hidesInvisibleActor(), me, targetActor, nil, nil, message, "", ToVict)
	}
}
