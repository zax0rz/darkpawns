# Depth handoff — 2026-08-31 — `future`

## Frontier and queue position

- Started from clean `main` at `d13b1bbaa` after the merged `fume` handoff,
  ran `git pull --ff-only`, confirmed `make fidelity-depth`, and reread
  `docs/fidelity/DEPTH_TESTING.md` plus the newest prior handoff,
  `2026-08-31-command-fume.md`.
- The frontier before this slice was 1,823 total, with 1,766
  proven/delegated, 16 blocked, and 41 excluded. The dedicated `future`
  manifest adds three proven cases: one entry-gate unit proof and two direct
  oracle cases. The post-slice frontier is 1,826 total, with 1,769
  proven/delegated, 16 blocked, and 41 excluded; actionable completion is
  1,769/1,785 (99.1%).
- The source-order command gap was `future`, registered at
  `src/interpreter.c:456`. The next command-table gap is `fwap` at line 457;
  the next session must return to clean `main`, pull, rerun the frontier
  check, reread this handoff, and begin `fwap`.

## C call path and branch inventory

`src/interpreter.c:456` registers `future` with `POS_DEAD`, no minimum level,
and `do_gen_ps`/`SCMD_FUTURE`. The actual C branch is
`src/act.informative.c:2117-2155`: the `SCMD_FUTURE` case calls
`page_string(ch->desc, future, 0)` and does not inspect command arguments.
`future` is boot-cached from `FUTURE_FILE` (`src/db.c:198`, `src/db.h:61`),
whose authoritative bytes are the eight-line `lib/text/future` file.

The branch inventory is therefore the unrestricted dead-position entry gate,
the exact one-page static text output, and repeated output with trailing
arguments ignored. The file fits within the C pager's one-page boundary, so
there is no pager prompt or navigation branch for this content. No audience,
state transition, RNG, authorization, or special-procedure branch is
reachable from this command.

## Coverage proof

The clean-main baseline was GREEN: `future-depth --show-oracle` reported the
exact C page for both bare `future` and `future trailing words are ignored`,
with no normalized divergence. The same vehicle was GREEN for seeds
`1,2,3,5,8`. `TestFutureRegistrationUsesCEntryGate` pins the C-derived
`(LVL 0, POS_DEAD)` registration. No Go behavior change was inferred, and no
`src/` or `darkpawns-c-oracle/` file was edited.

The work follows R1/R2/R4, R5e, and R5c: the static page bytes and command
surface remain C-authoritative, the real `do_gen_ps` path was verified, and
the absence of additional branches was established from the actual switch and
file length rather than invented behavior.

## Gates and merge

Local gates passed:

- `make fidelity-depth` — 1,826 total / 1,769 proven-or-delegated /
  16 blocked / 41 excluded;
- `go build ./...`;
- `go vet ./...`;
- `go test ./...`;
- `golangci-lint run ./...` — 0 issues;
- `gofumpt -l .` clean;
- `git diff --check` clean.

Implementation PR #881 was merged only after hosted `lint`, `security`, and
`test` checks were all green. The workflow's `build-and-push` and `deploy`
jobs were skipped by policy. This handoff must itself be merged with green
checks before the next session begins `fwap`.
