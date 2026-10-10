# Dead-code decisions (2026-10-09)

Companion to PR A (dead twin deletion). Everything not deleted in PR A is
classified here for ruling. Ground truth: `deadcode -test ./cmd/... ./tools/...`
at `c46dbf89a` (238 funcs) plus a fresh **whole-module** run
(`deadcode -test ./...`, 108 funcs — see the finding below).

## Finding: the conservative deadcode invocation is too conservative

`deadcode -test ./cmd/... ./tools/...` counts only the root packages' tests
as reachable. Functions used **by their own package's tests** (session,
game, …) were falsely listed — PR A initially deleted several and broke
live regression tests (weather characterization, heartbeat output, the
vuln-042 input-queue cap tests, spec smoke tests). All were restored and
marked class (d). **Future sweeps must run `deadcode -test ./...`**; the
count that matters is 108, not 238.

## Class (b) suspects — ported C behaviour that may be missing (do not delete)

| Item | Evidence | Needed |
|---|---|---|
| `scan`, `skills`, `learn`, `forget` commands (`command.CmdScan/CmdSkills/CmdLearn/CmdForget/CmdConfirmForget/CmdUseSkill/CmdSkillInfo/CmdListSkills`) | **Not in C's command table** — `src/interpreter.c` has no entries for any of them (verified 2026-10-09: `rg '"scan"|"skills"|"learn"|"forget"' src/interpreter.c` is empty). Go-only invented commands that were never wired. | Class **(c)**, Zach decides (wire or delete). Not missing C behaviour |
| `game.ParseClass`, `game.ParseRace` | Every **live** C call path has a live Go equivalent: creation menus and wiz set parse through `game.ClassAbbrevs` (char_creation.go:390, wiz_system set). C's third caller (`act.informative.c:2706`, `skills <class-abbrev>`) sits inside C's own unwired `skills` command. | Dead twins of the `ClassAbbrevs`-based live parsing — class (a) follow-up deletions, not (b) |
| `spells.MagAttackModifier` | Comment cites `src/magic.c mag_attack_modifier()` — **no such function exists in `src/`** | Classification: Go-only invention mislabelled as a port, or the C name drifted |
| `game.HouseCanEnter` | Sole Go implementation of C `house_can_enter`; no C call site found in this fork's `src/` | Confirm whether C's house enter gate is dead in C too, then delete or wire |
| `damage_stubs.go` quintet (`World.doDamage`, `getAttackerName`, `World.executeCommand`, `World.doForced`, `diceRoll`) | Real logic (DP-901/DP-1025 funnel routing), unreachable. Live damage/dispatch/force paths exist elsewhere. | Diff against the live funnels; if twins, delete in a follow-up |

Restored as class (d) test seams instead of deleted (live twins verified):
`game.DoHide` (twin `DoHideInWorld` via `command.CmdHide`; the stealth
mechanics tests drive it), the weather characterization surface
(`AnotherHour`, `WeatherChange`, the six event wrappers,
`weatherWorldSnapshot`, `GetMoon`, `ModifyWeatherChange`),
`session.enqueueInput`/`queueLen` (vuln-042 cap tests),
`spec_assign.AllSpecNames` (spec smoke tests), `dprng` draw-log hooks.
`InitializeWeather` was repointable: its tests now call the live
`ResetTime()`.

## Class (c) — Go-only, never wired (Zach rules: wire or delete)

- `pkg/command/admin_commands.go`: the whole parallel moderation layer
  (24 funcs incl. `NewAdminCommands`, `RegisterCommands`, `parseDuration`) —
  never instantiated; `wiz_system.go:744` says so itself. Overlaps live
  session moderation (`mute` at `commands.go:274`, kick/ban in wiz commands)
  and `pkg/moderation`.
- `pkg/command` registry + middleware: `Registry.Use`, `Registry.Execute`,
  `LoggingMiddleware`, `WhitelistMiddleware` — an unwired alternative command
  framework; the live one is session `registerCommand`.
- `web/`: `CORSMiddleware`, `LoggingMiddleware`, `WhitelistMiddleware`,
  `getAllowedOrigins`, `isDevMode`, `peerIsLocal`, `isOriginAllowed`,
  `CSPNonceFromContext` — security-relevant; report whether the protections
  exist some other way before deleting.
- `pkg/audit`: `Init`, `Close`, `LogLoginAttempt`, `LogAdminAction` —
  security-relevant, Go-only (the live audit is `audit.AuditLogger` wired in
  `main.go`).
- `pkg/metrics`: `ConnectionError`, `ErrorOccurred`, `DBQuery`.
- `pkg/validation`: `IsValidPlayerName`, `SanitizePlayerName`.
- `pkg/combat/roller.go` cluster (10): `SetRoller`, `WithRoller`,
  `NewScriptedRoller` + 4 methods, `NewSeededRoller` + 3 methods — an unused
  dice-injection seam (no test uses it either). `randPick` likewise.
- `pkg/game` leftovers: `SetLogWriter` (wiring question: C's `basic_mud_log`
  writes a log file; the Go writer is never set — flag or wire), `Alogf`,
  `CoreDump`, `ItemType.IsReadable/IsContainer/IsWeapon/IsArmor/IsFood`,
  `LocShop`, `objectVisibleFlags`, `World.listObjToChar`, `AsActor`,
  `mobHasLight`, `UpdateObject`, `DoStart`, `Spawner.findObjectInstance`,
  `IsDonationRoom`, `World.roomMessageExcludeTwo`.
