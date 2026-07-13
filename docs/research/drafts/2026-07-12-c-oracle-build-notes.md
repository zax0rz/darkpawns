# C Oracle Build Notes — Dark Pawns v2.3 on macOS Apple Silicon

**Date:** 2026-07-12
**Host:** macOS (Apple Silicon), Xcode Command Line Tools / clang
**C repo:** `https://github.com/rparet/darkpawns.git`
**C repo path:** `/Users/zach/.openclaw/workspace/darkpawns-c-oracle`
**Go repo path:** `/Users/zach/.openclaw/workspace/darkpawns_repo`
**Branch:** `docs/c-oracle-build-notes`

---

## TL;DR

The original C Dark Pawns server builds and boots natively on Apple Silicon with only
compiler-warning suppression flags. No source changes were required. A test character was
created, entered the game, looked around, and quit cleanly.

---

## 1. Clone

```bash
git clone https://github.com/rparet/darkpawns.git \
  /Users/zach/.openclaw/workspace/darkpawns-c-oracle
```

The C repo lives **outside** the Go repo, as a sibling directory.

---

## 2. Configure & build

### Dependencies

Only the Xcode Command Line Tools (clang) are required. `autoconf`/`automake` were **not**
needed because `configure` is pre-generated in the upstream repo.

### Configure

```bash
cd /Users/zach/.openclaw/workspace/darkpawns-c-oracle
./configure CFLAGS="-g -O2 -Wno-implicit-function-declaration \
  -Wno-implicit-int -Wno-int-conversion -Wno-return-type"
```

`configure` output highlights:

- `checking for gcc... gcc` — clang is masquerading as gcc on macOS; this is normal.
- `checking for crypt in -lcrypt... no` — expected; macOS has no separate `libcrypt`.
  `configure` warns that passwords will be stored in plaintext. This is acceptable for a
  local oracle.
- `checking for deflate in -lz... yes` — zlib ships with macOS.
- `checking for sqrt in -lm... yes` — libSystem includes libm.
- `checking arpa/telnet.h usability... yes` — present on macOS.

### Make

```bash
make -j4
```

Result: **build succeeded** with warnings only. The server binary is produced at
`bin/circle`.

### Compiler warnings observed

All were non-fatal warnings that did not stop the build:

- `-Wstrict-prototypes` — several old-style function declarations (e.g.
  `void extract_pending_chars();`, `float uniform();`, `uint32_t prng_next();`).
- `-Wunused-but-set-variable` — a few variables assigned but not read.
- `-Wmisleading-indentation` — in `weather.c` around the weather state machine.

Per the brief, these were left untouched because they are behavior-preserving
declarations/style issues, not logic changes.

---

## 3. Changes made to the C source tree

**None.** The tree built clean with only CFLAGS adjustments.

```bash
cd /Users/zach/.openclaw/workspace/darkpawns-c-oracle
git status --short
# Output: (empty — no tracked-file modifications)
```

The only untracked file produced during testing was `boot.log`, which was removed
after capture.

---

## 4. Boot

### Important: run from the directory containing `lib/`

The server resolves its data files relative to the current working directory. Running
`bin/circle` from outside the repo root causes it to fail to open `lib/text/*`,
`lib/misc/messages`, etc.

### Chosen port

- Default port in `start_dp.sh` is 4350.
- Port 4000 was already in use on the build host (`bind: Address already in use`).
- Port **4351** was free and used for this test.

### Start command

```bash
cd /Users/zach/.openclaw/workspace/darkpawns-c-oracle
./bin/circle 4351
```

For background logging:

```bash
cd /Users/zach/.openclaw/workspace/darkpawns-c-oracle
(./bin/circle 4351 > boot.log 2>&1) &
```

### Stop command

Find the `circle` process listening on 4351 and kill it:

```bash
lsof -P -n -i :4351
kill <pid>
```

### Boot-time runtime files

No empty files or directories had to be created manually. The server created the
following runtime files on first boot:

- `lib/etc/players` — player index (initially empty, then populated by test characters).
- `lib/etc/clans` — empty clan file.
- `lib/etc/badsites` — empty banned-sites file.
- `lib/etc/dns` — empty DNS cache.
- `lib/etc/plrmail` — empty player mail file.
- `lib/misc/messages` — loaded from source data; regenerated if missing.
- `lib/text/news.old` — generated on rotation.

### Non-fatal boot warnings

The following messages appear during boot but do not prevent the server from reaching the
game loop:

- `SYSERR: File etc/date_record not found, mud date will be reset to default!` —
  expected on first boot; date resets to default gametime.
- `Unknown social 'hiss' in social file`, `kneel`, `mutter` — social file references
  socials not present in the command table; harmless.
- `SYSERR: Char is already equipped: Big Mama McGarnicle, a sharp bronze knife` —
  a zone reset equips an already-equipped mob; harmless.
- `Error reading board: No such file or directory` (repeated) — board files are missing;
  the server continues and boards simply have no messages.
- `House control file does not exist.` — no houses defined; server creates one on demand.
- `Clan file does not exist. Will create a new one` — server creates an empty clan file.

---

## 5. Proof-of-life telnet transcript

