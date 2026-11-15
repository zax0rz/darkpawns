# DP-1535 combat skills: batch 2

## Scope

This stop-tier batch routes the next ten combat skills for switched mobile
bodies: `backstab`, `bash`, `charge`, `circle`, `cutthroat`, `disarm`,
`disembowel`, `dragon`, `groinrip`, and `headbutt`. The routes use
`CharacterCommandContext`; ordinary player bodies continue through their
existing skill handlers. The mobile body owns skill lookup, room, position,
equipment, combat state, damage, and waits. Output still goes through the
immortal's session descriptor.

`pkg/session/commands.go` registers the actor handler and
`pkg/game/skill_actor_combat.go` ports the command bodies. The shared mobile
damage entry in `pkg/game/spec_fighter.go` preserves C's command damage order:
protection gates, jail redirect, attacker enrollment, charm/switcheroo
redirects, victim enrollment, then damage and skill output
(`src/fight.c:1318-1464`). `mobileSkillHit` follows the mobile `hit()` path in
`src/fight.c:1763-1888`. No file under `src/` or `darkpawns-c-oracle/` changed.

## C path and mobile state

The command lookup and body gates remain the shared dispatcher path in
`src/interpreter.c:910-947`. The witness raises the clone prototype from level
0 to 1 because six commands have a C lookup minimum of 1. Its peaceful flag is
cleared so violent handlers reach their own gates.

All NPCs share the zeroed `dummy_mob` skill record
(`src/db.c:104,1281; src/utils.h:343`). The switched body therefore reads zero
even when the descriptor owner has trained skills. `do_restore` writes skills
through the target mobile's shared record when the target and restorer meet its
level gates (`src/act.wizard.c:1584-1618`). The restored witness sets the
descriptor owner's headbutt skill to zero, restores Satan Klaus, and proves
the switched body then reaches a damaging headbutt path.

Handler-specific branches follow their C sources: backstab's awareness and
fighting checks (`src/act.offensive.c:165-234`); bash's NPC no-bash behavior
(`:419-501`); disembowel and dragon kick (`:234-294,636-692`); headbutt's
low-health, position, and NPC no-bash checks (`src/new_cmds.c:368-470`);
cutthroat's byte probability and combat gate (`:552-665`); charge's mounted
and no-bash probability adjustments (`:880-965`); circle's awareness and
weapon gates (`:2391-2505`); groinrip's shopkeeper, sex, and peaceful gates
(`:2565-2662`); and disarm's current-target, weapon, and combat checks
(`src/new_cmds2.c:191-258`).

## C-first witnesses

- `switched-body-combat-skills-offensive-depth` places the clone's dagger and
  trainee in room 8162, with the descriptor owner in 8105. The owner knows all
  ten skills; the switched clone reads the shared zero skill record. Every
  probe reaches its C handler and matches that handler's mobile-skill
  refusal. The circle probe also passes its target, weapon, mount, and
  awareness checks before the skill check.
- `switched-body-combat-skills-offensive-restored-depth` places Satan Klaus
  and the trainee in room 8105 while the descriptor owner remains in 8162.
  Restore gives the shared mobile headbutt skill its C value; the switched
  body damages its own trainee with matching output and draws.

Both witnesses have zero hollow steps. The final candidate census rows for
these scenarios are PASS; the retained revert fails both. On revert, all ten
zero-skill commands fall through to the old Player-bound path and emit a
different response. The restored headbutt instead reads the descriptor owner's
peaceful room and refuses before C's damage draw.

## Census and retained revert

Candidate snapshot `ade8d36cd` (code commit `4181b8534`) is based on current
`origin/main` `2b0dba6938cbc4a5069f728facfabf62a75a3903`. Run
`/home/zach/Archives/darkpawns/oracle-runs/2026-11-15/luna-dp1535-offensive-rebased`
is CLEAN: 150/150 passed, with zero failed, infra, timeout, stale, unpinnable,
or unstable rows. `--draw-parity` reports zero divergent scenarios. Its two
hollow steps are the known `hit Nobody` probes in
`switched-body-combat-lane-wait-depth` and
`switched-body-combat-lane-wait-body-context-depth`; neither new witness is
hollow.

Retained revert branch
`/home/zach/wt/luna-dp1535-batch2-revert-v2` is at `baac2e43e`, rebased onto
the same `origin/main`. Its run
`/home/zach/Archives/darkpawns/oracle-runs/2026-11-15/luna-dp1535-offensive-revert-rebased`
is NOT_CLEAN: 148/150 pass. The only failures are
`switched-body-combat-skills-offensive-depth` and
`switched-body-combat-skills-offensive-restored-depth`; all other scenarios
pass. The census reports zero run-level draw-divergent scenarios; attempt
logs retain the expected command-level revert differences.

The targeted set contains every scenario that types one of the ten commands
plus every scenario that uses `switch` (150 total):

