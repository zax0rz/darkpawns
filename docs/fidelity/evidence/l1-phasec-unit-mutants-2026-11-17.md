# Phase C L1 long-shot unit mutants (2026-11-17)

Queue orders 339 and 343 are selected by the rare hunt draw cases in C
`src/graph.c:374-377` and `src/graph.c:390-392`. The call-boundary test forces
the selector draws and asserts the exact C command argument because a normal
seeded oracle transcript is not a reliable witness for these rolls.

The literal mutant checks used:

- Case 6: change `rabbit` to `hare` in `pkg/game/graph.go`. The
  `TestHuntVictim_CommandArgumentsMatchC/case_6_gossip` subtest failed and
  reported the changed argument against C's expected `Prey flees like a
  rabbit...`.
- Case 10: change `little` to `small` in `pkg/game/graph.go`. The
  `TestHuntVictim_CommandArgumentsMatchC/case_10_gossip` subtest failed and
  reported the changed argument against C's expected `Where is that little
  wimp, Prey?`.

Both production literals were restored. The clean `GOEXPERIMENT=nojsonv2 go
test -p 4 ./...` run passed after restoration. The two queue rows retain their
`unit-green` status only under the long-shot exception, with the C call sites
above and this revert proof (R1/R3/R5g/R5h).
