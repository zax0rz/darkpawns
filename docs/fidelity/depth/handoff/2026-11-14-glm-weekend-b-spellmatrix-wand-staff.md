# Handoff 2026-11-14 — glm (weekend bucket B): CAST_WAND / CAST_STAFF matrix

State of `docs/fidelity/depth/surface-inventory.tsv` at handoff time, and
what remains to close the two spell-matrix rows.

## What this round did

- Audited all 124 gap-reachable wand/staff cells against today's manifests
  (`docs/fidelity/evidence/dp-1456/witness.tsv`, `blocked-no-witness.tsv`,
  `unstable-scenarios.tsv`, `cast.tsv` case rows). Every cell cites a
  post-DP-1456-repair scenario whose case row is oracle-green; none of the
  124 needed a new scenario built from scratch.
- Opened six PR branches that flip the 41 wand cells whose scenarios are
  NOT in `evidence/dp-1456/unstable-scenarios.tsv` (glm/spellmatrix-wand-a
  through -e; a census backs each branch), plus glm/spellmatrix-staff-exclude
  which classifies (59, staff) Enchant Armor as excluded-with-C-reason
  (C's staff fan-out passes a NULL object: src/spell_parser.c:602,605;
  spell_enchant_armor returns on it: src/spells.c:837 — same R1a class as
  the (31, staff) Locate Object allowlist entry).
- Built and retained `spec-proc-combined-guard-arms` (DP-1507 cityguard
  half): three combat-engine defects fixed on `glm/weekend-b`; the vehicle
  still diverges at 3 of 5 seeds deeper in the exchange — see the PR body.

## What remains

### 1. The staff cells (80 of 82) are blocked on scenario stability

`evidence/dp-1456/unstable-scenarios.tsv` keeps spellmatrix-staff-olc-01
through -06 and -08 through -16 (15 of 16 vehicles; only staff-olc-07 is
clean) plus wand-olc-08 and potion-olc-03 on the cadence-flaky ledger. The
recorded instability: "the first use block is stable, the following
~dpclock pulse 20 combat block varies (multi-target fan-out order and
outcomes); reproduces only under 36-way census parallelism" — C itself
deterministic locally. Two consequences:

- Targeted (low-parallelism) censuses of these vehicles run clean, but a
  clean targeted run does NOT make the cells flip-safe: the release gate is
  36-way, which is exactly the condition that destabilizes them. Flipping
  on targeted evidence alone re-opens at the gate (R5h).
- The durable fix is a scenario redesign ("stagger"): make each use block's
  fan-out set deterministic so no ordering can vary.

### 2. The stagger design (worked template, per use block)

Current block: oedit 8054 (set spell value N) → load → get/hold staff →
`use staff` → `~dpclock pulse 20` → scores → look. With three peers in the
room the fan-out set is [Peer, View, Builder] and C's iteration order over
`world[].people` varies under load. Redesign:

1. During warmup keep the peers, but before EACH use block move the room to
   a single designated target: `teleport ItemView 3001` / `teleport
   ItemBuilder 3001` out, leaving ItemPeer as the only potential victim.
2. `use staff` (fan-out set = actor + ItemPeer; actor is skipped by C's
   staff loop's `ch == tch` guard, src/spell_parser.c:599-600), pulse,
   scores, look.
3. Re-edit the prototype for the next spell with the room composition
   unchanged — the earlier block's combat (peer retaliates for 20 pulses)
   is 1-vs-1 and its order cannot vary.
4. Spells that move the TARGET (shift-reality/teleport in staff-olc-01)
   must re-teleport the designated target back into 8102/8162 before the
   next block, or later blocks hollow out (the #2269 class).
5. Keep the witness requirement per cell: the spell-specific line must
   appear in C's block (the DP-1456 witness.tsv standard), not only the
   generic glow line.

Per-scenario spell sets (from the scenarios' oedit blocks): staff-olc-01 =
shift-reality, blindness, burning-hands, call-lightning, charm;
staff-olc-02 = chill-touch, color-spray, control-weather, create-water,
cure-light; staff-olc-03 = curse, detect-alignment, detect-invisibility,
detect-magic, detect-poison; staff-olc-04 = enchant-weapon, energy-drain,
fireball, harm, heal; staff-olc-05 = lightning-bolt, magic-missile, poison,
protection-from-evil, remove-curse; staff-olc-06 = shocking-grasp,
strength, summon, meteor-swarm, word-of-recall; staff-olc-08 = group-recall,
mass-heal, lycanthropy, vampirism, sobriety; staff-olc-09 = group-
invisibility, hellfire, mindpoke, mindblast (identify staff0903 stays
blocked on DP-1496 — leave alone); staff-olc-10 = mindblast?? (see file),
chameleon, levitate, metalskin, globe-of-invulnerability; staff-olc-11 =
vitality, invigorate, lesser-perception, greater-perception, mindattack;
staff-olc-12 = adrenaline-boost, psychic-shield, change-density, dominate,
cell-adjustment; staff-olc-13 = mirror-image, divine-intervention, mind-bar,
soul-leech, mindsight; staff-olc-14 = transparency, know-alignment,
lay-hands, mental-lapse, disintegration; staff-olc-15 = calliope,
protection-from-good, flame-strike, haste, slow; staff-olc-16 =
dream-travel, psiblast, glyph-of-summoning, drowning.

Order of work: staff-olc-01 first as the pattern proof (it contains the
teleport trap and a damage spell, so it exercises both template hazards),
then the rest in vnum order. After each scenario: targeted census
(--draw-parity, seeds 1,2,3,5,8), confirm CLEAN, flip that scenario's cells
(staff units step 81 → 76 → ...), and keep the 36-way caveat in the PR
until Zach has run the release-gate census on the staggered vehicles.

### 3. Two world vehicles with conflicting evidence

`spellmatrix-world-wand-create-water-depth` and
`spellmatrix-world-staff-acid-blast/-cure-critic/-dispel-evil/-earthquake-
live-depth` appear in BOTH witness.tsv (with C spell-specific lines) and
blocked-no-witness.tsv ("item never reached the actor; spawn-obj/no-settle
class"). The files disagree; only an empirical run settles them. The
glm-wand-d census includes create-water; if it comes back CLEAN with the
golden-chalice line present in C's block, cell (13, wand) can flip in a
follow-up; the four world-staff vehicles belong to the stagger queue above
(or a spawn-obj brief per the blocked-no-witness note).

### 4. Leave alone

- `spell.identify.matrix-staff-olc` (staff-olc-09): run-varying C age line,
  DP-1496, Zach's ruling pending. The cell stays gap-reachable so the
  CAST_STAFF inventory row cannot reach zero until it resolves.
- `spell.drowning.matrix-staff-olc` (103): unused-spell body; its scenario
  is ledger-stable but the cell's own classification (unused, unreachable
  from do_cast, staff fan-out calls it with spell 103) needs a ruling the
  same way as the Identify row if it cannot witness.
