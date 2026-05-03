package session

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/command"
	"github.com/zax0rz/darkpawns/pkg/common"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/game/systems"
)

// positionFailMessage returns an appropriate rejection message when
// the player's position is too low for a command.
func positionFailMessage(pos int) string {
	switch pos {
	case combat.PosDead:
		return "You are dead! You can't do that."
	case combat.PosMortally:
		return "You are mortally wounded and cannot do that."
	case combat.PosIncap:
		return "You are incapacitated and cannot do that."
	case combat.PosStunned:
		return "You are stunned and cannot do that."
	case combat.PosSleeping:
		return "You are asleep and cannot do that!"
	case combat.PosResting:
		return "You need to stand up first."
	default:
		return "You are in no position to do that!"
	}
}

// cmdRegistry is the global command registry, initialized on first use.
var cmdRegistry = command.NewRegistry()

// commandSession wraps a *Session to satisfy common.CommandSession.
// It adapts GetPlayer() *game.Player to GetPlayer() interface{}.
type commandSession struct {
	*Session
}

func (cs *commandSession) GetPlayer() interface{} {
	return cs.Session.GetPlayer()
}

// init registers all built-in commands at package initialization.
func init() {
	// Movement
	cmdRegistry.Register("north", wrapMove("north"), "Move north.", 0, combat.PosStanding, "n")
	cmdRegistry.Register("east", wrapMove("east"), "Move east.", 0, combat.PosStanding, "e")
	cmdRegistry.Register("south", wrapMove("south"), "Move south.", 0, combat.PosStanding, "s")
	cmdRegistry.Register("west", wrapMove("west"), "Move west.", 0, combat.PosStanding, "w")
	cmdRegistry.Register("up", wrapMove("up"), "Move up.", 0, combat.PosStanding, "u")
	cmdRegistry.Register("down", wrapMove("down"), "Move down.", 0, combat.PosStanding, "d")

	// Look
	cmdRegistry.Register("look", wrapArgs(cmdLook), "Look around the room.", 0, 0, "l")

	// Communication
	cmdRegistry.Register("say", wrapArgs(cmdSay), "Say something to the room.", 0, 0)
	cmdRegistry.Register("tell", wrapArgs(cmdTell), "Send a private message to a player.", 0, 0)
	cmdRegistry.Register("emote", wrapArgs(cmdEmote), "Perform a roleplay action.", 0, 0, "me")
	cmdRegistry.Register("shout", wrapArgs(cmdShout), "Shout to everyone in your zone.", 0, 0)
	cmdRegistry.Register("gtell", wrapArgs(cmdGtell), "Send a message to your group.", 0, 0, "gsay")

	// Combat
	cmdRegistry.Register("hit", wrapArgs(cmdHit), "Attack a target.", 0, combat.PosStanding, "attack", "kill")
	cmdRegistry.Register("flee", wrapNoArgs(cmdFlee), "Attempt to flee from combat.", 0, combat.PosFighting)

	// Position / Movement
	cmdRegistry.Register("stand", wrapNoArgs(cmdStand), "Stand up.", 0, 0)
	cmdRegistry.Register("sit", wrapNoArgs(cmdSit), "Sit down.", 0, 0)
	cmdRegistry.Register("rest", wrapNoArgs(cmdRest), "Rest.", 0, 0)
	cmdRegistry.Register("sleep", wrapNoArgs(cmdSleep), "Go to sleep.", 0, 0)
	cmdRegistry.Register("wake", wrapArgs(cmdWake), "Wake up or wake someone else.", 0, 0)

	// Items
	cmdRegistry.Register("inventory", wrapArgs(cmdInventory), "Show your inventory.", 0, 0, "i", "inv")
	cmdRegistry.Register("equipment", wrapArgs(cmdEquipment), "Show your equipped items.", 0, 0, "eq")
	cmdRegistry.Register("wear", wrapArgs(cmdWear), "Wear an item from your inventory.", 0, 0)
	cmdRegistry.Register("remove", wrapArgs(cmdRemove), "Remove an equipped item.", 0, 0)
	cmdRegistry.Register("wield", wrapArgs(cmdWield), "Wield a weapon.", 0, 0)
	cmdRegistry.Register("hold", wrapArgs(cmdHold), "Hold an item.", 0, 0)
	cmdRegistry.Register("get", wrapArgs(cmdGet), "Pick up an item from the room, container, or corpse.", 0, 0, "take")
	cmdRegistry.Register("give", wrapArgs(cmdGive), "Give an item or gold to another character.", 0, 0)
	cmdRegistry.Register("put", wrapArgs(cmdPut), "Put an item into a container.", 0, 0)
	cmdRegistry.Register("drop", wrapArgs(cmdDrop), "Drop an item from your inventory.", 0, 0)
	cmdRegistry.Register("eat", wrapArgs(cmdEat), "Eat some food.", 0, 0)
	cmdRegistry.Register("drink", wrapArgs(cmdDrink), "Drink from a container.", 0, 0)
	cmdRegistry.Register("quaff", wrapArgs(cmdQuaff), "Quaff a potion.", 0, 0, "q")

	// Info
	cmdRegistry.Register("score", wrapNoArgs(cmdScore), "Show your character stats.", 0, 0, "sc")
	cmdRegistry.Register("who", wrapNoArgs(cmdWho), "List all online players.", 0, 0)
	cmdRegistry.Register("where", wrapNoArgs(cmdWhere), "Show player locations.", 0, 0)
	cmdRegistry.Register("review", wrapNoArgs(cmdReview), "Show recent gossip history.", 2, 0)
	cmdRegistry.Register("whois", wrapArgs(cmdWhois), "Look up a player's info.", 2, 0)
	cmdRegistry.Register("help", wrapArgs(cmdHelp), "Show available commands or help for a topic.", 0, 0)

	// Group
	cmdRegistry.Register("follow", wrapArgs(cmdFollow), "Follow another player.", 0, 0)
	cmdRegistry.Register("group", wrapArgs(cmdGroup), "Manage your group.", 0, 0, "party")
	cmdRegistry.Register("ungroup", wrapArgs(cmdUngroup), "Disband or leave a group.", 0, 0, "disband")

	// Skills (delegated to pkg/command)
	cmdRegistry.Register("skills", wrapSkill(command.CmdSkills), "Show your learned skills.", 0, 0, "sk")
	cmdRegistry.Register("practice", wrapSkill(command.CmdPractice), "Practice a skill.", 0, 0)
	cmdRegistry.Register("learn", wrapSkill(command.CmdLearn), "Learn a new skill.", 0, 0)
	cmdRegistry.Register("listskills", wrapSkill(command.CmdListSkills), "List available skills.", 0, 0, "skills")

	// Shop
	cmdRegistry.Register("list", wrapArgs(cmdList), "List items for sale at a shop.", 0, 0)
	cmdRegistry.Register("buy", wrapArgs(cmdBuy), "Buy an item from a shop.", 0, 0)
	cmdRegistry.Register("sell", wrapArgs(cmdSell), "Sell an item to a shop.", 0, 0)
	cmdRegistry.Register("forget", wrapSkill(command.CmdForget), "Forget a skill.", 0, 0)
	cmdRegistry.Register("confirm", wrapSkill(command.CmdConfirmForget), "Confirm forgetting a skill.", 0, 0, "confirm forget")
	cmdRegistry.Register("use", wrapSkill(command.CmdUseSkill), "Use a skill.", 0, 0)
	cmdRegistry.Register("skillinfo", wrapSkill(command.CmdSkillInfo), "Show info about a skill.", 0, 0, "sinfo")

	// Combat skills (delegated to pkg/command)
	cmdRegistry.Register("backstab", wrapSkill(command.CmdBackstab), "Backstab a target with a piercing weapon.", 0, combat.PosStanding, "bs")
	cmdRegistry.Register("bash", wrapSkill(command.CmdBash), "Bash a target, potentially stunning them.", 0, combat.PosFighting)
	cmdRegistry.Register("kick", wrapSkill(command.CmdKick), "Kick a target for damage.", 0, combat.PosFighting)
	cmdRegistry.Register("trip", wrapSkill(command.CmdTrip), "Trip a target, knocking them down.", 0, combat.PosFighting)
	cmdRegistry.Register("headbutt", wrapSkill(command.CmdHeadbutt), "Headbutt a target for high damage.", 0, combat.PosFighting)
	cmdRegistry.Register("rescue", wrapSkill(command.CmdRescue), "Rescue someone from combat.", 0, combat.PosStanding)
	cmdRegistry.Register("sneak", wrapSkill(command.CmdSneak), "Attempt to move silently.", 0, combat.PosStanding)
	cmdRegistry.Register("hide", wrapSkill(command.CmdHide), "Attempt to hide in the shadows.", 0, combat.PosResting)
	cmdRegistry.Register("steal", wrapSkill(command.CmdSteal), "Steal from a target.", 0, combat.PosStanding)
	cmdRegistry.Register("pick", wrapArgs(cmdPick), "Pick a lock on a door.", 0, combat.PosStanding, "pick lock")

	// Admin / debug
	cmdRegistry.Register("summon", wrapArgs(cmdSummon), "Summon a player to your room.", 0, 0)

	// Doors
	cmdRegistry.Register("open", wrapArgs(cmdOpen), "Open a door in a direction: open <north|south|east|west|up|down>", 0, 0)
	cmdRegistry.Register("close", wrapArgs(cmdClose), "Close a door in a direction: close <north|south|east|west|up|down>", 0, 0)
	cmdRegistry.Register("lock", wrapArgs(cmdLock), "Lock a door with your key: lock <north|south|east|west|up|down>", 0, 0)
	cmdRegistry.Register("unlock", wrapArgs(cmdUnlock), "Unlock a door with your key: unlock <north|south|east|west|up|down>", 0, 0)
	cmdRegistry.Register("knock", wrapArgs(cmdKnock), "Knock on a door: knock <north|south|east|west|up|down>", 0, 0)
	cmdRegistry.Register("bashdoor", wrapArgs(cmdBashDoor), "Bash down a door: bashdoor <north|south|east|west|up|down>", 0, 0, "dbash")

	// Wizard commands
	cmdRegistry.Register("goto", wrapArgs(cmdGoto), "Teleport to a room by VNum.", LVL_IMMORT, 0)
	cmdRegistry.Register("at", wrapArgs(cmdAt), "Execute a command at another room.", LVL_IMMORT, 0)
	cmdRegistry.Register("load", wrapArgs(cmdLoad), "Load a mob or object by VNum.", LVL_IMMORT, 0)
	cmdRegistry.Register("purge", wrapArgs(cmdPurge), "Remove all mobs/items from a room.", LVL_GOD, 0)
	cmdRegistry.Register("teleport", wrapArgs(cmdTeleport), "Teleport another player to a room.", LVL_GOD, 0)
	cmdRegistry.Register("heal", wrapArgs(cmdHeal), "Fully heal a target.", LVL_IMMORT, 0)
	cmdRegistry.Register("restore", wrapArgs(cmdRestore), "Restore all stats of a target.", LVL_IMMORT, 0)
	cmdRegistry.Register("set", wrapArgs(cmdSet), "Set character fields.", LVL_IMMORT, 0)
	cmdRegistry.Register("switch", wrapArgs(cmdSwitch), "Enter another character's body.", LVL_IMMORT, 0)
	cmdRegistry.Register("return", wrapArgs(cmdReturn), "Return from switched body.", LVL_IMMORT, 0)
	cmdRegistry.Register("invis", wrapArgs(cmdInvis), "Become invisible to players.", LVL_IMMORT, 0)
	cmdRegistry.Register("vis", wrapArgs(cmdVis), "Become visible again.", LVL_IMMORT, 0)
	cmdRegistry.Register("gecho", wrapArgs(cmdGecho), "Echo a message to all players.", LVL_GOD, 0)
	cmdRegistry.Register("echo", wrapArgs(cmdEcho), "Echo a message to the room.", LVL_IMMORT, 0)
	cmdRegistry.Register("send", wrapArgs(cmdSend), "Send a message to another character.", LVL_GOD, 0)
	cmdRegistry.Register("force", wrapArgs(cmdForce), "Force a command on another character.", LVL_GRGOD, 0)
	cmdRegistry.Register("shutdown", wrapArgs(cmdShutdown), "Shutdown the server.", LVL_GRGOD, 0)
	cmdRegistry.Register("snoop", wrapArgs(cmdSnoop), "Spy on a player's input.", LVL_GOD, 0)
	cmdRegistry.Register("advance", wrapArgs(cmdAdvance), "Advance a player's level.", LVL_GRGOD, 0)
	cmdRegistry.Register("reload", wrapArgs(cmdReload), "Reload world data.", LVL_GOD, 0)

	// Wizard — stat/info
	cmdRegistry.Register("stat", wrapArgs(cmdStat), "Inspect a character, room, or object.", LVL_IMMORT, 0)
	cmdRegistry.Register("vnum", wrapArgs(cmdVnum), "Search for vnums by keyword.", LVL_IMMORT, 0)
	cmdRegistry.Register("vstat", wrapArgs(cmdVstat), "Show detailed prototype info by vnum.", LVL_IMMORT, 0)
	cmdRegistry.Register("wizlock", wrapArgs(cmdWizlock), "Toggle wizard-only login.", LVL_IMPL, 0)
	cmdRegistry.Register("dc", wrapArgs(cmdDc), "Disconnect a player.", LVL_GOD, 0)
	cmdRegistry.Register("home", wrapArgs(cmdHome), "Teleport to home room or specified room.", LVL_IMMORT, 0)
	cmdRegistry.Register("date", wrapArgs(cmdDate), "Show current system time or uptime.", LVL_IMMORT, 0)
	cmdRegistry.Register("last", wrapArgs(cmdLast), "Show last login info for a player.", LVL_IMMORT, 0)
	cmdRegistry.Register("wizutil", wrapArgs(cmdWizutil), "Player utility commands (reroll/pardon/notitle/squelch/freeze/thaw/unaffect).", LVL_IMMORT, 0)
	cmdRegistry.Register("show", wrapArgs(cmdShow), "Show system info (players/uptime/stats/reset).", LVL_IMMORT, 0)
	cmdRegistry.Register("dark", wrapArgs(cmdDark), "Stop combat in the current room.", LVL_IMMORT, 0)
	cmdRegistry.Register("syslog", wrapArgs(cmdSyslog), "Toggle system logging level.", LVL_IMMORT, 0)
	cmdRegistry.Register("idlist", wrapArgs(cmdIdlist), "Dump object ID list to file.", LVL_IMPL, 0)
	cmdRegistry.Register("checkload", wrapArgs(cmdCheckload), "Check zone load info for a mob/obj.", LVL_IMMORT, 0)
	cmdRegistry.Register("poofset", wrapArgs(cmdPoofset), "Set poof in/out messages.", LVL_IMMORT, 0)
	cmdRegistry.Register("wiznet", wrapArgs(cmdWiznet), "Send message on wizard net.", LVL_IMMORT, 0)
	cmdRegistry.Register("zreset", wrapArgs(cmdZreset), "Reset a zone by number.", LVL_GOD, 0)
	cmdRegistry.Register("zlist", wrapArgs(cmdZlist), "List zones matching a filter.", LVL_IMMORT, 0)
	cmdRegistry.Register("rlist", wrapArgs(cmdRlist), "List rooms matching a keyword.", LVL_IMMORT, 0)
	cmdRegistry.Register("olist", wrapArgs(cmdOlist), "List objects matching a keyword.", LVL_IMMORT, 0)
	cmdRegistry.Register("mlist", wrapArgs(cmdMlist), "List mobiles matching a keyword.", LVL_IMMORT, 0)
	cmdRegistry.Register("sysfile", wrapArgs(cmdSysfile), "Show system file path.", LVL_IMMORT, 0)
	cmdRegistry.Register("sethunt", wrapArgs(cmdSethunt), "Set hunt target for a character.", LVL_IMMORT, 0)
	cmdRegistry.Register("tick", wrapArgs(cmdTick), "Show current tick info.", LVL_IMMORT, 0)
	cmdRegistry.Register("newbie", wrapArgs(cmdNewbie), "Give newbie equipment to a player.", LVL_IMMORT, 0)

	// Informative
	cmdRegistry.Register("consider", wrapArgs(cmdConsider), "Compare yourself to a target.", 0, 0, "con")
	cmdRegistry.Register("examine", wrapArgs(cmdExamine), "Examine something in detail.", 0, 0, "exa")
	cmdRegistry.Register("time", wrapArgs(cmdTime), "Show the current time.", 0, 0)
	cmdRegistry.Register("weather", wrapArgs(cmdWeather), "Show the current weather.", 0, 0)
	cmdRegistry.Register("affects", wrapArgs(cmdAffects), "Show active affects.", 0, 0)
	cmdRegistry.Register("autoexit", wrapArgs(cmdAutoExit), "Toggle auto-exit display.", 0, 0)
	cmdRegistry.Register("title", wrapArgs(cmdTitle), "Set your title.", 0, 0)
	cmdRegistry.Register("describe", wrapArgs(cmdDescribe), "Set your description.", 0, 0, "desc")
	cmdRegistry.Register("spells", wrapArgs(cmdSpells), "List known spells.", 0, 0)

	// Quit
	cmdRegistry.Register("quit", wrapNoArgs(cmdQuit), "Quit the game.", 0, 0)

	// Offensive commands — delegated to pkg/command (C-10: real damage formulas)
	cmdRegistry.Register("assist", wrapArgs(cmdAssist), "Assist a target in combat.", LVL_IMMORT, combat.PosFighting)
	cmdRegistry.Register("disembowel", wrapSkill(command.CmdDisembowel), "Disembowel a target with a piercing weapon.", 0, combat.PosFighting, "gut")
	cmdRegistry.Register("dragonkick", wrapSkill(command.CmdDragonKick), "Dragon-style kick attack.", 0, combat.PosFighting, "dkick")
	cmdRegistry.Register("tigerpunch", wrapSkill(command.CmdTigerPunch), "Tiger-style punch attack (bare hands).", 0, combat.PosFighting, "tpunch")
	cmdRegistry.Register("shoot", wrapSkill(command.CmdShoot), "Shoot a target with a ranged weapon.", 0, combat.PosStanding)
	cmdRegistry.Register("subdue", wrapSkill(command.CmdSubdue), "Subdue a target (non-lethal).", 0, combat.PosStanding)
	cmdRegistry.Register("sleeper", wrapSkill(command.CmdSleeper), "Apply a sleeper hold to a target.", 0, combat.PosStanding)
	cmdRegistry.Register("neckbreak", wrapSkill(command.CmdNeckbreak), "Break a target's neck (bare hands).", 0, combat.PosStanding)
	cmdRegistry.Register("ambush", wrapSkill(command.CmdAmbush), "Ambush a target from hiding.", 0, combat.PosStanding)
	cmdRegistry.Register("parry", wrapSkill(command.CmdParry), "Toggle parry stance to deflect attacks.", 0, combat.PosStanding)
	cmdRegistry.Register("order", wrapArgs(cmdOrder), "Order a pet or follower.", LVL_IMMORT, 0)

	// Informative commands (act_informative.go)
	cmdRegistry.Register("color", wrapArgs(cmdColor), "Toggle ANSI color.", 0, 0)
	cmdRegistry.Register("commands", wrapArgs(cmdCommands), "List available commands.", 0, 0, "cmds")
	cmdRegistry.Register("description", wrapArgs(cmdDescription), "Set your character description.", 0, 0)
	cmdRegistry.Register("diagnose", wrapArgs(cmdDiagnose), "Diagnose health status of a target.", 0, 0, "diag")
	cmdRegistry.Register("toggle", wrapArgs(cmdToggle), "Toggle a player preference.", 0, 0)
	cmdRegistry.Register("users", wrapArgs(cmdUsersSafe), "Show connected players.", LVL_IMMORT, 0)

	// Other commands (act_other.go)
	cmdRegistry.Register("save", wrapArgs(cmdSave), "Save your character.", 0, 0)
	cmdRegistry.Register("report", wrapArgs(cmdReport), "Show report of your surroundings.", 0, 0)
	cmdRegistry.Register("split", wrapArgs(cmdSplit), "Split gold with your group.", 0, 0)
	cmdRegistry.Register("wimpy", wrapArgs(cmdWimpy), "Set your wimpy threshold.", 0, 0)
	cmdRegistry.Register("display", wrapArgs(cmdDisplay), "Set display preferences.", 0, 0)
	cmdRegistry.Register("transform", wrapArgs(cmdTransform), "Transform your appearance.", 0, 0)
	cmdRegistry.Register("ride", wrapArgs(cmdRide), "Ride a mount.", 0, 0)
	cmdRegistry.Register("dismount", wrapArgs(cmdDismount), "Dismount from your mount.", 0, 0)
	cmdRegistry.Register("yank", wrapArgs(cmdYank), "Yank someone from a mount or chair.", 0, 0)
	cmdRegistry.Register("peek", wrapArgs(cmdPeek), "Peek at another player's inventory.", 0, 0)
	cmdRegistry.Register("recall", wrapArgs(cmdRecall), "Recall to your home city.", 0, 0)
	cmdRegistry.Register("stealth", wrapArgs(cmdStealth), "Enter stealth mode.", 0, 0)
	cmdRegistry.Register("appraise", wrapArgs(cmdAppraise), "Appraise an item's value.", 0, 0)
	cmdRegistry.Register("scout", wrapArgs(cmdScout), "Scout ahead for danger.", 0, 0)
	cmdRegistry.Register("roll", wrapArgs(cmdRoll), "Roll a random number.", 0, 0)
	cmdRegistry.Register("visible", wrapArgs(cmdVisible), "Make yourself visible again.", 0, 0)
	cmdRegistry.Register("inactive", wrapArgs(cmdInactive), "Toggle inactive status.", 0, 0)
	cmdRegistry.Register("auto", wrapArgs(cmdAuto), "Toggle auto-attack mode.", 0, 0)
	cmdRegistry.Register("gentog", wrapArgs(cmdGenTog), "Toggle an option.", 0, 0, "gentoggle")
	cmdRegistry.Register("bug", wrapArgs(cmdBug), "Report a bug.", 0, 0)
	cmdRegistry.Register("typo", wrapArgs(cmdTypo), "Report a typo.", 0, 0)
	cmdRegistry.Register("idea", wrapArgs(cmdIdea), "Submit an idea.", 0, 0)
	cmdRegistry.Register("todo", wrapArgs(cmdTodo), "Submit a todo suggestion.", 0, 0)
	cmdRegistry.Register("afk", wrapArgs(cmdAFK), "Toggle away-from-keyboard status.", 0, 0)

	// Ban system (ported from ban.c)
	cmdRegistry.Register("ban", wrapArgs(cmdBan), "Ban a site (admin only).", LVL_GOD, 0)
	cmdRegistry.Register("unban", wrapArgs(cmdUnban), "Unban a site (admin only).", LVL_GOD, 0)

	// WHOD (ported from whod.c)
	cmdRegistry.Register("whod", wrapArgs(cmdWhod), "Toggle WHOD display mode (admin only).", LVL_IMMORT, 0)

	// Clan system (ported from clan.c)
	cmdRegistry.Register("clan", wrapArgs(cmdClan), "Clan management commands.", 0, 0, "clans")

	// Houses (ported from house.c)
	cmdRegistry.Register("house", wrapArgs(cmdHouse), "House management commands.", 0, 0)
	cmdRegistry.Register("hcontrol", wrapArgs(cmdHcontrol), "Admin house control.", 0, 0)
	cmdRegistry.Register("gossip", wrapArgs(cmdGossip), "Gossip on the channel.", 0, 0)
	cmdRegistry.Register("password", wrapArgs(cmdPassword), "Change your password.", 0, 0)
	cmdRegistry.Register("prompt", wrapArgs(cmdPrompt), "Set your prompt.", 0, 0)
	cmdRegistry.Register("reply", wrapArgs(cmdReply), "Reply to the last tell.", 0, 0, "r")
	cmdRegistry.Register("write", wrapArgs(cmdWrite), "Write on an object.", 0, 0)
	cmdRegistry.Register("page", wrapArgs(cmdPage), "Page a player.", 0, 0)
	cmdRegistry.Register("ignore", wrapArgs(cmdIgnore), "Ignore or stop ignoring a player.", 0, 0)
	cmdRegistry.Register("race_say", wrapArgs(cmdRaceSay), "Say something in your racial language.", 0, 0, "rac")
	cmdRegistry.Register("whisper", wrapArgs(cmdWhisper), "Whisper to someone in your room.", 0, 0, "whis")
	cmdRegistry.Register("ask", wrapArgs(cmdAsk), "Ask someone a question.", 0, 0)
	cmdRegistry.Register("qcomm", wrapArgs(cmdQcomm), "Send a team message.", 0, 0, "team")
	// Social (act_social.go)

	// Alias (game pkg)
	cmdRegistry.Register("alias", wrapArgs(cmdAlias), "Manage command aliases.", 0, 0)

	// Admin commands (game pkg bans) — duplicate of whod.c port; let the first one win
	// (no re-register here to avoid overwriting minPosition)
}

