# Depth-fidelity handoff — `snuggle`

Date: 2026-09-03

Branch: `glm/depth-snuggle`

Feature PR: #1269 (merged green)

Feature commit: `827abf037`

Main merge: `d501cc34a`

Handoff branch: `handoff/2026-09-03-command-snuggle`

## Queue position and result

This round checked out `main`, pulled with `git pull --ff-only`, confirmed the
frontier with `make fidelity-depth`, reread `docs/fidelity/DEPTH_TESTING.md`
and the newest dated handoff, and then took the next unclaimed interpreter
row after `snoop`.

The special-procedure inventory remains exhausted. The one blocked row,
`objmagic.sleep-entry-gates`, remains blocked after its one allowed
cast-sleep outlaw/reagent vehicle and was not repicked. The next source-order
row after this slice is `socials` at `src/interpreter.c:725`. The existing
`docs/fidelity/depth/socials.tsv` claims the generic `social`/`do_action`
family; it does not claim the distinct `socials`/`do_commands` listing row.

Pre-slice frontier: 3,798 total, 3,695 proven/delegated, 48 blocked, and 55
excluded. The `snuggle` manifest adds 12 proven/delegated cases. Post-slice
frontier: 3,810 total, 3,707 proven/delegated, 48 blocked, and 55 excluded;
actionable completion is 3,707/3,755 = 98.7%.

## C call path and observable contract

The registered C row is:

```c
/* src/interpreter.c:724 */
{ "snuggle"  , POS_RESTING , do_action   , 0, 0 },
```

`do_action` in `src/act.social.c:102-151` resolves the social, checks
`PLR_NOSHOUT`, parses only the first target token when `char_found` exists,
and then selects the no-argument, missing-target, self-target, victim-position,
or three-audience success branch. The `snuggle` record at
`lib/misc/socials:794-802` is:

```text
snuggle 1 5
Who?
#
you snuggle $M.
$n snuggles up to $N.
$n snuggles up to you.
They aren't here.
Hmmm...
#
```

The scenario named an actor, observer, target, and disposable NPC and covered
no argument, visible target, target audience delivery, first-token parsing,
NPC target, self target, missing target, and a sleeping target rejected by
the authored minimum victim position. Command POS_RESTING dispatch, the
common `PLR_NOSHOUT` refusal, and shared visible-room/Act audience behavior
remain delegated to `flip.position-gate`, `dance-noshout`, and
`socials-depth` under R5b/R5c.

## Evidence and implementation boundary

The durable evidence is:

- `cmd/dp-oracle-diff/scenarios/snuggle-depth.txt`;
- `docs/fidelity/depth/snuggle.tsv`; and
- `pkg/session/snuggle_depth_test.go`.

The clean-main-equivalent oracle run was GREEN before any runtime change was
considered: `snuggle-depth --show-oracle` reported no normalized divergence
at seed 1. Seeds 2, 3, 5, and 8 also reported no normalized divergence. The
focused registration/record test passed. This was a pure-coverage slice;
inventing a behavior change would violate R4.

No file under `src/` or `darkpawns-c-oracle/` was edited.

## Gates and review

The final local gates passed on the feature branch:

- `make fidelity-depth`
- `go build ./...`
- `go vet ./...`
- `go test ./...`
- `golangci-lint run ./...` — 0 issues
- `gofumpt -l .` — clean
- `git diff --check`

PR #1269's hosted lint, security, and full test checks completed green;
conditional build-and-push and deploy were skipped. The initial check query
reported no checks, so the one permitted retry was run with
`gh workflow run "Dark Pawns CI/CD" --ref glm/depth-snuggle`; the resulting
checks were green before self-merge.

This slice follows R1 (player-facing bytes), R2 (registered command surface),
R3 (seed matrix and audience ordering), R4 (no invented behavior), R5/R5e
(verify the actual C path and let C win), and R5b/R5c (shared social gate,
lookup, and audience ownership).

## Continuation

The next session must checkout `main`, pull with `--ff-only`, rerun
`make fidelity-depth`, reread the guide and newest handoff, and audit/claim
`socials` at `src/interpreter.c:725`. Follow `do_commands` through its C
call path and compare it against the unclaimed command-family manifests
before creating the next `glm/depth-socials` feature branch. Do not repick
`snuggle`, the generic `social` family, or the blocked sleep-entry row.
