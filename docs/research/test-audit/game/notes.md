
## Post-oracle redo, batch 2 (tests 51–100, T389–T438; 2026-07-13..14)

Marks: R 50 / F 0 / C 0 / D 0. Cites grepped this session: tell gates
src/act.comm.c:908-961, "You can't hold that." src/act.item.c:1705, movement
bytes src/act.movement.c:121,238,909, consider ladder
src/act.informative.c:2386-2388, how_good " (not learned)"
src/spec_procs.c:113.

R spot-checks (3 seeded picks): TestNewbieHometownRoom (Kir-Oshi room
swapped — red), TestDoTellEligibilityAndDelivery (not-found byte — red,
after recording that the NoPersonHere constant is not this path's source),
TestTimeWeatherSnapshotTracksCanonicalClock (snapshot loses the weather
half — red).
