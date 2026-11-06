# Brief: DP-1462, `transfer all` (suggested agent: DeepSeek)

2026-11-06, by Claude for Zach to hand out. This is a small fix, but it touches three C behaviours in one branch.

Read first: `AGENTS.md` and `docs/fidelity/RULEBOOK.md`, from `origin/main`. Work in `~/wt/<agent>-dp-1462`, on branch `<agent>/dp-1462-transfer-all` cut from `origin/main`.

## Ground truth (grepped by Claude at `origin/main`, 2026-11-06)

**C** is `do_trans`, at `src/act.wizard.c:309-360`:
- **The order:**
  - an empty argument gets "Whom do you wish to transfer?";
  - an argument other than `all` (compared with `str_cmp`) takes the single-target branch;
  - otherwise it's the all-branch (`:341`).
- **The all-branch:**
  1. Below `LVL_GRGOD`, it says "I think not." and returns (`:342-345`).
  2. Otherwise it walks `descriptor_list` **in list order**. It skips connected descriptors, missing characters and the caller, and `continue`s on victims at or above the caller's level (`:347-351`).
  3. For each victim, it sends these act lines (`:352-357`):
     - `$n disappears in a mushroom cloud.` (TO_ROOM)
     - `$n arrives from a puff of smoke.` (TO_ROOM)
     - `$n has transferred you!` (TO_VICT)
     - then `look_at_room`.
  4. It moves **no mounts**. Only the single-target branch moves a mount (`:332-336`), and that branch uses `blaze of hellfire`.
  5. Finally it sends `OK` (`:359`).

**Go** is `cmdTransfer`, at `pkg/session/wiz_movement.go:349`. It gets three things wrong:
1. Its all-branch test is `EqualFold(targetName, "all") && level >= LVL_GRGOD`. So below GRGOD, `all` falls into the single-target lookup and prints "No-one by that name here." That's the DP-1462 bug, which `transfer-depth` pins today.
2. The all-branch calls `transferCharacter`, the single-target helper. So it prints `blaze of hellfire` instead of `mushroom cloud`, and it may move mounts. Check that.
3. It walks `s.manager.sessions`, a map, so the victims come out in random order. C uses descriptor order. Find how other Go ports walk descriptors in C order (`grep -rn 'descriptor' pkg/session` for an existing ordered list), and use that. Don't invent a new order.

## Do this

- Make the all-branch match C exactly:
  - the gate and its message;
  - the order;
  - the act lines;
  - no mount moves;
  - `OK`.
- Find the `OK` string's C definition yourself (it's `#define OK` in a header).
- **Prove it:**
  - `transfer-depth` stops matching its pin: it goes STALE, because the divergence is gone.
  - Then move its `wizard.tsv` rows from `blocked` back to `oracle-green-multiseed` on the seeds it claims, take out the `divergence: DP-1462` notes, and run `make expected-divergences`. That removes the entries.
- **Add a scenario** for a GRGOD `transfer all` with two lower-level victims. It needs a C witness line showing `mushroom cloud` (R5j).
- **Revert the fix:** both scenarios must go red (R5h).
- **Census:** targeted, through `scripts/census.sh`: `transfer-depth`, your new scenario, and every other scenario that sends `transfer` (`grep -l 'transfer ' cmd/dp-oracle-diff/scenarios/*`).

## Rules

- **Gates:** run each on its own and check its exit status:
  - `go build ./...`
  - `go vet ./...`
  - `go test ./...`
  - `golangci-lint run ./...`
  - `make fmt`
  - `make deadcode-ratchet`
  - `make fidelity-depth`
  - `make expected-divergences-check`
- **C citations:** cite C from `src/`, grepped in your session (R5g).
- **Repository hygiene:** never switch branches in `/home/zach/darkpawns`, never `git stash`, and never stop processes by name.
- **Governing docs:** don't edit `RULEBOOK.md`, `north-star-1to1.md` or `AGENTS.md`.
- **No outside help:** don't invoke another agent, model or paid service.

Open a PR, ending its description with "Decisions for Zach". Then stop and report: what changed, the census run directories and the revert proof. Claude reviews in Zach's session.
