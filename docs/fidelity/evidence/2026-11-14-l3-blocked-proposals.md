# L3 blocked-row proposals, 2026-11-14

Companion to the L3 burn-down. These rows need a Zach decision rather than an
agent settlement; each lists what C does, what Go does, and the options. The
rows stay `blocked` until the decision lands.

## entry.identity-name-index-lookup (entry.tsv)

C folds player-table names for lookup and file indexing: `get_player_indir()`
and the binary-search player index treat names case-insensitively and index the
player files by foldered name (`src/db.c:519-523,2340-2353,2651-2678;
src/interpreter.c:1492-1498; src/objsave.c:263-269`). Player identity at every
consumer is *the name*.

Go persists players in SQLite (`pkg/db`) keyed by a stored identity, and the
world registry is keyed by name; the browser entry flow made name lookup a
separate consumer. Per-consumer parity (telnet nanny vs web admin vs oracle
harness) is unproven.

Options:
1. **Witness per consumer** (default): one focused vehicle per entry surface —
   telnet reconnect-by-name, web login, duplicate-name refusal — each pinned
   to C's case-fold and folder behavior.
2. **Ruling that Go's store is the spec for non-telnet surfaces** (R4
   allowance for modern surfaces): keep telnet nanny byte-faithful, declare
   the SQLite keying non-player-facing, close the row as divergent-approved.

## entry.identity-persistent-id-duplicate-sessions (entry.tsv)

C's `perform_dupe_check` resolves duplicate sessions through the persistent
IDNum stored in the player file (`src/db.c:2313-2336; src/interpreter.c:1528-1563`):
a reconnecting player takes over the live body by ID, not by name match.

Go's session manager keys live bodies by session/player object; a duplicate
login is matched through the store's identity column. The takeover semantics
(reconnect to live body, DP-1325) were proven for the telnet flow; the
*persistent-ID* equivalence (rename between sessions, file-level ID) is the
open proof.

Options:
1. Witness: rename a saved player on disk (or via web admin), reconnect both
   sessions, pin C's ID-based takeover bytes.
2. Ruling: declare the stored-ID path a persistence-layer concern invisible at
   the telnet prompt (R4), provided the observable takeover bytes stay
   pinned by the existing DP-1325 rows.

## entry.identity-persistent-id-mail (entry.tsv)

C associates mail with the sender/recipient IDNum (`src/db.c:2313-2336;
src/mail.c:553-585`); mail survives renames because the file is ID-keyed.

Go's mail store keys by the same identity column; rename coverage for mail is
unproven.

Options:
1. Witness: mail a player, rename, mail again, read both — pins C's ID-keyed
   delivery bytes across the rename.
2. Ruling as above if the store's keying is declared non-player-facing.

## entry.identity-persistent-id-house-access (entry.tsv)

C checks house ownership/access by IDNum (`src/house.c:620,734-738`), so a
renamed player keeps house access.

Go's house-control records carry the same identity; rename coverage unproven.

Options: same pair — witness via rename-then-enter, or an R4 ruling.

## entry.identity-password-retry-counter (entry.tsv)

C combines the saved character's bad-password count with a descriptor-local
three-failure threshold (`src/interpreter.c:1880-1894`): failures accumulate
in the char file across sessions and interact with the per-connection count.

Go tracks retries per session/descriptor; whether the cross-session counter
is reproduced (and whether it should be) is the question.

Options:
1. Witness: two sessions, wrong passwords across both, pin C's accumulation
   and lockout bytes.
2. Ruling: keep the session-local counter as a deliberate hardening (the C
   behavior stores a strike count that can lock a player out across
   connections) and mark the row divergent-approved.

## spell.identify.matrix-staff-olc (cast.tsv) — run-varying hours

C's Identify prints `age(victim).hours` (`src/spells.c:594-602`) with a
boot/offset-relative computation; seed-1 C reports `1 hours` where Go reports
`0`, and the value varies run to run for ItemBuilder (retained revert proof
`luna-l3-cast-staff-revert-proof`; classified run EXPECTED_UNSTABLE).

Options:
1. **Normalizer pin**: teach the census normalizer to fold the age-hours
   field (like `<WALL_CLOCK>`) and flip the row oracle-green — needs a
   rulebook-style ruling that hours is a time-derived display, not game state.
2. **Seam fix**: align the port's `age()` with C's `beginning_of_time`
   arithmetic and prove it by unit test across the seam's offset table.
3. Leave blocked as run-varying (status quo; it fails only this one
   EXPECTED_UNSTABLE scenario).

## dispatch.tsv rows (Luna, DP-1521)

`switched.body-combat-lane-wait` and `switched.body-flee-destination-look`
remain blocked on main and belong to Luna's switched-body follow-ups; no
proposal needed from this pass.
