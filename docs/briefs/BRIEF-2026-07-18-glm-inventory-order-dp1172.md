# BRIEF (glm-5.2) — carried-inventory order reversed vs C (DP-1172)

**Owner:** glm-5.2. **Gate:** Claude adds the oracle `inventory` probe + runs red→green (worker has no `DP_ORACLE_BIN`). **Branch off `main`, one PR.** Bounded, RNG-free, self-checkable from C source.

## ⚠️ Shared-clone sequencing
GLM and Kimi share a working clone. **Run this only AFTER Kimi's `reset_time` port PR has landed/branched** (see [[darkpawns-agent-clone-isolation]]). Different files (this = inventory/handler; Kimi = `weather.go`), but sequence the branches to avoid a dirty shared tree. One branch, one PR.

## The divergence (known, surfaced by DP-1156's inventory probe)
Go lists a character's carried inventory in the **reverse** of C's order. C prepends newly-acquired objects; Go appends. So after picking up A then B then C, C shows `C, B, A` (most-recent first) and Go shows `A, B, C`. This is orthogonal to any rename/RNG — pure list-ordering.

## C truth
- `obj_to_char` (`src/handler.c:~562`) **prepends**: the new object becomes the head of `ch->carrying` (`object->next_content = ch->carrying; ch->carrying = object;`). So `do_inventory` walks head→tail = most-recently-added first.
- Confirm the exact C insertion in `handler.c` and the `do_inventory` walk order (`act.informative.c`), and match Go to it byte-for-byte.

## The work
1. In Go, find the carried-list insertion (`pkg/game/...` — `Inventory.AddItem` / `obj_to_char` analog) and the `do_inventory` display walk (`DoInventory`).
2. Make Go's effective display order match C's **prepend** semantics (most-recently-added first). Prefer fixing at the insertion site (prepend) so *all* consumers match C, unless that risks other ordering-sensitive behavior — if so, document why and fix at the display walk instead. Check whether equipment/other consumers of the carried list depend on the current order before flipping it.
3. Do NOT change object identity, stacking, or counts — only order.

## Acceptance (Claude-gated)
1. Go's carried inventory display order matches C (most-recently-added first). Claude adds an `inventory` probe (get A, get B, get C, `inventory`) after an existing scenario's `remove` step and gates it green.
2. Full committed sweep stays green — especially `equipment-takename`, `inv-equip`, `object-inventory`, `equipment` (verify the flip doesn't redden their existing probes).
3. `go build ./... && go vet ./...` clean.
