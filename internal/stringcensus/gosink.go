package stringcensus

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"sort"
	"strings"
)

// goSinkKind says how a sink is spelled at a call site.
type goSinkKind int

const (
	// goSinkMethod is a call on a receiver: x.Send(...).
	goSinkMethod goSinkKind = iota
	// goSinkFunc is a package-level call: Act(...).
	goSinkFunc
	// goSinkBoth is a name that appears as both (SendToRoom does).
	goSinkBoth
)

// goSink is one player-output sink in the Go tree. This table is the whole
// scope of the Go side of the census; every string the tool reports as
// player-facing came through one of these. Counts in Why are the brief's grep
// counts for 2026-09-23, kept so the table can be re-derived.
type goSink struct {
	// Name is the bare call name, without a receiver dot.
	Name string
	// Kind says how the name is spelled at a call site.
	Kind goSinkKind
	// Why says what the sink is, for the sink table in the report.
	Why string
	// TextArgs lists the argument positions that carry player text, for a sink
	// whose parameters mix text with data (a channel name, a log line). An empty
	// list means every argument, the original table's reading.
	TextArgs []int
}

// goSinks is the Go sink table. It is the brief's list plus every function
// this tree wraps around those primitives: a literal passed to sendToChar is a
// literal send_to_char receives. Each entry was found by walking callers up
// from the listed sinks (a parameter that reaches a sink call is the new sink's
// text argument), then verified by reading the body: the parameter is the
// message, not a name, key or other datum the function parses or substitutes.
// The counts in Why are call sites in the census's four packages.
var goSinks = []goSink{
	{Name: "Send", Kind: goSinkMethod, Why: "Session.Send and wrapper sessions (679 call sites)"},
	{Name: "SendMessage", Kind: goSinkMethod, Why: "Player/MobInstance.SendMessage (599 call sites)"},
	{Name: "Act", Kind: goSinkFunc, Why: "pkg/game Act(): the act()-style primitive (427 call sites)"},
	{Name: "sendText", Kind: goSinkMethod, Why: "Session.sendText, group and mail text (73 call sites)"},
	{Name: "SendToRoom", Kind: goSinkBoth, Why: "World.SendToRoom and act.go SendToRoom(): room broadcast (24 call sites)"},
	{Name: "BroadcastToRoom", Kind: goSinkMethod, Why: "SessionManager.BroadcastToRoom, socket-level room broadcast (22 call sites)"},
	{Name: "actToRoom", Kind: goSinkBoth, Why: "World.actToRoom method and the package-level twin in other_helpers.go:94 (20 call sites)"},
	{Name: "SendToAll", Kind: goSinkMethod, Why: "World.SendToAll and SessionManager.SendToAll (10 call sites)"},
	{Name: "sendPromptText", Kind: goSinkMethod, Why: "Session.sendPromptText: prompt bytes are player-facing"},
	{Name: "SendToZone", Kind: goSinkMethod, Why: "World.SendToZone and the script adapter"},
	{Name: "SendToOutdoor", Kind: goSinkMethod, Why: "SessionManager.SendToOutdoor, weather broadcast"},
	{Name: "channelAct", Kind: goSinkMethod, Why: "pkg/game/out_of_band.go channelAct(): channel lines"},
	{Name: "channelSend", Kind: goSinkMethod, Why: "pkg/game/out_of_band.go channelSend(): the sender's own channel echo"},
	{Name: "SendToChar", Kind: goSinkFunc, Why: "act.go:744 SendToChar(): C send_to_char, a direct descriptor write with no SENDOK gate; callers supply their own line ending"},
	{Name: "sendToChar", Kind: goSinkFunc, Why: "spec_procs.go:85 sendToChar(): one-line SendMessage wrapper"},
	{Name: "movementSendToChar", Kind: goSinkFunc, Why: "movement_commands.go:12: one-line Act(ToChar|ToSleep) wrapper"},
	{Name: "communicationSend", Kind: goSinkFunc, Why: "directed_speech.go:56: one-line Act(ToChar|ToSleep) wrapper"},
	// Act delivery and the room-line wrappers built on it.
	{Name: "ActWithPrivateSink", Kind: goSinkFunc, Why: "pkg/game/act.go:592 Act with a private sink for a switched body (64 call sites)"},
	{Name: "sendActLine", Kind: goSinkFunc, Why: "pkg/game/act.go:608 Act's final write to one actor (2 call sites)"},
	{Name: "sendActLineWithWorld", Kind: goSinkFunc, Why: "pkg/game/act.go:612 Act delivery through an NPC body descriptor owner (11 call sites)"},
	{Name: "sendActRoomMessage", Kind: goSinkMethod, Why: "pkg/game/act.go:832 pre-rendered line to every act()-eligible room occupant (1 call site)"},
	{Name: "roomMessage", Kind: goSinkMethod, Why: "pkg/game/spec_procs.go:43 spec_procs room line (sendActRoomMessage plus CRLF) (4 call sites)"},
	{Name: "actToChar", Kind: goSinkMethod, Why: "pkg/game/item_helpers.go:373 legacy World.actToChar adapter to Act(ToChar) (37 call sites)"},
	{Name: "actToVictim", Kind: goSinkFunc, Why: "pkg/game/item_helpers.go:388 legacy actToVictim adapter to Act(ToVict) (2 call sites)"},
	{Name: "actToRoomExclude", Kind: goSinkMethod, Why: "pkg/game/item_helpers.go:395 legacy actToRoomExclude adapter to Act(ToNotVict) (2 call sites)"},
	{Name: "actForPrivateBody", Kind: goSinkFunc, Why: "pkg/game/flee.go:25 Act whose actor sink is a body message sink (6 call sites)"},
	{Name: "actDoorDestinationMessage", Kind: goSinkFunc, Why: "pkg/game/act_movement.go:725 door-move Act(ToRoom) line (3 call sites)"},
	{Name: "SpellActToChar", Kind: goSinkMethod, Why: "pkg/game/world_bridge.go:1151 spell Act(ToChar) bridge (1 call site)"},
	{Name: "SpellObjectAct", Kind: goSinkMethod, Why: "pkg/game/world_bridge.go:1135 act() on an object at an explicit target type (2 call sites)"},
	{Name: "SpellActToRoom", Kind: goSinkMethod, Why: "pkg/game/say_spell_act.go:79 spell Act(ToRoom) bridge (7 call sites)"},
	{Name: "SpellActNotVict", Kind: goSinkMethod, Why: "pkg/game/say_spell_act.go:67 spell Act(ToNotVict) bridge (2 call sites)"},
	{Name: "SpellActToVictim", Kind: goSinkMethod, Why: "pkg/game/say_spell_act.go:105 spell Act(ToVict) bridge (4 call sites)"},
	{Name: "SaySpellDeliver", Kind: goSinkMethod, Why: "pkg/game/say_spell_act.go:15 say-spell same-class, other-class and victim lines (1 call site)"},
	{Name: "sendAffectRoom", Kind: goSinkFunc, Why: "pkg/spells/affect_spells.go:1553 spell affect Act(ToRoom) through the world renderer (25 call sites)"},
	{Name: "spellActToRoom", Kind: goSinkFunc, Why: "pkg/spells/affect_spells.go:3540 spells-package Act(ToRoom) bridge (4 call sites)"},
	{Name: "castHellfireActToVictim", Kind: goSinkFunc, Why: "pkg/spells/affect_spells.go:3042 hellfire Act(ToVict) with a SendMessage fallback (4 call sites)"},
	{Name: "castHellfireActNotVictim", Kind: goSinkFunc, Why: "pkg/spells/affect_spells.go:3054 hellfire Act(ToNotVict) (3 call sites)"},
	{Name: "sendSpellActToCaster", Kind: goSinkFunc, Why: "pkg/spells/say_spell.go:264 spell Act(ToChar) through the world renderer (12 call sites)"},
	{Name: "sendActTargetToCaster", Kind: goSinkFunc, Why: "pkg/spells/damage_spells.go:169 Act(ToChar) with $N pasted from the target name (2 call sites)"},
	{Name: "sendToZone", Kind: goSinkFunc, Why: "pkg/spells/damage_spells.go:528 spells-package zone broadcast (2 call sites)"},
	{Name: "damageGateAct", Kind: goSinkMethod, Why: "pkg/game/damage_gate.go:77 damage-gate Act(ToChar) line (2 call sites)"},
	{Name: "tattooNoMagicRoom", Kind: goSinkMethod, Why: "pkg/game/other_economy.go:305 NOMAGIC tattoo Act(ToRoom) line (2 call sites)"},
	{Name: "puffEmote", Kind: goSinkFunc, Why: "pkg/game/spec_procs.go:720 spec_procs puff emote Act(ToRoom) (5 call sites)"},
	{Name: "elementTeleportPerson", Kind: goSinkFunc, Why: "pkg/game/actor_specials_rooms.go:113 element-room TO_CHAR/departure/arrival lines (3 call sites)", TextArgs: []int{3, 4, 5}},
	{Name: "RoomEcho", Kind: goSinkMethod, Why: "pkg/game/world.go:993 board system room broadcast (1 call site)"},
	{Name: "mobSendToZone", Kind: goSinkMethod, Why: "pkg/game/spec_procs3.go:781 mobile send_to_zone wrapper (1 call site)"},
	{Name: "sendToRoomExact", Kind: goSinkMethod, Why: "pkg/game/limits_condition.go:539 awake-gated room send_to_room (4 call sites)"},
	{Name: "sendWeatherRoom", Kind: goSinkFunc, Why: "pkg/game/weather.go:841 dusk-event send_to_room (7 call sites)"},
	{Name: "channelActWrapped", Kind: goSinkMethod, Why: "pkg/game/out_of_band.go:40 channelAct with an optional line wrapper (12 call sites)", TextArgs: []int{4}},
	// Message writers that choose a body or a private sink.
	{Name: "mobileSend", Kind: goSinkFunc, Why: "pkg/game/item_transfer_actor.go:354 PrivateOutput or mob SendMessage (321 call sites)"},
	{Name: "mobileActToChar", Kind: goSinkFunc, Why: "pkg/game/item_transfer_actor.go:362 Act(ToChar) with a private sink for an NPC body (39 call sites)"},
	{Name: "mobileActToCharWithTarget", Kind: goSinkFunc, Why: "pkg/game/skill_actor_itemmagic.go:354 Act(ToChar) with $N and a private sink (2 call sites)"},
	{Name: "mobileActionDescription", Kind: goSinkFunc, Why: "pkg/game/skill_actor_itemmagic.go:338 item action_desc Act(ToRoom) with fallback (6 call sites)"},
	{Name: "mobileActionDescriptionWithTarget", Kind: goSinkFunc, Why: "pkg/game/skill_actor_itemmagic.go:346 item action_desc Act(ToRoom) with a target (3 call sites)"},
	{Name: "sendBodyMessage", Kind: goSinkFunc, Why: "pkg/game/flee.go:14 player or NPC-body SendMessage (38 call sites)"},
	{Name: "combatSendMessage", Kind: goSinkFunc, Why: "pkg/game/party.go:412 player or NPC-body SendMessage (7 call sites)"},
	{Name: "sendCombatantMessage", Kind: goSinkFunc, Why: "pkg/game/skill_actor_shoot.go:185 player or NPC-body SendMessage (4 call sites)"},
	{Name: "sendCommandBodyMessage", Kind: goSinkFunc, Why: "pkg/game/other_helpers.go:83 command-body SendMessage for legacy bridges (4 call sites)"},
	{Name: "sendToVictim", Kind: goSinkFunc, Why: "pkg/spells/affect_spells.go:1485 spell victim SendMessage (22 call sites)"},
	{Name: "sendToCaster", Kind: goSinkFunc, Why: "pkg/spells/affect_spells.go:1492 spell caster SendMessage (125 call sites)"},
	{Name: "consumeReagent", Kind: goSinkFunc, Why: "pkg/spells/damage_spells.go:283 reagent consumption caster line (5 call sites)"},
	{Name: "wakeSelfLine", Kind: goSinkFunc, Why: "pkg/game/character_command.go:318 wake-up self line (5 call sites)"},
	{Name: "positionSelfLine", Kind: goSinkFunc, Why: "pkg/game/movement_commands.go:130 position-change self line (2 call sites)"},
	{Name: "cbSendToChar", Kind: goSinkFunc, Why: "pkg/combat/callbacks.go:260 combat callback send_to_char (1 call site)"},
	{Name: "cbActChar", Kind: goSinkFunc, Why: "pkg/combat/callbacks.go:231 combat callback act(ToChar) (1 call site)"},
	{Name: "sendCombatMessage", Kind: goSinkFunc, Why: "pkg/combat/callbacks.go:741 combat callback text sink (23 call sites)"},
	{Name: "sendColorBracketed", Kind: goSinkFunc, Why: "pkg/combat/skill_messages.go:53 color-bracketed skill message: message is argument 2, the colour code argument 1 (3 call sites)", TextArgs: []int{2}},
	// NPC speech and the channel adapters.
	{Name: "npcSay", Kind: goSinkMethod, Why: "pkg/game/npc_command.go:126 mobile do_say (20 call sites)"},
	{Name: "npcEmote", Kind: goSinkMethod, Why: "pkg/game/npc_command.go:157 mobile do_echo/emote (4 call sites)"},
	{Name: "npcEcho", Kind: goSinkMethod, Why: "pkg/game/npc_command.go:87 mobile do_echo (1 call site)"},
	{Name: "npcGlobalEcho", Kind: goSinkMethod, Why: "pkg/game/npc_command.go:98 mobile do_gecho (1 call site)"},
	{Name: "shopNpcSay", Kind: goSinkMethod, Why: "pkg/game/shop_npc.go:25 keeper do_say (4 call sites)"},
	{Name: "shopNpcTell", Kind: goSinkMethod, Why: "pkg/game/shop_npc.go:32 keeper do_tell buffer (22 call sites)"},
	{Name: "npcTell", Kind: goSinkMethod, Why: "pkg/game/npc_command.go:175 mobile do_tell buffer, target and text (2 call sites)"},
	{Name: "NPCTellCommand", Kind: goSinkMethod, Why: "pkg/game/comm_channel.go:491 NPC command adapter for do_tell (1 call site)"},
	{Name: "NPCSayCommand", Kind: goSinkMethod, Why: "pkg/game/npc_command.go:62 NPC command adapter for do_say (4 call sites)"},
	{Name: "NPCEchoCommand", Kind: goSinkMethod, Why: "pkg/game/npc_command.go:66 NPC command adapter for do_echo (1 call site)"},
	{Name: "NPCGechoCommand", Kind: goSinkMethod, Why: "pkg/game/npc_command.go:67 NPC command adapter for do_gecho (1 call site)"},
	{Name: "NPCEmoteCommand", Kind: goSinkMethod, Why: "pkg/game/npc_command.go:64 NPC command adapter for do_emote (3 call sites)"},
	{Name: "NPCGossipCommand", Kind: goSinkMethod, Why: "pkg/game/npc_command.go:83 NPC command adapter for gossip (1 call site)"},
	{Name: "NPCShoutCommand", Kind: goSinkMethod, Why: "pkg/game/npc_command.go:84 NPC command adapter for shout (1 call site)"},
	{Name: "NPCAuctionCommand", Kind: goSinkMethod, Why: "pkg/game/npc_command.go:85 NPC command adapter for auction (1 call site)"},
	{Name: "mobShoutGenComm", Kind: goSinkMethod, Why: "pkg/game/damage_gate.go:163 fixed-channel mobile do_gen_comm (3 call sites)"},
	{Name: "mobGlobalGossip", Kind: goSinkMethod, Why: "pkg/game/comm_channel.go:323 mobile gossip do_gen_comm (1 call site)"},
	{Name: "performTell", Kind: goSinkMethod, Why: "pkg/game/directed_speech.go:332 one tell line to a resolved recipient (4 call sites)"},
	{Name: "mobPerformTell", Kind: goSinkMethod, Why: "pkg/game/reply_body.go:138 one tell line from a mobile (2 call sites)"},
	{Name: "DoReply", Kind: goSinkMethod, Why: "pkg/game/directed_speech.go:275 reply command, text forwarded to performTell (2 call sites)"},
	{Name: "broadcastQuest", Kind: goSinkFunc, Why: "pkg/session/act_comm.go:111 quest broadcast to self and others (2 call sites)"},
	// Session output paths: pagers, OLC editors and text commands.
	{Name: "PageString", Kind: goSinkFunc, Why: "pkg/session/pager.go:111 session pager entry point (21 call sites)"},
	{Name: "pageString", Kind: goSinkFunc, Why: "pkg/session/pager.go:116 session pager body (2 call sites)"},
	{Name: "pageLiteralString", Kind: goSinkFunc, Why: "pkg/session/pager.go:114 session pager with verbatim C bytes (2 call sites)"},
	{Name: "sendPageText", Kind: goSinkMethod, Why: "pkg/session/pager.go:324 pager page write (4 call sites)"},
	{Name: "zeditSend", Kind: goSinkMethod, Why: "pkg/session/zedit.go:224 zedit buffered output (61 call sites)"},
	{Name: "sendTextEditor", Kind: goSinkMethod, Why: "pkg/session/tedit.go:234 tedit buffered output (56 call sites)"},
	{Name: "reditSend", Kind: goSinkMethod, Why: "pkg/session/redit.go:192 redit buffered output (40 call sites)"},
	{Name: "reditSendDirect", Kind: goSinkMethod, Why: "pkg/session/redit.go:200 redit immediate output (7 call sites)"},
	{Name: "oeditSend", Kind: goSinkMethod, Why: "pkg/session/oedit.go:350 oedit buffered output (11 call sites)"},
	{Name: "oeditSendLocked", Kind: goSinkMethod, Why: "pkg/session/oedit.go:360 oedit buffered output under the edit lock (39 call sites)"},
	{Name: "oeditActLocked", Kind: goSinkMethod, Why: "pkg/session/oedit.go:378 oedit Act(ToRoom) under the edit lock (2 call sites)"},
	{Name: "seditSend", Kind: goSinkMethod, Why: "pkg/session/sedit.go:230 sedit buffered output (27 call sites)"},
	{Name: "seditSendLocked", Kind: goSinkMethod, Why: "pkg/session/sedit.go:240 sedit buffered output under the edit lock (8 call sites)"},
	{Name: "meditSend", Kind: goSinkMethod, Why: "pkg/session/medit.go:253 medit output (10 call sites)"},
	{Name: "meditSendLocked", Kind: goSinkMethod, Why: "pkg/session/medit.go:259 medit output under the edit lock (19 call sites)"},
	{Name: "meditActLocked", Kind: goSinkMethod, Why: "pkg/session/medit.go:280 medit Act under the edit lock (2 call sites)"},
	{Name: "sendError", Kind: goSinkMethod, Why: "pkg/session/session_send.go:54 session error envelope text (10 call sites)"},
	{Name: "sendPlayerText", Kind: goSinkMethod, Why: "pkg/session/session_send.go:235 session text frame for a descriptor owner (1 call site)"},
	{Name: "sendLiteralText", Kind: goSinkMethod, Why: "pkg/session/session_send.go:146 session text frame preserving C bytes (10 call sites)"},
	{Name: "sendRawEvent", Kind: goSinkMethod, Why: "pkg/session/session_send.go:314 session raw transport event text (19 call sites)"},
	{Name: "sendStatMobLine", Kind: goSinkMethod, Why: "pkg/session/wiz_stats.go:225 stat mob output line (20 call sites)"},
	{Name: "sendShopKeeperTell", Kind: goSinkFunc, Why: "pkg/session/shop_cmds.go:58 keeper tell built from a format (18 call sites)", TextArgs: []int{2}},
	{Name: "sendPurgeRoomLine", Kind: goSinkFunc, Why: "pkg/session/wiz_object.go:263 purge room send_to_room (1 call site)"},
	{Name: "transferCharacter", Kind: goSinkFunc, Why: "pkg/session/wiz_movement.go:419 wiz transfer departure Act (2 call sites)"},
	{Name: "cmdGechoText", Kind: goSinkFunc, Why: "pkg/session/wiz_communication.go:18 gecho text (2 call sites)"},
	{Name: "cmdGtellText", Kind: goSinkFunc, Why: "pkg/session/cmd_group.go:216 gtell text (2 call sites)"},
	{Name: "cmdPageText", Kind: goSinkFunc, Why: "pkg/session/comm_cmds.go:180 page text (2 call sites)"},
	{Name: "cmdQsayText", Kind: goSinkFunc, Why: "pkg/session/act_comm.go:40 qsay text (2 call sites)"},
	{Name: "cmdQechoText", Kind: goSinkFunc, Why: "pkg/session/act_comm.go:68 qecho text (2 call sites)"},
	{Name: "cmdSendText", Kind: goSinkFunc, Why: "pkg/session/wiz_communication.go:74 wiz send text (2 call sites)"},
	{Name: "cmdWiznetText", Kind: goSinkFunc, Why: "pkg/session/wiz_communication.go:354 wiznet text (2 call sites)"},
}

