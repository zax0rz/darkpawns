---
tags: [active, governing, fidelity, port, rulebook]
last_updated: 2026-07-22
author: Claude Code (Opus) with The Architect
---
# The Port Rulebook — C→Go Translation Law

Every translation decision, decided once. This is the artifact the Anthropic
migration methodology calls the rulebook: when a fidelity failure repeats, the
fix lands **here** (plus a class audit), not just in the file that happened to
fail. Subordinate to [north-star-1to1](north-star-1to1.md); proven by the
oracle ([oracle-differential-testing](oracle-differential-testing.md)).

**How to amend:** an oracle red whose root cause is a *pattern* (not a one-off
typo) gets a rule entry with the incident that taught it. Then audit the class
— one confirmed instance means siblings exist (cf. the DP-597 → seven more
0o644 sites sweep). A rule without an incident citation is a guess; don't add it.

---

## R1. Player-facing bytes are law

Every prompt, message, menu, ordering, and byte a player can see must match
the C server exactly. Not "equivalent." Not "improved." Identical.

- **Taught by:** character creation (DP-1173) — the port re-skinned `nanny()`
  with invented bracket menus and a fabricated MOTD; `do_recall` had invented
  self-messages and a hardcoded room target. Both rewritten byte-for-byte.
- **Comply:** when porting a handler, open the C function and copy its output
  strings verbatim — including grammar errors, spacing, and `\r\n` discipline.
  If a Go message has no C counterpart, it is a bug (see R4).
- **Verify:** oracle scenario. No fix ships on "looks right" (oracle proof gate).

## R2. The command surface is part of the game

The C `cmd_info[]` table in `src/interpreter.c` defines the playable surface:
names, aliases, minimum positions, minimum levels, subcommands.

- **R2a. C names are law, including misspellings.** C registers `grats`; the
  port registered only `gratz` → C players get "Huh?!?". (Taught by DP-1185.)
- **R2b. Single-character aliases are player surface.** `.` `:` `;` `?` `'`
  are muscle-memory commands, not parser trivia. (Taught by DP-1186/DP-1187.)
- **R2c. Position/level gates mirror the C table exactly.** `mold` is
  LVL_IMMORT/POS_RESTING because interpreter.c says so — and there's a test
  asserting it (`pkg/session/commands_test.go`). New registrations get the
  same treatment.
- **R2d. Known open gap — prefix matching.** C matches abbreviations (`mur` →
  `murder`); the Go registry is exact-match only. Any per-command alias fix is
  a workaround for this class-level gap. Don't close individual instances and
  call the class done.
- **Verify:** `make reachability` — the deterministic C-table-vs-registry diff.
  A command regressing from reachable to unreachable turns the Monday embed red.

## R3. Determinism and draw parity

The oracle compares two live processes; they only stay comparable if the Go
port consumes randomness and time in exactly C's order.

- **R3a. RNG draw counts must match C exactly.** Every `number()`/`dice()`
  call in a C code path must have a matching draw in Go, in the same order.
  (Taught by fight messages: `skill_message`/`dam_message` draw counts desync
  everything downstream if they differ.)
- **R3b. Operation ordering is behavior.** Zone reset: C runs `initRare`
  *before* percent loads, and sets `last_cmd` only on success. Go had both
  inverted → a +2 draw offset that looked like unrelated combat divergence.
  When porting a loop, port its *order*, not just its effects.
- **R3c. Time flows through the seam.** Real-time pulses go through DP_CLOCK
  (and DP_FIXED_TIME for the game clock) so scenarios are reproducible. New
  time-dependent code must respect the seam or it breaks the oracle for
  everyone.

## R4. No invention

If C doesn't print it, Go doesn't print it. If C doesn't do it, Go doesn't do
it — on the player-facing surface. Modern additions (GMCP, agent vars, web
admin, persistence) are allowed **only** where a player at a telnet prompt
cannot observe them.

- **Taught by:** every re-skin found so far — nanny menus, recall messages,
  hunger/thirst drift. "Plausible MUD behavior" is the enemy; the C source is
  the only authority.
- **Comply:** fidelity briefs must carry the `**Cite:**` field with the exact
  C function name. A port PR that can't cite its C source isn't a port.

## R5. Process rules

- **R5a. Oracle proof gate.** A fidelity fix is done when its oracle scenario
  runs green, not before.
- **R5b. Repeat reds indict rules.** The second time a red has the same root
  cause, stop fixing files: amend this rulebook and audit the class.
- **R5c. Find one, find the class.** Every confirmed finding triggers the
  question "what else is in this class?" — answered with a grep/script, not a
  feeling. Prefer making the audit deterministic and rerunnable
  (`scripts/gen_reachability.py` is the model).
- **R5d. Flag uncertainty at the site.** Low-confidence translations get
  `// TODO(port): <what's unverified>` so later passes can queue them
  mechanically.

---

## Amendment log

| date | rule | incident |
|---|---|---|
| 2026-07-22 | R1–R5 seeded | July fidelity sprint (DP_CLOCK, zone-reset, nanny, recall) + reachability findings (DP-1185/1186/1187) |
