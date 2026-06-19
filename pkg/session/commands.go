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
	cmdRegistry.Register("think", wrapArgs(cmdThink), "Think a thought, optionally aloud.", 0, combat.PosResting)
	cmdRegistry.Register("insult", wrapArgs(cmdInsult), "Insult a target in the room.", 0, combat.PosResting)
	cmdRegistry.Register("dream", wrapArgs(cmdDream), "Dream — only does anything while asleep.", 0, combat.PosSleeping)

	// Combat
	cmdRegistry.Register("hit", wrapArgs(cmdHit), "Attack a target.", 0, combat.PosStanding, "attack")
	cmdRegistry.Register("kill", wrapArgs(cmdKill), "Kill a target (immortal instakill).", 0, combat.PosStanding)
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
	cmdRegistry.Register("junk", wrapArgs(cmdJunk), "Destroy an item for a small experience reward.", 0, 0)
	cmdRegistry.Register("donate", wrapArgs(cmdDonate), "Donate an item to the donation room.", 0, 0)
	cmdRegistry.Register("eat", wrapArgs(cmdEat), "Eat some food.", 0, 0)
	cmdRegistry.Register("taste", wrapArgs(cmdTaste), "Nibble a little bit of some food.", 0, 0)
	cmdRegistry.Register("drink", wrapArgs(cmdDrink), "Drink from a container.", 0, 0)
	cmdRegistry.Register("sip", wrapArgs(cmdSip), "Sip from a container without drinking it.", 0, 0)
	cmdRegistry.Register("pour", wrapArgs(cmdPour), "Pour liquid from one container to another.", 0, 0)
	cmdRegistry.Register("quaff", wrapArgs(cmdQuaff), "Quaff a potion.", 0, 0, "q")

	// Info
	cmdRegistry.Register("score", wrapNoArgs(cmdScore), "Show your character stats.", 0, 0, "sc")
	cmdRegistry.Register("who", wrapNoArgs(cmdWho), "List all online players.", 0, 0)
	cmdRegistry.Register("where", wrapNoArgs(cmdWhere), "Show player locations.", 0, 0)
	cmdRegistry.Register("coins", wrapNoArgs(cmdCoins), "Display your gold and bank balance.", 0, 0)
	// real C command name is "abilities" (src/interpreter.c); "abils" kept as alias.
	cmdRegistry.Register("abilities", wrapNoArgs(cmdAbils), "Show your ability scores.", 0, 0, "abils")
	cmdRegistry.Register("levels", wrapNoArgs(cmdLevels), "Show XP table for your class.", 0, 0)
	cmdRegistry.Register("review", wrapNoArgs(cmdReview), "Show recent gossip history.", 2, 0)
	cmdRegistry.Register("whois", wrapArgs(cmdWhois), "Look up a player's info.", 2, 0)
	cmdRegistry.Register("help", wrapArgs(cmdHelp), "Show available commands or help for a topic.", 0, 0)
	cmdRegistry.Register("credits", wrapArgs(cmdCredits), "Show who built this game.", 0, combat.PosDead)
	cmdRegistry.Register("news", wrapArgs(cmdNews), "Show current game news.", 0, combat.PosSleeping)
	cmdRegistry.Register("policy", wrapArgs(cmdPolicy), "Show the game's policies.", 0, combat.PosDead)
	cmdRegistry.Register("handbook", wrapArgs(cmdHandbook), "Show the immortal handbook.", LVL_IMMORT, combat.PosDead)
	cmdRegistry.Register("future", wrapArgs(cmdFuture), "Show planned future content.", 0, combat.PosDead)
	cmdRegistry.Register("whoami", wrapArgs(cmdWhoami), "Show your own name.", 0, combat.PosDead)
	cmdRegistry.Register("version", wrapArgs(cmdVersion), "Show the game version.", 0, combat.PosDead)

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
	cmdRegistry.Register("use", wrapArgs(cmdUse), "Use a wand/staff or a skill.", 0, 0)
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
	cmdRegistry.Register("berserk", wrapSkill(command.CmdBerserk), "Summon your battle rage for a hitroll/damroll boost.", 0, combat.PosFighting)
	cmdRegistry.Register("rin", wrapSkill(command.CmdKujiKiri(game.SkillKkRin)), "Kuji-kiri seal: harden body for an AC bonus and metalskin.", 0, combat.PosStanding)
	cmdRegistry.Register("kyo", wrapSkill(command.CmdKujiKiri(game.SkillKkKyo)), "Kuji-kiri seal: focus battle rage for a hitroll bonus.", 0, combat.PosStanding)
	cmdRegistry.Register("toh", wrapSkill(command.CmdKujiKiri(game.SkillKkToh)), "Kuji-kiri seal: focus inner strength for a damroll/AC bonus.", 0, combat.PosStanding)
	cmdRegistry.Register("kai", wrapSkill(command.CmdKujiKiri(game.SkillKkKai)), "Kuji-kiri seal: fortify your body, lowering damroll/AC.", 0, combat.PosStanding)
	cmdRegistry.Register("jin", wrapSkill(command.CmdKujiKiri(game.SkillKkJin)), "Kuji-kiri seal: focus on recuperation for faster HP regen.", 0, combat.PosStanding)
	cmdRegistry.Register("retsu", wrapSkill(command.CmdKujiKiri(game.SkillKkRetsu)), "Kuji-kiri seal: attempt to teleport away.", 0, combat.PosStanding)
	cmdRegistry.Register("zai", wrapSkill(command.CmdKujiKiri(game.SkillKkZai)), "Kuji-kiri seal: fade from view.", 0, combat.PosStanding)
	cmdRegistry.Register("zhen", wrapSkill(command.CmdKujiKiri(game.SkillKkZhen)), "Kuji-kiri seal: focus on endurance for faster movement regen.", 0, combat.PosStanding)
	cmdRegistry.Register("sha", wrapSkill(command.CmdKujiKiri(game.SkillKkSha)), "Kuji-kiri seal: heal your wounds.", 0, combat.PosStanding)
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
	// real C top-level names for two of wizutil's sub-actions (src/interpreter.c), stricter-gated than the wizutil meta-command itself.
	cmdRegistry.Register("reroll", wrapArgs(cmdReroll), "Reroll a player's ability scores.", LVL_GRGOD, 0)
	cmdRegistry.Register("unaffect", wrapArgs(cmdUnaffect), "Remove all spell affects from a player.", LVL_GOD, 0)
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
	cmdRegistry.Register("newbiegive", wrapArgs(cmdNewbie), "Give newbie equipment to a player.", LVL_IMMORT, 0)

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
	// "reallyquit" is src/interpreter.c's SCMD_REALLY_QUIT variant of do_quit — in the
	// original, plain "quit" only works from recall/temple rooms and "reallyquit" is
	// required elsewhere (and costs your equipment). This port's cmdQuit doesn't yet
	// implement that temple-gating/equipment-loss split, so for now both names behave
	// identically; aliasing at least makes the command reachable.
	cmdRegistry.Register("quit", wrapNoArgs(cmdQuit), "Quit the game.", 0, 0, "reallyquit")

	// Offensive commands — delegated to pkg/command (C-10: real damage formulas)
	cmdRegistry.Register("assist", wrapArgs(cmdAssist), "Assist a target in combat.", 0, combat.PosFighting)
	cmdRegistry.Register("disembowel", wrapSkill(command.CmdDisembowel), "Disembowel a target with a piercing weapon.", 0, combat.PosFighting, "gut")
	cmdRegistry.Register("dragonkick", wrapSkill(command.CmdDragonKick), "Dragon-style kick attack.", 0, combat.PosFighting, "dkick", "dragon")
	cmdRegistry.Register("tigerpunch", wrapSkill(command.CmdTigerPunch), "Tiger-style punch attack (bare hands).", 0, combat.PosFighting, "tpunch", "tiger")
	cmdRegistry.Register("shoot", wrapSkill(command.CmdShoot), "Shoot a target with a ranged weapon.", 0, combat.PosStanding)
	cmdRegistry.Register("subdue", wrapSkill(command.CmdSubdue), "Subdue a target (non-lethal).", 0, combat.PosStanding)
	cmdRegistry.Register("sleeper", wrapSkill(command.CmdSleeper), "Apply a sleeper hold to a target.", 0, combat.PosStanding)
	cmdRegistry.Register("neckbreak", wrapSkill(command.CmdNeckbreak), "Break a target's neck (bare hands).", 0, combat.PosStanding)
	cmdRegistry.Register("ambush", wrapSkill(command.CmdAmbush), "Ambush a target from hiding.", 0, combat.PosStanding)
	cmdRegistry.Register("parry", wrapSkill(command.CmdParry), "Toggle parry stance to deflect attacks.", 0, combat.PosStanding)

	// Port completion: skill handlers that were implemented in pkg/command but
	// never registered, leaving them unreachable by players. Positions/levels
	// mirror src/interpreter.c. See docs/port-reachability-map.md (Bucket A).
	cmdRegistry.Register("bearhug", wrapSkill(command.CmdBearhug), "Crush a target in a bear hug.", 0, combat.PosFighting)
	cmdRegistry.Register("behead", wrapSkill(command.CmdBehead), "Attempt to behead a target with a slashing weapon.", 0, combat.PosStanding)
	cmdRegistry.Register("bite", wrapSkill(command.CmdBite), "Bite a target.", 0, combat.PosResting)
	cmdRegistry.Register("carve", wrapSkill(command.CmdCarve), "Carve a corpse.", 0, combat.PosStanding)
	cmdRegistry.Register("compare", wrapSkill(command.CmdCompare), "Compare two items.", 0, combat.PosStanding)
	cmdRegistry.Register("cutthroat", wrapSkill(command.CmdCutthroat), "Slit a target's throat.", 0, combat.PosFighting)
	cmdRegistry.Register("disarm", wrapSkill(command.CmdDisarm), "Disarm a target's weapon.", 0, combat.PosFighting)
	cmdRegistry.Register("groinrip", wrapSkill(command.CmdGroinrip), "Rip a target's groin.", 0, combat.PosFighting)
	cmdRegistry.Register("mindlink", wrapSkill(command.CmdMindlink), "Form a psychic mind link.", 0, combat.PosStanding)
	cmdRegistry.Register("palm", wrapSkill(command.CmdPalm), "Palm an item discreetly.", 0, combat.PosStanding)
	cmdRegistry.Register("point", wrapSkill(command.CmdPoint), "Point out something.", 0, combat.PosResting)
	cmdRegistry.Register("scrounge", wrapSkill(command.CmdScrounge), "Scrounge for useful items.", 0, combat.PosStanding)
	cmdRegistry.Register("sharpen", wrapSkill(command.CmdSharpen), "Sharpen a bladed weapon.", 0, combat.PosResting)
	cmdRegistry.Register("slug", wrapSkill(command.CmdSlug), "Slug a target with a heavy blow.", 0, combat.PosFighting)
	cmdRegistry.Register("smackheads", wrapSkill(command.CmdSmackheads), "Smack two targets' heads together.", 0, combat.PosFighting)
	cmdRegistry.Register("strike", wrapSkill(command.CmdStrike), "Strike a target with a focused blow.", 0, combat.PosFighting)
	cmdRegistry.Register("tag", wrapSkill(command.CmdTag), "Tag a target.", 0, combat.PosResting)
	cmdRegistry.Register("turn", wrapSkill(command.CmdTurn), "Turn undead.", 0, combat.PosStanding)
	cmdRegistry.Register("aid", wrapSkill(command.CmdFirstAid), "Administer first aid to a target.", 0, combat.PosStanding)
	cmdRegistry.Register("alter", wrapSkill(command.CmdFleshAlter), "Alter flesh.", 0, combat.PosFighting, "flesh")
	cmdRegistry.Register("serpent", wrapSkill(command.CmdSerpentKick), "Serpent-style kick attack.", 0, combat.PosFighting)
	cmdRegistry.Register("scan", wrapSkill(command.CmdScan), "Scan adjacent rooms for creatures.", 0, combat.PosResting)

	cmdRegistry.Register("order", wrapArgs(cmdOrder), "Order a pet or follower.", 0, 0)

	// Informative commands (act_informative.go)
	cmdRegistry.Register("color", wrapArgs(cmdColor), "Toggle ANSI color.", 0, 0)
	cmdRegistry.Register("commands", wrapArgs(cmdCommands), "List available commands.", 0, 0, "cmds")
	cmdRegistry.Register("description", wrapArgs(cmdDescription), "Set your character description.", 0, 0)
	// "glance" is src/interpreter.c's other top-level name for do_diagnose — identical handler.
	cmdRegistry.Register("diagnose", wrapArgs(cmdDiagnose), "Diagnose health status of a target.", 0, 0, "diag", "glance")
	cmdRegistry.Register("toggle", wrapArgs(cmdToggle), "Toggle a player preference.", 0, 0)
	cmdRegistry.Register("lines", wrapArgs(cmdLines), "Set your screen line count (7-50).", 0, 0)
	cmdRegistry.Register("infobar", wrapArgs(cmdInfoBar), "Toggle the bottom status infobar.", 0, 0)
	cmdRegistry.Register("users", wrapArgs(cmdUsersSafe), "Show connected players.", LVL_IMMORT, 0)

	// Other commands (act_other.go)
	cmdRegistry.Register("save", wrapArgs(cmdSave), "Save your character.", 0, 0)
	cmdRegistry.Register("report", wrapArgs(cmdReport), "Show report of your surroundings.", 0, 0)
	cmdRegistry.Register("split", wrapArgs(cmdSplit), "Split gold with your group.", 0, 0)
	cmdRegistry.Register("wimpy", wrapArgs(cmdWimpy), "Set your wimpy threshold.", 0, 0)
	cmdRegistry.Register("display", wrapArgs(cmdDisplay), "Set display preferences.", 0, 0)
	cmdRegistry.Register("transform", wrapArgs(cmdTransform), "Transform your appearance.", 0, 0)
	// "mount" is src/interpreter.c's other top-level name for do_ride — identical handler, same subcmd.
	cmdRegistry.Register("ride", wrapArgs(cmdRide), "Ride a mount.", 0, 0, "mount")
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
	// Preference toggles (act.other.c do_gen_tog) — each is its own top-level
	// command in the original C, not a unified dispatcher; src/interpreter.c
	// lines 366-666.
	cmdRegistry.Register("nosummon", wrapToggle("nosummon"), "Toggle summon protection.", 0, 0)
	cmdRegistry.Register("nohassle", wrapToggle("nohassle"), "Toggle nohassle mode.", LVL_IMMORT, 0)
	cmdRegistry.Register("brief", wrapToggle("brief"), "Toggle brief room descriptions.", 0, 0)
	cmdRegistry.Register("compact", wrapToggle("compact"), "Toggle compact display mode.", 0, 0)
	cmdRegistry.Register("notell", wrapToggle("notell"), "Toggle deafness to tells.", 0, 0)
	cmdRegistry.Register("noauction", wrapToggle("noauction"), "Toggle deafness to auctions.", 0, 0)
	cmdRegistry.Register("noshout", wrapToggle("noshout"), "Toggle deafness to shouts.", 0, combat.PosSleeping)
	cmdRegistry.Register("nogossip", wrapToggle("nogossip"), "Toggle deafness to gossip.", 0, 0)
	cmdRegistry.Register("nograts", wrapToggle("nograts"), "Toggle congratulation messages.", 0, 0)
	cmdRegistry.Register("nowiz", wrapToggle("nowiz"), "Toggle deafness to the wiz channel.", LVL_IMMORT, 0)
	cmdRegistry.Register("quest", wrapToggle("quest"), "Toggle quest announcements.", 0, 0)
	cmdRegistry.Register("roomflags", wrapToggle("roomflags"), "Toggle room flag display.", LVL_IMMORT, 0)
	cmdRegistry.Register("norepeat", wrapToggle("norepeat"), "Toggle communication echo.", 0, 0)
	cmdRegistry.Register("holylight", wrapToggle("holylight"), "Toggle holylight mode.", LVL_IMMORT, 0)
	cmdRegistry.Register("nonewbie", wrapToggle("nonewbie"), "Toggle newbie channel.", 0, 0)
	cmdRegistry.Register("noctell", wrapToggle("noctell"), "Toggle deafness to clan tells.", 0, 0)
	cmdRegistry.Register("nobroadcast", wrapToggle("nobroadcast"), "Toggle deafness to broadcasts.", 0, 0)
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
	cmdRegistry.Register("auction", wrapArgs(cmdAuction), "Auction an item to the channel.", 0, 0)
	cmdRegistry.Register("gratz", wrapArgs(cmdGratz), "Congratulate someone on the channel.", 0, 0)
	cmdRegistry.Register("newbie", wrapArgs(cmdNewbieChannel), "Ask a question on the newbie channel.", 0, 0)
	cmdRegistry.Register("ctell", wrapArgs(cmdCTell), "Send a message to your clan.", 0, 0)
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

