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

	// Need combat engine for rescue
	return s.SendMessage("Rescue is not fully implemented yet.\r\n")
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

// ---------------------------------------------------------------------------
// Helper: send skill result to player, victim, and room
// ---------------------------------------------------------------------------

func sendSkillResult(s SessionInterface, ch *game.Player, target combat.Combatant, result game.SkillResult) error {
	// Send to character
	if result.MessageToCh != "" {
		s.SendMessage(result.MessageToCh + "\r\n")
	}

	// Apply damage
	if result.Damage > 0 {
		target.TakeDamage(result.Damage)
		if target.GetHP() <= 0 {
			s.SendMessage(fmt.Sprintf("%s is dead!\r\n", target.GetName()))
		}
	}

	// Apply position changes
	if result.SelfStumble {
		ch.SetPosition(combat.PosSitting)
		s.SendMessage("You fall to the ground!\r\n")
	}
	if result.TargetFalls {
		if p, ok := target.(*game.Player); ok {
			p.SetPosition(combat.PosSitting)
		}
		// Mobs don't have SetPosition in current interface — would need Combatant extension
	}

	// Send to victim
	if result.MessageToVict != "" {
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
			if p.Name == target.GetName() {
				continue
			}
			p.SendMessage(result.MessageToRoom + "\r\n")
		}
	}

	return nil
}

// RegisterSkillCommands registers all skill-related commands
func RegisterSkillCommands() {
	// Check if session package has command registry
	// This would typically be called during server initialization
}
