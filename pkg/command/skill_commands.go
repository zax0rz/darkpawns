package command

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

type rescueCombatEngine interface {
	StartCombat(combat.Combatant, combat.Combatant) error
	PerformInitialAttack(combat.Combatant, combat.Combatant) error
	StopCombat(string)
	// SkillMessage routes a combat message through the skill_message path
	// (fight.c:1023-1092), drawing Dice(1,N) and emitting the set's text.
	SkillMessage(dam int, ch, vict string, attackType int, roomVNum int) bool
}

// cmdSkills displays all learned skills
func CmdSkills(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}

	player := s.GetPlayer()
	skillManager := player.SkillManager
	if skillManager == nil {
		return s.SendMessage("You have no skills.\r\n")
	}

	learnedSkills := skillManager.GetLearnedSkills()
	if len(learnedSkills) == 0 {
		return s.SendMessage("You haven't learned any skills yet.\r\n")
	}

	var output strings.Builder
	output.WriteString("╔══════════════════════════════════════════════════════╗\r\n")
	output.WriteString("║                     Your Skills                      ║\r\n")
	output.WriteString("╠══════════════════════════╦══════╦════════╦═══════════╣\r\n")
	output.WriteString("║ Skill                    ║ Level║ Progress║ Type     ║\r\n")
	output.WriteString("╠══════════════════════════╬══════╬════════╬═══════════╣\r\n")

	for _, skill := range learnedSkills {
		// Truncate display name if too long
		displayName := skill.DisplayName
		if len(displayName) > 22 {
			displayName = displayName[:19] + "..."
		}

		// Get skill type as string
		typeStr := "Utility"
		switch skill.Type {
		case engine.SkillTypeCombat:
			typeStr = "Combat"
		case engine.SkillTypeMagic:
			typeStr = "Magic"
		}

		// Get progress percentage
		progress := skill.GetProgress()

		fmt.Fprintf(&output, "║ %-22s ║ %4d ║ %3d%%   ║ %-9s ║\r\n",
			displayName, skill.Level, progress, typeStr)
	}

	output.WriteString("╚══════════════════════════╩══════╩════════╩═══════════╝\r\n")

	// Add skill points and slots info
	points := skillManager.GetSkillPoints()
	usedSlots := skillManager.GetUsedSlots()
	totalSlots := skillManager.GetSlots()
	availableSlots := skillManager.GetAvailableSlots()

	fmt.Fprintf(&output, "\r\nSkill points: %d | Slots: %d/%d (%d available)\r\n",
		points, usedSlots, totalSlots, availableSlots)

	return s.SendMessage(output.String())
}

// cmdPractice practices a skill
func CmdPractice(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}

	if len(args) == 0 {
		return s.SendMessage("Practice what? Usage: practice <skill>\r\n")
	}

	skillName := strings.ToLower(strings.Join(args, " "))
	player := s.GetPlayer()
	skillManager := player.SkillManager

	if skillManager == nil {
		return s.SendMessage("You have no skills to practice.\r\n")
	}

	// Check if skill exists and is learned
	skill := skillManager.GetSkill(skillName)
	if skill == nil || !skill.Learned {
		return s.SendMessage(fmt.Sprintf("You haven't learned '%s'.\r\n", skillName))
	}

	// Can't practice beyond max level
	if skill.Level >= skill.MaxLevel {
		return s.SendMessage(fmt.Sprintf("You have already mastered %s.\r\n", skill.DisplayName))
	}

	// Determine which stat to use for practice check
	var stat int
	switch skill.Type {
	case engine.SkillTypeCombat:
		// Use strength or dexterity, whichever is higher
		str := player.GetStr()
		dex := player.GetDex()
		if str > dex {
			stat = str
		} else {
			stat = dex
		}
	case engine.SkillTypeMagic:
		// Use intelligence or wisdom, whichever is higher
		intel := player.GetInt()
		wis := player.GetWis()
		if intel > wis {
			stat = intel
		} else {
			stat = wis
		}
	case engine.SkillTypeUtility:
		// Use dexterity or intelligence
		dex := player.GetDex()
		intel := player.GetInt()
		if dex > intel {
			stat = dex
		} else {
			stat = intel
		}
	}

	// Practice the skill
	leveledUp := skillManager.PracticeSkill(skillName, player.GetLevel(), stat)

	if leveledUp {
		return s.SendMessage(fmt.Sprintf("You practice %s diligently and advance to level %d!\r\n",
			skill.DisplayName, skill.Level))
	}
	progress := skill.GetProgress()
	return s.SendMessage(fmt.Sprintf("You practice %s. Progress: %d%% (Level %d)\r\n",
		skill.DisplayName, progress, skill.Level))
}

// cmdLearn attempts to learn a new skill
func CmdLearn(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}

	if len(args) == 0 {
		// Show available skills to learn
		return CmdListSkills(s, args)
	}

	skillName := strings.ToLower(strings.Join(args, " "))
	player := s.GetPlayer()
	skillManager := player.SkillManager

	if skillManager == nil {
		skillManager = engine.NewSkillManager()
		player.SkillManager = skillManager
	}

	// Check if skill exists
	skill := skillManager.GetSkill(skillName)
	if skill == nil {
		return s.SendMessage(fmt.Sprintf("Skill '%s' doesn't exist.\r\n", skillName))
	}

	// Check if already learned
	if skill.Learned {
		return s.SendMessage(fmt.Sprintf("You already know %s (Level %d).\r\n",
			skill.DisplayName, skill.Level))
	}

	// Check requirements
	var stat int
	switch skill.Type {
	case engine.SkillTypeCombat:
		stat = player.GetStr()
	case engine.SkillTypeMagic:
		stat = player.GetInt()
	case engine.SkillTypeUtility:
		stat = player.GetDex()
	}

	if !skill.CanLearn(player.GetLevel(), stat) {
		return s.SendMessage(fmt.Sprintf("You don't meet the requirements to learn %s.\r\n",
			skill.DisplayName))
	}

	// Check skill points
	if skillManager.GetSkillPoints() < skill.Difficulty {
		return s.SendMessage(fmt.Sprintf("You need %d skill points to learn %s. You have %d.\r\n",
			skill.Difficulty, skill.DisplayName, skillManager.GetSkillPoints()))
	}

	// Check available slots
	if skillManager.GetAvailableSlots() <= 0 {
		return s.SendMessage("You don't have any available skill slots.\r\n")
	}

	// Learn the skill
	success := skillManager.LearnSkill(skill, player.GetLevel(), stat)
	if success {
		return s.SendMessage(fmt.Sprintf("You successfully learn %s!\r\n", skill.DisplayName))
	}
	return s.SendMessage(fmt.Sprintf("You failed to learn %s.\r\n", skill.DisplayName))
}

// CmdListSkills shows all available skills
func CmdListSkills(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}

	player := s.GetPlayer()
	skillManager := player.SkillManager

	if skillManager == nil {
		skillManager = engine.NewSkillManager()
		skillManager.InitializeDefaultSkills()
		player.SkillManager = skillManager
	}

	allSkills := skillManager.GetAllSkills()
	if len(allSkills) == 0 {
		skillManager.InitializeDefaultSkills()
		allSkills = skillManager.GetAllSkills()
	}

	// Group skills by type
	combatSkills := []*engine.Skill{}
	magicSkills := []*engine.Skill{}
	utilitySkills := []*engine.Skill{}

	for _, skill := range allSkills {
		switch skill.Type {
		case engine.SkillTypeCombat:
			combatSkills = append(combatSkills, skill)
		case engine.SkillTypeMagic:
			magicSkills = append(magicSkills, skill)
		case engine.SkillTypeUtility:
			utilitySkills = append(utilitySkills, skill)
		}
	}

	// Sort each group by name
	sort.Slice(combatSkills, func(i, j int) bool {
		return combatSkills[i].Name < combatSkills[j].Name
	})
	sort.Slice(magicSkills, func(i, j int) bool {
		return magicSkills[i].Name < magicSkills[j].Name
	})
	sort.Slice(utilitySkills, func(i, j int) bool {
		return utilitySkills[i].Name < utilitySkills[j].Name
	})

	var output strings.Builder
	output.WriteString("╔══════════════════════════════════════════════════════════════╗\r\n")
	output.WriteString("║                    Available Skills                         ║\r\n")
	output.WriteString("╠══════════════════════════════════════════════════════════════╣\r\n")

	// Helper function to add skill section
	addSkillSection := func(title string, skills []*engine.Skill) {
		if len(skills) > 0 {
			fmt.Fprintf(&output, "║ %-58s ║\r\n", title)
			output.WriteString("╠══════════════════════════════╦════════╦══════════════════╣\r\n")
			output.WriteString("║ Skill                        ║ Diff.  ║ Status           ║\r\n")
			output.WriteString("╠══════════════════════════════╬════════╬══════════════════╣\r\n")

			for _, skill := range skills {
				// Truncate display name if too long
				displayName := skill.DisplayName
				if len(displayName) > 26 {
					displayName = displayName[:23] + "..."
				}

				status := "Available"
				if skill.Learned {
					status = fmt.Sprintf("Learned (Lvl %d)", skill.Level)
				} else {
					// Check requirements
					var stat int
					switch skill.Type {
					case engine.SkillTypeCombat:
						stat = player.GetStr()
					case engine.SkillTypeMagic:
						stat = player.GetInt()
					case engine.SkillTypeUtility:
						stat = player.GetDex()
					}

					if !skill.CanLearn(player.GetLevel(), stat) {
						status = "Requirements"
					}
				}

				fmt.Fprintf(&output, "║ %-26s ║ %6d ║ %-16s ║\r\n",
					displayName, skill.Difficulty, status)
			}
			output.WriteString("╠══════════════════════════════╩════════╩══════════════════╣\r\n")
		}
	}

	addSkillSection("Combat Skills", combatSkills)
	addSkillSection("Magic Skills", magicSkills)
	addSkillSection("Utility Skills", utilitySkills)

	output.WriteString("║                                                          ║\r\n")
	output.WriteString("║ Use 'learn <skill>' to learn a new skill.                ║\r\n")
	output.WriteString("║ Use 'practice <skill>' to improve a learned skill.       ║\r\n")
	output.WriteString("╚══════════════════════════════════════════════════════════════╝\r\n")

	// Add skill points info
	points := skillManager.GetSkillPoints()
	availableSlots := skillManager.GetAvailableSlots()

	fmt.Fprintf(&output, "\r\nSkill points: %d | Available slots: %d\r\n",
		points, availableSlots)

	return s.SendMessage(output.String())
}

