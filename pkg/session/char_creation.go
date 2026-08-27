package session

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"

	"github.com/zax0rz/darkpawns/internal/dpclock"
	"github.com/zax0rz/darkpawns/pkg/auth"
	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/validation"
)

// C source race help constants from src/constants.c
const RaceMenuText = "\r\n" +
	"Choose a race:\r\n" +
	"  [H]uman        [E]lven       [D]warven      [K]enderkin\r\n" +
	"  [M]inotaur     [R]akshasan   [S]sauran\r\n" +
	"  [?]Help on races in general\r\n" +
	"  [?<race abbreviation>] Help on a specific race (i.e ?D for help on dwarves)\r\n"

const RaceHelpText = `
Your race is pretty much class independant; it affects innate abilities such
as:
The type of terrain you see best in: 
       RAKSHASA: desert              SSAUR: swamplands
       MINOTAUR & ELF: forest        DWARF: mountains
       KENDER & HUMAN: fairly good everywhere.
Magick resistance.: Elves and dwarves are a bit more hearty in this area.
Attitudes: Humans abound, so they are often suspicious of other races and
       give preferential treatment to their own kind.
Kender tend to 'acquire' other's objects unknowingly, and make excellent
       thieves. Only humans can belong to the ninja class.
Each race has its own language.`

const HelpHuman = `
Humans are the most common race on this world, and come in all sorts of shapes
and sizes. The appearance of humans are not the only thing that varies about
them, though, some are evil as sin, while others are good as good can be, but
most you shall find on your adventures are neutral, and will just mind their
own business and pay no attention to the affairs of adventurers. Also, humans
are the only race that can become ninjas, the dangerous oriental mercenaries.
They adapt easily to most climes, allowing them to build cities in almost any
location.`

const HelpDwarf = `
Dwarves are a noble race of demihumans who dwell under the earth, forging
great cities and waging massive wars against the forces of chaos and evil.
Dwarves also have much in common with the rocks and gems they love to work,
for they are both hard and unyielding. It's often been said that it's easier
to make a stone weep than it is to change a dwarf's mind. Standing an average
of four-and-a-half feet tall, dwarves tend to be stocky and muscular. They
have ruddy cheeks and bright eyes. Their skin is typically deep tan or light
brown. Their hair is usually black, grey, or brown, and worn long, though not
long enough to impair vision in any way. They favor long beards and moustaches
as well.`

const HelpElf = `
Though their lives span several human generations, elves appear at first
glance to be frail when compared to man, due to their delicate and finely
chiseled features. Elves have very pale complextions, which is odd because
they spend a great deal of time outdoors. They tend to be slim, almost 
fragile. Though they are not as sturdy as humans, elves are much more agile.
Elves have learned that it is very important to understand the creatures, both
good and evil, that share their forest homes.`

const HelpKender = `
Kender are small, kind, but somewhat annoying, elf-like beings that have
recently spread across the globe. They do not seem to have any sort of kingdom
and most are found just wandering throughout the lands, exploring. Although
some are trained thieves, the whole of the kender race seems to have a knack
for stealing, and occasionally, without even noticing it sometimes, they have
been known to steal from friends and enemies alike. They act much like humans,
but four things make a kender's personality drastically different from that of
a typical human. Kender are utterly fearless, insatiably curious, unstoppably
mobile and independant, and will pick up anything that is not nailed down.`

const HelpMinotaur = `
Minotaurs are either cursed humans or the offspring of minotaurs and humans.
They are usually found dwelling in underground labyrinths, for they seem to
have an innate ability to manuver in these places, and do not often lose their
sense of direction. Minotaurs are huge, well over seven feet tall, and their
broad bodies ripple with muscles. They have the head of a bull but the body of
a human male, there have been accounts of female minotaurs, but they are rare.
The color of their fur ranges from brown to black, while their body coloring
varies, as would a normal human's. Although they usually dwell in mazes
beneath the earth, it is noted that they also see very well in forests.`

