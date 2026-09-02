# Depth-fidelity handoff — `show`

Date: 2026-09-02

## Queue position and result

This round began from clean `main` after `git pull --ff-only`, a successful
`make fidelity-depth`, and rereading `docs/fidelity/DEPTH_TESTING.md`, the
brief amendment, and the newest `2026-09-02-command-shiver.md` handoff. The
special-procedure inventory remains exhausted. The one-time blocked
`objmagic.sleep-entry-gates` row remains bounded by its cast-sleep
outlaw/reagent vehicle and was not repicked. The interpreter sweep consumed
the next source-order family, `show`, at `src/interpreter.c:694`.

The pre-slice frontier was 3,279 total cases, with 3,197 proven/delegated, 30
blocked, and 52 excluded. The show manifest contributes 26 cases: 19 new
proven cases, two shared delegations to the existing hcontrol matrix, and
seven blocked stateful branches. The resulting frontier is:

- 3,305 total cases
- 3,217 proven/delegated
- 36 blocked
- 52 excluded

## C call path and behavior surface

The registered row is:

```c
/* src/interpreter.c:694 */
{ "show"     , POS_DEAD    , do_show     , LVL_IMMORT, 0 },
```

The handler is `src/act.wizard.c:2234-2630`. It skips leading spaces, emits
the no-argument option response, parses the first two tokens with
`two_arguments`, resolves the field by case-sensitive prefix in this order:
`zones`, `player`, `rent`, `stats`, `errors`, `death`, `godrooms`, `shops`,
`houses`, `tattoos`, `aggr`, `reagents`, `hooks`, and `neutral`, then applies
the field-specific level gate. Unknown fields use C's exact refusal.

The proven vehicle covers the fresh empty-player/quiet-mob outputs for the
field resolver, invalid zone and missing-name gates, stats, errant/death/
godroom/neutral scans, empty houses through the delegated hcontrol owner, the
18-entry tattoo table, the quiet aggressive-24 scan, the reagent catalog, and
the hooks missing/invalid-zone gates. C's overlapping `sprintf` behavior was
measured in the oracle rather than “corrected”: the no-argument response is
the visible `neutral` tail, stats exposes only `0 buf switches ... overflows`,
and the room reports expose their final visible rows.

The `houses` case directly calls `hcontrol_list_houses`; both empty and
defined-house branches are therefore delegated to `hcontrol-depth` under
R5b/R5c. The following branches remain blocked after two honest slice
attempts and an actual C-path capability audit: valid zone reports, loaded
player reports, existing rent reports, the complete 60-shop pager, valid
cross-zone hooks, and populated `MOB_AGGR24` ordering. Go lacks byte-faithful
state/data surfaces for these branches. The handler does not invent
replacement output for them.

## Evidence and verification

The durable evidence is:

- `cmd/dp-oracle-diff/scenarios/show-depth.txt`, with annotated fresh-world
  gates and reports;
- `docs/fidelity/depth/show.tsv`, with 26 cases and explicit blocked rows;
- `pkg/session/show_depth_test.go`, pinning the C entry gate and registered
  Go entry.

The first honest run was RED on clean main: the old Go handler advertised
invented `players`/`uptime`/`reset` topics and returned `Unknown topic` for C
fields. The second full matrix, after the C field resolver was implemented,
was RED only for the complete C `shops` pager and one tattoo alignment byte;
the alignment was corrected, while shops and the other unavailable stateful
branches remain blocked. The final retained `show-depth` vehicle reported no
normalized divergence at seeds 1, 2, 3, 5, and 8; seed 1 was run with
`--show-oracle`.

The required local gates passed:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .  # clean
git diff --check
```

The round follows R1 (player-facing bytes), R2 (command surface), R3
(multi-seed determinism), R4 (no invented reports), R5/R5e (actual C call
path), and R5b/R5c (shared hcontrol ownership).

## Integration and continuation

Feature branch: `glm/depth-show`

Feature commit: `908abf3c0` (`test: prove show depth fidelity`)

Feature PR: #1169 — hosted lint, security, and test passed; conditional
build-and-push and deploy jobs were skipped. It was self-merged only after
all applicable checks were green, as main commit `4a7073d44`.

The post-merge manifest correction delegates both `show.houses` branches to
the existing hcontrol proof, with `make fidelity-depth` still green at the
frontier above. The correction is included in this handoff branch.

The next unclaimed interpreter-table family is `shoot`:

```c
{ "shoot"    , POS_STANDING, do_shoot    , 0, 0 },
```

The next session must return to clean `main`, pull, rerun
`make fidelity-depth`, reread the guide and this handoff, then audit and prove
`shoot` in table order.
