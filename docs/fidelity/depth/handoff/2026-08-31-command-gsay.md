# Depth handoff — 2026-08-31 — `gsay` / `gtell`

## Frontier and queue position

- Started from clean `main` at `0174605af` after the merged `grumble` handoff,
  pulled `main`, confirmed `make fidelity-depth`, and reread
  `docs/fidelity/DEPTH_TESTING.md` plus `2026-08-31-command-grumble.md`.
- The starting frontier was 2,022 total, with 1,964 proven/delegated, 16
  blocked, and 42 excluded. The gsay-family manifest adds 14 proven/delegated
  cases for both adjacent C rows. The post-slice frontier is 2,036 total,
  with 1,978 proven/delegated, 16 blocked, and 42 excluded; actionable
  completion is 1,978/1,994 (99.2%).
- `gsay` and `gtell` are covered at `src/interpreter.c:483-484` and share
  `do_gsay`. The next unclaimed command-table family is `help` at
  `src/interpreter.c:486`; the next session must return to clean `main`, pull,
  rerun the frontier check, reread this handoff, and begin `help`.

## C call path and branch inventory

The adjacent C rows register `gsay` and `gtell` as `POS_SLEEPING`, minimum
level 0, both routed to `do_gsay`. The handler in `src/act.comm.c:824-870` was
traced before claiming either row:

- `skip_spaces` runs first. The handler then gates on `AFF_GROUP`, emitting
  `But you are not the member of a group!` before the empty-argument branch.
- An in-group caller with no message receives
  `Yes, but WHAT do you want to group-say?`.
- With a message, C selects `ch->master` or the caller as group head, sends
  `$n tells the group, '%s'` to every grouped head/follower other than the
  caller with `TO_VICT | TO_SLEEP`, and sends the caller either
  `You tell the group, '%s'` or the C `OK` macro when `PRF_NOREPEAT` is set.
- C runs `delete_ansi_controls` over the formatted text, removing every `&`,
  and preserves internal and trailing whitespace from the raw argument. There
  is no PLR_NOSHOUT gate in this handler; group membership is its first
  gameplay gate.

The Go `gtell` registration and `gsay` alias were compared against
`pkg/session/commands.go`, `pkg/session/cmd_group.go`, the raw telnet argument
transport, and the C recipient path. The audit confirmed three divergences:
the Go path ignored `PRF_NOREPEAT`, collapsed raw spacing by joining tokens,
and leaked ampersand control markers. Only those confirmed differences were
fixed. The raw-argument dispatch is shared by both command names.

## Coverage proof

The C-first `gsay-depth --seed 1 --show-oracle` vehicle compared a three-player
group and exercised the no-group refusal for both names, empty arguments, group
leader/follower audience delivery, the gtell alias, C no-repeat confirmation,
a sleeping recipient, internal spacing, and ampersand control deletion. After
the fixes it reported no normalized divergence at seeds `1,2,3,5,8`. It never
edits `src/` or `darkpawns-c-oracle/`.

The manifest records 14 cases across both rows: entry and position gates,
not-in-group and empty-argument branches, group audience delivery, alias
delivery, no-repeat, sleeping recipient, raw spacing, and ANSI-control
deletion. `pkg/session/gsay_test.go` pins both C gates and the shared handler,
and directly verifies the raw-message/no-repeat behavior.

This follows R1/R2/R3/R4/R5e: C messages, command rows, recipient audiences,
and shared-handler behavior remain authoritative; five-seed parity is recorded;
no PLR_NOSHOUT behavior was invented; and the actual `do_gsay` path was
verified. Shared command-position behavior follows R5b/R5c through the named
`group.position-gate` manifest.

## Changes, gates, and integration

- Added `cmd/dp-oracle-diff/scenarios/gsay-depth.txt` with the full group,
  alias, state, spacing, and control-marker vehicle.
- Added `docs/fidelity/depth/gsay.tsv` with 14 explicit rows for `gsay` and
  `gtell`.
- Added `pkg/session/gsay_test.go` for C registration and shared-handler/raw
  message proof.
- Updated `pkg/session/cmd_group.go` and `pkg/session/commands.go` only for
  the three confirmed C-vs-Go divergences.
- Local gates passed: `make fidelity-depth`, `go build ./...`, `go vet ./...`,
  `go test ./...`, `golangci-lint run ./...`, `gofumpt -l .`, and
  `git diff --check`.
- PR #925 (`glm/depth-gsay`) passed hosted `test`, `lint`, and `security`
  checks and was merged. Checks did not initially report, so the single
  permitted `gh workflow run "Dark Pawns CI/CD" --ref glm/depth-gsay` retry
  was used; the resulting checks were all green before merge.

The next session must begin from clean `main`, pull, run `make fidelity-depth`,
reread this handoff, and continue the interpreter-table sweep with `help` at
`src/interpreter.c:486`.
