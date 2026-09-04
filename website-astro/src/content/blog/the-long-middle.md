---
title: "The Long Middle"
date: 2026-09-01T12:00:00
description: "Four and a half months and 122,000 lines of Go into porting a dead MUD, and the finish line keeps moving. A midway check-in on the difference between covering a codebase and actually proving it, told in the data."
draft: false
textKind: "original"
source: "Dark Pawns repository: git history (2,238 commits from 2026-04-17), docs/fidelity/ (RULEBOOK.md, depth/*.tsv), docs/reports/scenario-coverage-*.tsv, and PRs #243 / #601."
voiceLayer: "mythic-admin"
---

<figure class="frontispiece" style="max-width: 22rem; margin: 0 auto var(--space-lg); text-align: center;">
  <img src="/images/blog/the-long-middle.png" alt="Pen-and-ink spot illustration: a hooded wanderer with a staff walking a winding mountain path toward a low horizon sun" style="display: block; width: 100%; height: auto; margin: 0 auto;" />
</figure>

In April this year I found the original Dark Pawns source on GitHub, read the C files, and figured porting it to Go would take a few weeks. Four months in, it is clear I vastly underestimated the work.

Translating syntax is the easy part. The real work is proving that runtime behavior matches the original down to the byte. It seems obvious now.

## The estimate, and the slog

The first commit was almost 1,300 lines and landed on April 17. By June I had a server that booted, a character that could walk around, and a growing suspicion that "a few weeks" was completely off. The commit history shows the dip: June sagged as progress slowed and hidden architectural complexity surfaced. I was genuinely thinking about giving up.

