# Practice skill storage audit — 2026-10-25

Sai and Lucci reported learned ninja arts rendered as `(not learned)`.
The guild stored `SkillStorageName(num)` while `RenderSkillList` read the
lowercased display name. All 18 entries in `catalogSkillKeys` were affected.
C `list_skills` reads `GET_SKILL(ch, i)` (`src/spec_procs.c:192`); its
`how_good` bands are `src/spec_procs.c:108-131` (R1, R5c).

## Class audit

Repeat from the repository root:

```sh
rg -n 'SkillCatalogName|GetSpellName' pkg --glob '*.go' --glob '!**/*_test.go'
rg -n '\.(GetSkill|SetSkill)\(' pkg --glob '*.go' --glob '!**/*_test.go'
rg -n 'GetSkillLevel\(|LearnSkill\(|RegisterSkill\(' pkg --glob '*.go' --glob '!**/*_test.go'
```

Following local variables from those producers to the skill reads/writes found:

| Boundary | Finding / disposition |
| --- | --- |
| Practice rendering (`pkg/game/practice.go`) | Fixed: display remains `SkillCatalogName`, percentage reads `SkillStorageName`. Both standalone practice and guild practice call this renderer. |
| Lua bridge (`pkg/game/world_bridge.go`) | Fixed: numbered `set_skill` writes `SkillStorageName`. Real binding calls adapter `SetSkill`; C writes `SET_SKILL(ch, group_num, skill_lvl)` at `src/scripts.c:1376-1379`. |
| Cast (`pkg/session/cast_cmds.go`) | Uses `SkillStorageName` for consistency with C `GET_SKILL(ch, spellnum)` at `src/spell_parser.c:970`. Current `maxCastSpell=130` excludes all mapped entries (135–189), so this has no current behavior change. Existing cast scenario checks the reachable boundary; no mapped-skill casting claim. |
| Guild, remort, first-player God, skillset | Already use `SkillStorageName`. Corrected skillset's stale comment claiming display-key storage. |
| Save restoration | Already uses `canonicalSkillKey`; existing migration test covers old display keys and duplicate-key maximum. Save format unchanged. |
| Combat callbacks, command gates and improvement | Read/write handler constants or keys; combat's numbered mapping returns the matching handler constants. No additional display-name producer found. |
| Restore / generic SkillManager consumers | Use existing `skill.Name` (storage identity), separate from `skill.DisplayName`, or command arguments. No catalog-derived display key found. |
| Other catalog producers | Name resolution, affect descriptions, say-spell text, wizard confirmations/logs, and the catalog-to-storage helper itself; no skill-state access. |

## Proof and limits

- `TestRenderSkillListMappedSkills`: all 18 translations, every `how_good` band,
  literal C row spacing and CRLF.
- `TestRenderSkillListNinjaLearnedArts`: ninja strike, serpent, tiger and all nine
  kuji-kiri arts at 40% show `(poor)` together.
- `TestLuaSetSkillMappedSkills`: real Lua engine trains all 18 mappings, verifies
  handler-key readback and no duplicate display key, then exercises dragon's
  self-target refusal after the learned gate (`src/act.offensive.c:642-663`).
- R5h: separately restoring the old practice read and old Lua write makes their
  corresponding regression tests fail. Each fixed source was restored afterward.
- Targeted census `practice-storage`: CLEAN, 5/5, seed 1. Selection:
  `practice-mapped-ninja` (learned mapped ninja list before/after Lua, dragon
  initially unlearned then trained), `skills-practice` (standalone list and
  redirect), `guild-practice` (guild list/learn/list), `lua-set-skill-depth`
  (existing Lua boundary), `cast-detect-invisibility-spoken-depth` (cast lookup).
- Evidence: `/home/zach/Archives/darkpawns/oracle-runs/2026-10-25/practice-storage/`.
  Census retains a source snapshot, source hashes, base HEAD and working-tree status.
  Revert logs and `--show-oracle` output are in the sibling
  `practice-storage-controls/` directory.
- This closes the storage-key mismatch; it does not certify all practice depth.
  In particular, spell mana rendering is separate; the new ninja fixture is
  level 28, below its first spell, so every listed entry is a skill.

## Decisions for Zach

Proposed addition under R5c (not applied to any governing document):

> When a C skill/spell number crosses the Go storage boundary, every read and
> write uses `SkillStorageName(num)`. `SkillCatalogName` and `GetSpellName` supply
> display text, not storage keys. Prove a mapped entry through both setter and
> reader, including its command consumer.

Motivation: DP-1342 established catalog-to-handler translation, but the
Sai/Lucci practice report exposed a reader still using the display name and
Lua's numbered setter repeating the same mistake. The audit above covers the
class. Zach decides whether this exact text belongs in the rulebook (R5b).
