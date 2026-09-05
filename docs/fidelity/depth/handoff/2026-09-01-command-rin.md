# Depth-fidelity handoff — `rin`

Date: 2026-09-01

## Queue position

This round began from clean `main` after `git pull --ff-only`, a successful
`make fidelity-depth`, and rereading `docs/fidelity/DEPTH_TESTING.md` plus the
latest `review` handoff. The special-procedure inventory remains exhausted, the
single blocked row `objmagic.sleep-entry-gates` remains queued after its one
cast-sleep vehicle, and the interpreter sweep advanced from `review` through
the shared `ride` row to `rin`.

The source-order audit confirms that `ride` is already owned by the `mount`
manifest and `roomflags` is already owned by `gen-tog`. The next unclaimed
interpreter-table family is `rlist` at `src/interpreter.c:661`.

Frontier before this slice: 3,001 total; 2,924 proven/delegated; 26 blocked;
51 excluded.

Frontier after this slice: 3,015 total; 2,938 proven/delegated; 26 blocked;
51 excluded.

## C call path and behavior surface

The command-table entry is:

```c
/* src/interpreter.c:660 */
{ "rin"      , POS_STANDING, do_kuji_kiri, 0, SKILL_KK_RIN },
```

`src/new_cmds.c:1552-1739` implements the shared `do_kuji_kiri` path. It
checks the non-Ninja mortal, fighting, aggregate `AFF_KUJI_KIRI`, and mounted
early returns, then consumes `check_kk_success()` before the per-seal mastery
gate. Rin's mastered success arm sets `APPLY_AC` to `-(15 + level/2)`, adds a
separate five-tick `AFF_METALSKIN` record, sends the harden-body actor text,
and sends the metal-skin room act through `act()`. Failure zeroes the numeric
payload, changes the second record to `AFF_NOTHING`, keeps the primary
lockout record, uses the generic concentration-failure actor text, and leaves
the room meditation act unchanged. The handler does not parse command
arguments.

## Evidence and confirmed divergence

Scenario: `cmd/dp-oracle-diff/scenarios/rin-depth.txt`

Manifest: `docs/fidelity/depth/rin.tsv` (14 rows)

Focused tests: `pkg/game/rin_depth_test.go`,
`pkg/session/rin_depth_test.go`, and the existing Rin state proof in
`pkg/game/skill_berserk_kuji_test.go`.

The clean-main RED scenario found one confirmed player-visible divergence:
C's `act()` expanded the Rin room template to
`Rinactor interlaces his fingers, and his skin becomes metal!`, while Go
returned the literal `$n/$s` template to the room audience. The Go fix changes
only Rin's room result to use the canonical `ActMessage` expansion; no `src/`
or C-oracle file was edited. The focused failure proof also pins C's unchanged
room act and generic actor failure, while the existing state proof pins AC and
metalskin success plus the failure lockout behavior.

The corrected scenario is GREEN with `--show-oracle` and across seeds 1, 2, 3,
5, and 8. The argument-bearing first probe proves trailing words are ignored,
and the second probe proves the successful aggregate lockout with no invented
room bytes.

## Verification and integration

All required local gates passed on the feature branch:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .  # clean
git diff --check
```

Feature branch: `glm/depth-rin`

Feature commit: `50401f11d` (`fix: expand C rin room act`)

Feature PR: #1121 — hosted lint, security, and test checks were green; the
workflow's build-and-push and deploy jobs were skipped by conditions. The
automatic workflow did not initially report checks, so the one permitted exact
retry `gh workflow run "Dark Pawns CI/CD" --ref glm/depth-rin` was used. The PR
was self-merged as main commit `52ee07e1fb73` only after all required hosted
checks were green.

The earlier open PRs for `plot`, `purge`, and `qecho` remain open because their
checks did not fire after their one permitted exact workflow retry; none was
merged.

## Fidelity rules

This slice follows R1 (player-facing bytes), R2 (command surface), R3
(determinism across the oracle matrix), R4 (no invention), R5 (process
discipline), and R5e (verify the actual C call path). Shared Kuji Kiri gates,
skill-key translation, and source-order ownership remain under R5b/R5c.
