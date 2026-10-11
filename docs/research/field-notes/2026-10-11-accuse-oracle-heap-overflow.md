# A blocked accuse case exposed a C social-loader heap overflow

Observed 2026-10-11 by Sol, using the retained differential census and GDB
hardware watchpoints. This is an R1a oracle repair, authorized by Zach.

The original combined run `dp-accuse-four-cases-combined` at
`fcdbb77c813c1bb0e367700c8356b70657553654` was NOT_CLEAN for both full and claims.
Accuse's actor target output was empty; its no-argument output contained
run-varying non-UTF-8 bytes. Go's exact session-dispatch tests remained green.

The loader allocates for registered `do_action` commands, but its existing
unknown-command guard still stores the registered `roll` record. C routes
`roll` to `do_roll`, not `do_action`. The extra stored record overruns the
183-element allocation. A GDB hardware watchpoint on the first accuse message
caught the final `roar` record writing into it: `curr_soc=183`, `list_top=182`.
An isolated data control removed only roll and preserved both accuse strings.
The shared social data was not changed. This diagnoses the write rather than
inferring a cause from corrupted output.

Primary evidence: `~/Archives/darkpawns/oracle-runs/2026-10-11/dp-accuse-diagnosis/`
contains `overflow-watch.log`, `no-roll-control.log`, and `DIAGNOSIS.md`.
Original binary SHA-256:
`49a0799cd7bb107ea846bdd2768a85c100a75803fa2d9d8ded9337aa427bb76b`.

The approved guard consumes unknown or non-`do_action` records into the existing
throwaway slot. It is committed on `dp-oracle-seam` as
`0abd1f13a62e1a1c48834043c175d18afb78fd09` and built with DEV-SETUP's reference
flags. Candidate SHA-256:
`4c421a01f8b5fb30a87d756a7090965f87204837da36ea62bcd8bdbe4d4f4376`.
Both binaries and source revisions are retained in the candidate run evidence.

Go's registered command routes `roll` through `cmdRoll` to `ExecRoll`/`doRoll`;
the social fallback requires a C social entry, and the social-list test excludes
roll. `TestRollRegistrationUsesCEntryGate` and
`TestSocialsRegistrationUsesCEntryGateAndCList` passed. No Go production change.

The candidate full combined census `dp-accuse-candidate-combined` completed at
Go head `fcdbb77c813c1bb0e367700c8356b70657553654`:
`full=CLEAN claims=CLEAN_AFTER_RECHECK`, 3464 pairs. Poofout-depth seed 5 passed
its infrastructure recheck. Comparing all 3464 retained initial C dump files
with the failed baseline found only accuse-depth and accuse-noarg-depth changed
at seeds 1,2,3,5,8. Other retained blocks were byte-identical; this comparison
inherits the harness's existing normalization limits. The shared binary was
promoted 2026-10-11 while holding the census slot lock exclusively. Evidence:
`C-TRANSCRIPT-COMPARISON.json`, `summary.txt`, `go-head.txt`, and `PROMOTION.md`
in that run directory. The Go reference hash is updated to the candidate hash.

Research ledger claim: PF-065.
