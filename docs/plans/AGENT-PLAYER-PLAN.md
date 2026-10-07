# Run Chungus on the VPS, plan remotely

> Plan: agent players for Dark Pawns (System One decision model + remote planner + agent wire). Target path: `docs/plans/AGENT-PLAYER-PLAN.md`. Continues `docs/plans/TRANSPORT-MCP-RESEARCH.md` (July 2026) and amends it. Research date 2026-10-06.

**Bottom line.** Build "Chungus the Warrior" as a small (0.5–0.8B) decoder LM served by `llama-server` on the game VPS. It emits exactly one parser command per decision under a grammar rebuilt every tick from player-visible text, so every output is well-formed by construction. A remote frontier-LLM planner hands it a typed "plan card" every 75–150 s or on an event, and is never on the per-round critical path. The tactical loop (observation parser, grammar builder, deterministic abort rules, model call) runs in a separate `chungus-runner` process on the VPS. That process logs into the game through the same agent socket and rate limits as any remote agent, so wide-area latency stays off the 2 s combat-round budget and the agent gets no privileged access. Three of the brief's premises are wrong and change the design. First, **Laya, Jev, CLM-8B and the OpenAI Decisions API all score options the caller supplies; none of them generates commands**. **s1decide could not be found**, and the Decisions API has no public schema. This plan therefore defaults to constrained decoding, and Phase 3 runs a Laya-picker bake-off that can overturn that default. Second, **the MCP 2026-07-28 revision removed the session, resumability and subscribe mechanisms the July transport spec was built on**, and Claude Code drops resource-update notifications. Session identity and replay therefore move to the application layer (a `since_seq` cursor and long-poll). Third, **every timing constant here comes from tbaMUD, CircleMUD and Merc, not from the Go source**, and every CPU latency figure is an estimate scaled from benchmarks of larger models. That is why Phase 0 verifies constants and benchmarks on the VPS before any other phase commits to a budget.

## The brief's prior art is a family of option-pickers, and that decides the architecture

The brief treats Laya, CLM-8B and s1decide as command-generating decision models and reads the OpenAI Decisions API as validation of the design. The research does not support that reading, so this section corrects it before the plan builds on anything.

| Item | What it is | Status | Consequence for Chungus |
|---|---|---|---|
| Jev 1.13 (TypeSafe) | API-only typed decision model. Primitives: `choice`, `score`, `noul`. P50 0.19–0.21 s, P95 0.34 s. Parameter count and architecture undisclosed. | Real, hosted only | Usable as a hosted reference baseline. Cannot be self-hosted. |
| Laya (convaiinnovations/laya) | **ModernBERT-large encoder** (421M) plus a decision head. Scores options supplied at `[MASK]` positions. **512-token context** (1,024 for typed-decisions). Apache 2.0. Ships a Jev-compatible `POST /v1/systemone`. | Real, open weights | A picker over candidates. It cannot generate commands, and it cannot use prefix caching. |
| CLM-8B (Stanford/NVIDIA) | Contrastive state/action scorer on a frozen Qwen3-8B encoder, served via vLLM on GPU | Real, open weights | Out of scope for a CPU VPS. The cached-action-embedding idea is worth borrowing. |
| s1decide (claimed Qwen3.8-27B LoRA) | Nothing found. The base model exists. | **Unverified** | Drop it from the plan. ⚑ Owner: confirm the source. |
| OpenAI Decisions API | GPT-6 Luna constrained to developer-defined options. Limited preview. **No public endpoint, schema, pricing or context limit.** | Real, undocumented | Confirms the pattern. There is no contract to copy. |