On July 12 I landed the first differential test harness, a tool I called the oracle. A few days later, Anthropic published their process for [AI code migrations](https://claude.com/blog/ai-code-migration), validating the exact same judge-first approach. Seeing a frontier lab articulate the same problem reinvigorated the project. Commit volume climbed and stayed high. Each subsystem I completed exposed interactions and edge cases I could never see from the outside.

<figure class="chart">
<svg viewBox="0 0 680 300" width="100%" role="img" aria-label="Bar chart of commits per month: April 302, May 335, June 156, July 538, August 731." xmlns="http://www.w3.org/2000/svg" style="font-family:var(--font-mono, ui-monospace, 'JetBrains Mono', monospace);">
<line x1="8" y1="236" x2="672" y2="236" stroke="var(--ink, #1A1614)" stroke-width="1"/>
<rect x="43.4" y="157.5" width="62" height="78.5" fill="var(--ink-muted, #56504A)"/>
<text x="74.4" y="149.5" text-anchor="middle" font-size="15" font-weight="600" fill="var(--ink, #1A1614)">302</text>
<text x="74.4" y="256" text-anchor="middle" font-size="13" fill="var(--ink-muted, #56504A)">Apr</text>
<rect x="176.2" y="148.9" width="62" height="87.1" fill="var(--ink-muted, #56504A)"/>
<text x="207.2" y="140.9" text-anchor="middle" font-size="15" font-weight="600" fill="var(--ink, #1A1614)">335</text>
<text x="207.2" y="256" text-anchor="middle" font-size="13" fill="var(--ink-muted, #56504A)">May</text>
<rect x="309.0" y="195.5" width="62" height="40.5" fill="var(--ink-muted, #56504A)"/>
<text x="340.0" y="187.5" text-anchor="middle" font-size="15" font-weight="600" fill="var(--ink, #1A1614)">156</text>
<text x="340.0" y="256" text-anchor="middle" font-size="13" fill="var(--ink-muted, #56504A)">Jun</text>
<text x="340.0" y="276" text-anchor="middle" font-size="11" fill="var(--accent-deep, #7A1812)">the slog</text>
<rect x="441.8" y="96.2" width="62" height="139.8" fill="var(--accent, #A8201A)"/>
<text x="472.8" y="88.2" text-anchor="middle" font-size="15" font-weight="600" fill="var(--ink, #1A1614)">538</text>
<text x="472.8" y="256" text-anchor="middle" font-size="13" fill="var(--ink-muted, #56504A)">Jul</text>
<text x="472.8" y="276" text-anchor="middle" font-size="11" fill="var(--accent-deep, #7A1812)">oracle · Jul 12</text>
<rect x="574.6" y="46.0" width="62" height="190.0" fill="var(--accent, #A8201A)"/>
<text x="605.6" y="38.0" text-anchor="middle" font-size="15" font-weight="600" fill="var(--ink, #1A1614)">731</text>
<text x="605.6" y="256" text-anchor="middle" font-size="13" fill="var(--ink-muted, #56504A)">Aug</text>
<text x="605.6" y="276" text-anchor="middle" font-size="11" fill="var(--accent-deep, #7A1812)">depth · Aug 23</text>
<text x="8" y="24" font-size="12" fill="var(--ink-muted, #56504A)">estimate: "a few easy weeks"</text>
<line x1="8" y1="38" x2="672" y2="38" stroke="var(--ink-muted, #56504A)" stroke-width="1" stroke-dasharray="3 4"/>
</svg>
<figcaption>Commits per month.</figcaption>
</figure>

## Breadth is a trap

For the first stretch, progress meant coverage: getting every command to run without crashing, walking room initialization, and touching each system once. Dark Pawns has 508 registered commands, and checking them off gave a false sense of momentum.

The problem is that "it ran without crashing" and "it behaves correctly" are entirely different benchmarks. Of those 508 commands, 183 are socials (`smile`, `cackle`, `grovel`) that emit static strings and rarely fail. Including them pushed reported coverage to 58.7% without actually exercising gameplay mechanics.

I needed a deterministic way to verify correctness.

## The oracle

The oracle boots the original C server and the Go port side by side on local ports, drives both with identical scripted inputs, and compares their output byte-for-byte. When they agree, the scenario passes. When they disagree, the harness shows the exact byte where the outputs split.

Applying that principle to a MUD port enforces one rule: the game is the game. A player in 2026 should receive the exact same bytes and combat rolls as a player in 1999.

The PRNG stream is a clear example. When the first character logged into a fresh world, it became the implementor (max level). The Go port mistakenly ran a level-up routine that the C server skips, pulling two extra random draws at character creation. Because all game events share a single PRNG stream, every subsequent roll (combat hits, saving throws, skill checks) was offset by two positions. The server did not crash, but the entire simulation ran out of sync.

Three combat skills (`bash`, `trip`, `headbutt`) failed immediately in the oracle. Two other skills drew wrong numbers but happened to trigger the same branching outcome, passing by coincidence. Outcome-based unit tests would have masked the desynchronization; byte-level tracing caught the root offset immediately. That is something not even player testing would have uncovered.

## Depth

After building the oracle, the goal shifted from verifying that commands ran to proving every branch against the C implementation.

<figure class="chart">
<svg viewBox="0 0 680 268" width="100%" role="img" aria-label="Left: a grid of 508 command squares, 298 filled, one per command touched by the breadth sweep. Right: a skyline of 245 bars, one per command with a depth ledger, height showing how many behaviors are pinned against the original, tallest is 60." xmlns="http://www.w3.org/2000/svg" style="font-family:var(--font-mono, ui-monospace, 'JetBrains Mono', monospace);">
<text x="8" y="20" font-size="14" font-weight="600" fill="var(--ink, #1A1614)">Breadth</text>
<text x="8" y="40" font-size="12" fill="var(--ink-muted, #56504A)">one probe each &middot; 298 / 508 touched</text>
<text x="356" y="20" font-size="14" font-weight="600" fill="var(--ink, #1A1614)">Depth</text>
<text x="356" y="40" font-size="12" fill="var(--ink-muted, #56504A)">every behavior pinned &middot; 245 commands</text>
<rect x="8.0" y="56.0" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="18.9" y="56.0" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="29.8" y="56.0" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="40.7" y="56.0" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="51.6" y="56.0" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="62.5" y="56.0" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="73.4" y="56.0" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="84.3" y="56.0" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="95.2" y="56.0" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="106.1" y="56.0" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="117.0" y="56.0" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="127.9" y="56.0" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="138.8" y="56.0" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="149.7" y="56.0" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="160.6" y="56.0" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="171.5" y="56.0" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="182.4" y="56.0" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="193.3" y="56.0" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="204.2" y="56.0" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="215.1" y="56.0" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="226.0" y="56.0" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="236.9" y="56.0" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="247.8" y="56.0" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="258.7" y="56.0" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="269.6" y="56.0" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="280.5" y="56.0" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="291.4" y="56.0" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="302.3" y="56.0" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="313.2" y="56.0" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="8.0" y="66.9" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="18.9" y="66.9" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="29.8" y="66.9" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="40.7" y="66.9" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="51.6" y="66.9" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="62.5" y="66.9" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="73.4" y="66.9" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="84.3" y="66.9" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="95.2" y="66.9" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="106.1" y="66.9" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="117.0" y="66.9" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="127.9" y="66.9" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="138.8" y="66.9" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="149.7" y="66.9" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="160.6" y="66.9" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="171.5" y="66.9" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="182.4" y="66.9" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="193.3" y="66.9" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="204.2" y="66.9" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="215.1" y="66.9" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="226.0" y="66.9" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="236.9" y="66.9" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="247.8" y="66.9" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="258.7" y="66.9" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="269.6" y="66.9" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="280.5" y="66.9" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="291.4" y="66.9" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="302.3" y="66.9" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="313.2" y="66.9" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="8.0" y="77.8" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="18.9" y="77.8" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="29.8" y="77.8" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="40.7" y="77.8" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="51.6" y="77.8" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="62.5" y="77.8" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="73.4" y="77.8" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="84.3" y="77.8" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="95.2" y="77.8" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="106.1" y="77.8" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="117.0" y="77.8" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="127.9" y="77.8" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="138.8" y="77.8" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="149.7" y="77.8" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="160.6" y="77.8" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="171.5" y="77.8" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="182.4" y="77.8" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="193.3" y="77.8" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="204.2" y="77.8" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="215.1" y="77.8" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="226.0" y="77.8" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="236.9" y="77.8" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="247.8" y="77.8" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="258.7" y="77.8" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="269.6" y="77.8" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="280.5" y="77.8" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="291.4" y="77.8" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="302.3" y="77.8" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="313.2" y="77.8" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="8.0" y="88.7" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="18.9" y="88.7" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="29.8" y="88.7" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="40.7" y="88.7" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="51.6" y="88.7" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="62.5" y="88.7" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="73.4" y="88.7" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="84.3" y="88.7" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="95.2" y="88.7" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="106.1" y="88.7" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="117.0" y="88.7" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="127.9" y="88.7" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="138.8" y="88.7" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="149.7" y="88.7" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="160.6" y="88.7" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="171.5" y="88.7" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="182.4" y="88.7" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="193.3" y="88.7" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="204.2" y="88.7" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="215.1" y="88.7" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="226.0" y="88.7" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="236.9" y="88.7" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="247.8" y="88.7" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="258.7" y="88.7" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="269.6" y="88.7" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="280.5" y="88.7" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="291.4" y="88.7" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="302.3" y="88.7" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="313.2" y="88.7" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="8.0" y="99.6" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="18.9" y="99.6" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="29.8" y="99.6" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="40.7" y="99.6" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="51.6" y="99.6" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="62.5" y="99.6" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="73.4" y="99.6" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="84.3" y="99.6" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="95.2" y="99.6" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="106.1" y="99.6" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="117.0" y="99.6" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="127.9" y="99.6" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="138.8" y="99.6" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="149.7" y="99.6" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="160.6" y="99.6" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="171.5" y="99.6" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="182.4" y="99.6" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="193.3" y="99.6" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="204.2" y="99.6" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="215.1" y="99.6" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="226.0" y="99.6" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="236.9" y="99.6" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="247.8" y="99.6" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="258.7" y="99.6" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="269.6" y="99.6" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="280.5" y="99.6" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="291.4" y="99.6" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="302.3" y="99.6" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="313.2" y="99.6" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="8.0" y="110.5" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="18.9" y="110.5" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="29.8" y="110.5" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="40.7" y="110.5" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="51.6" y="110.5" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="62.5" y="110.5" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="73.4" y="110.5" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="84.3" y="110.5" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="95.2" y="110.5" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="106.1" y="110.5" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="117.0" y="110.5" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="127.9" y="110.5" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="138.8" y="110.5" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="149.7" y="110.5" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="160.6" y="110.5" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="171.5" y="110.5" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="182.4" y="110.5" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="193.3" y="110.5" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="204.2" y="110.5" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="215.1" y="110.5" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="226.0" y="110.5" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="236.9" y="110.5" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="247.8" y="110.5" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="258.7" y="110.5" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="269.6" y="110.5" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="280.5" y="110.5" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="291.4" y="110.5" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="302.3" y="110.5" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="313.2" y="110.5" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="8.0" y="121.4" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="18.9" y="121.4" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="29.8" y="121.4" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="40.7" y="121.4" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="51.6" y="121.4" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="62.5" y="121.4" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="73.4" y="121.4" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="84.3" y="121.4" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="95.2" y="121.4" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="106.1" y="121.4" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="117.0" y="121.4" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="127.9" y="121.4" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="138.8" y="121.4" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="149.7" y="121.4" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="160.6" y="121.4" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="171.5" y="121.4" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="182.4" y="121.4" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="193.3" y="121.4" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="204.2" y="121.4" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="215.1" y="121.4" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="226.0" y="121.4" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="236.9" y="121.4" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="247.8" y="121.4" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="258.7" y="121.4" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="269.6" y="121.4" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="280.5" y="121.4" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="291.4" y="121.4" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="302.3" y="121.4" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="313.2" y="121.4" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="8.0" y="132.3" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="18.9" y="132.3" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="29.8" y="132.3" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="40.7" y="132.3" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="51.6" y="132.3" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="62.5" y="132.3" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="73.4" y="132.3" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="84.3" y="132.3" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="95.2" y="132.3" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="106.1" y="132.3" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="117.0" y="132.3" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="127.9" y="132.3" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="138.8" y="132.3" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="149.7" y="132.3" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="160.6" y="132.3" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="171.5" y="132.3" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="182.4" y="132.3" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="193.3" y="132.3" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="204.2" y="132.3" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="215.1" y="132.3" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="226.0" y="132.3" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="236.9" y="132.3" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="247.8" y="132.3" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="258.7" y="132.3" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="269.6" y="132.3" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="280.5" y="132.3" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="291.4" y="132.3" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="302.3" y="132.3" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="313.2" y="132.3" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="8.0" y="143.2" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="18.9" y="143.2" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="29.8" y="143.2" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="40.7" y="143.2" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="51.6" y="143.2" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="62.5" y="143.2" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="73.4" y="143.2" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="84.3" y="143.2" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="95.2" y="143.2" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="106.1" y="143.2" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="117.0" y="143.2" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="127.9" y="143.2" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="138.8" y="143.2" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="149.7" y="143.2" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="160.6" y="143.2" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="171.5" y="143.2" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="182.4" y="143.2" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="193.3" y="143.2" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="204.2" y="143.2" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="215.1" y="143.2" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="226.0" y="143.2" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="236.9" y="143.2" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="247.8" y="143.2" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="258.7" y="143.2" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="269.6" y="143.2" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="280.5" y="143.2" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="291.4" y="143.2" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="302.3" y="143.2" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="313.2" y="143.2" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="8.0" y="154.1" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="18.9" y="154.1" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="29.8" y="154.1" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="40.7" y="154.1" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="51.6" y="154.1" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="62.5" y="154.1" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="73.4" y="154.1" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="84.3" y="154.1" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="95.2" y="154.1" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="106.1" y="154.1" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="117.0" y="154.1" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="127.9" y="154.1" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="138.8" y="154.1" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="149.7" y="154.1" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="160.6" y="154.1" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="171.5" y="154.1" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="182.4" y="154.1" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="193.3" y="154.1" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="204.2" y="154.1" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="215.1" y="154.1" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="226.0" y="154.1" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="236.9" y="154.1" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="247.8" y="154.1" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="258.7" y="154.1" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="269.6" y="154.1" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="280.5" y="154.1" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="291.4" y="154.1" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="302.3" y="154.1" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="313.2" y="154.1" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="8.0" y="165.0" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="18.9" y="165.0" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="29.8" y="165.0" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="40.7" y="165.0" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="51.6" y="165.0" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="62.5" y="165.0" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="73.4" y="165.0" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="84.3" y="165.0" width="9.0" height="9.0" fill="var(--accent, #A8201A)"/>
<rect x="95.2" y="165.0" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="106.1" y="165.0" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="117.0" y="165.0" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="127.9" y="165.0" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="138.8" y="165.0" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="149.7" y="165.0" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="160.6" y="165.0" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="171.5" y="165.0" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="182.4" y="165.0" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="193.3" y="165.0" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="204.2" y="165.0" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="215.1" y="165.0" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="226.0" y="165.0" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="236.9" y="165.0" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="247.8" y="165.0" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="258.7" y="165.0" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="269.6" y="165.0" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="280.5" y="165.0" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="291.4" y="165.0" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="302.3" y="165.0" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="313.2" y="165.0" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="8.0" y="175.9" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="18.9" y="175.9" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="29.8" y="175.9" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="40.7" y="175.9" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="51.6" y="175.9" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="62.5" y="175.9" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="73.4" y="175.9" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="84.3" y="175.9" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="95.2" y="175.9" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="106.1" y="175.9" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="117.0" y="175.9" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="127.9" y="175.9" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="138.8" y="175.9" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="149.7" y="175.9" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="160.6" y="175.9" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="171.5" y="175.9" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="182.4" y="175.9" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="193.3" y="175.9" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="204.2" y="175.9" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="215.1" y="175.9" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="226.0" y="175.9" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="236.9" y="175.9" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="247.8" y="175.9" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="258.7" y="175.9" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="269.6" y="175.9" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="280.5" y="175.9" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="291.4" y="175.9" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="302.3" y="175.9" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="313.2" y="175.9" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="8.0" y="186.8" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="18.9" y="186.8" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="29.8" y="186.8" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="40.7" y="186.8" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="51.6" y="186.8" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="62.5" y="186.8" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="73.4" y="186.8" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="84.3" y="186.8" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="95.2" y="186.8" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="106.1" y="186.8" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="117.0" y="186.8" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="127.9" y="186.8" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="138.8" y="186.8" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="149.7" y="186.8" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="160.6" y="186.8" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="171.5" y="186.8" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="182.4" y="186.8" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="193.3" y="186.8" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="204.2" y="186.8" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="215.1" y="186.8" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="226.0" y="186.8" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="236.9" y="186.8" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="247.8" y="186.8" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="258.7" y="186.8" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="269.6" y="186.8" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="280.5" y="186.8" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="291.4" y="186.8" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="302.3" y="186.8" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="313.2" y="186.8" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="8.0" y="197.7" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="18.9" y="197.7" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="29.8" y="197.7" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="40.7" y="197.7" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="51.6" y="197.7" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="62.5" y="197.7" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="73.4" y="197.7" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="84.3" y="197.7" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="95.2" y="197.7" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="106.1" y="197.7" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="117.0" y="197.7" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="127.9" y="197.7" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="138.8" y="197.7" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="149.7" y="197.7" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="160.6" y="197.7" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="171.5" y="197.7" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="182.4" y="197.7" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="193.3" y="197.7" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="204.2" y="197.7" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="215.1" y="197.7" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="226.0" y="197.7" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="236.9" y="197.7" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="247.8" y="197.7" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="258.7" y="197.7" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="269.6" y="197.7" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="280.5" y="197.7" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="291.4" y="197.7" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="302.3" y="197.7" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="313.2" y="197.7" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="8.0" y="208.6" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="18.9" y="208.6" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="29.8" y="208.6" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="40.7" y="208.6" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="51.6" y="208.6" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="62.5" y="208.6" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="73.4" y="208.6" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="84.3" y="208.6" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="95.2" y="208.6" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="106.1" y="208.6" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="117.0" y="208.6" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="127.9" y="208.6" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="138.8" y="208.6" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="149.7" y="208.6" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="160.6" y="208.6" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="171.5" y="208.6" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="182.4" y="208.6" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="193.3" y="208.6" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="204.2" y="208.6" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="215.1" y="208.6" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="226.0" y="208.6" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="236.9" y="208.6" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="247.8" y="208.6" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="258.7" y="208.6" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="269.6" y="208.6" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="280.5" y="208.6" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="291.4" y="208.6" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="302.3" y="208.6" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="313.2" y="208.6" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="8.0" y="219.5" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="18.9" y="219.5" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="29.8" y="219.5" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="40.7" y="219.5" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="51.6" y="219.5" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="62.5" y="219.5" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="73.4" y="219.5" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="84.3" y="219.5" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="95.2" y="219.5" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="106.1" y="219.5" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="117.0" y="219.5" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="127.9" y="219.5" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="138.8" y="219.5" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="149.7" y="219.5" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="160.6" y="219.5" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="171.5" y="219.5" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="182.4" y="219.5" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="193.3" y="219.5" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="204.2" y="219.5" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="215.1" y="219.5" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="226.0" y="219.5" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="236.9" y="219.5" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="247.8" y="219.5" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="258.7" y="219.5" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="269.6" y="219.5" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="280.5" y="219.5" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="291.4" y="219.5" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="302.3" y="219.5" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="313.2" y="219.5" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="8.0" y="230.4" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="18.9" y="230.4" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="29.8" y="230.4" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="40.7" y="230.4" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="51.6" y="230.4" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="62.5" y="230.4" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="73.4" y="230.4" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="84.3" y="230.4" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="95.2" y="230.4" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="106.1" y="230.4" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="117.0" y="230.4" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="127.9" y="230.4" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="138.8" y="230.4" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="149.7" y="230.4" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="160.6" y="230.4" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="171.5" y="230.4" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="182.4" y="230.4" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="193.3" y="230.4" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="204.2" y="230.4" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="215.1" y="230.4" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="226.0" y="230.4" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="236.9" y="230.4" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="247.8" y="230.4" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="258.7" y="230.4" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="269.6" y="230.4" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="280.5" y="230.4" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="291.4" y="230.4" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="302.3" y="230.4" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="313.2" y="230.4" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="8.0" y="241.3" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="18.9" y="241.3" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="29.8" y="241.3" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="40.7" y="241.3" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="51.6" y="241.3" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="62.5" y="241.3" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="73.4" y="241.3" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="84.3" y="241.3" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="95.2" y="241.3" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="106.1" y="241.3" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="117.0" y="241.3" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="127.9" y="241.3" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="138.8" y="241.3" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="149.7" y="241.3" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="160.6" y="241.3" width="9.0" height="9.0" fill="none" stroke="var(--ink-muted, #56504A)" stroke-width="0.75"/>
<rect x="356.00" y="56.0" width="0.95" height="196.2" fill="var(--accent-deep, #7A1812)"/>
<rect x="357.30" y="88.7" width="0.95" height="163.5" fill="var(--accent-deep, #7A1812)"/>
<rect x="358.59" y="111.6" width="0.95" height="140.6" fill="var(--accent-deep, #7A1812)"/>
<rect x="359.89" y="121.4" width="0.95" height="130.8" fill="var(--accent-deep, #7A1812)"/>
<rect x="361.18" y="134.5" width="0.95" height="117.7" fill="var(--accent-deep, #7A1812)"/>
<rect x="362.48" y="137.8" width="0.95" height="114.5" fill="var(--accent-deep, #7A1812)"/>
<rect x="363.77" y="147.6" width="0.95" height="104.6" fill="var(--accent-deep, #7A1812)"/>
<rect x="365.07" y="154.1" width="0.95" height="98.1" fill="var(--accent-deep, #7A1812)"/>
<rect x="366.36" y="160.6" width="0.95" height="91.6" fill="var(--accent-deep, #7A1812)"/>
<rect x="367.66" y="163.9" width="0.95" height="88.3" fill="var(--accent-deep, #7A1812)"/>
<rect x="368.95" y="170.4" width="0.95" height="81.8" fill="var(--accent-deep, #7A1812)"/>
<rect x="370.25" y="180.3" width="0.95" height="71.9" fill="var(--accent-deep, #7A1812)"/>
<rect x="371.54" y="180.3" width="0.95" height="71.9" fill="var(--accent-deep, #7A1812)"/>
<rect x="372.84" y="183.5" width="0.95" height="68.7" fill="var(--accent-deep, #7A1812)"/>
<rect x="374.13" y="183.5" width="0.95" height="68.7" fill="var(--accent-deep, #7A1812)"/>
<rect x="375.43" y="183.5" width="0.95" height="68.7" fill="var(--accent-deep, #7A1812)"/>
<rect x="376.72" y="183.5" width="0.95" height="68.7" fill="var(--accent-deep, #7A1812)"/>
<rect x="378.02" y="186.8" width="0.95" height="65.4" fill="var(--accent-deep, #7A1812)"/>
<rect x="379.31" y="190.1" width="0.95" height="62.1" fill="var(--accent-deep, #7A1812)"/>
<rect x="380.61" y="190.1" width="0.95" height="62.1" fill="var(--accent-deep, #7A1812)"/>
<rect x="381.90" y="190.1" width="0.95" height="62.1" fill="var(--accent-deep, #7A1812)"/>
<rect x="383.20" y="193.3" width="0.95" height="58.9" fill="var(--accent-deep, #7A1812)"/>
<rect x="384.49" y="193.3" width="0.95" height="58.9" fill="var(--accent-deep, #7A1812)"/>
<rect x="385.79" y="193.3" width="0.95" height="58.9" fill="var(--accent-deep, #7A1812)"/>
<rect x="387.08" y="193.3" width="0.95" height="58.9" fill="var(--accent-deep, #7A1812)"/>
<rect x="388.38" y="193.3" width="0.95" height="58.9" fill="var(--accent-deep, #7A1812)"/>
<rect x="389.67" y="196.6" width="0.95" height="55.6" fill="var(--accent-deep, #7A1812)"/>
<rect x="390.97" y="196.6" width="0.95" height="55.6" fill="var(--accent-deep, #7A1812)"/>
<rect x="392.26" y="196.6" width="0.95" height="55.6" fill="var(--accent-deep, #7A1812)"/>
<rect x="393.56" y="196.6" width="0.95" height="55.6" fill="var(--accent-deep, #7A1812)"/>
<rect x="394.85" y="196.6" width="0.95" height="55.6" fill="var(--accent-deep, #7A1812)"/>
<rect x="396.15" y="199.9" width="0.95" height="52.3" fill="var(--accent-deep, #7A1812)"/>
<rect x="397.44" y="199.9" width="0.95" height="52.3" fill="var(--accent-deep, #7A1812)"/>
<rect x="398.74" y="199.9" width="0.95" height="52.3" fill="var(--accent-deep, #7A1812)"/>
<rect x="400.03" y="199.9" width="0.95" height="52.3" fill="var(--accent-deep, #7A1812)"/>
<rect x="401.33" y="199.9" width="0.95" height="52.3" fill="var(--accent-deep, #7A1812)"/>
<rect x="402.62" y="203.2" width="0.95" height="49.1" fill="var(--accent-deep, #7A1812)"/>
<rect x="403.92" y="203.2" width="0.95" height="49.1" fill="var(--accent-deep, #7A1812)"/>
<rect x="405.21" y="203.2" width="0.95" height="49.1" fill="var(--accent-deep, #7A1812)"/>
<rect x="406.51" y="203.2" width="0.95" height="49.1" fill="var(--accent-deep, #7A1812)"/>
<rect x="407.80" y="203.2" width="0.95" height="49.1" fill="var(--accent-deep, #7A1812)"/>
<rect x="409.10" y="203.2" width="0.95" height="49.1" fill="var(--accent-deep, #7A1812)"/>
<rect x="410.39" y="203.2" width="0.95" height="49.1" fill="var(--accent-deep, #7A1812)"/>
<rect x="411.69" y="203.2" width="0.95" height="49.1" fill="var(--accent-deep, #7A1812)"/>
<rect x="412.98" y="203.2" width="0.95" height="49.1" fill="var(--accent-deep, #7A1812)"/>
<rect x="414.28" y="203.2" width="0.95" height="49.1" fill="var(--accent-deep, #7A1812)"/>
<rect x="415.57" y="203.2" width="0.95" height="49.1" fill="var(--accent-deep, #7A1812)"/>
<rect x="416.87" y="206.4" width="0.95" height="45.8" fill="var(--accent-deep, #7A1812)"/>
<rect x="418.16" y="206.4" width="0.95" height="45.8" fill="var(--accent-deep, #7A1812)"/>
<rect x="419.46" y="206.4" width="0.95" height="45.8" fill="var(--accent-deep, #7A1812)"/>
<rect x="420.75" y="206.4" width="0.95" height="45.8" fill="var(--accent-deep, #7A1812)"/>
<rect x="422.05" y="206.4" width="0.95" height="45.8" fill="var(--accent-deep, #7A1812)"/>
<rect x="423.34" y="206.4" width="0.95" height="45.8" fill="var(--accent-deep, #7A1812)"/>
<rect x="424.64" y="206.4" width="0.95" height="45.8" fill="var(--accent-deep, #7A1812)"/>
<rect x="425.93" y="206.4" width="0.95" height="45.8" fill="var(--accent-deep, #7A1812)"/>
<rect x="427.23" y="206.4" width="0.95" height="45.8" fill="var(--accent-deep, #7A1812)"/>
<rect x="428.52" y="206.4" width="0.95" height="45.8" fill="var(--accent-deep, #7A1812)"/>
<rect x="429.82" y="206.4" width="0.95" height="45.8" fill="var(--accent-deep, #7A1812)"/>
<rect x="431.11" y="206.4" width="0.95" height="45.8" fill="var(--accent-deep, #7A1812)"/>
<rect x="432.41" y="206.4" width="0.95" height="45.8" fill="var(--accent-deep, #7A1812)"/>
<rect x="433.70" y="206.4" width="0.95" height="45.8" fill="var(--accent-deep, #7A1812)"/>
<rect x="435.00" y="206.4" width="0.95" height="45.8" fill="var(--accent-deep, #7A1812)"/>
<rect x="436.30" y="209.7" width="0.95" height="42.5" fill="var(--accent-deep, #7A1812)"/>
<rect x="437.59" y="209.7" width="0.95" height="42.5" fill="var(--accent-deep, #7A1812)"/>
<rect x="438.89" y="209.7" width="0.95" height="42.5" fill="var(--accent-deep, #7A1812)"/>
<rect x="440.18" y="209.7" width="0.95" height="42.5" fill="var(--accent-deep, #7A1812)"/>
<rect x="441.48" y="209.7" width="0.95" height="42.5" fill="var(--accent-deep, #7A1812)"/>
<rect x="442.77" y="209.7" width="0.95" height="42.5" fill="var(--accent-deep, #7A1812)"/>
<rect x="444.07" y="213.0" width="0.95" height="39.2" fill="var(--accent-deep, #7A1812)"/>
<rect x="445.36" y="213.0" width="0.95" height="39.2" fill="var(--accent-deep, #7A1812)"/>
<rect x="446.66" y="213.0" width="0.95" height="39.2" fill="var(--accent-deep, #7A1812)"/>
<rect x="447.95" y="213.0" width="0.95" height="39.2" fill="var(--accent-deep, #7A1812)"/>
<rect x="449.25" y="213.0" width="0.95" height="39.2" fill="var(--accent-deep, #7A1812)"/>
<rect x="450.54" y="213.0" width="0.95" height="39.2" fill="var(--accent-deep, #7A1812)"/>
<rect x="451.84" y="213.0" width="0.95" height="39.2" fill="var(--accent-deep, #7A1812)"/>
<rect x="453.13" y="213.0" width="0.95" height="39.2" fill="var(--accent-deep, #7A1812)"/>
<rect x="454.43" y="213.0" width="0.95" height="39.2" fill="var(--accent-deep, #7A1812)"/>
<rect x="455.72" y="213.0" width="0.95" height="39.2" fill="var(--accent-deep, #7A1812)"/>
<rect x="457.02" y="213.0" width="0.95" height="39.2" fill="var(--accent-deep, #7A1812)"/>
<rect x="458.31" y="213.0" width="0.95" height="39.2" fill="var(--accent-deep, #7A1812)"/>
<rect x="459.61" y="216.2" width="0.95" height="36.0" fill="var(--accent-deep, #7A1812)"/>
<rect x="460.90" y="216.2" width="0.95" height="36.0" fill="var(--accent-deep, #7A1812)"/>
<rect x="462.20" y="216.2" width="0.95" height="36.0" fill="var(--accent-deep, #7A1812)"/>
<rect x="463.49" y="216.2" width="0.95" height="36.0" fill="var(--accent-deep, #7A1812)"/>
<rect x="464.79" y="216.2" width="0.95" height="36.0" fill="var(--accent-deep, #7A1812)"/>
<rect x="466.08" y="216.2" width="0.95" height="36.0" fill="var(--accent-deep, #7A1812)"/>
<rect x="467.38" y="216.2" width="0.95" height="36.0" fill="var(--accent-deep, #7A1812)"/>
<rect x="468.67" y="216.2" width="0.95" height="36.0" fill="var(--accent-deep, #7A1812)"/>
<rect x="469.97" y="216.2" width="0.95" height="36.0" fill="var(--accent-deep, #7A1812)"/>
<rect x="471.26" y="216.2" width="0.95" height="36.0" fill="var(--accent-deep, #7A1812)"/>
<rect x="472.56" y="216.2" width="0.95" height="36.0" fill="var(--accent-deep, #7A1812)"/>
<rect x="473.85" y="216.2" width="0.95" height="36.0" fill="var(--accent-deep, #7A1812)"/>
<rect x="475.15" y="216.2" width="0.95" height="36.0" fill="var(--accent-deep, #7A1812)"/>
<rect x="476.44" y="216.2" width="0.95" height="36.0" fill="var(--accent-deep, #7A1812)"/>
<rect x="477.74" y="216.2" width="0.95" height="36.0" fill="var(--accent-deep, #7A1812)"/>
<rect x="479.03" y="216.2" width="0.95" height="36.0" fill="var(--accent-deep, #7A1812)"/>
<rect x="480.33" y="216.2" width="0.95" height="36.0" fill="var(--accent-deep, #7A1812)"/>
<rect x="481.62" y="216.2" width="0.95" height="36.0" fill="var(--accent-deep, #7A1812)"/>
<rect x="482.92" y="216.2" width="0.95" height="36.0" fill="var(--accent-deep, #7A1812)"/>
<rect x="484.21" y="216.2" width="0.95" height="36.0" fill="var(--accent-deep, #7A1812)"/>
<rect x="485.51" y="216.2" width="0.95" height="36.0" fill="var(--accent-deep, #7A1812)"/>
<rect x="486.80" y="216.2" width="0.95" height="36.0" fill="var(--accent-deep, #7A1812)"/>
<rect x="488.10" y="216.2" width="0.95" height="36.0" fill="var(--accent-deep, #7A1812)"/>
<rect x="489.39" y="216.2" width="0.95" height="36.0" fill="var(--accent-deep, #7A1812)"/>
<rect x="490.69" y="216.2" width="0.95" height="36.0" fill="var(--accent-deep, #7A1812)"/>
<rect x="491.98" y="216.2" width="0.95" height="36.0" fill="var(--accent-deep, #7A1812)"/>
<rect x="493.28" y="216.2" width="0.95" height="36.0" fill="var(--accent-deep, #7A1812)"/>
<rect x="494.57" y="216.2" width="0.95" height="36.0" fill="var(--accent-deep, #7A1812)"/>
<rect x="495.87" y="216.2" width="0.95" height="36.0" fill="var(--accent-deep, #7A1812)"/>
<rect x="497.16" y="216.2" width="0.95" height="36.0" fill="var(--accent-deep, #7A1812)"/>
<rect x="498.46" y="219.5" width="0.95" height="32.7" fill="var(--accent-deep, #7A1812)"/>
<rect x="499.75" y="219.5" width="0.95" height="32.7" fill="var(--accent-deep, #7A1812)"/>
<rect x="501.05" y="219.5" width="0.95" height="32.7" fill="var(--accent-deep, #7A1812)"/>
<rect x="502.34" y="219.5" width="0.95" height="32.7" fill="var(--accent-deep, #7A1812)"/>
<rect x="503.64" y="219.5" width="0.95" height="32.7" fill="var(--accent-deep, #7A1812)"/>
<rect x="504.93" y="219.5" width="0.95" height="32.7" fill="var(--accent-deep, #7A1812)"/>
<rect x="506.23" y="219.5" width="0.95" height="32.7" fill="var(--accent-deep, #7A1812)"/>
<rect x="507.52" y="219.5" width="0.95" height="32.7" fill="var(--accent-deep, #7A1812)"/>
<rect x="508.82" y="219.5" width="0.95" height="32.7" fill="var(--accent-deep, #7A1812)"/>
<rect x="510.11" y="219.5" width="0.95" height="32.7" fill="var(--accent-deep, #7A1812)"/>
<rect x="511.41" y="219.5" width="0.95" height="32.7" fill="var(--accent-deep, #7A1812)"/>
<rect x="512.70" y="219.5" width="0.95" height="32.7" fill="var(--accent-deep, #7A1812)"/>
<rect x="514.00" y="219.5" width="0.95" height="32.7" fill="var(--accent-deep, #7A1812)"/>
<rect x="515.30" y="219.5" width="0.95" height="32.7" fill="var(--accent-deep, #7A1812)"/>
<rect x="516.59" y="219.5" width="0.95" height="32.7" fill="var(--accent-deep, #7A1812)"/>
<rect x="517.89" y="219.5" width="0.95" height="32.7" fill="var(--accent-deep, #7A1812)"/>
<rect x="519.18" y="219.5" width="0.95" height="32.7" fill="var(--accent-deep, #7A1812)"/>
<rect x="520.48" y="219.5" width="0.95" height="32.7" fill="var(--accent-deep, #7A1812)"/>
<rect x="521.77" y="219.5" width="0.95" height="32.7" fill="var(--accent-deep, #7A1812)"/>
<rect x="523.07" y="219.5" width="0.95" height="32.7" fill="var(--accent-deep, #7A1812)"/>
<rect x="524.36" y="219.5" width="0.95" height="32.7" fill="var(--accent-deep, #7A1812)"/>
<rect x="525.66" y="219.5" width="0.95" height="32.7" fill="var(--accent-deep, #7A1812)"/>
<rect x="526.95" y="219.5" width="0.95" height="32.7" fill="var(--accent-deep, #7A1812)"/>
<rect x="528.25" y="219.5" width="0.95" height="32.7" fill="var(--accent-deep, #7A1812)"/>
<rect x="529.54" y="219.5" width="0.95" height="32.7" fill="var(--accent-deep, #7A1812)"/>
<rect x="530.84" y="222.8" width="0.95" height="29.4" fill="var(--accent-deep, #7A1812)"/>
<rect x="532.13" y="222.8" width="0.95" height="29.4" fill="var(--accent-deep, #7A1812)"/>
<rect x="533.43" y="222.8" width="0.95" height="29.4" fill="var(--accent-deep, #7A1812)"/>
<rect x="534.72" y="222.8" width="0.95" height="29.4" fill="var(--accent-deep, #7A1812)"/>
<rect x="536.02" y="222.8" width="0.95" height="29.4" fill="var(--accent-deep, #7A1812)"/>
<rect x="537.31" y="222.8" width="0.95" height="29.4" fill="var(--accent-deep, #7A1812)"/>
<rect x="538.61" y="222.8" width="0.95" height="29.4" fill="var(--accent-deep, #7A1812)"/>
<rect x="539.90" y="222.8" width="0.95" height="29.4" fill="var(--accent-deep, #7A1812)"/>
<rect x="541.20" y="222.8" width="0.95" height="29.4" fill="var(--accent-deep, #7A1812)"/>
<rect x="542.49" y="222.8" width="0.95" height="29.4" fill="var(--accent-deep, #7A1812)"/>
<rect x="543.79" y="222.8" width="0.95" height="29.4" fill="var(--accent-deep, #7A1812)"/>
<rect x="545.08" y="222.8" width="0.95" height="29.4" fill="var(--accent-deep, #7A1812)"/>
<rect x="546.38" y="222.8" width="0.95" height="29.4" fill="var(--accent-deep, #7A1812)"/>
<rect x="547.67" y="222.8" width="0.95" height="29.4" fill="var(--accent-deep, #7A1812)"/>
<rect x="548.97" y="226.0" width="0.95" height="26.2" fill="var(--accent-deep, #7A1812)"/>
<rect x="550.26" y="226.0" width="0.95" height="26.2" fill="var(--accent-deep, #7A1812)"/>
<rect x="551.56" y="226.0" width="0.95" height="26.2" fill="var(--accent-deep, #7A1812)"/>
<rect x="552.85" y="226.0" width="0.95" height="26.2" fill="var(--accent-deep, #7A1812)"/>
<rect x="554.15" y="226.0" width="0.95" height="26.2" fill="var(--accent-deep, #7A1812)"/>
<rect x="555.44" y="226.0" width="0.95" height="26.2" fill="var(--accent-deep, #7A1812)"/>
<rect x="556.74" y="226.0" width="0.95" height="26.2" fill="var(--accent-deep, #7A1812)"/>
<rect x="558.03" y="226.0" width="0.95" height="26.2" fill="var(--accent-deep, #7A1812)"/>
<rect x="559.33" y="226.0" width="0.95" height="26.2" fill="var(--accent-deep, #7A1812)"/>
<rect x="560.62" y="226.0" width="0.95" height="26.2" fill="var(--accent-deep, #7A1812)"/>
<rect x="561.92" y="226.0" width="0.95" height="26.2" fill="var(--accent-deep, #7A1812)"/>
<rect x="563.21" y="226.0" width="0.95" height="26.2" fill="var(--accent-deep, #7A1812)"/>
<rect x="564.51" y="226.0" width="0.95" height="26.2" fill="var(--accent-deep, #7A1812)"/>
<rect x="565.80" y="226.0" width="0.95" height="26.2" fill="var(--accent-deep, #7A1812)"/>
<rect x="567.10" y="226.0" width="0.95" height="26.2" fill="var(--accent-deep, #7A1812)"/>
<rect x="568.39" y="226.0" width="0.95" height="26.2" fill="var(--accent-deep, #7A1812)"/>
<rect x="569.69" y="226.0" width="0.95" height="26.2" fill="var(--accent-deep, #7A1812)"/>
<rect x="570.98" y="226.0" width="0.95" height="26.2" fill="var(--accent-deep, #7A1812)"/>
<rect x="572.28" y="226.0" width="0.95" height="26.2" fill="var(--accent-deep, #7A1812)"/>
<rect x="573.57" y="226.0" width="0.95" height="26.2" fill="var(--accent-deep, #7A1812)"/>
<rect x="574.87" y="226.0" width="0.95" height="26.2" fill="var(--accent-deep, #7A1812)"/>
<rect x="576.16" y="226.0" width="0.95" height="26.2" fill="var(--accent-deep, #7A1812)"/>
<rect x="577.46" y="226.0" width="0.95" height="26.2" fill="var(--accent-deep, #7A1812)"/>
<rect x="578.75" y="229.3" width="0.95" height="22.9" fill="var(--accent-deep, #7A1812)"/>
<rect x="580.05" y="229.3" width="0.95" height="22.9" fill="var(--accent-deep, #7A1812)"/>
<rect x="581.34" y="229.3" width="0.95" height="22.9" fill="var(--accent-deep, #7A1812)"/>
<rect x="582.64" y="229.3" width="0.95" height="22.9" fill="var(--accent-deep, #7A1812)"/>
<rect x="583.93" y="229.3" width="0.95" height="22.9" fill="var(--accent-deep, #7A1812)"/>
<rect x="585.23" y="229.3" width="0.95" height="22.9" fill="var(--accent-deep, #7A1812)"/>
<rect x="586.52" y="229.3" width="0.95" height="22.9" fill="var(--accent-deep, #7A1812)"/>
<rect x="587.82" y="229.3" width="0.95" height="22.9" fill="var(--accent-deep, #7A1812)"/>
<rect x="589.11" y="229.3" width="0.95" height="22.9" fill="var(--accent-deep, #7A1812)"/>
<rect x="590.41" y="229.3" width="0.95" height="22.9" fill="var(--accent-deep, #7A1812)"/>
<rect x="591.70" y="229.3" width="0.95" height="22.9" fill="var(--accent-deep, #7A1812)"/>
<rect x="593.00" y="229.3" width="0.95" height="22.9" fill="var(--accent-deep, #7A1812)"/>
<rect x="594.30" y="229.3" width="0.95" height="22.9" fill="var(--accent-deep, #7A1812)"/>
<rect x="595.59" y="229.3" width="0.95" height="22.9" fill="var(--accent-deep, #7A1812)"/>
<rect x="596.89" y="229.3" width="0.95" height="22.9" fill="var(--accent-deep, #7A1812)"/>
<rect x="598.18" y="229.3" width="0.95" height="22.9" fill="var(--accent-deep, #7A1812)"/>
<rect x="599.48" y="229.3" width="0.95" height="22.9" fill="var(--accent-deep, #7A1812)"/>
<rect x="600.77" y="229.3" width="0.95" height="22.9" fill="var(--accent-deep, #7A1812)"/>
<rect x="602.07" y="229.3" width="0.95" height="22.9" fill="var(--accent-deep, #7A1812)"/>
<rect x="603.36" y="232.6" width="0.95" height="19.6" fill="var(--accent-deep, #7A1812)"/>
<rect x="604.66" y="232.6" width="0.95" height="19.6" fill="var(--accent-deep, #7A1812)"/>
<rect x="605.95" y="232.6" width="0.95" height="19.6" fill="var(--accent-deep, #7A1812)"/>
<rect x="607.25" y="232.6" width="0.95" height="19.6" fill="var(--accent-deep, #7A1812)"/>
<rect x="608.54" y="232.6" width="0.95" height="19.6" fill="var(--accent-deep, #7A1812)"/>
<rect x="609.84" y="232.6" width="0.95" height="19.6" fill="var(--accent-deep, #7A1812)"/>
<rect x="611.13" y="232.6" width="0.95" height="19.6" fill="var(--accent-deep, #7A1812)"/>
<rect x="612.43" y="232.6" width="0.95" height="19.6" fill="var(--accent-deep, #7A1812)"/>
<rect x="613.72" y="232.6" width="0.95" height="19.6" fill="var(--accent-deep, #7A1812)"/>
<rect x="615.02" y="232.6" width="0.95" height="19.6" fill="var(--accent-deep, #7A1812)"/>
<rect x="616.31" y="232.6" width="0.95" height="19.6" fill="var(--accent-deep, #7A1812)"/>
<rect x="617.61" y="232.6" width="0.95" height="19.6" fill="var(--accent-deep, #7A1812)"/>
<rect x="618.90" y="232.6" width="0.95" height="19.6" fill="var(--accent-deep, #7A1812)"/>
<rect x="620.20" y="232.6" width="0.95" height="19.6" fill="var(--accent-deep, #7A1812)"/>
<rect x="621.49" y="232.6" width="0.95" height="19.6" fill="var(--accent-deep, #7A1812)"/>
<rect x="622.79" y="232.6" width="0.95" height="19.6" fill="var(--accent-deep, #7A1812)"/>
<rect x="624.08" y="232.6" width="0.95" height="19.6" fill="var(--accent-deep, #7A1812)"/>
<rect x="625.38" y="232.6" width="0.95" height="19.6" fill="var(--accent-deep, #7A1812)"/>
<rect x="626.67" y="235.9" width="0.95" height="16.4" fill="var(--accent-deep, #7A1812)"/>
<rect x="627.97" y="235.9" width="0.95" height="16.4" fill="var(--accent-deep, #7A1812)"/>
<rect x="629.26" y="235.9" width="0.95" height="16.4" fill="var(--accent-deep, #7A1812)"/>
<rect x="630.56" y="235.9" width="0.95" height="16.4" fill="var(--accent-deep, #7A1812)"/>
<rect x="631.85" y="235.9" width="0.95" height="16.4" fill="var(--accent-deep, #7A1812)"/>
<rect x="633.15" y="235.9" width="0.95" height="16.4" fill="var(--accent-deep, #7A1812)"/>
<rect x="634.44" y="235.9" width="0.95" height="16.4" fill="var(--accent-deep, #7A1812)"/>
<rect x="635.74" y="235.9" width="0.95" height="16.4" fill="var(--accent-deep, #7A1812)"/>
<rect x="637.03" y="235.9" width="0.95" height="16.4" fill="var(--accent-deep, #7A1812)"/>
<rect x="638.33" y="235.9" width="0.95" height="16.4" fill="var(--accent-deep, #7A1812)"/>
<rect x="639.62" y="239.1" width="0.95" height="13.1" fill="var(--accent-deep, #7A1812)"/>
<rect x="640.92" y="239.1" width="0.95" height="13.1" fill="var(--accent-deep, #7A1812)"/>
<rect x="642.21" y="239.1" width="0.95" height="13.1" fill="var(--accent-deep, #7A1812)"/>
<rect x="643.51" y="239.1" width="0.95" height="13.1" fill="var(--accent-deep, #7A1812)"/>
<rect x="644.80" y="239.1" width="0.95" height="13.1" fill="var(--accent-deep, #7A1812)"/>
<rect x="646.10" y="239.1" width="0.95" height="13.1" fill="var(--accent-deep, #7A1812)"/>
<rect x="647.39" y="239.1" width="0.95" height="13.1" fill="var(--accent-deep, #7A1812)"/>
<rect x="648.69" y="239.1" width="0.95" height="13.1" fill="var(--accent-deep, #7A1812)"/>
<rect x="649.98" y="239.1" width="0.95" height="13.1" fill="var(--accent-deep, #7A1812)"/>
<rect x="651.28" y="239.1" width="0.95" height="13.1" fill="var(--accent-deep, #7A1812)"/>
<rect x="652.57" y="239.1" width="0.95" height="13.1" fill="var(--accent-deep, #7A1812)"/>
<rect x="653.87" y="239.1" width="0.95" height="13.1" fill="var(--accent-deep, #7A1812)"/>
<rect x="655.16" y="239.1" width="0.95" height="13.1" fill="var(--accent-deep, #7A1812)"/>
<rect x="656.46" y="239.1" width="0.95" height="13.1" fill="var(--accent-deep, #7A1812)"/>
<rect x="657.75" y="239.1" width="0.95" height="13.1" fill="var(--accent-deep, #7A1812)"/>
<rect x="659.05" y="239.1" width="0.95" height="13.1" fill="var(--accent-deep, #7A1812)"/>
<rect x="660.34" y="239.1" width="0.95" height="13.1" fill="var(--accent-deep, #7A1812)"/>
<rect x="661.64" y="242.4" width="0.95" height="9.8" fill="var(--accent-deep, #7A1812)"/>
<rect x="662.93" y="242.4" width="0.95" height="9.8" fill="var(--accent-deep, #7A1812)"/>
<rect x="664.23" y="242.4" width="0.95" height="9.8" fill="var(--accent-deep, #7A1812)"/>
<rect x="665.52" y="242.4" width="0.95" height="9.8" fill="var(--accent-deep, #7A1812)"/>
<rect x="666.82" y="242.4" width="0.95" height="9.8" fill="var(--accent-deep, #7A1812)"/>
<rect x="668.11" y="245.7" width="0.95" height="6.5" fill="var(--accent-deep, #7A1812)"/>
<rect x="669.41" y="245.7" width="0.95" height="6.5" fill="var(--accent-deep, #7A1812)"/>
<rect x="670.70" y="248.9" width="0.95" height="3.3" fill="var(--accent-deep, #7A1812)"/>
<line x1="356" y1="252" x2="672" y2="252" stroke="var(--ink, #1A1614)" stroke-width="1"/>
<text x="672" y="70" text-anchor="end" font-size="11" fill="var(--accent-deep, #7A1812)">tallest: object-magic, 60 cases</text>
</svg>
<figcaption style="display: grid; grid-template-columns: 1fr 1fr; gap: var(--space-lg);">
  <div><strong>Breadth:</strong> 298 of 508 commands probed once (including 183 trivial socials).</div>
  <div><strong>Depth:</strong> 3,260 behavioral cases across 245 commands pinned byte-for-byte to C.</div>
</figcaption>
</figure>

Each bar in the right panel represents one command and the number of distinct behavioral cases pinned against the original C implementation (missing targets, peaceful rooms, fighting states, inventory constraints). `bash` alone requires a dozen cases. Today the depth ledgers track 3,260 catalogued cases across 245 commands, with 2,849 proven against the C binary and the rest categorized into unit tests, out of scope, or blocked. Each case cites the exact file and line in the original source that it matches. This is where August’s 731 commits went: one command at a time across 153 PRs.

Depth has its own gradient. Not every case is equally hard to prove.

<figure class="chart">
<svg viewBox="0 0 680 250" width="100%" role="img" aria-label="Horizontal bars of depth cases per tier: D1 944, D2 1046, D3 742, D4 325, D5 203." xmlns="http://www.w3.org/2000/svg" style="font-family:var(--font-mono, ui-monospace, 'JetBrains Mono', monospace);">
<text x="8" y="18" font-size="12" fill="var(--ink-muted, #56504A)">3,260 cases by depth tier. the deepest tiers are the thinnest, and the hardest-won</text>
<text x="8" y="52" font-size="14" font-weight="600" fill="var(--ink, #1A1614)">D1</text>
<text x="46" y="52" font-size="12" fill="var(--ink-muted, #56504A)">surface refusals & parse</text>
<rect x="252" y="38" width="323.1" height="22" fill="#F0997B"/>
<text x="583.1" y="54" font-size="13" fill="var(--ink, #1A1614)">944</text>
<text x="8" y="92" font-size="14" font-weight="600" fill="var(--ink, #1A1614)">D2</text>
<text x="46" y="92" font-size="12" fill="var(--ink-muted, #56504A)">core behavior</text>
<rect x="252" y="78" width="358.0" height="22" fill="#E5714A"/>
<text x="618.0" y="94" font-size="13" fill="var(--ink, #1A1614)">1046</text>
<text x="8" y="132" font-size="14" font-weight="600" fill="var(--ink, #1A1614)">D3</text>
<text x="46" y="132" font-size="12" fill="var(--ink-muted, #56504A)">state & interaction</text>
<rect x="252" y="118" width="254.0" height="22" fill="var(--accent, #A8201A)"/>
<text x="514.0" y="134" font-size="13" fill="var(--ink, #1A1614)">742</text>
<text x="8" y="172" font-size="14" font-weight="600" fill="var(--ink, #1A1614)">D4</text>
<text x="46" y="172" font-size="12" fill="var(--ink-muted, #56504A)">edge & timing</text>
<rect x="252" y="158" width="111.2" height="22" fill="var(--accent-deep, #7A1812)"/>
<text x="371.2" y="174" font-size="13" fill="var(--ink, #1A1614)">325</text>
<text x="8" y="212" font-size="14" font-weight="600" fill="var(--ink, #1A1614)">D5</text>
<text x="46" y="212" font-size="12" fill="var(--ink-muted, #56504A)">deep divergence</text>
<rect x="252" y="198" width="69.5" height="22" fill="#5A150F"/>
<text x="329.5" y="214" font-size="13" fill="var(--ink, #1A1614)">203</text>
</svg>
<figcaption>3,260 depth cases by tier, from surface syntax (D1) to multi-session state drift (D5).</figcaption>
</figure>

We track these cases across five depth tiers, labeled D1 through D5. The shallow tiers (D1 entry gates and D2 core outcomes) are straightforward: argument parsing and basic command dispatch. The middle tiers (D3 state mutations and D4 edge/timing boundaries) get harder: complex multi-entity interactions, container nesting, and tick-boundary events where world state has to mutate identically. The deepest cases (D5) are where two servers run through a long shared session and drift apart by a single byte hundreds of function calls later. Those take an afternoon of tracing to verify, but that is where the subtle regressions hide.

## Current status

Basic command breadth is largely in place. The remaining work is depth, specifically the spec-procs: 528 catalogued cases covering mob AI, shopkeepers, guildmasters, breath weapons, and quest logic.

Building the judge first and proving behavior at the byte level turns a massive legacy port into a tractable verification problem. It catches regressions immediately and ensures that modern Go produces the exact behavior of the 1990s original. Porting a game isn't finished when the Go binary compiles or when a player can walk around the temple. The real work is in proving that when an assassin backstabs or a spell is cast, the world responds with the exact same numbers it did twenty years ago.

---

*If you want to help close the gap on the remaining depth ledgers and mob AI, take a look at the [Contributing Guide](/docs/server/contributing/) or grab an open issue on [GitHub](https://github.com/zax0rz/darkpawns). You can also connect to the live port via telnet at `darkpawns.labz0rz.com 7777` or test it in your browser at [/play](/play/).*
