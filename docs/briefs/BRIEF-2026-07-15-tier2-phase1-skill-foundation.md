# BRIEF — Phase 1: C-faithful skill-system foundation (deterministic spine)

**For:** Claude (implementing — Option B, codex blocked to 07-21). **Gate owner:** Claude.
**Branch:** `refactor/skill-system-foundation` (already created off main).
**Findings:** DP-1116 (practice O36), DP-1128 (listskills collision O37), DP-1129 (spells stub O38),
+ the invented `learn`/`forget`. **Method:** C read directly from src (cited below). Oracle red→green
gated by `cmd/dp-oracle-diff/scenarios/skills-practice.txt` (written, captured RED below).

> This is the **deterministic display/command spine** only. The stateful mutation pieces —
> guild-mob `practice <skill>` learning + use-based `improve_skill` — are **§5 follow-on** (they need
> the guild spec proc + combat integration; less oracle-friendly). Land the spine first, oracle-proven.

---

## 0. The decisive discovery
**C has exactly ONE skill/spell command: `practice`** (interpreter.c:618 `{ "practice", POS_RESTING,
do_practice, 1, 0 }`). There is **no `skills`, no `spells`, no `learn`, no `forget`, no `*info`.**
Everything else in Go's `pkg/command/skill_commands.go` + `cmdSpells` is invented and gets retired.
Go also carries a **whole invented `SkillManager`** (points/slots/levels/progress bars, `player.SkillManager`)
that is a parallel representation to the C-faithful per-skill percentage store (`player.GetSkill/SetSkill`,
`player.Practices`). The two have drifted; the combat code already uses the faithful `GetSkill` percentages.

## 1. Oracle RED (captured 2026-07-15, DP_SEED=1, fresh L1 K warrior at 8162)
```
--- practice ---
C : You have 4 practice sessions remaining.
    You know of the following skills:
    kick                  (not learned)
Go: Practice what? Usage: practice <skill>

--- practice kick ---
C : You can only practice skills in your guild.
Go: You practice Kick. Progress: 11% (Level 10)
```
(Also note **practice count 4 vs 2** — a real creation-value divergence, see §3.)

## 2. The C model (exact — port verbatim)
### 2a. `do_practice` (act.other.c:543-553)
```
one_argument(argument, arg);
if (*arg) send_to_char("You can only practice skills in your guild.\r\n", ch);
else      list_skills(ch);
```
The guild MOB spec proc (spec_procs.c:201 `SPECIAL(guild)`) is what actually learns a named skill —
so a player NOT standing on a guildmaster always gets the "...in your guild." line. (§5.)

### 2b. `list_skills` (spec_procs.c:157-198)
```
practices line: !GET_PRACTICES → "You have no practice sessions remaining.\r\n"
                else → "You have %d practice session%s remaining.\r\n" (plural "s" unless ==1)
"You know of the following %ss:\r\n"   where %s = SPLSKL(ch) = prac_types[PRAC_TYPE[class]]
for sortpos 1..MAX_SKILLS:  i = spell_sort_info[sortpos]         // boot-built ALPHA sort by name
    if GET_LEVEL(ch) >= spell_info[i].min_level[class]:
        mana = mag_manacost(ch, find_skill_num(spells[i]))
        line = sprintf("%-20s %s %s\r\n", spells[i], how_good(GET_SKILL(ch,i)), mana? manastring : "")
page_string(...)   // paginated; harness sees full text
```
- `manastring` (spells only, mana>0): `"( %s%d %s%s )"` = `( <RED>N <"mana"|"psi pts"><NRM> )`
  (psi pts if psionic/mystic). Skills have mana 0 → the `%s` is empty → **trailing `" "`** after how_good.
- **`how_good(pct)`** (spec_procs.c:108) — note every string has a **leading space**:
  `0→" (not learned)"  ≤10→" (awful)"  ≤20→" (bad)"  ≤40→" (poor)"  ≤55→" (average)"  ≤70→" (fair)"
   ≤80→" (good)"  ≤85→" (very good)"  ≤98→" (superb)"  else→" (MASTER)"`
- Exact line for `kick` unlearned: `"kick"` + pad to 20 + `" "`(format) + `" (not learned)"`(how_good) +
  `" "` + `""` → the normalized RED shows `kick                  (not learned)`. **Match byte-for-byte.**

### 2c. `prac_params[4][NUM_CLASSES]` (class.c:261) — indexed `[param][class]`, class order
`MAG CLE THE WAR MAGU AVA ASS PAL NIN PSI RAN MYS`:
```
learned level : 95 95 85 80 95 95 85 80 85 95 80 95
max per prac  :100 100 25 25 100 100 25 25 25 100 25 100
min per prac  : 25 25  0  0 25 25  0  0  0 25  0 25
prac name     : SPELL SPELL SKILL SKILL SPELL BOTH SKILL BOTH BOTH BOTH SKILL BOTH
```
`prac_types[] = {"spell","skill","art"}`; SPELL=0, SKILL=1, BOTH=2→"art". Warrior→"skill".