Tool used: `expect` driving `nc 127.0.0.1 4351`.

Commands issued by the automated client are shown inline after each prompt. ANSI color
was disabled.

```text
By what name do you wish to be known? coracle
Please remember to choose an appropriate fantasy-oriented name.
Did I get that right, Coracle (Y/N)? Y
New character.
Give me a password for Coracle: testpass

Please retype password: testpass

Do you want ANSI color (Y/N)? N
What is your sex (M/F)? m

Choose a race:
  [H]uman        [E]lven       [D]warven      [K]enderkin
  [M]inotaur     [R]akshasan   [S]sauran
  [?]Help on races in general
  [?<race abbreviation>] Help on a specific race (i.e ?D for help on dwarves)

Race: h

Select a class:
  [C]leric     - Healers and warriors of the gods
  [T]hief      - Stealthy, quick-fingered, lock-picking back-stabbers
  [W]arrior    - Fierce, battle-trained fighters
  [M]agic-user - Spell-casters trained in the art of magick
  [N]inja      - Stealthy, magick-endowed warriors from the orient
  Ps[i]onic    - Fighters endowed with the powers of the mind
Class: w

Choose your home town:
  [K]ir Drax'in  - The Main City. New players should choose this.
  Kir-[O]shi     - The Port City.
  [A]laozar      - The Holy City.

Select: k

Your ability scores:
  Str: very good     Dex: decent        Int: poor
  Wis: average       Con: decent        Cha: poor

Press 'Y' to keep these stats, and 'N' to reroll: Y

Welcome to Dark Pawns: darkpawns.com 4300

Dark Pawns webpage: &chttp://www.darkpawns.com&n

**** ATTENTION:  This is now DP Classic
**** Dark Pawns 3.0 beta is live at darkpawns.com 4355
**** So go there immediately, if not sooner.


*** PRESS RETURN:


Welcome to Dark Pawns!

0) Exit from Dark Pawns.

1) Enter the game.

2) Enter description.

3) Read the background story.

4) Change password.

5) Delete this character.

   Make your choice: 1

Welcome to Dark Pawns! May your visit here be... Interesting.

A Burning Hut
You stand inside a burning hut; what you considered home until a few moments
ago when the orcs entered the village. You slept soundly until it was too
late; only the horrified screams of your parents dying woke you.
Now you stand bravely, though petrified with fear in your mind, as you watch
the hairy, tusked orcs advance toward you, swords in hand.
 [ Exits: None! ]

22H 100M 83V > look
   Suddenly the hairs on the back of your neck stand up as if lightning had
struck nearby. A keen wailing fills the air, and an ethereal image appears
before you.
   'Coracle, now is not your time to die,' speaks the figure.
   'Prove your worth and I may well grant you eternal life.'
   'Trust no one, for all here are but dark pawns above which you must
struggle to prove yourself.  All here strive to be a king... at any cost.'
   The figure glows a moment, then disappears, but his voice remains.
   'Your life begins now...' it says, then fades -- just as the world around
you does the same.

Temple Infirmary
   This is a large room where the sick and the dying are cared for by the
nuns and priestesses of the Church.  There are about fifty straw beds arranged
in four rows here, and almost all of them are full.  Most of the patients are
either common citizens or foreigners; the military has their own infirmary
as do the large temples.  The wealthier citizens can afford an in-home visit
by a Church healer.  The entire room smells of medicinal soap.  There is
a single exit through a marble archway to the north.
[ Exits: north ]
Buildbot the Warrior (linkless) is standing here.

22H 100M 83V > quit
Type REALLYQUIT to quit the game and lose your eq.
Return to the temple and QUIT to leave the game and keep your equipment.
You can type RECALL to return to your temple.

22H 100M 83V > REALLYQUIT
Goodbye, friend.. Come back soon!

22H 100M 83V >
```

After `REALLYQUIT`, the session returned to the main menu; the character was saved and
the server remained running.

### Success criteria check

| # | Criterion | Result |
|---|-----------|--------|
| 1 | `make` completes; server binary exists | ✅ `bin/circle` |
| 2 | Server boots and stays running on chosen port | ✅ port 4351 |
| 3 | `telnet localhost <port>` reaches "By what name..." prompt | ✅ via `nc 127.0.0.1 4351` |
| 4 | Create test char, reach playable prompt, look, quit cleanly | ✅ character "Coracle" |

---

## 6. What was NOT applied

- **No `DP_SEED` patch** — per the brief, the `comm.c:263` RNG determinism patch was
  left untouched; it is owned by Claude as a separate deliberate step.
- **No source code modifications** — no game behavior, formulas, messages, tables, or
  branches were altered.
- **No C source or build artifacts committed to the Go repo** — this PR is docs-only.

---

## 7. Reproducible one-liner

From a fresh clone on macOS Apple Silicon:

```bash
cd /Users/zach/.openclaw/workspace/darkpawns-c-oracle
./configure CFLAGS="-g -O2 -Wno-implicit-function-declaration \
  -Wno-implicit-int -Wno-int-conversion -Wno-return-type"
make -j4
./bin/circle 4351
```

Then connect with:

```bash
nc 127.0.0.1 4351
```