```text
attitude-loot-switched-output-depth, backstab-aware-trio, backstab-depth,
backstab-skill-message-death-depth, backstab-skill-message-god-depth,
backstab-skill-message-hit-depth, backstab-skill-message-miss-depth,
bash-depth, bash-failure-depth, bash-peaceful-depth, bash-refusals-depth,
bash-skill-message-death-depth, bash-skill-message-god-depth,
bash-skill-message-hit-depth, bash-skill-message-miss-depth,
bite-groinrip-peaceful, charge-depth, charge-failure-depth,
charge-skill-message-death-depth, charge-skill-message-hit-depth,
charge-skill-message-miss-depth, circle-aware-depth, circle-behead,
circle-depth, circle-gates-depth, circle-skill-message-death-depth,
circle-skill-message-god-depth, circle-skill-message-hit-depth,
circle-skill-message-miss-depth, combat-backstab-opener,
combat-bash-opener, combat-headbutt-opener, combat-skill-noskill,
combat-skill-noskill2, combat-transform-reject, combat-unlearned-bash-opener,
cutthroat-depth, cutthroat-failure, cutthroat-gates-depth,
cutthroat-low-level, cutthroat-mounted, cutthroat-peaceful,
cutthroat-peaceful-depth, cutthroat-skill-message-death-depth,
cutthroat-skill-message-god-depth, cutthroat-skill-message-hit-depth,
cutthroat-skill-message-miss-depth, disarm-depth, disarm-failure-depth,
disarm-gates-depth, disarm-no-weapon-depth, disembowel-cutthroat,
disembowel-depth, disembowel-refusals-depth,
disembowel-skill-message-awake-depth, disembowel-skill-message-death-depth,
disembowel-skill-message-god-depth, disembowel-skill-message-hit-depth,
dragon-depth, dragon-god-depth, dragon-kill-depth, dragon-success-depth,
dream-switched-sleeping-depth, fight-attitude-loot-switched-depth,
fighter-gates-depth, flee-get-on-feet-switched-turn,
follow-switched-charm-master-depth, god-batch2, god-skill-keys,
groinrip-depth, groinrip-fighting-fallback-depth, groinrip-immortal-depth,
groinrip-mounted-depth, groinrip-no-skill-depth, groinrip-nonmale-depth,
groinrip-outcome-depth, groinrip-shopkeeper-depth,
groinrip-skill-message-death-depth, groinrip-skill-message-hit-depth,
groinrip-skill-message-miss-depth, groinrip-sleeping-depth,
groinrip-sleeping-pk-depth, headbutt-depth, headbutt-immortal-depth,
headbutt-lowhp-depth, headbutt-mounted-depth, headbutt-noskill-depth,
headbutt-peaceful-depth, headbutt-self-depth,
headbutt-skill-message-death-depth, headbutt-skill-message-depth,
infobar-pulse-empty-switched-depth, l1-wearoff-divineint-switched-depth,
levels-npc-depth, mob-wearoff-switched-depth, monk-refusals-depth,
mudlog-help-body-depth, newbie-zone-entrance-switched-depth,
practice-mapped-ninja, real-level-pc-switch, shoot-switched-mob-depth,
shopkeeper-switched-steal-depth, skill-corpse-attack-type-depth,
skill-pain-scream-draw-depth, skill-refusals-gates-depth,
spec-proc-identifier-give-switched, spec-proc-rescuer-7909,
spell-mob-wait-script-depth, switch-mob-dispatch-gates-depth,
switch-mob-observation-depth, switched-actor-depth,
switched-body-combat-lane-wait-body-context-depth,
switched-body-combat-lane-wait-depth,
switched-body-combat-skills-offensive-depth,
switched-body-combat-skills-offensive-restored-depth,
switched-body-combat-skills-utility-depth,
switched-body-combat-skills-utility-restored-depth,
switched-body-command-matrix-depth, switched-body-consumables-depth,
switched-body-door-depth, switched-body-flee-destination-look-depth,
switched-body-get-drop-depth, switched-body-infobar-depth,
switched-body-kuji-kiri-depth,
switched-body-kuji-kiri-ninja-class-depth,
switched-body-kuji-kiri-restored-skill-depth,
switched-body-movement-position-depth, switched-body-object-handling-depth,
switched-body-peek-thief-class-depth, switched-body-reply-depth,
switched-body-reply-newest-mob-depth, switched-body-toggle-depth,
switched-body-wizutil-depth, switched-bridges-depth,
switched-communication-depth, switched-ctell-lowlevel-depth,
switched-immortal-depth, teleport-npc-refusal-switched-depth,
teleport-npc-switched-output-depth, teleport-switched-original-body-depth,
tell-switched-body-depth, violence-npc-switched-output-depth,
wait-queue-drain-depth, wait-queue-multi-drain-depth, who-out-of-body-depth,
wizard-residual-depth, wizard-switch-already-player, wizard-switch-depth,
wizard-switch-player, wizard-switch-player-level
```

## Verification

The implementation passed on the rebased snapshot:

- `make fmt`
- `go build ./...`
- `go vet ./...`
- `GOEXPERIMENT=nojsonv2 go test -p 4 ./...`
- `golangci-lint run --concurrency 4 ./...` (0 issues)
- `make deadcode-ratchet` (52 baselined, 0 new)
- `make db-migrate-test`
- `GOEXPERIMENT=nojsonv2 make fidelity-depth` (7,558 proven/delegated; 1
  pre-existing blocked case)

No rulebook amendment is proposed. This batch does not complete the combat
skill class; the remaining combat skills come next, followed by the remaining
C-reachable actor waits and `retreat-closed-door`.

## Decisions for Zach

- Review this stop-tier batch and its C-first, draw-parity, and retained-revert
  proof. Zach merges; this branch is not merged here.
- Confirm the shared `dummy_mob` skill state, mobile command damage path, and
  descriptor-side output handling before the next batch.
- The next work is the remaining combat-skill rows, then the remaining
  C-reachable actor waits, then `retreat-closed-door`.
