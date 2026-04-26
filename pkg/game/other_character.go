package game

import (
	"fmt"
	"strings"
)

// ---------------------------------------------------------------------------
// do_practice — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doPractice(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	if strings.TrimSpace(arg) != "" {
		ch.SendMessage("You can only practice skills in your guild.\r\n")
	} else {
		// list_skills
		skillList := ch.SkillManager.GetLearnedSkills()
		if len(skillList) == 0 {
			ch.SendMessage("You have no skills to practice.\r\n")
			return true
		}
		ch.SendMessage("Skills you can practice:\r\n")
		for _, s := range skillList {
			val := ch.SkillManager.GetSkillLevel(s.Name)
			msg := fmt.Sprintf("  %-20s %3d%%\r\n", s.DisplayName, val)
			ch.SendMessage(msg)
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// do_visible — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doVisible(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	// Kai zai check (simplified: skill name "kai_zai" or "kz")
	hasKaiZai := false
	affects := ch.ActiveAffects
	for _, a := range affects {
		if strings.Contains(strings.ToLower(a.Source), "zai") {
			hasKaiZai = true
			break
		}
	}
	if hasKaiZai {
		ch.SendMessage("You cannot become visible until your zai ends!\r\n")
		return true
	}

	// Immort visibility
	if ch.Level >= LVL_IMMORT {
		ch.SendMessage("You are visible.\r\n")
		return true
	}

	altered := false
	if ch.IsAffected(affInvisible) {
		ch.SetAffect(affInvisible, false)
		ch.SendMessage("You fade into view.\r\n")
		altered = true
	}
	if ch.IsAffected(affSneak) {
		ch.SendMessage("You stop sneaking.\r\n")
		ch.SetAffect(affSneak, false)
		altered = true
	}
	if !altered {
		ch.SendMessage("You are already visible.\r\n")
	}
	return true
}

// ---------------------------------------------------------------------------
// do_title — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doTitle(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		ch.SendMessage("Your title is fine... go away.\r\n")
		return true
	}

	title := strings.TrimSpace(arg)
	// Remove double dollars
	title = strings.ReplaceAll(title, "$$", "$")

	if ch.Flags&(1<<PlrNODELETE) != 0 {
		ch.SendMessage("You can't title yourself -- you shouldn't have abused it!\r\n")
		return true
	}
	if strings.Contains(title, "(") || strings.Contains(title, ")") {
		ch.SendMessage("Titles can't contain the ( or ) characters.\r\n")
		return true
	}
	if len(title) > 55 { // MAX_TITLE_LENGTH
		ch.SendMessage(fmt.Sprintf("Sorry, titles can't be longer than %d characters.\r\n", 55))
		return true
	}

	ch.Title = title
	msg := fmt.Sprintf("Okay, you're now %s %s.\r\n", ch.Name, ch.Title)
	ch.SendMessage(msg)
	return true
}

