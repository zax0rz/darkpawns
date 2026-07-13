# BRIEF 2026-07-13 — Session-vs-game audit, Tiers B & C (continuation)

**Executor:** ChatGPT / frontier (same deep three-sided read-and-report as Tier A; rate limit lifted
today, so finish the full command surface). **AUDIT, not fixes.** Claude verifies + files.
**Branch:** `docs/session-vs-game-audit-bc` off current `main`. **One PR** extending the research doc
(or a sibling doc — your call; keep the format identical).
**Read first (your own Tier-A work is the template):**
`docs/research/drafts/2026-07-13-session-vs-game-audit.md` and the Tier-A brief
`docs/briefs/BRIEF-2026-07-13-session-vs-game-audit.md`. Same method, same buckets, same deliverable
shape. This brief only says what's *different* for B/C.

---

## Goal

Complete the picture: audit the **remaining player commands** in `pkg/session/commands.go` beyond
Tier A, so we have the full session-vs-game-vs-C divergence map and can build the fix roadmap.

## Scope — the two tiers

Enumerate from `pkg/session/commands.go` (the registry is ground truth) everything NOT covered in
Tier A (Tier A = the informative + object/container set: look, examine, time, weather, get, put,
give, pour, drop, wear, remove, wield, hold, open/close/lock/unlock, exits, score, inventory,
equipment, who, where, consider, drink, eat, fill).

- **Tier B — everyday play (do first):** movement & positioning (`north`…`down`, `enter`, `follow`,
  `sit`/`stand`/`rest`/`sleep`/`wake`, grouping/`group`/`gtell`/`follow`), communication (`say`,
  `tell`, `emote`, `shout`, `whisper`, `ask`, `reply`, `gtell`, and the socials pathway), and misc
  everyday utility (`quit`, `save`, `title`, `description`, `toggle`/prefs, `color`, `diagnose`,
  `consider` is already done). Pair each with `pkg/game` equivalents (e.g. movement lives in
  `pkg/game/act_movement.go`; comms may reimplement in session) and the C command/handler.
- **Tier C — the rest:** combat commands (`hit`/`kill`/`flee`/`assist`/`rescue`/`bash`/`kick`… —
  **structural only**: command existence, gating, position, message text, targeting grammar; NOT
  RNG outcomes), skill/spell invocation (`cast`, `practice`, skill commands — again structural:
  does it exist, level/position gating, learn/practice messages, NOT roll results), and
  **immortal/wiz commands** (flag these as a **separate, lower-priority sub-section** — they're not
  player-facing fidelity, but privilege/gating divergences like the O11 `where` leak ARE worth
  noting since they're security-relevant).

**Explicitly out of scope (same as Tier A):** RNG-outcome parity for combat/skills/spells — that's
the oracle's Tier-2 (needs the `random.c` port). Structural issues in those commands ARE in scope.

## Same framework, format, guardrails (see Tier-A doc)

- **Three buckets** per command: (1) reimplemented-and-drifted, (2) delegates-but-game-buggy,
  (3) session-only/missing. Cite **file:line on all three sides** (session / game / C).
- **Deliverable:** extend the divergence table + new O-findings in the C:/Go-session:/Go-game:/
  Divergence:/Impact:/Fix-sketch format. **Continue the O-number sequence: start at O24.**
- **Read-and-report only**, doc-only PR, no code changes. `go build/vet/test` trivially green.
- Claude files the Linear issues — **do NOT re-file**. Dedup against the full filed set now:
  **DP-1083..DP-1093 (O1-O9)** and **DP-1094..DP-1107 (O10-O23)**. Reference, don't repeat.

## Carry forward two systemic threads from Tier A (watch for more instances)

1. **Session reimplements a `pkg/game` function and they've drifted** — often BOTH copies are
   incomplete vs C, so the fix is "consolidate to one canonical game op + dual WS/telnet renderers,"
   not a blind reroute. Movement is a prime suspect (session `wrapMove` vs `pkg/game/act_movement.go`).
2. **Prototype-mutation data corruption** — writing `item.Prototype.Values` (or any shared prototype
   field) mutates every instance of a vnum. Flag any B/C command that mutates prototype state instead
   of instance state (`GetValue`/`SetValue`, `pkg/game/object.go`).

## One value-add for the fix roadmap: tag oracle-provability

For each new finding, add a one-word **oracle-provability** tag so the fix phase knows how to *prove*
the fix (fixes will be gated on a red→green oracle-diff run, not just code QA):
- **`tier1`** — deterministic (text/structure/gating); provable now with a `cmd/dp-oracle-diff` scenario.
- **`tier2`** — depends on RNG-outcome parity; needs the `random.c` port before it's oracle-provable.
- **`manual`** — needs live/interactive verification the current harness can't script.

## Success criteria
1. Tier B fully audited + classified with three-sided citations; Tier C at least structurally swept
   (combat/skills structural + a flagged immortal-cmd sub-section).
2. New findings in O-finding format starting at O24, each with a bucket + oracle-provability tag.
3. Systemic-thread instances (reimplementation dups, prototype mutation) called out explicitly.
4. Dedup list honored (no re-filing DP-1083..1107).

## Wrap-up
Commit the doc; push; open PR; STOP — Claude reviews against `origin/main` + `src/*.c`, spot-verifies,
and files the confirmed findings. After this lands we draft refactor plans, then fix blocks — each
fix block **proven by oracle runs** before sign-off.
