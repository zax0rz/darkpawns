# Dark Pawns Brand Voice Guide v2.1

> Data-driven. Based on analysis of in-game help text, source code messages,
> race descriptions, room descriptions, social commands, wizard documentation
> from Dark Pawns 2.2 (CircleMUD-derived, c. 1999-2005), and website content
> from darkpawns.com (R.E. Paret / Frontline, preserved via Wayback Machine).

---

**Changelog:**
- **v2.1:** Added the Anti-LLM Prose Standard, the dash ban, archive authorship labels, and a pre-publication enforcement model. August 2026.
- **v2.0**: Incorporated Gemini structured review. Introduced three-layer voice framework (Engine / Edgelord DM / Mythic Admin), replaced Core Principles with Gemini Pillars, added Hostility Transfer Rule, expanded content sensitivities. Major restructure, not a patch.
- **v1.1**: Enriched with website content (Frontline), April 2026
- **v1.0**: Generated from source material analysis, April 2026

---

## 1. The Three-Layer Voice Framework

Dark Pawns does not have one voice. It has a geological stratification of voices, written by different people at different times, for different purposes. The v1.0 document tried to blend them into a single "world-weary dungeon master." That was wrong. The game speaks like **Jeremy Elson** (CircleMUD) being overwritten by **Derek Karnes** (Serapis), who is then being overwritten by **R.E. Paret** (Frontline). Each layer maps to a different modern channel. The framework solves the "whose voice is this?" problem by assigning the right historical voice to the right job.

### Layer 1: The Engine (CircleMUD / Jeremy Elson)

Dry, UNIX-like, purely functional. This is the C-programmer voice: terse, technical, no personality at all. It exists because someone needed to document command syntax and nothing more.

> "An alias is a single command used to represent one or more other commands." (help ALIAS)
> "The variables $1, $2, ... $9 may be used to represent arguments given with an alias." (help ALIAS)

**Maps to:** GitHub docs, technical prose, Go-port code comments, README internals. When you're writing for developers evaluating the codebase, use this voice. Lead with what the thing is. Technical clarity first.

### Layer 2: The Edgelord DM (Serapis / Derek Karnes)

Hostile, cliquey, unfiltered, insults-as-affection. This is the late-90s internet frat energy: the MUD admin as petty god who will delete you for fun. It's the voice that calls players "pansy-ass whiners" and threatens removal from the game for junking stolen equipment.

> "'Bout Fucking Time" (help BFT)
> "Pansy-ass whiner; also known as 'Delete me because I'm a fool'" (help PAW)
> "If you junk equipment you steal, you will be removed from the game." (help PSTEAL)
> "More than three players is not tolerated and will be severely punished when caught (I think this policy is More than generous, why push it?)" (help MULTIPLAY)

**Maps to:** In-game historical text (preserve as-is in the Go port), social commands, hidden Easter eggs. This layer is a historical artifact. It does NOT appear in any new public-facing 2026 content. See the Hostility Transfer Rule (Section 3) for the critical distinction.

### Layer 3: The Mythic Admin (Frontline / R.E. Paret)

Sweeping lore prose combined with a warm, blog-era community voice. This is the register that sells the world through literary imagery, then pivots to casual admin-to-player communication in the next paragraph. It's the voice of someone who genuinely loves the game and the people playing it.

> "Like a great game of chess, the world has become a board filled with bishops and kings, stately queens, white knights and dark pawns striving to rise through the ranks into godhood." (background.html)
> "Dark Pawns 3.0 is (finally!) out in beta." (main.html)
> "Booo... I died. What do I do now?" (faq.html)

**Maps to:** Website, marketing, modern community updates (Discord announcements, blog posts), FAQ, social media. This is the primary voice for the 2026 revival. When in doubt, write in this register.

### How to use the framework

| You are writing... | Use this layer | Why |
|---|---|---|
| GitHub README, API docs, code comments | Layer 1: The Engine | Developers need clarity, not flavor |
| New in-game help text, room descriptions, spell flavor | Layer 2 + Layer 3 blend | World-weary DM tone with Frontline's prose reach |
| Website lore, marketing copy, Discord announcements | Layer 3: The Mythic Admin | Seduces before it threatens |
| Social media, taglines | Layer 3 compressed | Gallows humor in 280 characters |
| Social commands, Easter eggs, hidden developer references | Layer 2: The Edgelord DM | Historical preservation only |
| Any new content threatening players | **NONE** | See Hostility Transfer Rule |

---

## 1.5 Frontline's Voice

R.E. Paret (Frontline) wrote nearly everything on the original website: the world lore, class descriptions, features list, FAQ, news posts, and wizard list. His writing reveals two distinct personas that exist in productive tension:

### The Worldbuilder (website lore, class descriptions)

