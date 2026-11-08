# Goldfish reviewers, goldfish builders

2026-11-08, Claude, for Zach. A field note relating a published method to PF-114 (amnesiac clones). This is a candidate framing, not a tested claim.

## The source

Google's open-source plugin `google/elephant-goldfish-model` (Apache-2.0, "not an officially supported Google product") turns Dave Rensin's essay "Elephants, Goldfish, and the New Golden Age of Software Engineering" (Medium) into agent skills. Its tagline is "Design with an elephant. Review with a goldfish." It has three roles:
- **Elephant:** an agent that holds the project context and drafts the specification.
- **Goldfish:** reviewer subagents started fresh, with no drafting conversation. Each one reads the draft and the source material it refers to. The plugin stops a review if it can't create such a fresh context.
- **Human:** makes the trade-offs and approves the spec.

The plugin's claim is that a reviewer who doesn't share the author's memory finds the assumptions the author never wrote down. Claude read the repository README on 2026-11-08. The essay returned HTTP 403 to automated fetching and hasn't been read yet, so cite it only after reading it.

## The relation to this port

Memorylessness is the same property in both places. Whether it helps depends on the role it fills:

- **On review it helps.** That's EGM's point. Dark Pawns already has the strongest possible goldfish: the C oracle. It has never seen any agent's conversation, and it judges only the bytes. R5h (revert proof) and R5j (a C witness line) make the human and agent reviewers act like goldfish too: they check against `src/`, not against the author's account.
- **On construction it hurts.** This port inverted EGM's arrangement. The builders were goldfish: hundreds of fresh coding-agent sessions over seven months, each starting with no memory of the code its predecessors wrote. Amnesiac clones (PF-114) are the price. A fresh builder can't know a port of `fname()` already exists, so it writes a second one, and the two disagree from birth.

So the candidate claim is that EGM's arrangement (elephant builds, goldfish reviews) is the safe one. A project that has to use goldfish builders needs a substitute elephant: a memory that lives in the repository, not in any agent. Dark Pawns has three partial substitutes:
- the C citations in Go comments
- the depth manifests
- the dead-code ratchet

The proposed C-citation duplicate ratchet would be a fourth (roadmap, 2026-11-08): CI fails when two Go functions cite the same C definition.

## What would test it

Count amnesiac clones per C function (via citations), and compare how they arise:
- clones created within one long session
- clones created across fresh sessions

If the framing holds, almost all clones arise across sessions.
