package command

import (
	"fmt"
	"sort"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/game"
)

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

		output.WriteString(fmt.Sprintf("║ %-22s ║ %4d ║ %3d%%   ║ %-9s ║\r\n",
			displayName, skill.Level, progress, typeStr))
	}

	output.WriteString("╚══════════════════════════╩══════╩════════╩═══════════╝\r\n")

	// Add skill points and slots info
	points := skillManager.GetSkillPoints()
	usedSlots := skillManager.GetUsedSlots()
	totalSlots := skillManager.GetSlots()
	availableSlots := skillManager.GetAvailableSlots()

	output.WriteString(fmt.Sprintf("\r\nSkill points: %d | Slots: %d/%d (%d available)\r\n",
		points, usedSlots, totalSlots, availableSlots))

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
			output.WriteString(fmt.Sprintf("║ %-58s ║\r\n", title))
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

				output.WriteString(fmt.Sprintf("║ %-26s ║ %6d ║ %-16s ║\r\n",
					displayName, skill.Difficulty, status))
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

	output.WriteString(fmt.Sprintf("\r\nSkill points: %d | Available slots: %d\r\n",
		points, availableSlots))

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

// #nosec G104
	s.SendMessage(output)

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
	output.WriteString(fmt.Sprintf("You attempt to use %s", skill.DisplayName))

	if targetName != "" {
		output.WriteString(fmt.Sprintf(" on %s", targetName))
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
	output.WriteString(fmt.Sprintf("║ %-50s ║\r\n", skill.DisplayName))
	output.WriteString("╠══════════════════════════════════════════════════════╣\r\n")

	// Skill type
	typeStr := "Utility"
	switch skill.Type {
	case engine.SkillTypeCombat:
		typeStr = "Combat"
	case engine.SkillTypeMagic:
		typeStr = "Magic"
	}
	output.WriteString(fmt.Sprintf("║ Type: %-44s ║\r\n", typeStr))

	// Difficulty
	output.WriteString(fmt.Sprintf("║ Difficulty: %-40d ║\r\n", skill.Difficulty))

	// Status
	if skill.Learned {
		output.WriteString(fmt.Sprintf("║ Status: Learned (Level %d) %30s ║\r\n", skill.Level, ""))
		output.WriteString(fmt.Sprintf("║ Proficiency: %-38s ║\r\n", skill.GetDisplayLevel()))
		output.WriteString(fmt.Sprintf("║ Practice Progress: %d%% %32s ║\r\n", skill.GetProgress(), ""))
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
			output.WriteString(fmt.Sprintf("║ Needed: Level %d, Stat %d %30s ║\r\n",
				skill.Difficulty, 10, ""))
		}
	}

	output.WriteString("╚══════════════════════════════════════════════════════╝\r\n")

	if skill.Learned {
		output.WriteString(fmt.Sprintf("\r\nUse 'practice %s' to improve this skill.\r\n", skill.Name))
	} else {
		output.WriteString(fmt.Sprintf("\r\nUse 'learn %s' to learn this skill.\r\n", skill.Name))
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
	if len(args) == 0 {
		return s.SendMessage("Backstab who?\r\n")
	}

	ch := s.GetPlayer()
	canUse, msg := game.CanUseSkill(ch, game.SkillBackstab)
	if !canUse {
		return s.SendMessage(msg + "\r\n")
	}

	// Find target in room
	targetName := strings.Join(args, " ")
	world := s.GetWorld()
	target, _, found := game.FindTargetInRoom(world, ch.GetRoom(), targetName, ch)
	if !found {
		return s.SendMessage("They don't seem to be here.\r\n")
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
		return s.SendMessage(msg + "\r\n")
	}

	// Find target — if in combat, default to fighting target
	var target combat.Combatant
	var found bool
	world := s.GetWorld()
	if ch.GetFighting() != "" && len(args) == 0 {
		return s.SendMessage("Bash who?\r\n")
	} else if len(args) > 0 {
		targetName := strings.Join(args, " ")
		target, _, found = game.FindTargetInRoom(world, ch.GetRoom(), targetName, ch)
		if !found {
			return s.SendMessage("Bash who?\r\n")
		}
	} else {
		return s.SendMessage("Bash who?\r\n")
	}

	if target.GetName() == ch.Name {
		return s.SendMessage("Aren't we funny today...\r\n")
	}

	result := game.DoBash(ch, target)
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
		return s.SendMessage(msg + "\r\n")
	}

	var target combat.Combatant
	var found bool
	world := s.GetWorld()
	if ch.GetFighting() != "" && len(args) == 0 {
		return s.SendMessage("Kick who?\r\n")
	} else if len(args) > 0 {
		targetName := strings.Join(args, " ")
		target, _, found = game.FindTargetInRoom(world, ch.GetRoom(), targetName, ch)
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
		return s.SendMessage(msg + "\r\n")
	}

	var target combat.Combatant
	var found bool
	world := s.GetWorld()
	if ch.GetFighting() != "" && len(args) == 0 {
		return s.SendMessage("Trip who?\r\n")
	} else if len(args) > 0 {
		targetName := strings.Join(args, " ")
		target, _, found = game.FindTargetInRoom(world, ch.GetRoom(), targetName, ch)
		if !found {
			return s.SendMessage("Trip who?\r\n")
		}
	} else {
		return s.SendMessage("Trip who?\r\n")
	}

	if target.GetName() == ch.Name {
		return s.SendMessage("You trip over your shoe laces...\r\n")
	}

	result := game.DoTrip(ch, target)
	return sendSkillResult(s, ch, target, result)
}

