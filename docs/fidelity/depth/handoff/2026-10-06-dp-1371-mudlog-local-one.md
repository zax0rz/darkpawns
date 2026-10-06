# D7 first grouped local producer train

**Local (mudlog-only).** Production edits only insert MudLog calls or replace
one slog call, with the fmt import for banking and correction of a false help
comment. No routing, Act/delivery, signatures, mutation order or error handling
changes. Fresh main `3ccb7de31`. Five cases, one commit per case. The complete
207-active-site inventory and train map are in #1819; the aggregate D7 row stays
blocked. This is producer parity, not whole-handler parity.

## Cases and C boundaries

| Case / test | C | Contract and existing Go boundary |
|---|---|---|
| `TestMudlogLocalOeditDefault` | `src/oedit.c:1545-1546` | Exact default diagnostic, BRF/31/file TRUE, before dirty flag/menu; replaces slog |
| `TestMudlogLocalSeditDefault` | `src/sedit.c:1165-1167` | Exact default diagnostic, BRF/31/file TRUE, after existing cleanup |
| `TestMudlogLocalHelp` | `src/act.informative.c:1608-1610` | HELP payload, NRM/31/file TRUE, after miss acknowledgement |
| `TestMudlogLocalClanWithdraw` | `src/clan.c:963-966` | withdraw payload, BRF/31/file TRUE, after mutation/ack and before saves |
| `TestMudlogLocalClanDeposit` | `src/clan.c:970-977` | adds payload, BRF/31/file TRUE, after mutation/ack and before saves |

Every test reaches the actual handler and registered provider, observes exact
un-normalized observer/file bytes and the state at delivery, checks minimum-1
and type-1 recipients stay silent, and uses an invis40 actor to catch an invented
invisibility threshold. Editor defaults deliberately inject impossible editor
modes: these are defensive unit proofs, not reachable-play or oracle claims.
Help's misc/help append and hidden wizonly branch are not changed or claimed.
Banking uses C HSHR's female `her`, not a clan name.

`mudlog-local-depth` stages an independent level34 complete-syslog observer,
help miss and both actual successful bank commands. Funding is an actual
warmup deposit. The first draft incorrectly used `clan set money`, which did
not fund the clan; retained C output showed the poverty refusal. That draft is
not the proof. Five seeds are claimed only after the combined census re-proves
the final vehicle. Actor and audience bytes both remain part of comparison.

## Held locks and delivery audit

| Producer | Live path | Locks held at producer |
|---|---|---|
| OEDIT default | editor input -> `handleOeditInput` -> `parseOeditLocked` | session `textEditMu` |
| SEDIT default | editor input -> `handleSeditInput` -> `parseSeditLocked` -> existing `finishSeditLocked(false)` | session `textEditMu`; cleanup's writing/player/world/manager operations have returned |
| Help miss | registered help command -> `cmdHelpText` | none; this argument branch does not hold the no-argument text-cache lock |
| Clan withdraw | registered clan -> `ExecClanCommand` -> `doClanBank` | none; clan lookup read lock, player setters/getters and acknowledgement delivery have returned; saves occur later |
| Clan deposit | same | none, for the same reason |

MudLog's file write completes before provider traversal. EachSession snapshots
under manager.mu and releases it before recipient visits; recipient player reads
release p.mu. Player.SendMessage reads/releases player and world locks before
calling the manager sink. Attached-body lookup releases manager.mu before
session send; heartbeat staging, snoop handling and sendMu do not acquire
textEditMu. Neither MudLog nor its live delivery calls editor/cache locks. Thus
none of the held editor locks can form a delivery -> editor reverse edge. No
world, player, clan, manager or lifecycle lock is held by these five producers.

Inspected: `pkg/game/logging.go`, `player_affects.go`, `pkg/session/manager.go`,
`switch_ownership.go`, `session_send.go`, `heartbeat_output.go`, editor input/finish paths, command dispatch,
`clan.go`, `clan_bank.go` and ClanManager lookups. Recheck source paths with:

```sh
rg -n 'MudLog|EachSession|MessageSink|bodySession|sendTextMessage|stageHeartbeat|forwardSnoop' pkg/game/logging.go pkg/game/player_affects.go pkg/session
rg -n 'textEditMu|finishSeditLocked|parseOeditLocked|parseSeditLocked|doClanBank|FindClan' pkg/session/oedit.go pkg/session/sedit.go pkg/game/clan* pkg/session/commands.go
```

## Reproduce the per-case R5h controls

On a disposable worktree at the PR tip (the script always restores source):

```sh
go test ./pkg/session -run '^TestMudlogLocal' -count=1
python3 docs/fidelity/depth/handoff/2026-10-06-dp-1371-mudlog-local-one-controls.py --output "$HOME/Archives/darkpawns/oracle-runs/2026-10-06/local-one-rerun"
go test -race ./pkg/session -run '^TestMudlogLocal' -count=1
```

Each control removes exactly its producer line, requires the named test's
assertion failure and rejects build failures, then restores and re-proves it.
Original pre-fix baseline failed all eight original candidate tests on missing
producers, not compile errors; only these five are this train's cases.
Evidence root: `~/Archives/darkpawns/oracle-runs/2026-10-06/dp-1371-mudlog-local-one-proofs/`.
Per-case committed-tree gates and HEAD files live under `commit-<case>/`.

## Separated findings, not silent scope expansion

Three original local-01 cases are parked, with their diffs/tests/vehicle under
`parked/` in the evidence root and paired failures in `oracle-1-fixed.txt`:

- File-editor save/delete (`src/file-edit.c:44,57`) print C's logical
  `scripts/mob/dog.lua`; using existing Go state.path prints an absolute
  disposable world path. No absolute-path normalizer was added. The shared
  tedit/Lua storage/display path needs a separate train and class audit.
- Shared report producer (`src/act.other.c:1120`): reference C reports
  `retained $$ bytes` while the Go site's already-collapsed argument reports
  `retained $ bytes`. C process_input (`src/comm.c:1975-1977`) first doubles each input `$`;
  do_gen_write then collapses those pairs (`src/act.other.c:1112`). Thus
  raw `$$` reaches this producer as `$$`. Go's site collapses the raw
  argument without that input escape step. The observed mismatch is explained
  by this input/handler seam, not an oracle anomaly. Repair and class-audit it
  in its own train; do not alter routing or preprocessing in this local train.

No pins, dropped seeds, expected-divergence entries, oracle edits or guessed
payload workarounds. These findings are not resolved by this five-case train.
