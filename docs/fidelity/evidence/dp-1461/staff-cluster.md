# DP-1461: the run-varying staff cluster

Linear: DP-1461. Investigated 2026-11-07 against `origin/main` b7fdd6ef5
(worktree `glm/dp-1461-staff-cluster`). Five `spellmatrix-staff-olc-*`
scenarios were pinned `run-varying`; the brief asked why their first
divergent block moves from run to run.

## TL;DR

The pins were stale at current main. The five scenarios are now
**deterministic** at seed 1 (three runs each, identical fingerprints):

| Scenario | Seed 1 (pre-fix) | Draw parity | Root cause | Post-fix |
|---|---|---|---|---|
| staff-olc-08 | green ×3 | MATCH | — (pin stale) | green, seeds 1,2,3,5,8 |
| staff-olc-09 | red ×3, one block | MATCH | identify age line (harness gap, below) | **stays red** |
| staff-olc-12 | green ×3 | MATCH | — (pin stale) | green, seeds 1,2,3,5,8 |
| staff-olc-15 | green ×3 | MATCH | — (pin stale) | green, seeds 1,2,3,5,8 |
| staff-olc-16 | red ×3 | MISMATCH at draw 4348 | zen HP assignment (fixed) | green, seeds 1,2,3,5,8 |

Two port bugs were found and fixed; one divergence is a harness
determinism gap that the port cannot fix at the source (reported for
filing).

## 1. Measuring the variation

Method: the rehearsal run
(`~/Archives/darkpawns/oracle-runs/2026-11-07/claude-rehearsal-1107`,
a full combined census of main at b7fdd6ef5 with draw parity) plus three
targeted runs of each scenario at seed 1, then one run per remaining
census seed (2, 3, 5, 8) with the fixed binary.

At seed 1 every scenario reproduced identically on all attempts — no
first-divergent-block movement. `draw_divergent` (the draw-parity line)
reported:

- 08: `MATCH entries=4217 stream=4217`, transcript green.
- 09: `MATCH entries=4344 stream=4361` (17 dice calls consumed >1 value),
  transcript red in exactly one block: `use staff0904 [actor]`.
- 12: `MATCH entries=4217 stream=4223`, transcript green.
- 15: `MATCH entries=4650 stream=4884`, transcript green.
- 16: `MISMATCH stream=4390 c=Dice(1,4) = 3 go=Number(1,20) = 15`,
  both steps `probe step 88: ~dpclock pulse 20` (the pulse after
  `use staff1605`, the zen staff).

So nothing here is run-varying *today*; the pins describe an older main.

## 2. staff-olc-16: zen assigns the caster's HP from the victim

Trace (R5e). The live `use staff` route is `mag_objectmagic`'s "castable
equipment" staff branch — `src/spell_parser.c:784-813` — not the
`equipd==WEAR_HOLD` branch at `:579-608` (that one fires only when the
command's *target argument* names an item equipped at wear position 16,
because `equipd = wear_pos+1` is compared against `WEAR_HOLD` = 17; an
off-by-one in the original). The castable-eq branch fans out over
`world[ch->in_room].people` — the one shared list that
`char_to_room` **prepends** to (`src/handler.c:537-538`), so the newest
arrival is hit first and players and mobiles interleave — and skips the
user. Mass/area spells (`:803-804`) get a single `call_magic` with a
NULL target instead; of this cluster only hellfire (09) is area-routined.

The room holds four players: ItemMage (user), ItemBuilder, ItemPeer,
ItemView — arrival order Mage, Builder, Peer, View, so C's fan-out hits
View, Peer, Builder. Psiblast (staff1602, spell 100, `MAG_DAMAGE`)
starts a four-way fist fight (damage() → set_fighting, `src/fight.c:
1441-1445`). Draw parity holds through the fight; the mismatch lands at
the pulse after `use staff1605` — zen, spell 78.

C's `spell_zen` (`src/spells.c:871-879`) does, per fan-out target:

```c
GET_POS(ch) = POS_STUNNED;
GET_HIT(ch) = MIN( GET_MAX_HIT(victim), GET_HIT(victim)+(2*GET_LEVEL(ch)) );
```

The **caster's** hit points are *assigned from the victim*, capped at
the *victim's* max. ItemBuilder is level 35 — a `set player ... level`
reset leaves him at **21/21** HP, and as an immortal psiblast never
damaged him (`dam = 0`, `src/fight.c:1477-1478`), so he sat at 21/21 as
the last fan-out target. C leaves the mage at `MIN(21, 21+60) = 21` out
of 5000. C witness (clone run with `send:builder stat ItemMage`
inserted): `Hit p.: [21/5000+10]`, then in the next pulse
`ItemPeer hits ItemMage.` → `ItemMage is stunned, but will probably
regain consciousness again.` → `ItemPeer punches ItemMage in the kidneys
causing immediate death!` → `ItemMage is dead! R.I.P.`

The port (`castZen`, `pkg/spells/affect_spells.go`) instead healed the
caster additively capped at the caster's own max: the mage stayed at
~4880/5000, survived the round, and kept fighting — the position
multiplier, death, and every draw after it diverged (the mismatched
`Dice(1,4)` is a mob damage roll from a later combat pair).

**Fix.** `castZen` now stuns the caster and assigns
`MIN(victim.MaxHP, victim.HP + 2×caster.Level)` (the caster's own level,
per DP-1468's rule that `GET_LEVEL(ch)` is the caster's level, not the
cast level). On the registered self-cast path (victim == caster) this
degenerates to the old plain heal, so `TestZenItemCastHealsWithCasterLevel`
still passes unchanged.