Sources: [OpenRouter jev-1.13](https://openrouter.ai/typesafe/jev-1.13), [Firecrawl](https://www.firecrawl.dev/blog/openai-decisions-api-vs-jev), [Laya model card](https://huggingface.co/convaiinnovations/laya), [CLM-v0.1-8B](https://huggingface.co/sekkit/CLM-v0.1-8B).

The industry meaning of "System One" is **selecting among options the caller already enumerated**. For a MUD, that puts the hard engineering in a candidate builder that turns the current screen into legal commands. Text-game research reached the same conclusion years ago. In Jericho, agents given the admissible-action handicap (DRRN 13.0%, KG-A2C 10.8%) beat CALM's learned GPT-2 generator (9.4%), whose top-30 candidates covered only about 40% of admissible actions ([CALM](https://arxiv.org/pdf/2010.02903)). Unconstrained small models do badly. In BALROG, Llama 3.2 1B scored 6.32% against GPT-4o's 32.34%, and the harness needs a fallback action for invalid outputs ([BALROG](https://arxiv.org/html/2411.13543v1)).

Laya's own numbers argue against it as the main path on this hardware. Accuracy **collapses on high-cardinality choices: Banking77 scored 0.425 against Jev's 0.870**, and the card warns that 50+ options degrade sharply. The `act_probability` escalation signal is unusable (AUROC 0.30). Raw calibration ships over-confident ([Laya card](https://huggingface.co/convaiinnovations/laya)). The bigger problem is serving cost. A bidirectional encoder must re-encode the whole input every tick. A 512-token pass costs about 4×10¹¹ FLOPs, which is **an estimated 1.6 s per decision on 4 CPU cores**. A decoder with a warm prefix cache needs an estimated 0.35–0.7 s. Both figures are estimates scaled from published llama.cpp benchmarks, not measurements. They conflict with Laya's reported ~9–33 ms, which were measured on a T4 GPU or under batching.

**Decision.** The default is a decoder LM emitting `VERB [ARG]` under a per-tick grammar. Laya enters Phase 3 as a bake-off contender for the combat subset, where candidate lists are short and its calibration gives a clean escalation threshold. Hosted Jev serves as a reference ceiling, reached through the same `Decider` interface (Laya's endpoint is Jev-compatible, so one client covers both).

## Decisions by research question

### Action representation: per-tick grammar over player-visible entities

The per-tick grammar contains only the verbs this character can use right now, filtered by class, position, combat state and the plan card's `allowed_verbs`. It never contains all ~500 commands. A warrior's tactical vocabulary is probably a few dozen verbs: movement, `kill`, `flee`, `bash`, `kick`, `rescue`, `get`, `wear`, `wield`, `eat`, `drink`, `rest`, `stand`, `look`, `score`, `consider`. ⚑ Verify that list against the Dark Pawns command table in Phase 0. Arguments come from entities parsed out of text the player has already seen: exits, mob keywords with CircleMUD ordinals (`2.goblin`), objects, containers. This is the MUD equivalent of Jericho's admissible-action enumeration, and it uses no engine state.

Grammar engine: **llguidance** inside llama.cpp. It computes a token mask in about 50 µs of single-core CPU time for a 128k-token vocabulary, has essentially no startup cost so grammars can change on every request, accepts Lark-style CFGs, and is integrated into llama.cpp behind a CMake option ([llguidance](https://github.com/guidance-ai/llguidance)). Native GBNF via the per-request `grammar` field is the fallback. If GBNF is used, write `x{0,N}` rather than chains of `x? x?`, because some nested-alternative constructs cause exponential sampling slowdowns ([llama.cpp grammars](https://github.com/ggml-org/llama.cpp/blob/master/grammars/README.md)). The llguidance version is ambiguous: GitHub lists v1.0.0 while a PyPI mirror lists 1.7.0 ([mirror](https://simple-repository.app.cern.ch/project/llguidance)). Pin whatever crates.io shows at implementation time. ⚑ Confirm the exact CMake flag name and in-request grammar syntax against the pinned llama.cpp build.

The output vocabulary also includes **`wait`**, a no-op that the runner does not send. Without it, the model learns to always type something during combat rounds where the expert typed nothing.

**Free-text commands** (`say`, `tell`, `emote`, `gossip`): Chungus decides *whether* to speak and *to whom*, never *what*. The grammar exposes `say <MSG_REF>` and `tell <player> <MSG_REF>`, and the runner substitutes text the planner queued earlier. Typed decision models cannot produce prose by design, and Jev "architecturally cannot" ([Firecrawl](https://www.firecrawl.dev/blog/openai-decisions-api-vs-jev)). Keeping player-visible text away from the 0.5B model is also a containment win. If tactical speech proves necessary, add a closed template set ("group: need heal", "fleeing") as a choice. A regex-capped free slot such as `[^\n]{1,80}` is the last resort. Every path strips newlines, color codes and any command-stacking separator. ⚑ Phase 0: does the Dark Pawns parser support a multi-command separator or aliases that free text could abuse?

**Training alignment.** Normalize recorded human commands into the grammar's canonical form using the parser's own abbreviation rules (for example `k gob` becomes `kill goblin`). That keeps SFT targets inside the constrained space, so constrained and unconstrained output distributions match.

### State representation: stable-first snapshot of at most ~550 tokens, built only from the text stream

The observation is a deterministic serialization of a parsed text stream. The parser is a client-side text-to-fields transform of what a human sees, the same thing a Mudlet trigger does. **GMCP is not model input.** It is a second input path that the byte-fidelity oracle does not cover. It can leak fields absent from text (vnums, exact mob HP). And the human demonstrations are text, so a GMCP-fed model would train on a different distribution. GMCP stays available for operator dashboards. Room identity uses a client-side hash of name plus description, never a vnum, unless mortals see vnums. ⚑ Phase 0: check whether they do.

Two research threads pull in opposite directions here. The state research proposes a fixed header with vitals first. The inference research shows that prompt processing of uncached tokens is about 95% of per-decision CPU cost, so anything volatile placed early invalidates the prefix cache. The plan resolves this by **ordering blocks from most stable to least stable**:

```
[system prompt: role, ruleset, output format]                 static        (cached)
<G> plan#7 step 2/4 "clear wolves, Forest Path" | flee_if hp<35% | avoid: "the forest ogre"   per plan
<R> room#a91f "Forest Path" exits N S E(closed door) | dark=no                               per room
<O> mob: a snarling wolf (fighting you); a rabbit | obj: a rusty sword | pc: Ayla            per room/event
<I> wield longsword; potions:2 (red); food:1; weight 61%                                     per event
<A> [-1 kill wolf ->ok] [-2 bash ->fail lag4s] [-3 n ->moved] ... (K=6)                      per tick
<T> last ≤15 salient lines, ANSI stripped, channels dropped unless they name the agent      per tick
<F> tgt "a snarling wolf" cond=bleeding | tank=me | grp: Ayla 80%                           per tick
<S> hp 142/210 ma 30/30 mv 88/110 | pos FIGHT | lag~1.6s | tick~41s                         per tick
```

The estimated budget is 350–550 tokens per observation, with **≤150 new (uncached) tokens per tick** as the design target, because that number alone sets agent capacity on the CPU (see the VPS section). The windowing rules are:

- Keep raw lines only since the last room change or the last two combat rounds, whichever is shorter.
- Drop gossip, auction and other channels unless they name the agent.
- Collapse repeated identical lines into a count.
- Older state lives only in the cumulative blocks and the action ring.

The rationale comes from two findings. Focused prompts of about 300 tokens beat full prompts across 18 models, and a single distractor lowers accuracy ([Chroma Context Rot](https://www.trychroma.com/research/context-rot)). Looping is the dominant MUD-agent failure, at 14–66% of frontier runs in CrucibleBench ([summary](https://daily.dev/posts/cruciblebench-old-worlds-for-new-agents-0icaqpu7j)). The `<A>` ring with outcomes is the cheapest defense against loops.

The lag estimate and tick timer are computed client-side from timestamps and the known WAIT_STATE table, which is exactly what human tick-timer scripts do. ⚑ Open: snapshot format versus an append-only multi-turn transcript. Append-only maximizes cache hits but grows the context and needs periodic resets. Phase 1 benchmarks both.

### Planner/decision split: typed plan cards, event-driven replanning, deterministic aborts

CircleMUD-lineage timing sets the budgets. One pulse is 0.1 s (`OPT_USEC 100000`). `PULSE_VIOLENCE` is 2 s on a global clock. The regen/affect tick is 75 s (`SECS_PER_MUD_HOUR`). Each descriptor gets at most one queued command per pulse, and only when its wait is zero ([tbaMUD structs.h](https://raw.githubusercontent.com/tbamud/tbamud/master/src/structs.h), [comm.c](https://raw.githubusercontent.com/tbamud/tbamud/master/src/comm.c), [utils.h](https://raw.githubusercontent.com/tbamud/tbamud/master/src/utils.h)). Combat skills impose 2.2–6 s of lag: `hit` is `PULSE_VIOLENCE+2`, `bash` and `backstab` are 2×, `kick` is 3× ([act.offensive.c](https://raw.githubusercontent.com/tbamud/tbamud/master/src/act.offensive.c)). Merc 2.2 differs: 3 s rounds, 30 s ticks, and `WAIT_STATE` takes the max where tbaMUD assigns ([merc.h](https://raw.githubusercontent.com/alexmchale/merc-mud/master/src/merc.h)). ⚑ **None of these numbers has been checked against the Go port. Phase 0 task.**

| Layer | Runs | Budget (Circle timing) | Blocking |
|---|---|---|---|
| Runner abort rules | every prompt line | < 10 ms | yes, pre-empts the model |
| Chungus decision | once per combat round; out of combat, per output batch or after a 0.5–1 s idle | inference p50 ≤ 300 ms, p95 ≤ 700 ms; end-to-end p95 ≤ 1.0 s; hard ceiling 1.8 s (2.7 s on Merc timing) | yes |
| Planner (frontier LLM) | event-driven, refresh ceiling 75–150 s | 2–30 s acceptable | never |

The planner sends a **plan card** containing `plan_id`, `issued_at` and `expires_after_s`; a `goal`; ordered `subgoals`, each with a `done_when` predicate and `max_rounds`; `constraints` such as `avoid_mobs`, `no_pk` and `max_gold_spend`; an `abort` block such as `flee_if_hp_pct_lt`, `flee_lookahead`, `recall_if_hp_pct_lt` and `abort_if_mob_level_gt`; `allowed_verbs`; and a queue of planner-written utterances keyed by `MSG_REF`. The runner keeps the card as JSON and serializes it into `<G>`.

The structure follows the asynchronous System 2 in DPT-Agent, which hands down behavior guidelines while an FSM acts in real time ([DPT-Agent](https://arxiv.org/html/2502.11882v1)), and the interrupt list in NetPlay ([NetPlay](https://arxiv.org/html/2403.00690v1)). NetPlay's gap is the strongest argument for the split: a zero-shot GPT-4 agent scored 284.85 against 11,341.94 for the handcrafted AutoAscend behavior tree. Chungus plays the role of that behavior tree, learned instead of handwritten.

**Replan triggers** (the runner emits these as `event` frames to the planner):

1. Death or respawn.
2. Level gain or new skill.
3. A subgoal is done, or the last subgoal is done.
4. `max_rounds` or `expires_after_s` exceeded.
5. An abort fired.
6. Unexpected room: off-path for more than 2 moves, or a forced move.
7. Loop: the same command 4 or more times with no state change, or no progress for N rounds.
8. Equipment or consumable break.
9. A tell or say naming the agent.
10. Decision confidence below threshold for M ticks (only available with a calibrated head).

When a card expires, the runner keeps executing a safe default (`rest` or hold) until a new card arrives.

**Abort rules are runner code, not model behavior.** The commands typed during lag queue up first-in first-out. A `flee` sent right after `bash` therefore waits up to 4 s. The runner keeps **at most one outstanding command in combat** and flees on a lookahead test: `hp − dmg_per_round × ceil((remaining_wait + transit)/round) < floor`. The hard floor must not depend on a 0.5B model.

### Training recipe: one human run as the gold seed, frontier distillation for volume, then DAgger and RL

**Capture.** Record the owner's 1→30 playthrough twice:

- Server-side, in the Postgres sidecar, as the source of truth: every input line with its receive time and every output flush with its seq.
- In Mudlet, with the logger package set to `format="txt"`, `timestamp=true` and `logAllSends=true` ([Mudlet logger](https://wiki.mudlet.org/w/Mudet_logger_package)).

Label with the **server-received** command, because aliases and triggers make typed text differ from executed text. Set a machine-parseable prompt for the recording. Tag trigger-fired commands, or ask the owner to minimize automation during the run.

**Segmentation.** Each input line closes one decision. Its observation is the serializer's output over everything up to that point. A combat round with no input becomes a `wait` sample. The outcome window (output until the next input) produces labels for filtering and reward, such as ΔXP, ΔHP, kills, deaths and "Huh?!", but is never model input.

**Volume.** These figures are planning estimates, not empirical laws.

- The human run yields about **20k–60k decisions**, used as gold data, as the held-out eval set, and upweighted to 20–30% of each SFT batch.
- Frontier distillation targets **200k–1M decisions**, starting at 50k and scaling on the slope of the eval curve.

No number of human hours makes pure behavioral cloning robust. On NetHack, BC on 3.48B AutoAscend transitions scored 554 against the demonstrator's ~10,105, and APPO+BC reached 1,282 ([Dungeons and Data](https://arxiv.org/pdf/2211.00539)). A pretrained LM is far more sample-efficient: FireAct got a 77% HotpotQA gain from 500 GPT-4 trajectories ([FireAct](https://arxiv.org/abs/2310.05915v1)). And 60 SFT demonstrations plus 400 RL episodes matched 5,000 pure-RL episodes on text environments ([practitioner's guide](https://www.themoonlight.io/en/review/a-practitioners-guide-to-multi-turn-agentic-reinforcement-learning)). Distilled trajectories are filtered by outcome (positive XP rate, no death, no loop), which amounts to rejection-sampling fine-tuning. BALROG's "knowing-doing gap" ([BALROG](https://arxiv.org/html/2411.13543v1)) means frontier play must be checked against the human curve before anyone trusts it.

**Base models.**

| Candidate | Notes |
|---|---|
| Qwen2.5-0.5B-Instruct | 24 layers, 2 KV heads, vocabulary 151,936, tied embeddings ([config](https://huggingface.co/Qwen/Qwen2.5-0.5B-Instruct/blob/main/config.json)) |
| Qwen3-0.6B | Pure transformer with mature GGUF support |
| Qwen3.5-0.8B | Unsloth-supported; uses hybrid Mamba-style layers ([Unsloth](https://unsloth.ai/docs/models/qwen3.5/fine-tune)). ⚑ llama.cpp CPU support for this architecture is unverified, and it is 60% over the 0.5B target. |
| SmolLM2-360M, Gemma 3 270M | Optional; not verified in this research |

**SFT** uses TRL v1.x (v1.0.0 released March 2026 with async GRPO; [TRL highlights](https://releases.sh/hugging-face/trl/highlights)). Settings:

- Full fine-tune in bf16, LR 1e-5 to 3e-5, cosine schedule, 3% warmup, effective batch 32–64, 2–3 epochs on human data.
- Loss on completion tokens only.
- If using LoRA instead: all linear layers, about 10× the full-FT LR ([LoRA Without Regret](https://thinkingmachines.ai/blog/lora/)).

These are planning defaults, not a sourced recipe. Expected cost is under $200 for SFT including sweeps, at $1.49–$6.98/hr for an H100 ([IntuitionLabs](https://intuitionlabs.ai/articles/h100-rental-prices-cloud-comparison)). This is an estimate; measure throughput on the first run.

**Behavioral-cloning failure modes and the mitigation for each:**

| Failure mode | Mitigation |
|---|---|
| Copycat ("repeat the last command" is a strong predictor in MUD combat) | Action-history dropout plus keyframe upweighting ([copycat](https://proceedings.neurips.cc/paper/2020/hash/1b113258af3968aaf3969ca67e744ff8-Abstract.html), [keyframes](https://arxiv.org/pdf/2106.06452)) |
| Causal confusion (fleeing on a mob message rather than on HP) | Fixed structured vitals block, plus counterfactual scenario tests |
| Distribution shift into states the expert never visited | DAgger: the frontier model relabels states the student actually visits |
| Loops | Runtime detector plus recovery examples in training |

**RL.** A self-hosted Go MUD makes a good environment. It requires a headless build with a configurable pulse length (tick acceleration), loopback transport, character snapshot/reset, and many instances per box. The 5,300-case oracle must pass on the accelerated build so that acceleration changes only timing.

- Algorithm: TRL async GRPO, or verl-agent GiGPO. GiGPO's step-level grouping of repeated states fits recurring rooms and mobs; Qwen2.5-1.5B reached 86.7% on ALFWorld ([verl-agent](https://github.com/langfengQ/verl-agent)). The guide reports PPO beats RLOO for small models, if a critic is affordable.
- Adapter: LoRA rank 1–16 is enough for policy-gradient RL ([LoRA Without Regret](https://thinkingmachines.ai/blog/lora/)).
- Reward: ΔXP normalized by XP-to-next-level, plus kill, minus a large death penalty, minus invalid commands, minus a loop penalty, minus a small per-step cost, plus a level-up bonus. Gold is never rewarded.
- Estimated cost: 5–30 GPU-hours per RL run.
- Evaluation must also run at real speed, because acceleration removes real-time pressure.

### VPS inference: llama-server over localhost, CPU-pinned, capacity set by new tokens per tick

The serving stack is `llama-server` from llama.cpp. As of early October 2026 the build is at or above b10780, inferred from yzma's pairing ([yzma](https://github.com/hybridgroup/yzma)). It provides continuous batching, prompt caching, `--cache-reuse` KV shifting, slots and per-request grammar, all on CPU ([server README](https://github.com/ggml-org/llama.cpp/blob/master/tools/server/README.md)).

Integrate over **localhost HTTP, not in-process**. This keeps C++ crashes and memory out of the 95K-LOC game binary and gives batching and slots for free. yzma (purego, no cgo) is the fallback. Pin each agent to a fixed `id_slot`. Quantize to Q8_0: prompt processing is compute-bound, so its speed is similar to Q4_K_M, and 10-token outputs make Q8_0's slower generation affordable.

The following are estimates for a 4 vCPU / 8 GB box (⚑ actual VPS specs unconfirmed), scaled from a Qwen2.5-3B benchmark on an i7-6700 of pp512 43 t/s and tg128 11.8 t/s ([computingforgeeks](https://computingforgeeks.com/run-local-llm-llama-cpp/)):

| Case (0.5B, 4 threads, pp ≈ 300 t/s, tg ≈ 60 t/s) | Estimated decision latency |
|---|---|
| Cold 1,000-token prompt | ≈ 3.5 s (**misses the round**) |
| Cold 500-token prompt | ≈ 1.8 s (borderline) |
| Warm cache, 150 new tokens | ≈ 0.7 s |
| Warm cache, 50 new tokens | ≈ 0.35 s |
| Laya encoder, 512 tokens, no caching possible | ≈ 1.6 s |

RAM is not a concern: about 0.7–1.0 GB covers the model plus KV for 8 slots, at about 12 KiB per token (f16 KV). CPU is the binding constraint. With 3 inference threads, estimated capacity is about **4 agents per 2 s round at 100 new tokens per tick, 6–8 at 50, and under 1 with cold 1,000-token prompts**. In order, the bottlenecks are uncached prompt tokens, vCPU steal on shared hosts, and thread oversubscription against the Go server.

The unit file below uses flags from the README; ⚑ verify the exact spellings against `--help` on the installed build.

```ini
[Service]
ExecStart=/opt/llama.cpp/bin/llama-server -m /opt/models/chungus-q8_0.gguf \
  --host 127.0.0.1 --port 8081 -t 3 -tb 3 -np 6 -c 12288 \
  --cache-reuse 64 -cram 512 --slots --no-webui
User=llama
AllowedCPUs=1-3
CPUQuota=300%
CPUWeight=50
Nice=10
MemoryMax=2G
Restart=on-failure
```

Set `-cram 512` explicitly; the default 8,192 MiB cache cap is the whole box. Keep `-t` at or below the number of allowed CPUs, because spinning ggml threads stall badly when oversubscribed. The runner times out a decision at about 1.5 s. On timeout it falls back to the rule layer: repeat the current attack, or flee below the abort threshold.

### Remote wire: NDJSON over WebSocket as the canonical stream, MCP as a thin long-poll facade

There are three clients and one stream:

- **`/agent`** (WSS, NDJSON, one JSON object per text message). This is the canonical agent stream. `chungus-runner` uses it over loopback, and custom remote harnesses use it directly.
- **The `dp` CLI** (Go, same repo) bridges the same frames to stdin/stdout: `dp tail`, `dp send`, `dp play`. Any framework that can run a subprocess, including Claude Code via Bash, can drive the game with it.
- **`/mcp`** is a thin facade over the same session for MCP clients.

Precedents: Evennia's JSON-over-WebSocket `["cmdname",[args],{kwargs}]` ([Evennia OOB](https://www.evennia.com/docs/latest/Concepts/OOB.html)), and MCP stdio's own newline-delimited framing ([MCP 2025-11-25 transports](https://modelcontextprotocol.io/specification/2025-11-25/basic/transports)).

Frames:

- **Server to client.** `out{seq, ts, text, raw_b64?}` carries game output exactly as the descriptor would have written it, `**OVERFLOW**` included. `gap{from, to, dropped_lines}`. `ack{id, seq_at_exec}`. `err{code, retry_after_ms}`. `event{kind, plan_id, ...}` is emitted by the runner session only. `bye{reason}`.
- **Client to server.** `cmd{id, text}`. `resume{session, since_seq}`. `ping`. `plan{card}` goes to runner sessions only.

There is no synthetic fallback on an invalid command. The parser's own "Huh?!" is the feedback, per the iron rule.

**Sessions.** The client authenticates with an `Authorization: Bearer` per-account API key: 256-bit, stored hashed, with `revoked_at`, and mapped 1:1 to an agent-flagged character. The server returns `{session: uuidv7, seq}`. On reconnect within the grace period, the client sends `resume`. If `since_seq` is still in the ring, the server replays; otherwise it sends `gap` and then live output. After the grace period, the client reattaches through CircleMUD's normal link-dead reconnect with no replay.

Link-dead behavior is unchanged from stock CircleMUD. The character stays in the world and keeps fighting. It is moved to the void after `idle_void = 8` ticks and rented after `idle_rent_time = 48` ([config.c](https://raw.githubusercontent.com/Yuffster/CircleMUD/master/src/config.c), [comm.c](https://raw.githubusercontent.com/Yuffster/CircleMUD/master/src/comm.c), [limits.c](https://raw.githubusercontent.com/Yuffster/CircleMUD/master/src/limits.c)). Add an agent-only session TTL of 4 h.

**Backpressure.** The engine's per-pass buffers and `**OVERFLOW**` behavior stay byte-identical for every descriptor ([structs.h](https://raw.githubusercontent.com/Yuffster/CircleMUD/master/src/structs.h)). The agent ring sits *downstream* of `process_output`:

- Size: 2,000 lines or 256 KB per session, to be tuned against measured combat output rates.
- Overflow: drop oldest, never newest, and emit `gap`.
- Slow consumer (more than 64 frames or 5 s behind): stop pushing and switch the client to pull via `resume`. Close the socket after the grace period, which leaves the character link-dead.
- Input: queue at most 4–8 pending commands. Above that, return `err rate_limited`.

**MCP facade** (amended; see the section on TRANSPORT-MCP-RESEARCH.md below). The tool surface is `create_account`, `send_command(session, text)`, `read_output(session, since_seq, wait_ms ≤ 10000)` (a long-poll that blocks until new output or timeout), and for runner sessions `set_plan(session, card)` and `read_events(session, since_seq, wait_ms)`. There are no per-command tools. Library: `mark3labs/mcp-go` v1.1.x (v1.0.0 added 2026-07-28 support while staying compatible with 2025-11-25 clients) or `modelcontextprotocol/go-sdk` v1.8.0 ([mcp-go releases](https://github.com/mark3labs/mcp-go/releases), [go-sdk releases](https://github.com/modelcontextprotocol/go-sdk/releases)). Serve both spec versions, because clients lag.

For Claude Code, an optional local stdio shim `dp mcp --channel` turns combat, tell, death and low-HP events into `notifications/claude/channel`. This is the only documented way to push events into Claude Code's context. It works only over local stdio, is a research preview, and needs `--dangerously-load-development-channels` for custom channels ([channels reference](https://code.claude.com/docs/en/channels-reference.md)).

### Evaluation and containment: extend the oracle to pass-rate scenarios, and contain at the transport layer, not the parser

The 5,300-case oracle's methodology carries over to agents, with assertions on outcomes in place of byte-exact transcripts, because a stochastic policy has no single golden output. A scenario is (character snapshot, zone state, seed, optional plan card), run for N steps, then asserted on, and scored as a **pass rate over N seeds against a threshold**. Examples:

- HP below 20% with an aggressive mob: flees or quaffs within 2 rounds.
- Hungry: eats within K steps.
- Invalid target: does not repeat the failed command more than twice.
- Dead end: leaves within K steps.
- Better weapon in inventory: wields it.

Golden transcripts remain useful as regression fixtures for greedy-decoded policies.

Around the scenarios sit three more layers:

- **Offline metrics:** exact and equivalent-action match on held-out human decisions, reported separately on keyframes to catch copycat inflation.
- **Run metrics:** XP/hour by level band against the human's level-versus-hours curve (the analogue of BALROG's progression metric), deaths/hour, invalid-command rate, loop rate, wait ratio, flee success, planner-escalation rate.
- **LLM-as-judge** on sampled transcript windows, for trend tracking only. Its reliability is unmeasured.

Train on a subset of zones and evaluate on held-out zones of the same level band.

**Containment** adds layers around unchanged game rules.

- **Identity.** A `PLR_AGENT` flag, set only through API-key account creation. Agent characters cannot use the human login path, and human characters cannot use the agent path.
- **Rollout stages.** S0 is a separate instance with its own world copy and player DB (for example `:7778`). S1 is the live world with a newbie-zone allowlist, enforced by auto-freeze on zone exit, not by refusing movement. S2 is the full world.
- **Rate limit.** A transport token bucket of 2 commands/s sustained, burst 6, under CircleMUD's ceiling of 10/s. Retune it from the human log's p99 burst.
- **Line length.** Cap at the engine's `MAX_INPUT_LENGTH`. ⚑ CircleMUD 3.x uses 256 and tbaMUD uses 512; Phase 0 confirms which applies.
- **Channels.** Global channels muted by default. `say` and `tell` limited to 1 per 10 s.
- **Kill switches.**
  - `agents freeze|thaw|purge`, which uses the existing FROZEN flag and pauses input queues, in global and per-agent forms.
  - `AGENTS_ENABLED=false`, which rejects `/agent` and `/mcp` connections.
  - Key revocation.
  - Auto-freeze tripwires on quotas: commands/hour, deaths/hour, PK attempts, tells to humans.
- **Audit.** Every agent command is logged with session, `seq_at_exec`, harness and model ID, and the output seq range it caused.
- **Prompt injection.** Treat all inbound player text as untrusted in the planner harness.

Unsanctioned automation is bannable on most MUDs. Aardwolf bans "any automation that plays the game while you're absent" ([Aardwolf](https://www.aardwolf.com/trigbot.html)). Agents must therefore be opt-in, disclosed (`help agents`, MOTD) and identifiable. ⚑ Identifiability conflicts with byte-for-byte fidelity; see Open questions.

## Phased implementation plan

The dependency chain is 0 → 1 → 2 → 3 → 4 → 5 → 6. Phases 1 and 2 can overlap once the Phase 0 parser facts are in. The human recording (2a) can start as soon as the sidecar capture from 1c exists.

### Phase 0: Verify constants and benchmark the box (gate for every budget in this plan)

**Entry:** none.

**Tasks**

- Grep the Go source and record each answer in `docs/plans/agent-constants.md`, next to the tbaMUD, CircleMUD and Merc values cited above:
  - `PULSE_VIOLENCE`, pulse length, `SECS_PER_MUD_HOUR`, `PULSE_MOBILE`, `PULSE_ZONE`.
  - Per-skill `WAIT_STATE` values (warrior skills especially).
  - `WAIT_STATE` semantics: assign or max.
  - One-command-per-pulse dequeue.
  - `MAX_INPUT_LENGTH`.
  - `idle_void` and `idle_rent_time`.
  - Link-dead behavior in `close_socket`.
  - Whether `**OVERFLOW**` is reproduced.
- Parser facts:
  - Command-stacking separator.
  - Abbreviation rules.
  - `N.keyword` / `all.keyword` support.
  - Whether vnums or numeric mob HP are visible to mortals.
  - Whether GMCP or MSDP is implemented server-side.
  - Existing mute or notell flags.
  - The warrior's command list.
  - Whether the pulse can be decoupled from wall-clock time (needed for RL acceleration).
- VPS:
  - Record vCPU count, dedicated or shared status, CPU model and flags (AVX2/AVX-512), and baseline `%steal`.
  - Build llama.cpp (pinned tag) with llguidance enabled.
  - Run `llama-bench -m <model>.gguf -p 128,512,1024 -n 16 -t 2,3,4` on Qwen2.5-0.5B-Instruct and Qwen3-0.6B (Q8_0 and Q4_K_M) as stock GGUFs, and attempt Qwen3.5-0.8B.
  - Benchmark Laya via `laya[onnx]` (0.3.20) at 256 and 512 tokens.
  - Measure grammar-constrained decode of 10 tokens with a 50-alternative regenerated grammar.
  - Load-test N = 1..8 concurrent slots with the game server running and pinned to CPU 0.

**Exit criteria**

- The constants file is merged. Every budget in this document is re-derived from it, and a delta note is filed if Dark Pawns follows Merc timing.
- Measured warm-cache p95 is ≤ 700 ms at the target concurrency (start with 3 agents) with ≤150 new tokens per tick. **If this fails:** shrink the observation, move to a smaller model, or upgrade the VPS. That decision is recorded before Phase 3.
- The go/no-go for Qwen3.5-0.8B on llama.cpp CPU is recorded.

### Phase 1: Agent session layer, wire and containment skeleton

**Entry:** Phase 0 parser facts (stacking separator, `MAX_INPUT_LENGTH`, idle constants).

**Tasks**

- **1a. Game-side session layer.**
  - UUIDv7 game sessions (`google/uuid` v1.6.0, per the July spec) decoupled from the socket.
  - A per-session output ring downstream of `process_output`, with a monotonic `seq` (reuse `ServerMessage.Seq`).
  - `resume` handling with grace-period replay.
- **1b. Agent transport.**
  - `/agent` NDJSON over `coder/websocket`.
  - The `dp` CLI.
  - Bearer API keys (hashed, revocable).
  - The `PLR_AGENT` flag and login-path separation.
  - Token-bucket rate limit, pending-queue cap, line-length cap.
  - Slow-consumer policy.
  - ⚑ Decide whether `/agent` evolves the existing custom `/ws` JSON protocol or replaces it.
- **1c. Postgres sidecar schema.**
  - `sessions(session_uuid, account, character, harness_id, model_id, started_at, ended_at)`.
  - `frames(session_uuid, seq, dir in|out, ts, text)`.
  - `commands(session_uuid, cmd_id, text, recv_ts, seq_at_exec, out_seq_from, out_seq_to)`.
  - An opt-in capture flag for the owner's human account. This stream is both the audit log and the training source.
- **1d. Kill switches.** `agents freeze|thaw|purge` (global and per-agent), `AGENTS_ENABLED`, mudlog at `LVL_IMMORT`.
- **1e. S0 sandbox.** A second instance of the same binary on its own world copy and player DB.
- **1f. MCP facade.** `send_command`, `read_output` long-poll, `create_account`, on the amended surface.

**Exit criteria**

- The 5,300-case oracle passes unchanged; agent descriptors produce byte-identical output to human descriptors for the same input.
- A kill-switch drill (freeze, purge, key revoke) works in under 5 s on S0.
- A forced disconnect mid-combat followed by `resume` within grace replays with no lost seq. After grace, the client gets `gap` or a link-dead reattach.
- Rate-limit and queue caps return `err rate_limited` with `retry_after_ms`.
- A frontier model plays on S0 through both `dp` and MCP `read_output` long-poll.

### Phase 2: Observation serializer, grammar builder, and the data pipeline

**Entry:** Phase 1 sidecar capture live on S0. Phase 0 parser facts recorded.

**Tasks**

- **2a. Human recording.** The owner records the 1→30 run with server-side capture plus Mudlet (`txt`, `timestamp=true`, `logAllSends=true`), a machine-parseable prompt, and automation minimized or documented.
- **2b. `pkg/agentobs` (Go), a single library shared by the runner and the dataset exporter.** This is what guarantees train/serve parity. It contains:
  - The text-to-fields parser, built against the engine's own `act()`/`send_to_char` format strings and tested from oracle transcripts.
  - The stable-first serializer.
  - The windowing rules.
  - The per-tick grammar builder (Lark for llguidance, GBNF fallback).
  - The abbreviation normalizer.
  - Lag and tick estimators.
- **2c. `dp export`.** Turns sidecar frames into JSONL samples `{obs, grammar, action, outcome}`, with `wait` samples, trigger-fired tags, and outcome labels.
- **2d. Distillation runs on S0.** A frontier model plays in direct mode through `/agent`, given the planner prompt plus full scrollback, across level bands and zones, logged via the sidecar. Filter by outcome.
- **2e. Datasets.** Build the held-out human eval set and the zone split.

**Exit criteria**

- Every human command in the recording either normalizes into the grammar generated for its own observation or is classified as out-of-grammar (planner-level, social, typo). The out-of-grammar rate is measured and reviewed.
- Observation token count is p95 ≤ 550, with new tokens per tick at p95 ≤ 150, or the Phase 0 measured equivalent.
- At least 50k distilled decisions plus the full human set are exported.
- The frontier agent's XP/hour on levels 1–10 is compared against the human curve and the gap is documented, as a check on the knowing-doing gap.

### Phase 3: System One v0 and the decision-backend bake-off

**Entry:** Phase 2 exit. Phase 0 benchmark go.

**Tasks**

- **3a. `chungus-runner` (Go, separate systemd unit on the VPS).** It connects to `/agent` over loopback with its own agent key, runs `pkg/agentobs` and the abort rules, and calls a `Decider` interface. Backends:
  1. `llama-server` with the per-request grammar.
  2. `laya-serve` `/v1/systemone`, using a two-stage choice (verb family, then a fully formed target command), each list capped at 40 options.
  3. Hosted Jev through the same request shape, as a reference only. Note that data leaves the box.
- **3b. SFT.** Train decoder candidates (Qwen2.5-0.5B, Qwen3-0.6B, Qwen3.5-0.8B if it passed Phase 0) in TRL, with the human data upweighted, action-history dropout and keyframe weighting. Fine-tune Laya from `laya-typed-decisions` on the same samples, recast as choice questions.
- **3c. Bake-off.** Run on S0 in real time with identical observations. Measures:
  - Equivalent-action accuracy on the human held-out set and on keyframes.
  - Scenario-suite pass rate (first 30 scenarios).
  - p95 end-to-end latency at 3 and 6 concurrent agents on the VPS.
  - Invalid-command rate.
  - Loop rate.
  - XP/hour and deaths/hour on levels 1–10.

**Exit criteria**

- One backend is chosen by a written rule. The decoder wins unless Laya beats it by at least 5 points on scenario pass rate at equal or better p95 latency. A hybrid is allowed: Laya for combat choice, decoder elsewhere.
- The chosen model reaches p95 end-to-end ≤ 1.0 s at 3 concurrent agents.
- It beats a scripted rule-bot baseline on scenario pass rate.
- Invalid-command rate is under 2%. The decoder should be near 0 by construction; the remainder is staleness ("They aren't here").
- Deaths/hour on levels 1–10 is within 2× the human's.

### Phase 4: Planner integration and real-time evaluation

**Entry:** Phase 3 backend chosen.

**Tasks**

- **Plan-card schema.** Versioned JSON Schema, kept in the repo.
- **Runner frames.** `plan`/`event` frames and the MCP `set_plan`/`read_events`.
- **Replan triggers and expiry.** The ten triggers above, plus the safe-default behavior on expiry.
- **`MSG_REF` utterance queue.**
- **Remote planner harness.** A frontier model reading `read_events` and summarized `read_output`. All player text is fenced as untrusted.
- **Scenario suite to 100+ cases.** Each case is (snapshot, zone state, seed, card), asserted on and scored as a pass rate over 10 seeds, wired into CI alongside the oracle.
- **LLM-as-judge rubric.** Calibrated against about 50 owner-labeled windows.

**Exit criteria**

- On S0, at real speed, planner plus Chungus takes a fresh warrior from level 1 to level 10 with no human intervention.
- XP/hour is at least 70% of the human curve, and deaths/hour is within 2× the human's.
- Loop rate is under 3% of steps.
- Planner calls average at least 60 s apart, which shows the split is doing its job.
- No abort-rule miss is attributable to the model.

### Phase 5: Interactive improvement (DAgger, then RL)

**Entry:** Phase 4 exit. Phase 0 confirmed the pulse can be decoupled from wall-clock time. If it cannot, run DAgger only.

**Tasks**

- **DAgger.** The frontier model relabels states the student visits, especially low HP, loops and post-death states. Retrain.
- **Headless accelerated build.** A build flag for pulse length, loopback transport, and character snapshot/reset. The oracle must pass on it.
- **RL.** TRL async GRPO or verl-agent GiGPO with LoRA r = 1–16, on the shaped reward above with each component logged separately, 8 rollouts per start state, and episodes of 200–500 decisions.
- **Reward-hacking monitors.** Trivial-mob farming, fight avoidance.

**Exit criteria**

- Each stage improves scenario pass rate.
- No regression in invalid-command rate, loop rate or deaths/hour.
- Real-speed evaluation confirms gains measured on the accelerated build.

### Phase 6: Live-world playtest

**Entry:** Phase 5 exit, or Phase 4 exit if RL is deferred. Disclosure decisions from Open questions are resolved.

**Tasks**

- **S1.** Live world, newbie-zone allowlist with auto-freeze on exit. At most 5 concurrent agents. No PK and no looting human corpses, enforced by tripwire. Global channels muted. Disclosure published (`help agents`, MOTD, website).
- **Watch.** Daily review of the audit log and tripwire events.
- **S2.** Full world, after 2 weeks of S1 with no player complaints and no tripwire false negatives.

**Exit criteria (S1 → S2)**

- Zero unhandled incidents.
- Kill-switch drill passes on live.
- Measured agent command rates sit inside the human p99 band.
- Owner signs off.

## Amendments to TRANSPORT-MCP-RESEARCH.md

The July spec's architecture still holds: external adapter first, UUID game sessions, push over poll in spirit. Six of its specifics are superseded.

1. **Sessions.** "UUID session ↔ `Mcp-Session-Id`" is removed. MCP 2026-07-28 drops protocol-level sessions and the initialize handshake. Cross-call state now uses "explicit, server-minted handles passed as ordinary tool arguments" (SEP-2567) ([changelog](https://modelcontextprotocol.io/specification/2026-07-28/changelog)). The game session UUID becomes a tool argument.
2. **Resumability.** "`Seq` → `Last-Event-ID`" is removed. 2026-07-28 drops SSE event IDs and redelivery, and a broken stream loses the in-flight request ([changelog](https://modelcontextprotocol.io/specification/2026-07-28/changelog)). Replay is application-level: `read_output(session, since_seq, wait_ms)` plus `resume` on `/agent`.
3. **Push.** "Combat push via `resources/subscribe` + `notifications/resources/updated` on the GET SSE stream" is replaced. The GET endpoint and resources/subscribe were replaced by `subscriptions/listen`, and Claude Code silently drops `notifications/resources/updated` (issue #47823, closed "not planned") ([mirror](https://claudeissues.com/issue/47823-mcp-notifications-resources-updated-not-handled-per-spec)). The baseline is the `read_output` long-poll. `subscriptions/listen` is optional for clients that support it, and the Claude Code channel shim is optional. The premise that "combat rounds must flow through SSE notifications, not tool results" is withdrawn.
4. **Tool surface.** Per-command tools (`look`, `north`, `attack`, …) and the dynamic per-session tool surface are replaced by `send_command` plus `read_output` (plus `create_account`, `set_plan`, `read_events`). This enforces the iron rule that the parser is the interface, and it removes drift between the tool list and roughly 500 commands.
5. **Resources and GMCP.** `mud://player/vitals` and similar resources derived from engine state are a structured gameplay channel. Restrict them to operator use, or derive them from the text stream. GMCP is not model input (see State representation).
6. **Library and open question 5.**
   - mcp-go v0.56.0 becomes v1.1.x, or go-sdk v1.8.0 (⚑ the go-sdk release dates were displayed as 2024; verify on GitHub).
   - ⚑ Check whether mcp-go v1.x's 2025-11-25 compatibility includes the GET stream and an event store (`server/streamable_http.go` at the pinned tag). This design no longer depends on either.
   - Open question 5 is corrected: stock CircleMUD has **no separate link-dead timer**. Link-dead characters idle to the void after 8 ticks (~10 min) and are rented after 48 ticks (~60 min) ([config.c](https://raw.githubusercontent.com/Yuffster/CircleMUD/master/src/config.c)).
   - The risk-register row "MCP resource subscription latency" is replaced by "long-poll wake latency", to be measured in Phase 1.

## Open questions

1. **Agent visibility versus fidelity.** Showing `[AI]` in `who` or in rooms changes output that the oracle pins. Options: (a) a new `whois`-style command excluded from the oracle, (b) a name-suffix convention plus `help agents`, (c) an oracle-whitelisted divergence in `who`. *Owner decision before Phase 6.*
2. **`noagent` opt-out for players.** Blocks agent tells, follows and groups. This is a game-behavior change. *Owner decision.*
3. **Timing lineage.** Circle (2 s rounds, 75 s ticks, assigning `WAIT_STATE`) or Merc (3 s, 30 s, max)? *Phase 0.*
4. **Observation format.** Snapshot versus append-only transcript. *Phase 0/1 benchmark.*
5. **Base model.** Qwen2.5-0.5B, Qwen3-0.6B or Qwen3.5-0.8B. Whether llama.cpp CPU supports Qwen3.5's hybrid layers. *Phase 0/3.*
6. **Decoder versus Laya picker.** *Phase 3 bake-off rule.*
7. **Existing `/ws` JSON protocol.** Evolve it into `/agent` or replace it? *Phase 1.*
8. **`MAX_INPUT_LENGTH`.** 256 or 512, plus the stacking separator and abbreviation rules. *Phase 0.*
9. **VPS hardware.** Actual vCPU count, dedicated or shared, AVX-512. All capacity numbers here assume 4 vCPU / 8 GB. *Phase 0.*
10. **Human rate baseline** for the token bucket. *Derive from the 2a recording.*
11. **Planner model and distillation budget.** Frontier API pricing as of October 2026 was not verified. 500k decisions at 2–3k input tokens each is about 1–1.5B input tokens, the dominant cost line.
12. **Hosted Jev access.** Waitlist, OpenRouter or Cloudflare Workers AI. Gameplay data leaves the box if it is used as a reference.
13. **s1decide provenance.** Not found. Confirm or drop.
14. **Non-Claude-Code MCP clients.** How Claude Desktop, claude.ai connectors, Cursor and Codex CLI surface notifications is unverified. The long-poll baseline makes this non-blocking.
15. **LLM-as-judge reliability** for game transcripts is unmeasured. Use it for trends only until it is calibrated.

## Conclusion

The design holds up because nothing on the critical path depends on something the research could not verify. The parser stays the only gameplay interface. The grammar comes from text a human sees, so the 0.5B model cannot emit a malformed command. Deterministic runner rules, not model reliability, guarantee the flee floor. The MCP layer is reduced to a long-poll facade, so it works whichever spec revision a client speaks. The real uncertainty sits in two places, and the phase gates are built around them. The first is CPU throughput: capacity scales almost entirely with uncached tokens per tick, so observation design is an inference-cost decision as much as a modeling one. The second is how far distillation plus DAgger can carry a sub-1B model before RL is needed.

The naming in the brief points the wrong way. "System One" in its 2026 sense is a fast picker over options someone else enumerated. For Dark Pawns, that someone is a careful text parser and grammar builder, and `pkg/agentobs` is the highest-leverage piece of code in this plan. It feeds training, serving, evaluation and containment from one deterministic source.