const HelpRakshasa = `
Rakshasas are a race of malevolent spirits encased in flesh that hunt and
torment humanity. No one knows where these creatures originate, some say they
are the embodiment of nightmares. The only way to describe their form is that
they are humanoid tigers, with hands whose palms curve backward, away from the
body. Most of the worlds rakshasa are evil, but recently many have decided to
stop their tyrannical living and become adventurers, although they still
retain their fondness towards the great sandy wastes of their homeland.`

const HelpSsaur = `
Ssaurs are a relatively new race in the world. They are a more evolved type of
lizardman, and most are more intelligent than their aggressive ancestors, and
for that are shunned from the lizardman tribes, and the few that are born into
those tribes are cast out almost as soon as they are hatched. Other than the 
intelligence, they appear to be the same as lizardman, although less evil-
looking. Ssaurs spend most of their lives in swamps and marshes, but some have
been known to adventure far away from their homes.`

const ClassMenuText = "\r\n" +
	"Select a class:\r\n" +
	"  [C]leric     - Healers and warriors of the gods\r\n" +
	"  [T]hief      - Stealthy, quick-fingered, lock-picking back-stabbers\r\n" +
	"  [W]arrior    - Fierce, battle-trained fighters\r\n" +
	"  [M]agic-user - Spell-casters trained in the art of magick\r\n" +
	"  Ps[i]onic    - Fighters endowed with the powers of the mind"

const HumanClassMenuText = "\r\n" +
	"Select a class:\r\n" +
	"  [C]leric     - Healers and warriors of the gods\r\n" +
	"  [T]hief      - Stealthy, quick-fingered, lock-picking back-stabbers\r\n" +
	"  [W]arrior    - Fierce, battle-trained fighters\r\n" +
	"  [M]agic-user - Spell-casters trained in the art of magick\r\n" +
	"  [N]inja      - Stealthy, magick-endowed warriors from the orient\r\n" +
	"  Ps[i]onic    - Fighters endowed with the powers of the mind"

const HometownMenuText = "\r\n" +
	"Choose your home town:\r\n" +
	"  [K]ir Drax'in  - The Main City. New players should choose this.\r\n" +
	"  Kir-[O]shi     - The Port City.\r\n" +
	"  [A]laozar      - The Holy City.\r\n"

const maxCreationPasswordLength = 10

