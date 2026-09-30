# DP-1371 Phase 3c.1 — RawKill teardown

The M042 strengthening exposed a real production gap: the player RawKill path
left spell/transform affects on and called a corpse helper that returned nil.
This PR wires C's teardown into that path, uses the existing corpse/dust builders,
and adds the retained real-command reproducer. Phase 3c remains paused until
this PR merges. The six Phase 2b outcomes and their original evidence are unchanged.

Base: `4958c781f` (fresh origin/main, #1725 merged).
Production/test commit: `ff3ea8d7c`.

## C step audit (R5c, R5g)

C's authority is `src/fight.c:534-581`, not the old Go helper comments.

| C step | Go before | Go after | Test |
|---|---|---|---|
| NOWHERE guard (:538-539), stop_fighting (:541-542) | Player guard/stop existed; world NPC arm lacked guard | NPC world guard added; stop retained before cleanup | `TestRawKillTeardownOrder`, NPC proof |
| Every affect removed (:544-545; handler.c:428-437) | No RemoveAllAffects callback; NPC arm skipped removal | Live Player.RemoveAffectBySpell, actual legacy master removal, MobInstance.RemoveAffectBySpell; spell flags/modifiers removed without wear-off bytes | `TestRawKillClearsEverySpellAffect`, `TestRawKillNPCUsesFullTeardown` |
| Tattoo off (:547-552) | Skipped | Existing TattooAf(false), tattoo and timer reset for players | `TestRawKillRemovesTattoo` |
| Nightbreed bits/mana (:554-561) | Player werewolf left on; no vampire clamp | Both bits cleared; excess vampire mana clamped after affects/tattoo removal, ordinary mana preserved; NPC equivalents wired | `TestRawKillVampireManaClamp`, NPC proof |
| Unmount (:562-565; utils.c:378-384) | Player's mount name only; NPC arm skipped | Existing riddenMount/clearMountedPair; rider and mount flags cleared | `TestRawKillUnmountAndForget`, NPC proof |
| MOB_MEMORY forget; all hunting stops (:566-572) | Skipped | Existing Forget and ClearHunting, across all world mobs; memory remains gated, hunting does not | `TestRawKillUnmountAndForget`, NPC proof |
| Death cry (:573) | Existing combat.RawKill ordering | Retained before any body or extraction | `TestRawKillDeathCryPrecedesCorpse`, `TestRawKillTeardownOrder` |
| Race-specific dust or corpse (:575-578) | MakeCorpse reached nil stub; MakeDust absent; NPC arm only checked disintegrate | Real shared builders, race dispatch, actual dust prototypes 18/1230 (fight.c:430-480), no invented disintegration line | `TestRawKillCorpseHoldsInventoryEquipmentAndGold`, `TestRawKillRaceDust`, NPC proof |
| extract_char (:580; handler.c:1194-1254) | Player flag path implemented; NPC removed through handleMobDeath | Player deferred extraction retained; NPC world removal retained | `TestCmdSpike_PlayerRawKillContract`, corpse proof, NPC proof; oracle pulse |

All player steps occur in the exact C order; the ordering proof wraps the actual
world callbacks and executes their implementations. The NPC arm is reached from
both World.RawKillCombatant and direct combat.RawKill (the spell entry).
**Explicit gap:** MobInstance has no tattoo field or tattoo-removal counterpart.
No NPC tattoo machinery was invented. Player tattoo removal is fully wired.

## Caller audit

| Caller | C citation | Disposition and boundary proof |
|---|---|---|
| Spike/stake command tail | src/new_cmds.c:1155-1175 | Success acts and PLR nightbreed flag removals precede PK/death counters and TYPE_UNDEFINED RawKill. Retained `TestCmdSpike_PlayerRawKillContract` checks counters, flags, fight stop, affect removal, corpse, queued and actual extraction. Game-level SkillResult proof stays for Phase 3c. |
| Protection from evil/good backfire | src/magic.c:1142-1148,1162-1168 | Existing caller correctly kills the caster, testing the target alignment, with TYPE_BLAST. Exact refusal bytes, blasted corpse, extraction, and absence of die() XP/death-counter penalties are asserted by `TestRawKillProtectionBackfire`. |
| Lua binding | src/scripts.c:1225-1258 | Actual Lua script runs raw_kill(ch,me,TYPE_BLAST), through the typed bridge and world RawKill. `TestRawKillLuaBinding` checks its explicit attack type and that subsequent table write-back cannot restore vampire/mana state after extraction is queued. |

**Lua C limitation (R1a):** scripts.c:1231 declares a three-argument raw_kill and
:1253 passes (vict,killer,type), while fight.c:534 defines a two-argument function.
That C call has an incompatible signature. The Go binding continues forwarding
its explicitly supplied third argument; no pointer-derived attack type is copied.
The existing `lua-bind-misc` oracle uses TYPE_UNDEFINED and cannot certify other
Lua attack types. The TYPE_BLAST binding test certifies the Go API boundary only;
resolving the C binding's undefined behavior needs an owner-approved oracle repair.

## Corpse builder and registered-stub audit

`rg 'MakeCorpse\(|RawKill|return nil'` and the complete callback builder were read.
`act_item_stubs.go` contained one function, MakeCorpse, with an unconditional nil
return. Its sole production caller was cb.MakeCorpse. The file is deleted; the
callback now reaches the existing makeCorpse builder. No second builder was added.

Further corpse details were confirmed while exercising that real builder:

- TYPE_BLAST was absent from attackTypeToCorpseAttack; it now maps to C's
  "A blasted corpse lies here in pieces." (fight.c:296-302).
- C transfers the carrying list intact (fight.c:399-402). The shared builder's
  prepending container API reversed that list; its traversal now preserves order.
- The corpse keyword name keeps ch->player.name's case (fight.c:277-279), so
  `look in corpse` prints `Witness (here):`, rather than `witness (here):`.
- The object movement detach arm returned worn objects to inventory while also
  moving them to a corpse or room. It now detaches completely (handler.c:754-783),
  for both players and mobs. The player corpse proof includes two carried items,
  a worn item, gold, exact pointer order, and a post-extraction check against
  re-scattering an already transferred item. Existing object movement tests pass.
- makeDust previously created synthetic ash and printed an invented line.
  It now loads the same prototypes C loads (fight.c:430-480) and remains silent.
  This shared correction also applies to existing disintegrate callers.

| Callback / stub | Disposition |
|---|---|
| MakeCorpse → unconditional nil helper | Deleted helper; shared real builder and world placement |
| Missing MakeDust / RemoveAllAffects | Installed real implementations |
| Missing tattoo/nightbreed/memory stages | Explicit callbacks added for existing world helpers |
| Unmount → player-name-only mutation | Existing pair cleanup wired, including mount affect |
| GetWeaponInfo's NPC branch always returns zeroes; GetWeaponDescription's NPC branch returns empty | Pre-existing partial callback gaps, left unchanged; RawKillNPC uses concrete world/mob state and does not trust these fallbacks |
| GetSkill, GetDrunk, GetExp, GetKills, GetDeaths, GetPks, GetConstitution, GetGold, GetWimpyLev are player-only lookups with default-zero nonplayer branches | Partial/default query surfaces retained; no claim of broad NPC query fidelity from this PR |
| Other zero/false/empty returns in combat_wire.go | Conditional unknown-character/flag/room guards or species-specific preference/group defaults, with real non-default implementations; no other wholly zero-valued callback body found |

Adjacent World.Instakill and handlePlayerDeath already use the real body builders;
they now inherit the item-order/name/TYPE_BLAST/dust corrections. Their independent
teardown sequencing was not rewritten into a second RawKill implementation here.
No governing documents or manifest statuses, seeds, pins, or expected ledgers changed.

## Failure evidence (R5h)

[Revert outcomes](../evidence/rawkill-3c1-reverts.tsv) lists each clean/broken/restored
command result. Every accepted broken run ends at a test assertion, followed by
restoration and a passing exact-name run. Mutation work is isolated in a detached
worktree. An initial unused-variable mutation caused a build failure; it is
retained separately and is excluded from assertion evidence.

Full terminal events, individual diffs, source snapshots, hashes and replay driver:
`~/Archives/darkpawns/proof-integrity/2026-09-30/dp-1371-3c1/`.
The prior pristine failure remains in sibling directory `dp-1371-3c/`.

## Oracle proof and iteration

`raw-kill-protection-backfire` has a surviving mage observer force a mage victim's
self-protection backfire. Victim alignment/skill and level 31 establish the actual
spell arm; pulse 1 drains the queued extraction; the observer inspects the blasted
corpse and all carried items. Both audiences are compared.

The final targeted run `dp-1371-3c1-backfire-v6` is CLEAN (one PASS).
Its retained C transcript confirms the refusal, death cry, blasted corpse, contents,
and post-pulse removal; a no-diff result alone was not accepted as proof.

Earlier iteration evidence is retained:
- v1 used `alignment` instead of C's exact `align` field and did not backfire.
- v2 reached the backfire but exposed a pre-existing non-mage incantation mismatch:
  C `sfahkaminauab yfaw ozur`, Go `sfainfrauab yfaw ozur`. Not fixed here.
  Diagnostic: `backfire-diagnostic.txt` in the proof archive. The final observer
  is the caster's class, which hears the ordinary spell name in both engines.
- v3 was green but did not cast (the mage's class min-level gate rejected it).
  It is not counted as RawKill evidence; level 31 in v4 fixed the fixture.
- v4 exposed the corpse list order and keyword case defects; both are fixed and
  assertion-reverted above. v5 matched the body; v6 adds the extraction pulse.

## Final gates

Build, vet, go test ./..., clean lint cache and lint run, formatter and diff check
pass. Focused RawKill/command/combat tests also pass under the race detector.
`make fidelity-units`: 1,144/1,144 PASS (925 symbols, nine packages).
`make fidelity-depth`: 5,188 cases; 98.6% actionable completion; unchanged manifests.

Final full census: **NOT_CLEAN**, so the claims census and PR are not started.
Retained run: `/home/zach/Archives/darkpawns/oracle-runs/2026-09-30/dp-1371-3c1-full`
(HEAD `ff3ea8d7c`): 1,038 scenarios, 1,027 passed, five expected, three failed,
two infrastructure rows, one expected unstable; 493.027 seconds.

## Stop findings

The main baseline at `4958c781f` ran the three failed scenarios through
`census.sh`, with a blocking wait. Evidence:
`/home/zach/Archives/darkpawns/oracle-runs/2026-09-30/dp-1371-3c1-main-failure-baseline`.

| Scenario | Main | Candidate | Finding |
|---|---|---|---|
| object-doors | FAIL | FAIL | Fixture uses Orodreth's Cuchi promotion, which the approved DP-1373 security divergence disables (`pkg/game/spec_procs.go:1125`). |
| observation-roomflags | FAIL | FAIL | Same Cuchi promotion dependency. |
| spec-proc-dragon-breath-combat | PASS | FAIL | Correct corpse contents order exposes the separate reversed mob-looter traversal. |

The last finding is a pair of compensating reversals. C transfers the carrying
list intact (`src/fight.c:399-402`), then `attitude_loot` invokes `do_get("all
corpse")` (`src/fight.c:1120`). The get loop visits `cont->contains` forward
(`src/act.item.c:246-253`). Go's shared corpse builder previously reversed the
inventory, and `pkg/game/attitude_loot.go:22` reverses the corpse snapshot again.
The builder fix is necessary for the real RawKill backfire corpse proof, but
exposes the looter defect. C gets backpack, tunic, sword; the candidate gets
sword, tunic, backpack, and its subsequent junk messages change order too.
Diagnostic retained at
`/home/zach/Archives/darkpawns/proof-integrity/2026-09-30/dp-1371-3c1/dragon-diagnostic.txt`.

No looter fix, fixture edits, expected-divergence changes or governing-document
edits have been made. The unsettled infrastructure rows are `goto-private-depth`
and `recall-depth`; they are not the content stop finding. This branch cannot
meet Addendum C's clean final census gate without resolving the looter scope and
the two baseline fixture failures. Phase 3c remains paused.
