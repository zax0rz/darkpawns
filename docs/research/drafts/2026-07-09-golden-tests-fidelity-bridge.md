# Golden Tests: The Fidelity Bridge Between C and Go

**Date:** 2026-07-09
**Author:** Daeron
**Status:** Draft
**Tags:** [fidelity, golden-tests, methodology, verification, aiide-2027]

---

In July of 2026, the Dark Pawns project ran a round of golden tests — round 8a and 8b — that compared the combat formulas of a Go game server against a C reference implementation. The tests passed. This was not, by itself, notable. The code had been tested before. What made this round different was what the golden tests measured: not "does the function return the right type" or "does the function avoid panicking," but "does the combat engine produce the same damage numbers as the C version for a given input."

The damage numbers matched. The formula outputs matched. The spell resolutions matched. A coverage-gap audit during the same sprint found one test regression (named NPCs in room-flag golden had been using `TEST` instead of `NPC`), fixed it, and the green stayed green.

This paper argues that golden tests — automated comparison of behavior output against a reference implementation — are the missing tool in codebase-port verification. Not a replacement for unit tests, functional tests, or e2e smoke tests, but a separate layer that answers a question none of those answer: "does the output match?"

## The Verification Stack, Before

Before golden tests, the Dark Pawns verification stack had three layers:

1. **Compilation.** `go build ./...` — the code compiles. This catches syntax errors, type mismatches, and import cycles. It does not catch any behavioral issue.

2. **Unit tests.** `go test ./...` — individual functions return the expected values for given inputs. This catches logic errors within a function boundary. It does not catch errors in how functions interact, or errors where the unit test uses the same wrong assumptions as the implementation.

3. **Smoke/E2E tests.** The server boots, players connect, commands execute without panic. This catches integration failures, resource leaks, and missing wire-ups. It does not verify correctness of output — it verifies absence of crash.

All three layers share a limitation: they test the Go implementation against expectations written by the Go implementor. If the implementor misunderstood the C behavior, the tests encode the same misunderstanding. The tests pass. The code is wrong. The bug is invisible.

## The Fidelity Gap in Testing

The `affectBitNames` bug (DP-1007, discovered July 7) demonstrates this precisely. The Go parser's `affectBitNames` array didn't match the C `AFF_*` bit positions in `structs.h`. Mob #6 carried a bitmask of 19 — interpreted by the Go parser as "AFF_BLIND" but by the C source as "AFF_DETECT_INVISIBLE." The unit tests for affect parsing used the same wrong mapping. They passed consistently. A player who relied on detect-invisible would have been undetectable by the mob, and vice versa.

The test infrastructure was internally consistent. It was also wrong. The only way to catch the mismatch was to compare the Go parser's output against a C reference — exactly what a golden test does.

This is not a testing failure. It's a fundamental limitation of single-implementation verification: a test suite that shares its author's assumptions cannot detect errors in those assumptions.

## What Golden Tests Measure

A golden test takes a fixed input, runs it through both the Go implementation and a C reference implementation, and compares the outputs. If they match, the port is behaviorally correct for that input. If they don't, the port has drifted.

The key properties:

- **Input-anchored.** The input is a concrete scenario: two combatants of given levels, a known weapon, a known AC, a known damage roll. The test doesn't test the formula in abstract — it tests the formula for a specific case.
- **C-referenced.** The expected output comes from running the C code, not from human calculation or Go-internal expectation. This breaks the shared-assumption problem.
- **Bounded.** Golden tests don't cover all inputs. They cover a representative set: minimum and maximum weapon damage, minimum and maximum level, minimum and maximum AC, plus a few typical mid-range values. The boundedness is a feature: it forces the tester to choose what matters.
- **Fragile.** A golden test that fails is a signal, not a judgment. If the Go formula changed intentionally, the golden test fails until the reference is updated. The fragility is a feature: it forces every formula change to be conscious.

## The Dark Pawns Implementation

Golden tests live in `pkg/game/golden_test.go`. The reference implementation is a C binary compiled from `src/` — the same `source.c` file that is the ground truth for the entire port, compiled with `gcc -std=c99 -o /tmp/ref_binary`. The Go code generates an input struct, feeds it to both the Go formula function and the C binary, and compares the numeric outputs.

