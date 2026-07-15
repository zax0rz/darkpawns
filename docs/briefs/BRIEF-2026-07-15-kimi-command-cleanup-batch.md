# BRIEF — Kimi cleanup batch (skill-system-independent, parallel to Phase 1)

**For:** Kimi (K2.7-code). **Gate owner:** Claude (runs every oracle gate; workers lack DP_ORACLE_BIN).
**Why now:** codex blocked to 07-21; these findings are orthogonal to the Phase 1 skill foundation and to
each other, so Kimi can land them while Claude builds the foundation. **One PR per section is fine**, or
bundle §1+§2 (both `pkg/game` cleanup) and keep §3 separate. **Rule:** never touch the C oracle; if a
change hits an unspecified decision, STOP and ask — do not improvise (esp. anything near the oracle).

Each section is a filed finding; read it in Linear for full context. C is cited — read the referenced
`src/*.c` directly, don't trust this brief's paraphrase over source.

---

## §1 — DP-1126: `title` truncates to first word + conflicting describe/description (O34)
**C `do_title`** (act.other.c:595-620), exact:
```
skip_spaces(&argument); delete_doubledollar(argument); delete_ansi_controls(argument);
IS_NPC            → "Your title is fine... go away.\r\n"
PLR_NOTITLE       → "You can't title yourself -- you shouldn't have abused it!\r\n"
has '(' or ')'    → "Titles can't contain the ( or ) characters.\r\n"
len > MAX_TITLE_LENGTH → "Sorry, titles can't be longer than %d characters.\r\n"  (MAX_TITLE_LENGTH from C)
else              → set_title(ch, argument); "Okay, you're now %s %s.\r\n" (GET_NAME, GET_TITLE)
```
- **Go bug:** stores only `args[0]` (first word) + skips all validation. Use the **full** argument
  (join, strip leading spaces, strip `$$`→`$` and ANSI controls), apply the 4 gates in C's order.
- **`describe`/`description`:** C has **no** such player command. Go registered two conflicting ones —
  **retire both** (remove registrations so they hit unknown-command), unless Zach flags a product reason.
- Gate: unit tests for each branch (exact strings, full-arg preservation, paren reject, length reject);
  Claude adds an oracle probe (`title Some Long Title` + verify `score`/`who` reflect it).

## §2 — DP-1160: social messages omit TO_SLEEP (O32 nit)
In `pkg/game/act_social.go` `DoAction`, two `act()` calls pass `ToChar` where C `do_action`
(act.social.c:141-144) uses `TO_CHAR | TO_SLEEP`:
- victim-position-fail `"$N is not in a proper position for that."`
- actor `char_found` message
**Fix:** change both to `ToChar|ToSleep` (Go's Act audience flags). A sleeping actor must still see these.
Gate: unit test asserting a sleeping actor receives both. Tiny.

## §3 — DP-1159: delete dead directed-speech duplicates
After 6a/6b consolidated onto `DoSpecComm`, these are dead but present:
- `pkg/game/comm_channel.go` `doWhisper` / `doAsk` (hard-coded ANSI, drifted wording — the O31 dup).
- Orphaned `Exec*` bridges that only routed to them (grep to confirm zero live callers).
**Fix:** delete them + any now-unused helpers. **Compile + full `go test ./...` is the safety net** —
if anything still references them, the build breaks (then STOP and report, don't rewire). Gate: build +
comms scenarios stay green (Claude runs `--scenario communication` + `communication-channels`).

---

## NOT in this batch (Claude will prep first)
- **DP-1115 `quit`/`reallyquit` (O33)** — needs an oracle RED capture + the safe-room list + equipment-loss
  contract mapped from C `do_quit`/`do_not_here` before it's safe to hand off. Claude establishes the RED,
  then it becomes a Kimi §. Don't start it speculatively.
- **DP-1133 color parity** — oracle-blind (normalizer strips ANSI); golden/unit-gated only, lower priority;
  now also has a combat-`DamMessage` sibling (CCYEL/CCRED). Bundle later as a dedicated color pass.

## Handoff notes for Claude (gating)
- Run each PR's oracle scenario yourself; Kimi's "unit-tests-green ≠ done" — verify the real diff.
- §1/§3 touch `pkg/game`; §2 too — if bundled, one branch `refactor/command-cleanup-batch`.
- GLM alternative: give GLM only §3 (pure deletion, lowest drift risk) if splitting workers.
