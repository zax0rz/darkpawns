// Port of src/act.other.c — miscellaneous player-level commands (do_quit,
// do_save, do_sneak, do_hide, do_steal, do_practice, do_visible, do_title,
// do_group, do_ungroup, do_report, do_split, do_use, do_wimpy, do_display,
// do_gen_write, do_gen_tog, do_afk, do_auto, do_transform, do_ride,
// do_dismount, do_yank, do_peek, do_recall, do_stealth, do_appraise,
// do_inactive, do_scout, do_roll, do_not_here).
package game

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// ---------------------------------------------------------------------------
// PLR flag bit positions (from structs.h PLR_* constants)
// ---------------------------------------------------------------------------

const (
	PlrOutlaw   = 0
	PlrNODELETE = 13
	PlrCRYO     = 15
	PlrWerewolf = 16
	PlrVampire  = 17
)

// ---------------------------------------------------------------------------
// PRF flag bit positions (from structs.h, shifted to avoid PLR collision)
// ---------------------------------------------------------------------------

const (
	PrfBrief      = 20
	PrfCompact    = 21
	PrfDeaf       = 22
	PrfNotell     = 23
	PrfDisphp     = 24
	PrfDispmmana  = 25
	PrfDispmove   = 26
	PrfAutoexit   = 27
	PrfNohassle   = 28
	PrfHolyLight  = 29
	PrfNoRepeat   = 30
	PrfColor1     = 31
	PrfColor2     = 32
	PrfNowiz      = 33
	PrfLog1       = 34
	PrfLog2       = 35
	PrfNoAuctions = 36
	PrfNoGossip   = 37
	PrfNoGratz    = 38
	PrfRoomFlags  = 39
	PrfAFK        = 40
	PrfAutoLoot   = 41
	PrfAutoGold   = 42
	PrfAutoSplit  = 43
	PrfDispTank   = 44
	PrfDispTarget = 45
	PrfNoNewbie   = 46
	PrfInactive   = 47
	PrfSummonable = 48
	PrfQuest      = 49
	PrfNoCTell    = 50
	PrfNoBroad    = 51
)

// ---------------------------------------------------------------------------
// AFF flag bits used in this file (other bits defined in act_movement.go /
// act_offensive.go)
// ---------------------------------------------------------------------------