Frontline's lore writing is genuinely literary. It is not purple. He does not waste words, but he reaches for imagery that the in-game help text never attempts.

> "Like a great game of chess, the world has become a board filled with bishops and kings, stately queens, white knights and dark pawns striving to rise through the ranks into godhood." (background.html)
>
> "Outside those thick walls that enable 'civilized' persons to sleep so well lies an untamed land, some partly still ruled by the power of fangs, claws, and magic." (background.html)
>
> "Twisted and warped by forgotten times into a vision that would make any sane man's nightmare seem like a pleasant summer daydream." (background.html)

The class descriptions follow the same register: more elevated than the help files, more willing to paint with a broad brush:

> "Assassins are cold and deadly killers for hire, whose knowledge of subturfuge, stealth and the Black Arts makes them extremely efficent, extremely deadly opponents." (class.html)
>
> "The Magus represents the extreme power of magickal spellcraft, wielding power on a cosmic scale, able to create, destroy, or shape realities at whim." (class.html)

Key traits of the worldbuilder register:
- **Sentence rhythm varies deliberately**: short declarative punches followed by longer, rolling clauses. The in-game help is more uniformly staccato.
- **"Extremely" as intensifier**: appears twice in the Assassin description alone. The in-game help uses this word less often.
- **Worldbuilding through implication**: "because the southern culture views all non-humans as animals" (Ninjas) tells you as much about the world's bigotry as it does about the class.
- **"Wyldlands" (with a y)**: the website spelling. The in-game help uses "Wyldlands" too, but the website is where it gets its mythic weight: "magick runs strong and wild, crafting the very earth itself and giving the Wyldlands their name."

### The Admin (news posts, FAQ, features page)

This is a completely different Frontline: casual, self-deprecating, personally invested. The news posts read like a blog, not a changelog:

> "Dark Pawns 3.0 is (finally!) out in beta." (main.html)
>
> "Oh my god forums! Please please please register with your correct account name!" (main.html)
>
> "Moral of the story: Turn your speakers down if you are going to mud during lecture. :)" (main.html: the Aiko story)

The FAQ sits between the two personas: still helpful, but warmer:

> "Booo... I died. What do I do now?" (faq.html)

The features page does humble-bragging with a wink:

> "Dark Pawns has too many unique featues to list here, but this should give you a brief overview..." (followed by a comprehensive list) (features.html)
>
> "'TAG! You're it!', Capture the Flag, and other annoying player on player games." (features.html: note the self-aware "annoying")

Key traits of the admin register:
- **Parenthetical asides as personality injection**: "(finally!)", implied in the telnet quip about masochists
- **Exclamation points appear here**: the one place in the entire brand where the voice gets genuinely enthusiastic. Contrast with the in-game help, which almost never uses them.
- **Emoticons**: the lone :) on the Aiko story is the emotional watermark of the admin persona. The in-game text would never.
- **Self-aware humor**: "other annoying player on player games" names the thing honestly instead of selling it. This is trust-building, not marketing.
- **"Please please please"**: deliberate repetition that reads as genuine exasperation, not corporate enthusiasm.

### Touchstone quotes

These are the lines that *are* the voice. When writing new content in Frontline's register, these should feel like neighbors:

| Context | Quote | Persona |
|---------|-------|---------|
| World lore | "white knights and dark pawns striving to rise through the ranks into godhood" | Worldbuilder |
| World lore | "a vision that would make any sane man's nightmare seem like a pleasant summer daydream" | Worldbuilder |
| World lore | "Beware the unseen, adventurer. The power is there, cloaked in a forgotten tomb or a dragon's hoard, but it is there." | Worldbuilder |
| Classes | "wielding power on a cosmic scale, able to create, destroy, or shape realities at whim" | Worldbuilder |
| Features | "A custom mobile AI that makes the mobs in the game more interactive than any other mud out there" | Admin |
| Features | "and other annoying player on player games" | Admin |
| FAQ | "Booo... I died. What do I do now?" | Admin |
| News | "Dark Pawns 3.0 is (finally!) out in beta." | Admin |
| News | "Please please please register with your correct account name!" | Admin |
| Wizlist | "Wizards are a secret lot, and very powerful. They are not to be trifled with." | Both |

---

## 2. The Four Pillars

These four characteristics define the Dark Pawns voice. They replace the v1.0 Core Principles, which have been absorbed into the pillars where they fit. The remaining principles that don't map cleanly are preserved as supplementary items in Section 2.5.

### Pillar 1: Hostile Helpfulness

"Aggressive RTFM." The text gives you exactly the information you need, wrapped in contempt. It answers your question but makes you feel slightly stupid for asking. This absorbs the old "Answer the question, then stop" and "The world is dangerous" principles.