// wrapArgs adapts a func(*Session, []string) error to command.Handler.
func wrapArgs(fn func(*Session, []string) error) command.Handler {
	return func(s common.CommandSession, args []string) error {
		return fn(s.(*commandSession).Session, args)
	}
}

// wrapNoArgs adapts a func(*Session) error to command.Handler.
func wrapNoArgs(fn func(*Session) error) command.Handler {
	return func(s common.CommandSession, args []string) error {
		return fn(s.(*commandSession).Session)
	}
}

// wrapMove adapts cmdMove to the registry handler signature.
func wrapMove(direction string) command.Handler {
	return func(s common.CommandSession, args []string) error {
		return cmdMove(s.(*commandSession).Session, direction)
	}
}

// wrapSkill adapts a skill command (which uses command.SessionInterface) to command.Handler.
func wrapSkill(fn func(command.SessionInterface, []string) error) command.Handler {
	return func(s common.CommandSession, args []string) error {
		return fn(s.(*commandSession).Session, args)
	}
}

// ExecuteCommand processes a game command.
func ExecuteCommand(s *Session, cmdStr string, args []string) error {
	// Split command from arguments if args not provided separately
	if len(args) == 0 {
		if idx := strings.IndexByte(cmdStr, ' '); idx >= 0 {
			args = strings.Fields(cmdStr[idx+1:])
			cmdStr = cmdStr[:idx]
		}
	}
	cmd := strings.ToLower(cmdStr)

	// Check for mob scripts with oncmd trigger before processing
	// Based on the original MUD's script handling
	if s.player != nil && s.player.GetRoomVNum() > 0 {
		// Get mobs in the room
		mobs := s.manager.world.GetMobsInRoom(s.player.GetRoomVNum())
		fullCommand := cmdStr
		if len(args) > 0 {
			fullCommand = cmdStr + " " + strings.Join(args, " ")
		}

		// Check each mob for oncmd script
		for _, mob := range mobs {
			if mob.HasScript("oncmd") {
				// Create script context
				ctx := mob.CreateScriptContext(s.player, nil, fullCommand)
				// Run the script
				handled, err := mob.RunScript("oncmd", ctx)
				if err != nil {
					// Log error but continue
					slog.Error("error running oncmd script", "mob_vnum", mob.GetVNum(), "error", err)
				}
				if handled {
					// Script handled the command, don't process further
					return nil
				}
			}
		}
	}

	entry, ok := cmdRegistry.Lookup(cmd)
	if !ok {
		// Check social emotes before giving up
		if social, found := game.Socials[cmd]; found {
			return cmdSocial(s, social, args)
		}
		s.sendText(fmt.Sprintf("Unknown command: %s", cmdStr))
		return nil
	}

	// Enforce MinPosition gate — dead players can't attack, sleeping players can't backstab, etc.
	if entry.MinPosition > 0 && s.player != nil {
		playerPos := s.player.GetPosition()
		if playerPos < entry.MinPosition {
			s.sendText(positionFailMessage(playerPos))
			return nil
		}
	}

	// C-10: WAIT_STATE enforcement — combat skills set cooldowns
	if s.player != nil && s.player.GetWaitState() > 0 {
		s.sendText("You're too busy!\r\n")
		return nil
	}

	return entry.Handler(&commandSession{Session: s}, args)
}