const (
	affInvisible = 2
	affWerewolf  = 32
	affVampire   = 33
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// isPlayerNPC returns true if the character is a mob (me != nil).
func isPlayerNPC(ch *Player, me *MobInstance) bool {
	return me != nil
}

// actToRoom broadcasts a formatted message to all players in the given room,
// optionally excluding one player by name.
func actToRoom(w *World, roomVNum int, format string, excludeName string) {
	players := w.GetPlayersInRoom(roomVNum)
	for _, p := range players {
		if p.IsNPC() {
			continue
		}
		if excludeName != "" && p.Name == excludeName {
			continue
		}
		p.SendMessage(format)
	}
}

// getPlayerByName finds a player by name in a slice.
func getPlayerByName(players []*Player, name string) *Player {
	for _, p := range players {
		if strings.EqualFold(p.Name, name) {
			return p
		}
	}
	return nil
}

// strCompare returns true if strings differ case-insensitively (matching C str_cmp).
func strCompare(a, b string) bool {
	return !strings.EqualFold(a, b)
}

// hasRoomFlag checks if a room has the named flag (e.g. "indoors", "death", "tunnel").
func hasRoomFlag(room *parser.Room, flag string) bool {
	for _, f := range room.Flags {
		if strings.EqualFold(f, flag) {
			return true
		}
	}
	return false
}

// isDark returns true if the room is dark (no light source, not outdoors with sun).
func isDark(room *parser.Room) bool {
	return hasRoomFlag(room, "dark")
}

// isOutdoors returns true if the room is outdoors.
func isOutdoors(room *parser.Room) bool {
	return !hasRoomFlag(room, "indoors")
}

// findCharInRoom finds a player or mob by name (case-insensitive, visible check)
// in the same room. Returns the player and mob — exactly one will be non-nil.
// Simplified: searches players first, then mobs.
// getMount finds the mount mob for a rider.
func getMount(ch *Player, w *World) *MobInstance {
	if !ch.IsAffected(affMounted) {
		return nil
	}
	mobs := w.GetMobsInRoom(ch.GetRoomVNum())
	for _, m := range mobs {
		if m.HasFlag("mount") {
			// Check if this mob is mounted with this player as rider
			// We check if the mob is also marked as mounted
			return m
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// do_quit
// ---------------------------------------------------------------------------

func (w *World) doQuit(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		ch.SendMessage("No way, you're a monster!\r\n")
		return true
	}

	roomVNum := ch.GetRoomVNum()
	room := w.GetRoomInWorld(roomVNum)

	// Check valid quit rooms
	isValidRoom := false
	validRooms := []int{8004, 8008, 18201, 21202, 21258}
	for _, v := range validRooms {
		if roomVNum == v {
			isValidRoom = true
			break
		}
	}

	if !isValidRoom {
		// Check if player owns the room (has room key)
		if room == nil {
			ch.SendMessage("You can't quit here!\r\n")
			return true
		}
	}

	if ch.Position == combat.PosFighting {
		ch.SendMessage("No way!  You are fighting!\r\n")
		return true
	}

	// Kill duplicates
	w.RemovePlayer(ch.Name)

	// Save player
	if err := SavePlayer(ch); err != nil {
		slog.Warn("failed to save player on quit", "error", err)
	}

	// Extract — broadcast leave message
	msg := fmt.Sprintf("%s has left the game.\r\n", ch.Name)
	actToRoom(w, roomVNum, msg, ch.Name)

	ch.SendMessage("Good bye... Come again soon!\r\n")

	// Signal disconnect
	ch.Send <- []byte("CLOSE_CONNECTION")

	return true
}

// ---------------------------------------------------------------------------
// do_save
// ---------------------------------------------------------------------------

func (w *World) doSave(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	if err := SavePlayer(ch); err != nil {
		ch.SendMessage("Could not save your data. Contact an admin!\r\n")
		return true
	}

	ch.SendMessage("Saving.\r\n")
	return true
}

// ---------------------------------------------------------------------------
// do_not_here
// ---------------------------------------------------------------------------

func (w *World) doNotHere(ch *Player, me *MobInstance, cmd string, arg string) bool {
	ch.SendMessage("Sorry, but you cannot do that here!\r\n")
	return true
}

// ---------------------------------------------------------------------------
// do_sneak — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doSneak(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	if ch.IsAffected(affMounted) {
		ch.SendMessage("You can't sneak around on a mount!\r\n")
		return true
	}

	skill := ch.GetSkill("sneak")
	if skill <= 0 {
		ch.SendMessage("You have no idea how to sneak!\r\n")
		return true
	}

	percent := randRange(1, 101)
	if percent > skill+ch.Stats.Dex {
		ch.SendMessage("You try to sneak but fail.\r\n")
		return true
	}

	ch.SetAffect(affSneak, true)
	ch.SendMessage("Okay, you'll try to move silently for a while.\r\n")
	return true
}

// ---------------------------------------------------------------------------
// do_hide — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doHide(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	if ch.IsAffected(affMounted) {
		ch.SendMessage("You can't hide while mounted!\r\n")
		return true
	}

	room := w.GetRoomInWorld(ch.GetRoomVNum())
	if room != nil && !isOutdoors(room) {
		ch.SendMessage("You can't hide indoors!\r\n")
		return true
	}

	if room != nil && room.Sector == SECT_CITY {
		ch.SendMessage("There's nowhere to hide here!\r\n")
		return true
	}

	skill := ch.GetSkill("hide")
	if skill <= 0 {
		ch.SendMessage("You have no idea how to hide!\r\n")
		return true
	}

	percent := randRange(1, 101)
	if percent > skill+ch.Stats.Dex {
		ch.SendMessage("You try to hide but fail.\r\n")
		return true
	}

	ch.SetAffect(affHide, true)
	ch.SendMessage("You blend into the shadows.\r\n")
	return true
}

// ---------------------------------------------------------------------------
// do_steal — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doSteal(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	// Parse arguments: "victim item" or "victim gold"
	parts := strings.SplitN(arg, " ", 2)
	if len(parts) < 1 || parts[0] == "" {
		ch.SendMessage("Steal what from whom?\r\n")
		return true
	}

	victimName := parts[0]
	objName := ""
	if len(parts) > 1 {
		objName = parts[1]
	}

	victimPl, victimMob := w.findCharInRoom(ch, ch.GetRoomVNum(), victimName)
	if victimPl == nil && victimMob == nil {
		ch.SendMessage("They aren't here.\r\n")
		return true
	}

	// Determine victim
	var victNPC bool
	if victimPl != nil {
		// Stealing from player
		if victimPl.Level >= LVL_IMMORT {
			ch.SendMessage("You cannot steal from immortals!\r\n")
			return true
		}
		victNPC = false
	} else {
		victNPC = true
	}

	if (ch.Flags&(1<<PlrOutlaw)) != 0 && !victNPC {
		ch.SendMessage("You are an outlaw!  Wait until your crime is forgotten.\r\n")
		return true
	}

	// Check level difference — no stealing from players > 10 levels below
	victLevel := 1
	if victimPl != nil {
		victLevel = victimPl.Level
	}
	if victimMob != nil {
		victLevel = 1 // mobs are always stealable level-wise
	}

	if !victNPC && victLevel > ch.Level/2 {
		ch.SendMessage("You can't steal from someone so high above you.\r\n")
		return true
	}

	ohoh := false

	if objName != "" && !strings.EqualFold(objName, "coins") && !strings.EqualFold(objName, "gold") {
		ch.SendMessage("You can only steal coins for now.\r\n")
		return true
	}

	// Steal gold
	var victGold int
	if victimPl != nil {
		victGold = victimPl.Gold
	} else {
		victGold = 0 // mobs might have gold
	}

	if victGold <= 0 {
		ch.SendMessage("You couldn't get any gold...\r\n")
		ohoh = true
	} else {
		gold := (victGold * randRange(1, 10)) / 100
		if gold > 1782 {
			gold = 1782
		}
		if gold > 0 {
			ch.Gold += gold
			if victimPl != nil {
				victimPl.Gold -= gold
			}

			if gold > 1 {
				msg := fmt.Sprintf("Bingo!  You got %d gold coins.\r\n", gold)
				ch.SendMessage(msg)
				// improve_skill
				skillVal := ch.GetSkill("steal")
				if skillVal > 0 && skillVal < 97 {
					if randRange(1, 200) <= ch.Stats.Wis+ch.Stats.Int {
						inc := randRange(1, 3)
						skillVal += inc
						if skillVal > 97 {
							skillVal = 97
						}
						ch.SetSkill("steal", skillVal)
						if inc == 3 {
							ch.SendMessage("Your skill in steal improves.\r\n")
						}
					}
				}
			} else {
				ch.SendMessage("You manage to swipe a solitary gold coin.\r\n")
			}
		} else {
			ch.SendMessage("You couldn't get any gold...\r\n")
		}
	}

	// If victim is a mob and awake, they hit back
	if ohoh && victimMob != nil && victimMob.GetPosition() > combat.PosSleeping {
		// hit(vict, ch, TYPE_UNDEFINED) — simplified: start combat
		victimMob.Attack(ch, w)
	}

	if ohoh && victimPl != nil {
		ch.Flags |= 1 << PlrOutlaw
		ch.SendMessage("You are now an outlaw!\r\n")
	}

	return true
}

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