- `pkg/game` legacy save layer: `save.go` (`encodeSave`,
  `saveDataToPlayer`, `SerializePlayer`, `DeserializePlayer`, `sanitizeName`)
  and `serialize.go` (`SerializeInventory`, `SerializeEquipment`) — superseded
  by the SQLite path (`db.SavePlayer`, `persistence_seam.SavePlayerRecord`)
  but **entangled with live test scaffolding** (`legacy_save_test.go` helpers
  and the carry-weight / save-world / home-load tests ride it). Migrating
  those tests to the db path is a prerequisite for deletion.
- `pkg/session` leftovers: `cmdZap`, `cmdAutoLoot`, `broadcastCombatMsg`,
  `renderForBrowserTerminal`, `startCharCreation`, `isUniqueConstraintError`,
  `objectivePronoun`, `parseCastArguments`, `resolveCastTarget`,
  `executeCommand`, `resolveDirection`, `cmdBroadcast` — each needs its live
  twin named before deletion (the live flee/qcomm/gmcp/queue twins were
  verified and the dead copies deleted in PR A).
- `session cmdQcomm` (quest-channel variant, deleted in PR A? **no** —
  restored decision): the *deleted* quest-qcomm was Go-only and unwired;
  C's `do_qcomm` (`act.comm.c:1301`, immortal qecho) is live as
  `World.ExecQcomm`. The quest variant and its test are gone; if the quest
  channel is wanted, that is a new feature decision.
- `engine.BasicMudLog`: compile dependency of `engine.MudLog` (0 callers),
  which the mudlog-push track deletes by design; it goes with that PR.
- Orphan packages (nothing imports them): `pkg/privacy` (1,852 lines; earlier
  review: a client with no server whose fallback destroys logs), 
  `pkg/optimization` (4,020), `pkg/secrets` (318), `profiling` (801),
  `examples` + `examples/performance` (489), `benchmarks` (329),
  `load_test` (459) — ≈8,300 lines; per the brief, Zach rules and current
  docs mentioning them are updated in the same PR.
- `pkg/events` (10 unpublished types): assigned to Claude's filing batch
  (mudlog-push design §8).

## Tests deleted with their dead functions (assertions that rode only dead
code): `TestCmdPractice`, `TestCmdReview_NoPlayer`, `TestCmdWhois_No*`,
`TestCmdDig_*`, `TestCmdSummon`, `TestFidelityDigCosmeticStub`,
`TestSaveLoadLocationRoundTrip`, `TestCmdQcomm_NonQuestPlayerFiltered`
(Go-only quest channel; C's do_qcomm is live as `World.ExecQcomm`),
`TestCmdFleeMovement_XPLossAtLowLevel` (canonical live flee tests cover
the XP paths), `TestValidNameOnlineDuplicate` (merge_bridge free
function; live name rules are the db unique constraint + BanManager).

Repointed, not deleted: `TestCanonicalFightMessagesData` and
`loadMessagesFile` → `ParseFightMessages`; the Dice tests → `pkg/dprng`;
the GMCP tests → `gmcpRoomInfoForPlayer`; `TestDoSimpleMove_ClosedDoor`
and the follower-move tests → `performMoveResult`; the weather init test
→ `ResetTime`; `NewMobInstance` call sites → `NewMob`. The weather
characterization, heartbeat, spec, stealth and input-queue-cap suites
are intact — an earlier automated pass over-deleted them and they were
restored from the base commit before these commits were finalized.

## Orphan packages — ruled 2026-10-10 (Zach): DELETE ALL SEVEN

Executed in this PR, one commit per package, pure deletion, references
swept (Makefile privacy-test target, current docs, C_FUNCTIONS.json's
two dangling go_location entries — both had mapped C functions into
Go-only invented packages and were already wrong).

Nothing in `cmd/` or `tools/` imported any of these (~8,300 lines
deleted). The table keeps each package's doc-mention map as the record
of what was updated:

| Package | Lines | Current docs mentioning it | Note |
|---|---|---|---|
| `pkg/privacy` | 1,852 | docs/operational/privacy-filter.md, docs/architecture/ARCHITECTURE.md | Earlier review: a client with no server; its fallback destroys logs |
| `pkg/optimization` | 4,020 | docs/architecture/port-status.md, docs/architecture/ARCHITECTURE.md | |
| `pkg/secrets` | 318 | docs/architecture/ARCHITECTURE.md, docs/operational/SECURITY_HARDENING_GUIDE.md, docs/research/RESEARCH-LOG.md | Overlaps the env-by-name decision (mudlog-push §12.1) |
| `profiling` | 801 | profiling/README.md, docs/research/RESEARCH-LOG.md | |
| `examples` + `examples/performance` | 489 | (website content mentions "examples" generically; no path references) | |
| `benchmarks` | 329 | website-astro/PRODUCT.md, website-astro/src/content/blog/the-long-middle.md (prose, not paths) | |
| `load_test` | 459 | docs/research/RESEARCH-LOG.md | |

Rulings land as deletions in this PR (stop tier); each class (b) item
becomes a Linear issue filed by Claude.
