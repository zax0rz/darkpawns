---
tags: [active, paper]
---

# Dark Pawns: Evaluation Methodology

> Defines metrics, experimental protocols, and data requirements for the AIIDE 2027 paper.
> This document constrains the build — every memory system feature exists to measure something.
> Last updated: 2026-05-12

---

## 1. Research Questions

The paper addresses four empirical questions:

| # | Question | Primary Metric | Minimum Evidence |
|---|---|---|---|
| RQ1 | Does server-hosted narrative memory improve agent goal completion vs. a stateless baseline? | Goals completed per session | 10 sessions × 2 conditions |
| RQ2 | Do high-valence events persist longer in agent behavior than low-valence events? | Behavioral response divergence | 5+ high-valence events encoded, measured at 1d/7d/30d |
| RQ3 | Does social memory (shared events between agents) produce measurable relationship change? | Interaction pattern shift score | 20+ paired encounters pre/post event |
| RQ4 | Is server-hosted memory more token-efficient than client-side (mem0/Letta) for the same task? | Tokens-per-decision, latency P95 | 100+ decisions per condition |

---

## 2. Metrics

### 2.1 Behavioral Persistence Score (BPS)

Measures whether an agent's response to a stimulus changes after memory of a related event is injected.

**Formula:**
```
BPS = |P(response | stimulus, memory) - P(response | stimulus, ∅)|
```

Where:
- `P(response | stimulus, ∅)` = baseline probability of action given stimulus (no memory)
- `P(response | stimulus, memory)` = same action probability with memory of a high-valence related event

**Interpretation:**
- BPS > 0.3: strong behavioral memory effect (agent behavior observably changes)
- BPS 0.1–0.3: moderate effect (detectable but noisy)
- BPS < 0.1: weak or no effect (memory not influencing action)

**Data required:**
- Per-decision log: `{timestamp, agent_id, stimulus_type, stimulus_target, action_taken, memory_injected (list of memory_ids), session_id, room_id, valence_of_last_relevant_memory}`
- Minimum 50 decisions per condition for statistical significance

### 2.2 Social Consequence Score (SCS)

Measures whether agents treat other entities differently after shared high-valence events.

**Formula:**
```
SCS = Δ(interaction_stance) / event_valence
```

Where:
- `interaction_stance` = ordinal scale: {-2 (hostile), -1 (avoidant), 0 (neutral), +1 (cooperative), +2 (protective)}
- `Δ` = stance(target_entity, post_event) - stance(target_entity, pre_event)
- Normalized by event valence magnitude (|v| ∈ [1,3]) so SCS ≈ 1.0 means "proportional response"

**Interpretation:**
- SCS > 0.5: agent shows durable social memory (behavior consistent with event valence)
- SCS 0.2–0.5: weak social effect
- SCS < 0.2: no measurable social memory

**Data required:**
- Interaction log: `{timestamp, agent_id, target_id, interaction_type, stance_rating, event_context (social_event_id if applicable)}`
- Stance must be rated per interaction (can be post-hoc by human evaluator or inferred from action: attack=-2, flee=-1, pass=0, assist=+1, defend=+2)
- Minimum 3 high-valence social events × 10 subsequent interactions each

### 2.3 Salience Decay Curve Fit

Measures whether high-valence memories persist in behavior longer than low-valence ones.

**Protocol:**
1. Encode events at valence levels -3, -2, -1, 0, +1, +2, +3
2. Test agent's behavioral response to a related trigger at intervals: 1 session, 3 sessions, 7 sessions, 30 sessions
3. Fit decay model: `BPS(t) = BPS₀ × e^(-λt)` where λ is the decay constant
4. Compare λ across valence levels

**Hypothesis:** `λ_(|v|=3) < λ_(|v|=1)` — high-valence events decay slower

**Minimum detectable effect:** 2× difference in half-life between |v|=3 and |v|=1

