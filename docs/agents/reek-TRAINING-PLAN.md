# Reek — Training Plan

> "My name is Reek. I crawl the pipes."
> A Go code review model that speaks in the voice of a broken man from Game of Thrones, trained on a 20-year-old MUD codebase.

---

## Overview

Three-phase approach. Phase 1 is cheap and runs on Mac Mini. Phase 2 is the real deal — rented GPU, continued pre-training on domain data. Phase 3 is polish and release.

**End state:** A 7B parameter model that deeply understands Go, knows the Dark Pawns codebase and lore, channels Reek from Game of Thrones, and produces nightly code review reports that are genuinely useful and deeply unsettling.

---

## Phase 1: QLoRA Fine-Tune (Mac Mini, $0)

**Goal:** Teach an existing Go coder model to speak like Reek. Personality layer only.

**Base model:** Qwen2.5-Coder-7B-Instruct (HuggingFace)
- 7B params, 4-bit quantized fits in ~6GB VRAM
- Already knows Go well. Already follows instructions.
- Fine-tuning teaches it HOW to respond, not WHAT to know.

**Toolchain:**
- **Unsloth-MLX** — Mac-native fine-tuning library. Same API as Unsloth, runs on Apple Silicon. SFT (Supervised Fine-Tuning) support.
- **mlx-lm** — Apple's ML framework for LLMs. Alternative to Unsloth-MLX. CLI for quick inference testing.
- **Hardware:** Mac Mini M4, 32GB unified memory. More than enough for 7B QLoRA.

**Training data needed (~200-500 examples):**

Generate these by hand (with LLM assist):

1. **Reek-voiced reviews (150-300 pairs):**
   - Input: Go code snippet (real Dark Pawns code)
   - Output: Reek's review in character
   - Cover: nil safety, race conditions, dead code, error handling, idiomatic Go
   - Vary severity: CRITICAL findings get more voice, LOW findings stay brief

2. **System prompt reinforcement (50-100 pairs):**
   - Input: "Review this code for bugs" + Go snippet
   - Output: Structured Reek report format (severity, file, what, why)
   - Teaches the report structure alongside the personality

3. **Negative examples (20-50 pairs):**
   - Input: Go code with bugs
   - Output labeled "WRONG" — examples of what NOT to say
   - Fixes hallucinated bugs, overconfident findings, architectural suggestions

4. **Edge cases (20-50 pairs):**
   - Clean code with no bugs → "nothing. clean. didn't find anything tonight."
   - Test files → skip, not production code
   - Lua files → skip, not Go

**Data format:** ShareGPT or Alpaca JSONL. Unsloth-MLX supports both.

**Training process:**
```
1. pip install unsloth-mlx
2. Load Qwen2.5-Coder-7B-Instruct in 4-bit
3. Add LoRA adapters (rank 16-32, alpha 32)
4. Train on dataset, 2-3 epochs, learning rate 2e-4
5. Merge adapters → final model
6. Test with: mlx_lm.generate -m ./reek-v1 --prompt "Review fight.go for nil safety issues"
7. Iterate: add more training data for outputs that don't land right
```

**Expected time:** 2-6 hours on Mac Mini (7B, 500 examples)

**Expected cost:** $0 (local hardware)

**Output:** `reek-v1-qlora.gguf` — usable immediately on Mac Mini via Ollama or mlx-lm

---

## Phase 2: Continued Pre-Training (Rented GPU, ~$50-150)

**Goal:** Bake Dark Pawns knowledge and GoT lore into the model's weights. Not context injection — actual internalized knowledge.

**Base model:** The Phase 1 QLoRA output (or fresh Qwen2.5-Coder-7B, then re-apply QLoRA after)

**What this phase actually does:**
- The model will *know* Dark Pawns internals without needing them in context
- GoT dialogue, character dynamics, and Reek's arc will feel natural, not prompted
- Modern Go best practices, common patterns, CircleMUD quirks — all baked in

