# Modernization Phase 4 — mechanical handler dedup handoff

Date: 2026-09-05
Branch: `glm/modernize-phase-4-1` (item 4.1 is first; merged before item 4.2)
Base: `origin/main` after PR #1388 merge (`9588f759567dc3b0e526aa2d1ddf43413071b85a`)

## Queue and process

Phase 4 is being executed as seven serial PRs, in roadmap order:

1. 4.1 clan family through `resolveClanForImmortal` (−250)
2. 4.2 skill-command prologue helpers (−275)
3. 4.3 small pairs (−180)
4. 4.4 parameterized channel wrapper (−58)
5. 4.5 table-driven position commands (−50)
6. 4.6 verbatim duplicates plus the `LVL_IMMORT` import-cycle fix (−85)
7. 4.7 script-trigger shared spine (−37)

No item N+1 branch will be created until item N's PR is merged. Each PR must
carry its changed-file list and named proven scenarios, a complete
`make oracle-regression` tally, standard local gates, and hosted CI evidence.
The RED set remains human-only.

## Item 4.1 coverage boundary

The clan family already has depth-proven scenarios in:

- `clan-depth.txt`
- `clan-member-depth.txt`
- `clan-applicant-depth.txt`
- `clan-plan-depth.txt`
- `clan-plan-mortal-depth.txt`
- `clan-rename-depth.txt`

The implementation must remain a pure refactor of the repeated clan context
selection in the handlers named by the roadmap. Any changed file without a
direct mapping to these clan rows is human-merge-only under the amendment.

### Item 4.1 changed-file map

- `pkg/game/clan_bank.go` — `clan.bank-immortal-path`, `clan.mortal-bank`,
  and `clan.mortal-money-gates` in `docs/fidelity/depth/clan.tsv`.
- `pkg/game/clan_economy.go` — `clan.bank-immortal-path`, `clan.set-money`,
  `clan.set-applev`, and their mortal counterparts in `clan.tsv`.
- `pkg/game/clan_settings.go` — `clan.set-ranks`, `clan.set-privilege`,
  `clan.admin-private`, `clan.plan-editor-prompt`,
  `clan.plan-editor-completion`, and the matching mortal rows in `clan.tsv`.

The verifying scenario files are `clan-depth`, `clan-member-depth`,
`clan-applicant-depth`, `clan-plan-depth`, `clan-plan-mortal-depth`, and
`clan-rename-depth`. The C-specific first-token lookup in `doClanSP` is kept
by adapting only the helper's lookup input; no command or message behavior is
intentionally changed.

Focused item-4.1 regression before the full corpus: 6/6 scenarios passed,
with 0 failed, 0 infra, and 0 timed out (107.402s).

## Item 4.1 verification

Standard gates passed on the branch:

- `/usr/local/go/bin/go build ./...`
- `/usr/local/go/bin/go vet ./...`
- `/usr/local/go/bin/go test ./...`
- `golangci-lint run ./...` — 0 issues
- `gofumpt -l .` — clean
- `git diff --check` — clean
- `/usr/local/go/bin/go test ./pkg/game/...` — pass
- `make fidelity-depth` — 4,760 cases; 4,654 proven/delegated, 55 blocked,
  51 excluded

The required full corpus then passed:

```text
oracle-regression: scenarios=934 passed=934 failed=0 infra=0 timed_out=0 elapsed=7306.563s started=2026-09-05T07:49:23-0400 finished=2026-09-05T09:51:10-0400
```

Eight infrastructure-shaped startup retries recovered on the runner's single
retry (`applaud-depth`, `bash-peaceful-depth`, `berserk-failure-depth`,
`bounce-depth`, `clan-depth`, `force-mob`, `love-depth`, and
`wizard-residual-depth`); all eight finished `PASS` on retry, so the final
authoritative tally is `infra=0` and `timed_out=0`.

PR #1390 (`glm/modernize-phase-4-1`) passed hosted lint, security, and test
checks and self-merged on 2026-09-05 as `f2aa2caccf3133f6216dc7660488a38335afd244`.
The next serial branch is `glm/modernize-phase-4-2`, based on that merge.

## Item 4.2 coverage boundary

Item 4.2 changes only `pkg/command/skill_commands.go` and introduces shared
nil-player, `CanUseSkill`, and `OneArgument` + `FindTargetInRoom` helpers. The
helper is applied only where the original C ordering and parser path are the
same; `headbutt`, `ambush`, and `groinrip` retain their special gate order.

The named verifying scenario subset is:

`ambush-depth`, `backstab-depth`, `bash-depth`, `bearhug-depth`,
`behead-object-depth`, `berserk-depth`, `bite-depth`, `carve-depth`,
`charge-depth`, `circle-depth`, `cutthroat-depth`, `dig-depth`,
`disarm-depth`, `disembowel-depth`, `dragon-depth`, `flesh-alter-depth`,
`groinrip-depth`, `headbutt-depth`, `hide-depth`, `mindlink-entry-depth`,
`mold-depth`, `neckbreak-depth`, `point-depth`, `rescue-roll`,
`review-depth`, `scrounge-default-depth`, `serpent-depth`,
`sharpen-depth`, `shoot-entry-depth`, `sleeper-depth`, `slug-depth`,
`smackheads-outcome-depth`, `spike-depth`, `steal-depth`, `stealth-reject`,
`thief-sneak`, `combat-bash-opener`, `combat-headbutt-opener`,
`combat-kick-opener`, and `combat-trip-opener`.

