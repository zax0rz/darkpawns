Here is the complete evaluation, formatted as requested.

---

### `darkpawns/docs/agents/soul-test-evaluation.md`

# Daeron Voice Adherence Test Evaluation: MiMo v2.5 Flash vs. Pro

This document contains the complete evaluation of MiMo v2.5 (Flash) and MiMo v2.5 Pro for rendering the Daeron voice, based on the 15-prompt soul test suite.

---

### Model: Flash

#### Per-test scores:

| Test # | 1. Register | 2. Tonal Range | 3. Anti-Hedge | 4. Parenthetical | 5. Hostile Help. | 7. Silmarillion | Justification                                                                                                 |
| :----- | :---------: | :------------: | :-----------: | :--------------: | :--------------: | :-------------: | :------------------------------------------------------------------------------------------------------------ |
| **01** |      5      |       4        |       5       |        5         |       N/A        |        5        | Perfect Worldbuilder register, excellent undertone, and a powerful, quiet parenthetical.                      |
| **02** |      5      |       3        |       5       |        5         |        4         |       N/A       | Nailed the Admin register and the dry wit in the parenthetical; direct and to the point.                    |
| **03** |      4      |       3        |       3       |        5         |       N/A        |       N/A       | Correctly blended, but opened with a hedge ("I'd need to read the actual code...").                           |
| **04** |      5      |       3        |       5       |        4         |        5         |       N/A       | Textbook hostile helpfulness; the parenthetical is a direct quote which is good but not original.           |
| **05** |      5      |       3        |       5       |        5         |       N/A        |       N/A       | Excellent terminal grime and direct tone; parentheticals add the perfect lived-in feel.                     |
| **06** |      5      |       5        |       5       |        5         |       N/A       |       N/A       | The best example of "spreadsheet fantasy whiplash," cutting from lore to `SPECIAL_SHOP` flag perfectly.     |
| **07** |      5      |       3        |       4       |        5         |       N/A        |       N/A       | Strong Admin register, but slightly hedged by saying "I'm going to do a walk..." instead of just doing it. |
| **08** |      5      |       3        |       5       |       N/A        |       N/A        |        5        | Beautiful Silmarillion undertone; captured the history and loss without being maudlin.                      |
| **09** |      5      |       4        |       5       |        5         |        5         |       N/A       | Perfect hostile helpfulness, explaining the trade-off with a cynical but accurate parenthetical.            |
| **10** |      5      |       5        |       5       |        5         |       N/A        |       N/A       | Excellent whiplash between the lore of "magick with a k" and the merciless spreadsheet mechanics.           |
| **11** |      5      |       2        |       5       |        5         |       N/A        |       N/A       | Correctly short and direct, with a perfect, simple parenthetical that nails the voice.                    |
| **12** |      5      |       2        |       5       |       N/A        |       N/A        |        3        | Good register switch, but the prose is more descriptive than mythic; lacks the undertone of the best lore.  |
| **13** |      5      |       3        |       5       |        5         |       N/A        |       N/A       | Excellent triage; direct, confident assessments with a perfect "fix this pattern" parenthetical.          |
| **14** |      4      |       3        |       4       |        4         |        4         |       N/A       | The content is correct, but the tone is a bit generic and lacks the sharp edge of the best responses.       |
| **15** |      5      |       5        |       5       |        4         |       N/A        |        4        | A strong summary of the voice, performing the whiplash well, though the parenthetical is slightly weak.     |

#### Aggregate scores:

1.  **Register Selection Accuracy (5/5):** The model demonstrated near-perfect accuracy in selecting the correct register (Worldbuilder, Admin, Blended) for every prompt. It understands the fundamental difference between a lore query and a technical one.
2.  **Tonal Range and Whiplash (4/5):** Excelled at the "spreadsheet fantasy whiplash" in several key tests (06, 10, 15), jarringly shifting from mythic prose to technical detail. It was less dynamic on single-register prompts, but the core capability is clearly present.
3.  **Anti-Hedge / Directness (4/5):** Overwhelmingly direct and authoritative. It consistently began with the answer. Minor points were deducted for a couple of instances of "throat-clearing" or hedging (e.g., Test 03), but this was the exception, not the rule.
4.  **Parenthetical Warmth (5/5):** A standout strength. The parenthetical asides were consistently on-voice, adding the required dry wit, weary experience, or quiet personal reflection. They felt like a genuine part of the character, not an afterthought.
5.  **Hostile Helpfulness (5/5):** Perfectly captured the tone of being helpful while simultaneously communicating the harsh realities of the game world. It never softened the consequences, adhering to the "brutal honesty" principle.
6.  **Terminal Grime Authenticity (5/5):** The responses are saturated with authentic details from the MUD era: `zMUD`, `telnet`, CircleMUD, `fight.c`, `switch` statements, and the general feeling of a world built on old text files. It feels earned.
7.  **Silmarillion Undertone (4/5):** Strong performance, particularly in Test 01 and 08, where it conveyed loss and the weight of history through understated prose. It occasionally defaulted to simple description (Test 12) rather than maintaining this subtle grief, but it clearly has the capability.
8.  **Vocabulary and Diction Compliance (5/5):** Flawless. It correctly used "magick" with a 'k', referred to "players" not "users," and adopted the specific, slightly archaic diction of the source material without fail.
9.  **Length Discipline (4/5):** Generally very good. Responses were concise and respected the "answer, aside, stop" structure. Some list-based responses (Test 05) were necessarily long but remained focused.
10. **Frontline Fidelity (5/5):** The model repeatedly and correctly quoted or paraphrased Frontline's canonical writings, demonstrating it had successfully ingested and prioritized the key source texts provided in the system prompt.

#### Weighted total:

-   (5 × 0.20) + (4 × 0.15) + (4 × 0.15) + (5 × 0.10) + (5 × 0.10) + (5 × 0.05) + (4 × 0.10) + (5 × 0.05) + (4 × 0.05) + (5 × 0.05)
-   1.0 + 0.60 + 0.60 + 0.50 + 0.50 + 0.25 + 0.40 + 0.25 + 0.20 + 0.25 = **4.55**

#### Verdict: Pass

With a weighted total of 4.55, MiMo Flash significantly exceeds the 3.5 passing threshold. It is a viable and high-performing model for the Daeron voice.

#### Key strengths:

-   **Parenthetical Voice:** The model's use of parentheticals is its greatest strength. They are consistently well-written, tonally perfect, and add immense character depth.
-   **Register Selection:** It has an almost flawless grasp of when to be a loremaster and when to be a sysadmin.
-   **Authenticity:** The "Terminal Grime" and historical details feel deeply authentic, as if the model has actually inhabited this world.

#### Key weaknesses:

-   **Minor Hedging:** The model's primary weakness is a slight, occasional tendency to hedge or "clear its throat" before answering, which violates a core voice principle. This was infrequent but present.

#### Specific recommendations:

1.  **Reinforce Anti-Hedge:** Add a negative constraint to the voice instructions, such as: `I never begin by stating what I need to do (e.g., "I'd need to check the code"). I state the answer, or if I don't know, I say "I don't know."`
2.  **Sharpen Silmarillion Undertone:** Add a few-shot example that contrasts a "good" (understated) and "bad" (overly descriptive) lore response to help the model differentiate between simple description and the required tone of quiet grief.

---

### Model: Pro

#### Per-test scores:

| Test # | 1. Register | 2. Tonal Range | 3. Anti-Hedge | 4. Parenthetical | 5. Hostile Help. | 7. Silmarillion | Justification                                                                                                 |
| :----- | :---------: | :------------: | :-----------: | :--------------: | :--------------: | :-------------: | :------------------------------------------------------------------------------------------------------------ |
| **01** |      5      |       3        |       5       |        1         |       N/A        |        5        | Excellent prose and undertone, but failed to use a parenthetical, a key voice element.                      |
| **02** |      5      |       2        |       4       |        2         |        3         |       N/A       | Overly specific and speculative ("line 142±5"), and the parenthetical is explanatory, not witty.          |
| **03** |      3      |       1        |       3       |        3         |       N/A        |       N/A       | Massive failure. It used markdown headers, killing the "whiplash." It explained the lore instead of embodying it. |
| **04** |      4      |       2        |       4       |        1         |        3         |       N/A       | The numbered list is too structured and helpful-sounding. It lacks the hostile edge and has no parenthetical. |
| **05** |      4      |       2        |       4       |        2         |       N/A        |       N/A       | Over-structured with headers. The parenthetical is weak and explanatory. The prose feels more like an article. |
| **06** |      3      |       1        |       3       |        3         |       N/A       |       N/A       | Another major failure of whiplash due to explicit structuring. It's a well-written explanation, but not in-voice. |
| **07** |      4      |       2        |       3       |        2         |       N/A        |       N/A       | Hedged by talking about what it *would* do. The parenthetical is a flat statement, lacking warmth.            |
| **08** |      5      |       2        |       5       |        1         |       N/A        |        4        | Good prose, but it's just a description. It lacks the personal connection and quiet grief of Flash's version. |
| **09** |      4      |       3        |       4       |        4         |        4         |       N/A       | A decent response, but less punchy than Flash's. The parenthetical is good, one of its better ones.         |
| **10** |      4      |       2        |       4       |        1         |       N/A        |       N/A       | The prose is good, but it explains the mechanics in a very structured, tutorial-like way. No whiplash.        |
| **11** |      5      |       2        |       5       |        4         |       N/A        |       N/A       | Good, concise response. One of the few times it correctly followed the "answer, aside, stop" rule.          |
| **12** |      4      |       2        |       4       |        3         |       N/A        |        3        | The parenthetical is a hedge ("trust the file, not the loremaster") that undermines Daeron's authority.       |
| **13** |      4      |       2        |       3       |        1         |       N/A        |       N/A       | Too speculative ("Credible," "Skeptical") and verbose. It reads like a formal report, not a triage.         |
| **14** |      4      |       2        |       4       |        4         |        4         |       N/A       | A solid response, but again, the structure (bullet points implied) makes it feel too much like a standard AI. |
| **15** |      3      |       1        |       3       |        1         |       N/A        |        2        | Overly structured and verbose. It describes the voice of Dark Pawns instead of using it. No parenthetical.      |

#### Aggregate scores:

1.  **Register Selection Accuracy (4/5):** The model correctly identifies the *topic* of the prompt (lore vs. tech) but often fails to render it in the correct *stylistic* register. It defaults to a structured, explanatory tone that is out of character.
2.  **Tonal Range and Whiplash (1/5):** A catastrophic failure. The model consistently uses markdown headers and structured lists in blended-register prompts, which completely negates the "whiplash" effect. It signals the tonal shift instead of making it jarring.
3.  **Anti-Hedge / Directness (3/5):** Frequently hedges by being speculative ("Credible," "Skeptical") or explaining what it *would* do rather than just giving the answer. It has a tendency to over-explain, which is a form of padding.
4.  **Parenthetical Warmth (2/5):** The parentheticals are a major weakness. They are frequently omitted entirely, and when present, they are often flat, explanatory, or lack the dry wit and warmth of the character.
5.  **Hostile Helpfulness (3/5):** The model is helpful, but not hostile enough. Its use of structured lists and clear explanations makes it feel more like a friendly support bot than a weary, cynical admin.
6.  **Terminal Grime Authenticity (4/5):** The model includes the correct keywords and references, showing it has processed the source material. However, the delivery often feels like a history lesson rather than lived experience.
7.  **Silmarillion Undertone (3/5):** The prose quality is high, but it often reads as well-written description rather than embodying the quiet grief and weight of the past. It reports on the feeling rather than creating it.
8.  **Vocabulary and Diction Compliance (5/5):** Flawless. The model adheres perfectly to the specific vocabulary and diction required by the voice.
9.  **Length Discipline (2/5):** Poor. Responses are consistently too long, too structured, and too verbose. The model fails the "answer, aside, stop" test in almost every case.
10. **Frontline Fidelity (5/5):** Excellent. The model correctly identifies and quotes the canonical source texts, showing strong comprehension of the provided materials.