// ---------------------------------------------------------------------------
// do_split — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doSplit(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	arg = strings.TrimSpace(arg)
	if arg == "" {
		ch.SendMessage("How many coins do you wish to split with your group?\r\n")
		return true
	}

	amount := 0
	fmt.Sscanf(arg, "%d", &amount)
	if amount <= 0 {
		ch.SendMessage("Sorry, you can't do that.\r\n")
		return true
	}
	if amount > ch.Gold {
		ch.SendMessage("You don't seem to have that much gold to split.\r\n")
		return true
	}

	leaderName := ch.Following
	if leaderName == "" {
		leaderName = ch.Name
	}

	// Count group members in same room
	num := 0
	players := w.GetPlayersInRoom(ch.GetRoomVNum())
	for _, p := range players {
		if p.IsNPC() {
			continue
		}
		if p.Following != leaderName && p.Name != leaderName {
			continue
		}
		if p.IsAffected(affGroup) {
			num++
		}
	}

	if num <= 1 || !ch.IsAffected(affGroup) {
		ch.SendMessage("With whom do you wish to share your gold?\r\n")
		return true
	}

	share := amount / num
	ch.Gold -= share * (num - 1)

	for _, p := range players {
		if p.IsNPC() {
			continue
		}
		if p.Following != leaderName && p.Name != leaderName {
			continue
		}
		if !p.IsAffected(affGroup) || p.Name == ch.Name {
			continue
		}
		p.Gold += share
		p.SendMessage(fmt.Sprintf("%s splits %d coins; you receive %d.\r\n", ch.Name, amount, share))
	}

	ch.SendMessage(fmt.Sprintf("You split %d coins among %d members -- %d coins each.\r\n", amount, num, share))
	return true
}

// ---------------------------------------------------------------------------
// do_use — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doUse(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	parts := strings.SplitN(arg, " ", 2)
	itemArg := strings.TrimSpace(parts[0])
	_ = itemArg // suppress unused
	if len(parts) > 1 {
		_ = strings.TrimSpace(parts[1]) // subArg placeholder
	}

	if itemArg == "" {
		ch.SendMessage(fmt.Sprintf("What do you want to %s?\r\n", cmd))
		return true
	}

	// Handle tattoo use
	if strings.EqualFold(itemArg, "tattoo") {
		ch.SendMessage("Tattoo functionality not yet implemented.\r\n")
		return true
	}

	// Find item via findObjNear
	item := w.findObjNear(ch, itemArg)

	if item == nil {
		ch.SendMessage(fmt.Sprintf("You don't seem to have %s %s.\r\n", "a", itemArg))
		return true
	}

	// Simplified: just use the item (item-type routing TBD)
	_ = item.Prototype.TypeFlag

	// Call mag_objectmagic (simplified)
	ch.SendMessage(fmt.Sprintf("You use %s.\r\n", itemArg))
	return true
}

// ---------------------------------------------------------------------------
// do_wimpy — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doWimpy(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	arg = strings.TrimSpace(arg)

	if arg == "" {
		if ch.WimpLevel > 0 {
			ch.SendMessage(fmt.Sprintf("Your current wimp level is %d hit points.\r\n", ch.WimpLevel))
		} else {
			ch.SendMessage("At the moment, you're not a wimp. (sure, sure...)\r\n")
		}
		return true
	}

	wimpLevel := 0
	fmt.Sscanf(arg, "%d", &wimpLevel)

	if wimpLevel > 0 {
		if wimpLevel < 0 {
			ch.SendMessage("Heh, heh, heh.. we are jolly funny today, eh?\r\n")
		} else if wimpLevel > ch.MaxHealth {
			ch.SendMessage("That doesn't make much sense, now does it?\r\n")
		} else if wimpLevel > (ch.MaxHealth / 3) {
			ch.SendMessage("You can't set your wimp level above one third your hit points.\r\n")
		} else {
			ch.WimpLevel = wimpLevel
			ch.SendMessage(fmt.Sprintf("Okay, you'll wimp out if you drop below %d hit points.\r\n", wimpLevel))
		}
	} else {
		ch.WimpLevel = 0
		ch.SendMessage("Okay, you'll now tough out fights to the bitter end.\r\n")
	}
	return true
}