// cmdForget forgets a skill
func CmdForget(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}

	if len(args) == 0 {
		return s.SendMessage("Forget what? Usage: forget <skill>\r\n")
	}

	skillName := strings.ToLower(strings.Join(args, " "))
	player := s.GetPlayer()
	skillManager := player.SkillManager

	if skillManager == nil {
		return s.SendMessage("You have no skills to forget.\r\n")
	}

	// Check if skill exists and is learned
	skill := skillManager.GetSkill(skillName)
	if skill == nil || !skill.Learned {
		return s.SendMessage(fmt.Sprintf("You haven't learned '%s'.\r\n", skillName))
	}

	// Confirm forget
	output := fmt.Sprintf("Are you sure you want to forget %s (Level %d)?\r\n",
		skill.DisplayName, skill.Level)
	output += "This will refund half the skill points spent. Type 'confirm forget' to proceed.\r\n"

	_ = s.SendMessage(output)

	// Store the skill to forget in session context
	s.SetTempData("skill_to_forget", skillName)

	return nil
}

// cmdConfirmForget confirms forgetting a skill
func CmdConfirmForget(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}

	skillName, ok := s.GetTempData("skill_to_forget").(string)
	if !ok || skillName == "" {
		return s.SendMessage("No skill pending to forget.\r\n")
	}

	player := s.GetPlayer()
	skillManager := player.SkillManager

	if skillManager == nil {
		return s.SendMessage("You have no skills to forget.\r\n")
	}

	// Forget the skill
	success := skillManager.ForgetSkill(skillName)
	if success {
		// Clear the temp data
		s.ClearTempData("skill_to_forget")
		return s.SendMessage(fmt.Sprintf("You forget %s and regain some skill points.\r\n", skillName))
	} else {
		return s.SendMessage(fmt.Sprintf("Failed to forget %s.\r\n", skillName))
	}
}

// cmdUseSkill uses a skill (generic skill check)
func CmdUseSkill(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}

	if len(args) == 0 {
		return s.SendMessage("Use what skill? Usage: use <skill> [target]\r\n")
	}

	skillName := strings.ToLower(args[0])
	player := s.GetPlayer()
	skillManager := player.SkillManager

	if skillManager == nil {
		return s.SendMessage("You have no skills to use.\r\n")
	}

	// Check if skill exists and is learned
	skill := skillManager.GetSkill(skillName)
	if skill == nil || !skill.Learned {
		return s.SendMessage(fmt.Sprintf("You haven't learned '%s'.\r\n", skillName))
	}

	// Determine target level (default to player's level)
	targetLevel := player.GetLevel()
	targetName := ""

	if len(args) > 1 {
		targetName = strings.Join(args[1:], " ")
		// In a real MUD, we would look up the target's level here
		// For now, use player's level + random offset
		targetLevel = player.GetLevel() + (s.RandomInt(5) - 2) // -2 to +2
		if targetLevel < 1 {
			targetLevel = 1
		}
	}

	// Determine which stat to use
	var stat int
	switch skill.Type {
	case engine.SkillTypeCombat:
		stat = player.GetStr()
	case engine.SkillTypeMagic:
		stat = player.GetInt()
	case engine.SkillTypeUtility:
		stat = player.GetDex()
	}

	// Use the skill
	success, improved := skillManager.UseSkill(skillName, player.GetLevel(), stat, targetLevel)

	var output strings.Builder
	fmt.Fprintf(&output, "You attempt to use %s", skill.DisplayName)

	if targetName != "" {
		fmt.Fprintf(&output, " on %s", targetName)
	}
	output.WriteString("... ")

	if success {
		output.WriteString("Success!\r\n")
	} else {
		output.WriteString("Failed.\r\n")
	}

	if improved {
		output.WriteString("You feel like you've improved your understanding of this skill.\r\n")
	}

	return s.SendMessage(output.String())
}

// cmdSkillInfo shows detailed information about a skill
func CmdSkillInfo(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}

	if len(args) == 0 {
		return s.SendMessage("Info on what skill? Usage: skillinfo <skill>\r\n")
	}

	skillName := strings.ToLower(strings.Join(args, " "))
	player := s.GetPlayer()
	skillManager := player.SkillManager

	if skillManager == nil {
		return s.SendMessage("You have no skills.\r\n")
	}

	skill := skillManager.GetSkill(skillName)
	if skill == nil {
		return s.SendMessage(fmt.Sprintf("Skill '%s' doesn't exist.\r\n", skillName))
	}

	var output strings.Builder
	output.WriteString("╔══════════════════════════════════════════════════════╗\r\n")
	fmt.Fprintf(&output, "║ %-50s ║\r\n", skill.DisplayName)
	output.WriteString("╠══════════════════════════════════════════════════════╣\r\n")

	// Skill type
	typeStr := "Utility"
	switch skill.Type {
	case engine.SkillTypeCombat:
		typeStr = "Combat"
	case engine.SkillTypeMagic:
		typeStr = "Magic"
	}
	fmt.Fprintf(&output, "║ Type: %-44s ║\r\n", typeStr)

	// Difficulty
	fmt.Fprintf(&output, "║ Difficulty: %-40d ║\r\n", skill.Difficulty)

	// Status
	if skill.Learned {
		fmt.Fprintf(&output, "║ Status: Learned (Level %d) %30s ║\r\n", skill.Level, "")
		fmt.Fprintf(&output, "║ Proficiency: %-38s ║\r\n", skill.GetDisplayLevel())
		fmt.Fprintf(&output, "║ Practice Progress: %d%% %32s ║\r\n", skill.GetProgress(), "")
	} else {
		output.WriteString("║ Status: Not learned %36s ║\r\n")

		// Check requirements
		var stat int
		switch skill.Type {
		case engine.SkillTypeCombat:
			stat = player.GetStr()
		case engine.SkillTypeMagic:
			stat = player.GetInt()
		case engine.SkillTypeUtility:
			stat = player.GetDex()
		}

		if skill.CanLearn(player.GetLevel(), stat) {
			output.WriteString("║ Requirements: Met %37s ║\r\n")
		} else {
			output.WriteString("║ Requirements: Not met %35s ║\r\n")
			fmt.Fprintf(&output, "║ Needed: Level %d, Stat %d %30s ║\r\n",
				skill.Difficulty, 10, "")
		}
	}

	output.WriteString("╚══════════════════════════════════════════════════════╝\r\n")

	if skill.Learned {
		fmt.Fprintf(&output, "\r\nUse 'practice %s' to improve this skill.\r\n", skill.Name)
	} else {
		fmt.Fprintf(&output, "\r\nUse 'learn %s' to learn this skill.\r\n", skill.Name)
	}

	return s.SendMessage(output.String())
}

// ---------------------------------------------------------------------------
// Dark Pawns skill commands — backstab, bash, kick, trip, rescue, sneak, hide, steal, pick
// ---------------------------------------------------------------------------

// CmdBackstab handles the backstab command.
func CmdBackstab(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}
	ch := s.GetPlayer()
	// C do_backstab (act.offensive.c:166) checks GET_SKILL(BACKSTAB) BEFORE the
	// target lookup — a no-skill caller gets "You have no idea how." regardless
	// of args (subcmd==0 returns). CanUseSkill carries that exact message
	// (SkillUnknownMsg, DP-1206).
	canUse, msg := game.CanUseSkill(ch, game.SkillBackstab)
	if !canUse {
		return s.SendMessage(msg)
	}
	if len(args) == 0 {
		return s.SendMessage("Backstab who?\r\n")
	}

	// C do_backstab uses one_argument: skip fill words, lowercase the first
	// target token, and ignore the remainder (act.offensive.c:174).
	targetName, _ := game.OneArgument(strings.Join(args, " "))
	world := s.GetWorld()
	target, _, found := game.FindTargetInRoom(world, ch.GetRoom(), targetName, ch)
	if !found {
		return s.SendMessage("Backstab who?\r\n")
	}

	// Can't backstab self
	if target.GetName() == ch.Name {
		return s.SendMessage("How can you sneak up on yourself?\r\n")
	}

	result := game.DoBackstab(ch, target, world)
	return sendSkillResult(s, ch, target, result)
}

// CmdBash handles the bash command.
func CmdBash(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}

	ch := s.GetPlayer()
	canUse, msg := game.CanUseSkill(ch, game.SkillBash)
	if !canUse {
		return s.SendMessage(msg)
	}

	// C do_bash checks ROOM_PEACEFUL before looking up the target
	// (act.offensive.c:435-439), so this early return must stay at the command
	// layer rather than relying only on DoBash after target resolution.
	world := s.GetWorld()
	if world != nil && world.RoomHasFlag(ch.GetRoom(), "peaceful") {
		return s.SendMessage("This room just has such a peaceful, easy feeling...\r\n")
	}

	// Find target — if in combat, default to fighting target
	var target combat.Combatant
	var found bool
	if len(args) > 0 {
		// C do_bash uses one_argument: skip fill words and ignore the remainder
		// after the first target token (act.offensive.c:425).
		targetName, _ := game.OneArgument(strings.Join(args, " "))
		target, _, found = game.FindTargetInRoom(world, ch.GetRoom(), targetName, ch)
		if !found {
			return s.SendMessage("Bash who?\r\n")
		}
	} else if ch.GetFighting() != "" {
		target, _, found = game.FindTargetInRoom(world, ch.GetRoom(), ch.GetFighting(), ch)
		if !found {
			return s.SendMessage("Bash who?\r\n")
		}
	} else {
		return s.SendMessage("Bash who?\r\n")
	}

	if target.GetName() == ch.Name {
		return s.SendMessage("Aren't we funny today...\r\n")
	}

	result := game.DoBash(ch, target, world)
	return sendSkillResult(s, ch, target, result)
}