// CmdHeadbutt handles the headbutt command.
func CmdHeadbutt(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}

	ch := s.GetPlayer()
	canUse, msg := game.CanUseSkill(ch, game.SkillHeadbutt)
	if !canUse {
		return s.SendMessage(msg + "\r\n")
	}

	var target combat.Combatant
	var found bool
	world := s.GetWorld()
	if ch.GetFighting() != "" && len(args) == 0 {
		return s.SendMessage("Headbutt who?\r\n")
	} else if len(args) > 0 {
		targetName := strings.Join(args, " ")
		target, _, found = game.FindTargetInRoom(world, ch.GetRoom(), targetName, ch)
		if !found {
			return s.SendMessage("Headbutt who?\r\n")
		}
	} else {
		return s.SendMessage("Headbutt who?\r\n")
	}

	if target.GetName() == ch.Name {
		return s.SendMessage("You contemplate headbutting yourself... maybe later.\r\n")
	}

	result := game.DoHeadbutt(ch, target)
	return sendSkillResult(s, ch, target, result)
}

// CmdRescue handles the rescue command.
func CmdRescue(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}
	if len(args) == 0 {
		return s.SendMessage("Whom do you want to rescue?\r\n")
	}

	ch := s.GetPlayer()
	canUse, msg := game.CanUseSkill(ch, game.SkillRescue)
	if !canUse {
		return s.SendMessage(msg + "\r\n")
	}

	targetName := strings.Join(args, " ")
	world := s.GetWorld()
	target, _, found := game.FindTargetInRoom(world, ch.GetRoom(), targetName, ch)
	if !found {
		return s.SendMessage("They don't seem to be here.\r\n")
	}

	if target.GetName() == ch.Name {
		return s.SendMessage("What about fleeing instead?\r\n")
	}

	// Execute the rescue
	result := game.DoRescue(ch, target, world, s.GetCombatEngine().(interface {
		StartCombat(combat.Combatant, combat.Combatant) error
		StopCombat(string)
	}))
	return sendSkillResult(s, ch, target, result)
}

// CmdSneak handles the sneak command.
func CmdSneak(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}

	ch := s.GetPlayer()
	canUse, msg := game.CanUseSkill(ch, game.SkillSneak)
	if !canUse {
		return s.SendMessage(msg + "\r\n")
	}

	result := game.DoSneak(ch)
	return s.SendMessage(result.MessageToCh + "\r\n")
}