// handleCharInput processes character creation input from the client.
// Implements the nanny() flow from interpreter.c.
// Stages: confirm_name → create_password → confirm_password → color → sex → race → class → hometown → stats_roll → motd
func (s *Session) handleCharInput(data json.RawMessage) error {
	var input CharInputData
	if err := json.Unmarshal(data, &input); err != nil {
		return err
	}

	choice := strings.TrimSpace(input.Choice)

	switch s.charStage {
	case "get_name":
		if !validation.IsValidPlayerName(choice) || !game.ValidName(choice) {
			s.sendCharCreatePrompt("get_name", "Invalid name, please try another.\r\nName: ", nil)
			return nil
		}
		s.charName = choice
		s.charStage = "confirm_name"
		s.sendCharCreatePrompt("confirm_name", fmt.Sprintf("Please remember to choose an appropriate fantasy-oriented name.\r\nDid I get that right, %s (Y/N)? ", choice), nil)

	case "confirm_name":
		switch strings.ToUpper(choice) {
		case "Y":
			s.charStage = "create_password"
			s.sendCharCreatePromptWithSecret("create_password", fmt.Sprintf("New character.\r\nGive me a password for %s: ", s.charName), nil, true)
		case "N":
			s.charStage = "get_name"
			s.charName = ""
			s.sendCharCreatePrompt("get_name", "Okay, what IS it, then? ", nil)
		default:
			s.sendCharCreatePrompt("confirm_name", "Please type Yes or No: ", nil)
		}

	case "create_password":
		if len(choice) < 3 || len(choice) > maxCreationPasswordLength || strings.EqualFold(choice, s.charName) {
			s.sendCharCreatePromptWithSecret("create_password", "\r\nIllegal password.\r\nPassword: ", nil, true)
			return nil
		}
		s.charPassword = choice
		s.charStage = "confirm_password"
		s.sendCharCreatePromptWithSecret("confirm_password", "\r\nPlease retype password: ", nil, true)

	case "confirm_password":
		if choice != s.charPassword {
			s.sendCharCreatePromptWithSecret("create_password", "\r\nPasswords don't match... start over.\r\nPassword: ", nil, true)
			s.charStage = "create_password"
			return nil
		}
		if err := s.hashCharPassword(); err != nil {
			return err
		}
		s.charStage = "color"
		s.sendCharCreatePrompt("color", "\r\nDo you want ANSI color (Y/N)? ", nil)

	case "color":
		switch strings.ToUpper(choice) {
		case "Y":
			s.charColor = true
			s.advanceCharStage("sex", "What is your sex (M/F)? ", nil)
		case "N":
			s.charColor = false
			s.advanceCharStage("sex", "What is your sex (M/F)? ", nil)
		default:
			s.sendCharCreatePrompt("color", "Please answer Y or N.\r\nDo you want ANSI color (Y/N)? ", nil)
		}

	case "sex":
		switch strings.ToUpper(choice) {
		case "M":
			s.charSex = 0
			s.advanceCharStage("race", RaceMenuText+"\r\nRace: ", s.getRaceOptions())
		case "F":
			s.charSex = 1
			s.advanceCharStage("race", RaceMenuText+"\r\nRace: ", s.getRaceOptions())
		default:
			s.sendCharCreatePrompt("sex", "That is not a sex..\r\nWhat IS your sex? ", nil)
		}

	case "race":
		upperChoice := strings.ToUpper(choice)
		if strings.HasPrefix(upperChoice, "?") {
			var help string
			if len(upperChoice) > 1 {
				switch upperChoice[1:2] {
				case "H":
					help = HelpHuman
				case "E":
					help = HelpElf
				case "D":
					help = HelpDwarf
				case "K":
					help = HelpKender
				case "M":
					help = HelpMinotaur
				case "R":
					help = HelpRakshasa
				case "S":
					help = HelpSsaur
				default:
					help = "\r\nThat is not a race..\r\n"
				}
			} else {
				help = RaceHelpText
			}
			s.sendCharCreatePrompt("race", creationCRLF(help)+RaceMenuText+"\r\nRace: ", s.getRaceOptions())
			return nil
		}

		var race int
		switch upperChoice {
		case "H":
			race = game.RaceHuman
		case "E":
			race = game.RaceElf
		case "D":
			race = game.RaceDwarf
		case "K":
			race = game.RaceKender
		case "M":
			race = game.RaceMinotaur
		case "R":
			race = game.RaceRakshasa
		case "S":
			race = game.RaceSsaur
		default:
			s.sendCharCreatePrompt("race", "That is not a race..\r\nWhat IS your race? ", s.getRaceOptions())
			return nil
		}

		s.charRace = race
		var classMenu string
		if s.charRace == game.RaceHuman {
			classMenu = HumanClassMenuText
		} else {
			classMenu = ClassMenuText
		}
		s.advanceCharStage("class", classMenu+"\r\nClass: ", s.getClassOptions(s.charRace))

	case "class":
		upperChoice := strings.ToUpper(choice)
		var classID int
		switch upperChoice {
		case "M":
			classID = game.ClassMageUser
		case "C":
			classID = game.ClassCleric
		case "T":
			classID = game.ClassThief
		case "W":
			classID = game.ClassWarrior
		case "I":
			classID = game.ClassPsionic
		case "N":
			if s.charRace == game.RaceHuman {
				classID = game.ClassNinja
			} else {
				s.sendCharCreatePrompt("class", "\r\nThat's not a class.\r\nClass: ", s.getClassOptions(s.charRace))
				return nil
			}
		default:
			s.sendCharCreatePrompt("class", "\r\nThat's not a class.\r\nClass: ", s.getClassOptions(s.charRace))
			return nil
		}

		s.charClass = classID
		s.advanceCharStage("hometown", HometownMenuText+"\r\nSelect: ", charOpts("K", "Kir Drax'in", "O", "Kir-Oshi", "A", "Alaozar"))

	case "hometown":
		upperChoice := strings.ToUpper(choice)
		var hometown int
		switch upperChoice {
		case "K":
			hometown = 1
		case "O":
			hometown = 2
		case "A":
			hometown = 3
		default:
			s.sendCharCreatePrompt("hometown", "Invalid choice!\r\nSelect: ", charOpts("K", "Kir Drax'in", "O", "Kir-Oshi", "A", "Alaozar"))
			return nil
		}

		s.charHometown = hometown
		s.charStats = game.RollRealAbils(s.charClass, s.charRace)
		s.sendStatsRollPrompt()

	case "stats_roll":
		switch strings.ToUpper(choice) {
		case "Y":
			// Show MOTD and transition to PRESS RETURN state
			motd := game.ShowMOTD(s.manager.world.WorldPath)
			s.charStage = "motd"
			s.sendCharCreatePrompt("motd", motd+"\r\n\n*** PRESS RETURN: ", nil)
		case "N":
			s.charStats = game.RollRealAbils(s.charClass, s.charRace)
			s.sendStatsRollPrompt()
		default:
			s.sendCharCreatePrompt("stats_roll", "Invalid choice! Select 'Y' or 'N':", nil)
		}

	case "motd":
		// Both new and returning characters choose what to do after the MOTD.
		s.charCreating = false
		s.showMainMenu()

	default:
		return fmt.Errorf("unexpected char creation stage: %s", s.charStage)
	}
	return nil
}

