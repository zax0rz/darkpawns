# BRIEF (codex) — R2 command surface: grats, single-char commands, `'` say shorthand

**Owner:** codex (frontier). **Gate:** Claude runs the differential oracle red→green.
**Branch off `main`.** Sized to one PR.
**Closes:** DP-1185, DP-1186, DP-1187.
**Cite:** `src/interpreter.c` master command table + `command_interpreter()`; rules **R2a**,
**R2b**, **R4** (`docs/fidelity/RULEBOOK.md`).

## Why this chunk

Three reachability findings that are one underlying class: the Go registry is missing
C command-surface entries, and (probably) the tokenization rule that makes single-char
commands work. All player-facing, all oracle-gateable with no mobs and no RNG — the
cheapest red→green wins in the backlog.

## The C truth (all from `src/interpreter.c`, read-only)

| line | entry | handler | min pos | min lvl | subcmd |
|---|---|---|---|---|---|
| 473 | `grats` | do_gen_comm | POS_SLEEPING | 0 | SCMD_GRATZ |
| — | `.` | do_reply | POS_SLEEPING | 0 | 0 |
| 430 | `:` | do_echo | POS_RESTING | 1 | SCMD_EMOTE |
| — | `;` | do_wiznet | POS_DEAD | 0 | 0 |
| — | `?` | do_help | POS_DEAD | 0 | 0 |
| 671 | `'` | do_say | POS_RESTING | 0 | 0 |

(Grep the exact lines for `.`/`;`/`?` yourself — the table above came from the generated
`docs/reports/reachability-2026-07-22.tsv`, but the C source is the authority.)

**Tokenization — check this FIRST.** In C's `command_interpreter()`, if the first
non-space char of input is **not a letter**, C takes that single char as the command and
the rest as argument (`'hello` → command `'`, arg `hello` — no space needed). Read the
exact C logic and mirror it in the Go input path (`pkg/session/commands.go`
`ExecuteCommand` or wherever tokens are split). If Go splits on whitespace only, then
registering `'` alone fixes `' hello` but not `'hello` — that is a fidelity failure. The
tokenization rule is the class fix; the registrations are instances.

## The fixes

1. **`grats` (DP-1185, R2a — C names are law, including misspellings).** Register
   `grats` → the existing gratz/channel path (`pkg/session/commands.go:372`, cmdGratz →
   DoChannel "gratz"), POS_SLEEPING, level 0. **And remove the `gratz` registration**:
   C has no `gratz` command (only `grats` at :473 and `nograts` at :573 — `nograts`
   is already correctly registered). A Go-only command name is an R4 invention — a C
   player typing `gratz` gets "Huh?!?" and so must a Go player. Internal channel id
   "gratz" can stay; only the player-typed name changes.
2. **Single-char commands (DP-1186, R2b).** Register `.` `:` `;` `?` per the table,
   pointing at the existing handlers (cmdreply, cmdecho w/ emote subcmd, cmdwiznet,
   cmdhelp — see the TSV's go_evidence column). Mirror C's position/level gates from the
   table exactly; if a gate looks odd (`;` at level 0), the handler gates internally in
   C — mirror that, don't "fix" it (R1: no approximations).
3. **`'` say shorthand (DP-1187, R2b).** With tokenization fixed, register `'` →
   the say path, POS_RESTING, level 0. Note: the reachability generator found `'`
   handled only via speech spec-procs (`pkg/game/spec_procs3.go`) — that intercept is a
   separate mechanism and must keep working (spec-procs run before registry lookup);
   your change adds the fallthrough for every other room.

## Oracle gate (Claude runs — you don't need DP_ORACLE_BIN)

Provide scenario sketches in the PR description; Claude authors/normalizes and runs
`dp-oracle-diff`. All are mob-free and RNG-free — pure message diffs:

- `'hello` (no space) and `' hello` in a plain room → C say output both ways
- `:grins broadly` → emote; `.hi` with no tell pending → C's no-reply message
- `?` → help screen; `;test` as a mortal → whatever C does (mirror, don't assume)
- `grats everyone` → channel output; `gratz everyone` → must now be "Huh?!?" (the
  removal is as gateable as the addition)

Each must be RED on pre-fix `main` and GREEN after.

## Guardrails

- **Never** edit anything under `darkpawns-c-oracle/` — reference only.
- Branch off `main`; keep the PR focused. Add/extend the registration test
  (`pkg/session/commands_test.go` has the pattern — see the `mold` gate assertions).
- After your change, run `make reachability` — `grats`, `.`, `:`, `;`, `?`, `'` must all
  move to `registered`, and **nothing may regress** (the run exits non-zero on
  regression). Include the before/after counts in the PR description.
- Don't stage `website/static/map/world-sphere.json` or `docs/reports/reek/*`.

## Deliverable

Faithful tokenization + six surface names (five additions, one removal), unit tests,
`make reachability` before/after, per-command oracle scenario sketches. Claude
reconciles + runs the oracle gate.