// goSinkIndex maps every call spelling ("Send" as a function, ".Send" as a
// method) to its sink, so one table entry covers both forms.
func goSinkIndex() map[string]goSink {
	m := make(map[string]goSink, len(goSinks)*2)
	for _, s := range goSinks {
		if s.Kind != goSinkFunc {
			m["."+s.Name] = s
		}
		if s.Kind != goSinkMethod {
			m[s.Name] = s
		}
	}
	return m
}

// sortedGoSinkNames returns the sink table's names in report order.
func sortedGoSinkNames() []string {
	names := make([]string, 0, len(goSinks))
	for _, s := range goSinks {
		names = append(names, s.Name)
	}
	sort.Strings(names)
	return names
}

// unresolvedSite is a sink argument the census refused to guess about: an
// expression that mentions a local variable whose value it could not resolve to
// literals. It is reported so a human can tighten the sink table or the
// extractor; it never gates the ratchet.
type unresolvedSite struct {
	file string
	line int
	sink string
	expr string
}

// goValue is the classification of one sink argument.
type goValue struct {
	// literals are the unescaped literals that reach the sink.
	literals []string
	// via names the local variable a resolved literal travelled through.
	via string
	// unresolved is the source text of an argument the census could not
	// resolve (empty when there is nothing to report).
	unresolved string
}