// CmdHide handles the hide command.
func CmdHide(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}

	ch := s.GetPlayer()
	canUse, msg := game.CanUseSkill(ch, game.SkillHide)
	if !canUse {
		return s.SendMessage(msg + "\r\n")
	}

	result := game.DoHide(ch)
	return s.SendMessage(result.MessageToCh + "\r\n")
}

// CmdSteal handles the steal command.
func CmdSteal(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}
	if len(args) < 2 {
		return s.SendMessage("Steal what from who?\r\n")
	}

	ch := s.GetPlayer()
	canUse, msg := game.CanUseSkill(ch, game.SkillSteal)
	if !canUse {
		return s.SendMessage(msg + "\r\n")
	}

	// Parse: "steal <item> <target>" or "steal coins <target>"
	itemName := args[0]
	targetName := strings.Join(args[1:], " ")
	world := s.GetWorld()

	target, _, found := game.FindTargetInRoom(world, ch.GetRoom(), targetName, ch)
	if !found {
		return s.SendMessage("Steal what from who?\r\n")
	}

	result := game.DoSteal(ch, target, itemName)
	return sendSkillResult(s, ch, target, result)
}

// CmdPickLock handles the pick command.
func CmdPickLock(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}

	ch := s.GetPlayer()
	canUse, msg := game.CanUseSkill(ch, game.SkillPickLock)
	if !canUse {
		return s.SendMessage(msg + "\r\n")
	}

	result := game.DoPickLock(ch)
	return s.SendMessage(result.MessageToCh + "\r\n")
}

// CmdCarve handles the carve command.
func CmdCarve(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}
	if len(args) == 0 {
		return s.SendMessage("You want to carve what?!?!\r\n")
	}

	ch := s.GetPlayer()
	if ch.GetPosition() == combat.PosFighting {
		return s.SendMessage("How can you think of food at a time like this?!?!\r\n")
	}

	canUse, msg := game.CanUseSkill(ch, game.SkillCarve)
	if !canUse {
		return s.SendMessage(msg + "\r\n")
	}

	targetName := strings.ToLower(strings.Join(args, " "))
	world := s.GetWorld()

	// Check if target is a character
	target, _, found := game.FindTargetInRoom(world, ch.GetRoom(), targetName, ch)
	if found {
		if target.GetName() == ch.Name {
			return s.SendMessage("This game doesn't support self-mutilation!\r\n")
		}
		return s.SendMessage("You kill it first and THEN you can eat it!\r\n")
	}

	result := game.DoCarve(ch, targetName, world)
	return sendSkillResult(s, ch, nil, result)
}

// CmdCutthroat handles the cutthroat command.
func CmdCutthroat(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}
	if len(args) == 0 {
		return s.SendMessage("Cut what throat where?\n\r")
	}

	ch := s.GetPlayer()
	canUse, msg := game.CanUseSkill(ch, game.SkillCutthroat)
	if !canUse {
		return s.SendMessage(msg + "\r\n")
	}

	// Find target
	targetName := strings.Join(args, " ")
	world := s.GetWorld()
	target, _, found := game.FindTargetInRoom(world, ch.GetRoom(), targetName, ch)
	if !found {
		return s.SendMessage("Cut what throat where?\n\r")
	}

	if target.GetName() == ch.Name {
		return s.SendMessage("That would be bad.\n\r")
	}

	result := game.DoCutthroat(ch, target)
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
		return s.SendMessage(msg + "\r\n")
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