> "Yep, there are some, go find 'em." (help AREAS)
> "Just to prevent accidental quittings... This command doesn't DO anything, it simply is." (help QUI)
> "BACKSTAB: A way to sneak up on a person and attempt to place your dagger in his back, at exactly the point where it does most damage." (help BACKSTAB)
> "Fleeing from battle will also cost you some experience points, but hey, thats what you get for wimping out." (help FLEE)

The hostility is pointed at the *situation*, not the player. "Wimping out" is the voice's diagnosis, not an insult. The player already knows they're in trouble; the text just refuses to pretend otherwise.

**Do:** Lead with the useful information. Add exactly one dry aside, not a paragraph.
**Don't:** Pad help text with pleasantries, preamble, or marketing. Don't actually insult the reader's intelligence: the contempt is for the situation, not the person.

### Pillar 2: Spreadsheet Fantasy

Mythological prose immediately followed by raw mechanics and pricing. The whiplash between fantasy and math is a core aesthetic. This absorbs the old "Lore is woven into mechanics" principle.

> "Carving your own permenant niche in Dark Pawns is not easy," followed immediately by "2 level 30 guards: 60,000 / 1 Butler: 10,000 (min)" (help CASTLE)

Race descriptions do the same thing: flavor and stats share the same sentence.

> "Rakshasas are a race of malevolent spirits encased in flesh that hunt and torment humanity." (help RAKSHASA). This is both lore and a warning about their in-game behavior.

The website's features page turns spreadsheet into marketing:

> "Dreams and Nightmares" as a bullet point, not a system description. "Magical Tattoos, Talking Weaponry, and other very cool things": the specificity of the first two items earns the vagueness of the third.

**Do:** Let flavor and mechanics share the same sentence. Transition from mythic to mundane without signaling the shift.
**Don't:** Separate "lore entries" from "mechanics entries." Don't apologize for the math in the middle of the poetry.

### Pillar 3: Terminal Grime

Real-world intrusion. Keyboards, beer, telnet clients, late nights. The fantasy bleeds into physical reality. The game does not pretend the software is absent. It expects you to know about mud-clients, acknowledges that your link can drop, and tells you to go AFK when you grab a beer.

> "Use this command when you're going afk to get a beer, smoke a cigarette, etc." (help AFK)
> "If your link jams (freezes), you have a problem." (help LINK)
> "Three popular ones are Tintin, Tinyfugue(both for UNIX shell accounts) and Zmud (Windows based). Please do not bother the Imms and Gods about mud clients!" (help MULTIPLAY)
> "Note that in most clients, this typically erases scrollback." (help CLEAR)

This voice reminds you that you are sitting in a dorm room at 2 AM with a CRT monitor and a stack of pizza boxes. That does not break immersion. It makes the terminal part of the world.

**Do:** Let the real world bleed into the text. Reference keyboards, clients, screens, connections. Use the word "mud" as a verb.
**Don't:** Pretend the player isn't sitting at a computer. Don't hide the software layer in in-game help.

### Pillar 4: Cliquey and Unfiltered

Insider references, insults-as-affection, developer Easter eggs. This is the Layer 2 voice (The Edgelord DM). It is real, but it is **entirely contained** within socials, wizhelp, and credits. It rarely bleeds into actual gameplay mechanics.

> "'Bout Fucking Time" (help BFT)
> "Pansy-ass whiner; also known as 'Delete me because I'm a fool'" (help PAW)
> "Shit, he's beyond help." (help DEREK, referring to Derek Karnes)
> "Twink is a state of mind." (help TWINK)

**CRITICAL:** This pillar lives ONLY in in-game socials, hidden help entries, and wizard documentation. It never appears in public-facing 2026 content: no GitHub README, no Discord announcement, no website copy, no social media post.

**Do:** Preserve community in-jokes as discoverable secrets. New Easter eggs should follow the same pattern: hidden, discoverable, not required for gameplay.
**Don't:** Reference BFT, PAW, named developers, or insider language in any public-facing channel.

---

## 2.5 Supplementary Principles

These didn't fit cleanly into the pillars but are still binding.

### Side comments belong in parentheses

The voice injects personality through asides, parentheticals, and trailing thoughts rather than through full sentences of editorializing.

> "This command doesn't DO anything, it simply is." (help QUI/SHUTDOW)
> "Remember what your mother said however... its not nice to point." (help POINT)
> "(finally!)" (news post, main.html)

**Do:** Use asides and parentheticals for personality.
**Don't:** Write paragraphs of opinion in what should be reference material.

### Gods are distant and the staff is practical

Immortal documentation (wizhelp) is no-nonsense technical writing with an edge. It's the one place the voice shifts slightly more casual: gods talking to gods.

> "Obviously, this command should only be used in extreme disciplinary circumstances." (help FREEZE)
> "WARNING: This OLC will let you set values to values that shouldn't be set... (Hey, that rhymes!)." (help OLC)