// CmdKick handles the kick command.
func CmdKick(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}

	ch := s.GetPlayer()
	canUse, msg := game.CanUseSkill(ch, game.SkillKick)
	if !canUse {
		return s.SendMessage(msg)
	}

	var target combat.Combatant
	var found bool
	world := s.GetWorld()
	if len(args) > 0 {
		// C do_kick uses one_argument: skip fill words, lowercase the first
		// target token, and ignore the remainder (act.offensive.c:600).
		targetName, _ := game.OneArgument(strings.Join(args, " "))
		target, _, found = game.FindTargetInRoom(world, ch.GetRoom(), targetName, ch)
		if !found {
			return s.SendMessage("Kick who?\r\n")
		}
	} else if ch.GetFighting() != "" {
		target, _, found = game.FindTargetInRoom(world, ch.GetRoom(), ch.GetFighting(), ch)
		if !found {
			// C uses FIGHTING(ch) as a direct pointer after an empty
			// argument; a mob's multi-word short description is not reparsed
			// through get_char_room_vis (act.offensive.c:601-605).
			target, found = game.FindFightingTargetInRoom(world, ch.GetRoom(), ch.GetFighting(), ch)
		}
		if !found {
			return s.SendMessage("Kick who?\r\n")
		}
	} else {
		return s.SendMessage("Kick who?\r\n")
	}

	if target.GetName() == ch.Name {
		return s.SendMessage("Aren't we funny today...\r\n")
	}

	result := game.DoKick(ch, target)
	return sendSkillResult(s, ch, target, result)
}

// CmdTrip handles the trip command.
func CmdTrip(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}

	ch := s.GetPlayer()
	canUse, msg := game.CanUseSkill(ch, game.SkillTrip)
	if !canUse {
		return s.SendMessage(msg)
	}

	var target combat.Combatant
	var found bool
	world := s.GetWorld()
	if len(args) > 0 {
		targetName := strings.Join(args, " ")
		target, _, found = game.FindTargetInRoom(world, ch.GetRoom(), targetName, ch)
		if !found {
			return s.SendMessage("Trip who?\r\n")
		}
	} else if ch.GetFighting() != "" {
		target, _, found = game.FindTargetInRoom(world, ch.GetRoom(), ch.GetFighting(), ch)
		if !found {
			return s.SendMessage("Trip who?\r\n")
		}
	} else {
		return s.SendMessage("Trip who?\r\n")
	}

	if target.GetName() == ch.Name {
		return s.SendMessage("You trip over your shoe laces...\r\n")
	}

	result := game.DoTrip(ch, target, world)
	return sendSkillResult(s, ch, target, result)
}

// CmdHeadbutt handles the headbutt command.
func CmdHeadbutt(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}

	ch := s.GetPlayer()
	world := s.GetWorld()
	// C do_headbutt (new_cmds.c:378): ROOM_PEACEFUL is checked BEFORE the skill
	// gate (unlike bash/kick), so a peaceful room rejects even a no-skill caller.
	if world != nil && world.RoomHasFlag(ch.GetRoom(), "peaceful") {
		return s.SendMessage("The Gods prevent thy violent act.\r\n")
	}
	canUse, msg := game.CanUseSkill(ch, game.SkillHeadbutt)
	if !canUse {
		return s.SendMessage(msg)
	}

	var target combat.Combatant
	var found bool
	if len(args) > 0 {
		// C one_argument consumes only the first word and ignores the remainder.
		targetName, _ := game.OneArgument(strings.Join(args, " "))
		target, _, found = game.FindTargetInRoom(world, ch.GetRoom(), targetName, ch)
		if !found {
			return s.SendMessage("Headbutt who?\r\n")
		}
	} else if ch.GetFighting() != "" {
		target, _, found = game.FindTargetInRoom(world, ch.GetRoom(), ch.GetFighting(), ch)
		if !found {
			return s.SendMessage("Headbutt who?\r\n")
		}
	} else {
		return s.SendMessage("Headbutt who?\r\n")
	}

	result := game.DoHeadbutt(ch, target, world)
	return sendSkillResult(s, ch, target, result)
}

// CmdRescue handles the rescue command.
func CmdRescue(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}
	ch := s.GetPlayer()
	// C do_rescue (act.offensive.c:501) checks GET_SKILL(RESCUE) BEFORE the
	// no-argument path — a no-skill caller gets "But only true warriors can do
	// this!" regardless of args. CanUseSkill carries that exact message
	// (SkillUnknownMsg, DP-1206).
	canUse, msg := game.CanUseSkill(ch, game.SkillRescue)
	if !canUse {
		return s.SendMessage(msg)
	}
	if len(args) == 0 {
		return s.SendMessage("Whom do you want to rescue?\r\n")
	}

	targetName := strings.Join(args, " ")
	world := s.GetWorld()
	target, _, found := game.FindTargetInRoom(world, ch.GetRoom(), targetName, ch)
	if !found {
		// C do_rescue answers a resolution failure with the same prompt as the
		// no-argument path (act.offensive.c:515-519), not NOPERSON.
		return s.SendMessage("Whom do you want to rescue?\r\n")
	}

	if target.GetName() == ch.Name {
		return s.SendMessage("What about fleeing instead?\r\n")
	}

	combatEngine, ok := s.GetCombatEngine().(rescueCombatEngine)
	if !ok || combatEngine == nil {
		return s.SendMessage("Combat is unavailable right now.\r\n")
	}

	// Execute the rescue
	result := game.DoRescue(ch, target, world, combatEngine)
	return sendSkillResult(s, ch, target, result)
}

// CmdDisembowel handles the disembowel command (C-10).
func CmdDisembowel(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}
	ch := s.GetPlayer()
	if canUse, msg := game.CanUseSkill(ch, game.SkillDisembowel); !canUse {
		return s.SendMessage(msg)
	}
	var target combat.Combatant
	world := s.GetWorld()
	targetName, _ := game.OneArgument(strings.Join(args, " "))
	if targetName != "" {
		resolved, found := world.ResolveCharInRoom(ch, targetName)
		if found {
			target = resolved.Combatant
		}
	}
	if target == nil && ch.GetFighting() != "" {
		resolved, found := world.ResolveFightingTarget(ch)
		if found {
			target = resolved.Combatant
		}
	}
	if target == nil {
		return s.SendMessage("Disembowel who?\r\n")
	}
	if target.GetName() == ch.Name {
		return s.SendMessage("Nah. Hari Kari is for wimps.\r\n")
	}
	return sendSkillResult(s, ch, target, game.DoDisembowel(ch, target))
}

// CmdDragonKick handles the dragon kick command (C-10).
func CmdDragonKick(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}
	ch := s.GetPlayer()
	if canUse, msg := game.CanUseSkill(ch, game.SkillDragonKick); !canUse {
		return s.SendMessage(msg)
	}
	var target combat.Combatant
	var found bool
	world := s.GetWorld()
	if len(args) > 0 {
		targetName, _ := game.OneArgument(strings.Join(args, " "))
		target, _, found = game.FindTargetInRoom(world, ch.GetRoom(), targetName, ch)
		if !found {
			return s.SendMessage("Kick who?\r\n")
		}
	} else if ch.GetFighting() != "" {
		target, _, found = game.FindTargetInRoom(world, ch.GetRoom(), ch.GetFighting(), ch)
		if !found {
			return s.SendMessage("Kick who?\r\n")
		}
	} else {
		return s.SendMessage("Kick who?\r\n")
	}
	if target.GetName() == ch.Name {
		return s.SendMessage("Aren't we funny today...\r\n")
	}
	return sendSkillResult(s, ch, target, game.DoDragonKick(ch, target))
}

// CmdTigerPunch handles the tiger punch command (C-10).
func CmdTigerPunch(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}
	ch := s.GetPlayer()
	if canUse, msg := game.CanUseSkill(ch, game.SkillTigerPunch); !canUse {
		return s.SendMessage(msg)
	}
	var target combat.Combatant
	var found bool
	world := s.GetWorld()
	if len(args) > 0 {
		target, _, found = game.FindTargetInRoom(world, ch.GetRoom(), strings.Join(args, " "), ch)
		if !found {
			return s.SendMessage("Tiger punch whom?\r\n")
		}
	} else if ch.GetFighting() != "" {
		target, _, found = game.FindTargetInRoom(world, ch.GetRoom(), ch.GetFighting(), ch)
		if !found {
			return s.SendMessage("Tiger punch whom?\r\n")
		}
	} else {
		return s.SendMessage("Tiger punch whom?\r\n")
	}
	if target.GetName() == ch.Name {
		return s.SendMessage("Aren't we funny today...\r\n")
	}
	return sendSkillResult(s, ch, target, game.DoTigerPunch(ch, target))
}