**Toolchain:**
- **Unsloth** (not MLX) — NVIDIA GPU version. Much faster for pre-training.
- **Axolotl** — alternative. More configurable, supports continued pre-training natively. Probably the better choice for this phase.
- **Hardware:** Rent 1× A100 80GB on RunPod ($1.19-1.49/hr)

**Training data needed (corpus, not examples):**

### Corpus A: Dark Pawns Codebase (~71K lines Go, ~73K lines C)
- All `.go` files in `pkg/`, `cmd/`, `server/`
- Original C source in `src/` (for CircleMUD pattern knowledge)
- Lua scripts in `lib/world/scripts/` (for context on what the Go code serves)
- World files: `.mob`, `.obj`, `.wld`, `.zon` (for domain understanding)
- SOUL.md files, RESEARCH-LOG.md, ROADMAP.md (for project knowledge)

### Corpus B: Dark Pawns Lore
- `docs/lore/history.md` — the full timeline
- `docs/wayback/background.md` — the canonical backstory (Friar Drake letter)
- `docs/wayback/classes.md`, `faq.md`, `features.md` — game mechanics
- Brand voice docs, agent protocol spec
- Any historical player logs or transcripts that exist

### Corpus C: Game of Thrones — Reek Specific
- All Theon/Reek chapters from A Dance with Dragons (ADwD)
  - "The Prince of Winterfell"
  - "The Turncloak"
  - "A Ghost in Winterfell"
  - "Theon I" through "Theon VII"
- Key Ramsay/Theon scenes from A Clash of Kings
  - "Theon III" (taking Winterfell)
  - "Theon V" (Reek chapters)
- Dialogue transcripts: every Reek interaction with Ramsay, every "my name is Reek" moment
- Character analysis: Reek's psychology, PTSD, identity dissolution, the power dynamics

### Corpus D: Modern Go Knowledge
- Effective Go (go.dev/doc/effective_go)
- Go Code Review Comments (github.com/golang/go/wiki/CodeReviewComments)
- Common Go patterns: error handling, concurrency, interfaces, generics
- Go 1.22+ changelogs (recent features)
- Static analysis common findings (staticcheck rules, go vet patterns)

### Corpus E: Code Review Domain
- Real Go code review examples from open source repos
- Common Go bugs: nil pointer, goroutine leak, race condition, slice misuse
- MUD/game server patterns: game loops, state machines, event systems

**Corpus size estimate:**
- A: ~200K tokens
- B: ~20K tokens
- C: ~150K tokens (Reek chapters are substantial)
- D: ~100K tokens
- E: ~100K tokens
- **Total: ~570K tokens**

For continued pre-training this is small — that's actually good. Means fast training.

**Training process:**
```
1. Rent RunPod A100 pod (Ubuntu, PyTorch preinstalled)
2. pip install axolotl
3. Configure axolotl.yml:
   - base_model: Qwen2.5-Coder-7B
   - dataset: your JSONL corpus
   - sequence_length: 4096
   - batch_size: 4 (A100 80GB handles this easily)
   - learning_rate: 2e-5 (lower than fine-tuning)
   - epochs: 2-3
4. Run training: python -m axolotl.cli.train config.yml
5. Download merged model
6. Re-apply Phase 1 QLoRA on top (or do both in one pass)
```

**Expected time:** 4-12 hours on 1× A100

**Expected cost:** $5-18 per run (at $1.19-1.49/hr)
- Budget 3-5 iterations for quality: **$15-90**
- Total with iteration buffer: **~$50-150**

**Output:** `reek-v2-base.gguf` — model that *knows* Dark Pawns and GoT internally

---

## Phase 3: Polish, Integration & Release ($0-50)

### A. Evaluation
- Build a test set of 50 known bugs from the Dark Pawns codebase
- Run Reek against them. Measure: recall (did it find the bug?), false positive rate (did it report non-bugs?)
- Compare against raw Qwen2.5-Coder-7B with SOUL.md as context (the "no training" baseline)
- If Reek-v2 doesn't beat the baseline on accuracy, the training data needs work