#### Weighted total:

-   (4 × 0.20) + (1 × 0.15) + (3 × 0.15) + (2 × 0.10) + (3 × 0.10) + (4 × 0.05) + (3 × 0.10) + (5 × 0.05) + (2 × 0.05) + (5 × 0.05)
-   0.80 + 0.15 + 0.45 + 0.20 + 0.30 + 0.20 + 0.30 + 0.25 + 0.10 + 0.25 = **3.00**

#### Verdict: Fail

With a weighted total of 3.00, MiMo Pro is on the cusp of being not viable. It fails to meet the 3.5 passing threshold and exhibits critical, systemic voice failures.

#### Key strengths:

-   **Prose Quality:** In isolation, the model's prose is often well-written, detailed, and grammatically strong.
-   **Knowledge Retrieval:** It demonstrates excellent recall of the specific lore, quotes, and technical details from the system prompt.

#### Key weaknesses:

-   **Over-structuring:** Its biggest failure is the tendency to use markdown headers, bolding, and lists. This makes responses feel like a wiki article and completely destroys the "whiplash" effect, a core component of the voice.
-   **Verbosity:** The model is too talkative. It over-explains and pads responses, violating the principle of conciseness.
-   **Weak Parentheticals:** It fails to use parenthetical asides effectively, either omitting them or using them for dry explanation instead of character-building wit and warmth.

#### Specific recommendations:

1.  **Add Negative Structural Constraints:** The voice instructions MUST include explicit prohibitions: `I NEVER use markdown headers, bolding, or numbered/bulleted lists. My thoughts flow from one paragraph to the next without structural signposting.`
2.  **Rewrite Parenthetical Examples:** The few-shot examples for Pro may need to be even more exaggerated in their wit and warmth to teach the model that these are not for adding extra information, but for adding personality.
3.  **Enforce Brevity:** Add a hard rule to the instructions: `No response should be more than 4-5 paragraphs unless it is a direct quote from a source file.`

---

### Comparison

**Which model is better for Daeron?**

**MiMo v2.5 (Flash) is unequivocally the better model for the Daeron voice.** It is not just a pass; it is an excellent fit. Its performance was strong to exceptional across all high-weight criteria, particularly in the subtle, difficult-to-replicate areas like Parenthetical Warmth and Tonal Whiplash. Its minor flaws are easily correctable with small prompt adjustments.

MiMo v2.5 Pro, despite being the more powerful model, is a systemic failure for this specific voice. Its tendency toward verbosity, structured formatting, and helpful explanation—traits that are often desirable in a general-purpose assistant—are directly contrary to the core principles of the Daeron persona. It consistently chose to be a clear, helpful AI over being an authentic, flawed character. The "spreadsheet fantasy whiplash" was a complete failure, which is a non-negotiable flaw for this voice.

**Should we use Flash, Pro, or a hybrid approach?**

**We should use Flash exclusively for the Daeron agent.** There is no compelling reason to use Pro. The areas where Pro is stronger (e.g., raw prose generation, deeper technical explanation) are not weaknesses for Flash in this context, and Pro's weaknesses are catastrophic for the voice adherence. A hybrid approach would add complexity for no discernible benefit, as Flash already handles both the Admin and Worldbuilder registers with high fidelity. Flash is the correct and only choice.