// CmdShoot handles the shoot command (C-10).
func CmdShoot(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}
	ch := s.GetPlayer()
	if canUse, msg := game.CanUseSkill(ch, game.SkillShoot); !canUse {
		return s.SendMessage(msg)
	}
	// C half_chop() consumes three fields before any object or direction
	// lookup (act.offensive.c:782-799). Keep those parser gates ahead of every
	// later branch; the old Go handler invented a same-room target form.
	if len(args) == 0 {
		return s.SendMessage("Shoot what where?\r\n")
	}
	if len(args) == 1 {
		return s.SendMessage("Where would you like to shoot it?\r\n")
	}
	if len(args) == 2 {
		return s.SendMessage("Who would you like to shoot it at in that direction?\r\n")
	}

	world := s.GetWorld()
	projectileName := args[0]
	directionName := strings.ToLower(args[1])
	targetName := strings.Join(args[2:], " ")
	directions := map[string]string{
		"north": "north", "n": "north",
		"east": "east", "e": "east",
		"south": "south", "s": "south",
		"west": "west", "w": "west",
		"up": "up", "u": "up",
		"down": "down", "d": "down",
	}
	direction, validDirection := directions[directionName]
	if !validDirection {
		return s.SendMessage("Interesting direction.\r\n")
	}

	projectile, found := world.ResolveObjectInInventory(ch, projectileName)
	if !found {
		return s.SendMessage(fmt.Sprintf("You don't seem to have any %ss.\r\n", projectileName))
	}
	if projectile.GetTypeFlag() != int(game.ItemMissile) {
		return s.SendMessage(game.CapitalizeSentence(projectile.GetShortDesc()+" is not a projectile!") + "\r\n")
	}

	room := world.GetRoomInWorld(ch.GetRoom())
	if room == nil {
		return s.SendMessage("You are nowhere.\r\n")
	}
	exit, ok := room.Exits[direction]
	if !ok {
		return s.SendMessage("Interesting Direction.\r\n")
	}
	if ch.Equipment == nil {
		return s.SendMessage("You must wield a bow or sling to fire a projectile.\r\n")
	}
	bow, wielded := ch.Equipment.GetItemInSlot(game.SlotWield)
	if !wielded || bow == nil || bow.GetTypeFlag() != int(game.ItemFireWeapon) {
		return s.SendMessage("You must wield a bow or sling to fire a projectile.\r\n")
	}

	if exit.ToRoom < 0 {
		return s.SendMessage("Alas, you cannot shoot that way...\r\n")
	}
	if exit.ExitInfo&parser.ExitClosed != 0 {
		if keyword := strings.Fields(exit.Keywords); len(keyword) > 0 {
			return s.SendMessage(fmt.Sprintf("The %s seems to be closed.\r\n", keyword[0]))
		}
		return s.SendMessage("It seems to be closed.\r\n")
	}
	targetRoom := world.GetRoomInWorld(exit.ToRoom)
	if targetRoom == nil {
		return s.SendMessage("Alas, you cannot shoot that way...\r\n")
	}
	if targetRoom.HasFlag(4) || room.HasFlag(4) {
		return s.SendMessage("You feel too peaceful to contemplate violence.\r\n")
	}

	// C falls back to the first person in the destination room when the named
	// lookup misses. Preserve the explicit no-target branch for now; the
	// hit/miss state machine remains a separate depth case.
	targetInfo, found := world.ResolveCharInRoomAt(ch, exit.ToRoom, targetName)
	if !found {
		if err := world.MoveObjectToRoom(projectile, exit.ToRoom); err != nil {
			return fmt.Errorf("drop projectile in destination room: %w", err)
		}
		return s.SendMessage("Twang...\r\n")
	}
	target := targetInfo.Combatant
	if mob, ok := target.(*game.MobInstance); ok && mob.HasFlag(game.MobSentinel) {
		return s.SendMessage("You cannot see well enough to aim...\r\n")
	}

	result := game.DoShoot(ch, target)
	err := sendSkillResult(s, ch, target, result)
	if err != nil {
		return err
	}

	// On hit, drag target into shooter's room (C: char_from_room + char_to_room)
	if result.Success && target != nil {
		if mover, ok := target.(interface{ SetRoom(int) }); ok {
			world.MovePlayerToRoom(mover, ch.GetRoom())
		}
	}

	return nil
}

// CmdSubdue handles the subdue command (C-10).
func CmdSubdue(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}
	ch := s.GetPlayer()
	if canUse, msg := game.CanUseSkill(ch, game.SkillSubdue); !canUse {
		return s.SendMessage(msg)
	}
	if len(args) == 0 {
		return s.SendMessage("Subdue who?\r\n")
	}
	world := s.GetWorld()
	target, _, found := game.FindTargetInRoom(world, ch.GetRoom(), strings.Join(args, " "), ch)
	if !found {
		return s.SendMessage("They aren't here.\r\n")
	}
	if target.GetName() == ch.Name {
		return s.SendMessage("Aren't we funny today...\r\n")
	}
	return sendSkillResult(s, ch, target, game.DoSubdue(ch, target))
}

// CmdSleeper handles the sleeper hold command (C-10).
func CmdSleeper(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}
	ch := s.GetPlayer()
	if canUse, msg := game.CanUseSkill(ch, game.SkillSleeper); !canUse {
		return s.SendMessage(msg)
	}
	if len(args) == 0 {
		return s.SendMessage("Sleeper who?\r\n")
	}
	world := s.GetWorld()
	// C one_argument() keeps only the first target token and discards the rest.
	targetName, _ := game.OneArgument(strings.Join(args, " "))
	target, _, found := game.FindTargetInRoom(world, ch.GetRoom(), targetName, ch)
	if !found {
		return s.SendMessage("Sleeper who?\r\n")
	}
	return sendSkillResult(s, ch, target, game.DoSleeper(ch, target, world))
}

// CmdNeckbreak handles the neck break command (C-10).
func CmdNeckbreak(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}
	ch := s.GetPlayer()
	if canUse, msg := game.CanUseSkill(ch, game.SkillNeckbreak); !canUse {
		return s.SendMessage(msg)
	}
	// C checks WEAR_WIELD before one_argument() and target lookup
	// (act.offensive.c:1304-1308). Keep this gate in the command wrapper so
	// an invalid target cannot move it after the parser-visible boundary.
	if ch.Equipment != nil {
		if _, wielded := ch.Equipment.GetItemInSlot(game.SlotWield); wielded {
			return s.SendMessage("You can't do this and wield a weapon at the same time!\r\n")
		}
	}
	// C one_argument() discards fill words and keeps only the first token
	// (interpreter.c:1265-1285), so "the victim trailing" targets victim.
	targetName, _ := game.OneArgument(strings.Join(args, " "))
	world := s.GetWorld()
	target, _, found := game.FindTargetInRoom(world, ch.GetRoom(), targetName, ch)
	if !found {
		return s.SendMessage("I don't see them here.\r\n")
	}
	return sendSkillResult(s, ch, target, game.DoNeckbreak(ch, target, world))
}

// CmdAmbush handles the ambush command (C-10).
func CmdAmbush(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}
	ch := s.GetPlayer()
	// C do_ambush (act.offensive.c:1459): target lookup runs FIRST — no-arg →
	// "Ambush who?". The GET_SKILL gate (line 1467) is AFTER target, so
	// reposition it here (never delete — per the scout #541 lesson).
	if len(args) == 0 {
		return s.SendMessage("Ambush who?\r\n")
	}
	world := s.GetWorld()
	target, _, found := game.FindTargetInRoom(world, ch.GetRoom(), strings.Join(args, " "), ch)
	if !found {
		return s.SendMessage("Ambush who?\r\n")
	}
	if canUse, msg := game.CanUseSkill(ch, game.SkillAmbush); !canUse {
		return s.SendMessage(msg)
	}
	if ch.GetAmbushAction() != 0 {
		return s.SendMessage("You are a little busy for that right now!\r\n")
	}
	if target.GetName() == ch.Name {
		return s.SendMessage("Ambush yourself? You idiot!\r\n")
	}
	room := world.GetRoomInWorld(ch.GetRoom())
	if room == nil || (room.Sector != game.SECT_FOREST && room.Sector != game.SECT_HILLS &&
		room.Sector != game.SECT_MOUNTAIN && room.Sector != game.SECT_CITY) {
		return s.SendMessage("Ambush someone here? Impossible!\r\n")
	}
	if target.GetFighting() != "" {
		return s.SendMessage("They're too alert for that, currently.\r\n")
	}
	gameWorld := s.GetWorld()
	gameWorld.PlanAmbush(ch, target)
	return nil
}

// CmdBerserk handles the berserk command (C-10/C-12).
func CmdBerserk(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}
	ch := s.GetPlayer()
	return sendSkillResult(s, ch, nil, game.DoBerserk(ch))
}

// CmdKujiKiri returns a command handler bound to the given kuji-kiri seal
// (one of the game.SkillKk* constants), used for the nine seal commands
// (rin, kyo, toh, kai, jin, retsu, zai, zhen, sha).
func CmdKujiKiri(seal string) func(SessionInterface, []string) error {
	return func(s SessionInterface, args []string) error {
		if s.GetPlayer() == nil {
			return fmt.Errorf("not logged in")
		}
		ch := s.GetPlayer()
		return sendSkillResult(s, ch, nil, game.DoKujiKiri(ch, seal, s.GetWorld()))
	}
}

// CmdSneak handles the sneak command.
func CmdSneak(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}

	ch := s.GetPlayer()
	result := game.DoSneak(ch)
	return s.SendMessage(result.MessageToCh + "\r\n")
}

// CmdStealth handles the stealth command. C do_stealth == do_sneak with the
// stealth skill/affect; routes through game.DoStealth (self-only, no target).
func CmdStealth(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}

	ch := s.GetPlayer()
	result := game.DoStealth(ch)
	return s.SendMessage(result.MessageToCh + "\r\n")
}

// CmdHide handles the hide command.
func CmdHide(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}

	ch := s.GetPlayer()
	result := game.DoHideInWorld(ch, s.GetWorld())
	return s.SendMessage(result.MessageToCh + "\r\n")
}

// CmdKabuki handles the kabuki command (do_hide SCMD_KABUKI, src/act.other.c:247-306).
// Same flow as hide but rolls against SkillKabuki and uses the kabuki message.
func CmdKabuki(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}

	ch := s.GetPlayer()
	result := game.DoKabukiInWorld(ch, s.GetWorld())
	return s.SendMessage(result.MessageToCh + "\r\n")
}

// CmdSteal handles the steal command.
func CmdSteal(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}

	// C runs one_argument() twice: the first token is the object and the
	// second token is the room target. This also skips leading fill words and
	// discards trailing input after the target.
	itemName, rest := game.OneArgument(strings.Join(args, " "))
	targetName, _ := game.OneArgument(rest)
	if itemName == "" || targetName == "" {
		return s.SendMessage("Steal what from who?\r\n")
	}

	ch := s.GetPlayer()
	world := s.GetWorld()

	target, _, found := game.FindTargetInRoom(world, ch.GetRoom(), targetName, ch)
	if !found {
		return s.SendMessage("Steal what from who?\r\n")
	}

	result := game.DoSteal(ch, target, itemName, world)
	return sendSkillResult(s, ch, target, result)
}

