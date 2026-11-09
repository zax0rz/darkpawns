# Live divergence retained during DP-1511 witness collection

No game code changed; no pin added. Claude files the item after review.

## Recipient weight refusal through live give

- Run: `/home/zach/Archives/darkpawns/oracle-runs/2026-11-09/sol-dp1511-general-witness`
- Snapshot: `db9370cd3` plus frozen dirty changes in that run's `source/`.
- Diagnostic source retained beside this file as `general-witness-scenario.txt`.
- First divergent block: `give sword Gatepeer [actor]`, following `load obj 5805`
  and `set Gatepeer str 3` in room 8162 with the two connected players.
- C: `He can't carry that much weight.`
- Go: `He can't carry any more.`
- First capture: `initial-logs/attempt.PoFicX/warmup-refusal-general-inventory.attempt1.log:106-110`.
- C path: do_give → perform_give, `src/act.item.c:767-809,692-695` (read this session).
- `give.weight` demoted to blocked: its previous unit-green helper boundary
  did not prove this live command (R5f). The helper test was not edited.

This is a newly created diagnostic, not a failure of a pre-existing warmup.
The diagnostic has no new green manifest claim and is retained outside the
shipping scenario corpus; its C lines remain valid matcher witnesses.

## Fill-word-only door argument; switched charmed NPC follow path

- Run: `/home/zach/Archives/darkpawns/oracle-runs/2026-11-09/sol-dp1511-charm-witness`.
- Diagnostic retained as `charm-witness-scenario.txt` outside the shipped corpus.
- First divergent block: `open the [actor]`: C `What is it you want to open?`,
  Go `Open what?`. `close the` analogously differs. C `do_gen_door` →
  `find_door` reaches `src/act.movement.c:403-410,598-613`.
- Added blocked `door.fillword-only-argument`; bare open/close proof remains
  valid for its previously tested input.
- A subsequent switched-horse follow probe shows C enforcing the charm/master
  guard (`But you only feel like following Charmhelper!`) while Go stops
  following the helper. C site `src/act.movement.c:904-911`. Added blocked
  `follow.switched-npc-charm-master-guard` (R2e/R5f).
- Earlier `follow Charmhelper` audience output also substitutes the player's
  name in Go where C names the horse. Subsequent `north`, `ride 2.horse` and
  `ride horse` differences follow the divergent follow/location state; retain
  them as observations rather than asserting separate root causes.
- Evidence: `initial-logs/*/warmup-refusal-charm-inventory.attempt1.log`, first
  door difference lines67-81, switched-follow/movement differences127-192.
- The diagnostic's `set Charmhelper level 40` is itself refused identically:
  C `do_set` refuses values >= caller level (`src/act.wizard.c:2897-2902`).
  This does not establish state; none of the guard witnesses above depend on
  the helper being level40 (the actor subsequently explicitly sets it to1).

## Further live differences

All run names below are under `/home/zach/Archives/darkpawns/oracle-runs/2026-11-09/`.
The complete first-attempt log and both normalized engine transcripts are retained
in each run; `divergent-blocks.tsv` indexes every differing block, including
consequences after the first difference. No diagnostic is shipped as green.

| Run | First divergent block | C → Go | Claim disposition |
|---|---|---|---|
| `sol-dp1511-item-witness-r2` | wield axe while holding scroll | `It is too heavy for you to use.` → `Both hands must be free to wield that.` | blocked `wield.weight-before-two-hand-conflict`; `src/act.item.c:1492-1502` orders weight first |
| `sol-dp1511-item-witness-r3` | give scroll Gateitems | `Gateitems seems to have his hands full.` → transfer succeeds | `give.hands-full` demoted from unit-green; `src/act.item.c:687-689`; later inventory/drop differences are downstream |
| `sol-dp1511-clan-transform-witness` (also later clan runs) | clan demote victim audience | `You've demoted within your clan!` → `You've been demoted within your clan!` | `clan.demote-output-gates` demoted; `src/clan.c:451-488` |
| `sol-dp1511-write-witness` | write Active author | `You have no active.` → `You don't have 'author'.` | blocked `write.two-argument-missing-paper`; this first attempt lacked a board, so it proves only generic write; `src/act.comm.c:1048-1056` |
| `sol-dp1511-write-witness-r2` | write missing | `There is no missing in your inventory.` → `Write?  With what?  ON what?  What are you trying to do?!?` | blocked `write.single-argument-missing-object`; `src/act.comm.c:1058-1062` |
| `sol-dp1511-mounted-charm-witness-r3` (also prior mounted runs) | helper teleport Ridergod 8161, primary audience while switched into horse | original-body departure only → additionally recipient notification and room look | blocked `teleport.switched-original-body-output`; `src/act.wizard.c:383-388`, `src/act.informative.c:1103-1104`; subsequent movement/snoop blocks are observations, not independent root-cause claims |
| `sol-dp1511-clan-load-boardfull-witness` | alias `post` at capacity | `The board is full.` plus later `Huh?!?` inputs → editor `]` | `boards.full-board-refusal` demoted from unit-green; root cause within alias/editor sequence unisolated; `src/boards.c:221-226` |

The later invisible-pen diagnostics retain the generic write differences too.
Their first failed attempts did not establish invisibility while held; only
`sol-dp1511-write-held-invisible-witness` does. Similarly, the first no-moon
attempt used the wrong time; `sol-dp1511-no-moon-witness-r2` supplies the actual
C moon refusal. The initial PC charm attempt used a C-invalid peer name
(`Charmer`, INFRA); the next was peaceful-room blocked. Only the final outdoor,
non-peaceful `sol-dp1511-mounted-charm-witness-r3` establishes the PC charm guards.
These retries retain their first attempts rather than replacing them.

## Diagnostic artifacts and out-of-range input

- `sol-dp1511-item-witness` first differed at `load obj 16200`: the diagnostic
  added object index files 58 and 162, but `applyObjIndexFixtures` inserts both
  at the first file's position. C's binary-search index consequently misses
  16200 while Go's map finds it. This is a harness fixture-ordering defect,
  not proof of a game divergence. Later item diagnostics use stock indexed
  objects. The same insertion design exists in `applyWldIndexFixtures`.
  R5c audit found five existing multi-object-file scenarios:
  `spellmatrix-world-scroll-enchant-armor-depth`,
  `spellmatrix-world-potion-multislot-gap-depth`,
  `spellmatrix-world-potion-native-depth`,
  `spellmatrix-world-scroll-enchant-weapon-depth`, and
  `spellmatrix-world-potion-movement-depth`. Both existing world-index users
  add just one file. This follow-up needs its own repair and evidence.
- `sol-dp1511-shop-witness` differs only at `buy 2147483648 gold`: C's out-of-range
  `atoi` result reaches the negative-count message; Go reaches the carry cap.
  This retained line is a real C matcher witness, but an out-of-range conversion
  is not asserted as a defined gameplay parity defect (R1a).

No game implementation, expected-divergence pin, or governing rule changed.