**Do:** Write wizard docs with technical clarity and collegial directness.
**Don't:** Use wizard-style informality in mortal-facing text.

### Worldbuilding as invitation, not instruction

The website sells the world through prose, not feature lists. Frontline's background page does not explain the game's setting. It evokes it. The reader is not told what the world is; they are made to feel what it is like to be in it.

> "Outside those thick walls that enable 'civilized' persons to sleep so well lies an untamed land..."

This is seduction, not documentation.

**Do:** Write world copy that makes the reader want to know more.
**Don't:** Explain the world's systems before the reader cares about the world.

---

## 3. The Hostility Transfer Rule

This is the most important rule in this document.

**In 2026, the World is hostile, but the Developers are not.**

The original Dark Pawns frequently threatens players with admin action. "You will be removed from the game" (help PSTEAL). "Severely punished when caught" (help MULTIPLAY). "Do not abuse this command; if you do, it will be taken from you" (help TITLE). In 1999, MUD admins were gods of scarce server space. This tone was normal.

In 2026, threatening your userbase reads as amateurish and toxic. The hostility must transfer from the admins to the world.

| 1999 (preserve in-game) | 2026 (new content) |
|---|---|
| "If you junk equipment you steal, you will be removed from the game." | The *game world* punishes thieves. NPCs remember. Factions retaliate. The world is hostile: the developers are not. |
| "More than three players is not tolerated and will be severely punished." | The game enforces this mechanically. No threatening language needed. |
| "Do not abuse this command or it will be taken from you." | State the restriction clearly. "This command has a cooldown." Done. |

**Specific rules:**

- **Preserve** all historical threatening admin text in the Go port for authenticity. This is a museum piece, not a style guide for new writing.
- **NEVER** threaten players in new GitHub, Discord, website, or social media copy. Not even as a joke.
- **The mob/NPC AI should be mean.** The developers should not be.
- **Consequences remain punchlines**, but the punchline is "the game killed you," not "we deleted your account."

---

## 4. Tone Spectrum

```
Deadpan/Helpful ──────────────────────────── Irreverent
     │                                            │
     │         Layer 1: The Engine               │
     │         (GitHub docs, technical prose)      │
     │                                            │
     │   Layer 3: Mythic Admin (website)           │
     │   (accessible, seductive, keeps the edge)   │
     │                                            │
     │                In-game help                 │
     │                (mechanical, dry asides)      │
     │                                            │
     │                     Room descriptions       │
     │                     (atmospheric, immersive) │
     │                                            │
     │                        Social commands      │
     │                        (Layer 2 only)       │
     │                                            │
     │                        Wizard help           │
     │                        (collegial, blunt)     │
     │                                            │
     └────────────────────────────────────────────┘
```

| Channel | Layer | Position | Notes |
|---|---|---|---|
| **GitHub README** | Layer 1 | Left | Developer-facing, technical clarity first |
| **Website marketing** | Layer 3 | Center-left | Accessible to non-MUD players, keeps the edge but drops the jargon |
| **In-game help** | Layer 2 + 3 blend | Center | Mechanical, factual, with dry asides and world-weary DM tone |
| **Room descriptions** | Layer 3 | Center | Atmospheric and immersive, second-person |
| **Social commands** | Layer 2 | Center-right | Jokes, developer refs. Historical preservation only. |
| **Wizard help** | Layer 2 | Center-right | Technical but collegial, no patience for stupidity |
| **Social media** | Layer 3 | Right | Compressed irreverence, short-form gallows humor |

---

## 5. Vocabulary & Diction

### Signature spellings and capitalizations

- **"magick"** (always with a k): the energy field; "magic" appears only in class names ("magic user") and the wizardry context
- **"magickal"**: adjective form of magick
- **"Wyldlands"** (with a y): the untamed wilderness beyond civilization. Used on both the website and in-game help, but the website gives it its fullest mythic treatment: "magick runs strong and wild, crafting the very earth itself and giving the Wyldlands their name."
- **ALL CAPS for emphasis** in help cross-references and key terms (NEWBIE, FAQ, RENT, REMORT)
- **"mob"** / **"mobs"**: standard MUD terminology for NPCs, used freely
- **"&c" and "&n"**: CircleMUD color codes (cyan, normal); structural, not voice

### Typo and grammar stance

Favor slightly unpolished, direct C-programmer style over slick modern copywriting. The original text contains deliberate and accidental roughness: "loose EXP," "gorey details," "subturfuge," "efficent," "veracious appetites" (instead of voracious), missing commas, awkward clauses.

> "Fleeing from battle will also cost you some experience points, but hey, thats what you get for wimping out." (help FLEE: "thats" without apostrophe)
> "Because of their sneaky nature, the thief class is barred from using shields of any sort." (class.html: slightly off-rhythm)
> "The civilized world fears sorcery and sorcerers for their destructive powers, but they have no problem with religious magicks." (help MAGICK: casual "but they" shift)