// CmdCarve handles the carve command.
func CmdCarve(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}

	ch := s.GetPlayer()
	if ch.GetPosition() == combat.PosFighting {
		return s.SendMessage("How can you think of food at a time like this?!?\r\n")
	}

	targetName, _ := game.OneArgument(strings.Join(args, " "))
	if targetName == "" {
		return s.SendMessage("You want to carve what?!?\r\n")
	}
	world := s.GetWorld()
	result := game.DoCarve(ch, targetName, world)
	return sendSkillResult(s, ch, nil, result)
}

// CmdCutthroat handles the cutthroat command.
func CmdCutthroat(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}
	ch := s.GetPlayer()
	// C do_cutthroat (new_cmds.c:559): GET_SKILL(CUTTHROAT) checked BEFORE the
	// no-arg path — a no-skill caller gets "You're not trained in slitting
	// throats!" regardless of args.
	canUse, msg := game.CanUseSkill(ch, game.SkillCutthroat)
	if !canUse {
		// msg carries its own C-exact terminator (cutthroat's is "\n\r",
		// new_cmds.c:561; see SkillUnknownMsg). Send as-is — no append.
		return s.SendMessage(msg)
	}
	if len(args) == 0 {
		return s.SendMessage("Cut what throat where?\n\r")
	}

	// C one_argument() runs before get_char_room_vis(), so only the first
	// whitespace-delimited target token participates in lookup.
	targetName, _ := game.OneArgument(strings.Join(args, " "))
	world := s.GetWorld()
	target, _, found := game.FindTargetInRoom(world, ch.GetRoom(), targetName, ch)
	if !found {
		return s.SendMessage("Cut what throat where?\n\r")
	}

	if target.GetName() == ch.Name {
		return s.SendMessage("That would be bad.\n\r")
	}

	result := game.DoCutthroat(ch, target, world)
	return sendSkillResult(s, ch, target, result)
}

// CmdStrike handles the strike command.
func CmdStrike(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}

	ch := s.GetPlayer()
	canUse, msg := game.CanUseSkill(ch, game.SkillStrike)
	if !canUse {
		return s.SendMessage(msg)
	}

	// Determine target
	var target combat.Combatant
	var found bool

	if len(args) == 0 {
		// Try to strike whoever we're fighting
		fighting := ch.GetFighting()
		if fighting == "" {
			return s.SendMessage("Strike who?\r\n")
		}
		// Find fighter by name
		world := s.GetWorld()
		target, _, found = game.FindTargetInRoom(world, ch.GetRoom(), fighting, ch)
		if !found {
			return s.SendMessage("They don't seem to be here.\r\n")
		}
	} else {
		targetName := strings.Join(args, " ")
		world := s.GetWorld()
		target, _, found = game.FindTargetInRoom(world, ch.GetRoom(), targetName, ch)
		if !found {
			return s.SendMessage("They don't seem to be here.\r\n")
		}
	}

	if target.GetName() == ch.Name {
		ch.SendMessage("You beat yourself about the face and neck.\r\n")
		// Send room act
		roomVNum := ch.GetRoom()
		world := s.GetWorld()
		players := world.GetPlayersInRoom(roomVNum)
		for _, p := range players {
			if p.Name != ch.Name {
				p.SendMessage(fmt.Sprintf("%s slaps %s around a little.\r\n",
					ch.Name, genderPronoun(ch.Sex)))
			}
		}
		return nil
	}

	result := game.DoStrike(ch, target)
	return sendSkillResult(s, ch, target, result)
}

// CmdCompare handles the compare command — a faithful port of do_compare
// (src/new_cmds.c:1952). C half_chops the argument into (arg, arg2) — the first
// word and the remainder — and does NOT gate the command on a skill (APPRAISE
// only sets the success probability inside DoCompare). The prior Go wrapper
// invented a CanUseSkill gate, a "compare to equipped" path, and an unreachable
// "Compare what and what?" — all removed here.
func CmdCompare(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}
	ch := s.GetPlayer()

	// half_chop: first word → arg, the rest → arg2.
	arg, arg2 := "", ""
	if len(args) > 0 {
		arg = args[0]
	}
	if len(args) > 1 {
		arg2 = strings.Join(args[1:], " ")
	}

	result := game.DoCompare(ch, arg, arg2)
	return sendSkillResult(s, ch, nil, result)
}

// CmdScan handles the scan command.
func CmdScan(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}

	ch := s.GetPlayer()
	canUse, msg := game.CanUseSkill(ch, game.SkillScan)
	if !canUse {
		return s.SendMessage(msg)
	}

	world := s.GetWorld()
	result := game.DoScan(ch, world)
	return sendSkillResult(s, ch, nil, result)
}

// CmdSharpen handles the sharpen command.
func CmdSharpen(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}
	ch := s.GetPlayer()
	objName := ""
	if len(args) > 0 {
		objName = args[0]
	}
	result := game.DoSharpen(ch, objName)
	return sendSkillResult(s, ch, nil, result)
}

// CmdScrounge handles the scrounge command.
func CmdScrounge(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}

	ch := s.GetPlayer()
	world := s.GetWorld()
	result := game.DoScrounge(ch, world)
	return sendSkillResult(s, ch, nil, result)
}

// CmdFirstAid handles the first aid command.
func CmdFirstAid(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}
	ch := s.GetPlayer()
	// C do_first_aid (new_cmds2.c:146): GET_SKILL checked BEFORE the no-arg
	// path — a no-skill caller gets "You have no idea how!" regardless of args.
	canUse, msg := game.CanUseSkill(ch, game.SkillFirstAid)
	if !canUse {
		return s.SendMessage(msg)
	}
	if len(args) == 0 {
		return s.SendMessage("Aid who?\r\n")
	}

	targetName := strings.Join(args, " ")
	world := s.GetWorld()
	target, _, found := game.FindTargetInRoom(world, ch.GetRoomVNum(), targetName, ch)
	if !found {
		return s.SendMessage("They don't seem to be here.\r\n")
	}

	result := game.DoFirstAid(ch, target)
	return sendSkillResult(s, ch, target, result)
}

// CmdDisarm handles the disarm command.
func CmdDisarm(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}

	ch := s.GetPlayer()
	// C do_disarm (new_cmds2.c): GET_SKILL(DISARM) is checked BEFORE the target
	// lookup — a no-skill caller is rejected regardless of args. CanUseSkill
	// carries the exact C message (SkillUnknownMsg, DP-1206).
	canUse, msg := game.CanUseSkill(ch, game.SkillDisarm)
	if !canUse {
		return s.SendMessage(msg)
	}

	// C resolves FIGHTING(ch) before looking at the typed argument. The
	// argument is therefore ignored whenever combat is already engaged.
	var target combat.Combatant
	world := s.GetWorld()

	if ch.GetFighting() != "" {
		resolved, found := world.ResolveFightingTarget(ch)
		if !found {
			// A live C FIGHTING pointer cannot be absent from the room. Keep the
			// command's only C target failure text if the Go state is stale.
			return s.SendMessage("Disarm who?\r\n")
		}
		target = resolved.Combatant
	} else {
		targetName, _ := game.OneArgument(strings.Join(args, " "))
		if targetName == "" {
			return s.SendMessage("Disarm who?\r\n")
		}
		var found bool
		target, _, found = game.FindTargetInRoom(world, ch.GetRoomVNum(), targetName, ch)
		if !found {
			return s.SendMessage("Disarm who?\r\n")
		}
	}

	result := game.DoDisarm(ch, target, world)
	return sendSkillResult(s, ch, target, result)
}

// CmdMindlink handles the mindlink command.
func CmdMindlink(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}
	if len(args) == 0 {
		return s.SendMessage("Link your mind to whose?\r\n")
	}

	ch := s.GetPlayer()
	targetName, _ := game.OneArgument(strings.Join(args, " "))
	world := s.GetWorld()
	target, _, found := game.FindTargetInRoom(world, ch.GetRoomVNum(), targetName, ch)
	if !found {
		return s.SendMessage("They don't seem to be here.\r\n")
	}

	result := game.DoMindlink(ch, target)
	return sendSkillResult(s, ch, target, result)
}

// CmdDetect handles the detect command.
func CmdDetect(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}

	ch := s.GetPlayer()
	world := s.GetWorld()
	result := game.DoDetect(ch, world)
	return sendSkillResult(s, ch, nil, result)
}

// CmdSerpentKick handles the serpent kick command.
func CmdSerpentKick(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}

	ch := s.GetPlayer()
	// C do_serpent_kick (new_cmds2.c:698): GET_SKILL checked BEFORE target
	// lookup — a no-skill caller is rejected regardless of args.
	canUse, msg := game.CanUseSkill(ch, game.SkillSerpentKick)
	if !canUse {
		return s.SendMessage(msg)
	}

	var target combat.Combatant
	var found bool
	world := s.GetWorld()

	if len(args) == 0 {
		// Try to kick whoever we're fighting
		fighting := ch.GetFighting()
		if fighting == "" {
			return s.SendMessage("Kick who?\r\n")
		}
		target, found = game.FindFightingTargetInRoom(world, ch.GetRoomVNum(), fighting, ch)
		if !found {
			return s.SendMessage("Kick who?\r\n")
		}
	} else {
		// C's one_argument consumes only the first token; trailing words are
		// ignored before get_char_room_vis (new_cmds2.c:703-705).
		targetName := args[0]
		target, _, found = game.FindTargetInRoom(world, ch.GetRoomVNum(), targetName, ch)
		if !found {
			// C falls back to FIGHTING(ch) after an unsuccessful named lookup,
			// not only when the command has no argument (new_cmds2.c:705-713).
			fighting := ch.GetFighting()
			if fighting == "" {
				return s.SendMessage("Kick who?\r\n")
			}
			target, found = game.FindFightingTargetInRoom(world, ch.GetRoomVNum(), fighting, ch)
			if !found {
				return s.SendMessage("Kick who?\r\n")
			}
		}
	}

	result := game.DoSerpentKick(ch, target, world)
	return sendSkillResult(s, ch, target, result)
}

