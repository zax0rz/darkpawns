# Goldfish reviewers, goldfish builders

2026-11-08, Claude, for Zach. A field note relating a published method to PF-114 (amnesiac clones). This is a candidate framing, not a tested claim.

## The source

Google's open-source plugin `google/elephant-goldfish-model` (Apache-2.0, "not an officially supported Google product") turns Dave Rensin's essay "Elephants, Goldfish, and the New Golden Age of Software Engineering" (Medium) into agent skills. Its tagline is "Design with an elephant. Review with a goldfish." It has three roles:
- **Elephant:** an agent that holds the project context and drafts the specification.
- **Goldfish:** reviewer subagents started fresh, with no drafting conversation. Each one reads the draft and the source material it refers to. The plugin stops a review if it can't create such a fresh context.
- **Human:** makes the trade-offs and approves the spec.

The plugin's claim is that a reviewer who doesn't share the author's memory finds the assumptions the author never wrote down.

**The essay.** Dave Rensin (Distinguished Engineer, Google), "Elephants, Goldfish and the New Golden Age of Software Engineering", Medium, 2026-04-27, updated 2026-10-21 to announce the open-source plugin. Zach saved a PDF on 2026-11-08, and Claude read all of it. What it says, in brief:
- **The goldfish role.** The elephant is a long, context-rich session that writes a design document: the problem, the technical plan, the rejected alternatives, and a detailed implementation that lists every file to be created or changed. Goldfish sessions test that *document*:
  - a comprehension test: can a fresh session explain the system from the doc alone?
  - a critic review: Rensin's own estimate is that about 30% of its suggestions are valuable
  - a readiness check
- **Implementation is also handed to a fresh session**, with the finished doc. If a session crashes, a new one takes the doc and carries on.
- **The motive is the one PF-114 documents.** Rensin warns that fast AI coding without a high signal-to-noise ratio ends in unmaintainable slop. He also argues that design docs are far smaller than code, so docs are the cheap way to give an agent context.
- **For existing code,** he builds a README hierarchy by recursive summarization from the leaf directories upward ("peanuts and hay"). A human corrects each leaf, and he estimates the AI's leaf summaries at about 50% wrong.

All numbers in the essay are the author's anecdotes, not measurements.

## The relation to this port

Memorylessness is the same property in both places. Whether it helps depends on the role it fills:

- **On review it helps.** That's EGM's point. Dark Pawns already has the strongest possible goldfish: the C oracle. It has never seen any agent's conversation, and it judges only the bytes. R5h (revert proof) and R5j (a C witness line) make the human and agent reviewers act like goldfish too: they check against `src/`, not against the author's account.
- **On construction it hurts.** This port inverted EGM's arrangement. The builders were goldfish: hundreds of fresh coding-agent sessions over seven months, each starting with no memory of the code its predecessors wrote. Amnesiac clones (PF-114) are the price. A fresh builder can't know a port of `fname()` already exists, so it writes a second one, and the two disagree from birth.

**A correction to this note's first draft.** EGM does use goldfish builders: implementation runs in a fresh session. What protects them is the document, which lists every file to create or change and the reasons. So the candidate claim is narrower: **a goldfish builder is safe only when it is handed an elephant's map of where things already live.** Dark Pawns' early builders (April 2026, when `fname` was first ported) had no such map. The later briefs, which give verified ground truth with file and line citations, are the same remedy, arrived at independently. A project that has to use goldfish builders needs that memory to live in the repository, not in any agent. EGM's README hierarchy is one form of it. Dark Pawns has three partial substitutes:
- the C citations in Go comments
- the depth manifests
- the dead-code ratchet

The proposed C-citation duplicate ratchet would be a fourth (roadmap, 2026-11-08): CI fails when two Go functions cite the same C definition.

**One difference from EGM.** EGM's elephant has to write the design. A port inherits one: the C source *is* the design doc, and the oracle checks against it mechanically. So for a fidelity port, the clone problem is mostly about finding things, not about specifying them: was this C function already ported, and where?

## What would test it

Count amnesiac clones per C function (via citations), and compare how they arise:
- clones created within one long session
- clones created across fresh sessions

If the framing holds, almost all clones arise across sessions.