## 3. Newbie practice count (do_start → advance_level, class.c)
`do_start` (class.c:501) sets base stats then calls `advance_level(ch)` once for L1. Warrior branch adds
`GET_PRACTICES(ch) += MAX(2, wis_app[GET_WIS(ch)].bonus)` (class.c:616-ish; assassin/paladin/ninja/ranger
use `MIN(2, MAX(1, wis_app.bonus))`, casters `MAX(2, wis_app.bonus)`). Under DP_SEED=1 the rolled WIS is
identical both servers → deterministic. The test warrior rolled a WIS giving bonus 4 → **4 practices**.
Go hardcodes `p.Practices = 2` (player.go:341). **Fix:** compute newbie practices via the per-class
advance_level formula using rolled WIS + `wis_app[].bonus`, in the SAME advance_level port that already
does HP/mana/move draw order (pkg/game/level.go). Verify `wis_app` table exists + matches C. This closes
the `4 vs 2` gap so the no-arg `practice` gate goes fully green.

## 4. Go implementation plan (this branch)
1. **Rewrite `CmdPractice`** (pkg/command/skill_commands.go:80) to `do_practice`: `len(args)==0` →
   `renderListSkills(player)`; else → `"You can only practice skills in your guild.\r\n"`. Delete the
   SkillManager path.
2. **New `renderListSkills`** faithful to §2b: practices line (plural rule) + `You know of the following
   <SPLSKL>s:` + per-skill loop over the **alpha-sorted** catalog filtered by `MinLevel[class] <= level`,
   `%-20s %s %s` with `how_good` + mana. Source data: `pkg/spells/spell_info.go` (`MinLevel[12]`), the
   `spells[]` name table + boot sort order (build `spellSortInfo` alpha by name if not present), and
   `mag_manacost`. **Read `GET_SKILL` percentages from the canonical `player.GetSkill`, NOT SkillManager.**
   Add `howGood(pct)` + `pracParams`/`pracTypes` tables verbatim from §2b/§2c.
3. **Retire invented commands** — remove registrations for `skills`, `spells`, `learn`, `forget`,
   `listskills` (commands.go:137-146,249) so they hit C's unknown-command path ("Huh?!?"). Delete
   `CmdSkills/CmdLearn/CmdForget/CmdListSkills` + `cmdSpells`. **Watch the cascade:** `SkillManager` and
   `engine.SkillType*` may be referenced elsewhere — grep before deleting; if `SkillManager` has non-display
   consumers, leave the struct but stop wiring it to player-facing commands (retire in a later pass).
   Keep `GetSkill/SetSkill/Practices` (canonical, used by combat).
4. **Newbie practices fix** (§3) in the advance_level port.
5. Unit tests: `howGood` boundaries (every band incl. leading space), plural practices line, SPLSKL per
   class, min_level filter + alpha sort, mana rendering for a caster spell, retired commands → unknown.

## 5. FOLLOW-ON (not this PR — needs its own slice)
- **Guild `practice <skill>` learning** (spec_procs.c:201): the guildmaster mob spec proc — GET_PRACTICES>0
  gate, `find_skill_num`, min_level gate, LEARNED cap, `percent += MIN(MAXGAIN, MAX(MINGAIN,
  int_app[INT].learn))`, `SET_SKILL(min(LEARNED, percent))`, the "You practice for a while..." /
  "already learned" / "now learned" strings. Needs the guild mob spec assigned + an oracle scenario that
  walks a newbie to their guildmaster (warrior guild room 8015 per guild_info, class.c).
- **`improve_skill`** (use-based gain on successful bash/kick/backstab/etc.) — RNG; wire into the combat
  skill path. Oracle-provable once combat-round capture (B-2 multi-round) matures.
- **`cast` contract** (DP-1117 O39) — spell target/eligibility/gating + mana pre-deduction bug. Large;
  its own brief. Depends on this foundation (canonical skill %s = spell knowledge).

## 6. Acceptance gate (this PR)
1. **Oracle:** `--scenario skills-practice` red→green — `practice` no-arg matches C's 3-line catalog
   (incl. `4 practice sessions` after §3) AND `practice kick` → "You can only practice skills in your
   guild." I run it (workers lack DP_ORACLE_BIN).
2. Unit tests §4.5 green; `skills`/`spells`/`learn`/`forget` return unknown-command (add a probe).
3. `make check-fmt vet` + `go test ./...` + import guard green; no WS schema break; SkillManager cascade
   resolved (build clean).

## 7. Gotchas
- **Retire, don't alias.** C has no `skills`/`spells` — they must be unknown commands, not hidden
  aliases (that's the faithful behavior + the whole point of retiring inventions).
- **Leading spaces in `how_good`** and the trailing space after it (empty mana `%s`) are fidelity.
- **Canonical state = `GetSkill`/`Practices`**, never SkillManager — the combat layer already trusts it.
- **Sort order is alpha-by-name**, boot-built (`spell_sort_info`); the display order must match or the
  catalog diverges as more skills unlock at higher levels.
- Don't touch the oracle; don't let the SkillManager cascade tempt a broad rewrite — scope to the spine.