func getAbilName(score int) string {
	abilNames := []string{
		"terrible",      // 0
		"terrible",      // 1
		"awful",         // 2
		"awful",         // 3
		"bad",           // 4
		"bad",           // 5
		"poor",          // 6
		"poor",          // 7
		"below average", // 8
		"average",       // 9
		"average",       // 10
		"decent",        // 11
		"decent",        // 12
		"good",          // 13
		"good",          // 14
		"very good",     // 15
		"very good",     // 16
		"excellent",     // 17
		"excellent",     // 18
		"superior",      // 19
		"godlike",       // 20
		"godlike",       // 21
		"godlike",       // 22
		"godlike",       // 23
		"godlike",       // 24
		"godlike",       // 25
	}
	if score < 0 {
		return "terrible"
	}
	if score >= len(abilNames) {
		return "godlike"
	}
	return abilNames[score]
}

// sendStatsRollPrompt displays the current rolled stats and asks to keep or reroll.
func (s *Session) sendStatsRollPrompt() {
	stats := &CharStatsDisplay{
		Str: s.charStats.Str,
		Int: s.charStats.Int,
		Wis: s.charStats.Wis,
		Dex: s.charStats.Dex,
		Con: s.charStats.Con,
		Cha: s.charStats.Cha,
	}
	prompt := fmt.Sprintf(
		"\r\nYour ability scores:\r\n  Str: %-13s Dex: %-13s Int: %-13s\r\n  Wis: %-13s Con: %-13s Cha: %-13s\r\n\r\nPress 'Y' to keep these stats, and 'N' to reroll:",
		getAbilName(stats.Str), getAbilName(stats.Dex), getAbilName(stats.Int),
		getAbilName(stats.Wis), getAbilName(stats.Con), getAbilName(stats.Cha),
	)
	data := CharCreateData{
		Stage:   "stats_roll",
		Prompt:  prompt,
		Options: charOpts("Y", "Keep", "N", "Reroll"),
		Stats:   stats,
	}
	s.charStage = "stats_roll"
	msg, err := json.Marshal(ServerMessage{
		Type: MsgCharCreate,
		Data: data,
	})
	if err != nil {
		slog.ErrorContext(s.sessionCtx, "json.Marshal error", s.logAttrs(slog.Any("error", err))...)
		return
	}
	s.send <- msg
}

