# MOTD/color evidence boundary — 2026-09-11

Scope: resolve the `entry.motd-color` expected-divergence association on the
integrated PR #1437 candidate. No production behavior, normalizer, generator,
oracle source, or world data changed.

## Reproduced gate failure

Command:

```text
PATH=/usr/local/go/bin:$PATH DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle make expected-divergences-check
```

The command exited 2. The generator reported 27 rows across 11 scenarios and
added exactly this stale row to the generated file:

```text
character-creation-name-retry	entry.tsv	entry.motd-color	blocked
```

The complete output is preserved at
`/tmp/dp-review-1437-expected-divergences-initial.log` with SHA-256
`146bebf4beb09e3f8fa4007eaf656257a7111e5af3a926e4a86f278ed6042702`.

## What the scenario proves

`character-creation-name-retry.txt` has no `empty-players` fixture, so the C
vehicle copies the existing `etc/players` table. The copied table is non-empty
(6,624 bytes); `build_player_index()` therefore leaves `top_of_p_table`
nonzero, and `init_char()` does not take its first-player God branch. The actor
is a new mortal. The setup sends:

```text
aiko, N, Aiko, Y, password, password, Y, M, K, T, K, Y, <ENTER>, 1
```

Thus it exercises C `CON_NAME_CNFRM` → `CON_GET_NAME` retry, new-character
creation, `CON_COLOR` with ANSI enabled (both color bits), accepted-stat
initialization/save, the mortal MOTD, the post-MOTD menu, and first entry. The
C `--show-oracle` block visibly contains the retry confirmation, MOTD, menu,
and entry room. The run exited 0 with no normalized divergence; its complete
output is preserved at
`/tmp/dp-review-1437-character-creation-name-retry-seed1-show-oracle.log`
with SHA-256
`bacb63eb1ce4cfeb6d9a55c3bef6349eca8e0f8e17d933356c1fd1cde277a209`.

The vehicle does not exercise a saved immortal login, the C `imotd` selection,
or color-off. More importantly, `internal/oraclediff/normalize.go:29-31`
strips ANSI CSI escapes before comparison and `cmd/dp-oracle-diff/main.go:476-480`
prints that normalized block. Therefore its green result proves normalized
player-facing text for one mortal/color-on creation cell, not raw ANSI parity.

## Reconciliation

The old aggregate blocked row was split into one proven normalized cell and four
explicit blocked raw-byte cells in `docs/fidelity/depth/entry.tsv`:

| row | disposition |
|---|---|
| `entry.motd-mortal-color-on-normalized` | `oracle-green-multiseed`, scenario-backed normalized proof only |
| `entry.motd-mortal-color-on-raw` | blocked; raw color proof pending |
| `entry.motd-mortal-color-off-raw` | blocked; color-off proof pending |
| `entry.motd-immortal-color-on-raw` | blocked; saved-immortal/imotd raw proof pending |
| `entry.motd-immortal-color-off-raw` | blocked; saved-immortal/imotd color-off raw proof pending |

The existing scenario remains annotated against the normalized cell and remains
regression coverage for the retry/MOTD/menu path. No expected-divergence pin was
minted: the scenario passes, and the raw gaps are not observed divergences.
