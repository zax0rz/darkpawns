package session

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/command"
	"github.com/zax0rz/darkpawns/pkg/common"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// commandNumber is a test seam for verifying command_interpreter draw parity.
// Production always uses the process-wide deterministic stream.
var commandNumber = dprng.Number

// positionFailMessage returns an appropriate rejection message when
// the player's position is too low for a command.
func positionFailMessage(pos int) string {
	switch pos {
	case combat.PosDead:
		return "Lie still; you are DEAD!!! :-(\r\n"
	case combat.PosMortally, combat.PosIncap:
		return "You are in a pretty bad shape, unable to do anything!\r\n"
	case combat.PosStunned:
		return "All you can do right now is think about the stars!\r\n"
	case combat.PosSleeping:
		return "In your dreams, or what?\r\n"
	case combat.PosResting:
		return "Nah... You feel too relaxed to do that..\r\n"
	case combat.PosSitting:
		return "Maybe you should get on your feet first?\r\n"
	case combat.PosFighting:
		return "No way!  You're fighting for your life!\r\n"
	default:
		return "You are in no position to do that!\r\n"
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
	registerCommand("north", wrapMove("north"), "Move north.", "n")
	registerCommand("east", wrapMove("east"), "Move east.", "e")
	registerCommand("south", wrapMove("south"), "Move south.", "s")
	registerCommand("west", wrapMove("west"), "Move west.", "w")
	registerCommand("up", wrapMove("up"), "Move up.", "u")
	registerCommand("down", wrapMove("down"), "Move down.", "d")
	registerCommand("enter", wrapArgs(cmdEnter), "Enter a nearby doorway or indoor area.")
	registerCommand("leave", wrapNoArgs(cmdLeave), "Leave for the outdoors.")

	// Look
	registerCommand("look", wrapArgs(cmdLook), "Look around the room.", "l")
	registerCommand("read", wrapArgs(cmdRead), "Read a nearby object or room feature.")

	// Communication
	registerCommand("say", wrapArgs(cmdSay), "Say something to the room.")
	registerCommand("'", wrapArgs(cmdSay), "Say something to the room.")
	registerCommand("rsay", wrapArgs(cmdRaceSay), "Say something in your racial tongue.")
	registerCommand("tell", wrapArgs(cmdTell), "Send a private message to a player.")
	registerCommand("emote", wrapArgs(cmdEmote), "Perform a roleplay action.", "me")
	registerCommand(":", wrapArgs(cmdEmote), "Perform a roleplay action.")
	registerCommand("shout", wrapArgs(cmdShout), "Shout to everyone in your zone.")
	registerCommand("holler", wrapArgs(cmdHoller), "Holler to everyone in the world.")
	registerCommand("gtell", wrapArgs(cmdGtell), "Send a message to your group.", "gsay")
	registerCommand("think", wrapArgs(cmdThink), "Think a thought, optionally aloud.")
	registerCommand("insult", wrapArgs(cmdInsult), "Insult a target in the room.")
	registerCommand("dream", wrapArgs(cmdDream), "Dream — only does anything while asleep.")

	// Combat
	registerCommand("hit", wrapArgs(cmdHit), "Attack a target.", "attack")
	registerCommand("murder", wrapArgs(cmdHit), "Attack a target (C alias of hit).")
	registerCommand("kill", wrapArgs(cmdKill), "Kill a target (immortal instakill).")
	registerCommand("flee", wrapNoArgs(cmdFlee), "Attempt to flee from combat.")
	registerCommand("escape", wrapNoArgs(cmdRetreat), "Attempt to escape from combat.")

	// Position / Movement
	registerCommand("stand", wrapNoArgs(cmdStand), "Stand up.")
	registerCommand("sit", wrapNoArgs(cmdSit), "Sit down.")
	registerCommand("rest", wrapNoArgs(cmdRest), "Rest.")
	registerCommand("sleep", wrapNoArgs(cmdSleep), "Go to sleep.")
	registerCommand("wake", wrapArgs(cmdWake), "Wake up or wake someone else.")

	// Items
	registerCommand("inventory", wrapArgs(cmdInventory), "Show your inventory.", "i", "inv")
	registerCommand("equipment", wrapArgs(cmdEquipment), "Show your equipped items.", "eq")
	registerCommand("wear", wrapArgs(cmdWear), "Wear an item from your inventory.")
	registerCommand("remove", wrapArgs(cmdRemove), "Remove an equipped item.")
	registerCommand("wield", wrapArgs(cmdWield), "Wield a weapon.")
	registerCommand("hold", wrapArgs(cmdHold), "Hold an item.")
	// C gives grab and hold distinct level gates despite sharing do_grab.
	registerCommand("grab", wrapArgs(cmdHold), "Hold an item.")
	registerCommand("get", wrapArgs(cmdGet), "Pick up an item from the room, container, or corpse.", "take")
	registerCommand("give", wrapArgs(cmdGive), "Give an item or gold to another character.")
	registerCommand("put", wrapArgs(cmdPut), "Put an item into a container.")
	registerCommand("drop", wrapArgs(cmdDrop), "Drop an item from your inventory.")
	registerCommand("junk", wrapArgs(cmdJunk), "Destroy an item for a small experience reward.")
	registerCommand("donate", wrapArgs(cmdDonate), "Donate an item to the donation room.")
	registerCommand("eat", wrapArgs(cmdEat), "Eat some food.")
	registerCommand("taste", wrapArgs(cmdTaste), "Nibble a little bit of some food.")
	registerCommand("drink", wrapArgs(cmdDrink), "Drink from a container.")
	registerCommand("sip", wrapArgs(cmdSip), "Sip from a container without drinking it.")
	registerCommand("pour", wrapArgs(cmdPour), "Pour liquid from one container to another.")
	registerCommand("fill", wrapArgs(cmdFill), "Fill a container from a fountain.")
	registerCommand("quaff", wrapArgs(cmdQuaff), "Quaff a potion.", "q")

	// Info
	registerCommand("score", wrapNoArgs(cmdScore), "Show your character stats.", "sc")
	registerCommand("who", wrapArgs(cmdWho), "List all online players.")
	registerCommand("where", wrapArgs(cmdWhere), "Show player locations.")
	registerCommand("coins", wrapNoArgs(cmdCoins), "Display your gold and bank balance.")
	registerCommand("gold", wrapNoArgs(cmdCoins), "Display your gold and bank balance (C alias of coins).")
	registerCommand("abilities", wrapNoArgs(cmdAbils), "Show your ability scores.")
	registerCommand("levels", wrapNoArgs(cmdLevels), "Show XP table for your class.")
	registerCommand("review", wrapNoArgs(cmdReview), "Show recent gossip history.")
	registerCommand("whois", wrapArgs(cmdWhois), "Look up a player's info.")
	registerCommand("finger", wrapArgs(cmdWhois), "Look up a player's info (C alias of whois).")
	registerCommand("help", wrapArgs(cmdHelp), "Show available commands or help for a topic.")
	registerCommand("?", wrapArgs(cmdHelp), "Show available commands or help for a topic.")
	registerCommand("credits", wrapArgs(cmdCredits), "Show who built this game.")
	registerCommand("news", wrapArgs(cmdNews), "Show current game news.")
	registerCommand("info", wrapArgs(cmdInfoText), "Show game information.")
	registerCommand("policy", wrapArgs(cmdPolicy), "Show the game's policies.")
	registerCommand("handbook", wrapArgs(cmdHandbook), "Show the immortal handbook.")
	registerCommand("future", wrapArgs(cmdFuture), "Show planned future content.")
	registerCommand("motd", wrapArgs(cmdMotd), "Show the message of the day.")
	registerCommand("imotd", wrapArgs(cmdImotd), "Show the immortal MOTD.")
	registerCommand("wizlist", wrapArgs(cmdWizlist), "Show the list of wizards.")
	registerCommand("immlist", wrapArgs(cmdImmlist), "Show the list of immortals.")
	registerCommand("players", wrapArgs(cmdPlayers), "Show all registered players.")
	registerCommand("clear", wrapArgs(cmdClear), "Clear the screen.", "cls")
	registerCommand("whoami", wrapArgs(cmdWhoami), "Show your own name.")
	registerCommand("version", wrapArgs(cmdVersion), "Show the game version.")

	// Group
	registerCommand("follow", wrapArgs(cmdFollow), "Follow another player.")
	registerCommand("shadow", wrapArgs(cmdShadow), "Follow another player quietly (shadow skill).")
	registerCommand("group", wrapArgs(cmdGroup), "Manage your group.", "party")
	registerCommand("ungroup", wrapArgs(cmdUngroup), "Disband or leave a group.", "disband")

	// Skills — Dark Pawns has exactly ONE skill command: `practice`
	// (src/act.other.c do_practice; interpreter.c:618). `skills`/`spells`/`learn`/
	// `forget`/`listskills` were Go inventions (retired: DP-1116/1128/1129). No-arg
	// `practice` lists the catalog; a named arg directs to the guild.
	registerCommand("practice", wrapArgs(cmdPractice), "List your skills / practice at your guild.")

	// Shop
	registerCommand("list", wrapArgs(cmdList), "List items for sale at a shop.")
	registerCommand("buy", wrapArgs(cmdBuy), "Buy an item from a shop.")
	registerCommand("sell", wrapArgs(cmdSell), "Sell an item to a shop.")
	// C registers these names under do_not_here (act.other.c:208). Room
	// special procedures intercept them first; these handlers preserve C's
	// generic fallback when no matching special procedure is present.
	for _, name := range []string{
		"balance", "check", "collect", "deposit", "hire", "mail",
	} {
		registerCommand(name, wrapArgs(cmdNotHere), "Unavailable outside its special procedure.")
	}
	registerCommand("offer", wrapArgs(cmdNotHere), "Unavailable outside its special procedure.")
	for _, name := range []string{
		"recharge", "receive", "remort", "rent", "retrieve", "stable", "value",
		"withdraw",
	} {
		registerCommand(name, wrapArgs(cmdNotHere), "Unavailable outside its special procedure.")
	}
	registerCommand("use", wrapArgs(cmdUse), "Use a wand/staff or a skill.")
	registerCommand("skillinfo", wrapSkill(command.CmdSkillInfo), "Show info about a skill.", "sinfo")

	// Combat skills (delegated to pkg/command)
	registerCommand("backstab", wrapSkill(command.CmdBackstab), "Backstab a target with a piercing weapon.", "bs")
	registerCommand("spike", wrapSkill(command.CmdSpike), "Spike a werewolf with a spiked weapon.")
	registerCommand("stake", wrapSkill(command.CmdStake), "Stake a vampire with a wooden stake.")
	registerCommand("bash", wrapSkill(command.CmdBash), "Bash a target, potentially stunning them.")
	registerCommand("circle", wrapSkill(command.CmdCircle), "Circle behind a target for a piercing attack.")
	registerCommand("charge", wrapSkill(command.CmdCharge), "Charge a target with a sword or lance.")
	registerCommand("kick", wrapSkill(command.CmdKick), "Kick a target for damage.")
	registerCommand("trip", wrapSkill(command.CmdTrip), "Trip a target, knocking them down.")
	registerCommand("headbutt", wrapSkill(command.CmdHeadbutt), "Headbutt a target for high damage.")
	registerCommand("rescue", wrapSkill(command.CmdRescue), "Rescue someone from combat.")
	registerCommand("sneak", wrapSkill(command.CmdSneak), "Attempt to move silently.")
	registerCommand("hide", wrapSkill(command.CmdHide), "Attempt to hide in the shadows.")
	registerCommand("kabuki", wrapSkill(command.CmdKabuki), "Practice the art of kabuki (hide variant).")
	// C do_dig (src/new_cmds2.c:818) is the LVL_BUILDER OLC exit-creator. The
	// unrelated mortal foraging skill remains available only through its game
	// layer API; the command name belongs to the C OLC surface.
	registerCommand("dig", wrapArgs(cmdDig), "Create a room exit.")
	registerCommand("steal", wrapSkill(command.CmdSteal), "Steal from a target.")
	registerCommand("berserk", wrapSkill(command.CmdBerserk), "Summon your battle rage for a hitroll/damroll boost.")
	registerCommand("rin", wrapSkill(command.CmdKujiKiri(game.SkillKkRin)), "Kuji-kiri seal: harden body for an AC bonus and metalskin.")
	registerCommand("kyo", wrapSkill(command.CmdKujiKiri(game.SkillKkKyo)), "Kuji-kiri seal: focus battle rage for a hitroll bonus.")
	registerCommand("toh", wrapSkill(command.CmdKujiKiri(game.SkillKkToh)), "Kuji-kiri seal: focus inner strength for a damroll/AC bonus.")
	registerCommand("kai", wrapSkill(command.CmdKujiKiri(game.SkillKkKai)), "Kuji-kiri seal: fortify your body, lowering damroll/AC.")
	registerCommand("jin", wrapSkill(command.CmdKujiKiri(game.SkillKkJin)), "Kuji-kiri seal: focus on recuperation for faster HP regen.")
	registerCommand("retsu", wrapSkill(command.CmdKujiKiri(game.SkillKkRetsu)), "Kuji-kiri seal: attempt to teleport away.")
	registerCommand("zai", wrapSkill(command.CmdKujiKiri(game.SkillKkZai)), "Kuji-kiri seal: fade from view.")
	registerCommand("zhen", wrapSkill(command.CmdKujiKiri(game.SkillKkZhen)), "Kuji-kiri seal: focus on endurance for faster movement regen.")
	registerCommand("sha", wrapSkill(command.CmdKujiKiri(game.SkillKkSha)), "Kuji-kiri seal: heal your wounds.")
	registerCommand("pick", wrapArgs(cmdPick), "Pick a lock on a container or exit.", "pick lock")

	// Admin / debug
	registerCommand("summon", wrapArgs(cmdSummon), "Summon a player to your room.")

	// Doors
	registerCommand("open", wrapArgs(cmdOpen), "Open a container or exit.")
	registerCommand("close", wrapArgs(cmdClose), "Close a container or exit.")
	registerCommand("lock", wrapArgs(cmdLock), "Lock a container or exit with its key.")
	registerCommand("unlock", wrapArgs(cmdUnlock), "Unlock a container or exit with its key.")
	registerCommand("knock", wrapArgs(cmdKnock), "Knock on a door: knock <north|south|east|west|up|down>")

	// Wizard commands
	registerCommand("goto", wrapArgs(cmdGoto), "Teleport to a room by VNum.")
	registerCommand("at", wrapArgs(cmdAt), "Execute a command at another room.")
	registerCommand("load", wrapArgs(cmdLoad), "Load a mob or object by VNum.")
	registerCommand("purge", wrapArgs(cmdPurge), "Remove all mobs/items from a room.")
	registerCommand("teleport", wrapArgs(cmdTeleport), "Teleport another player to a room.")
	registerCommand("heal", wrapArgs(cmdHeal), "Fully heal a target.")
	registerCommand("restore", wrapArgs(cmdRestore), "Restore all stats of a target.")
	registerCommand("set", wrapArgs(cmdSet), "Set character fields.")
	registerCommand("switch", wrapArgs(cmdSwitch), "Enter another character's body.")
	registerCommand("return", wrapArgs(cmdReturn), "Return from switched body.")
	registerCommand("invis", wrapArgs(cmdInvis), "Become invisible to players.")
	registerCommand("vis", wrapArgs(cmdVis), "Become visible again.")
	registerCommand("gecho", wrapArgs(cmdGecho), "Echo a message to all players.")
	registerCommand("echo", wrapArgs(cmdEcho), "Echo a message to the room.")
	registerCommand("send", wrapArgs(cmdSend), "Send a message to another character.")
	registerCommand("force", wrapArgs(cmdForce), "Force a command on another character.")
	registerCommand("shutdown", wrapArgs(cmdShutdown), "Shutdown the server.")
	registerCommand("snoop", wrapArgs(cmdSnoop), "Spy on a player's input.")
	registerCommand("admobs", wrapNoArgs(cmdAdmobs), "Adjust all mob prototypes.")
	registerCommand("advance", wrapArgs(cmdAdvance), "Advance a player's level.")
	registerCommand("skillset", wrapArgs(cmdSkillset), "Set a player's skill value.")
	registerCommand("reload", wrapArgs(cmdReload), "Reload world data.")

	// Wizard — stat/info
	registerCommand("stat", wrapArgs(cmdStat), "Inspect a character, room, or object.")
	registerCommand("vnum", wrapArgs(cmdVnum), "Search for vnums by keyword.")
	registerCommand("vstat", wrapArgs(cmdVstat), "Show detailed prototype info by vnum.")
	registerCommand("wizlock", wrapArgs(cmdWizlock), "Toggle wizard-only login.")
	registerCommand("dc", wrapArgs(cmdDc), "Disconnect a player.")
	registerCommand("home", wrapArgs(cmdHome), "Teleport to home room or specified room.")
	registerCommand("date", wrapArgs(cmdDate), "Show current system time or uptime.")
	registerCommand("uptime", wrapArgs(cmdUptime), "Show server uptime.")
	registerCommand("last", wrapArgs(cmdLast), "Show last login info for a player.")
	registerCommand("wizutil", wrapArgs(cmdWizutil), "Player utility commands (reroll/pardon/notitle/squelch/freeze/thaw/unaffect).")
	// real C top-level names for wizutil's sub-actions (src/interpreter.c), stricter-gated than the wizutil meta-command itself.
	registerCommand("reroll", wrapArgs(cmdReroll), "Reroll a player's ability scores.")
	registerCommand("unaffect", wrapArgs(cmdUnaffect), "Remove all spell affects from a player.")
	registerCommand("freeze", wrapArgs(cmdFreeze), "Freeze a player.")
	registerCommand("thaw", wrapArgs(cmdThaw), "Thaw a frozen player.")
	registerCommand("pardon", wrapArgs(cmdPardon), "Pardon a player's outlaw flag.")
	registerCommand("notitle", wrapArgs(cmdNotitle), "Toggle a player's notitle flag.")
	registerCommand("mute", wrapArgs(cmdMute), "Toggle a player's squelch (PLR_NOSHOUT) flag.")
	registerCommand("show", wrapArgs(cmdShow), "Show system info (players/uptime/stats/reset).")
	registerCommand("dark", wrapArgs(cmdDark), "Stop combat in the current room.")
	registerCommand("syslog", wrapArgs(cmdSyslog), "Toggle system logging level.")
	registerCommand("dns", wrapArgs(cmdDns), "Manage the DNS cache.")
	registerCommand("idlist", wrapArgs(cmdIdlist), "Dump object ID list to file.")
	registerCommand("checkload", wrapArgs(cmdCheckload), "Check zone load info for a mob/obj.")
	registerCommand("poofset", wrapArgs(cmdPoofset), "Set poof in/out messages.")
	registerCommand("poofin", wrapArgs(cmdPoofin), "Set your poof-in message.")
	registerCommand("poofout", wrapArgs(cmdPoofout), "Set your poof-out message.")
	registerCommand("wiznet", wrapArgs(cmdWiznet), "Send message on wizard net.")
	registerCommand(";", wrapArgs(cmdWiznet), "Send message on wizard net.")
	registerCommand("zreset", wrapArgs(cmdZreset), "Reset a zone by number.")
	registerCommand("zlist", wrapArgs(cmdZlist), "List zones matching a filter.")
	registerCommand("rlist", wrapArgs(cmdRlist), "List rooms matching a keyword.")
	registerCommand("olist", wrapArgs(cmdOlist), "List objects matching a keyword.")
	registerCommand("mlist", wrapArgs(cmdMlist), "List mobiles matching a keyword.")
	registerCommand("sysfile", wrapArgs(cmdSysfile), "Show system file path.")
	registerCommand("sethunt", wrapArgs(cmdSethunt), "Set hunt target for a character.")
	registerCommand("tick", wrapArgs(cmdTick), "Show current tick info.")
	registerCommand("newbiegive", wrapArgs(cmdNewbie), "Give newbie equipment to a player.")
	registerCommand("wnewbie", wrapArgs(cmdNewbie), "Give newbie equipment to a player (C name).")

	// Informative
	registerCommand("consider", wrapArgs(cmdConsider), "Compare yourself to a target.", "con")
	registerCommand("examine", wrapArgs(cmdExamine), "Examine something in detail.", "exa")
	registerCommand("exits", wrapArgs(cmdExits), "List obvious exits.")
	registerCommand("time", wrapArgs(cmdTime), "Show the current time.")
	registerCommand("weather", wrapArgs(cmdWeather), "Show the current weather.")
	registerCommand("affects", wrapArgs(cmdAffects), "Show active affects.")
	registerCommand("autoexit", wrapArgs(cmdAutoExit), "Toggle auto-exit display.")
	registerCommand("title", wrapArgs(cmdTitle), "Set your title.")

	// Quit — two explicit entries mirroring C's SCMD_QUIT / SCMD_REALLY_QUIT
	// subcmds of do_quit (src/interpreter.c:630,657): quit is safe-room gated
	// and keeps equipment; reallyquit logs out anywhere but loses equipment
	// outside a safe room. Both delegate to one game-owned logout op.
	registerCommand("quit", wrapNoArgs(cmdQuit), "Quit the game.")
	// C abbreviation stubs (interpreter.c:629, :698): the table entries that
	// force exact typing. Their refusal messages are player-facing surface.
	registerCommand("qui", wrapNoArgs(cmdQuiStub), "You have to type quit in full.")
	registerCommand("shutdow", wrapNoArgs(cmdShutdowStub), "Type shutdown in full.")
	registerCommand("reallyquit", wrapNoArgs(cmdReallyQuit), "Quit the game, losing your equipment.")

	// Offensive commands — delegated to pkg/command (C-10: real damage formulas)
	registerCommand("assist", wrapArgs(cmdAssist), "Assist a target in combat.")
	registerCommand("disembowel", wrapSkill(command.CmdDisembowel), "Disembowel a target with a piercing weapon.", "gut")
	registerCommand("dragonkick", wrapSkill(command.CmdDragonKick), "Dragon-style kick attack.", "dkick", "dragon")
	registerCommand("tigerpunch", wrapSkill(command.CmdTigerPunch), "Tiger-style punch attack (bare hands).", "tpunch", "tiger")
	registerCommand("shoot", wrapSkill(command.CmdShoot), "Shoot a target with a ranged weapon.")
	registerCommand("subdue", wrapSkill(command.CmdSubdue), "Subdue a target (non-lethal).")
	registerCommand("sleeper", wrapSkill(command.CmdSleeper), "Apply a sleeper hold to a target.")
	registerCommand("neckbreak", wrapSkill(command.CmdNeckbreak), "Break a target's neck (bare hands).")
	registerCommand("ambush", wrapSkill(command.CmdAmbush), "Ambush a target from hiding.")

	// Port completion: skill handlers that were implemented in pkg/command but
	// never registered, leaving them unreachable by players. Positions/levels
	// mirror src/interpreter.c. See docs/port-reachability-map.md (Bucket A).
	registerCommand("bearhug", wrapSkill(command.CmdBearhug), "Crush a target in a bear hug.")
	registerCommand("behead", wrapSkill(command.CmdBehead), "Attempt to behead a target with a slashing weapon.")
	registerCommand("bite", wrapSkill(command.CmdBite), "Bite a target.")
	registerCommand("carve", wrapSkill(command.CmdCarve), "Carve a corpse.")
	registerCommand("compare", wrapSkill(command.CmdCompare), "Compare two items.")
	registerCommand("cutthroat", wrapSkill(command.CmdCutthroat), "Slit a target's throat.")
	registerCommand("search", wrapSkill(command.CmdDetect), "Search for hidden exits.")
	registerCommand("detect", wrapSkill(command.CmdDetect), "Detect hidden exits (alias for search).")
	registerCommand("disarm", wrapSkill(command.CmdDisarm), "Disarm a target's weapon.")
	registerCommand("groinrip", wrapSkill(command.CmdGroinrip), "Rip a target's groin.")
	registerCommand("mindlink", wrapSkill(command.CmdMindlink), "Form a psychic mind link.")
	registerCommand("mold", wrapSkill(command.CmdMold), "Mold a clay item.")
	registerCommand("palm", wrapSkill(command.CmdPalm), "Palm an item discreetly.")
	registerCommand("point", wrapSkill(command.CmdPoint), "Point out something.")
	registerCommand("scrounge", wrapSkill(command.CmdScrounge), "Scrounge for useful items.")
	registerCommand("sharpen", wrapSkill(command.CmdSharpen), "Sharpen a bladed weapon.")
	registerCommand("slug", wrapSkill(command.CmdSlug), "Slug a target with a heavy blow.")
	registerCommand("smackheads", wrapSkill(command.CmdSmackheads), "Smack two targets' heads together.")
	registerCommand("strike", wrapSkill(command.CmdStrike), "Strike a target with a focused blow.")
	registerCommand("tag", wrapSkill(command.CmdTag), "Tag a target.")
	registerCommand("turn", wrapSkill(command.CmdTurn), "Turn undead.")
	registerCommand("aid", wrapSkill(command.CmdFirstAid), "Administer first aid to a target.")
	registerCommand("alter", wrapSkill(command.CmdFleshAlter), "Alter flesh.", "flesh")
	registerCommand("serpent", wrapSkill(command.CmdSerpentKick), "Serpent-style kick attack.")
	registerCommand("scan", wrapSkill(command.CmdScan), "Scan adjacent rooms for creatures.")

	registerCommand("order", wrapArgs(cmdOrder), "Order a pet or follower.")

	// Informative commands (act_informative.go)
	registerCommand("color", wrapArgs(cmdColor), "Toggle ANSI color.")
	registerCommand("commands", wrapArgs(cmdCommands), "List available commands.", "cmds")
	registerCommand("socials", wrapArgs(cmdSocials), "List available socials.")
	registerCommand("wizhelp", wrapArgs(cmdWizhelp), "List privileged (immortal) commands.")
	// "glance" is src/interpreter.c's other top-level name for do_diagnose — identical handler.
	registerCommand("diagnose", wrapArgs(cmdDiagnose), "Diagnose health status of a target.", "diag", "glance")
	registerCommand("toggle", wrapArgs(cmdToggle), "Toggle a player preference.")
	registerCommand("lines", wrapArgs(cmdLines), "Set your screen line count (7-50).")
	registerCommand("infobar", wrapArgs(cmdInfoBar), "Toggle the bottom status infobar.")
	registerCommand("users", wrapArgs(cmdUsersSafe), "Show connected players.")

	// Other commands (act_other.go)
	registerCommand("save", wrapArgs(cmdSave), "Save your character.")
	registerCommand("report", wrapArgs(cmdReport), "Show report of your surroundings.")
	registerCommand("split", wrapArgs(cmdSplit), "Split gold with your group.")
	registerCommand("wimpy", wrapArgs(cmdWimpy), "Set your wimpy threshold.")
	registerCommand("display", wrapArgs(cmdDisplay), "Set display preferences.")
	registerCommand("transform", wrapArgs(cmdTransform), "Transform your appearance.")
	// "mount" is src/interpreter.c's other top-level name for do_ride — identical handler, same subcmd.
	registerCommand("ride", wrapArgs(cmdRide), "Ride a mount.", "mount")
	registerCommand("dismount", wrapArgs(cmdDismount), "Dismount from your mount.")
	registerCommand("yank", wrapArgs(cmdYank), "Yank someone from a mount or chair.")
	registerCommand("peek", wrapArgs(cmdPeek), "Peek at another player's inventory.")
	registerCommand("recall", wrapArgs(cmdRecall), "Recall to your home city.")
	registerCommand("stealth", wrapSkill(command.CmdStealth), "Enter stealth mode.")
	registerCommand("appraise", wrapArgs(cmdAppraise), "Appraise an item's value.")
	registerCommand("scout", wrapArgs(cmdScout), "Scout ahead for danger.")
	registerCommand("roll", wrapArgs(cmdRoll), "Roll a random number.")
	registerCommand("visible", wrapArgs(cmdVisible), "Make yourself visible again.")
	registerCommand("inactive", wrapArgs(cmdInactive), "Toggle inactive status.")
	registerCommand("auto", wrapArgs(cmdAuto), "Toggle auto-attack mode.")
	// Preference toggles (act.other.c do_gen_tog) — each is its own top-level
	// command in the original C, not a unified dispatcher; src/interpreter.c
	// lines 366-666.
	registerCommand("nosummon", wrapToggle("nosummon"), "Toggle summon protection.")
	registerCommand("nohassle", wrapToggle("nohassle"), "Toggle nohassle mode.")
	registerCommand("brief", wrapToggle("brief"), "Toggle brief room descriptions.")
	registerCommand("compact", wrapToggle("compact"), "Toggle compact display mode.")
	registerCommand("notell", wrapToggle("notell"), "Toggle deafness to tells.")
	registerCommand("noauction", wrapToggle("noauction"), "Toggle deafness to auctions.")
	registerCommand("noshout", wrapToggle("noshout"), "Toggle deafness to shouts.")
	registerCommand("nogossip", wrapToggle("nogossip"), "Toggle deafness to gossip.")
	registerCommand("nograts", wrapToggle("nograts"), "Toggle congratulation messages.")
	registerCommand("nowiz", wrapToggle("nowiz"), "Toggle deafness to the wiz channel.")
	registerCommand("quest", wrapToggle("quest"), "Toggle quest announcements.")
	registerCommand("roomflags", wrapToggle("roomflags"), "Toggle room flag display.")
	registerCommand("norepeat", wrapToggle("norepeat"), "Toggle communication echo.")
	registerCommand("holylight", wrapToggle("holylight"), "Toggle holylight mode.")
	registerCommand("nonewbie", wrapToggle("nonewbie"), "Toggle newbie channel.")
	registerCommand("noctell", wrapToggle("noctell"), "Toggle deafness to clan tells.")
	registerCommand("nobroadcast", wrapToggle("nobroadcast"), "Toggle deafness to broadcasts.")
	registerCommand("ident", wrapToggle("ident"), "Toggle ident lookups.")
	registerCommand("slowns", wrapToggle("slowns"), "Toggle nameserver resolution.")
	registerCommand("bug", wrapArgs(cmdBug), "Report a bug.")
	registerCommand("typo", wrapArgs(cmdTypo), "Report a typo.")
	registerCommand("idea", wrapArgs(cmdIdea), "Submit an idea.")
	registerCommand("todo", wrapArgs(cmdTodo), "Submit a todo suggestion.")
	registerCommand("afk", wrapArgs(cmdAFK), "Toggle away-from-keyboard status.")

	// Ban system (ported from ban.c)
	registerCommand("ban", wrapArgs(cmdBan), "Ban a site (admin only).")
	registerCommand("unban", wrapArgs(cmdUnban), "Unban a site (admin only).")

	// WHOD (ported from whod.c)
	registerCommand("whod", wrapArgs(cmdWhod), "Toggle WHOD display mode (admin only).")

	// Clan system (ported from clan.c)
	registerCommand("clan", wrapArgs(cmdClan), "Clan management commands.", "clans")

	// Houses (ported from house.c)
	registerCommand("house", wrapArgs(cmdHouse), "House management commands.")
	registerCommand("hcontrol", wrapArgs(cmdHcontrol), "Admin house control.")
	registerCommand("gossip", wrapArgs(cmdGossip), "Gossip on the channel.")
	registerCommand("auction", wrapArgs(cmdAuction), "Auction an item to the channel.")
	registerCommand("grats", wrapArgs(cmdGratz), "Congratulate someone on the channel.")
	registerCommand("newbie", wrapArgs(cmdNewbieChannel), "Ask a question on the newbie channel.")
	registerCommand("ctell", wrapArgs(cmdCTell), "Send a message to your clan.")
	registerCommand("password", wrapArgs(cmdPassword), "Change your password.")
	registerCommand("prompt", wrapArgs(cmdPrompt), "Set your prompt.")
	registerCommand("reply", wrapArgs(cmdReply), "Reply to the last tell.", "r")
	registerCommand(".", wrapArgs(cmdReply), "Reply to the last tell.")
	registerCommand("write", wrapArgs(cmdWrite), "Write on an object.")
	registerCommand("page", wrapArgs(cmdPage), "Page a player.")
	registerCommand("ignore", wrapArgs(cmdIgnore), "Ignore or stop ignoring a player.")
	registerCommand("race_say", wrapArgs(cmdRaceSay), "Say something in your racial language.", "rac")
	registerCommand("whisper", wrapArgs(cmdWhisper), "Whisper to someone in your room.", "whis")
	registerCommand("ask", wrapArgs(cmdAsk), "Ask someone a question.")
	registerCommand("qcomm", wrapArgs(cmdQcomm), "Send a team message.", "team")
	registerCommand("qsay", wrapArgs(cmdQsay), "Say something to quest participants.")
	registerCommand("qecho", wrapArgs(cmdQecho), "Echo text to quest participants (immortal).")
	// Social (act_social.go)

	// Alias (game pkg)
	registerCommand("alias", wrapArgs(cmdAlias), "Manage command aliases.")

	// Admin commands (game pkg bans) — duplicate of whod.c port; let the first one win
	// (no re-register here to avoid overwriting minPosition)
}

// wrapArgs adapts a func(*Session, []string) error to command.Handler.
func wrapArgs(fn func(*Session, []string) error) command.Handler {
	return func(s common.CommandSession, _cmd string, args []string) error {
		return fn(s.(*commandSession).Session, args)
	}
}

// wrapNoArgs adapts a func(*Session) error to command.Handler.
func wrapNoArgs(fn func(*Session) error) command.Handler {
	return func(s common.CommandSession, _cmd string, args []string) error {
		return fn(s.(*commandSession).Session)
	}
}

// wrapMove adapts cmdMove to the registry handler signature.
func wrapMove(direction string) command.Handler {
	return func(s common.CommandSession, _cmd string, args []string) error {
		return cmdMove(s.(*commandSession).Session, direction)
	}
}

// wrapToggle adapts a named player-preference toggle (brief, compact, notell,
// etc. — src/interpreter.c each registers do_gen_tog under its own command
// name, not a unified "toggle <name>" dispatcher) to command.Handler.
func wrapToggle(key string) command.Handler {
	return func(s common.CommandSession, _cmd string, args []string) error {
		sess := s.(*commandSession).Session
		sess.manager.world.ExecGenTog(sess.player, key)
		return nil
	}
}

// wrapSkill adapts a skill command (which uses command.SessionInterface) to command.Handler.
func wrapSkill(fn func(command.SessionInterface, []string) error) command.Handler {
	return func(s common.CommandSession, _cmd string, args []string) error {
		return fn(s.(*commandSession).Session, args)
	}
}

const cCommandWhitespace = " \t\n\v\f\r"

// splitCommandInput mirrors command_interpreter's command tokenization. A
// non-letter first character is always a one-character command, so punctuation
// commands do not require a separating space (for example, "'hello").
func splitCommandInput(input string) (string, []string) {
	input = strings.TrimLeft(input, cCommandWhitespace)
	if input == "" {
		return "", nil
	}

	first := input[0]
	isLetter := (first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z')
	if !isLetter {
		return input[:1], strings.Fields(input[1:])
	}

	if idx := strings.IndexAny(input, cCommandWhitespace); idx >= 0 {
		return input[:idx], strings.Fields(input[idx+1:])
	}
	return input, nil
}

// SplitCommandInput exposes the C-faithful tokenization for input layers that
// pre-split raw lines (the telnet listener). Without this, attached punctuation
// forms ("'hello", ":grins") are split on whitespace upstream and arrive here
// as unknown multi-char commands — caught by the command-surface-punctuation
// oracle scenario, invisible to direct ExecuteCommand unit tests.
func SplitCommandInput(input string) (string, []string) {
	return splitCommandInput(input)
}

// CommandArgumentText returns the argument remainder after C's command-word
// scan. It skips leading command whitespace but preserves internal and
// trailing whitespace, matching command_interpreter's pointer passed to a
// handler after any_one_arg.
func CommandArgumentText(input string) string {
	input = strings.TrimLeft(input, cCommandWhitespace)
	if input == "" {
		return ""
	}
	if first := input[0]; (first < 'A' || first > 'Z') && (first < 'a' || first > 'z') {
		return strings.TrimLeft(input[1:], cCommandWhitespace)
	}
	idx := strings.IndexAny(input, cCommandWhitespace)
	if idx < 0 {
		return ""
	}
	return strings.TrimLeft(input[idx+1:], cCommandWhitespace)
}

// ExecuteCommand processes a game command.
func ExecuteCommand(s *Session, cmdStr string, args []string) error {
	return executeCommandRaw(s, cmdStr, args, true, "")
}

// executeCommand mirrors comm.c's aliased flag. A normal input line may
// expand one player alias; commands placed at the front of the queue by a
// complex alias are executed with alias expansion disabled to prevent
// recursive aliases.
func executeCommand(s *Session, cmdStr string, args []string, allowAlias bool) error {
	return executeCommandRaw(s, cmdStr, args, allowAlias, "")
}

// executeCommandRaw is the transport-aware command path. rawArgs is retained
// only for command handlers whose C implementation consumes the original
// argument remainder instead of tokenized words.
func executeCommandRaw(s *Session, cmdStr string, args []string, allowAlias bool, rawArgs string) error {
	// Moderation pre-check: mute, ban
	if s.manager.modChecker != nil && s.player != nil {
		errMsg, reject := s.manager.modChecker.CheckPreCommand(s.player.Name, cmdStr)
		if reject {
			s.sendText(errMsg)
			return nil
		}
	}
	// Split command from arguments if args were not provided separately.
	if len(args) == 0 {
		cmdStr, args = splitCommandInput(cmdStr)
	}
	if cmdStr == "" {
		return nil
	}
	cmd := strings.ToLower(cmdStr)

	// C performs alias expansion before command_interpreter, so the command
	// interpreter's leading RNG draw belongs to the resolved command, not to
	// the alias trigger itself.
	if allowAlias && s.player != nil && len(s.player.Aliases) > 0 {
		fullInput := cmd
		if len(args) > 0 {
			fullInput = cmd + " " + strings.Join(args, " ")
		}
		if expanded, ok := game.ExpandAlias(s.player.Aliases, fullInput); ok {
			if len(expanded) == 0 {
				return nil
			}
			if len(expanded) > 1 {
				for i, command := range expanded {
					if i > 0 && s.player.GetWaitState() > 0 {
						s.inputMu.Lock()
						s.prependAliasedInputs(expanded[i:])
						s.inputMu.Unlock()
						return nil
					}
					if i > 0 {
						// C's game loop emits the command-cycle separation
						// between entries pulled from the alias queue. Prompt
						// normalization removes the prompt itself but retains
						// these two blank lines.
						s.Send("\r\n\r\n")
					}
					nextCmd, nextArgs := splitCommandInput(command)
					if err := executeCommandRaw(s, nextCmd, nextArgs, false, ""); err != nil {
						return err
					}
				}
				return nil
			}
			cmdStr, args = splitCommandInput(expanded[0])
			rawArgs = ""
			if cmdStr == "" {
				return nil
			}
			cmd = strings.ToLower(cmdStr)
		}
	}

	// C command_interpreter draws number(0,3) at the top of every playing
	// character command and clears AFF_HIDE on 0 (interpreter.c:889-890).
	// This must follow alias expansion and precede command lookup.
	// #nosec G404 — game RNG, not cryptographic
	if s.player != nil && commandNumber(0, 3) == 0 {
		s.player.SetAffect(game.AffHide, false)
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
			"quit": true, "reallyquit": true,
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

	// Spec procedure command interception — fast path skips room-bearing scans
	// when the room is known to contain no spec-bearing entities. Equipment and
	// inventory scans are unconditional: they iterate the player's own items and
	// were unconditional before the fast path was introduced.
	if s.player != nil && s.player.GetRoomVNum() > 0 {
		roomVNum := s.player.GetRoomVNum()
		argStr := strings.Join(args, " ")

		if s.manager.world.HasSpecInRoom(roomVNum) {
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

			// 3. Room items
			roomItems := s.manager.world.GetItemsInRoom(roomVNum)
			for _, item := range roomItems {
				if item != nil {
					if objSpec := game.GetObjSpecForObject(item.VNum); objSpec != nil {
						if objSpec(s.manager.world, s.player, item, cmd, argStr) {
							return nil
						}
					}
				}
			}
		}

		// 4. Equipped item spec procedures
		if s.player.Equipment != nil {
			equipped := s.player.Equipment.GetEquippedItems()
			for _, item := range equipped {
				if item != nil {
					if objSpec := game.GetObjSpecForObject(item.VNum); objSpec != nil {
						if objSpec(s.manager.world, s.player, item, cmd, argStr) {
							return nil
						}
					}
				}
			}
		}

		// 5. Inventory item spec procedures
		if s.player.Inventory != nil {
			invItems := s.player.Inventory.FindItems("")
			for _, item := range invItems {
				if item != nil {
					if objSpec := game.GetObjSpecForObject(item.VNum); objSpec != nil {
						if objSpec(s.manager.world, s.player, item, cmd, argStr) {
							return nil
						}
					}
				}
			}
		}
	}

	// R2d: C prefix/abbreviation resolution. Scan the ordered C table (law 2:
	// table order wins), level-filter DURING the scan (law 3 — load-bearing: a
	// mortal typing `go` must resolve to gossip, not the earlier goto which is
	// immortal-gated). First prefix match wins; typed-longer-than-entry never
	// matches. Go-only commands are not in the C table, so a prefix that only
	// matches them misses here and falls through to the exact-match registry
	// below — the only way Go-only names resolve (R4). Position/frozen gating
	// stays in commandGateRejected, post-resolution; not duplicated here.
	if canonical, ok := resolveCommandPrefix(cmd, getEffectiveLevel(s)); ok {
		cmd = canonical
	}

	entry, ok := cmdRegistry.Lookup(cmd)
	if !ok {
		// Check social emotes before giving up
		if _, found := game.Socials[cmd]; found {
			if commandGateRejected(s, mustCommandGate(cmd)) {
				return nil
			}
			game.DoAction(s.manager.world, s.player, cmd, strings.Join(args, " "))
			return nil
		}
		// C: interpreter.c:916 send_to_char("Huh?!?\r\n", ch) for any unmatched command.
		s.sendText("Huh?!?\r\n")
		return nil
	}

	if commandGateRejected(s, commandGate{MinLevel: entry.MinLevel, MinPosition: entry.MinPosition}) {
		return nil
	}
	if (cmd == "gtell" || cmd == "gsay") && rawArgs != "" {
		return cmdGtellText(s, rawArgs)
	}
	if cmd == "gecho" && rawArgs != "" {
		return cmdGechoText(s, rawArgs)
	}

	// NOTE: C's WAIT_STATE no longer gates commands here. comm.c:603's game-loop
	// drain short-circuits get_from_q while wait>0 — the command STAYS QUEUED
	// and drains later, with NO message. The invented "You're too busy!" gate
	// (plus its bypass allowlist) was R4 surface invention and has been deleted.
	// The wait gate now lives in the heartbeat's per-pulse drain
	// (Manager.DrainInputQueues), which queues wait>0 commands at the
	// handleCommand funnel (session_login.go) and drains one per pulse.
	return entry.Handler(&commandSession{Session: s}, cmd, args)
}

// commandGateRejected mirrors C interpreter.c:910-947 after a command row is
// found: level/lookup hiding, frozen, switched-NPC immortal denial, position.
// Go has no registered nil-handler/not-implemented state, so that C step is
// intentionally absent.
func commandGateRejected(s *Session, gate commandGate) bool {
	effectiveLevel := getEffectiveLevel(s)
	if effectiveLevel < gate.MinLevel {
		s.sendText("Huh?!?\r\n")
		return true
	}
	if s.player == nil {
		return false
	}
	if s.player.GetFlags()&(1<<uint(game.PlrFrozen)) != 0 && effectiveLevel < LVL_IMPL-1 {
		s.sendText("You try, but the mind-numbing cold prevents you...\r\n")
		return true
	}
	if s.isSwitched && s.switchedMob != nil && gate.MinLevel >= LVL_IMMORT {
		s.sendText("You can't use immortal commands while switched.\r\n")
		return true
	}
	playerPos := s.player.GetPosition()
	if playerPos < gate.MinPosition {
		s.sendText(positionFailMessage(playerPos))
		return true
	}
	return false
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

// cmdUse handles using an item through C's do_use path. The original command
// has no skill-use fallback: an unmatched target is reported by do_use.
func cmdUse(s *Session, args []string) error {
	if s.player == nil {
		return fmt.Errorf("not logged in")
	}
	if len(args) == 0 {
		s.sendText("What do you want to use?\r\n")
		return nil
	}

	itemArg := args[0]

	// C do_use SCMD_USE (act.other.c:920): "tattoo" is a keyword special-case;
	// otherwise the target must be EQUIPPED (WEAR_HOLD then any worn slot),
	// matched by keyword — never inventory, and only wand/staff are usable.
	// When nothing equipped matches, continue through C do_use so object-special
	// fallthrough preserves the original command surface and bytes.
	if strings.EqualFold(itemArg, "tattoo") {
		s.manager.world.DoUse(s.player, strings.Join(args, " "))
		return nil
	}
	if item := s.manager.world.FindEquippedVis(s.player, itemArg); item != nil {
		itemType := item.GetTypeFlag()
		if itemType == game.ITEM_WAND || itemType == game.ITEM_STAFF {
			s.manager.world.DoUse(s.player, strings.Join(args, " "))
			return nil
		}
		s.sendText("You can't seem to figure out how to use it.\r\nTry holding it.(?)\r\n")
		return nil
	}

	s.manager.world.DoUse(s.player, strings.Join(args, " "))
	return nil
}

// cmdSave saves the player's character.
