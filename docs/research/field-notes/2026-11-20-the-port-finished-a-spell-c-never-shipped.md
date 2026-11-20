# The port finished a spell C never shipped

2026-11-20. Field note by Claude, from landing DP-1609 (PR #2912, main `621c882dd`).

## What C has

`src/spells.c:883-912` holds a complete spell body, `spell_silken_missile`. It
turns a worn item into an arrow:

- C's message is `"You create an arrow from $p."`;
- `src/spells.h:373` declares the function.

Nothing else in C refers to it:
- no `SPELL_` constant;
- no `spello()` registration;
- no arm in the manual-spell dispatcher (`src/spell_parser.c:517-530`).

A player who types the spell reaches the unknown-spell return
(`src/spell_parser.c:978-988`). The original authors wrote the spell and never
wired it up. To players it never existed.

## What Go had

Commit `6b3dd9778` (2026-04-26, "Full implementations: spell_coc and
spell_silken_missile") wired it up anyway:

- it invented spell ID 200;
- it added a manual-dispatch arm;
- it ported the body with a fixed `"You create an arrow from it."` in place of
  C's `$p`.

The constant's own comment admitted the invention: `SpellSilkenMissile = 200 //
DP original — not in C spell_parser spello table`.

Nothing ever dispatched ID 200, so no player could reach it. It was dead
invented code with one Go-only line of text.

## Who looked and missed it

**A code review saw it and asked the wrong question.** Daeron's 2026-05-10
review (`e83c5d105`) quoted the "DP original" comment. It asked only whether ID
200 might collide with breath-weapon IDs, and suggested moving "DP custom
spells" to 500+. It treated the invention as a feature to protect rather than
an R4 violation to remove.

**The string census couldn't see it.** The Go text contains C's segment
`You create an arrow from `, so it matched as covered. This is cause C5 in
PF-142.

**The oracle couldn't reach it.** Both engines reject the cast, so every
scenario is CLEAN.

**What finally found it:** Luna's randomly drawn Round 2 mutation sample (row
43). It picked the literal, the mutant stayed green, and tracing why led to
the dispatcher. The fix deletes the ID, the dispatch arm and the body. The
revert proof is a unit test, `TestUnregisteredSilkenMissileIDHasNoManualDispatch`,
because no transcript can reach dead code.

## Why it's a different specimen from `yank`

`yank` (DP-1214) was slop as **paraphrase**: a real command reworded into
something plausible. Silken missile is slop as **completion**: the porting
pass found an unfinished feature in the source and helpfully finished it.

Both look like diligence. "Full implementations" reads as progress, and the
body is a faithful port apart from one pronoun. But R4 judges what players
saw, not what the source contains. Shipping the spell would have invented a
spell that Dark Pawns never had.

The oracle can't catch this class, because C's behaviour is "nothing happens",
and Go also did nothing until something dispatched ID 200. Only a method that
samples code rather than transcripts finds it.

## Limits

- One instance. The DP-1609 R5c audit
  (`docs/fidelity/depth/evidence/2026-11-20-dp1609-r5c.md`) searched for more
  orphan C bodies wired up in Go only on the spell surface. Other surfaces are
  unaudited.
- The author of `6b3dd9778` (an earlier AI pass, run under Zach's account)
  isn't recorded beyond the commit.
