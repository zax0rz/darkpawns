# 2026-09-04 — social queue correction

## Finding

The roadmap's Phase 0.2 request to write 15 missing social scenarios is stale
and cannot be executed faithfully as written. The current command corpus has
already exhausted the social queue through `yuball`; the dedicated scenario
files cover `whine`, `whistle`, `wiggle`, `wink`, `worship`, `yawn`, `yodel`,
and `yuball`, while `pray` is covered by its special/fallback scenarios and
`roll`/`snowball` have their own vehicles. The authoritative modeled corpus
remains 4,758 cases: 4,653 proven/delegated, 54 blocked, and 51 excluded.

## C evidence for the three apparent omissions

`hiss`, `kneel`, and `mutter` are present in `lib/misc/socials`, but
`src/interpreter.c` has no command-table entry for any of them. The C oracle
also logs `Unknown social` for each record at boot. They are therefore
verified-excluded from the C command surface under R2/R4/R5e; adding scenarios
or registering them in Go would invent player-facing commands. `roll` is C's
separate `do_roll` command, `snowball` is immortal-gated, and `pray` is
already represented by the existing command/special path proofs.

## Disposition

Phase 0.2 is complete as a stale-doc correction, not as a request to create
unregistered-social scenarios. No social behavior is changed in this slice.
The remaining social loaderization work stays gated on the existing corpus
and its proven coverage, with no blanket exclusion of reachable C socials.