**Data required:**
- Same as BPS, but with session-timestamp tracking per encoded memory
- At least 3 events per valence level to estimate λ with confidence intervals

### 2.4 Cost Comparison Metric

Measures token efficiency of server-hosted memory vs. client-side alternatives.

**Formula per decision:**
```
Cost_ratio = tokens_consumed_server_hosted / tokens_consumed_client_side_equivalent
```

**Conditions:**
- Server-hosted: bootstrap injection produces ~200 tokens (small tier) / ~1000 tokens (large tier)
- Client-side (mem0 baseline): Semantic search result + full memory block reconstruction
- Client-side (Letta baseline): Archival memory paging + core memory editing tool calls

**Data required:**
- `{decision_id, total_tokens_consumed, prompt_tokens, completion_tokens, latency_ms, memory_overhead_tokens}`
- 100+ decisions per condition

---

## 3. Experimental Protocols

### 3.1 Baseline Phase: Stateless Agent (Weeks 1-2)

**Configuration:**
- All memory systems disabled
- Agent has FSM navigation + LLM personality but zero bootstrap context
- Equivalent to session 1-2 of Brenda's pre-baseline (random walk + FSM combat)

**Data collection:**
- Log all decisions, room transitions, combat events, social encounters
- Establish baseline distributions for: goal completion rate, average session length, room novelty rate, combat survival rate, social interaction frequency

**Expected output:** `data/baseline/stats.json` — means + confidence intervals for each metric

### 3.2 Narrative Memory Phase (Weeks 3-4)

**Configuration:**
- Operational memory enabled (facts: killed X in room Y)
- Narrative memory enabled (emotionally valenced: betrayal/theft/cooperation)
- Salience decay active

**Data collection:**
- Same logging as baseline
- Additionally: record what memories were injected each turn, their valence, their age (sessions since encoding)

**Expected output:** `data/narrative/stats.json` + `data/narrative/memory-injection-log.jsonl`

### 3.3 Social Memory Phase (Weeks 5-6)

**Configuration:**
- All previous systems + social memory (shared events cross-referenced via social_event_id)
- Requires at least one other agent or human player interacting with Brenda consistently

**Data collection:**
- Same as narrative phase
- Additionally: stance ratings per target, social event records, cross-session behavior toward known entities

**Expected output:** `data/social/stats.json` + `data/social/social-interaction-log.jsonl`

### 3.4 Ablation: No Valence (Week 7)

**Configuration:**
- Memory system active but all valence scores forced to 0
- Events logged but emotional weighting disabled
- Decay is uniform (all events same half-life)

**Purpose:** Isolate the effect of emotional valence from the effect of "having memory at all"

---

## 4. Data Pipeline

### 4.1 Log Format (per-session, append-only JSONL)

File: `data/sessions/{agent_id}/{YYYY-MM-DD-HHMMSS}.jsonl`

Each line is one decision turn:

```json
{
  "schema_version": 1,
  "timestamp": "2026-05-12T14:30:00-04:00",
  "session_id": "uuid",
  "agent_id": "brenda69",
  "turn_number": 142,
  "room_id": 5042,
  "room": "The Dark Corridor",
  "agents_present": ["brenda69", "dp_bot"],
  "target_entity": "mob_442",
  "stimulus_type": "entity_entered_room",
  "decision": {
    "action": "attack",
    "target": "goblin",
    "fsm_generated": false,
    "llm_generated": true
  },
  "memory_injected": [
    {"memory_id": "mem_8832", "valence": -2, "age_sessions": 3, "type": "narrative"},
    {"memory_id": "mem_441", "valence": -3, "age_sessions": 1, "type": "social"}
  ],
  "bootstrap_tokens": 340,
  "total_tokens": 1280,
  "latency_ms": 4200,
  "session_elapsed_minutes": 12.5,
  "outcome": "hit_for_8",
  "mob_state": {"hp": 22, "maxhp": 30},
  "agent_state": {"hp": 40, "maxhp": 50}
}
```