// CmdCompare handles the compare command.
func CmdCompare(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}

	ch := s.GetPlayer()
	if ch.GetPosition() == combat.PosFighting {
		return s.SendMessage("You're pretty busy right now!\n\r")
	}

	canUse, msg := game.CanUseSkill(ch, game.SkillCompare)
	if !canUse {
		return s.SendMessage(msg + "\r\n")
	}

	if len(args) == 0 {
		return s.SendMessage("Compare what and what?\r\n")
	}

	var objName1, objName2 string
	var compareToEquipped bool

	if len(args) == 1 {
		// Compare with equipped item
		objName1 = args[0]
		compareToEquipped = true
	} else {
		objName1 = args[0]
		objName2 = strings.Join(args[1:], " ")
	}

	result := game.DoCompare(ch, objName1, objName2, compareToEquipped)
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
		return s.SendMessage(msg + "\r\n")
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
	if len(args) == 0 {
		return s.SendMessage("Sharpen what?\r\n")
	}

	ch := s.GetPlayer()
	if ch.GetPosition() == combat.PosFighting {
		return s.SendMessage("You're too busy to be sharpening anything!\n\r")
	}

	canUse, msg := game.CanUseSkill(ch, game.SkillSharpen)
	if !canUse {
		return s.SendMessage(msg + "\r\n")
	}

	objName := strings.Join(args, " ")
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
	if len(args) == 0 {
		return s.SendMessage("Aid who?\r\n")
	}

	ch := s.GetPlayer()
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

	// Determine target: either specified or current fighting target
	var target combat.Combatant
	var found bool
	world := s.GetWorld()

	if len(args) == 0 {
		fighting := ch.GetFighting()
		if fighting == "" {
			return s.SendMessage("Disarm who?\r\n")
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
	targetName := strings.Join(args, " ")
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

	var target combat.Combatant
	var found bool
	world := s.GetWorld()

	if len(args) == 0 {
		// Try to kick whoever we're fighting
		fighting := ch.GetFighting()
		if fighting == "" {
			return s.SendMessage("Kick who?\r\n")
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

	result := game.DoSerpentKick(ch, target)
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
func genderPronoun(sex int) string {
	switch sex {
	case 1:
		return "himself"
	case 0:
		return "herself"
	default:
		return "itself"
	}
}

// ---------------------------------------------------------------------------
// Helper: send skill result to player, victim, and room
// ---------------------------------------------------------------------------

func sendSkillResult(s SessionInterface, ch *game.Player, target combat.Combatant, result game.SkillResult) error {
	// Send to character
	if result.MessageToCh != "" {
// #nosec G104
		s.SendMessage(result.MessageToCh + "\r\n")
	}

	// Apply damage
	if result.Damage > 0 && target != nil {
		target.TakeDamage(result.Damage)
		if target.GetHP() <= 0 {
// #nosec G104
			s.SendMessage(fmt.Sprintf("%s is dead!\r\n", target.GetName()))
		}
	}

	// Apply position changes
	if result.SelfStumble {
		ch.SetPosition(combat.PosSitting)
// #nosec G104
		s.SendMessage("You fall to the ground!\r\n")
	}
	if result.TargetFalls && target != nil {
		if p, ok := target.(*game.Player); ok {
			p.SetPosition(combat.PosSitting)
		}
		// Mobs don't have SetPosition in current interface — would need Combatant extension
	}

	// Send to victim
	if result.MessageToVict != "" && target != nil {
		if p, ok := target.(*game.Player); ok {
			p.SendMessage(result.MessageToVict + "\r\n")
		}
	}

	// Send to room (excluding ch and target)
	if result.MessageToRoom != "" {
		roomVNum := ch.GetRoom()
		world := s.GetWorld()
		players := world.GetPlayersInRoom(roomVNum)
		for _, p := range players {
			if p.Name == ch.Name {
				continue
			}
			if target != nil && p.Name == target.GetName() {
				continue
			}
			p.SendMessage(result.MessageToRoom + "\r\n")
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

	// Need 3 args: <object> <new_name> <new_description>
	if len(args) < 3 {
		return s.SendMessage("Usage: mold <object> <new name> <new description>\r\n")
	}

	objName := args[0]
	newName := args[1]
	newDesc := strings.Join(args[2:], " ")

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
		return s.SendMessage("Behead what?\r\n")
	}

	targetName := strings.Join(args, " ")
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
		return s.SendMessage(msg + "\r\n")
	}

	var target combat.Combatant
	var found bool
	world := s.GetWorld()

	if len(args) == 0 {
		fighting := ch.GetFighting()
		if fighting == "" {
			return s.SendMessage("Bear hug who?\r\n")
		}
		target, _, found = game.FindTargetInRoom(world, ch.GetRoomVNum(), fighting, ch)
		if !found {
			return s.SendMessage("They don't seem to be here.\r\n")
		}
	} else {
		targetName := strings.Join(args, " ")
		target, _, found = game.FindTargetInRoom(world, ch.GetRoomVNum(), targetName, ch)
		if !found {
			return s.SendMessage("Bear hug who?\r\n")
		}
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
		return s.SendMessage(msg + "\r\n")
	}

	var target combat.Combatant
	var found bool
	world := s.GetWorld()

	if len(args) == 0 {
		fighting := ch.GetFighting()
		if fighting == "" {
			return s.SendMessage("Slug who?\r\n")
		}
		target, _, found = game.FindTargetInRoom(world, ch.GetRoomVNum(), fighting, ch)
		if !found {
			return s.SendMessage("They don't seem to be here.\r\n")
		}
	} else {
		targetName := strings.Join(args, " ")
		target, _, found = game.FindTargetInRoom(world, ch.GetRoomVNum(), targetName, ch)
		if !found {
			return s.SendMessage("Slug who?\r\n")
		}
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
		return s.SendMessage(msg + "\r\n")
	}

	if len(args) < 2 {
		return s.SendMessage("Smack whose heads together?\r\n")
	}

	victim1Name := args[0]
	victim2Name := args[1]
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

	if len(args) == 0 {
		fighting := ch.GetFighting()
		if fighting == "" {
			return s.SendMessage("Bite who?\r\n")
		}
		target, _, found = game.FindTargetInRoom(world, ch.GetRoomVNum(), fighting, ch)
		if !found {
			return s.SendMessage("They don't seem to be here.\r\n")
		}
	} else {
		targetName := strings.Join(args, " ")
		target, _, found = game.FindTargetInRoom(world, ch.GetRoomVNum(), targetName, ch)
		if !found {
			return s.SendMessage("Bite who?\r\n")
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

	targetName := ""
	if len(args) > 0 {
		targetName = strings.Join(args, " ")
	}

	world := s.GetWorld()
	result := game.DoPoint(ch, targetName, world)
	return sendSkillResult(s, ch, nil, result)
}

// CmdGroinrip handles the groinrip command.
func CmdGroinrip(s SessionInterface, args []string) error {
	if s.GetPlayer() == nil {
		return fmt.Errorf("not logged in")
	}
	ch := s.GetPlayer()

	canUse, msg := game.CanUseSkill(ch, game.SkillGroinrip)
	if !canUse {
		return s.SendMessage(msg + "\r\n")
	}

	var target combat.Combatant
	var found bool
	world := s.GetWorld()

	if len(args) == 0 {
		fighting := ch.GetFighting()
		if fighting == "" {
			return s.SendMessage("Groinrip who?\r\n")
		}
		target, _, found = game.FindTargetInRoom(world, ch.GetRoomVNum(), fighting, ch)
		if !found {
			return s.SendMessage("They don't seem to be here.\r\n")
		}
	} else {
		targetName := strings.Join(args, " ")
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

	result := game.DoReview(ch)
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
		return s.SendMessage(msg + "\r\n")
	}

	result := game.DoFleshAlter(ch)
	return sendSkillResult(s, ch, nil, result)
}

// RegisterSkillCommands registers all skill-related commands.
func RegisterSkillCommands() {
	// Registration placeholder — commands are called directly via Cmd* handlers.
}
