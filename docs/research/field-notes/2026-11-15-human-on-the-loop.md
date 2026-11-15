# Human on the loop: how the port is run

2026-11-15, morning. Field note by Claude, written with Zach while the weekend agents were still running. This note describes the method. The weekend's results get their own notes.

## What to call it

Zach found the name for this setup this morning. It's **human on the loop**, with **hierarchical oversight**. In a human-in-the-loop setup, a person approves each step before it happens. Here the agents act on their own, and the human watches the system, sets its limits and steps in when something needs his judgment. "Hierarchical" means there is a layer between them: one model reviews and routes the others' work, and the human oversees that layer.

The port wasn't designed from these terms. It grew into this shape, and the terms came after.

## The roles

- **Zach (human, on the loop).** He owns the rules and the irreversible decisions:
  - he changes the governing documents (`docs/fidelity/RULEBOOK.md`, `docs/fidelity/north-star-1to1.md`, `AGENTS.md`);
  - he rules on any divergence from C, such as the DP-1539 infobar ruling ("you make a solid case, agree");
  - he merges stop-tier PRs: harness code and `pkg/session` dispatch and lifecycle;
  - he hands each prompt to its agent and starts and stops the agents.
- **Claude (orchestrator, the middle layer).** Claude reads the queue and the evidence, then:
  - writes each agent's next prompt;
  - reviews every PR against C (`src/`, grepped in that session, R5g) and the census evidence, not against the agent's report;
  - fixes small CI reds and mechanical conflicts;
  - stacks PRs that touch the same files into one branch;
  - files Linear issues;
  - merges proof-only and game PRs after review, CI and a clean census;
  - escalates anything that needs a ruling, with the C cites and a recommendation.
- **Coding agents (implementers).** This weekend: DeepSeek, Hydrogen, GLM (two sessions, one of which was lost), Luna and Kimi, plus a batch of short-lived Flash sessions. Each works from a prompt in its own git worktree, opens a PR and ends with a report. Agents don't review each other, and they don't call other models (`AGENTS.md`, *Handoffs*).
- **The C oracle (the judge).** The original C server, built a fixed way, runs the same scripted scenario as the Go port, and the census compares the bytes each one sends to the player. A green census row means the two matched. With `--draw-parity` it also checks that both drew the same random numbers in the same order (R3).

## One trip around the loop

1. Claude writes a prompt from verified ground truth: C lines grepped that session, the Linear issue, the census run that showed the bug. It names a check that can fail, such as a scenario that is red on main.
2. Zach hands the prompt to an agent.
3. The agent writes a C-first witness scenario, fixes the Go code, keeps a revert proof (the witness goes red again when the fix is removed, R5h), runs a targeted census and opens a PR.
4. Claude reviews the diff against C, opens the census files (`summary.txt`, `results.tsv`, the divergence logs) and reruns what it doubts.
5. Then one of three things happens:
   - it merges;
   - it goes back to the agent with a follow-up prompt;
   - it goes to Zach, as a stop-tier merge or a question only he can rule on.
6. Before a release, Claude stacks the reviewed PRs and runs one full combined census on the exact commit being tagged.

The constraint that holds this together is that **no layer trusts the layer below it on its word.** Claude reviews against C and census output, not against the agent's summary. The agent tests against the C oracle, not against its own reading of C. Zach trusts the rules and the census more than either model.

## What review caught this weekend

These are from the PR threads and prompts of 2026-11-14 and 2026-11-15:

| PR | What the agent reported | What review found |
|---|---|---|
| #2675 (DeepSeek) | Its two new scenarios passed | It broke `switched-body-reply-newest-mob-depth`, which was green on main. The rewrite had replaced the zero-record arm of `reply` (`src/act.comm.c:952-958`). The stack census caught it, and the agent's census never ran that scenario. |
| #2675 and #2677 (DeepSeek) | Two separate, working branches | Both declared the same lock (`World.npcDummyMu`), so stacking them didn't compile. |
| #2676 (Luna) | Kuji Kiri works through switched bodies | It dropped C's ninja-class gate (`src/new_cmds.c:1559`). A mob `set` to ninja passes the gate in C and fails it in Go. |
| #2675, #2677, #2680 | Gates green | `make fidelity-depth` failed on all three: scenario annotations and manifest rows didn't match. |
| #2686 (Hydrogen) | Fixed the late `Okay.` | The cause and proof hold, but when two or more commands become ready together, C probably prints a prompt between them and Go prints one. *Not yet verified*: a C-first witness was requested on 2026-11-15. |

Not every catch went downward. Hydrogen's #2637 found that its own wider wait had caused the `quiet-mobs` timeouts, and fixed that cause instead of raising another limit.

## Scale

On 2026-11-13 through 2026-11-15, 142 PRs merged into `main` (GitHub search `merged:2026-11-13..2026-11-15`). Claude wrote 85 prompts in those three days (`~/PROMPT-2026-11-1[345]-*.md`, outside the repo). Every merge shows as `zax0rz`, because Zach and Claude use the same GitHub account, so the merge log can't tell who merged what. The PR threads can.

## What it costs

- **The human sleeps.** Luna's next prompt was written on 2026-11-14 but handed over the next morning, because Zach fell asleep first. Luna sat idle overnight. In Zach's words, a drawback of a human anywhere near the loop is "sleep/eat/etc." The orchestrator can't hand prompts over or start agents itself, so the loop runs only as fast as the human's hours.
- **Budgets run unevenly.** On the morning of 2026-11-15 the agents had very different amounts left: GLM 22% of its week, Hydrogen 58%, Luna 82% plus three banked resets. Claude routes work to fit: a small, focused bug for GLM, larger jobs for Luna.
- **The middle layer is a bottleneck too.** Every PR waits for Claude's review, and Claude's context fills up. This weekend's session was compacted at least once, and the summary had to carry state forward.
- **Tiering only works if the tiers are written down.** "Claude merges proof-only PRs" and "stop tier is Zach's" are recorded rules (memory and `AGENTS.md`). Without them every PR would wait for Zach, and the loop would turn back into human-in-the-loop.

## Open questions

- How much of the review catch rate comes from the C oracle, and how much from a second model reading the diff? The table suggests both: the reply regression came from a census, and the ninja gate came from reading C.
- Would agents reviewing each other's PRs catch the same things? This weekend they didn't, by rule.
- What does the overnight idle time cost in throughput? The prompt files' timestamps and the PRs' open times could measure it.

## Data not captured yet

Zach pointed out a gap: nobody counts **the work Claude sends back**. That covers follow-up prompts to an agent (`~/PROMPT-*-changes.md`, `*-reply.md` and the numbered weekend prompts), fixes Claude pushed to an agent's branch, and findings an agent reported but didn't file. Same day example: #2688 said it had filed a mob-regeneration finding, it hadn't, and Claude filed it as DP-1542. The evidence is all retained in Git (Claude's commits on agent branches), the PR comment threads, the prompt files and the census runs. A later pass could dig it out and give each agent a send-back rate.
