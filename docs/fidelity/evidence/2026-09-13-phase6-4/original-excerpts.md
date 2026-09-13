# Phase 6.4 original-source excerpts — 2026-09-13

This file preserves the relevant original audit text verbatim. It is
provenance, not a current diagnosis. The Phase 6.4 handoff reconciles each
excerpt against current Go and C call paths under R5.

## Recovered source identity

The original reports were recovered from `/home/zach/dp-modernization`.
Their source clone identified itself as `/home/zach/dp-modernization/darkpawns`,
HEAD `0d6a9d1ff`, dated 2026-09-04. SHA-256 hashes of the four inputs used for
this audit are:

| report | SHA-256 |
|---|---|
| `reports/06-roadmap.md` | `5a167fcdf2e7a3183cc7e84c14e75742e7d7d09bc52780be3da35827a50497c6` |
| `reports/02-c-isms.md` | `d2ea1a557e6870a1cebfe13bca6a72c2b10a708b6057008a9b7754c61ecccf2c` |
| `reports/raw/cisms-structural.md` | `49feac1bb347fc8b91612b7d668a2ff51445346f84474ce5451d74dcc452acc5` |
| `reports/05-risk-tiers.md` | `9c08c536a54abe985cae7f16f14b1e4cf0537d50b5e789f6208a5913b2808a80` |

## `reports/02-c-isms.md:10-20`

```text
## 1. Global mutable state — largely SOLVED

345 package-level `var` declarations, but only **31 true mutable state globals**; the ROM world tables live in `type World struct` (world.go:44, mutex-guarded with a documented lock-ordering law, manager.go:1073). 259 declarations are inert lookup tables (31 exported constants in game/constants.go).

Remaining residue (all worth a cleanup bite, none load-bearing):
- `pkg/game/mail.go` — 8 globals (mailIndex, freeList, 2 function hooks)
- `pkg/game/merge_bridge.go:15` — `banManager`, shadows `World.Bans`
- `pkg/game/weather.go:171` — `weatherWorld`
- `pkg/game/spec_assign.go:403,408` — spec-proc registries
- `pkg/combat/roller.go:36-38` — global Roller singleton (test seam; production default is sacred)
- `pkg/dprng/cmwc.go:92` — `streamMu` (determinism-critical, DO NOT touch)
```

The historical “8 globals” label is retained as written. Current-source
inspection finds eight declarations in the `mail.go` state block, but the
structural table below enumerates only the state/hooks it classified as
residue; the current audit inventories the additional file-position, lock, and
composition-map state explicitly.

## `reports/raw/cisms-structural.md:29-42`

```text
### (a) World/state globals — exhaustive table

| File:line | Var | Type | Guard | Notes |
|---|---|---|---|---|
| `pkg/game/mail.go:72` | `mailIndex` | `*MailIndex` | `mailGlobalMu` (:81) | mail subsystem state outside World |
| `pkg/game/mail.go:73` | `freeList` | `*MailPosList` | same | |
| `pkg/game/mail.go:75-76` | `worldNameFunc`, `worldIDFunc` | func hooks | none | cross-package wiring globals |
| `pkg/game/limits.go:69` | `fieldObjs` | `[]FieldObject` | none | mutable world-adjacent list |
| `pkg/game/merge_bridge.go:15` | `banManager` | `*BanManager` | none (nil-checked lazy init) | **shadows** `World.Bans` field (`world.go:111`) — two ban managers |
| `pkg/game/merge_bridge.go:35` | `HasActiveCharacter` | func hook | none | set by session pkg |
| `pkg/game/houses.go:64,71` | `getPlayerNameByID`, `getPlayerIDByName` | func hooks | none | |
| `pkg/game/weather.go:171` | `weatherWorld` | `*World` | `weatherMu` (:170) | back-pointer global; events reach World only through it |
| `pkg/game/spec_assign.go:403` | `SpecRegistry` | `map[string]SpecFunc` | none | mutable, written at :412 by `RegisterSpec` |
| `pkg/game/spec_assign.go:408` | `ObjSpecRegistry` | `map[string]ObjSpecFunc` | none | written at :418 |
```

The current audit checked this historical table against current source and
did not assume its notes (“none load-bearing”) were still true.

## `reports/05-risk-tiers.md:23-35` and relevant RED boundary