### B. Quantization for Production
- Quantize to Q4_K_M for Mac Mini inference (~4.5GB VRAM)
- Test: mlx_lm.generate with real Dark Pawns files
- Verify personality holds at system prompt + single-shot, not just few-shot

### C. Nightly Crawl Integration
```
Cron: 3 AM daily
→ Script triggers Ollama/llama.cpp with Reek model
→ Feeds Go source files (batched)
→ Collects structured report
→ Posts to Discord #dark-pawns
→ Daeron triages in the morning
```

### D. Web Interface (the fun one)
Simple web app (Go or Python) that:
- Lets you paste Go code and get a Reek review in real-time
- Shows the report in a terminal-style dark UI (monospace, green text, flicker effect?)
- Maybe: a simple "Submit code for review" page on darkpawns.com
- Could be a Hugo page with a WASM-inference backend (edge deployment, no server needed)
- OR: simple Flask/FastAPI server on the Mac Mini that accepts Go snippets

### E. HuggingFace Release
- Model card: "Reek-7B — Go code review model trained on a 20-year-old MUD, speaking in the voice of a broken man"
- Include: training data description, evaluation results, sample outputs
- License: Apache 2.0 (Qwen's license)
- The GoT data needs careful licensing: book excerpts are copyrighted. Options:
  1. Only use fan wiki content (less copyright risk, less literary quality)
  2. Paraphrase key scenes instead of quoting directly
  3. Use dialogue-only excerpts under fair use (short quotations)
  4. Don't include GoT text in the weights — keep it as RAG context only

---

## Cost Summary

| Phase | Hardware | Cost | Time |
|-------|----------|------|------|
| 1. QLoRA Fine-Tune | Mac Mini (local) | $0 | 2-6 hours |
| 2. Continued Pre-Training | RunPod A100 ($1.19-1.49/hr) | $50-150 | 4-12 hours × 3-5 iterations |
| 3. Integration | Mac Mini (local) | $0 | 2-3 days dev |
| 3E. Web hosting | GitHub Pages / Vercel | $0 | 1 day |
| **Total** | | **$50-150** | **1-2 weeks** |

---

## Risk Register

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| 7B too small for quality reviews | Medium | High | Try 14B with Q4 (fits in 10GB, tight on Mac Mini). Or do Phase 2 on 32B and ship the 7B fine-tune |
| GoT copyright on HuggingFace | High | Low | Use fan wiki/paraphrased data. The Reek personality can exist without direct book quotes |
| Model hallucinates bugs | Medium | Medium | Evaluation test set catches this. Add more negative training examples |
| Personality too thick, loses accuracy | Medium | Medium | A/B test with and without personality layer. Keep the structural report format strict |
| Mac Mini can't serve model fast enough | Low | Low | 7B Q4_K_M inference at ~8-12 tok/s on M4. A nightly report doesn't need speed |
| Nobody cares about the HuggingFace release | High | Zero | Who cares. The point is the Dark Pawns pipeline. The release is dessert. |

---

## Timeline (Post-Mac Mini)

**Week 1:** Phase 1 QLoRA. Generate training data. Run first fine-tune. Test.
**Week 2:** Phase 2 continued pre-training. First GPU run. Evaluate.
**Week 3:** Iterate on Phase 2 based on eval results. Polish.
**Week 4:** Phase 3. Crawl integration. Test with real codebase.
**Week 5:** Web interface MVP. HuggingFace release. Tweet about it.

---

## Open Questions

- [ ] Qwen2.5-Coder-7B or 14B as base? 7B is safer for Mac Mini inference. 14B is better quality. Test both.
- [ ] GoT data licensing strategy for HuggingFace release
- [ ] Does the Mac Mini need more than 32GB for anything? (No, 7B Q4 fits in 6GB)
- [ ] Web interface: WASM edge inference or simple API server?
- [ ] Should Daeron's model also be fine-tuned, or just use SOUL.md as context? (Probably context — Daeron needs to be flexible)
- [ ] How does Reek's report get scored? Daeron triage → Zach verdict → feedback signal for future training?

---

_This plan lives at darkpawns/docs/agents/reek-TRAINING-PLAN.md_
