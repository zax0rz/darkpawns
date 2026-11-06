# Brief: DP-1465, the OLC duplicate-edit refusal (suggested agent: DeepSeek, after DP-1462)

2026-11-06, by Claude for Zach to hand out. This is a small fix across the five telnet editors.

Read first: `AGENTS.md` and `docs/fidelity/RULEBOOK.md`, from `origin/main`. Work in `~/wt/<agent>-dp-1465`, on branch `<agent>/dp-1465-olc-duplicate-refusal` cut from `origin/main`.

## Ground truth (grepped by Claude at `origin/main`, 2026-11-06)

**C** has one refusal for every editor (`src/olc.c:157-158`):
`sprintf(buf, "That %s is currently being edited by %s.\r\n", olc_scmd_info[subcmd].text, GET_NAME(d->character));`
It prints the plain character name and always ends with a period.

**Go** prints `Zeditadmin (telnet; idle 1s)`. Census run `~/Archives/darkpawns/oracle-runs/2026-11-06/dp1456-claude-zedit-pin` (EXPECTED) has it.

**The path:**
- The five telnet editors claim through:
  - `claimObjEdit` (`pkg/session/oedit.go:219`);
  - `claimZoneEdit` (`zedit.go:130`);
  - `claimShopEdit` (`sedit.go:135`);
  - `claimRoomEdit` (`redit.go:127`);
  - `claimMobEdit` (`medit.go:141`).
- Each returns `Manager.olcConflictDescription` (`pkg/session/olc_registry.go:128`). That calls `olc.ClaimConflictDescription` (`pkg/olc/registry.go:257`), which adds the frontend and the idle time.
- `olc_registry.go:130` is the only production caller of `ClaimConflictDescription`.
- **The save-path scans** (`zedit.go:103`, `oedit.go:191`, `redit.go:99`, `medit.go:113`, `sedit.go:112`) use `olcHolder`. That calls `olc.HolderDescription` (`pkg/olc/registry.go:240`), which gives the plain name for a telnet holder and `Name (web)` for a webOLC holder.
- **A second divergence:** `sedit.go:113` has no final period, but C's single format has one.

## Do this

**Decisions, made for you:**
- `olcConflictDescription` returns `olc.HolderDescription(holder)`, the same label the save-path scans use. A telnet holder then shows as the plain name, as in C.
- A webOLC holder keeps the `(web)` suffix. C has no web editor, so there's no C byte to match. List this under Decisions for Zach.
- Add the period at `sedit.go:113`.
- If `ClaimConflictDescription` then has no production caller, delete it and its tests. The dead-code ratchet would flag it otherwise. Check whether anything in webOLC (`pkg/olc`, the admin API) still needs it first. If something does, keep it and say what.

**Prove it:**
- `zedit-duplicate-gate-depth` stops matching its pin: it goes STALE.
- Move its `zedit.tsv` row back from `blocked` at its claimed seeds, and drop the `divergence: DP-1465` note.
- Run `make expected-divergences`.
- **Add one scenario** for a second editor's duplicate refusal (`redit` or `oedit`), built like `zedit-duplicate-gate-depth`: two real descriptors, and the second sends with `send:peer`, not a `[probe:X]` section, which the parser now rejects. It needs a C witness line (R5j).
- A unit test for the `sedit` save-path message.
- Revert the fix, and the scenarios must go red (R5h).
- **Census:** targeted, through `scripts/census.sh`: both duplicate scenarios plus every `*edit*` scenario (`ls cmd/dp-oracle-diff/scenarios | grep edit`).

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
  - `make db-migrate-test`, if you remove an exported symbol.
- **C citations:** cite C from `src/`, grepped in your session (R5g).
- **Repository hygiene:** never switch branches in `/home/zach/darkpawns`, never `git stash`, and never stop processes by name.
- **Governing docs:** don't edit `RULEBOOK.md`, `north-star-1to1.md` or `AGENTS.md`.
- **No outside help:** don't invoke another agent, model or paid service.

Open a PR, ending its description with "Decisions for Zach". Then stop and report: what changed, the census run directories and the revert proof. Claude reviews in Zach's session.