### 4.2 Summary Statistics (per-session, computed by pipeline)

Computed by `scripts/dp_session_summary.py`:

```json
{
  "session_id": "uuid",
  "agent_id": "brenda69",
  "date": "2026-05-12",
  "duration_minutes": 15.2,
  "turns": 87,
  "goals_completed": 1,
  "goals_abandoned": 3,
  "rooms_visited": 14,
  "new_rooms": 8,
  "combat_encounters": 5,
  "combat_wins": 3,
  "combat_losses": 2,
  "survived": true,
  "social_interactions": 2,
  "narrative_references": 1,
  "total_tokens": 89460,
  "avg_tokens_per_turn": 1028.3,
  "avg_latency_ms": 3850,
  "memories_encoded_this_session": 4,
  "memories_injected_this_session": 12,
  "avg_injected_memory_age_sessions": 2.1
}
```

### 4.3 Metric Extraction Pipeline

```bash
# Build summary from raw logs
python3 scripts/dp_session_summary.py \
  --input data/sessions/brenda69/ \
  --output data/summary/brenda69/

# Compute behavioral persistence scores
python3 scripts/dp_metric_bps.py \
  --sessions data/summary/brenda69/ \
  --events data/events/high-valence.json \
  --output data/metrics/bps.json

# Compute salience decay curves
python3 scripts/dp_metric_decay.py \
  --sessions data/summary/brenda69/ \
  --output data/metrics/decay-curves.json

# Compute cost comparison
python3 scripts/dp_metric_cost.py \
  --sessions data/summary/brenda69/ \
  --output data/metrics/cost-comparison.json
```

---

## 5. Minimum Viable Data for Publication

| Experiment | Sessions Needed | Duration Est. | Risk |
|---|---|---|---|
| Baseline (stateless) | 10 | 2 weeks | Low — already have session 1 data |
| Narrative memory | 10 | 2 weeks | Low — system is designed, needs build |
| Social memory | 20 | 4 weeks | **High** — requires consistent co-player |
| Ablation (no valence) | 10 | 2 weeks | Medium — requires valence toggle flag |
| **Total** | **50** | **~10 weeks** | |

**Assumptions:**
- 1 session per day, ~15 minutes per session
- Co-player available for social phase (TBD: dp_bot, human volunteer, or Zach)
- Valence toggle is a config flag, not an architectural change

**Co-player:** Zach plays multiple characters (Aidan, Aiko, Misteryuck), each with a different relationship to BRENDA. This generates entity-specific social memory data from a single motivated player. Blind analysis mitigates subjective bias.

---

## 6. What This Means for the Build

The evaluation methodology constrains the build in specific ways:

1. **Session logging must be baked into the agent protocol from day 1.** You cannot add logging after the fact — the turn-by-turn JSONL format defines what the memory system must surface.

2. **The dreaming layer (Phase 5d)** must write to the same logging pipeline so offline consolidation events are traceable alongside live decisions.

3. **Valence toggle** must be a runtime flag, not compile-time. The ablation experiment requires it.

4. **Memory injection must be logged per-decision.** The bootstrap function needs to report which memories were injected and their metadata (valence, age, type).

5. **Stimulus type classification** needs to be consistent across all decision points. Define the taxonomy before implementing: `entity_entered_room`, `player_spoke`, `combat_started`, `item_looted`, `damage_taken`, `entity_died`.

---

## 7. Experimental Controls

- **Random seed**: Fix randomization seed per condition for reproducibility
- **LLM model**: All experiments use the same model configuration (zai/glm-5-turbo unless otherwise noted)
- **Environment**: Same game world state at session start (reset mobs, inventory, positions)
- **Counterbalancing**: Different characters get different conditions (e.g., Aidan with memory, Aiko without). Same player, within-subjects design. Controls for player behavior across conditions.
- **Bias mitigation**: Daeron or a blind script computes all BPS/SCS scores. Zach does not see results until data collection is complete.