// ---------------------------------------------------------------------------
// do_display (do_prompt) — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doDisplay(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		ch.SendMessage("Monsters don't need displays.  Go away.\r\n")
		return true
	}

	arg = strings.TrimSpace(arg)

	if arg == "" {
		ch.SendMessage("Usage: prompt { H | M | V | T | F | all | none }\r\n")
		return true
	}

	if strings.EqualFold(arg, "on") || strings.EqualFold(arg, "all") {
		ch.Flags |= 1 << PrfDisphp
		ch.Flags |= 1 << PrfDispmmana
		ch.Flags |= 1 << PrfDispmove
		ch.Flags |= 1 << PrfDispTank
		ch.Flags |= 1 << PrfDispTarget
		ch.SendMessage("Ok.\r\n")
		return true
	}

	ch.Flags &^= 1 << PrfDisphp
	ch.Flags &^= 1 << PrfDispmmana
	ch.Flags &^= 1 << PrfDispmove
	ch.Flags &^= 1 << PrfDispTank
	ch.Flags &^= 1 << PrfDispTarget

	if !strings.EqualFold(arg, "off") {
		for _, c := range strings.ToLower(arg) {
			switch c {
			case 'h':
				ch.Flags |= 1 << PrfDisphp
			case 'f':
				ch.Flags |= 1 << PrfDispTarget
			case 'm':
				ch.Flags |= 1 << PrfDispmmana
			case 't':
				ch.Flags |= 1 << PrfDispTank
			case 'v':
				ch.Flags |= 1 << PrfDispmove
			}
		}
	}

	ch.SendMessage("Ok.\r\n")
	return true
}

// ---------------------------------------------------------------------------
// do_gen_write — from act.other.c subcmd=SCMD_BUG/SCMD_TYPO/SCMD_IDEA/SCMD_TODO
// ---------------------------------------------------------------------------

func (w *World) doGenWrite(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	if arg == "" {
		switch cmd {
		case "bug":
			ch.SendMessage("Describe the bug you've discovered?\r\n")
		case "typo":
			ch.SendMessage("What typo did you find?\r\n")
		case "idea":
			ch.SendMessage("What is your idea?\r\n")
		case "todo":
			ch.SendMessage("What would you like to see added?\r\n")
		default:
			ch.SendMessage("Report what?\r\n")
		}
		return true
	}

	switch cmd {
	case "bug":
		ch.SendMessage("Bug reported. Thanks!")
	case "typo":
		ch.SendMessage("Typo reported. Thanks!")
	case "idea":
		ch.SendMessage("Idea noted. Thanks!")
	case "todo":
		ch.SendMessage("Todo noted. Thanks!")
	default:
		ch.SendMessage("Reported. Thanks!")
	}
	return true
}

// ---------------------------------------------------------------------------
// do_gen_tog — from act.other.c subcmd toggle commands
// ---------------------------------------------------------------------------

