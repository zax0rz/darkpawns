package game

import (
	"strings"
	"testing"
)

// Every C skill (SKILL_BACKSTAB 131 through SKILL_SCOUT 192, src/spells.h)
// stores under the key its handler reads. Handler keys are identifiers, so a
// stored name with a space is a C display name no handler looks up, which is
// how the first-player God lost tiger punch and seven more (DP-1342).
func TestCSkillsStoreUnderHandlerKeys(t *testing.T) {
	for num := 131; num <= 192; num++ {
		name := SkillStorageName(num)
		if name == "" || strings.ContainsAny(name, " ") {
			t.Errorf("skill %d (%q) stores under %q, not a handler key", num, SkillCatalogName(num), name)
		}
	}
	for num, key := range map[int]string{
		135: SkillPickLock, 155: SkillStrike, 156: SkillSerpentKick, 168: SkillFleshAlter,
		179: SkillFirstAid, 180: SkillDetect, 188: SkillDragonKick, 189: SkillTigerPunch,
	} {
		if got := SkillStorageName(num); got != key {
			t.Errorf("skill %d stores under %q, want %q", num, got, key)
		}
	}
}

// Saves written before DP-1342 hold these skills under the display name;
// they load under the handler's key, keeping the higher value when a save
// holds both.
func TestRestoreSkillsMigratesCatalogNames(t *testing.T) {
	p := NewPlayer(1, "Ninja", 1001)
	restoreSkills(p, map[string]int{"tiger punch": 72, "dragon kick": 40, SkillDragonKick: 55, "kick": 30})
	for key, want := range map[string]int{SkillTigerPunch: 72, SkillDragonKick: 55, "kick": 30} {
		if got := p.GetSkill(key); got != want {
			t.Errorf("GetSkill(%q) = %d, want %d", key, got, want)
		}
	}
	if got := p.GetSkill("tiger punch"); got != 0 {
		t.Errorf("the old key still holds %d", got)
	}
}

// The first-player God can use every C skill, as C's init_char gives it
// each at 100 (src/db.c:3059-3064).
func TestFirstPlayerGodHasMartialArts(t *testing.T) {
	p := NewPlayer(1, "Firstgod", 1001)
	BootstrapFirstPlayerGod(p)
	for _, key := range []string{SkillTigerPunch, SkillDragonKick, SkillPickLock, SkillFirstAid, SkillSerpentKick} {
		if got := p.GetSkill(key); got != 100 {
			t.Errorf("God's %s = %d, want 100", key, got)
		}
	}
}
