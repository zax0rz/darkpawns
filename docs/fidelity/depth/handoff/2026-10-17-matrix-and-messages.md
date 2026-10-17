# DP-1371 matrix and messages handoff — 2026-10-17

## Status

Work remains in two long-lived draft PRs from fresh `origin/main` at `a185332db`:

- Skill-message lane: [PR #2095](https://github.com/zax0rz/darkpawns/pull/2095), worktree `/home/zach/wt/luna-dp1371-skill-messages`.
- Spell-matrix lane: [PR #2096](https://github.com/zax0rz/darkpawns/pull/2096), worktree `/home/zach/wt/luna-dp1371-spell-matrix`.

Neither PR is merged and nothing was deployed. Keep both draft-only. The requested scope is incomplete; Claude owns the eventual batched full census.

## Inventory movement

The user-provided baseline is **494 blocked units out of 1,392**. Eleven real-world potion cells are closed in PR #2096, reducing the combined inventory to **483 / 1,381**. PR #2095 does not close an inventory unit, so the combined total remains 483 / 1,381 after stacking both PRs.

| Inventory row | Before | After both drafts |
|---|---:|---:|
| `fight-skill / lib/misc/messages skill-message corpus` | 51 | 51 |
| `spell-matrix / CAST_POTION quaff vector` | 89 | 78 |
| Total blocked inventory | 494 | 483 |
| Total inventory units | 1,392 | 1,381 |

The spell matrix has 337 reachable gaps remaining, down from 348. In the potion column, the counts are 1 proven / 13 delegated / 89 gap-reachable / 196 C-unreachable before, and 12 / 13 / 78 / 196 after.

The skill-message matrix still has 32 proven records, 50 gap-reachable records, and one DP-1448 gap-blocked record. Its weighted inventory row remains 51 because it includes the 50 reachable records plus that separate blocked record. M@0189 remains gap-reachable: death and hit victim/room cells are proven, hit-attacker is `#`, miss victim/room are proven, miss-attacker remains open, and the God triplet is C-unreachable.

## Evidence and findings

World potion 4399 proves ordered Invigorate, Vitality, and Poison quaff calls at item level 30. Direct oracle transcripts show Poison's save branch at seed 1 and applied affect at seed 8. World potion 13101 proves ordered Sanctuary, Bless, and Armor calls at item level 25, with live score output exposing the resulting affects. The five `luna-dp1371-world-items-seed1/2/3/5/8` targeted censuses pass four named scenarios with zero RNG draw divergence. Additional five-seed `--draw-parity` censuses prove Strength through potion 4826 at item level 30; Cure Critic through 4811; Remove Poison then Cure Light through 4814 (with Poison applied by a registered cast); and Detect Invisibility through potion 8051 at item level 12 by making an invisible PC visible after quaff. Every corrected run is CLEAN with zero draw divergence: Strength `luna-dp1371-strength-potion-seed1-password-fix` and seeds 2/3/5/8; Cure Critic, Cure Light, and Remove Poison `luna-dp1371-potion-heal-remove-seed1/2/3/5/8`; Detect Invisibility `luna-dp1371-potion-detect-invis-seed1-primary-fix` and seeds 2/3/5/8. The initial Strength seed-1 run failed during character creation because passwords exceeded C's 10-character limit; the first Detect Invisibility seed-1 run referenced the primary quaffer as an unconfigured peer. Both setup failures are retained, and corrected runs are clean. The R5c fixture audit force-loads the intended 20%-load potion, 10%-load water container, 25%-load pouch, and 3%-load torch; the seed-1 fill/inv-equip census is clean.

For M@0189, `DoSpellDamage` now preserves C attack IDs through the complete post-position damage path, and level-zero Meteor Swarm no longer receives an invented minimum damage of one. Live hit, death, and miss-victim/room cases pass at seeds 1, 2, 3, 5, and 8 with zero draw divergence (`luna-dp1371-meteor-final-seed1/2/3/5/8`). The connected level-zero PC route through silent room `lua_spell` reaches C's miss-attacker line, but its seed-1 Go comparison is red (`luna-dp1371-meteor-miss-attacker-red-seed1`). That cell remains gap-reachable; no false completion claim is made. The God triplet is unreachable because `spell_meteor_swarm` skips immortal PC victims before `damage()` (`src/spells.c:1120-1121`), while `SkillMessage`'s God branch requires an immortal PC victim (`src/fight.c:1039-1044`).

## Verification

After adding the five new potion proofs, PR #2096 passed each requested gate independently: `make fmt`, `go build ./...`, `go vet ./...`, `go test ./...`, `go test ./pkg/game/...`, `golangci-lint run ./...`, `make deadcode-ratchet`, `make fidelity-depth`, `make fidelity-units`, `make string-census`, `gosec -severity=high -quiet ./...`, and the required manifest proof-token check. Fidelity depth reports 6,313 cases, 6,226 proven/delegated, 9 blocked, 66 excluded, and 12 divergent-approved; the 299-cell spell matrix validates with zero errors. Fidelity units passes all 1,946 claims. String census exits 0 with 98 known Go-only segments and zero unreviewed; its existing warning reports 36 stale baseline entries, and generated reports were left unchanged. PR #2095 had already passed its gates and has no source changes in this follow-up. No final full or combined census was run.

## Resume frontier

Fix and prove the connected miss-attacker Lua dispatch divergence before changing M@0189's record status. Then continue the remaining 50 skill-message records outcome by outcome through registered dispatch or a real combat round, classifying DP-1448-dependent cells as `gap-blocked`. Continue the 337 reachable spell-matrix cells, prioritizing spells on player-usable world items. Keep each proof column to `scenario@seeds` or `Test*` symbols, put census run names in notes, and keep the full census batched for Claude.

Elapsed goal time at handoff: pending final gate run. Draft PR count: 2.
