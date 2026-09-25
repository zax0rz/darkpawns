package game

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/scripting"
)

// NPCCommand is command_interpreter (interpreter.c:1270-1349) run by a
// mobile, as lua_action does for a script's action(me, "...") (scripts.c:
// 122-140). A mobile has no descriptor, so every refusal C would send it
// ("Huh?!?", the position messages, a command's own error lines) reaches no
// one; what matters is which commands run and what they show the room.
//
// The commands ported for mobiles are the ones C's attached scripts use:
// say, tell, emote, give and the socials. A line naming any other command is
// logged and does nothing, rather than guessing at its NPC behaviour. The
// special() hook C runs before the command (room, mobile and object
// specials that may intercept it) is not reached from here yet.
func (w *World) NPCCommand(me *MobInstance, line string) {
	if me == nil {
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
	case "give":
		if position >= combat.PosResting {
			w.npcGive(me, rest)
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
	targetName, message := splitDirectedSpeech(argument)
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
	w.channelAct("tell", false, me, target, deleteANSIControls(fmt.Sprintf("$n tells you, '%s'", message)), ToVict|ToSleep)
	if target.GetAFK() {
		return // "$E is AFK right now..." goes to the mobile
	}
}

// npcVisiblePlayer is get_char_vis for a mobile looking for a player to
// tell: the room first, then the world, visible to the mobile.
func (w *World) npcVisiblePlayer(me *MobInstance, name string) *Player {
	for _, p := range w.GetPlayersInRoom(me.GetRoom()) {
		if isnameWithAbbrevs(name, p.GetName()) && canSee(me, p) {
			return p
		}
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
	me.mu.RLock()
	carried := append([]*ObjectInstance(nil), me.Inventory...)
	me.mu.RUnlock()
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
	target := mobRescueVictim(w, me, targetName)
	if target == nil {
		return // action->not_found to the mobile
	}
	targetActor, _ := target.(Actor)
	if target.GetName() == me.GetName() {
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