**For new content:** Don't over-polish. Write clearly but don't sand off every rough edge. A missing comma or slightly awkward clause is more authentic than a perfectly proofread sentence. The voice is a C programmer writing fantasy, not a copywriter.
**For historical text in the Go port:** Preserve verbatim. Don't fix typos. They're part of the texture.

### Words and phrases used

- "thy enemies": archaic touch in spell descriptions (help HELLFIRE)
- "the Powers that Be": refers to the immortal staff (help BUG)
- "sanity intact": "survive the perils with your life and sanity intact" (help SHOPS)
- "gorey details": [sic] used in PSTEAL help
- "winking out": not used, but "snuffed" implied by death trap language
- "veracious appetites": [sic, probably meant voracious] help BITE

### Words avoided

- Modern tech terms (no "UI," "interface," "users": it's "players" and "mortals")
- Corporate language ("experience," "onboarding," "features" in the modern SaaS sense)
- Exclamation points (almost never in-game; the website FAQ/news posts use them sparingly: see Section 1.5 on Frontline's admin persona)
- Emojis or emoticons (the lone :) in the Aiko news post is the exception that proves the rule: it's personal, admin-to-player communication, not game text)

### Sentence structure

- **Mixed.** Short imperative sentences handle mechanics ("Type ALIAS alone to see a list"). Longer, flowing sentences handle lore and room descriptions. The syntax occasionally falters with missing commas and awkward clauses. That is part of the authenticity.

### Formality with players

- Addressed as "you" throughout
- No "sir/ma'am," no deference, no "please" in help text
- The FAQ is slightly more conversational: "I'm a psychopath.. help?" → "BEHEAD"
- Wizard docs use "you" but with the implicit understanding that the reader is competent

---

## 5.5 Anti-LLM Prose Standard

This section governs all new public copy. The voice framework tells us who is speaking. This standard stops that speaker from sounding like a language model wearing Dark Pawns as a costume.

### The dash ban

Em dashes and en dashes are banned in new output copy. This includes headlines, eyebrows, pills, body text, quotations selected for presentation, attribution, captions, button text, alt text, metadata, and social posts.

Do not replace every banned dash with a hyphen. Restructure the sentence first. A period, comma, colon, semicolon, or pair of parentheses will usually make the thought clearer. Use a hyphen only when the word or range actually calls for one.

Verbatim historical artifacts are the sole exception. In-game text, forum posts, logs, and archival transcriptions remain unchanged under the Fidelity Law. They must be labeled as verbatim preservation, not presented as newly authored website copy. Surrounding captions and attribution still follow the dash ban.

### No synthetic importance

Do not tell the reader that something is iconic, remarkable, immersive, groundbreaking, beloved, vibrant, rich, timeless, unique, or deeply significant. Show the room, rule, person, incident, or surviving artifact that earns the judgment. If there is no specific evidence, cut the claim.

Avoid launch-copy inflation:

- "More than a game"
- "Not just X, but Y"
- "Where X meets Y"
- "From humble beginnings to..."
- "At its core"
- "A testament to"
- "A rich tapestry"
- "A labor of love"
- "A living, breathing world"
- "Step into," "delve into," "embark on," "discover," or "unleash"
- "Seamlessly," "robust," "dynamic," "curated," or "elevated"
- "Whether you are a veteran or a newcomer"

These phrases are not always false. They are worse than false in this context: they could describe anything.

### No trailer rhythm

Do not manufacture drama with sentence fragments, one-line paragraphs, or escalating sets of three.

Bad:

> A lost world. A forgotten codebase. One final resurrection.

Better:

> Dark Pawns ran from 1994 until roughly 2010. Its code and world files survived.

Vary sentence length because the thought requires it, not because a cadence generator expects a reveal. Frontline can write a long sentence. Let him. Do not chop it into fake gravitas.

### Concrete before atmospheric

Name the thing. Prefer a dated forum argument, a room description, a class rule, a host name, or a line from Frontline over a paragraph about community, nostalgia, innovation, or legacy.

Every paragraph of new website copy should contain at least one concrete noun that belongs specifically to Dark Pawns. If the paragraph still works after replacing "Dark Pawns" with another game's name, it is probably filler.

Atmosphere must grow from evidence. "Players argued about assassins at 3 AM" has a person, a subject, and a time. "A passionate community forged unforgettable memories" has none of them.

### State what kind of text this is

Historical material must carry one of these labels in its metadata or presentation:

- **Verbatim:** Exact source text. Preserve spelling, punctuation, and period language.
- **Transcription:** Source text converted from another format. Only structural formatting changed.
- **Edited excerpt:** Source text shortened or lightly cleaned. Disclose what changed.
- **Summary:** A factual account written from a cited source. Never style it as the original post.
- **Reconstruction:** Multiple incomplete sources assembled into one account. Name the gaps and the reasoning.
- **Original:** New Dark Pawns copy governed by this guide.

An LLM summary is not an archive. A title and plausible paragraph cannot stand in for a missing post. Publish the source, label the summary honestly, describe the gap, or stay silent.

### Source or silence

No new date, number, superlative, technical claim, historical claim, or statement about community sentiment ships without a source. Repository data can source a count. A preserved page can source a quotation. Zach can source his own role and experience. A model's confidence sources nothing.

When sources disagree, expose the disagreement or resolve it in the relevant single source of truth. Do not average competing facts into smooth prose.

### Drafting order

Before writing copy:

1. Choose the channel and voice layer.
2. Gather the facts, source text, and links.
3. Label the material as verbatim, transcription, edited excerpt, summary, reconstruction, or original.
4. Draft in plain language first.
5. Add only the amount of Frontline voice the channel permits.
6. Run the mechanical voice checks.
7. Have a human read it aloud before publication.

Voice is the last pass over facts, not a substitute for them.

### Enforcement model

The guide is policy. Publication should require evidence that the policy was followed.

1. **Frontmatter contract:** New editorial and archive content declares `textKind`, `source`, and `voiceLayer`. Builds fail when required provenance is missing.
2. **Mechanical lint:** A repository script rejects em dashes, en dashes, banned launch phrases, unlabeled archive material, and common synthetic constructions in newly authored public copy.
3. **Protected historical paths:** Verbatim game and archive directories are explicit lint exemptions. Exemption follows provenance, not convenience.
4. **Changed-lines gate:** CI and the pre-commit hook lint added copy before it reaches the site. Existing debt stays visible without blocking unrelated work.
5. **Human approval:** Original marketing, About, and Blog copy requires a named human reviewer. A model can draft or critique; it cannot approve voice.
6. **Reader test:** The reviewer must identify the concrete fact in each paragraph and the selected voice layer. If either answer is unclear, the paragraph is not ready.

Automated checks can catch punctuation and clichés. They cannot decide whether a joke sounds like Frontline or whether a sentence earned its atmosphere. That remains editorial judgment.

---

## 6. Humor Framework

### What kind of humor appears

- **Consequence humor:** The game is hard, and the text finds this funny. Death, lost equipment, and stupidity are punchlines. "Fleeing from battle will also cost you some experience points, but hey, thats what you get for wimping out." (help FLEE)
- **Deadpan non-sequiturs:** Help QUI: "This command doesn't DO anything, it simply is." Help TWINK: "Twink is a state of mind."
- **Monopoly references:** "Do not pass go, do not collect $200.00" (help STAKE)
- **Meta-humor:** The FAQ answers "I'm broke and hungry" with EAT and CARVE: the system says what everyone's thinking.
- **Developer in-jokes:** Named social entries (Derek, Kate, Terry, Serapis) are Easter eggs in the socials file, hidden from normal gameplay flow. These are Layer 2 content.
- **Understatement:** "A failed save throw... results in immediate, painful death." The tone is casual about catastrophe.
- **Self-awareness:** "and other annoying player on player games" (features.html) names the thing honestly instead of selling it.

### What's off-limits

The source material contains nothing that rises to hate speech. There are no racial, homophobic, or sexist slurs in the help files. Specific flags are in Section 7.

### Handling insider references in new content

- **Preserve** named developer references in social commands and hidden help entries (Derek, Kate, Terry, Serapis, etc.): they're community history
- **Never reference** them in public-facing marketing, the website, or README
- New Easter eggs should follow the same pattern: hidden, discoverable, not required for gameplay
- **Never threaten players with admin action in new content**: see the Hostility Transfer Rule

---

## 7. Content Sensitivities

Items flagged from the actual source material, with context from the Gemini review:

### Direct text flags

| Instance | Source | Text | Verdict |
|---|---|---|---|
| "BFT" definition | `socials.hlp` | "'Bout Fucking Time" | **[Preserve with context]**: In-game social Easter egg, never shown in public docs |
| "PAW" definition | `socials.hlp` | "Pansy-ass whiner; also known as 'Delete me because I'm a fool'" | **[Preserve with context]**: Same as BFT |
| "Shit, he's beyond help" | `socials.hlp` (help DEREK) | Developer in-joke | **[Preserve with context]**: Social file Easter egg |
| "MmmmmmmmMMMMMMMMMmmmmm ain't she fine?!?!" | `socials.hlp` (help TERRY/KATE) | Refers to real person | **[Cut or rewrite]**: References a real person's appearance. If Kate is a real person who didn't consent to this, cut it. If it's a character, preserve. |
| "slave girls" in castle staff | `info.hlp` (help CASTLE) | "Butlers, maids, slave girls, guards etc, are 1k per level" | **[Rewrite]**: Change to "servants" or "staff." This is a purchasing menu in a help file; "slave girls" as a buyable NPC type is the most straightforwardly problematic text. |
| "groinrip" skill | `commands.hlp`, `info.hlp` | "The victim can be male only, of course." | **[Preserve]**: It's a combat skill in a violent game. The gender restriction reads as period-appropriate combat logic, not malice. Flag for awareness. |
| "oriental" for ninja/races | `info.hlp`, `constants.c` | "dangerous oriental mercenaries"; "oriental society of mercenaries"; "ancient Oriental play-acting" | **[Rewrite for new content]**: "Oriental" is dated. In new writing, use "eastern" or the specific fictional culture name. Historic in-game text can remain as-is in the Go port for authenticity. |
| ".AFK means away from keyboard. Use this command when you're going afk to get a beer, smoke a cigarette, etc." | `info.hlp` | Normalizing alcohol/smoking | **[Preserve]**: 1999 MUD culture. Harmless period detail. |
| "mold shit a pile of shit" | `wizhelp.hlp` | Wizard building command example | **[Preserve in wizhelp]**: Wizard docs are internal. The irreverence is the point. Never expose this publicly. |

### Broader cultural tensions

**Admin authority → World authority:** The shift from punitive admins to a hostile world is the central modernization tension. In 1999, admins were petty gods. In 2026, the game world should be the threat: not the developers running it. All threatening admin language in new content transfers to the world. See the Hostility Transfer Rule (Section 3).

**"Boys Club" elements:** The Terry/Kate entry ("ain't she fine?!?!"), the broader male gaze of the social commands, and the "groinrip" skill all reflect a late-90s male-dominated MUD culture. The game should acknowledge this as period context, not emulate it in new writing. New social commands and Easter eggs should not reproduce this dynamic.

**Insults-as-community culture:** In 1999, calling a player a "Pansy-ass whiner" was standard internet banter. In 2026, this reads as exclusionary or toxic. The solution: keep it in the historical Layer 2 content (social commands, wizhelp), where it's discoverable by choice. Never let it bleed into Discord, GitHub, or the website.

**General note:** The source material is overwhelmingly clean for its era. The issues above are narrow and fixable without changing the voice. The worst item ("slave girls" in the castle pricing) is a one-word fix.

---

## 8. Channel Tuning

### Website

The website speaks to people who may have never seen a MUD. Frontline's actual site writing shows how this works in practice:
- The background page uses the **worldbuilder register** (Layer 3): literary, sweeping, second-person. Address the reader as "adventurer." Use the mythic voice.
- The features page uses the **admin register** (Layer 3): casual, confident, self-aware. Lead with the humble brag ("too many unique features to list here") and then actually list them.
- The FAQ uses a **conversational tone**: "Booo... I died." is the header for the death section. This is the one place where warmth is appropriate.
- News posts are **personal and casual**: Frontline as a human admin who finds this funny, not a brand voice. Parenthetical asides ("(finally!)"), emoticons (" :)"), and repeated words ("please please please") are all in play here.
- Explain what a MUD is in one sentence before diving in
- Translate "mob" → "monster/enemy" on first use
- Drop ALL CAPS cross-references; use links instead
- Don't be afraid of the word "neat": Frontline uses it unironically
- Self-deprecating humor is appropriate here

**Applicable pillars:** Spreadsheet Fantasy, Terminal Grime
**Layer:** Layer 3 (The Mythic Admin)

### GitHub

The README speaks to developers evaluating the codebase:
- Lead with what the project is (Go port of a CircleMUD derivative)
- Technical clarity first, personality second
- Use the wizard-doc tone: competent, direct, no patience for ambiguity
- Include "magick" spelling only when quoting game text; use "magic" in technical prose
- Reference the DikuMUD/CircleMUD lineage directly

**Applicable pillars:** Terminal Grime (reference the terminal/client context)
**Layer:** Layer 1 (The Engine)

### In-game (Go port)

The Go port preserves original C messages exactly. For new content:
- Match the existing SendMessage style: short, direct, second-person
- New help entries should follow the established format (heading, syntax, body, examples, "See also")
- New room descriptions should use the same second-person immersive style: describe what the player experiences, not what the room "is"
- New spell/skill descriptions should include both mechanics and flavor in the same entry
- Do not add exclamation points to new messages that wouldn't have had them in the original
- Let the real world bleed in where appropriate (Terminal Grime): reference keyboards, screens, connections
- Never threaten players with admin action (Hostility Transfer Rule)

**Applicable pillars:** Hostile Helpfulness, Spreadsheet Fantasy, Terminal Grime
**Layer:** Layer 2 + Layer 3 blend

### Social media

The voice compresses for short-form:
- Lead with gallows humor: the platform is the punchline
- Use room-description style imagery: sensory, second-person
- Drop "See also" cross-references entirely
- Tagline territory: "The only safe prospect is death in the end"
- Developer in-jokes (BFT, PAW) do NOT go here
- Keep it under 280 characters per thought
- Never threaten the player: threaten the *world*

**Applicable pillars:** Hostile Helpfulness (compressed)
**Layer:** Layer 3 (The Mythic Admin, compressed)

---

## 9. Quick Reference Card

### The framework at a glance

| Layer | Voice | Who | Modern channel |
|---|---|---|---|
| The Engine | Dry, UNIX-like, functional | CircleMUD / Jeremy Elson | GitHub docs, README, code |
| The Edgelord DM | Hostile, cliquey, unfiltered | Serapis / Derek Karnes | In-game socials, Easter eggs (preserved, not new) |
| The Mythic Admin | Mythic prose + warm community | Frontline / R.E. Paret | Website, Discord, marketing, FAQ |

### The four pillars

1. **Hostile Helpfulness**: Answer the question. Make the situation the butt of the joke. Stop.
2. **Spreadsheet Fantasy**: Mythic prose, then raw numbers. Don't apologize for the math.
3. **Terminal Grime**: Keyboards, beer, telnet. The software is part of the world.
4. **Cliquey and Unfiltered**: Insider refs, insults-as-affection. **In-game only.** Never public-facing.

### The hostility transfer

1999: "We will delete you." → 2026: "The game will kill you."
The world is hostile. The developers are not.

### 5 things to remember when writing for Dark Pawns

1. **Pick your layer first.** GitHub → Engine. Website → Mythic Admin. In-game → blend. Social commands → Edgelord DM (historical only).
2. **"magick" with a k.** Always. It's the energy field. "Magic" is only for class names or non-lore technical prose.
3. **Answer the question, then add one dry aside.** Not two. Not a paragraph. One parenthetical or trailing thought.
4. **The player is "you."** Second person, direct address, no deference. They're in a dangerous world and they chose to be here.
5. **Consequences are punchlines, not tragedies.** But the punchline is "the game killed you," not "we deleted your account."

### 5 things to never do

1. **Never threaten players in new public-facing content.** The hostility transfers to the world, not the developers. See Section 3.
2. **Never break the fourth wall in mortal-facing text.** No "this feature was added in version 2.2" in help entries. The world is the world. Save meta-talk for wizard docs and the credits file.
3. **Never sanitize the danger.** If a mechanic is brutal, describe it brutally. "You will be dead forever" is better than "permanent character deletion may occur."
4. **Never use em dashes or en dashes in new output copy.** Restructure the sentence. Verbatim historical artifacts remain unchanged.
5. **Never pass a summary off as an archive.** Label the text kind and cite the source, or stay silent.

### Example rewrites

**Wrong → Right**

❌ Welcome to the exciting Thief class! Thieves are amazing characters who use stealth and cunning to overcome their enemies.

✅ Thieves are known to have very special qualities, that no other class offers. Their specialty tends to be in the darker, sneakier art. Because of their sneaky nature, the thief class is barred from using shields of any sort.

❌ The FLEE command allows you to escape from combat! Please note that you will lose some experience points when you flee.

✅ If you are in a fight and things are beginning to get a little uncomfortable (maybe you are dying), just type 'flee', and presto! You are out of harms way. That is, IF there is a suitable exit nearby, and IF that exit doesn't simply bring you from the ashes to the fire... but then, who wants to live forever?

❌ To use the LOCK command, type LOCK followed by the door name and optionally a direction. Example: LOCK DOOR SOUTH.

✅ To open, close, lock, and unlock doors, of course.
```
> open portal
> lock door
> unlock door south
> close gate
```

❌ Death is a serious consequence in Dark Pawns. When your character dies, you will lose all your equipment and some experience points.

✅ When you die, you come back into the game in the temple. All equipment and gold remains in your corpse at the place you died. Every time you die, there is a chance you will lose 1 point of CONSTITUTION. If your constitution drops below 1, you will be dead forever.

❌ The Dream system adds immersive roleplay elements to Dark Pawns. Characters who sleep may experience prophetic dreams, especially in the Wyldlands zone.

✅ When you sleep on Dark Pawns, there is a good chance that you might DREAM. Most people in the world consider DREAMS to be harmless, but many mystics and oracles consider dreams to be very important, even prophetic. DREAMS that occur in the Wyldlands are considered especially meaningful. It is rumored that in the magick-rich Wyldlands DREAMS have the power to physically effect the dreamer, in essence, the DREAMS become real.

---

*Document version 2.1. Added the Anti-LLM Prose Standard, dash ban, archive labels, and enforcement model. August 2026.*
*Version 1.1: enriched with website content (Frontline), April 2026*
*Version 1.0: generated from source material analysis, April 2026*
