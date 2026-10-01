# Vampire consumption proof evidence

Scope: `drink.vampire` and `eat.vampire`, plus day/sunset/dark audience vehicles.
Base `065c2e85d528a8f9270d90702da700bebed8d75d` (fresh origin/main after #1740).
No production Go, harness, oracle, shipped world or governing document changes.

`TestVampireConsumableDepth` runs 128 state/output combinations: four sunlight
bands, independent PLR/AFF vampire bits, drink/sip with water/beer/blood, and
eat/taste. It checks exact actor bytes, condition deltas, liquid/weight and
food consumption, unchanged tattoo cooldown, prototype isolation, and draw-call
counts/bounds with a controlled return value. This proves the consumable draw
boundary, not equality of the entire live RNG stream. Poison, general lookup,
condition entry gates and shared observer visibility retain their existing
manifest owners.

`revert-triples.tsv` records 18 assertion-only 0/1/0 triples. `revert_proofs.py`
reproduces them; stage logs and restored source SHA are retained under
`~/Archives/darkpawns/oracle-runs/2026-10-01/dp-1371-p4-vampire-proofs/`.
An initial drink-drunk mutation caused an unused-local compilation failure;
it was rejected, replaced by a compiling arithmetic mutation, and all triples
rerun. No build failure counts as proof.

The live vehicles load shipped bread 8010 and beer glass 4104, use Dracula's
real look/bite to set PLR_VAMPIRE, and compare actor/observer outputs before
and after. Night clocks advance while only the immortal exists; the mortal
observer is created afterward. The drink amount is arithmetic, with the
one-serving container clamping it. Full hidden consumption state belongs to
the unit matrix. The five-seed runs and live gate mutations are retained via
`oracle_proofs.py` and `scripts/census.sh`. Every new seed claim's C capture is
checked for the actual bite and vampire branch, not only an empty diff.

Development evidence remains retained: `dp-1371-p4-vampire-initial`,
`dp-1371-p4-vampire-diagnostic`, and `dp-1371-p4-vampire-seed1`.
The first two had C observer EOF during bulk clock warmup (before consumption);
the first daytime fixture also lacked its intended starter items, which was
caught by reading C output and was never counted as proof. Explicit object
loads and clock advancement before observer creation repair those fixtures.

The water version of the third run exposed a repeated daylight water-drink
output discrepancy against unchanged origin/main: C printed the
thirst-satisfied message on the second drink while Go did not. No candidate
production edit was present (source hashes equal main). That live water/condition-state
question is retained as a follow-up; this PR claims the vampire gate,
not water RNG outcome parity. No seed, pin or divergence ledger was changed.
The final beer vehicle preserves all five seeds and makes the gate independent
of that draw. Final census results and capture checks are listed in the handoff.