**Test / revert proof (R5h).** `TestZenStaffCastAssignsCasterHPFromVictim`
(`pkg/spells/caster_level_test.go`). Reverting the fix to
`healHP(ch, 2*getLevel(ch))`:

```
--- FAIL: TestZenStaffCastAssignsCasterHPFromVictim (0.00s)
    caster_level_test.go:138: zen staff-cast HP = 4940, want 21 = MIN(21, 21+2*30) assigned from the victim (src/spells.c:878)
```

With the fix, `spellmatrix-staff-olc-16-depth` is green with
`draw-parity MATCH` at seeds 1, 2, 3, 5, 8.

**Witness for the fan-out order.** A throwaway probe scenario (not
committed) zen-staffed a level-40 user against three victims teleported
in as A, B, C with HPs 21/22/24. C left the user at **101 = 21 + 2×40**:
the last fan-out target is the *oldest* teleport, confirming the
newest-arrival-first people-list order for the staff fan-out.

## 3. The fan-out order class (R3b) — latent, fixed at the source

The brief's first suspect — room-order fan-out — was **not** what turned
any of the five red (their rooms hold only players, and Go's
`GetPlayersInRoom` already sorts newest-first). But the Go fan-out
`useStaff` walks `w.actChar()` (`pkg/game/other_economy.go:431`), which
returned **all players first, then all mobs in Go map order**
(`GetMobsInRoom` walks `w.activeMobs` — randomized per process). Any
staff use in a mixed room was order-divergent *and nondeterministic* —
exactly the run-varying signature the pins describe, and a live R3b
violation.

**Fix.** `actChar` now merges `GetPlayersInRoom` + `COrderedRoomMobs` on
the shared room-entry sequence (newest first, name tiebreak), matching
C's single `people` list. This also feeds the other `people`-walk
callers (say-spell broadcasts, mobact room scripts, movement echoes),
so the class is fixed at the source, not per-caller.

**Test / revert proof (R5h).** `TestActCharUsesCPeopleOrderAcrossBodyTypes`
(`pkg/game/character_order_test.go`): arrivals P1, M1, P2, M2 must yield
`[M2, P2, M1, P1]` (test names: `[mobB, Second, mobA, First]`). Reverting `actChar` to
players-then-mobs:

```
--- FAIL: TestActCharUsesCPeopleOrderAcrossBodyTypes (0.00s)
    character_order_test.go:153: actChar order = [Second First a probe a probe], want newest-first interleave [mobB, Second, mobA, First]
```

## 4. staff-olc-09: identify's age line is real-time quantized (not fixable port-side)

The only divergent block is `use staff0904 [actor]` — identify, spell
60, fanned out on the staff. The diff is one line:

```
-ItemPeer is 17 years, 0 months, 0 days and 1 hours old.
+ItemPeer is 17 years, 0 months, 0 days and 0 hours old.
```

`spell_identify` prints `age(victim)` (`src/spells.c:596-600`), and
`age()` is `mud_time_passed(time(0), birth)` (`src/utils.c:374-383`)
with `SECS_PER_MUD_HOUR = 63` (`src/utils.h:135`). The oracle's
`time(0)` is the **real wall clock** — `DP_FIXED_TIME` pins only the
game calendar (`reset_time`, oracle `src/db.c:419-433`) — and so is the
port's (`Age()` calls `time.Now()`, `pkg/game/time_utils.go:79`, like C).
Between ItemPeer's creation and the identify cast, the `-O0` C oracle
has burned ≥63 s of real time (→ "1 hours") while the Go port has not
(→ "0 hours"). Both engines are individually faithful; the byte class
quantizes real elapsed time across two processes of different speed, so
it can flip run to run with host load. Draw parity is a full MATCH here.

This is a harness determinism gap, not a port bug: any port-side seam
(wall-time pinning, age normalization) would *systematically* diverge
from C's real-time behavior. Filing recommendation for Claude: seam
`time(0)` in the C oracle (advance with `~dpclock` pulses like the Go
port's DP_CLOCK) through the reference-binary promotion process, then
mirror it in the port; until then the pin stays. staff-olc-09's five
rows stay `blocked` with their pin.

## 5. Scenarios and pins

- `spellmatrix-staff-olc-08/12/15/16-depth`: pins removed (STALE — the
  divergences no longer reproduce), rows moved to `oracle-green-multiseed`
  (green + draw parity on seeds 1, 2, 3, 5, 8), with C witness lines in
  the manifest notes.
- `spellmatrix-staff-olc-09-depth`: stays red, first divergent block
  `use staff0904 [actor]`, identify age line (§4). Pin left in place.

## 6. Census set (named)

Every `spellmatrix-*staff*` scenario (26:
`spellmatrix-create-food-staff-fanout-depth`, `spellmatrix-staff-olc-01`
through `-16`, `spellmatrix-world-staff-{acid-blast,cure-critic,
dispel-evil,earthquake-live,gate-live,ray-of-disruption,sense-life-live,
sleep,smokescreen}-depth`) plus every scenario that `use`s a staff:
`spellmatrix-itemroute-infravision-depth`, `staff-use-depth`,
`wand-staff-use-depth`. (Excluded after checking: `spec-proc-horn`
uses a type-11 horn, `egon-equipment-apply-spell` only wields, the
`wand-olc` scenarios use ITEM_WANDs.)

Run directory: see the PR description.
