# Pan et al. translation-bug taxonomy (verbatim)

Source: Rangeet Pan, Ali Reza Ibrahimzada, Rahul Krishna, Divya Sankar, Lambert
Pouguem Wassi, Michele Merler, Boris Sobolev, Raju Pavuluri, Saurabh Sinha,
Reyhaneh Jabbarvand. *"Lost in Translation: A Study of Bugs Introduced by Large
Language Models while Translating Code."* ICSE 2024. arXiv:2308.03109v3
(16 Jan 2024). Taxonomy defined in §4.1 ("Taxonomy of Translation Bugs");
prevalence in §4.2 and Table 4.

The paper reports **15 bug categories organized into five groups** (§4.1.1).
Category names below are copied exactly as printed. Group labels are the row
labels used in Table 4.

Research-question numbering (§1): **RQ1** Effectiveness in Code Translation
(RQ1.1, RQ1.2); **RQ2** LLM-Based Translation Bugs (RQ2.1 types / RQ2.2
prevalence / RQ2.3 real-world projects); **RQ3** Comparison with Alternative
Approaches; **RQ4** Mitigating Translation Bugs. Our study's research
question 2 (the mistakes) is their RQ2.

---

## Group A — Syntactic and semantic differences between languages
*(Table 4 row label; defined §4.1.2 "Syntactic and semantic differences between
the source and the target languages")*

| Code | Category (verbatim) | Definition as given |
|---|---|---|
| A1 | Violating target language requirements | Each PL has its own rules the code must adhere to (e.g. Go rejects unused imports). |
| A2 | Duplicating source syntax to the target language | Copying source-PL syntax that does not exist in the target PL (e.g. `atan2l` left in Go). |
| A3 | Mismatch of API behaviors in the source and target | A source API is mapped to a target API whose behaviour differs (e.g. Java `String.substring` → Go `strings.IndexByte`). |
| A4 | Incorrect use of operators | Operators differ across PLs (e.g. Python `//` floor division translated as Java `/`). |

*(Note: §4.1.2 prose says "There are five bug categories in this group", but
Table 4 and the §4.1.2 body list only A1–A4. The count of 15 total categories
across the five groups is consistent only with A having four; recorded as a
discrepancy, not resolved.)*

## Group B — Dependency & logic bugs in the translated code
*(Table 4 row label; defined §4.1.3 "Dependency and logic bugs in the translated
code")*

| Code | Category (verbatim) | Definition as given |
|---|---|---|
| B1 | Missing library imports | Import statements omitted or incorrect after translation. |
| B2 | Missing definition | Definitions/implementations of data types, methods, etc. omitted. |
| B3 | Incorrect loop and conditional statements | Bugs in translating loops and conditionals (e.g. off-by-one in slicing). |
| B4 | Inclusion of logic not in the source | Code unrelated to any source logic, so behaviour diverges (e.g. wrong initial value). |
| B5 | Removal of logic from source | Part of the source program not translated (e.g. a `#define` dropped). |
| B6 | Mismatch of behavior after replacing API call | A source API call replaced by custom logic instead of an equivalent target API. |

## Group C — Data-related Bugs
*(Table 4 row label; defined §4.1.4 "Data-related Bugs")*

| Code | Category (verbatim) | Definition as given |
|---|---|---|
| C1 | Incorrect data type | Incorrect types assigned to variables, including equivalent types with different behaviour across PLs. |
| C2 | Incorrect input parsing | Mistranslating code that reads data in a specific format. |
| C3 | Output formatting | Logic correct but output formatted differently (e.g. an extra space). |

## Group D — Model specific constraints
*(Table 4 row label; defined §4.1.5 "D: Model-Specific Bugs"; no sub-category)*

| Code | Category (verbatim) | Definition as given |
|---|---|---|
| D | Model specific constraints | Bugs specific to LLM design: natural language between code, exceeding token size, etc., causing compilation errors or no output. |

## Group E — Others
*(Table 4 row label; §4.1.5; no sub-category; excluded from the taxonomy proper)*

| Code | Category (verbatim) | Definition as given |
|---|---|---|
| E | Others | Unsuccessful translations related to the experiment setup (e.g. memory issues). The paper states these "do not represent LLM-introduced bugs" and are "not include[d] … in the taxonomy". |

---

## Prevalence in the paper (§4.2, Table 4)

Across 6,197 attempted translations (31 language pairs) there were 1,558
failures, manually labelled into 1,748 bugs (a failure can carry several labels).

| Group | Share of all bugs (Table 4, Total) |
|---|---:|
| A: Syntactic and semantic differences between languages | 30.5% |
| B: Dependency & logic bugs in the translated code | 24.2% |
| C: Data-related Bugs | 33.5% |
| D: Model specific constraints | 6.6% |
| E: Others | 5.1% |

Within-group leaders named in the paper's findings: A1 (24.3% of all bugs) is
almost 80% of group A; C2 "Incorrect input parsing" is 54% of group C (18.1% of
all bugs); C1 is 34.3% of group C (11.5% of all bugs). For real-world projects
(RQ2.3, Finding 6) the most prevalent are model-specific constraints (29.4%),
missing imports (27.4%) and removal of source logic (15.7%).