// goExtractor renders expressions for the unresolved report.
type goExtractor struct {
	fset *token.FileSet
}

func (g *goExtractor) render(n ast.Node) string {
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, g.fset, n); err != nil {
		return fmt.Sprintf("<%T>", n)
	}
	out := buf.String()
	if len(out) > 120 {
		out = out[:117] + "..."
	}
	return out
}

func mergeGoValues(a, b goValue) goValue {
	out := goValue{literals: append(append([]string{}, a.literals...), b.literals...)}
	switch {
	case a.unresolved != "":
		out.unresolved = a.unresolved
	case b.unresolved != "":
		out.unresolved = b.unresolved
	}
	if a.via == b.via {
		out.via = a.via
	}
	return out
}

// concatGoValues joins the two sides of a Go "+". When both sides are pure
// literals the result is one literal, which is what C's adjacent literals "a"
// "b" produce; when either side is runtime text the fragments stay separate,
// because joining them would invent a string the output never contains.
func concatGoValues(a, b goValue) goValue {
	if a.via == "" && b.via == "" && a.unresolved == "" && b.unresolved == "" {
		return goValue{literals: []string{strings.Join(append(append([]string{}, a.literals...), b.literals...), "")}}
	}
	return mergeGoValues(a, b)
}

func isFmtFormat(callee string) bool {
	return callee == "fmt.Sprintf" || callee == "fmt.Sprint"
}