// startCharCreation begins the character creation flow for a new player.
//
// Deprecated: prefer startNewCharFlow to match stateful C nanny flow.
func (s *Session) startCharCreation(playerName string) {
	s.startNewCharFlow(playerName)
}

// startNewCharFlow begins the C nanny flow at name confirmation. Passwords for
// new characters are always collected here, never by a transport auth shim.
func (s *Session) startNewCharFlow(playerName string) {
	s.charCreating = true
	s.charName = playerName
	s.charPassword = ""
	s.charStage = "confirm_name"
	s.sendCharCreatePrompt("confirm_name", fmt.Sprintf("Please remember to choose an appropriate fantasy-oriented name.\r\nDid I get that right, %s (Y/N)? ", playerName), nil)
}

func (s *Session) restartNameEntry() {
	s.charCreating = true
	s.charStage = "get_name"
	s.charName = ""
	s.charPassword = ""
	s.sendCharCreatePrompt("get_name", "Invalid name, please try another.\r\nName: ", nil)
}

func creationCRLF(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\n", "\r\n")
	if !strings.HasSuffix(text, "\r\n") {
		text += "\r\n"
	}
	return text
}

// sendCharCreatePrompt sends a character creation prompt to the client.
func (s *Session) sendCharCreatePrompt(stage, prompt string, options []CharCreateOption) {
	s.sendCharCreatePromptWithSecret(stage, prompt, options, false)
}

func (s *Session) sendCharCreatePromptWithSecret(stage, prompt string, options []CharCreateOption, secret bool) {
	data := CharCreateData{
		Stage:   stage,
		Prompt:  prompt,
		Options: options,
		Secret:  secret,
	}

	msg, err := json.Marshal(ServerMessage{
		Type: MsgCharCreate,
		Data: data,
	})
	if err != nil {
		slog.ErrorContext(s.sessionCtx, "json.Marshal error", s.logAttrs(slog.Any("error", err))...)
		return
	}

	s.send <- msg
}

