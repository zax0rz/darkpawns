# 2026-10-03 — Where the defects were: review versus the differential census

## Scope

- **Tracks:** legacy-port fidelity; test-oracle validity; multi-agent software engineering
- **Window:** 2026-10-01 to 2026-10-03, DP-1371 Phase 4 (resolving the remaining blocked depth cases)
- **Primary artifacts:** merged zax0rz/darkpawns PRs #1748 (`9fa21c7e8`), #1749 (`1f7e32940`), #1751 (`3e28658bc`), #1752 (`adbceb768`), #1757 (`e8c757af5`), #1758 (`4591db664`), #1759 (`0f26c977f`), #1761 (`f0e8546bf`), #1763 (`45bd1994a`), #1764 (`42f878b49`), #1765 (`84476ab83`); review comments on #1748 and #1749; combined-census evidence under `~/Archives/darkpawns/oracle-runs/2026-10-0{1,2,3}/`.
- **Status:** artifact-grounded field note. The counts cover the PRs listed here, not a complete tabulation of every Phase 4 census verdict.

## Setup

During this window a coding agent (Sol) worked through the remaining blocked depth cases in "trains": several cases per branch, then one combined oracle census (every scenario at seed 1, plus every claimed scenario/seed pair, about 3,100–3,200 pairs) at the train's tip. Claude reviewed every train against the C source and ran an "other readers" audit: what else, including Go-only code, depends on the state the change touches. The census compares C and Go output byte for byte. It can only see paths that a scenario exercises, and only behaviour that has a C counterpart.

## What found the defects

**1. Go-only shared state (no C counterpart, so no census could see it).** Each was found in review, after a green or clean-after-recheck census:

- #1748: a pre-login WebSocket session claimed an entry name but was never unregistered on disconnect, so the claim was never released. Typing a character's name in the browser client and closing the tab locked that character out until restart. Telnet released correctly. Reproduced live, then fixed with a test that fails without the fix.
- #1749: deleting a character began retaining its row with C's `PLR_DELETED` flag (faithful to C). The Go-only admin panel login still checked only the password hash, so a deleted immortal kept builder or admin web access. The OLC and file-edit authorization reads had the same gap.
- #1751 (security track, not a fidelity train): wiring the dead audit logger would have started writing raw text typed at the name prompt, often a mistyped password, to disk. That's a data-handling defect introduced by enabling a correct component.
- #1758: the faithful `switch` port made `MovementLook` take the global lifecycle lock on every movement. No failure was observed. It was a design defect: it broke the "no change when nobody is switched" invariant, and it made deadlock safety depend on the lock state of every movement caller (combat flee, Lua, special procedures).

**2. C-parity defects on paths no scenario exercised.** Found by reader audits, not by the census:

- #1764/#1765: zone resets equipped mobs under C's `WEAR_*` slot indices (`src/structs.h:406`, `WEAR_WIELD = 16`), while Go's mob readers looked up the weapon under the Go player-slot enum (`SlotWield = 6`). Every zone-armed mob attacked with bare-hand verbs, lost blessed-weapon bonuses, and read as unarmed to the fighter, paladin and disarm logic. No census scenario fought a zone-armed mob. #1765 added one.
- #1765 pre-review: the mob loot-and-wear path skipped `equip_char`'s follow-up effects (zap messages, room light, bad-stat effects) and inverted C `perform_wear`'s order (wear message before equip, `src/act.item.c:1416`).

**3. Census non-clean rows.** Every one traced to harness or oracle infrastructure, or to known C nondeterminism, not to a Go regression:

- #1752: `wizard-zreset-depth@2` lost Go's `Reset world.` under 36-worker load, because a 300 ms quiescence window ended the capture before the first byte. Fixed with a 2 s first-byte allowance for the issuing connection.
- #1757: `mount-ridden-depth@5` (telnet port bind collision), `informative-residual-depth@2` (the `users` Login@ wall-clock second differed between engines), and `accuse-depth@3` (known run-varying C output). Each passed on an isolated rerun.

Local game-code trains #1759, #1761 and #1763 needed no review changes.

## Methodological observations

- **The differential oracle and review cover different failure classes.** The census proved C parity on exercised paths and caught no Go regression in this window. Every defect found was either Go-only (the census can't see it) or on an unexercised path (the census can't reach it). Neither check replaces the other.
- **Faithful changes create Go-only defects.** #1749 and #1758 were correct ports of C behaviour that broke assumptions in Go-only code built around the old behaviour: a row deleted on delete, a body owned by name. The risk sits next to the faithful change, not in it.
- **Enabling a dormant component is itself a change.** #1751's logger was correct, but wiring it exposed what its callers passed in. This is the "registered but never called" pattern from the other side.
- **Process consequence:** on 2026-10-03 Zach approved splitting trains into a "local" tier (Claude reviews and merges after a clean census) and a "stop" tier (sessions, lifecycle, identity, login, admin, persistence, transports, harness, tooling), based on where these defects had concentrated. See `~/GOAL-2026-09-30-dp-1371-phase4.md`.
