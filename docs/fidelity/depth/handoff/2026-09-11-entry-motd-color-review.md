# Entry MOTD/color proof-boundary handoff — 2026-09-11

## Scope and exact blocker

This handoff covers the `entry.motd-color` manifest/expected-divergence
mismatch on PR #1437 (`glm/depth-shop-live-inventory`). The integrated shop
candidate before this evidence checkpoint was `84f68a94b88903bcbc7a41b988406ba66bf9a942`.
No production behavior, oracle source, normalization, generator, save format,
or shop assertion changed.

The old manifest row was:

```text
nanny entry entry.motd-color D1 transport blocked character-creation-name-retry src/interpreter.c:2130 Creation retry normalized oracle green at seeds 1,2,3,5,8; complete mortal/immortal color-on/off byte matrix remains pending.
```

Because the row was `blocked` with proof `character-creation-name-retry`,
`scripts/gen_expected_divergences.py` associated that scenario with the
expected-divergence baseline. The scenario now reports PASS, so the generated
baseline became stale. Reproduction output is preserved at
`/tmp/dp-review-1437-expected-divergences-initial.log` (SHA-256
`146bebf4beb09e3f8fa4007eaf656257a7111e5af3a926e4a86f278ed6042702`).

## Evidence boundary

`cmd/dp-oracle-diff/scenarios/character-creation-name-retry.txt` has no
`empty-players` fixture. Its C vehicle therefore copies the non-empty oracle
`etc/players` table (6,624 bytes), so `build_player_index()` leaves
`top_of_p_table` nonzero and `init_char()` does not take the first-player God
branch. The actor is a new mortal. The fixed setup is:

```text
aiko, N, Aiko, Y, password, password, Y, M, K, T, K, Y, <ENTER>, 1
```

This exercises the fresh-name path, `CON_NAME_CNFRM` → `CON_GET_NAME` retry,
new-character creation, `CON_COLOR` with ANSI enabled, accepted-stat
initialization/save, mortal MOTD, post-MOTD menu, and first entry. It compares
one normalized `creation` block. The `--show-oracle` output visibly includes
the retry confirmation, MOTD, menu, and entry room. The focused seed-1 run
exited 0; output is preserved at
`/tmp/dp-review-1437-character-creation-name-retry-seed1-show-oracle.log`
(SHA-256
`bacb63eb1ce4cfeb6d9a55c3bef6349eca8e0f8e17d933356c1fd1cde277a209`).

This proves only the normalized mortal/color-on creation cell. It does not
prove raw ANSI bytes because `internal/oraclediff/normalize.go:29-31` strips
ANSI CSI escapes and `cmd/dp-oracle-diff/main.go:476-480` prints the
normalized block. It also does not exercise color-off, a saved immortal, or
the returning-login `imotd` branch.

The relevant C path is `src/interpreter.c:1743-1818` for name entry and
`1848-1853` for the retry, `1978-2008` for color selection,
`2123-2132` for accepted stats, save, mortal `motd`, and transition to
`CON_RMOTD`, and `2173-2217` for menu entry. The returning-login MOTD choice
is `src/interpreter.c:1919-1922`. The corresponding Go path is
`pkg/session/char_creation.go:168-212`, `334-355`, `470-496`, and
`526-584`, with MOTD rendering in `pkg/game/limits_condition.go:318-337` and
returning menu flow in `pkg/session/menu.go:37-48`.

## Manifest reconciliation

The aggregate row is replaced by these five explicit proof scopes in
`docs/fidelity/depth/entry.tsv`:

```text
nanny entry entry.motd-mortal-color-on-normalized D1 transport oracle-green-multiseed character-creation-name-retry@1,2,3,5,8 src/interpreter.c:1848-1853;src/interpreter.c:1978-2008;src/interpreter.c:2123-2132 Normalized-only proof: the non-empty C player table makes the new Aiko a mortal; the Y color choice and N name retry run, and --show-oracle confirms the creation block through MOTD, menu, and first entry. ANSI bytes are stripped, so this does not prove raw color parity or any other matrix cell.
nanny entry entry.motd-mortal-color-on-raw D1 transport blocked pending src/interpreter.c:1978-2008;src/interpreter.c:2123-2132 Raw ANSI/color bytes remain unproven for the mortal color-on cell; the normalized scenario cannot certify them. Smallest next experiment: capture paired raw telnet bytes for the existing creation/retry vehicle at one fixed seed.
nanny entry entry.motd-mortal-color-off-raw D1 transport blocked pending src/interpreter.c:1978-2008;src/interpreter.c:2123-2132 The mortal color-off cell remains unproven. Smallest next experiment: run the same non-empty-player creation/retry vehicle with N at CON_COLOR and compare raw telnet bytes at one fixed seed.
nanny entry entry.motd-immortal-color-on-raw D1 transport blocked pending src/interpreter.c:1919-1922;src/interpreter.c:1978-2008 The immortal color-on cell, including the imotd selection on a returning immortal, remains unproven. Smallest next experiment: use a saved immortal fixture and capture raw bytes after successful password entry with Y preference.
nanny entry entry.motd-immortal-color-off-raw D1 transport blocked pending src/interpreter.c:1919-1922;src/interpreter.c:1978-2008 The immortal color-off cell, including the imotd selection on a returning immortal, remains unproven. Smallest next experiment: use the same saved immortal fixture with color disabled and capture raw bytes after successful password entry.
```

The existing creation scenario remains regression coverage and is now
annotated to the normalized row. No expected-divergence pin is minted: a
passing normalized scenario is evidence, not an observed divergence.

## R5c sibling audit

The same blocked/excluded-row proof-association pattern was audited across
`docs/fidelity/depth/*.tsv`. There are 26 other blocked rows whose proof
resolves to an existing oracle scenario: four `accuse`, two `force`, four
`medit`, four `redit`, four `sedit`, and eight `shoot` rows. The separate
`entry.browser-name-routing` proof points to
`scripts/entry-client.test.mjs`, not an oracle scenario file. Those siblings
remain unchanged; none has the specific stale-baseline condition found here.
This audit is recorded under R5c without expanding the scope into unrelated
entry repairs.

## Focused gate state

`make fidelity-depth` passes with 4,798 cases, 4,679 proven/delegated, 68
blocked, and 51 excluded. `make expected-divergences-check` passes with 26
baseline rows across 10 scenarios and `expected_divergence_pins: OK`.
Regeneration was performed with `make expected-divergences`; after the stale
`entry.motd-color` association was removed, the generated file had no net diff
from the committed baseline. `expected_divergence_pins.tsv` was unchanged.

## Final integrated validation

The evidence checkpoint was `f3017c69f347b3523a6ce95448643a6e31fa9049`.
The following passed on that state: `make check-fmt`, `go build ./...`,
`go vet ./...`, `go test ./pkg/game/...`, `go test ./...`, and
`golangci-lint run ./...`. The required focused gates also passed:
`make fidelity-depth` reported 4,798 cases (4,679 proven/delegated, 68
blocked, 51 excluded), and `make expected-divergences-check` reported 26
baseline rows across 10 scenarios with `expected_divergence_pins: OK`.

The full command was run with `PATH=/usr/local/go/bin:$PATH`,
`DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle`, seed 1, four jobs,
and the Makefile default 240-second scenario timeout. Its preserved output is
`/tmp/dp-review-1437-oracle-regression-final.log` (SHA-256
`1fabb2d3396ace6c718ac1b6f4ed2ab5c889821922c3dcb9dc908a1c2a64f253`). Final
tally:

```text
scenarios=938 passed=928 expected=9 unpinnable=1 stale=0 failed=0 infra=0 timed_out=0
```

The process exited 2 solely because the one unpinnable result was the
previously human-cleared `accuse-noarg-depth` baseline. The nine EXPECTED
results are the existing pinned accuse/force/medit/redit/sedit/shoot ledger
rows. The log contains 13 infrastructure-shaped retries; each recovered to a
named `PASS (... after infra retry)` and none became a final `INFRA` or
timeout. Manual classification of the preserved output found no `INFRA`,
`TIMEOUT`, `STALE`, or `FAIL` result and no other `UNPINNABLE` scenario.

The shop regression cases remain present and PASS in the same census, including
the list pepper, list pep, and list cinnamon cases covered by the shop-live
inventory scenarios. PR #1437 was not merged.

## Remaining debts

- Raw mortal color-on parity is unproven.
- Mortal color-off parity is unproven.
- Saved-immortal `imotd` color-on parity is unproven.
- Saved-immortal `imotd` color-off parity is unproven.

The smallest next experiments are the fixed-seed paired raw-byte captures
specified in the four blocked manifest notes. A normalized transcript must
not be promoted to raw ANSI proof.