// ---------------------------------------------------------------------------
// do_group — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doGroup(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	arg = strings.TrimSpace(arg)

	if arg == "" {
		// print_group
		if !ch.IsAffected(affGroup) {
			ch.SendMessage("But you are not the member of a group!\r\n")
			return true
		}
		ch.SendMessage("Your group consists of:\r\n")

		// Show leader
		leaderName := ch.Following
		if leaderName == "" {
			leaderName = ch.Name // self is leader
			msg := fmt.Sprintf("     [%3dH %3dM %3dV] $N (Head of group)\r\n", ch.Health, ch.Mana, ch.Move)
			ch.SendMessage(msg)
		} else {
			leader, _ := w.GetPlayer(leaderName)
			if leader != nil && leader.IsAffected(affGroup) {
				msg := fmt.Sprintf("     [%3dH %3dM %3dV] $N (Head of group)\r\n", leader.Health, leader.Mana, leader.Move)
				ch.SendMessage(msg)
			}
		}

		// Show followers in group
		players := w.GetPlayersInRoom(ch.GetRoomVNum())
		for _, p := range players {
			if p.Name == leaderName || p.Name == ch.Name {
				continue
			}
			if p.Following == leaderName || p.Following == ch.Name {
				if p.IsAffected(affGroup) {
					msg := fmt.Sprintf("     [%3dH %3dM %3dV] %s\r\n", p.Health, p.Mana, p.Move, p.Name)
					ch.SendMessage(msg)
				}
			}
		}

		// Show mob followers
		mobs := w.GetMobsInRoom(ch.GetRoomVNum())
		for _, m := range mobs {
			msg := fmt.Sprintf("     [---H ---M ---V] %s\r\n", m.GetShortDesc())
			ch.SendMessage(msg)
		}
		return true
	}

	if ch.Following != "" {
		ch.SendMessage("You can not enroll group members without being head of a group.\r\n")
		return true
	}

	if strings.EqualFold(arg, "all") {
		// Add self
		ch.SetAffect(affGroup, true)
		found := 0

		players := w.GetPlayersInRoom(ch.GetRoomVNum())
		for _, p := range players {
			if p.Name == ch.Name {
				continue
			}
			if p.Following == ch.Name && !p.IsAffected(affGroup) {
				p.SetAffect(affGroup, true)
				msg := fmt.Sprintf("%s is now a member of your group.\r\n", p.Name)
				ch.SendMessage(msg)
				p.SendMessage(fmt.Sprintf("You are now a member of %s's group.\r\n", ch.Name))
				found++
			}
		}
		if found == 0 {
			ch.SendMessage("Everyone following you here is already in your group.\r\n")
		}
		return true
	}

	// Add/kick specific player
	victimPl, _ := w.findCharInRoom(ch, ch.GetRoomVNum(), arg)
	if victimPl == nil {
		ch.SendMessage("There is no such person!\r\n")
		return true
	}

	if victimPl == ch {
		ch.SetAffect(affGroup, true)
		ch.SendMessage("You have been added to your own group.\r\n")
		return true
	}

	if victimPl.Following != ch.Name {
		ch.SendMessage(fmt.Sprintf("%s must follow you to enter your group.\r\n", victimPl.Name))
		return true
	}

	if !victimPl.IsAffected(affGroup) {
		victimPl.SetAffect(affGroup, true)
		ch.SendMessage(fmt.Sprintf("%s is now a member of your group.\r\n", victimPl.Name))
		victimPl.SendMessage(fmt.Sprintf("You are now a member of %s's group.\r\n", ch.Name))
	} else {
		// Kick
		if ch.Name != victimPl.Name {
			ch.SendMessage(fmt.Sprintf("%s is no longer a member of your group.\r\n", victimPl.Name))
		}
		victimPl.SendMessage(fmt.Sprintf("You have been kicked out of %s's group!\r\n", ch.Name))
		victimPl.SetAffect(affGroup, false)
	}
	return true
}

// ---------------------------------------------------------------------------
// do_ungroup — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doUngroup(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	arg = strings.TrimSpace(arg)

	if arg == "" {
		if ch.Following != "" || !ch.IsAffected(affGroup) {
			ch.SendMessage("But you lead no group!\r\n")
			return true
		}

		// Disband entire group
		msg := fmt.Sprintf("%s has disbanded the group.\r\n", ch.Name)

		players := w.GetPlayersInRoom(ch.GetRoomVNum())
		for _, p := range players {
			if p.Name == ch.Name {
				continue
			}
			if p.Following == ch.Name && p.IsAffected(affGroup) {
				p.SendMessage(msg)
				p.SetAffect(affGroup, false)
				if !p.IsAffected(3) { // AFF_CHARM
					p.Following = ""
				}
			}
		}

		ch.SetAffect(affGroup, false)
		ch.SendMessage("You disband the group.\r\n")
		return true
	}

	// Kick specific
	victimPl, _ := w.findCharInRoom(ch, ch.GetRoomVNum(), arg)
	if victimPl == nil {
		ch.SendMessage("There is no such person!\r\n")
		return true
	}
	if victimPl.Following != ch.Name {
		ch.SendMessage("That person is not following you!\r\n")
		return true
	}
	if !victimPl.IsAffected(affGroup) {
		ch.SendMessage("That person isn't in your group.\r\n")
		return true
	}

	victimPl.SetAffect(affGroup, false)
	ch.SendMessage(fmt.Sprintf("%s is no longer a member of your group.\r\n", victimPl.Name))
	victimPl.SendMessage(fmt.Sprintf("You have been kicked out of %s's group!\r\n", ch.Name))

	if !victimPl.IsAffected(3) { // AFF_CHARM
		victimPl.Following = ""
	}
	return true
}

// ---------------------------------------------------------------------------
// do_report — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doReport(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	ch.SendMessage("You report:\r\n")

	players := w.GetPlayersInRoom(ch.GetRoomVNum())
	for _, p := range players {
		if p.IsNPC() {
			continue
		}
		msg := fmt.Sprintf("    [%d/%d]H [%d/%d]M [%d/%d]V [%d]Kills [%d]PKs [%d]Deaths\r\n",
			ch.Health, ch.MaxHealth,
			ch.Mana, ch.MaxMana,
			ch.Move, ch.MaxMove,
			ch.Kills, ch.PKs, ch.Deaths)
		p.SendMessage(msg)
	}
	return true
}
