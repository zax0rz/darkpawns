# Surface-inventory handoff — 2026-09-03

## Baseline and method

This enumeration starts from merged `main` at `566eabb26` after the blocked
clinic. The depth manifest baseline is 4,111 total cases: 4,014
proven/delegated, 46 blocked, and 51 excluded. The existing untracked
`website/static/images/` directory was preserved.

`docs/fidelity/depth/surface-inventory.tsv` is a separate, deliberately
conservative denominator. It contains 69 auditable family rows and 4,925
weighted units:

- 50 source-file activity families covering 2,954 mechanical
  `act()`/`send_to_char()` call sites in `src/*.c`;
- five spell-vector rows covering the complete 299-spell cross-product for
  `CAST_SPELL`, `CAST_POTION`, `CAST_SCROLL`, `CAST_WAND`, and `CAST_STAFF`
  (1,495 vector cells);
- four fight/skill families covering 463 counted message/call-site units;
- seven lifecycle families covering ten state/pulse units; and
- three shop families covering the core, special-procedure, and storage
  surfaces.

The activity count is reproducible with the repository's literal call forms:
`rg -o '\\b(act|send_to_char)\\s*\\(' src/*.c`, grouped by source file. The
spell count is anchored to `TOP_SPELL_DEFINE 299` in `src/spells.h`. Skill
message count is the 82 `M` blocks in `lib/misc/messages`.

## Classification

One row is `proven-already`: the `src/act.social.c` family, owned by the
exhausted socials and social-command inventories. The other 68 rows are
explicitly `unproven`. This is intentional: existing focused proofs cover
selected handlers, spells, specials, and combat seams, but do not justify
promoting an entire source-file family or a full 299-spell vector matrix.
There are zero `excluded-with-C-reason` rows, so this inventory introduces no
unverified exclusion. Known exclusions remain in their owning depth manifests
with their C call-path reasons.

## Standard-protocol work order

The unproven rows are ordered first by C source-file path, then by the matrix
and subsystem classes requested by the objective. Each row records its exact C
scope and the reason a focused proof is still missing. The first-pass work
therefore performs the required source census and ownership check without
inventing oracle vehicles or treating partial command proofs as whole-family
proof. The next safe work items are the first unproven activity families in
file order (`act.comm.c`, `act.display.c`, and `act.informative.c`), followed
by the spell vectors; their statuses must remain `unproven` until independent
D1–D5 vehicles cover the residual branches.

This handoff applies R1/R2/R3/R4/R5b/R5c/R5e: count player-visible bytes,
preserve the configured dispatch and draw surface, do not invent coverage,
delegate shared owners, and verify the actual C call path.

