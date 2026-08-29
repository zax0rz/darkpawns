# 2026-08-29 — `pet_shops` depth slice

## Frontier and queue

- Session boundary returned to `main`; the required pull could not contact GitHub because DNS could not resolve `github.com`, but local `main` was already at `origin/main`.
- The pre-slice `make fidelity-depth` frontier on `main` was 904 total, 882 proven/delegated, 6 blocked, and 16 excluded (99.3% actionable).
- The next source-order special-procedure item after Dracula was `pet_shops`; it is now proven locally and must not be repicked as an implementation slice. Its publication remains pending on the retained local branch because the external Git transport was unavailable.
- Next source-order item after this slice: `enter_circle`.

## C path and reachable cases

- `src/interpreter.c:1407-1415` calls the registered room special before ordinary command dispatch; `src/spec_assign.c:618` assigns room 21235 to `pet_shops`.
- `src/spec_procs.c:1844-1907` computes storage room `ch->in_room + 1`, lists `world[pet_room].people`, and handles `buy` in this order: missing pet, follower cap, insufficient gold, `read_mobile`, charm/name/description state, follower relation, room act, and success text. Any command other than `list` or `buy` returns FALSE.
- `src/handler.c:865-882` supplies the C `get_char_room` keyword-abbreviation and ordinal semantics. `src/comm.c:2392-2555` supplies the follower and purchase audience substitutions.

## Proof and implementation

- Added `spec-proc-pet-shops-branches`, `spec-proc-pet-shops-funds`, `spec-proc-pet-shops-unnamed`, and the focused base vehicle. The branch vehicle covers list order, bare/unknown buy, named success, follower-cap precedence, and non-buy/list fallthrough; the other vehicles isolate affordability and unnamed success.
- Clean-main RED first showed both `list` and `buy` falling through to Go's ordinary-shop rejection. Subsequent C-confirmed fixes cover the room-special nil-`me` contract, exact direct-message framing, reverse storage-list order, silent `read_mobile`-style spawn, C keyword matching, follower count/cap, `AFF_CHARM`, follower and room audiences, and per-instance lowercase custom name/description state without mutating the shared prototype.
- Added `TestSpecPetShops_EntryGates`, `TestSpecPetShops_ListOrder`, `TestSpecPetShops_BuyGates`, and `TestSpecPetShops_PurchasedNameState`; all pass.
- Oracle differential results: all four vehicles green at seed 1; the branch vehicle also green at seeds 2, 3, 5, and 8, with `--show-oracle` confirming the intended room-special blocks.
- Added nine manifest rows to `docs/fidelity/depth/spec-procs.tsv`; the branch frontier is 913 total, 891 proven/delegated, 6 blocked, and 16 excluded.

## Gates and publication

- On `glm/spec-pet-shops`, commit `67c50b2ff` (`fix: align pet shop special with C (R1/R2/R5)`) passed `make fidelity-depth`, `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run ./...` (with `/usr/local/go/bin` on PATH), and repository-wide `gofumpt -l .`.
- The branch push was attempted twice and failed before PR creation: `ssh: Could not resolve hostname github.com`. There is no PR or CI state to call green, and no merge was performed. Retain the local branch and commit; resume publication before terminal accounting.
- The unrelated pre-existing untracked file `docs/briefs/BRIEF-2026-08-28-economy-specproc-cluster.md` remains untouched.

Rules applied: R1 (player-facing bytes), R2 (room-special command surface and return), R3 (seeded vehicle checks), R4 (no invented failure output), and R5 (actual C dispatch/call path and shared-class boundaries).
