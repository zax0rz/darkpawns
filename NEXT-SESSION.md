---
tags: [active, prompt, next-session]
---

# Session Kickoff: Dark Pawns — Lint Cleanup + Dead Code Removal

_Read SOUL.md, USER.md, AGENTS.md, HEARTBEAT.md. Read today's daily note from ~/brain/inbox/. Query GBrain for "dark pawns wave16" to get full context._

## Context

Wave 16 lint/QA pipeline is 88% complete. 6 commits landed this session reducing findings from **3,463 → 406** (394 excluding intentional TODO-placeholders and false positives). The golangci-lint config was migrated to v2 with extensive exclusions for game-code patterns.

Full Wave 16 state is in GBrain page `darkpawns-wave16` and the research log at `darkpawns/RESEARCH-LOG.md`.

## Immediate Tasks

### 1. Dead Code Removal (388 `unused` findings)

The bulk of remaining findings are **dead code from the game→session migration**. Old C-ported functions in `pkg/game/` that have modern replacements in `pkg/session/`:

**Biggest offenders:**
- `pkg/game/act_comm.go` — 30+ unused functions/consts: `doSay`, `doGSay`, `doTell`, `doReply`, `doWrite`, `doPage`, `subcmdHoller` through `subcmdAsk`, `undercommonSyllables`, `drunkSyllables`, `speakUndercommon`, `speakDrunk`, `twoArguments`, `getCharVis`, `deleteAnsiControls`, `lastTellersData`, `getLastTellers`, `initLastTellers`, `setLastTeller`, `getLastTeller`
- `pkg/game/act_informative.go` — 15+ unused: `dirList`, `doLook`, `lookAtRoom`, `listObjToChar`, `listCharToChar`, `listOneChar`, `lookAtTarget`, `lookAtChar`, `lookInDirection`, `lookInObj`, `doAutoExits`, `doExits`, `getExitForDirection`, `showObjToChar`, `showObjExamine`, `getObjectExtraFlags`, `doScore`
- `pkg/session/display_cmds.go` — `cmdLines`, `cmdInfoBar`, `cmdInfoBarOn`, `cmdInfoBarOff`, `cmdInfoBarUpdate`
- `pkg/session/movement_cmds.go` — `cmdFleeMovement`, `cmdFollowMovement`, `cmdSneak`
- `pkg/session/info_cmds.go` — `className`, `positionName`, `conditionLabel`, `cmdInfo`
- `pkg/session/spell_level.go` — `spellLearnEntry` type, `classSpells` var
- `pkg/session/tattoo.go` — `useTattoo`, `tattooAf`, `applyModifier`
- `pkg/session/time_weather.go` — `timePeriods` var
- `pkg/session/manager.go` — unused struct fields: `charSex`, `charRace`, `charClass`, `charHometown`, `charStats`, `screenSize`, `infobarMode`
- `pkg/session/wizard_cmds.go` — `cmdBroadcast`
- `pkg/spells/call_magic.go` — `applyDamageWithSave`
- `pkg/spells/spell_info.go` — `setupSpellInfo`, `setSpellLevel`
- `pkg/combat/fight_core.go` — `replaceString`
- `pkg/combat/formulas.go` — `backstabMult`
- `pkg/command/registry.go` — `buildChain`

**Strategy:** Verify each function has NO callers (grep the codebase), then delete in batches. Group by file. Build after each batch. Commit per logical group.

### 2. Remaining ineffassign (10 in `level.go`)

`pkg/game/level.go` has 9 instances of `wis` computed but unused (practice system not implemented) and 1 `addMove`. These are TODO placeholders. Options:
- **Option A:** Exclude the file from ineffassign in `.golangci.yml` until practice system is implemented
- **Option B:** Restructure the function to avoid the dead computation (larger refactor)
- **Recommendation:** Option A — add exclusion for `level.go` ineffassign

### 3. Exclude final false positives

Add to `.golangci.yml`:
- `pkg/spells/say_spell.go` misspell `ect` → already excluded but may need adjustment for v2 syntax
- `pkg/privacy/client.go:116` errcheck `io.Copy(io.Discard)` → add to `exclude-functions` or per-file rule
- `pkg/telnet/listener.go:334` errcheck `ReadByte` → add per-file rule

### 4. Final QA Pass

After dead code removal:
1. `golangci-lint run ./...` — confirm near-zero findings
2. `go build ./...` — build must pass
3. `go vet ./...` — vet must pass
4. Consider running `go test ./...` if tests exist

### 5. Optional: QA Code Review

Zach suggested using specialized models for a final QA pass:
- **GLM-5.1 + Kimi K2.6** for end-to-end review (functionality, idiomatic Go, security)
- **Opus or GPT-5.5** for deep architectural review
- Spawn as subagents with clear scope: review modified files from this session

## Files Modified This Session (6 commits)

```
a8ed760 fix: gofmt + remaining errcheck/staticcheck/gocritic fixes
3b49e52 refactor: errcheck sweep — 18 unchecked returns fixed
e1c4615 refactor: staticcheck sweep — S1039, SA9003, QF1001, QF1003, QF1007
85d1379 refactor: gocritic + staticcheck cleanups (K2.6 partial + manual)
3b5ff55 refactor: fix gocritic findings (octalLiteral, emptyStringTest, dupBranchBody, etc.)
02b84f2 refactor: fix lint findings — gofmt, misspell, ineffassign, QF1012
```

## Key Config: `.golangci.yml`

Migrated to **v2** format. Important exclusions:
- `examples/`, `benchmarks/`, `_test.go` — excluded from most linters
- `captLocal`, `unnamedResult`, `ifElseChain`, `commentedOutCode`, `nestingReduce`, `typeAssertChain` — noisy gocritic rules excluded
- `SA4000`, `SA4004`, `QF1008` — intentional patterns excluded
- `say_spell.go` — misspell `ect` excluded (spell fragment, not typo)

## Constraints

- **Do NOT change game logic** — behavior-preserving modernization only
- **Build must pass** after every batch of changes
- `fmt.Fprintf` to `player.Writer` is MUD output, NOT slog candidates
- Social message strings were intentionally fixed (quietly, stretches, etc.)
- Telnet `listener.go`: the `Conn` embedding is intentional (QF1008 excluded)

## Workspace

```
darkpawns/    — Go MUD codebase
darkpawns/.golangci.yml — lint config (v2)
darkpawns/RESEARCH-LOG.md — living research log
```
