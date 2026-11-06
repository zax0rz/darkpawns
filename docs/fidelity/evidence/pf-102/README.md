# PF-102 blind-probe follow-up

Base: `0d68905a9f44600d2bceabac2d85848cb69d4902` (`origin/main`, 2026-11-06).
The 17 sampled output sites are classified in [`verdicts.tsv`](verdicts.tsv); all census directories, including superseded and timed-out attempts, are listed in [`runs.tsv`](runs.tsv).

| Verdict | Count | Probes |
|---|---:|---|
| A — reachable Go literal | 10 | 0, 2, 8, 10, 15, 16, 18, 21, 37, 39 |
| B — selected Go site unreachable | 4 | 3, 4, 23, 38 |
| C — Go-only or non-C bytes | 3 | 11, 19, 27 |
| D — harness cannot drive | 0 | — |

Eight A probes now have a paired targeted proof: mutation red and clean source green. Probe 21's clean run is `CLEAN_AFTER_RECHECK`, which the census policy counts as clean. Probes 2 and 8 reached C with new switched-actor fixtures, but each unmutated run exposed an existing C/Go mismatch; their temporary scenario files were removed, and no game code was changed. Probe 15's original 240-hour attempts timed out, so the fixture was shortened to the first C witness after 16 mud hours and the paired proof was rerun.

## Divergences found

- Probe 2: the room-assigned newbie entrance special runs for the switched actor. C emits `Nah, you're too much of a badass to go in there!`; Go emits `The door seems to be closed.`
- Probe 8: on `steal barrel` while switched into the horse, C emits `Thib the jeweler shouts 'a horse is a bloody thief!'`; Go adds a slap and substitutes the switched player name in the warning.
- Probe 11: Go's selected room broadcast from `AffectUpdate` has no C counterpart; C sends the wear-off message directly to the affected character.
- Probe 19: Go registers an immortal `warn` command with player-facing bytes; C's command table and `src/` have no `warn` command or `do_warn` implementation.
- Probe 27: Go's selected paladin victim failure line omits C's trailing `instead!`.

Probe 23 is dead Go code rather than a production divergence: `CmdSkills` has no production caller or string registration, and C's separate `do_skills` declaration is not in the command table. Probe 4 is one unreachable duplicate `Kick who?` return; its sibling copies are reachable. Probes 3 and 38 have registered/defined special bodies but no C or Go production assignment.

## Decisions for Zach

- File follow-up fidelity issues for the divergences above, especially probes 2 and 8, which became visible only after driving switched-actor paths.
- Decide whether the Go-only immortal `warn` feature should remain; it has no C command counterpart.
- Decide how the dead, unregistered `CmdSkills` table should be handled in future cleanup. It was left untouched here.

No game code, C oracle code, or governing fidelity document was changed.
