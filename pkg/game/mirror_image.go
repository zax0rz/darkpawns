package game

// spell_mirror_image's world-side effects (src/spells.c:1132-1164). The spell
// layer owns the caster output and the room act; these two methods own the
// clone's identity copy and the mob memory/hunting hand-off, both of which need
// pkg/game state the spell package cannot import.

// ConfigureMirrorClone applies spell_mirror_image's identity copies to a
// freshly created clone (src/spells.c:1145-1150):
//
//	SET_NAME(clone, GET_NAME(ch));                        // NPC: short_descr
//	clone->player.short_descr = GET_NAME(ch);
//	sprintf(szBuffer, "%s %s\r\n", GET_NAME(ch), title);
//	clone->player.long_descr = szBuffer;
//	clone->player.description = ch->player.description;
//	clone->player.sex = ch->player.sex;
//
// SET_NAME on a mobile writes short_descr (src/utils.h:277-279), so the clone's
// prototype keywords ("clone") are left untouched, matching C's `stat clone`.
func (w *World) ConfigureMirrorClone(clone, caster any) {
	mob, ok := clone.(*MobInstance)
	if !ok {
		return
	}
	casterIdentity, ok := caster.(interface {
		GetName() string
		GetTitle() string
		GetDescription() string
		GetSex() int
	})
	if !ok {
		return
	}
	name := casterIdentity.GetName()
	mob.SetName(name)
	mob.SetLiveLongDesc(name + " " + casterIdentity.GetTitle() + "\r\n")
	mob.SetLiveDetailedDesc(casterIdentity.GetDescription())
	mob.SetSexOverride(casterIdentity.GetSex())
}

// MirrorImageHandoff points every watching mob's memory and hunting from the
// caster to the clone (src/spells.c:1155-1162):
//
//	for (mob = character_list; mob; mob = mob->next) {
//	  if (MOB_FLAGGED(mob, MOB_MEMORY) && MEMORY(mob))
//	    if (forget(mob, ch)) remember(mob, clone);
//	  if (HUNTING(mob) == ch) set_hunting(mob, clone);
//	}
//
// remember() is keyed by name, and the clone's short_descr is the caster's name
// (ConfigureMirrorClone), so the hand-off keeps the same name a watching mob
// already remembered. The hunting arm follows the port's established
// name-based translation of `HUNTING(mob) == ch` (cf. forgetRawKillVictim,
// pkg/game/raw_kill.go:110).
func (w *World) MirrorImageHandoff(clone, caster any) {
	casterNamed, ok := caster.(interface{ GetName() string })
	if !ok {
		return
	}
	casterName := casterNamed.GetName()
	cloneName := casterName
	if cloneNamed, ok := clone.(interface{ GetName() string }); ok {
		cloneName = cloneNamed.GetName()
	}
	cloneActor, cloneIsActor := clone.(Actor)

	for _, mob := range w.GetAllMobs() {
		if mob.HasMobFlag(MobFlagMemory) && len(mob.GetMemory()) > 0 {
			if mob.Forget(casterName) {
				mob.Remember(cloneName)
			}
		}
		if cloneIsActor && mob.GetHunting() == casterName {
			w.SetHuntingTarget(mob, cloneActor)
		}
	}
}