// CmdDig handles the dig command.
func CmdDig(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}

	ch := s.GetPlayer()
	world := s.GetWorld()
	result := game.DoDig(ch, world)
	return sendSkillResult(s, ch, nil, result)
}

// CmdTurn handles the turn command.
func CmdTurn(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}

	ch := s.GetPlayer()
	world := s.GetWorld()

	// Turn affects ALL undead in the room, but for simplicity we
	// target a specific enemy if specified or whoever we're fighting
	var target combat.Combatant
	var found bool

	if len(args) == 0 {
		fighting := ch.GetFighting()
		if fighting == "" {
			return s.SendMessage("Turn who?\r\n")
		}
		target, _, found = game.FindTargetInRoom(world, ch.GetRoomVNum(), fighting, ch)
		if !found {
			return s.SendMessage("They don't seem to be here.\r\n")
		}
	} else {
		targetName := strings.Join(args, " ")
		target, _, found = game.FindTargetInRoom(world, ch.GetRoomVNum(), targetName, ch)
		if !found {
			return s.SendMessage("They don't seem to be here.\r\n")
		}
	}

	result := game.DoTurn(ch, target)
	return sendSkillResult(s, ch, target, result)
}

// genderPronoun returns the appropriate "himself" / "herself" / "itself" pronoun.
// Go encoding: 0=male, 1=female, 2=neutral.
func genderPronoun(sex int) string {
	switch sex {
	case game.SexMale:
		return "himself"
	case game.SexFemale:
		return "herself"
	default:
		return "itself"
	}
}

// ---------------------------------------------------------------------------
// Helper: send skill result to player, victim, and room
// ---------------------------------------------------------------------------

func sendSkillResult(s SessionInterface, ch *game.Player, target combat.Combatant, result game.SkillResult) error {
	if target == nil && len(result.Targets) > 0 {
		target = result.Targets[0]
	}
	targets := result.Targets
	if len(targets) == 0 && target != nil {
		targets = []combat.Combatant{target}
	}

	sendSkillMessage := func() {
		if result.SkillMsgType != 0 {
			if eng, ok := s.GetCombatEngine().(rescueCombatEngine); ok && eng != nil {
				for _, skillTarget := range targets {
					if skillTarget != nil {
						eng.SkillMessage(result.Damage, ch.GetName(), skillTarget.GetName(), result.SkillMsgType, ch.GetRoom())
					}
				}
			}
		}
	}
	sendLiteralMessages := func() {
		if result.MessageToCh != "" {
			// C act() CAPs the assembled string (comm.c:2477); lines that begin
			// with $e/$n render lowercase and must be capitalized here.
			_ = s.SendMessage(game.CapitalizeSentence(result.MessageToCh) + "\r\n")
		}
		if result.MessageToVict != "" && target != nil {
			if p, ok := target.(*game.Player); ok {
				p.SendMessage(game.CapitalizeSentence(result.MessageToVict) + "\r\n")
			}
		}
		if result.MessageToRoom != "" {
			roomVNum := ch.GetRoom()
			world := s.GetWorld()
			players := world.GetPlayersInRoom(roomVNum)
			for _, p := range players {
				if p.Name == ch.Name && !result.RoomIncludesActor {
					continue
				}
				if target != nil && !result.RoomIncludesTarget && p.Name == target.GetName() {
					continue
				}
				p.SendMessage(game.CapitalizeSentence(result.MessageToRoom) + "\r\n")
			}
		}
		if result.MessageToRoomSecond != "" {
			roomVNum := ch.GetRoom()
			world := s.GetWorld()
			players := world.GetPlayersInRoom(roomVNum)
			for _, p := range players {
				if p.Name == ch.Name && !result.RoomIncludesActor {
					continue
				}
				if target != nil && !result.RoomIncludesTarget && p.Name == target.GetName() {
					continue
				}
				p.SendMessage(game.CapitalizeSentence(result.MessageToRoomSecond) + "\r\n")
			}
		}
	}
	performRetaliateHit := func() {
		if !result.RetaliateHit || target == nil || target.GetPosition() == combat.PosDead {
			return
		}
		if engine, ok := s.GetCombatEngine().(rescueCombatEngine); ok && engine != nil {
			if err := engine.StartCombat(target, ch); err == nil {
				_ = engine.PerformInitialAttack(target, ch)
			}
		}
	}

	if result.RetaliateHitBeforeSkillMessage {
		performRetaliateHit()
	}

	// Emit the combat message. When SkillMsgType is set, the message comes from
	// lib/misc/messages via the skill_message path (fight.c:1023-1092): the
	// engine draws Dice(1,N) and emits the selected set's char/vict/room text
	// itself, so we do NOT also emit MessageToCh/Vict/Room (R4 — no invented
	// strings; R3 — the Dice draw is DRAW 2, ordered after the skill's
	// number(1,101) in DoBackstab, so it must run before any combat-initiation
	// side effects that could draw). Otherwise emit the literal MessageTo*
	// fields as before.
	if result.SkillMsgAfterDamage {
		sendLiteralMessages()
		for _, skill := range result.PreDamageImprove {
			game.ImproveSkill(ch, skill)
		}
	} else if result.SkillMsgType != 0 && target != nil && !result.SkillMsgInDamage {
		sendSkillMessage()
	} else if result.RetaliateHitAfterMessages {
		sendLiteralMessages()
	} else if result.MessageToCh != "" && !result.MessageToChAfterRoom {
		// C act() CAPs the assembled string (comm.c:2477); lines that begin
		// with $e/$n render lowercase and must be capitalized here.
		_ = s.SendMessage(game.CapitalizeSentence(result.MessageToCh) + "\r\n")
	}

	// C's failed cutthroat arm calls hit(ch, victim, SKILL_CUTTHROAT), which
	// resolves one ordinary weapon attack synchronously after the literal
	// lunge messages. PerformInitialAttack is the shared hit() path used by
	// do_hit and preserves its to-hit, damage, audience, and combat state.
	if result.InitialAttack && target != nil {
		if engine, ok := s.GetCombatEngine().(rescueCombatEngine); ok && engine != nil {
			if err := engine.StartCombat(ch, target); err != nil {
				slog.Error("cutthroat initial attack combat start failed", "attacker", ch.GetName(), "target", target.GetName(), "error", err)
			} else if err := engine.PerformInitialAttack(ch, target); err != nil {
				slog.Error("cutthroat initial attack failed", "attacker", ch.GetName(), "target", target.GetName(), "error", err)
			}
		}
	} else if result.SkillMsgInDamage && target != nil {
		// do_disembowel calls damage(), which owns the post-position skill_message
		// and the death_cry/raw_kill ordering. Keep that complete path together so
		// lethal and zero-damage hit() outcomes both select the right C variant.
		switch result.DamageSkill {
		case game.SkillDisembowel:
			s.GetWorld().DoDisembowelDamage(ch, target, result.Damage)
		case game.SkillGroinrip:
			s.GetWorld().DoGroinripDamage(ch, target, result.Damage)
		case game.SkillNeckbreak:
			s.GetWorld().DoNeckbreakDamage(ch, target, result.Damage)
		case game.SkillSmackheads:
			for _, damageTarget := range targets {
				if damageTarget != nil {
					s.GetWorld().DoSmackheadsDamage(ch, damageTarget, result.Damage)
				}
			}
		}
	} else if result.Damage > 0 && len(targets) > 0 {
		// Route through DoSpellDamage so skill damage uses the same death
		// pipeline as combat and spells: corpse creation, XP award, kill counter,
		// event bus publish, removal from world, and combat initiation for both
		// parties. Previously this only called TakeDamage + printed "is dead!".
		// See DP-942 / pkg/game/damage_stubs.go.
		for _, damageTarget := range targets {
			if damageTarget == nil {
				continue
			}
			switch result.DamageSkill {
			case game.SkillCutthroat:
				s.GetWorld().DoCutthroatDamage(ch, damageTarget, result.Damage)
			case game.SkillSmackheads:
				s.GetWorld().DoSmackheadsDamage(ch, damageTarget, result.Damage)
			default:
				s.GetWorld().DoSpellDamage(ch, damageTarget, result.Damage, result.DamageSkill)
			}
		}
	}

	// Bite emits its literal act() messages before damage(), then the C damage
	// call emits skill_message() after applying the damage and setting combat.
	if result.SkillMsgAfterDamage && !result.InitialAttack && !result.SkillMsgInDamage {
		sendSkillMessage()
	}

	// Initiate engine combat whenever the skill signals it and the target
	// SURVIVED. C's damage() calls set_fighting (mutual, adds both to
	// combat_list) unconditionally on every hit — miss, zero-damage hit, and
	// positive-damage hit alike. The old `result.Damage <= 0` gate only
	// enrolled misses and zero-damage hits (L1 kick): positive-damage hits
	// (trip, headbutt) went through DoSpellDamage, which sets the victim's
	// FIGHTING field but enrolls NEITHER combatant in the engine's
	// combatOrder, so no combat rounds ever fired (DP-1213, R1/R3b).
	// Order matters and matches C: skill_message dice → damage →
	// set_fighting → improve_skill (the DeferredImprove loop below).
	// DoSpellDamage runs the death pipeline at POS_DEAD; C does not enroll a
	// corpse, so skip enrollment when the hit killed the target.
	// engine.StartCombat consumes ZERO dprng draws (pure combat_list
	// manipulation, like C's set_fighting) — inserting it between the
	// message dice and the improvement does not perturb the stream (R3a).
	if !result.InitialAttack && result.StartCombat && target != nil && target.GetPosition() != combat.PosDead {
		if engine, ok := s.GetCombatEngine().(rescueCombatEngine); ok && engine != nil {
			// DoCutthroatDamage uses the complete C damage() seam, which sets
			// FIGHTING before returning but does not own the command engine's
			// combat-pair enrollment. The C command's damage() call does both.
			if err := engine.StartCombat(ch, target); err != nil && ch.GetFighting() != target.GetName() {
				slog.Error("skill combat start failed", "attacker", ch.GetName(), "target", target.GetName(), "error", err)
			}
		}
	}
	if !result.InitialAttack && result.StartCombat && len(targets) > 1 {
		if engine, ok := s.GetCombatEngine().(rescueCombatEngine); ok && engine != nil {
			for _, secondary := range targets[1:] {
				if secondary == nil || secondary.GetPosition() == combat.PosDead {
					continue
				}
				if err := engine.StartCombat(secondary, ch); err != nil {
					slog.Error("secondary skill combat start failed", "attacker", secondary.GetName(), "target", ch.GetName(), "error", err)
				}
			}
		}
	}

	// C hit(vict, ch, TYPE_UNDEFINED): the MOB_AWARE guard swings back at once.
	// Enroll target->ch and run one synchronous swing from the target, emitted
	// after the notice lines above — exactly like C's aware-backstab branch.
	if result.RetaliateHit && !result.RetaliateHitBeforeSkillMessage && !result.RetaliateHitAfterMessages {
		performRetaliateHit()
	}

	// C's serpent-kick training branch creates the hunting mob after damage()
	// returns and before improve_skill() (new_cmds2.c:734-740). The C expression
	// evaluates number(0,80) before the level check, so every successful hit
	// consumes this draw even below level 19. Keep creation quiet: create_mobile
	// followed by char_to_room does not announce a normal world spawn.
	if result.SpawnMobVNum != 0 {
		// #nosec G404 — game RNG, not cryptographic
		if dprng.Number(0, 80) == 0 && ch.GetLevel() > 18 {
			mob, err := s.GetWorld().SpawnMobQuiet(result.SpawnMobVNum, result.SpawnMobRoom)
			if err != nil {
				slog.Error("skill training mob spawn failed", "vnum", result.SpawnMobVNum, "room", result.SpawnMobRoom, "error", err)
			} else {
				mob.ConfigureCreatedMobile(result.SpawnMobLevel)
				if result.SpawnMobHunting {
					s.GetWorld().SetHunting(mob.GetName(), ch.GetName(), true)
				}
			}
		}
	}

	// Run deferred skill improvement AFTER the skill_message dice and the
	// damage/enrollment step, matching C's order: damage()/hit() draws the
	// skill_message dice first, then improve_skill() runs back in the command
	// handler (act.offensive.c do_kick/do_backstab, new_cmds.c do_trip/
	// do_headbutt). Uses the real improveSkill — it draws number(1,200) [+
	// number(1,3)] and may print "Your skill in X improves." (R1/R3b/R4 —
	// no stubbing or dummy draws). DoSpellDamage draws no RNG for these
	// skills (fixed damage formula; ApplyDamageModifiers is draw-free), so
	// message-dice → improve-draw is the exact C sequence. DP-1212.
	if !result.DeferredImproveAfterRoom && !result.DeferredImproveAfterActor {
		for _, skill := range result.DeferredImprove {
			game.ImproveSkill(ch, skill)
		}
	}

	// Apply position changes
	if result.SelfStumble {
		ch.SetPosition(combat.PosSitting)
	}
	if result.TargetFalls && target != nil {
		target.SetPosition(combat.PosSitting)
	}
	if result.SleepTarget && target != nil {
		target.SetPosition(combat.PosSleeping)
		if p, ok := target.(*game.Player); ok {
			p.SetAffect(game.AffSleep, true)
		}
	}

	// Send to victim
	if !result.SkillMsgAfterDamage && !result.RetaliateHitAfterMessages && result.MessageToVict != "" && target != nil {
		if p, ok := target.(*game.Player); ok {
			p.SendMessage(game.CapitalizeSentence(result.MessageToVict) + "\r\n")
		}
	}

	// Send to room (excluding ch and target)
	if !result.SkillMsgAfterDamage && !result.RetaliateHitAfterMessages && (result.MessageToRoom != "" || result.MessageToRoomSecond != "") {
		roomVNum := ch.GetRoom()
		world := s.GetWorld()
		players := world.GetPlayersInRoom(roomVNum)
		sendRoom := func(message string) {
			if message == "" {
				return
			}
			for _, p := range players {
				if p.Name == ch.Name && !result.RoomIncludesActor {
					continue
				}
				if target != nil && !result.RoomIncludesTarget && p.Name == target.GetName() {
					continue
				}
				p.SendMessage(game.CapitalizeSentence(message) + "\r\n")
			}
		}
		sendRoom(result.MessageToRoom)
		sendRoom(result.MessageToRoomSecond)
	}

	if result.SpawnPuke {
		s.GetWorld().MaybeSpawnPuke(ch.GetRoom())
	}
	if result.DeferredImproveAfterRoom {
		for _, skill := range result.DeferredImprove {
			game.ImproveSkill(ch, skill)
		}
	}
	if result.MessageToChAfterRoom && result.MessageToCh != "" {
		_ = s.SendMessage(game.CapitalizeSentence(result.MessageToCh) + "\r\n")
	}
	if result.DeferredImproveAfterActor {
		for _, skill := range result.DeferredImprove {
			game.ImproveSkill(ch, skill)
		}
	}
	if result.SelfStunnedAfterMessage {
		ch.SetPosition(combat.PosStunned)
	}

	if result.RetaliateHit && result.RetaliateHitAfterMessages {
		performRetaliateHit()
	}

	// do_spike/do_stake emit their authored acts first, then update the PK/death
	// counters and call raw_kill (new_cmds.c:1155-1175). Keep that tail after
	// all three audiences so NPC death-cry bytes follow the success act exactly.
	if result.RawKill && target != nil {
		if victim, ok := target.(*game.Player); ok {
			ch.PKs++
			victim.Deaths++
		}
		s.GetWorld().RawKillCombatant(target, combat.TYPE_UNDEFINED)
	}

	// Apply WAIT_STATE (C-10: cooldown in PULSE_VIOLENCE ticks)
	if result.WaitChPulses > 0 {
		ch.SetWaitStatePulses(result.WaitChPulses)
	} else if result.WaitCh > 0 {
		ch.SetWaitState(result.WaitCh)
	}
	if result.WaitTarget > 0 {
		for _, waitTarget := range targets {
			switch t := waitTarget.(type) {
			case *game.Player:
				t.SetWaitState(result.WaitTarget)
			case *game.MobInstance:
				t.SetWaitState(result.WaitTarget)
			}
		}
	}

	return nil
}

