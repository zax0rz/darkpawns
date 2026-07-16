# BRIEF — Phase 1 slice 3: `improve_skill` use-based gain (fidelity rewrite)

**Status:** keystone = Claude (this slice). Carve-out candidates flagged at bottom.
**Oracle gate owner:** Claude (workers lack `DP_ORACLE_BIN`).
**Branch off:** `main` (has #356/#357/#359 merged).

---

## The finding

`pkg/game/combat_helpers.go:37 improveSkill` is **entirely invented** — it is not a
port of C's `improve_skill`. It diverges on every axis: gate formula, draw ranges,
draw *order* (hence draw count for maxed skills), the increment, the skill ceiling,
and the player-facing message. Because it runs on the success branch of ~9 combat
skills, every one of those commands currently desyncs the seeded PRNG stream and
prints the wrong text.

## Canonical C — `src/act.other.c:1704`

```c
void improve_skill(struct char_data *ch, int skill)
{
  extern char *spells[];
  int percent, newpercent;
  char skillbuf[MAX_STRING_LENGTH];

  if (IS_NPC(ch))
    return;
  percent = GET_SKILL(ch, skill);
  if (number(1, 200) > GET_WIS(ch) + GET_INT(ch))   /* DRAW #1 — always, for a PC */
     return;
  if (percent >= 97 || percent <= 0)
     return;
  newpercent = number(1, 3);                          /* DRAW #2 — only if gate passed */
  percent += newpercent;
  SET_SKILL(ch, skill, percent);
  if (newpercent == 3) {
     sprintf(skillbuf, "Your skill in %s improves.\r\n", spells[skill]);
     send_to_char(skillbuf, ch);
  }
}
```

### Draw-parity contract (LAW)

1. `number(1,200)` is drawn on **every** PC call — *before* the `percent` bounds
   check. A skill already at ≥97 still consumes exactly one draw. The current Go
   code checks `cur<=0 || cur>=100` first and draws **zero** for a maxed skill →
   stream desync the moment anyone uses a mastered skill. **Order is not negotiable.**
2. `number(1,3)` is drawn **only** when the stat gate passes *and* `percent∈(1,96)`.
3. `Number(from,to)` = one `Uniform()` draw regardless of range (verified
   `pkg/dprng/cmwc.go:72`), so range differences change *values/branches*, never
   the per-call draw cost. The desync above is purely the reordered bounds check.

### Other divergences to correct
- **Ceiling `97`, not `100`.** A skill caps its use-based gain at 96→(97..99); it
  never self-improves past that band.
- **Increment is `number(1,3)`,** not `+1`.
- **Message is `"Your skill in %s improves.\r\n"` and fires only on a +3 roll** —
  not "You feel a bit more competent…" on every gain. `spells[skill]` is the DP
  catalog name; Go's `Skill*` constants already ARE those names
  (`skills.go`: `SkillBackstab="backstab"` etc.), so pass `skill` straight through.
- **`IS_NPC` guard** — keep it (mobs must not roll or desync via this path).

## Faithful Go (target)

```go
// improveSkill ports src/act.other.c:1704 improve_skill(). Use-based skill gain on
// a successful skill use. Draw-parity is law: number(1,200) is drawn on EVERY PC
// call BEFORE the percent bounds check (a maxed skill still burns one draw);
// number(1,3) is drawn only when the stat gate passes and percent is in (0,97).
// The "improves" line fires only on a +3 roll. spells[skill] == the Skill* constant.
func improveSkill(ch *Player, skill string) {
	if ch.IsNPC() {
		return
	}
	percent := ch.GetSkill(skill)
	if dprng.Number(1, 200) > ch.GetWis()+ch.GetInt() {
		return
	}
	if percent >= 97 || percent <= 0 {
		return
	}
	newpercent := dprng.Number(1, 3)
	percent += newpercent
	ch.SetSkill(skill, percent)
	if newpercent == 3 {
		ch.SendMessage(fmt.Sprintf("Your skill in %s improves.\r\n", skill))
	}
}
```

## Oracle gate

New scenario `cmd/dp-oracle-diff/scenarios/improve-skill.txt`: spawn a mob, engage,
repeat a reliably-improving skill (backstab is cleanest — thief, big base gain) until
a +3 fires, probing char output each swing. Must be RED against current invented code
and GREEN after the rewrite. Also re-run `combat-swing`, `skills-practice`,
`guild-practice` — all must stay green (the reordered draw is the whole point; a
regression there means the wiring order is still wrong).

## Call-site audit (the fan-out)

The 9 `improveSkill(...)` sites in `pkg/game/skill_combat.go` (+ `skill_berserk_kuji.go`,
which has a known C dangling-else quirk) must each sit at the exact point in their C
`do_*` function's control/draw order. Rewriting the core doesn't guarantee each site
fires on the right branch. This is per-site C-reading against `act.offensive.c`,
`new_cmds*.c`, `act.item.c`.

---

## Kimi/GLM worthiness

- **Core rewrite + backstab oracle proof — Claude only.** ~15 lines but pure
  draw-parity + first-class message text; precisely GLM's historical failure mode.
- **Call-site audit (slice 3b) — borderline Kimi.** Real, well-bounded C-reading,
  but its success criterion is "oracle green," which the worker can't run — it would
  reason about draw-order blind. Only hand off with each C site pre-cited AND with
  Claude gating every site individually. Not GLM.
- **GLM — no draw-parity work.** Reserve GLM for deterministic, no-RNG,
  heavily-cited ports (some of the Phase-2 spell-effect tail) where "green == done"
  actually holds.
