# C-claims audit — 2026-11

Every Go comment on `origin/main` that makes a negative or behavioral claim
about what C does, checked against the C source. Audit only: no code was
changed. Brief: `docs/briefs/BRIEF-2026-11-28-toh-seal-and-c-claim-sweep.md`.
Motivation: PF-192 (`docs/research/field-notes/2026-11-28-faithful-to-a-bug-that-wasnt-there.md`)
— a comment invented a C omission and told readers not to fix the port's bug.

## Patterns used

The brief's base pattern, then the widened variants, then one more sweep for
`C <word> <negative-verb>` shapes the first two miss (that is the sweep that
catches the Toh comment itself):

```
git grep -n -iE '//.*\bC (has no|never|does not|doesn.t|lacks|forgot|omits)' -- 'pkg/**/*.go'          # 86
git grep -n -iE "//.*\bC('s own bug| bug| quirk| also| skips| ignores| always)|//.*as C does" -- 'pkg/**/*.go'  # +46
git grep -n -iE '//.*\bC [a-z_]+ (has no|hasn.t|never|does not|doesn.t|lacks|omits|skips|ignores|forgot)' -- 'pkg/**/*.go'  # +20
```

152 unique comments (`c-claims.tsv`, one row each: `go_site`, the claim
quoted, the C lines that settle it grepped this session (R5g), `verdict`,
and for `false`/`partly` what Go does differently because of it).

## Counts per verdict (awk-checked against the TSV)

```
$ awk -F'\t' 'NR>1 {print $4}' c-claims.tsv | sort | uniq -c
      1 false
      4 partly
    146 true
      1 unverifiable
$ awk -F'\t' 'NR>1' c-claims.tsv | wc -l
152
$ awk -F'\t' 'NR>1 && NF!=5' c-claims.tsv | wc -l
0
```

## The `false` row (code wrong, not just the comment)

- `pkg/game/skill_berserk_kuji.go:126` — "the original C switch has no case
  for SKILL_KK_TOH … the Toh seal's success check always fails. Ported
  faithfully; do not 'fix'." C has the case: `src/new_cmds.c:1514-1516`
  (`case SKILL_KK_TOH: prob = GET_SKILL(ch, SKILL_KK_TOH); break;`).
  **Reach:** every ninja's Toh seal failed its success check for five months
  (the switch left `prob` at 0, so the 1-101 roll always exceeded it) — a
  player-facing, census-visible divergence on a live command (`toh`,
  `src/interpreter.c:781`). Fixed in the sibling PR (`kimi/toh-seal-fix`):
  the case is added in C's order, the comment replaced with a correct
  citation, and the fix is proven by a unit test and a `toh-depth` census
  scenario that is red on `origin/main` and green on the branch.

## The `partly` rows (comment wrong in detail, code right)

- `pkg/admin/olc_entity_write.go:220` — the re-prompt and vnum-0 claims are
  true, but the quoted message is a splice of two per-kind C strings
  (`src/zedit.c:1052,1063`), not a C literal.
- `pkg/game/look.go:329` — the MOB_EXTRACT skip is real but lives in
  `list_one_char` (`src/act.informative.c:508`), called after
  `list_char_to_char`'s CAN_SEE gate — not "before visibility checks".
  Behaviorally inert.
- `pkg/game/skill_known_gate_test.go:100` — "C has no class gate on skill
  USE (only on learning)" holds for the audited warrior skills but
  overgeneralizes: `do_kuji_kiri` class-gates use (`src/new_cmds.c:1557`).
- `pkg/game/spec_act_templates_test.go:6` — true for the two bodies the test
  pins (enter_circle, elevator), false as a blanket statement about
  spec_procs4: 12 of its specials are assigned in `src/spec_assign.c`.

## The `unverifiable` row

- `pkg/game/character_command.go:26` — a port-coverage statement about the
  enum ("C behavior has not been ported for that actor"), not a claim about
  C's behavior; nothing to check it against.