func (w *World) doGenTog(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	toggleMessages := map[string][2]string{
		"summon":    {"You are now summonable by others.\r\n", "You are no longer summonable.\r\n"},
		"nohassle":  {"You are now immune to annoying players.\r\n", "You may now be hassled again.\r\n"},
		"brief":     {"Brief mode on.\r\n", "Brief mode off.\r\n"},
		"compact":   {"Compact mode on.\r\n", "Compact mode off.\r\n"},
		"notell":    {"You are now deaf to tells.\r\n", "You may now receive tells again.\r\n"},
		"noauction": {"You are now deaf to auctions.\r\n", "You can now hear auctions again.\r\n"},
		"deaf":      {"You are now deaf to all shouts.\r\n", "You can now hear shouts again.\r\n"},
		"nogossip":  {"You are now deaf to gossip.\r\n", "You can now hear gossip again.\r\n"},
		"nogratz":   {"You will no longer see grat messages.\r\n", "You will now see grat messages again.\r\n"},
		"nowiz":     {"You are now deaf to the WizChannel.\r\n", "You can now hear the WizChannel again.\r\n"},
		"quest":     {"You will now see quest announcements.\r\n", "You will no longer see quest announcements.\r\n"},
		"roomflags": {"Room flags on.\r\n", "Room flags off.\r\n"},
		"norepeat":  {"No repeat mode on.\r\n", "No repeat mode off.\r\n"},
		"holylight": {"Holy light mode on.\r\n", "Holy light mode off.\r\n"},
		"autocxits": {"Auto exits on.\r\n", "Auto exits off.\r\n"},
		"npcident":  {"NPC identify mode on.\r\n", "NPC identify mode off.\r\n"},
		"nonewbie":  {"You will now see newbie chat.\r\n", "You will no longer see newbie chat.\r\n"},
		"noctell":   {"You are now deaf to clan tells.\r\n", "You can now hear clan tells again.\r\n"},
		"nobroad":   {"You are now deaf to broadcasts.\r\n", "You can now hear broadcasts again.\r\n"},
	}

	toggleFlags := map[string]int{
		"summon":    PrfSummonable,
		"nohassle":  PrfNohassle,
		"brief":     PrfBrief,
		"compact":   PrfCompact,
		"notell":    PrfNotell,
		"noauction": PrfNoAuctions,
		"deaf":      PrfDeaf,
		"nogossip":  PrfNoGossip,
		"nogratz":   PrfNoGratz,
		"nowiz":     PrfNowiz,
		"quest":     PrfQuest,
		"roomflags": PrfRoomFlags,
		"norepeat":  PrfNoRepeat,
		"holylight": PrfHolyLight,
		"autocxits": PrfAutoexit,
		"npcident":  PrfAutoexit, // reuse autoexit flag for ident
		"nonewbie":  PrfNoNewbie,
		"noctell":   PrfNoCTell,
		"nobroad":   PrfNoBroad,
	}

	// Map user-facing cmd to internal toggle key
	cmdMap := map[string]string{
		"summon":    "summon",
		"nohassle":  "nohassle",
		"brief":     "brief",
		"compact":   "compact",
		"notell":    "notell",
		"noauction": "noauction",
		"deaf":      "deaf",
		"nogossip":  "nogossip",
		"nogratz":   "nogratz",
		"nowiz":     "nowiz",
		"quest":     "quest",
		"roomflags": "roomflags",
		"norepeat":  "norepeat",
		"holylight": "holylight",
		"autocxits": "autocxits",
		"npcident":  "npcident",
		"nonewbie":  "nonewbie",
		"noctell":   "noctell",
		"nobroad":   "nobroad",
	}

	key, ok := cmdMap[cmd]
	if !ok {
		ch.SendMessage("Unknown toggle.\r\n")
		return true
	}

	flag, ok := toggleFlags[key]
	if !ok {
		ch.SendMessage("Unknown toggle.\r\n")
		return true
	}
	msgs, ok := toggleMessages[key]
	if !ok {
		ch.SendMessage("Unknown toggle.\r\n")
		return true
	}

	// Special checks
	if key == "nowiz" && ch.Level < LVL_IMMORT {
		ch.SendMessage("You are not holy enough to use that feature.\r\n")
		return true
	}

// TODO: noctell implementation (clan system)

	if ch.Flags&(1<<flag) != 0 {
		ch.Flags &^= 1 << flag
		ch.SendMessage(msgs[1])
	} else {
		ch.Flags |= 1 << flag
		ch.SendMessage(msgs[0])
	}

	return true
}