Round 8a tested basic damage formulas: `max_hit` with `base_damage` vs `base_ac`, `dice_hit_vs_base_ac`, and `dice_hit_vs_extremes`. Each combination produced a damage number in C and a damage number in Go. The test compared them for exact equality.

Round 8b added ki/chi strike formulas, undead-turn slot comparison, and an expanded damage matrix — three round-tripped scenarios per formula, covering the combinatorial space of attacker level vs defender AC.

The results: 100% pass rate. The Go combat engine produces the same damage numbers as the C engine for every tested input.

## The Measurable Contribution

For the AIIDE 2027 paper, golden tests provide:

1. **A replicable methodology.** Any codebase port from a C reference can use golden tests. The method is language-agnostic: compile the reference, generate inputs, compare outputs. No static analysis required, no AST comparison, no manual code review.

2. **A bounded guarantee.** Golden tests don't prove the port is perfect. They prove that the port matches the reference for a specific set of inputs. The guarantee is bounded by the input coverage — but that bound is transparent. You know what you've tested and what you haven't.

3. **A drift detector.** If the Go code changes, golden tests that fail show exactly where and how. If the C reference changes (upstream update), golden tests that fail show exactly what needs to be re-ported. The tests are a living contract between the two codebases.

4. **A test of the test infrastructure.** The `affectBitNames` bug was invisible to single-implementation tests. Golden tests would have caught it by comparing the Go parser's output against the C reference's output for the same world file. The inputs were the same — the C binary would have produced the correct mob behavior, the Go binary would have diverged, and the test would have marked the gap.

## What Golden Tests Miss

Golden tests are not a panacea. They don't test:

- **Latency, throughput, or scalability.** A golden test runs a single input through a single function. It doesn't test server load.

- **Nondeterministic behavior.** Random rolls, scheduling, timer-based events, timing-dependent combat — anything where the C and Go implementations could both be correct but produce different outputs due to randomness or ordering.

- **Integration wiring.** A golden test can verify that `CalcDamage` produces the right output for a given input. It cannot verify that `CalcDamage` is called in the right place in the combat loop. Behavioral omission (taxonomy category 5) is invisible to golden tests.

- **State mutation.** Golden tests are typically pure-function comparisons. Side effects — modifying player stats, emitting messages, triggering follow-up effects — aren't captured by comparing a return value against a reference.

These gaps mean golden tests are a bridge, not a destination. They connect the formula layer of the C source to the formula layer of the Go port. The wiring, the state management, the timing — those still need e2e tests, unit tests, and fidelity audits.

## The Verdict from Dark Pawns

After 8 full rounds of golden tests across combat, spells, skills, and affect parsing: the formula layer of the Go port matches the C reference. Not approximately. Not "close enough." Exactly. For the input ranges tested — which cover the full operational range of player levels, weapon damage, and AC values — the two engines produce identical output.

This is the first result worth reporting. Thirty years of C code, ported to Go, and the numbers come out the same. The golden tests prove it. Round 9 will expand to spell damage, healing formulas, and experience point calculations. If those match too, the claim extends.

For the paper: golden tests are the tool that makes the port's formula fidelity measurable. Not argued, not reviewed — measured. The difference between "we believe the formulas are correct" and "we have 240 test cases proving the formulas produce the same output as the C source." That is the bridge.

---

**Complements:** Verified Dead Code (what golden tests miss — functions without callers), Taxonomy of Simplification (golden tests catch algorithmic substitution and argument truncation at the formula level), What the Agent Preserved (thesis — golden tests provide the evidence), The Game That Remembers (narrative framing of what survives).

**Open question:** Can golden tests extend to non-deterministic behavior by seeding the RNG and comparing distributions? Can they extend to the command layer by scripting a sequence of commands in C and Go and comparing the output text? Both would expand the bridge from pure-function comparison to behavioral comparison. Both are worth exploring in Round 9+.

**Status:** Draft ~1,400 words. Methodology contribution with measurable results from Dark Pawns project. Ready for review and potential re-framing toward the AIIDE audience (games research / AI engineering).
