# DP-1371 Phase 4 E2: vampire drink/eat

Fresh base: `065c2e85d528a8f9270d90702da700bebed8d75d` (origin/main, #1740 merged).
Branch: `sol/p4-vampire-consume`. This is a proof-only batch: no production Go,
harness, reference oracle, shipped world, save format or governing-doc edits.

`drink.vampire` and `eat.vampire` are now `unit-green`, owned by
`TestVampireConsumableDepth`. Six additional day/sunset/dark audience rows are
`oracle-green-multiseed` at 1,2,3,5,8 (15 distinct scenario/seed pairs). The depth
report has 5,203 cases, 67 blocked; the unit census passes all 1,158 claims.

C authority (read during this batch, R5e/R5g):

- `src/interpreter.c:421,427`: drink/eat registrations; session `eat_cmds.go`
  and world `act_other_bridge.go:80-90` reach the tested live handlers.
- `src/act.item.c:953-1007`: drink versus sip, amount/draw, weight and DRUNK
  update before the PLR_VAMPIRE/non-blood/sunset-or-dark gate; FULL/THIRST and
  their messages only on the ordinary arm, with the final FULL message outside.
- `src/act.item.c:1022-1030`: depletion, empty-container values/name and puddle
  extraction; `:1108-1128`: eat/taste acts, vampire gate and FULL; `:1146-1154`:
  whole-food extraction versus decrementing taste bites.
- `src/spec_procs.c:1798-1833`, especially `:1827-1830`: real Dracula look/bite
  sets PLR_VAMPIRE. AFF_VAMPIRE transformation is not required.
- `src/weather.c:42-80`: clock progression to sunset/dark.
- `src/constants.c:967-984`: water/beer/blood condition multipliers.

The 128-combination unit matrix covers all four sunlight bands, independent
PLR and AFF flags, water/beer/blood, drink/sip and eat/taste, exact actor bytes,
condition deltas, consumption/weight, prototype isolation and unchanged tattoo
cooldown. Water/blood call number(3,8) once on drinking, beer uses arithmetic,
sip/taste consume no such draw. This certifies draw calls/bounds at the handler
boundary with a controlled return, not the whole live RNG stream. Shared
poison, lookup, entry conditions and general act visibility keep their owners.

Three live vehicles create a named observer, load bread 8010 and beer glass
4104, and probe before and after the actual Dracula bite. For night, the world
clock advances while only the immortal exists, then the mortal observer is
created. Captures show ordinary pre-bite consumption, the bite, the four
night vampire messages, no vampire messages by day, room audiences, consumed
food lookup failure and the empty glass. All 15 C captures were inspected and
checked, not inferred from a green diff (R1/R3/R5a/R5h).

Fail-capability: 18 unit assertion-only 0/1/0 mutation triples, plus live gate
triples for day, sunset and dark. Night-gate removal fails sunset/dark, and
making the gate apply by day fails the day vehicle. Restored runs pass. No
compile error, timeout or infrastructure failure is counted as a revert proof.
Scripts, tables, capture checks, source SHA and census lines are in
`docs/fidelity/depth/evidence/2026-10-01-vampire-consume/`; stage logs and oracle
captures are retained under `~/Archives/darkpawns/oracle-runs/2026-10-01/` with
`dp-1371-p4-vampire-*` run names and wrapper-generated manifests.

Validation: make fmt; build; vet; all Go tests; lint cache clean and lint (zero
issues); diff check; depth; units; string census (zero unreviewed segments).
Each gate ran independently and its exit status was checked. The three final
scenarios passed targeted censuses at every claimed seed. No full or claims
census is needed for this isolated test/docs batch under the goal's rules.

Development findings retained, with no new pin/ledger row or dropped seed:

1. The first-created immortal had no starter pack. An early green daytime
   capture therefore did not exercise consumption; it was rejected as proof
   and replaced by explicit shipped object loads.
2. A mortal observer closed its C connection during bulk clock warmup, before
   consumption (`run C oracle warmup: ... read observer ... EOF`). Moving clock
   advancement before observer creation removes that fixture failure. The
   initial and diagnostic NOT_CLEAN runs remain retained; they make no claims.
3. A water variant against unchanged main produced a second-drink byte
   difference: C printed "You don't feel thirsty any more." while Go did not
   (`dp-1371-p4-vampire-seed1`, reference oracle; both production source trees
   are identical to the fresh main base). The root cause is not established.
   Proposed follow-up: compare actual amount, FULL/THIRST state and stream at
   `src/act.item.c:959-962,993-1003` through this vehicle's spawn/load/setup.
   This PR uses beer's arithmetic amount to prove the vampire gate independently;
   it retains water/blood draw-call state checks and does not claim live water
   RNG outcome parity. No shared production repair is included.

Open the PR and stop. The next batch after merge is objmagic.sleep-entry-gates.