// ---------------------------------------------------------------------------
// do_afk — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doAFK(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	if ch.Flags&(1<<PrfAFK) != 0 {
		ch.Flags &^= 1 << PrfAFK
		ch.AFK = false
		ch.AFKMessage = ""
		msg := fmt.Sprintf("%s is no longer AFK.\r\n", ch.Name)
		actToRoom(w, ch.GetRoomVNum(), msg, ch.Name)
		ch.SendMessage("You are no longer AFK.\r\n")
	} else {
		ch.Flags |= 1 << PrfAFK
		ch.AFK = true
		ch.AFKMessage = arg
		msg := fmt.Sprintf("%s is now AFK.\r\n", ch.Name)
		actToRoom(w, ch.GetRoomVNum(), msg, ch.Name)
		if arg != "" {
			ch.SendMessage("You are now AFK: " + arg + "\r\n")
		} else {
			ch.SendMessage("You are now AFK.\r\n")
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// do_auto — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doAuto(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	arg = strings.TrimSpace(arg)

	if arg == "" {
		var autos []string
		if ch.Flags&(1<<PrfAutoexit) != 0 {
			autos = append(autos, "exits")
		}
		if ch.Flags&(1<<PrfAutoLoot) != 0 {
			autos = append(autos, "loot")
		}
		if ch.Flags&(1<<PrfAutoGold) != 0 {
			autos = append(autos, "gold")
		}
		if ch.Flags&(1<<PrfAutoSplit) != 0 {
			autos = append(autos, "split")
		}

		if len(autos) == 0 {
			ch.SendMessage("None.\r\n")
		} else {
			ch.SendMessage("Autos: " + strings.Join(autos, ", ") + "\r\n")
		}
		return true
	}

	switch strings.ToLower(arg) {
	case "exit", "exits":
		if ch.Flags&(1<<PrfAutoexit) != 0 {
			ch.Flags &^= 1 << PrfAutoexit
			ch.SendMessage("Auto exits off.\r\n")
		} else {
			ch.Flags |= 1 << PrfAutoexit
			ch.SendMessage("Auto exits on.\r\n")
		}
	case "loot":
		if ch.Flags&(1<<PrfAutoLoot) != 0 {
			ch.Flags &^= 1 << PrfAutoLoot
			ch.SendMessage("Auto loot off.\r\n")
		} else {
			ch.Flags |= 1 << PrfAutoLoot
			ch.SendMessage("Auto loot on.\r\n")
		}
	case "gold":
		if ch.Flags&(1<<PrfAutoGold) != 0 {
			ch.Flags &^= 1 << PrfAutoGold
			ch.SendMessage("Auto gold off.\r\n")
		} else {
			ch.Flags |= 1 << PrfAutoGold
			ch.SendMessage("Auto gold on.\r\n")
		}
	case "split":
		if ch.Flags&(1<<PrfAutoSplit) != 0 {
			ch.Flags &^= 1 << PrfAutoSplit
			ch.SendMessage("Auto split off.\r\n")
		} else {
			ch.Flags |= 1 << PrfAutoSplit
			ch.SendMessage("Auto split on.\r\n")
		}
	default:
		ch.SendMessage("What do you want to make automatic?\r\n")
	}
	return true
}

// ---------------------------------------------------------------------------
// do_transform — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doTransform(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	if ch.Flags&(1<<PlrWerewolf) != 0 {
		// Werewolf: toggle affWerewolf
		if ch.IsAffected(affWerewolf) {
			ch.SetAffect(affWerewolf, false)
			if ch.Health > ch.MaxHealth {
				ch.Health = ch.MaxHealth
			}
			ch.SendMessage("You revert back to your human form.\r\n")
			actToRoom(w, ch.GetRoomVNum(), fmt.Sprintf("%s transforms back into %s human form.\r\n", ch.Name, hisHer(ch.GetSex())), ch.Name)
		} else {
			ch.SetAffect(affWerewolf, true)
			bonus := randRange(2, 6) * 10
			ch.Health += bonus
			if ch.Health > 666 {
				ch.Health = 666
			}
			if ch.Health > ch.MaxHealth {
				ch.MaxHealth = ch.Health
			}
			ch.SendMessage("You transform into a werewolf!\r\n")
			actToRoom(w, ch.GetRoomVNum(), fmt.Sprintf("%s transforms into a werewolf!\r\n", ch.Name), ch.Name)
		}
	} else if ch.Flags&(1<<PlrVampire) != 0 {
		// Vampire: toggle affVampire
		if ch.IsAffected(affVampire) {
			ch.SetAffect(affVampire, false)
			ch.SendMessage("You revert back to your human form.\r\n")
			actToRoom(w, ch.GetRoomVNum(), fmt.Sprintf("%s transforms back into %s human form.\r\n", ch.Name, hisHer(ch.GetSex())), ch.Name)
		} else {
			ch.SetAffect(affVampire, true)
			bonus := randRange(2, 6) * 10
			ch.Mana += bonus
			ch.SendMessage("You transform into a vampire!\r\n")
			actToRoom(w, ch.GetRoomVNum(), fmt.Sprintf("%s transforms into a vampire!\r\n", ch.Name), ch.Name)
		}
	} else {
		ch.SendMessage("You have no idea how to transform!\r\n")
	}
	return true
}

// ---------------------------------------------------------------------------
// do_ride — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doRide(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	arg = strings.TrimSpace(arg)
	if arg == "" {
		ch.SendMessage("Ride what?\r\n")
		return true
	}

	if ch.IsFighting() {
		ch.SendMessage("You are fighting!\r\n")
		return true
	}

	room := w.GetRoomInWorld(ch.GetRoomVNum())
	if room == nil || !isOutdoors(room) {
		ch.SendMessage("You can't ride in here!\r\n")
		return true
	}

	// Find mount
	_, mountMob := w.findCharInRoom(ch, ch.GetRoomVNum(), arg)
	if mountMob == nil {
		ch.SendMessage("There's nothing here to ride!\r\n")
		return true
	}

	if !mountMob.HasFlag("mountable") {
		ch.SendMessage("You can't ride that!\r\n")
		return true
	}

	if ch.IsAffected(affMounted) {
		ch.SendMessage("You're already riding!\r\n")
		return true
	}

	// For now, check if mount already has a rider by checking affMounted
	// We can check via a simple loop
	mobs := w.GetMobsInRoom(ch.GetRoomVNum())
	mountAlreadyRidden := false
	for _, m := range mobs {
		if m.HasFlag("mount") && m.HasFlag("mounted") {
			mountAlreadyRidden = true
			break
		}
	}
	_ = mountAlreadyRidden

	ch.SetAffect(affMounted, true)
	ch.Following = mountMob.GetShortDesc()
	ch.SendMessage(fmt.Sprintf("You climb onto %s.\r\n", mountMob.GetShortDesc()))
	actToRoom(w, ch.GetRoomVNum(), fmt.Sprintf("%s climbs onto %s.\r\n", ch.Name, mountMob.GetShortDesc()), ch.Name)
	return true
}

// ---------------------------------------------------------------------------
// do_dismount — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doDismount(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	if !ch.IsAffected(affMounted) {
		ch.SendMessage("You need to be riding before you can dismount!\r\n")
		return true
	}

	ch.SetAffect(affMounted, false)
	ch.SendMessage("You dismount.\r\n")
	actToRoom(w, ch.GetRoomVNum(), fmt.Sprintf("%s dismounts.\r\n", ch.Name), ch.Name)
	return true
}

