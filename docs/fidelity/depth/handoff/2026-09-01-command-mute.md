# Depth-fidelity handoff — `mute`

Date: 2026-09-01  
Queue: un-manifested interpreter command families, source-table order  
Rules: R1, R2, R3, R4, R5b, R5c, R5e

## Frontier

Started from fresh `main` after the `mount` handoff.  `make fidelity-depth`
reported 2,602 cases: 2,532 proven/delegated, 22 blocked, and 48 excluded
(99.1% actionable).  After the `mute` evidence was merged, the frontier is
2,616 total: 2,546 proven/delegated, 22 blocked, and 48 excluded (99.1%
actionable).

The source-order sweep treats the existing social table and generic
`do_not_here` family manifests as family claims, and recognizes `murder` as
already owned by the shared `do_hit`/combat-entry proof.  The next actually
unclaimed family is `neckbreak` at `src/interpreter.c:564`
(`POS_FIGHTING`, level 0, `do_neckbreak`).  The later `news` and `newbie`
rows remain queued after it; `muhaha` and `mumble` belong to the existing
social-family claim.

## C path and RED → GREEN proof

The source row is `{ "mute", POS_DEAD, do_wizutil, 1, SCMD_SQUELCH }` at
`src/interpreter.c:559`.  The real handler is `src/act.wizard.c:2077-2140`:
the LVL_IMMORT/PLR_CHOSEN inner authorization guard runs first, then C
`one_argument`, visible target lookup, NPC rejection, higher-immortal
protection, and the SCMD_SQUELCH PLR_NOSHOUT toggle.  The successful branch
emits only the actor's `(GC) Squelch ON/OFF ...` acknowledgement.  C's
interpreter prefix scan also makes unique `mut` reach the same row.

The first attempted vehicle was invalid because `[probe:muteon]` made the
mortal peer the actor; its `Huh?!?` output was discarded rather than counted
as proof.  With the primary Implementor restored as actor, clean `main`
reached the intended no-argument, missing-target, visible-NPC, ON/OFF,
first-token, and `mut` prefix blocks with no normalized divergence (R1/R2/R5e).
The separate ordinary-mortal vehicle reached the pre-lookup inner-auth
refusal for bare, targeted, and prefix forms.

## Durable proof

Added:

- `cmd/dp-oracle-diff/scenarios/mute-depth.txt` — authorized actor, target
  gates, state toggles, actor-only acknowledgement, trailing arguments, and
  the C `mut` abbreviation.
- `cmd/dp-oracle-diff/scenarios/mute-low-depth.txt` — ordinary level-1
  authorization refusal before argument/target lookup.
- `pkg/session/mute_depth_test.go` — C gate registration, target branches,
  state transitions, audience silence, higher-immortal protection, and
  mortal authorization.
- `docs/fidelity/depth/mute.tsv` — 14 rows covering the reachable branches
  and the C-prefix surface.

Both vehicles were GREEN at seeds 1, 2, 3, 5, and 8; `mute-depth` was run
with `--show-oracle` at seed 1 and its blocks reached the intended C handler.
No Go behavior required changing; the existing `cmdMute`/`wizutilDispatch`
path matched the C oracle.  No `src/` or C-oracle files were edited (R1-R5e).

## Gates and merge

Local gates passed on `glm/depth-mute`:

- `make fidelity-depth`
- `go build ./...`
- `go vet ./...`
- `go test ./...`
- `golangci-lint run ./...` (0 issues)
- `gofumpt -l .` clean

Feature PR #1036 (`glm/depth-mute`) had green hosted `lint`, `security`, and
`test` checks; build/deploy were skipped by the PR workflow.  It was
self-merged under the 2026-08-27 amendment as squash commit `cdb282ebf`.

The post-merge `main` checkout/pull and frontier rerun passed with the counts
above.  The next session must start on `main`, pull, rerun
`make fidelity-depth`, reread `docs/fidelity/DEPTH_TESTING.md` and this newest
handoff, then map and attempt only `neckbreak` at
`src/interpreter.c:564`.
