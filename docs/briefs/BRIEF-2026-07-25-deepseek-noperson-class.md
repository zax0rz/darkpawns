# BRIEF (deepseek) — DP-1200: NOPERSON fidelity class — "No-one by that name here." + do_sethunt hunter message

**Owner:** deepseek (mechanical class sweep). **Gate:** byte-exact unit tests
(mirror `pkg/session/cmd_skillset_test.go:120-129` /
`pkg/game/yank_port_test.go:46-51`); orchestrator probes `give <obj> nobody`
via the oracle after merge. CI green.
**Git:** branch off `main` as `fix/noperson-class`. Edit → commit → push →
open a PR. Do NOT merge. If the sandbox cannot create/push refs, commit
locally and leave the branch in place (outbox pattern) — say so in the summary.
Sized to one PR (S — string swaps + one missing branch + tests).
**Finding:** DP-1200 — C's `NOPERSON` is one canonical global string
(`src/config.c:93`: `char *NOPERSON = "No-one by that name here.\r\n";`,
declared `src/db.h:237`), referenced ~20 times across the C command surface
for "target person not found". Go mostly has it right today; a sweep
(2026-07-25, R5e-verified) found **one live invented variant** plus a related
`do_sethunt` gap. Rules **R1/R4/R5c/R5e**.

**Cite:** `src/config.c:93`, `src/act.item.c:708-725` (`give_find_vict`),
`src/act.wizard.c:3444-3460` (`do_sethunt`); Go
`pkg/game/item_transfer.go:451-468`, `pkg/session/wiz_zone.go:290-312`,
`pkg/session/wiz_player.go:490-493` (existing `noPersonHere` const).

---

## Verified current state (2026-07-25 — re-run these greps as your entry condition)

Already correct (do NOT churn): `pkg/session/wiz_player.go:493` (canonical
const `noPersonHere`), `pkg/game/other_mount.go:121`, `pkg/game/look.go:634`,
`pkg/game/directed_speech.go:165,294,299`,
`pkg/game/movement_commands.go:153,227`, `pkg/session/wiz_player.go:404`,
plus tests `cmd_skillset_test.go`, `yank_port_test.go`,
`directed_speech_test.go`.

Known bad:
- `pkg/game/item_transfer.go:460` — `"There doesn't seem to be anyone here by
  that name.\r\n"` (invented). This is the oracle-confirmed divergence from
  DP-1200 (`give sword nobody` → C: `No-one by that name here.`).
- `pkg/session/wiz_zone.go:301,308` — see Fix 2.

## Fix 1 — give_find_vict NOPERSON (R1/R4)

C (`src/act.item.c:708-725`): `give_find_vict` prints `"To who?\r\n"` for an
empty arg and `NOPERSON` for a missing victim — Go's empty-arg and self-give
(`"What's the point of that?\r\n"`) strings already match; only the not-found
string is wrong.

- Replace the invented string at `pkg/game/item_transfer.go:460` with the
  canonical NOPERSON bytes: `No-one by that name here.\r\n`.
- **Unify the constant (R5c):** hoist one exported canonical const (e.g.
  `NoPersonHere = "No-one by that name here.\r\n"` — cite `config.c:93` in its
  doc comment) into a shared `pkg/game` location and use it at
  `item_transfer.go:460` and the other `pkg/game` literal sites listed above
  (`other_mount.go:121`, `movement_commands.go:153,227`,
  `directed_speech.go:165,294,299`). `pkg/session` keeps its own
  `noPersonHere` (import cycle rules — check; if `pkg/session` can import the
  const without a cycle, use it there too). **Bytes must not change at any
  already-correct site** — this is deduplication, not rewording.
- Verify the helpers: `look.go:634` uses `result.literal(...)` without an
  explicit `\r\n`, and `directed_speech.go` routes via `communicationSend` —
  confirm each emits exactly `No-one by that name here.\r\n` on the wire
  (their tests already assert this; keep them green).

## Fix 2 — do_sethunt messages (R1, found by the sweep)

C `do_sethunt` (`src/act.wizard.c:3444-3460`) has DISTINCT messages:
- no arg → `"Who do you wish to hunt?\n\r"`
- victim not found → `"No-one by that name around.\n\r"`
- hunter not found → `"Who shall be the hunter?\n\r"`
- hunter == victim → `"Yeah right."` (check C's exact terminator)

Go (`pkg/session/wiz_zone.go:290-312`) prints `"No-one by that name around."`
for BOTH the victim-miss AND the hunter-miss (`:301`, `:308`), and the
no-arg/same-name branches need byte-checking against C.

- `:308` (hunter miss) → `"Who shall be the hunter?"` (+ C's terminator).
- Terminators: C uses `\n\r` (reversed!) for these `do_sethunt` lines —
  `wiz_player.go:378` already documents this C quirk ("C uses `\r\n` for the
  syntax line and NOPERSON, `\n\r` elsewhere"). Determine what
  `Session.Send`/`SendMessage` appends and make the emitted bytes match C
  exactly, `\n\r` included. If the send helper normalizes endings, pass the
  exact bytes through whatever path preserves them.
- Also byte-check the `"Yeah right."` branch against C (hunter==victim at
  `act.wizard.c:3459-3460`) and fix the terminator if it differs.

## Fix 3 — class audit, remaining C sites (R5c; verify, don't churn)

For EACH C `NOPERSON` call site below, trace the Go counterpart command and
confirm the not-found path emits the canonical bytes; fix any that don't
(most should already be fine — the 2026-07-25 grep found no other invented
variants, but the grep only catches known phrasings, so trace the call paths,
R5e):

`src/act.comm.c:910,912,1003` · `src/act.informative.c:2443` ·
`src/act.item.c:720` (Fix 1) · `src/act.movement.c:851,896` ·
`src/act.offensive.c:75` · `src/act.other.c:720,1565,1631,1685` ·
`src/act.wizard.c:168,321,375,1593,1870,3528` · `src/modify.c:283` ·
`src/new_cmds.c:1123`.

Report a table in the PR: C site → Go file:line → status (already-canonical /
fixed / no-Go-counterpart). If a C command has no Go counterpart at all, do
NOT implement it — note it and move on.

## Tests

- `pkg/game/item_transfer_test.go` (extend or new): `give sword nobody` with
  no matching mob/player in room → exactly `No-one by that name here.\r\n`.
  Mirror the `yank_port_test.go:46-51` byte assertion pattern. If a give
  scenario needs a world fixture, copy the smallest existing one.
- `pkg/session/wiz_zone_test.go` (extend or new): `sethunt` with missing
  hunter → `Who shall be the hunter?` + C terminator; missing victim →
  `No-one by that name around.` + C terminator. Assert exact bytes including
  the `\n\r`.
- Any additional divergence found in Fix 3 gets the same byte-exact test.

## Oracle gate (orchestrator, after merge — informational)

Probe `give <obj> nobody` against the C oracle (the DP-1200 red) — expect
green. `sethunt` is a wizard command outside the current scenario corpus;
unit bytes are its gate.

## Guardrails

- **Never** edit `src/`, `darkpawns-c-oracle/`, or `lib/misc/messages` — read-only.
- All gates (AGENTS.md §Build & Verify): build, vet, `test ./... -race`,
  `golangci-lint run`, `gofumpt -l .` empty, `make reachability`.
- Do not reword any string that already matches C (R4 cuts both ways).

## Deliverable

`item_transfer.go` NOPERSON fixed + canonical const unified across `pkg/game`;
`do_sethunt` hunter/victim/no-arg messages byte-exact vs C (terminators
included); the Fix-3 audit table in the PR body; byte-exact tests; all gates
green.
