# Depth handoff — 2026-08-31 — `gasp`

## Frontier and queue position

- Started from clean `main` at `30c9306c7` after merging the gag handoff,
  ran `git pull --ff-only`, confirmed `make fidelity-depth`, and reread
  `docs/fidelity/DEPTH_TESTING.md` plus the newest prior handoff,
  `2026-08-31-command-gag.md`.
- The frontier before this slice was 1,846 total, with 1,789
  proven/delegated, 16 blocked, and 41 excluded. The dedicated `gasp` manifest
  adds eight proven/delegated cases: four direct cases, one entry-gate unit
  proof, and three shared delegations. The post-slice frontier is 1,854 total,
  with 1,797 proven/delegated, 16 blocked, and 41 excluded; actionable
  completion is 1,797/1,813 (99.1%).
- A fresh source-order audit of `src/interpreter.c:459-475` confirms `give`,
  `glance`, and `gold` already have depth coverage. The next un-manifested
  command is `gecho` at line 462, so the next session must return to clean
  `main`, pull, rerun the frontier check, reread this handoff, and begin
  `gecho`.

## C call path and branch inventory

`src/interpreter.c:461` registers `gasp` with `POS_RESTING`, no minimum level,
and `do_action`. Its authored social record is `lib/misc/socials:295-298`:
`gasp 0 0`, followed by the actor line, room line, and `#` sentinels for all
target branches. The actual handler path is `src/act.social.c:102-151`:
the PLR_NOSHOUT gate runs first, `char_found == NULL` clears the argument,
and every typed, missing, self-named, or trailing argument reaches the
no-argument actor/room pair.

The vehicle uses a primary actor and room peer to prove the exact actor and
audience bytes for no argument, a visible target argument, a missing target,
and a self-named target with trailing words. The shared POS_RESTING command
gate, PLR_NOSHOUT refusal, and Act visibility behavior are delegated to
existing depth owners. No Go behavior change was indicated.

## Coverage proof

The pure-coverage vehicle was GREEN on the unchanged main implementation;
there was no pre-fix RED because no divergence was found to fix. The
`gasp-depth --show-oracle` run showed the intended C blocks and exact output
for seed 1. The same scenario was GREEN with no normalized divergence for
seeds 1, 2, 3, 5, and 8. `TestGaspRegistrationUsesCEntryGate` passed.

The work follows R1/R2/R4, R5e, and R5c: C bytes and the registered command
surface remain authoritative, the actual `do_action` call path was verified,
the no-target social behavior is proven without inventing target output, and
shared social behavior is delegated rather than duplicated.

## Gates and merge

Local gates passed:

- `make fidelity-depth` — 1,854 total / 1,797 proven-or-delegated /
  16 blocked / 41 excluded;
- `go build ./...`;
- `go vet ./...`;
- `go test ./...`;
- `golangci-lint run ./...` — 0 issues;
- `gofumpt -l .` clean; and
- `git diff --check` clean.

Implementation PR #890 was merged only after hosted `lint`, `security`, and
`test` checks were all green. The workflow's `build-and-push` and `deploy`
jobs were skipped by policy. This handoff must itself be merged with green
checks before the next session begins `gecho`.