// cmdSocial performs a social emote.
// Based on the original ROM: act.social.c do_action()
var directions = map[string]string{
	"north": "north", "n": "north",
	"east": "east", "e": "east",
	"south": "south", "s": "south",
	"west": "west", "w": "west",
	"up": "up", "u": "up",
	"down": "down", "d": "down",
}

// resolveDirection returns the full direction name or empty string if invalid.
func resolveDirection(input string) string {
	if dir, ok := directions[input]; ok {
		return dir
	}
	return ""
}

// doorBroadcast sends a door-related message to all players in the same room, excluding the actor.
func doorBroadcast(s *Session, message string) {
	if s.player == nil {
		return
	}
	roomVNum := s.player.GetRoom()
	msg, err := json.Marshal(ServerMessage{
		Type: MsgEvent,
		Data: EventData{
			Type: "door",
			Text: message,
		},
	})
	if err != nil {
		slog.Error("json.Marshal error", "error", err)
		return
	}
	s.manager.BroadcastToRoom(roomVNum, msg, s.player.Name)
}

// playerHasKey checks if the player has an item with the given VNum in their inventory.
func playerHasKey(s *Session, keyVNum int) bool {
	if s.player == nil {
		return false
	}
	inv := s.player.Inventory
	if inv == nil {
		return false
	}
	for _, item := range inv.Items {
		if item.VNum == keyVNum {
			return true
		}
	}
	return false
}

// getDoorManager returns the DoorManager from the world.
func getDoorManager(s *Session) *systems.DoorManager {
	if s.manager == nil {
		return nil
	}
	return s.manager.doorManager
}

// cmdSave saves the player's character.
