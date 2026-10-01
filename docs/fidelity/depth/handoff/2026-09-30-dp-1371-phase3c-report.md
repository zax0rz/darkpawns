# DP-1371 Phase 3c — strengthen six weak mutation proofs

Fresh base: `34c3d437e` (origin/main, #1727 merged). Tests and proof metadata only; no production, scenario, expected-divergence or governing-document changes. Phase 2b’s original 100-test results and patches remain unchanged.

## Before and after (R5h)

“Before” is the retained Phase 2b classification, not a reclassification of the historical experiment. “After” re-applies each exact stored patch from `scripts/fidelity_mutations/evidence/mutations.json` on this main base with the strengthened proofs. M042 intentionally switches from the SkillResult-only proof to the real command proof.

| ID | Before | After | Clean → broken → restored |
|---|---|---|---|
| M010 | survived | killed (assertion) | 0 → 1 → 0 |
| M021 | survived | killed (assertion) | 0 → 1 → 0 |
| M025 | survived | killed (assertion) | 0 → 1 → 0 |
| M035 | survived | killed (assertion) | 0 → 1 → 0 |
| M042 | survived | killed (assertion) | 0 → 1 → 0 |
| M100 | timeout | killed (assertion) | 0 → 1 → 0 |

## Proof changes (R3/R5c/R5g)

- **M010:** Both Kick (134) miss variants are pinned byte for byte through wireKickMessages → InitFightMessages, including C table order and rendered victim pronouns. The mutation emits Backstab bytes and fails that equality. Authority: src/fight.c:1023-1092 and lib/misc/messages:252-285. The callback captures message payloads before the transport adds its line ending.
- **M021:** A failed roll consumes only number(0,3). The success seed (50) reaches a real frost CallMagic → MagAreas → MagDamage → skill_message(204,0), captures its caller/victim/type/damage boundary, enrolls the previously unfighting victim and preserves HP. The shared magic_user tail chooses neutral alignment roll 12, avoiding a second spell. The next draw is 792 after breath roll, SAVING_SPELL, message selector, victim selector and spell selector. Authority: src/spec_procs.c:937-963,418-452; magic.c:823-827,1611. The impossible-roll mutation produces no breath and no enrollment.
- **M025:** Non-pray look has both immortality and an empty argument with a live item_for_Serapis altar. Both must return FALSE with no output or level/XP/flags/gold/inventory/room-item changes. The disabled gate emits refusal or generates a reward and charges gold. The approved DP-1373 immortality refusal remains unchanged. Authority: src/spec_procs.c:2077,2081-2152.
- **M035:** Seed 1 produces number(1,101)=44 then dice(1,8)=8; exact mounted damage is 50+2*8=66. The 49 mutation yields 65. Authority: src/new_cmds.c:917-925,939-952.
- **M042:** The player-state manifest now references TestCmdSpike_PlayerRawKillContract (merged in #1727), which executes the real command and world callbacks: PLR flags, PK/death counters, fighting stop, AFF removal, corpse, queued extraction, extraction pass and absence from the world. The original game-level test also explicitly requires RawKill=true. C raw_kill does not set negative HP/POS_DEAD; no invented assertion is added. Authority: src/new_cmds.c:1155-1175; fight.c:534-581; handler.c:1194-1254.
- **M100:** Read the next complete prompt, terminating for either Password or name confirmation, with a 512-byte bound and a five-second I/O guard. Assert exact Password: bytes before sending the password. The case-sensitive lookup mutation immediately emits the new-character confirmation; the failure is a prompt equality assertion, not the guard deadline. Real TCP and real SQLite remain in use. Authority: src/interpreter.c:1743,1860.

## Retained evidence

[Six outcome rows](../../../../scripts/fidelity_mutations/evidence/phase3c-results.tsv) include prior and current symbols, mutation SHA-256, assertion text and exits. The original Phase 2b evidence is not overwritten.

Full events, exact mutation patches, source hashes and replay driver:
`/home/zach/Archives/darkpawns/proof-integrity/2026-09-30/dp-1371-3c/`.
Each clean/restored run has a real named test PASS event. Each broken run has a named FAIL event and assertion output; none timed out, panicked or failed to build. Mutations ran in `/home/zach/dp-integrity-3c-mutations`, and production files were restored after each run. Elapsed values in the TSV include compilation for all three runs.

## Verification

`go build ./...`, `go vet ./...`, `go test ./...` and `golangci-lint run ./...` each exit 0; lint reports zero issues. `make fmt` and `git diff --check` are clean. `make fidelity-units`: 1,144 PASS, 925 symbols across nine packages (evidence `~/Archives/darkpawns/oracle-runs/2026-09-30/dp-1371-units-201050/`). `make fidelity-depth`: 5,188 cases, 5,063 proven/delegated, 72 blocked, 53 excluded; unchanged 98.6%. Focused game/command/telnet proofs also pass under `go test -race`. No census is required by ADDENDUM B because the branch changes tests and proof metadata only.