// completeCharCreation finalizes character creation and enters the world.
func (s *Session) completeCharCreation() error {
	// Create the player with collected attributes
	s.player = game.NewCharacterWithStats(0, s.charName, s.charClass, s.charRace, s.charSex, s.charStats)
	s.player.Description = s.menuDescription

	// Set hometown
	s.player.Hometown = s.charHometown

	// First-player-God bootstrap (init_char, db.c:3016). On a fresh MUD the
	// very first character becomes an Implementor. Runs before the DB save so
	// the persisted row reflects Level 40 / God stats. The God gets every skill
	// at 100, so skip the class's low starting-skill grants for it (C's
	// init_char sets all-100 after do_start would, unconditionally).
	isGod := s.manager.shouldCrownFirstPlayer()
	if isGod {
		game.BootstrapFirstPlayerGod(s.player)
	} else {
		// C: do_start()→advance_level() runs only `if (!GET_LEVEL(ch))`
		// (interpreter.c:2214) — i.e. for a real level-1 mortal, never for the
		// God (already LVL_IMPL). AdvanceLevel grants the level-1 HP/move bonus
		// (and its 2 RNG draws); GiveStartingSkills then layers the class skills,
		// matching C's do_start order. The constructor no longer calls
		// AdvanceLevel (DP-1212: it consumed 2 phantom draws on the God path).
		s.player.AdvanceLevel()
		game.GiveStartingSkills(s.player)
	}

	// Level-based entry routing (interpreter.c:2191-2243). Immortals enter
	// ImmortStartRoom (1204); mortals use the newbie intro room (8099) then
	// transition to their hometown room after the birth sequence.
	newbieRoom := game.NewbieHometownRoom(s.charHometown)
	if s.player.GetLevel() >= game.LVL_IMMORT {
		s.player.SetRoom(game.ImmortStartRoom)
	} else {
		s.player.SetRoom(newbieRoom)
	}

	// Save to DB if available
	if s.manager.hasDB {
		if r, err := db.PlayerToRecord(s.player, nil); err == nil {
			// Apply the hashed password collected during login
			r.Password = s.charPassword
			if err := s.manager.db.CreatePlayer(r); err != nil {
				// Check for constraint violation (player already exists)
				if isUniqueConstraintError(err) {
					slog.WarnContext(s.sessionCtx, "duplicate character name, rejecting creation", s.logAttrs()...)
					return fmt.Errorf("a character named '%s' already exists", s.charName)
				}
				slog.ErrorContext(s.sessionCtx, "DB create error during char creation", s.logAttrs(slog.Any("error", err))...)
				return fmt.Errorf("failed to save character: %w", err)
			}
			s.player.ID = r.ID
		}
	}

	// Populate legacy spell-catalog metadata; proficiency still requires practice.
	grantClassSpells(s.player)

	// Register and add to world
	s.authenticated = true
	s.playerName = s.charName

	if err := s.manager.Register(s.charName, s); err != nil {
		return err
	}

	// C source intro: brand-new mortals first appear in the Burning Hut (8099);
	// immortals enter ImmortStartRoom (1204) and skip the intro entirely.
	if s.player.GetLevel() >= game.LVL_IMMORT {
		s.player.SetRoom(game.ImmortStartRoom)
	} else {
		s.player.SetRoom(game.NewbieStartRoom)
	}
	if err := s.manager.world.AddPlayer(s.player); err != nil {
		s.manager.Unregister(s.charName)
		return err
	}

	// C skips do_start() for the first-player God because init_char already
	// assigned a nonzero level (interpreter.c:2214). That means the God gets
	// no mortal starter gear; only a newly created mortal receives it.
	if !isGod {
		s.manager.world.GiveStartingItems(s.player)
	}

	slog.InfoContext(s.sessionCtx, "completeCharCreation: player added to world", s.logAttrs()...)

	// Safety: catch any panic in the rest of char creation
	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(s.sessionCtx, "PANIC in completeCharCreation", s.logAttrs(slog.Any("recover", r))...)
		}
	}()

	// Clear char creation state
	s.charCreating = false
	s.charStage = ""
	s.charName = ""
	s.charPassword = ""
	s.charColor = false
	s.charSex = 0
	s.charRace = 0
	s.charClass = 0
	s.charHometown = 0
	s.charStats = game.CharStats{}
	s.clearMenuState()
	slog.InfoContext(s.sessionCtx, "completeCharCreation: state cleared", s.logAttrs()...)

	// Generate JWT token
	token, err := auth.GenerateJWT(s.player.Name, s.isAgent, s.agentKeyID, "")
	if err != nil {
		slog.ErrorContext(s.sessionCtx, "failed to generate JWT token", s.logAttrs(slog.Any("error", err))...)
	}

	// Send welcome with token
	s.sendWelcome(token)

	// C's start_room special emits this birth transition on the first
	// PULSE_MOBILE after entry. Go completes it synchronously; under the frozen
	// oracle clock, expose the equivalent output at that existing transition.
	// Immortals skip the Burning Hut intro entirely (C never routes them
	// through 8099 — interpreter.c:2191-2243).
	if s.player.GetLevel() < game.LVL_IMMORT {
		if dpclock.Frozen() {
			s.sendText(newbieBirthMessage(s.player.Name))
		}
		s.player.SetRoom(newbieRoom)
		if dpclock.Frozen() {
			s.sendRoomObservation(newbieRoom, false, "")
		}
	}

	// Mirror the agent initialization that handleLogin sends for returning players.
	if s.isAgent || s.wantsStructuredData {
		s.sendFullVarDump()
		if s.isAgent {
			s.SendMemoryBootstrap()
			s.SendMemorySummary()
		}
	}

	// Broadcast arrival
	enterMsg, err := json.Marshal(ServerMessage{
		Type: MsgEvent,
		Data: EventData{
			Type: "enter",
			Text: s.player.Name + " has arrived.",
		},
	})
	if err != nil {
		slog.ErrorContext(s.sessionCtx, "json.Marshal error", s.logAttrs(slog.Any("error", err))...)
		return nil
	}
	s.manager.BroadcastToRoom(s.player.GetRoom(), enterMsg, s.player.Name)

	return nil
}

