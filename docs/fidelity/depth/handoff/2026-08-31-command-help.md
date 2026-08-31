# Depth handoff — 2026-08-31 — `help` / `?`

## Frontier and queue position

- Started from clean `main` at `0e74dc7fd` after the merged `gsay` / `gtell`
  handoff, pulled `main`, confirmed `make fidelity-depth`, and reread
  `docs/fidelity/DEPTH_TESTING.md` plus `2026-08-31-command-gsay.md`.
- The starting frontier was 2,036 total, with 1,978 proven/delegated, 16
  blocked, and 42 excluded. The help feature manifest added 12
  proven/delegated cases. The final manifest also records the existing focused
  empty-table proof and the descriptor-less early return as an explicit
  exclusion, producing 2,051 total, 1,992 proven/delegated, 16 blocked, and
  43 excluded; actionable completion is 1,992/2,008 (99.2%).
- `help` is registered at `src/interpreter.c:486` and `?` at `:488`, both
  routed to `do_help`; the intervening `headbutt` row at `:487` is a separate
  command family. Because the help family owns both rows reaching the same C
  handler, the next unclaimed interpreter-table family is `headbutt` at
  `src/interpreter.c:487`. The next session must return to clean `main`, pull,
  rerun the frontier check, reread this handoff, and begin that family.

## C call path and branch inventory

The C table has level-zero, `POS_DEAD` rows for both `help` and `?`. The actual
handler in `src/act.informative.c:1566-1674` was traced before any Go change:

- `do_help` returns silently when `ch->desc` is absent, then calls
  `skip_spaces(&argument)`.
- With no remaining argument it calls `page_string(ch->desc, help, 0)` on the
  booted `lib/text/help/screen` text. A screen longer than one page enters the
  C output pager; pager input is routed by `comm.c:617` to `show_string`.
- With a nonempty argument and no help table it sends exactly
  `No help available.\r\n`.
- Otherwise it binary-searches the help table with
  `strn_cmp(argument, keyword, strlen(argument))`, then walks backward to the
  first matching prefix entry. A miss sends
  `There is no help on: %s\r\n`; the mudlog and `misc/help` append are
  server-side only.
- Mortal callers whose first matching entry contains the literal `wizonly`
  receive the same miss line. Immortals receive the entry.
- A visible hit uppercases the matched keyword, emits the C ANSI header and
  the leading-space-plus-75-dash separator, skips the keyword line, and
  `page_string`s the body. The table and screen are loaded from the indexed
  help files by `db.c:644-685` and `db.c:212`.

The live vehicle exercised both command rows, the raw internal-spacing miss,
prefix-first-match, topic header/body bytes, mortal `wizonly` hiding, the `?`
alias raw tail, the no-argument screen pager, pager quit, and normal command
dispatch after pager exit. The focused unit test also covers the empty-table
line and the authoritative entry gates. The descriptor-less early return is
explicitly excluded because the registered player command is descriptor-issued
and no command-surface vehicle can reach that branch without inventing a
different call site.

## Confirmed divergence and fix

The new `help-depth` scenario was run on pre-fix `main` before the feature
branch. It was RED in two player-visible cases:

1. C preserved `help zzqx  tail` as `There is no help on: zzqx  tail`, while
   Go joined tokenized arguments as `zzqx tail`.
2. C treated `? cure  light` as a miss and printed the two spaces, while Go
   collapsed the tokens to `cure light` and incorrectly displayed the CURE
   LIGHT page.

Only those confirmed divergences were fixed. `cmdHelpText` now accepts the
transport's original argument remainder; `executeCommandRaw` routes raw tails
for both `help` and `?`, while direct unit callers retain the existing
tokenized `cmdHelp` API. No source or C-oracle file was edited.

## Coverage proof

- Added `cmd/dp-oracle-diff/scenarios/help-depth.txt` for the mortal path and
  `help-immortal-depth.txt` for the immortal path. Both were shown once with
  `--show-oracle` during development.
- Both scenarios reported no normalized divergence at seeds `1, 2, 3, 5, 8`.
- Added `pkg/session/cmd_info_help_test.go` coverage for the C gate, alias
  registration, raw spacing, and direct handler behavior.
- Added 15 manifest rows in `docs/fidelity/depth/help.tsv`, with the `?`
  cases recorded against its actual `src/interpreter.c:488` registration.

This follows R1/R2/R3/R4/R5e: the C bytes, command rows, pager routing, raw
argument surface, and mortal/immortal audience rule remain authoritative;
multi-seed proof was recorded; no registry fallback or argument normalization
was invented; and the actual `do_help` path was verified before the fix.

## Changes, gates, and integration

- PR #927 (`glm/depth-help`) passed hosted `test`, `lint`, and `security`
  checks; the release-only build/deploy jobs were skipped as expected. It was
  merged only after all reported checks were green; merged main is
  `8db8bf59f`.
- Local gates passed: `make fidelity-depth`, `go build ./...`, `go vet ./...`,
  `go test ./...`, `golangci-lint run ./...`, `gofumpt -l .`, and
  `git diff --check`.
- The final manifest-only additions in this handoff branch are the
  `help.topic-display` scenario annotation, the focused empty-table row, and
  the explicit descriptor-less exclusion; they do not alter the Go runtime.

The next session must begin from clean `main`, pull, run `make fidelity-depth`,
reread this handoff, and continue the interpreter-table sweep with
`headbutt` at `src/interpreter.c:487`.
