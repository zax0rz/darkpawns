# Depth-fidelity handoff — `reload`

Date: 2026-09-01

## Queue position

This round began from refreshed `main` after `git pull --ff-only`, a
successful `make fidelity-depth`, and rereading `docs/fidelity/DEPTH_TESTING.md`
plus the latest `report` handoff. The special-procedure inventory remains
exhausted, the one blocked row `objmagic.sleep-entry-gates` remains queued after
its single cast-sleep vehicle, and the interpreter sweep advanced from `report`
to `reload`.

The next unmanifested interpreter family is `recharge` at
`src/interpreter.c:639`. `recite` is already owned by the recite/object-magic
manifests; `recall` and `receive` remain queued after `recharge`. The claimed
`qecho` family and its open feature and handoff PRs must not be repicked. The
open purge and plot handoff PRs likewise remain unmerged because their checks
did not fire after their permitted retries.

Frontier before this slice: 2,943 total; 2,870 proven/delegated; 22 blocked;
51 excluded.

Frontier after this slice: 2,949 total; 2,876 proven/delegated; 22 blocked;
51 excluded.

## C call path and behavior surface

The command-table entry is:

```c
/* src/interpreter.c:638 */
{ "reload"   , POS_DEAD    , do_reboot   , LVL_IMPL-1, 0 },
```

`src/db.c:179-245` calls `one_argument`, so only the first lowercased token is
examined. `all` and any token whose first character is `*` reload the named
static text globals; individual tokens select `wizlist`, `immlist`, `news`,
`credits`, `motd`, `imotd`, `help`, `info`, `policy`, `handbook`, `background`,
or `future`. `xhelp` rebuilds the indexed help table through `index_boot`. A
valid arm sends only the global `OK` string, `Okay.\r\n`; an empty or unknown
token sends `Unknown reload option.\r\n`. File-read/rebuild failures are not
surfaced by this handler. The shared dispatcher enforces the POS_DEAD and
LVL_IMPL-1 entry gate before the C function.

## Evidence and confirmed divergence

Scenario: `cmd/dp-oracle-diff/scenarios/reload-depth.txt`

Manifest: `docs/fidelity/depth/reload.tsv` (6 rows)

Focused test: `pkg/session/reload_depth_test.go`, pinning the C
LVL_IMPL-1/POS_DEAD gate and the registered Go entry.

The clean-main RED exposed one confirmed class divergence: Go treated `reload`
as a world-data reboot, emitted invented actor/global progress and failure
messages, attempted to parse the nonexistent `world/` directory, and did not
return C's bare acknowledgment. The fix ports the C text-cache arms, help
screen reload, xhelp table rebuild, first-token parser, silent refresh-error
behavior, and exact player-facing responses. The corrected vehicle covers
empty/unknown input, every named option, `all`, a `*`-prefixed option, and
`xhelp`; it was GREEN at seeds 1, 2, 3, 5, and 8, with seed 1 run using
`--show-oracle`. No `src/` or C-oracle file was edited.

## Verification and integration

All local gates passed:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .  # clean
```

Feature branch: `glm/depth-reload`

Feature commit: `f1484e579` (`fix: match reload text cache behavior`)

Feature PR: #1106 — merged as `9e375e18d`. Hosted lint, security, and test
checks were green in run `33574960842`; build/deploy were skipped by workflow
conditions. The PR was merged only after every reported hosted check was green.

## Fidelity rules

This slice follows R1 (player-facing bytes), R2 (command surface), R3
(determinism and ordering), R4 (no invention), R5 (process discipline), R5b
and R5c (shared behavior ownership), and R5e (verify the actual C call path).
The source-order claim is maintained for the next `recharge` slice.
