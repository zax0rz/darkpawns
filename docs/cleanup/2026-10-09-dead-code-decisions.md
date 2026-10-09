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
| `scan`, `skills`, `learn`, `forget` commands (`command.CmdScan/CmdSkills/CmdLearn/CmdForget/CmdConfirmForget/CmdUseSkill/CmdSkillInfo/CmdListSkills`) | No live registration found for any of them (`practice` and `review` are the registered neighbours). C has them. | R5g pass: C callers → Go owners. Either live twins exist under other names, or these are missing player commands |
| `game.ParseClass`, `game.ParseRace` | C calls both: `interpreter.c:2082` (class selection), `act.informative.c:2706`, `act.wizard.c:2935/3006` (set class/race). No live Go twin found. | Check the char-creation and wiz-set owners; possibly missing |
| `spells.MagAttackModifier` | Comment cites `src/magic.c mag_attack_modifier()` — **no such function exists in `src/`** | Classification: Go-only invention mislabelled as a port, or the C name drifted |
| `game.HouseCanEnter` | Sole Go implementation of C `house_can_enter`; no C call site found in this fork's `src/` | Confirm whether C's house enter gate is dead in C too, then delete or wire |
| `damage_stubs.go` quintet (`World.doDamage`, `getAttackerName`, `World.executeCommand`, `World.doForced`, `diceRoll`) | Real logic (DP-901/DP-1025 funnel routing), unreachable. Live damage/dispatch/force paths exist elsewhere. | Diff against the live funnels; if twins, delete in a follow-up |
| `game.DoHide` skill logic | Deleted with a test asserting dex-bonus/toggle/improve; live path is `command.CmdHide`. | Confirm `CmdHide`'s tests cover the same skill mechanics; if not, that coverage moved with the deletion |

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

## Tests deleted with their dead functions (assertions that rode only dead code)

`TestPlayerSerializationRoundTrip`, `TestEncodeSave_*`, `TestSanitizeName`
(restored with the save cluster), `TestCmdPractice`, `TestCmdReview_*`,
`TestCmdWhois_No*`, `TestCmdDig_*`, `TestCmdSummon`, `TestCmdSkills`-none
(kept; `CmdSkills` not deleted), `TestFidelityDigCosmeticStub`,
`TestSaveLoadLocationRoundTrip`, `TestCmdQcomm_NonQuestPlayerFiltered`,
`TestCmdFleeMovement_XPLossAtLowLevel` (canonical live flee tests cover the
XP paths), `TestDoHideDexBonusToggleAndImprove`, `TestValidNameOnlineDuplicate`.

Repointed, not deleted: `TestCanonicalFightMessagesData` and
`loadMessagesFile` → `ParseFightMessages`; the Dice tests → `pkg/dprng`;
`TestGMCPOffUntilNegotiated`/GMCP gating tests → `gmcpRoomInfoForPlayer`;
`TestDoSimpleMove_ClosedDoor` → `performMoveResult`; `NewMobInstance` test
call sites → `NewMob`.
