# Depth handoff — 2026-08-31 — `gecho`

## Frontier and queue position

- Started from clean `main` at `d0710e82c` after merging the gasp handoff,
  ran `git pull --ff-only`, confirmed `make fidelity-depth`, and reread
  `docs/fidelity/DEPTH_TESTING.md` plus the newest prior handoff,
  `2026-08-31-command-gasp.md`.
- The frontier before this slice was 1,854 total, with 1,797
  proven/delegated, 16 blocked, and 41 excluded. The dedicated `gecho`
  manifest adds six proven cases: five direct/unit cases and one D5 queue
  proof. The post-slice frontier is 1,860 total, with 1,803
  proven/delegated, 16 blocked, and 41 excluded; actionable completion is
  1,803/1,819 (99.1%).
- A fresh source-order audit of `src/interpreter.c:459-475` confirms `give`,
  `glance`, and `gold` already have depth coverage. With `gecho` now covered,
  the next un-manifested command is `giggle` at line 464, so the next session
  must return to clean `main`, pull, rerun the frontier check, reread this
  handoff, and begin `giggle`.

## C call path and branch inventory

`src/interpreter.c:462` registers `gecho` with `POS_DEAD` and `LVL_GOD`,
dispatching to `do_gecho`. The actual handler path is
`src/act.wizard.c:1690-1707`: `skip_spaces` removes only leading argument
whitespace; empty input emits `That must be a mistake...`; non-empty input is
sent to every other connected playing descriptor; and the issuer receives
the same text unless `PRF_NOREPEAT` is set, in which case C sends
`Okay.\r\n`.

The primary God plus a connected peer vehicle proved empty input, ordinary
global broadcast, internal/trailing spacing, and the norepeat actor/peer
audiences. The implementation carries the raw argument remainder through
telnet and C wait-state queues while preserving tokenized arguments for other
handlers. The command gate is pinned by `TestGechoRegistrationUsesCEntryGate`.

## Coverage proof

The unchanged-main `gecho-depth --seed 1 --show-oracle` run was RED for
internal spacing: C emitted `Alpha  beta  gamma`, while Go emitted
`Alpha beta gamma`. The unchanged-main `gecho-norepeat-depth --seed 1
--show-oracle` run was RED because Go repeated the message to the actor while
C emitted `Okay.`. After the fix, both vehicles showed the intended C blocks
and no normalized divergence for seeds `1,2,3,5,8`. The D5 queue test also
passed and confirmed raw spacing survives delayed execution.

The work follows R1/R2/R4, R5e, and R5c: C bytes and the registered command
surface remain authoritative, the actual handler and wait-queue call paths
were verified, no invented gecho length branch remains, and the raw-spacing
fix was carried through the whole command transport class.

## Gates and merge

Local gates passed:

- `make fidelity-depth` — 1,860 total / 1,803 proven-or-delegated /
  16 blocked / 41 excluded;
- `go build ./...`;
- `go vet ./...`;
- `go test ./...`;
- `golangci-lint run ./...` — 0 issues;
- `gofumpt -l .` clean; and
- `git diff --check` clean.

Implementation PR #892 was merged only after hosted `lint`, `security`, and
`test` checks were all green. The workflow's `build-and-push` and `deploy`
jobs were skipped by policy. This handoff must itself be merged with green
checks before the next session begins `giggle`.