// CmdMold handles the mold command — rename/redescribe clay items.
func CmdMold(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}
	ch := s.GetPlayer()

	// C do_mold parses the object with one_argument, the new name with
	// one_word (including quoted names), then treats the entire remainder as
	// the description before it checks object/name/description gates.
	argument := strings.Join(args, " ")
	objName, argument := game.OneArgument(argument)
	newName, newDesc := game.OneWord(argument)

	result := game.DoMold(ch, objName, newName, newDesc)
	return sendSkillResult(s, ch, nil, result)
}

// CmdBehead handles the behead command.
func CmdBehead(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}
	ch := s.GetPlayer()

	if ch.GetPosition() == combat.PosFighting {
		return s.SendMessage("You're a little busy for that!\r\n")
	}

	if len(args) == 0 {
		return s.SendMessage("Behead who?\r\n")
	}

	targetName, _ := game.OneArgument(strings.Join(args, " "))
	world := s.GetWorld()
	result := game.DoBehead(ch, targetName, world)
	return sendSkillResult(s, ch, nil, result)
}

// CmdBearhug handles the bearhug command.
func CmdBearhug(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}
	ch := s.GetPlayer()

	canUse, msg := game.CanUseSkill(ch, game.SkillBearhug)
	if !canUse {
		return s.SendMessage(msg)
	}

	// C do_bearhug checks IS_MOUNTED before resolving its argument
	// (new_cmds.c:487-491).
	if ch.IsMounted() {
		return s.SendMessage("Dismount first!\r\n")
	}

	var target combat.Combatant
	var found bool
	world := s.GetWorld()

	// C uses one_argument and falls back to FIGHTING(ch) whenever the named
	// lookup fails, even if an argument was supplied (new_cmds.c:477,493-501).
	targetName, _ := game.OneArgument(strings.Join(args, " "))
	if targetName != "" {
		target, _, found = game.FindTargetInRoom(world, ch.GetRoomVNum(), targetName, ch)
	}
	if !found && ch.GetFighting() != "" {
		target, _, found = game.FindTargetInRoom(world, ch.GetRoomVNum(), ch.GetFighting(), ch)
	}
	if !found {
		return s.SendMessage("Bear hug who?\r\n")
	}

	result := game.DoBearhug(ch, target, world)
	return sendSkillResult(s, ch, target, result)
}

// CmdSlug handles the slug command.
func CmdSlug(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}
	ch := s.GetPlayer()

	canUse, msg := game.CanUseSkill(ch, game.SkillSlug)
	if !canUse {
		return s.SendMessage(msg)
	}

	var target combat.Combatant
	var found bool
	world := s.GetWorld()

	// C runs one_argument before lookup, then falls back to FIGHTING(ch) when
	// the parsed name is absent or misses, even when an argument was supplied
	// (new_cmds.c:826, 833-840). The stored Go fighting name needs the exact
	// pointer-style helper because a mob's short description is not its keyword
	// list.
	targetName, _ := game.OneArgument(strings.Join(args, " "))
	target, _, found = game.FindTargetInRoom(world, ch.GetRoomVNum(), targetName, ch)
	if !found && ch.GetFighting() != "" {
		target, found = game.FindFightingTargetInRoom(world, ch.GetRoomVNum(), ch.GetFighting(), ch)
	}
	if !found {
		return s.SendMessage("Slug who?\r\n")
	}

	result := game.DoSlug(ch, target)
	return sendSkillResult(s, ch, target, result)
}

