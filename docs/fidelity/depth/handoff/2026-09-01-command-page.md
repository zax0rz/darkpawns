# Depth-fidelity handoff — `page`

Date: 2026-09-01  
Branch: `glm/depth-page`  
Feature commit: `0bcb2b282`  
Feature PR: #1055 (merged to `main` as `ba1871a5e`)

## Frontier

The clean-main frontier before this slice was 2,702 total cases: 2,632
proven/delegated, 22 blocked, and 48 excluded. After merging `page`, it is
2,717 total: 2,646 proven/delegated, 22 blocked, and 49 excluded. Actionable
completion is 2,646/2,668 (99.2%). The special-procedure inventory remains
exhausted; `objmagic.sleep-entry-gates` remains the single explicitly blocked
vehicle; the command-table sweep continues after `page`.

## C-first call path

The registration at `src/interpreter.c:598` is `{ "page", POS_DEAD,
do_page, LVL_IMMORT, 0 }`. The handler is `src/act.comm.c:1107-1136`:

1. `half_chop` splits the target and preserves the remainder's internal and
   trailing whitespace.
2. NPC callers receive `Monsters can't page.. go away.`; an empty target
   receives `Whom do you wish to page?`.
3. `all` is accepted only when the caller is strictly above `LVL_GOD`; the
   lower-level refusal is `You will never be godly enough to do that!`.
4. Otherwise `get_char_vis` resolves a visible player, including the C
   `self`/`me` room aliases. A missing target or visible NPC reaches `There is
   no such person in the game!`.
5. C `act` sends the bell-prefixed page to the target and a second copy to the
   actor unless `PRF_NOREPEAT`, in which case the actor receives `OK`. The
   shared `SENDOK` gate suppresses delivery to a sleeping target. `all` walks
   the global descriptor list, so awake playing descriptors outside the room
   are included.

The shared act/audience behavior remains delegated to the verified
`socials-depth` machinery; this slice proves the `do_page` call boundary and
its target/audience decisions.

## RED and confirmed fixes

The initial Go path was RED on seed 1: it collapsed C `half_chop` body
spacing, used exact-case session lookup, did not resolve `self`/`me`, delivered
to a sleeping target, and omitted the blank line between the two observable
self-target `act` messages. These were confirmed against the C transcript.

The Go fix parses raw command text with C-compatible `half_chop` semantics,
preserves the C `act` CRLF bytes, resolves player names case-insensitively and
through `self`/`me`, applies the sleeping-target gate, and implements the
strict `all` authorization and `PRF_NOREPEAT` audience behavior. No oracle or
C source was edited.

## Evidence

- Scenario: `cmd/dp-oracle-diff/scenarios/page-depth.txt`.
- Manifest: `docs/fidelity/depth/page.tsv` (14 proven/delegated rows and one
  explicit unreachable-NPC exclusion).
- Focused tests: `pkg/session/page_depth_test.go`.
- Oracle matrix: seeds `1, 2, 3, 5, 8`, all `result: no normalized divergence`.
- `--show-oracle --seed 1` confirmed the intended C `do_page` blocks executed.
- Local gates passed: `make fidelity-depth`, `go build ./...`, `go vet ./...`,
  `go test ./...`, `golangci-lint run ./...`, and clean `gofumpt -l .`.
- Hosted checks for PR #1055 were green after the one permitted exact
  workflow retry (`Dark Pawns CI/CD`); lint, security, and test jobs passed,
  with build/deploy skipped by the workflow.

## Fidelity rules

This slice follows R1 (player-facing bytes), R2 (the registered command
surface), R3 (multi-seed determinism), R4 (no invented NPC/direct surface),
and R5/R5e (verify the actual C call path). Shared act behavior is handled at
the owning boundary under R5b/R5c rather than duplicated here.

## Next queue position

Return to clean `main`, pull, run `make fidelity-depth`, reread the depth guide
and this newest handoff, then resweep `src/interpreter.c` against all depth
manifests. `page` is claimed; continue with the next unclaimed family in
interpreter table order. Do not re-pick a command already owned by a manifest
or delegated boundary.