func newbieBirthMessage(name string) string {
	return "\r\n   Suddenly the hairs on the back of your neck stand up as if lightning had\r\n" +
		"struck nearby. A keen wailing fills the air, and an ethereal image appears\r\n" +
		"before you.\r\n" +
		fmt.Sprintf("   '%s, now is not your time to die,' speaks the figure.\r\n", name) +
		"   'Prove your worth and I may well grant you eternal life.'\r\n" +
		"   'Trust no one, for all here are but dark pawns above which you must\r\n" +
		"struggle to prove yourself.  All here strive to be a king... at any cost.'\r\n" +
		"   The figure glows a moment, then disappears, but his voice remains.\r\n" +
		"   'Your life begins now...' it says, then fades -- just as the world around\r\n" +
		"you does the same.\r\n"
}

// hashCharPassword bcrypt-hashes the plaintext password currently in
// s.charPassword, storing the hash back into s.charPassword. Shared by the
// confirm_password stage and the confirm_name skip-branch (DP-909).
func (s *Session) hashCharPassword() error {
	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(s.charPassword), bcrypt.DefaultCost)
	if err != nil {
		slog.ErrorContext(s.sessionCtx, "bcrypt hash error", s.logAttrs(slog.Any("error", err))...)
		return err
	}
	s.charPassword = string(hashedPwd)
	return nil
}

// advanceCharStage moves to the next char creation stage.
func (s *Session) advanceCharStage(stage, prompt string, options []CharCreateOption) {
	s.charStage = stage
	s.sendCharCreatePrompt(stage, prompt, options)
}

// getRaceOptions returns available races for character creation in a stable
// order (Human, Elven, Dwarven, …) matching the RaceMenuText layout and C's
// static race array. A slice — not a map — so render order is deterministic
// across renders (DP-909).
func (s *Session) getRaceOptions() []CharCreateOption {
	return []CharCreateOption{
		{"H", "Human"},
		{"E", "Elven"},
		{"D", "Dwarven"},
		{"K", "Kenderkin"},
		{"M", "Minotaur"},
		{"R", "Rakshasan"},
		{"S", "Ssauran"},
	}
}

// getClassOptions returns available classes for character creation, filtered
// by race, in stable menu order (Cleric, Thief, Warrior, Magic-user, Psionic,
// and Ninja for humans) matching ClassMenuText/HumanClassMenuText (DP-909).
func (s *Session) getClassOptions(race int) []CharCreateOption {
	opts := []CharCreateOption{
		{"C", "Cleric"},
		{"T", "Thief"},
		{"W", "Warrior"},
		{"M", "Mage"},
		{"I", "Psionic"},
	}
	if race == game.RaceHuman {
		opts = append(opts, CharCreateOption{"N", "Ninja"})
	}
	return opts
}

// charOpts is a small helper to build a []CharCreateOption from key/label
// pairs inline, keeping call sites readable (DP-909).
func charOpts(pairs ...string) []CharCreateOption {
	opts := make([]CharCreateOption, 0, len(pairs)/2)
	for i := 0; i+1 < len(pairs); i += 2 {
		opts = append(opts, CharCreateOption{Key: pairs[i], Label: pairs[i+1]})
	}
	return opts
}

// isUniqueConstraintError checks if a DB error is a PostgreSQL unique constraint violation.
func isUniqueConstraintError(err error) bool {
	if pqErr, ok := err.(*pq.Error); ok {
		return pqErr.Code == "23505"
	}
	return strings.Contains(err.Error(), "unique constraint") ||
		strings.Contains(err.Error(), "duplicate key")
}