// CmdSmackheads handles the smackheads command.
func CmdSmackheads(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}
	ch := s.GetPlayer()

	canUse, msg := game.CanUseSkill(ch, game.SkillSmackheads)
	if !canUse {
		return s.SendMessage(msg)
	}

	victim1Name := ""
	victim2Name := ""
	if len(args) > 0 {
		victim1Name = args[0]
	}
	if len(args) > 1 {
		// C half_chop passes the entire remainder to the second lookup.
		victim2Name = strings.Join(args[1:], " ")
	}
	world := s.GetWorld()
	result := game.DoSmackheads(ch, victim1Name, victim2Name, world)
	return sendSkillResult(s, ch, nil, result)
}

// CmdBite handles the bite command.
func CmdBite(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}
	ch := s.GetPlayer()

	var target combat.Combatant
	var found bool
	world := s.GetWorld()

	// C do_bite (new_cmds.c): a peaceful room is a silent no-op — the command
	// returns without sending anything.
	if world != nil && world.RoomHasFlag(ch.GetRoomVNum(), "peaceful") {
		return nil
	}

	if len(args) == 0 {
		fighting := ch.GetFighting()
		if fighting == "" {
			return s.SendMessage("Bite who?!\r\n")
		}
		// C assigns FIGHTING(ch) and returns immediately for an empty argument;
		// it neither validates the target's visibility nor emits a byte.
		return nil
	} else {
		targetName, _ := game.OneArgument(strings.Join(args, " "))
		target, _, found = game.FindTargetInRoom(world, ch.GetRoomVNum(), targetName, ch)
		if !found {
			return s.SendMessage("Bite who?!\r\n")
		}
	}

	result := game.DoBite(ch, target)
	return sendSkillResult(s, ch, target, result)
}

// CmdTag handles the tag command.
func CmdTag(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}
	ch := s.GetPlayer()

	if len(args) == 0 {
		return s.SendMessage("Tag who?\r\n")
	}

	targetName := strings.Join(args, " ")
	world := s.GetWorld()
	result := game.DoTag(ch, targetName, world)
	return sendSkillResult(s, ch, nil, result)
}

// CmdPoint handles the point command.
func CmdPoint(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}
	ch := s.GetPlayer()

	argument := strings.Join(args, " ")
	targetName, _ := game.OneArgument(argument)
	world := s.GetWorld()
	result := game.DoPoint(ch, targetName, world)

	var target combat.Combatant
	if targetName != "" {
		if resolved, found := world.ResolveCharInRoom(ch, targetName); found {
			target = resolved.Combatant
		}
	}
	return sendSkillResult(s, ch, target, result)
}

// CmdGroinrip handles the groinrip command.
func CmdGroinrip(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}
	ch := s.GetPlayer()

	// C do_groinrip (new_cmds.c): the peaceful-room rejection runs BEFORE the
	// skill gate — anyone in a peaceful room is stopped here regardless of skill.
	if w := s.GetWorld(); w != nil && w.RoomHasFlag(ch.GetRoomVNum(), "peaceful") {
		return s.SendMessage("You cannot commit acts of violence here!\r\n")
	}

	canUse, msg := game.CanUseSkill(ch, game.SkillGroinrip)
	if !canUse {
		return s.SendMessage(msg)
	}
	if ch.IsMounted() {
		return s.SendMessage("Dismount first!\r\n")
	}

	var target combat.Combatant
	var found bool
	world := s.GetWorld()

	if len(args) == 0 {
		fighting := ch.GetFighting()
		if fighting == "" {
			return s.SendMessage("Groinrip who?\r\n")
		}
		// C falls back to the FIGHTING(ch) pointer when no argument is
		// supplied. Go stores that state by display name, so resolve it
		// through the pointer-preserving helper rather than get_char_room_vis
		// keyword matching (R1/R5e).
		resolved, ok := world.ResolveFightingTarget(ch)
		if ok {
			target = resolved.Combatant
			found = true
		}
		if !found {
			return s.SendMessage("They don't seem to be here.\r\n")
		}
	} else {
		// C one_argument() consumes only the first word and discards the rest.
		targetName := args[0]
		target, _, found = game.FindTargetInRoom(world, ch.GetRoomVNum(), targetName, ch)
		if !found {
			return s.SendMessage("Groinrip who?\r\n")
		}
	}

	result := game.DoGroinrip(ch, target, world)
	return sendSkillResult(s, ch, target, result)
}

// CmdReview handles the review command — show recent gossip history.
func CmdReview(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}
	ch := s.GetPlayer()
	world := s.GetWorld()
	if world == nil {
		return fmt.Errorf("world not available")
	}

	result := game.DoReview(ch, world)
	return sendSkillResult(s, ch, nil, result)
}

// CmdWhois handles the whois command — check player info.
func CmdWhois(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}
	ch := s.GetPlayer()

	if len(args) == 0 {
		return s.SendMessage("For whom do you wish to search?\r\n")
	}

	targetName := strings.Join(args, " ")
	result := game.DoWhois(ch, targetName)
	return sendSkillResult(s, ch, nil, result)
}

// CmdPalm handles the palm command — hide a small item up your sleeve.
func CmdPalm(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}
	ch := s.GetPlayer()

	if len(args) == 0 {
		return s.SendMessage("Palm what?\r\n")
	}

	objName := strings.Join(args, " ")
	world := s.GetWorld()
	result := game.DoPalm(ch, objName, world)
	return sendSkillResult(s, ch, nil, result)
}

// CmdFleshAlter handles the flesh_alter command — transform your hand into a weapon.
func CmdFleshAlter(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}
	ch := s.GetPlayer()

	canUse, msg := game.CanUseSkill(ch, game.SkillFleshAlter)
	if !canUse {
		return s.SendMessage(msg)
	}

	result := game.DoFleshAlter(ch)
	return sendSkillResult(s, ch, nil, result)
}

// CmdSpike handles the spike command (werewolf destruction).
func CmdSpike(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}
	if len(args) == 0 {
		return s.SendMessage("Whom do you wish to spike?\r\n")
	}

	ch := s.GetPlayer()
	targetName := strings.Join(args, " ")
	world := s.GetWorld()
	target, _, found := game.FindTargetInRoom(world, ch.GetRoom(), targetName, ch)
	if !found {
		return s.SendMessage("No-one by that name here.\r\n")
	}

	result := game.DoSpike(ch, target, 0, world)
	return sendSkillResult(s, ch, target, result)
}

// CmdStake handles the stake command (vampire destruction).
func CmdStake(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}
	if len(args) == 0 {
		return s.SendMessage("Whom do you wish to stake?\r\n")
	}

	ch := s.GetPlayer()
	targetName := strings.Join(args, " ")
	world := s.GetWorld()
	target, _, found := game.FindTargetInRoom(world, ch.GetRoom(), targetName, ch)
	if !found {
		return s.SendMessage("No-one by that name here.\r\n")
	}

	result := game.DoSpike(ch, target, 1, world)
	return sendSkillResult(s, ch, target, result)
}

// CmdCircle handles the circle command.
func CmdCircle(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}

	ch := s.GetPlayer()
	// C do_circle (new_cmds.c): NO skill gate — goes straight to the target
	// lookup. A no-target / no-fight caller gets "Circle who?" regardless of
	// skill. The former CanUseSkill gate was invented (R4).
	world := s.GetWorld()
	var target combat.Combatant
	var found bool
	if len(args) > 0 {
		// C do_circle uses one_argument: skip fill words, lowercase the first
		// target token, and ignore the remainder (new_cmds.c:2396).
		targetName, _ := game.OneArgument(strings.Join(args, " "))
		target, _, found = game.FindTargetInRoom(world, ch.GetRoom(), targetName, ch)
		if !found {
			return s.SendMessage("Circle who?\r\n")
		}
	} else if ch.GetFighting() != "" {
		// Default to current fighting target.
		target, _, found = game.FindTargetInRoom(world, ch.GetRoom(), ch.GetFighting(), ch)
		if !found {
			return s.SendMessage("Circle who?\r\n")
		}
	} else {
		return s.SendMessage("Circle who?\r\n")
	}

	if target.GetName() == ch.Name {
		return s.SendMessage("How can you stab yourself in the back?\r\n")
	}

	result := game.DoCircle(ch, target)
	return sendSkillResult(s, ch, target, result)
}

// CmdCharge handles the charge command.
func CmdCharge(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}

	ch := s.GetPlayer()
	canUse, msg := game.CanUseSkill(ch, game.SkillCharge)
	if !canUse {
		return s.SendMessage(msg)
	}

	world := s.GetWorld()
	var target combat.Combatant
	var found bool
	if len(args) > 0 {
		// C do_charge uses one_argument: skip fill words, lowercase the first
		// target token, and ignore the remainder (new_cmds.c:887).
		targetName, _ := game.OneArgument(strings.Join(args, " "))
		target, _, found = game.FindTargetInRoom(world, ch.GetRoom(), targetName, ch)
		if !found {
			return s.SendMessage("Great! Fine! Charge who?!?!\r\n")
		}
	} else if ch.GetFighting() != "" {
		target, _, found = game.FindTargetInRoom(world, ch.GetRoom(), ch.GetFighting(), ch)
		if !found {
			return s.SendMessage("Great! Fine! Charge who?!?!\r\n")
		}
	} else {
		return s.SendMessage("Great! Fine! Charge who?!?!\r\n")
	}

	if target.GetName() == ch.Name {
		return s.SendMessage("You charge headlong into the ground, impressing everyone..\r\n")
	}

	result := game.DoCharge(ch, target)
	return sendSkillResult(s, ch, target, result)
}

// RegisterSkillCommands registers all skill-related commands.
//
// NOTE (M-04): This is currently a no-op placeholder. Skill commands are
// registered via init() functions in pkg/command/ files and wired through
// the session layer. This function should be the explicit entry point for
// all skill command registration once the init()-based pattern is migrated.
// See pkg/command/registry.go for the migration plan.
func RegisterSkillCommands() {
	// Registration placeholder — commands are called directly via Cmd* handlers.
}