The focused item-4.2 regression passed 40/40 scenarios with 0 failed, 0 infra,
and 0 timed out (328.824s).

The required full corpus passed on this branch:

```text
oracle-regression: scenarios=934 passed=934 failed=0 infra=0 timed_out=0 elapsed=7087.597s started=2026-09-05T10:16:24-0400 finished=2026-09-05T12:14:32-0400
```

Six infrastructure-shaped retries recovered on the runner's single retry
(`escape-no-skill`, `mortal-batch21`, `pour-multi`, `rsay-noshout`,
`spec-proc-bank-kir-oshi`, and `spec-proc-castle-guard-up`); all six finished
`PASS` on retry, so the final authoritative tally is `infra=0` and
`timed_out=0`.

The standard gates also pass on this branch: `go build ./...`, `go vet ./...`,
`go test ./...`, `golangci-lint run ./...`, `gofumpt -l .`, `git diff --check`,
and the command/session package tests. The changed-file list is limited to
`pkg/command/skill_commands.go` and this handoff; the focused scenario mapping
above is the condition-three coverage citation for the item.

PR #1391 (`glm/modernize-phase-4-2`) passed hosted lint, security, and test
checks and self-merged on 2026-09-05 as
`30e17f0f1dc3ac403882164fa6097de28026318e`. The next serial branch is
`glm/modernize-phase-4-3`, based on that merge.

## Item 4.3 coverage boundary

Item 4.3 shares only the named small pairs: directed tell/reply message
filtering, sneak/stealth execution, SendToAll/SendToOutdoor delivery, room-mob
and room-item target disambiguation, the three signed-prefix parsers, spike/
stake dispatch, mail block read/write positioning, and Sprintbit/sprintnbit.
The changed-file list is:

`pkg/command/skill_commands.go`, `pkg/game/logging.go`, `pkg/game/mail.go`,
`pkg/game/mail_test.go`, `pkg/game/skill_stealth.go`,
`pkg/session/agent_vars.go`, `pkg/session/comm_cmds.go`,
`pkg/session/parse_helpers.go`, `pkg/session/session_manager.go`,
`pkg/session/session_test.go`, `pkg/session/wiz_movement.go`,
`pkg/session/wiz_player.go`, and `pkg/session/wiz_system.go`.

The named verifying coverage is:

- tell/reply: `comm-depth`, `reply-noarg`, `tell-linkless-depth`,
  `comm-soundproof`, `comm-notell`, `comm-tell-nodesc`, and `comm-noshout`;
- sneak/stealth: `sneak-mounted-depth`, `stealth-reject`, `thief-sneak`,
  `TestDoSneakTimedAffectAndReroll`, `TestDoSneakMountedGatePrecedesRoll`,
  and `TestDoSneakFailedRerollClearsSneakAndStealthAffects`;
- SendToAll/Outdoor: `TestManager_SendToAll` and
  `TestManager_SendToOutdoor`;
- buildRoomMobs/Items: `TestBuildRoomMobs`,
  `TestBuildRoomMobs_KeywordDisambiguation`, `TestBuildRoomItems`, and
  `TestBuildRoomItems_KeywordDisambiguation`;
- parse×3: `dc-depth`, `advance-depth`, `dig-depth`,
  `TestParseDCNumberMirrorsCAtoi`, and
  `TestParseDigRoomNumberMatchesAtoiPrefix`;
- spike/stake: `spike-depth`, `TestDoSpike_KillsWerewolf`,
  `TestDoStake_KillsVampire`, and the existing spike gate/state tests;
- mail read/write: `TestMailReadWriteSharedFilePositioning`;
- Sprintbit/sprintnbit: `idlist-depth`, `idlist-gate-depth`, and
  `TestIDListBitArrayUsesUndefinedForOutOfTableBits`.

The focused item-4.3 regression passed 22/22 scenarios with 0 failed, 0 infra,
and 0 timed out (149.769s). `tell-linkless-depth` retried once as an
infrastructure-shaped failure and passed on retry.

The required full corpus passed on this branch:

```text
oracle-regression: scenarios=934 passed=934 failed=0 infra=0 timed_out=0 elapsed=7091.014s started=2026-09-05T12:34:21-0400 finished=2026-09-05T14:32:32-0400
```

Six infrastructure-shaped retries recovered on the runner's single retry
(`abilities-depth`, `combat-trip-opener`, `communication`, `disembowel-depth`,
`imotd-depth`, and `yuball-depth`); all six finished `PASS` on retry, so the
final authoritative tally is `infra=0` and `timed_out=0`.

The standard gates also pass on this branch: `go build ./...`, `go vet ./...`,
`go test ./...`, `golangci-lint run ./...`, `gofumpt -l .`, and
`git diff --check`.

PR #1392 (`glm/modernize-phase-4-3`) passed hosted lint, security, and test
checks and self-merged on 2026-09-05 as
`6f43267dfeac97684866e3169b1949df4ef4c9ef`. The next serial branch is
`glm/modernize-phase-4-4`, based on that merge.

## Standing fidelity constraints

Apply R1 (player-facing bytes), R3 (draw/order parity), R4 (no invention), and
R5e (verify the reachable call path). Do not edit `src/` or
`darkpawns-c-oracle/`, change save format, touch the RED set, or broaden the
scope beyond the numbered item.