// wrapToggle adapts a named player-preference toggle (brief, compact, notell,
// etc. — src/interpreter.c each registers do_gen_tog under its own command
// name, not a unified "toggle <name>" dispatcher) to command.Handler.
func wrapToggle(key string) command.Handler {
	return func(s common.CommandSession, args []string) error {
		sess := s.(*commandSession).Session
		sess.manager.world.ExecGenTog(sess.player, key)
		return nil
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
	// Moderation pre-check: mute, ban
	if s.manager.modChecker != nil && s.player != nil {
		errMsg, reject := s.manager.modChecker.CheckPreCommand(s.player.Name, cmdStr)
		if reject {
			s.sendText(errMsg)
			return nil
		}
	}
	// Split command from arguments if args not provided separately
	if len(args) == 0 {
		if idx := strings.IndexByte(cmdStr, ' '); idx >= 0 {
			args = strings.Fields(cmdStr[idx+1:])
			cmdStr = cmdStr[:idx]
		}
	}
	cmd := strings.ToLower(cmdStr)

	// Expand player aliases before command dispatch (DP-415)
	if s.player != nil && len(s.player.Aliases) > 0 {
		fullInput := cmd
		if len(args) > 0 {
			fullInput = cmd + " " + strings.Join(args, " ")
		}
		if expanded, ok := game.PerformAlias(s.player.Aliases, fullInput); ok {
			expanded = strings.TrimSpace(expanded)
			if idx := strings.IndexByte(expanded, ' '); idx >= 0 {
				cmd = strings.ToLower(expanded[:idx])
				cmdStr = expanded[:idx]
				args = strings.Fields(expanded[idx+1:])
			} else {
				cmd = strings.ToLower(expanded)
				cmdStr = expanded
				args = nil
			}
		}
	}

	if s.isGuest {
		guestAllowedCmds := map[string]bool{
			"north": true, "n": true,
			"east": true, "e": true,
			"south": true, "s": true,
			"west": true, "w": true,
			"up": true, "u": true,
			"down": true, "d": true,
			"look": true, "l": true,
			"examine": true, "exa": true,
			"score": true, "sc": true,
			"who": true, "where": true,
			"affects": true,
			"help":    true,
			"say":     true,
			"gossip":  true,
			"tell":    true,
			"reply":   true, "r": true,
			"newbie": true,
			"shout":  true,
			"gtell":  true, "gsay": true,
			"emote": true, "me": true,
			"stand": true, "sit": true,
			"rest": true, "sleep": true,
			"wake": true,
			"quit": true,
		}
		if !guestAllowedCmds[cmd] {
			s.sendText("Guest accounts are restricted from using this command.")
			return nil
		}
	}

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

	// Spec procedure command interception
	if s.player != nil && s.player.GetRoomVNum() > 0 {
		roomVNum := s.player.GetRoomVNum()
		argStr := strings.Join(args, " ")

		// 1. Mob spec procedures
		mobs := s.manager.world.GetMobsInRoom(roomVNum)
		for _, mob := range mobs {
			if mob != nil {
				if mobSpec := game.GetMobSpec(mob.VNum); mobSpec != nil {
					if mobSpec(s.manager.world, s.player, mob, cmd, argStr) {
						return nil
					}
				}
			}
		}

		// 2. Room spec procedure
		if roomSpec := game.GetRoomSpec(roomVNum); roomSpec != nil {
			if roomSpec(s.manager.world, s.player, nil, cmd, argStr) {
				return nil
			}
		}

		// 2. Object spec procedures
		// 2a. Equipped items
		if s.player.Equipment != nil {
			equipped := s.player.Equipment.GetEquippedItems()
			for _, item := range equipped {
				if item != nil {
					if objSpec := game.GetObjSpec(item.VNum); objSpec != nil {
						if objSpec(s.manager.world, s.player, nil, cmd, argStr) {
							return nil
						}
					}
				}
			}
		}

		// 2b. Inventory items
		if s.player.Inventory != nil {
			invItems := s.player.Inventory.FindItems("")
			for _, item := range invItems {
				if item != nil {
					if objSpec := game.GetObjSpec(item.VNum); objSpec != nil {
						if objSpec(s.manager.world, s.player, nil, cmd, argStr) {
							return nil
						}
					}
				}
			}
		}

		// 2c. Room items
		roomItems := s.manager.world.GetItemsInRoom(roomVNum)
		for _, item := range roomItems {
			if item != nil {
				if objSpec := game.GetObjSpec(item.VNum); objSpec != nil {
					if objSpec(s.manager.world, s.player, nil, cmd, argStr) {
						return nil
					}
				}
			}
		}
	}

	entry, ok := cmdRegistry.Lookup(cmd)
	if !ok {
		// Check social emotes before giving up
		if _, found := game.Socials[cmd]; found {
			game.DoAction(s.manager.world, s.player, cmd, strings.Join(args, " "))
			return nil
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

	// C-10: WAIT_STATE enforcement — combat skills set cooldowns.
	// Non-combat informational commands bypass the wait so players can still
	// look, check inventory, and communicate while their attack is pending.
	waitBypass := map[string]bool{
		"look": true, "l": true,
		"inventory": true, "inv": true, "i": true,
		"equipment": true, "eq": true,
		"score": true, "sc": true,
		"say": true, "'": true,
		"tell":    true,
		"who":     true,
		"time":    true,
		"weather": true,
		"help":    true,
		"exits":   true,
		"quit":    true,
	}
	if s.player != nil && s.player.GetWaitState() > 0 && !waitBypass[cmd] {
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

// cmdUse handles using an item (wand/staff/potion/scroll) or falls back to using a skill.
func cmdUse(s *Session, args []string) error {
	if s.player == nil {
		return fmt.Errorf("not logged in")
	}
	if len(args) == 0 {
		s.sendText("Use what? Usage: use <item> [target] OR use <skill> [target]\r\n")
		return nil
	}

	itemArg := args[0]
	var item *game.ObjectInstance
	if s.player.Inventory != nil {
		item, _ = s.player.Inventory.FindItem(itemArg)
	}
	if item == nil && s.player.Equipment != nil {
		equipped := s.player.Equipment.GetEquippedItems()
		for _, eqItem := range equipped {
			if eqItem != nil && (strings.Contains(strings.ToLower(eqItem.GetKeywords()), strings.ToLower(itemArg)) || strings.Contains(strings.ToLower(eqItem.GetShortDesc()), strings.ToLower(itemArg))) {
				item = eqItem
				break
			}
		}
	}

	if item != nil {
		itemType := item.GetTypeFlag()
		if itemType == game.ITEM_WAND || itemType == game.ITEM_STAFF || itemType == game.ITEM_POTION || itemType == game.ITEM_SCROLL {
			argStr := strings.Join(args, " ")
			s.manager.world.DoUse(s.player, argStr)
			return nil
		}
	}

	return command.CmdUseSkill(s, args)
}

// cmdSave saves the player's character.