```text
## YELLOW — partial cover (write oracle cases first, then bite)

| Zone | Coverage state |
|---|---|
| `pkg/session/commands.go` (dispatcher, 26 units) | 18 proven / 7 partial — the per-command `number(0,3)` draw at :628 is inside proven paths, but the partial 7 (incl. OLC-adjacent fallthrough) gate any dispatch restructure |
| `pkg/command/skill_commands.go` (2,421 L) | 45 proven / 2 partial / 9 breadth-only — prologue-dedup bite (reports/03 #3) is safe for the 45; write cases for the 11 first |
| `pkg/session/combat_cmds.go`, `cast_cmds.go` | 2/6 and 0/2 fully proven — combat entry points; the `shoot` blocked cluster (9 cases) gates CmdShoot specifically |
| `pkg/spells/*` (9,014 L incl. affect_spells.go 3,473) | NO direct units — indirect via cast/will/objmagic depth cases + golden tests. Spell-dispatch consolidation (≥6 fragmented `switch spellNum` blocks) is YELLOW: player bytes + dice order inside dispatch are unproven per-spell |
| `pkg/game/socials.go` + `act_social.go` | 172/187 socials proven; **12 bare + roll/snowball need scenarios before the loader-ization bite** (reports/03) — cheap to fix, do it |
| `pkg/game/spec_procs*.go` (6,373 L) | 241 scenario-proven / 243 unit / 7 blocked / 26 excluded — dedup per family (reports/03 #2) only where its spec-proc's cases are green |
| `pkg/session/char_creation.go:139-478` (FSM menus) | 42-case stage switch; proven via God-fixture scenarios but the FSM restructure is YELLOW (creation bytes interleave with tick draws) |
| `pkg/session/wiz_set.go` (51-case switch) | `set` command proven; table-driven conversion mechanical — GREEN for the toggle majority, YELLOW for fields touching blocked surfaces (rent/player reports) |
| `pkg/scripting/*` (3,476 L) | Zero oracle reach (C oracle's Lua headers not even in tree). Unit tests only. Untouchable for behavior; mechanical cleanup only. |
```

The original risk report also places the relevant weather/tick and boot-order
surfaces in its RED section:

```text
| Boot ordering `cmd/server/main.go:81-187` | — | ResetTime pressure draw (#1, :169) → ParseWorld mob-stat draws (parser/mob.go:405) → boot zone-reset draws (spawner.go) → mob HP/gold draws (mob.go:104-136). Adding even an "innocuous" boot-time draw shifts every subsequent roll. |
| `pkg/game/spawner.go` + `pkg/game/mobact.go` + `pkg/game/weather.go` | ~2,000 | Fixed per-tick draw chains (mobact DP-1170 chain, weather 5-draw chain + branch draws). Sentinel/sound draws must burn even when behaviorally inert — that hoisting was a whole bug fix. |
```

## `reports/06-roadmap.md:69-77`

```text
## Phase 6 — C-isms mechanical wave (~−2,300; subset of reports/02)

| # | Target | Δ lines | Risk | Verifying cases |
|---|---|---|---|---|
| 6.1 | Giant switches → data tables where mechanical: `wiz_set` toggle majority (51 cases), `findExp` class/level ladders, equipment slot↔name maps, small lookup switches | −1,200–1,800 (mechanical subset only) | GREEN for the listed ones; spell-dispatch consolidation is **NOT here** | affected proven units |
| 6.2 | String cleanup in proven files: 94 nested Sprintf → flatten; 57 loop-concats → Builder | −200–400 | GREEN | per-file unit scenarios |
| 6.3 | Keyed tables + constants: key the unkeyed data tables (spellDB 509 L, THAC0, saving-throws 1,943 L), name the literal ladders, single-source LVL_IMMORT | ~0 net (churn) | GREEN | unit tests; **highest bug-class value per line in the census** |
| 6.4 | Global→struct injection: mail.go (8 globals), weather.go, merge_bridge banManager, spec_assign registries | −250–400 | YELLOW (behavior-adjacent) | weather/mail/ban scenarios + unit tests |
| 6.5 | Production `_ =` → handle-or-slog (AGENTS.md:63) | ~0 (adds lines) | GREEN | build + vet |
```

The `−250–400` estimate and the original YELLOW label are historical planning
metadata. They are not current line savings, authorization, or proof.