// ---------------------------------------------------------------------------
// do_yank — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doYank(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	arg = strings.TrimSpace(arg)
	if arg == "" {
		ch.SendMessage("Yank whom from what?\r\n")
		return true
	}

	victimPl, _ := w.findCharInRoom(ch, ch.GetRoomVNum(), arg)
	if victimPl == nil {
		ch.SendMessage("They aren't here.\r\n")
		return true
	}

	// Must be a follower
	if victimPl.Following != ch.Name {
		ch.SendMessage("They aren't following you!\r\n")
		return true
	}

	if victimPl.GetPosition() >= combat.PosStanding {
		ch.SendMessage("They're already on their feet.\r\n")
		return true
	}

	victimPl.SetPosition(combat.PosStanding)
	ch.SendMessage(fmt.Sprintf("You yank %s to %s feet.\r\n", victimPl.Name, hisHer(victimPl.GetSex())))
	victimPl.SendMessage(fmt.Sprintf("%s yanks you to your feet.\r\n", ch.Name))
	actToRoom(w, ch.GetRoomVNum(), fmt.Sprintf("%s yanks %s to %s feet.\r\n", ch.Name, victimPl.Name, hisHer(victimPl.GetSex())), ch.Name)
	return true
}

// ---------------------------------------------------------------------------
// do_peek — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doPeek(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	arg = strings.TrimSpace(arg)
	if arg == "" {
		ch.SendMessage("Peek at whom?\r\n")
		return true
	}

	if ch.Class != ClassThief && ch.Class != ClassAssassin && ch.Level < LVL_IMMORT {
		ch.SendMessage("You have no idea how to peek!\r\n")
		return true
	}

	victimPl, _ := w.findCharInRoom(ch, ch.GetRoomVNum(), arg)
	if victimPl == nil {
		ch.SendMessage("They aren't here.\r\n")
		return true
	}

	percent := randRange(1, 101)
	skill := ch.GetSkill("peek")
	if percent > skill {
		ch.SendMessage(fmt.Sprintf("You try to peek at %s but fail.\r\n", victimPl.Name))
		return true
	}

	ch.SendMessage(fmt.Sprintf("You peek at %s's belongings:\r\n", victimPl.Name))
	ch.SendMessage("[Equipment and inventory]\r\n")
	// Improve skill
	skillVal := ch.GetSkill("peek")
	if skillVal > 0 && skillVal < 97 && randRange(1, 200) <= ch.Stats.Wis+ch.Stats.Int {
		skillVal += randRange(1, 3)
		if skillVal > 97 {
			skillVal = 97
		}
		ch.SetSkill("peek", skillVal)
		if randRange(1, 3) == 3 {
			ch.SendMessage("Your skill in peek improves.\r\n")
		}
	}

	return true
}

// ---------------------------------------------------------------------------
// do_recall — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doRecall(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	if ch.Level > 5 {
		ch.SendMessage("You are too powerful to be teleported to the temple!\r\n")
		return true
	}

	if ch.IsFighting() {
		ch.SendMessage("No way!  You are fighting!\r\n")
		return true
	}

	room := w.GetRoomInWorld(ch.GetRoomVNum())
	if room != nil && (hasRoomFlag(room, "no_recall") || hasRoomFlag(room, "bfr")) {
		ch.SendMessage("You can't recall from this room!\r\n")
		return true
	}

	ch.SetPosition(combat.PosStanding)

	// Recall to temple (room 8004)
	recallRoom := 8004

	ch.SendMessage("You close your eyes and pray...\r\n")
	actToRoom(w, ch.GetRoomVNum(), fmt.Sprintf("%s closes %s eyes and prays...\r\n", ch.Name, hisHer(ch.GetSex())), ch.Name)

	ch.SendMessage("You are recalled!\r\n")
	ch.RoomVNum = recallRoom
	actToRoom(w, recallRoom, fmt.Sprintf("%s appears in the room.\r\n", ch.Name), "")

	return true
}

// ---------------------------------------------------------------------------
// do_stealth — from act.other.c (superior sneak)
// ---------------------------------------------------------------------------

func (w *World) doStealth(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	if ch.IsAffected(affMounted) {
		ch.SendMessage("You can't sneak around on a mount!\r\n")
		return true
	}

	skill := ch.GetSkill("stealth")
	if skill <= 0 {
		ch.SendMessage("You have no idea how to become one with the shadows.\r\n")
		return true
	}

	percent := randRange(1, 101)
	if percent > skill {
		ch.SendMessage("You try to become one with the shadows, but fail.\r\n")
		return true
	}

	ch.SetAffect(affSneak, true)
	ch.SendMessage("You become one with the shadows.\r\n")
	return true
}

