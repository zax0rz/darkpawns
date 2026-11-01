# C coding guide, v2

Authority: STUDY.md at 56571bbc606c43a3ca9c23d817cabfd3607272a8.

- **A check** is any step whose result the project used as evidence: a scenario verdict, a test exit code, a CI job, or a manifest or proof status.
- **Group C** applies when that step failed, never ran, or produced known-bad output, and it was recorded as a non-failure (pass, expected, INFRA, flake, blocked) without its cause being resolved.
- **Not C:** the step ran properly but watched too little (A), or ran against a wrong reference or erased bytes (B).
- **Setup is part of the check.** A scenario whose required setup was refused, yet still recorded PASS, is C.
- **A false green withdrawn before anyone relied on it is not C.** The process caught it; code it as a detection, in the project's favour.
- **"Relied on" means the claim reached main, a release, a report to Zach, or a decision** (Zach, 2026-11-01). A claim credited and withdrawn on an unmerged branch was not relied on, even if an agent's working handoff counted it.

Prefix-proof is A: the longer tests existed and ran; substring proof-label validation missed an inexact name. Save-room-false-UB is C: the never-run proof was recorded excluded on main from PR #1637 merge 11:23:22Z until PR #1641 merge 14:53:34Z on Sept 25. Teleport remains a detection: the false claim was withdrawn while its PR was open. Existing setup and reliance rulings remain unchanged.