// ---------------------------------------------------------------------------
// do_appraise — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doAppraise(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	arg = strings.TrimSpace(arg)
	if arg == "" {
		ch.SendMessage("Appraise what?\r\n")
		return true
	}

	// Find object in inventory
	obj := w.findObjNear(ch, arg)
	if obj == nil {
		ch.SendMessage("You don't have that item.\r\n")
		return true
	}

	cost := obj.Prototype.Cost
	skill := ch.GetSkill("appraise")
	percent := randRange(1, 101)

	if percent > skill {
		// Failed appraise — random value
		badCost := cost + randRange(-cost, cost*2)
		if badCost < 0 {
			badCost = 0
		}
		ch.SendMessage(fmt.Sprintf("You estimate it's worth %d gold coins.\r\n", badCost))
		return true
	}

	// Successful appraise
	actual := cost + randRange(-20, 20)
	if actual < 0 {
		actual = 0
	}
	ch.SendMessage(fmt.Sprintf("You estimate it's worth %d gold coins.\r\n", actual))

	// Improve skill
	skillVal := ch.GetSkill("appraise")
	if skillVal > 0 && skillVal < 97 && randRange(1, 200) <= ch.Stats.Wis+ch.Stats.Int {
		skillVal += randRange(1, 3)
		if skillVal > 97 {
			skillVal = 97
		}
		ch.SetSkill("appraise", skillVal)
		if randRange(1, 3) == 3 {
			ch.SendMessage("Your skill in appraise improves.\r\n")
		}
	}

	return true
}

// ---------------------------------------------------------------------------
// do_inactive — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doInactive(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	if ch.Flags&(1<<PrfInactive) != 0 {
		ch.Flags &^= 1 << PrfInactive
		ch.SendMessage("You are now active.\r\n")
	} else {
		ch.Flags |= 1 << PrfInactive
		ch.SendMessage("You are now inactive.\r\n")
	}
	return true
}

// ---------------------------------------------------------------------------
// do_scout — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doScout(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	skill := ch.GetSkill("scout")
	if skill <= 0 {
		ch.SendMessage("You have no idea how to scout.\r\n")
		return true
	}

	arg = strings.TrimSpace(arg)
	if arg == "" {
		ch.SendMessage("Scout which direction?\r\n")
		return true
	}

	room := w.GetRoomInWorld(ch.GetRoomVNum())
	if room == nil || !isOutdoors(room) {
		ch.SendMessage("You can't scout from in here!\r\n")
		return true
	}

	currentRoom := w.GetRoomInWorld(ch.GetRoomVNum())
	if currentRoom == nil {
		ch.SendMessage("Nothing in that direction.\r\n")
		return true
	}
	exitObj, exitOk := currentRoom.Exits[strings.ToLower(arg)]
	if !exitOk {
		ch.SendMessage("Nothing in that direction.\r\n")
		return true
	}

	toRoom := w.GetRoomInWorld(exitObj.ToRoom)
	if toRoom == nil {
		ch.SendMessage("Nothing in that direction.\r\n")
		return true
	}

	// Sector description
	sectorNames := map[int]string{
		0:  "the cobblestones of a city",
		1:  "a wide swath of field",
		2:  "the dense forest",
		3:  "high hills",
		4:  "jagged mountains",
		5:  "a large stretch of water",
		6:  "thin air",
		7:  "a murky swamp",
		8:  "the inside of a structure",
		9:  "a vast wasteland",
		10: "the watery depths",
		11: "the endless elemental plane",
	}

	sectorDesc, ok := sectorNames[toRoom.Sector]
	if !ok {
		sectorDesc = "the endless elemental plane"
	}

	ch.SendMessage(fmt.Sprintf("There is %s to the %s.\r\n", sectorDesc, arg))

	// Room flags
	if isDark(toRoom) {
		ch.SendMessage("It is dark in that direction.\r\n")
	}
	if hasRoomFlag(toRoom, "death") {
		ch.SendMessage("You sense certain death in that direction.\r\n")
	}
	if hasRoomFlag(toRoom, "tunnel") {
		ch.SendMessage("It looks narrow in that direction.\r\n")
	}

	// Count people
	players := w.GetPlayersInRoom(toRoom.VNum)
	mobs := w.GetMobsInRoom(toRoom.VNum)

	playerCount := 0
	for _, p := range players {
		if !p.IsNPC() {
			playerCount++
		}
	}

	totalCount := playerCount + len(mobs)
	if totalCount == 0 {
		ch.SendMessage("You see no one there.\r\n")
	} else if totalCount == 1 {
		ch.SendMessage("You see one being there.\r\n")
	} else if totalCount < 10 {
		ch.SendMessage(fmt.Sprintf("You see a group of %d beings there.\r\n", totalCount))
	} else {
		ch.SendMessage("You see a huge crowd there!\r\n")
	}

	return true
}

// ---------------------------------------------------------------------------
// do_roll — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doRoll(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	arg = strings.TrimSpace(arg)
	maxRoll := 100
	if arg != "" {
		fmt.Sscanf(arg, "%d", &maxRoll)
		if maxRoll < 1 {
			maxRoll = 1
		}
	}

	result := randRange(1, maxRoll)
	ch.SendMessage(fmt.Sprintf("You roll %d out of %d.\r\n", result, maxRoll))
	actToRoom(w, ch.GetRoomVNum(), fmt.Sprintf("%s rolls %d out of %d.\r\n", ch.Name, result, maxRoll), ch.Name)
	return true
}
