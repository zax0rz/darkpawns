#!/usr/bin/env python3
"""DP-1511 hollow-green audit.

A hollow green is an oracle scenario that passes because both engines refused
the same setup step, so the event its manifest row claims never happened
(three known: DP-1510 cast-create-water-depth, group-recall-sleeping-member,
summon-backfire-output-depth). The harness's warmup-refusal gate
(internal/oraclediff/setuprefusal.go, setupRefusalLines) matches whole lines
exactly, so a refusal containing a name -- "You don't see a chalice here." --
slips through. DP-1511 is that gap; fixing it is Zach's, not this script's.
This script finds every scenario the gap may have hidden.

Audit universe: the depth manifests and scenario files **as they existed at
the census commit** -- the snapshot the rehearsal census actually proved. Rows
repaired on main after the census (the DP-1510 repair among them) are judged
in their census-era form, which is the form whose green this census produced.
Everything is read from one `git archive <census-ref>` snapshot; the census
dumps are the seed-1 normalized C blocks the run retained.

Screening (per manifest row whose status starts with oracle-green, for each
scenario named in its proof):

  warmup-refusal  A C actor probe block contains a setup refusal while the
                  scenario has [warmup] steps. The harness discards warmup
                  output (oraclediff.RunWarmup drops the blocks), so a refusal
                  surfacing in a probe block is the retained fingerprint of a
                  failed setup -- the DP-1510 shape.
  claimed-absent  No string literal printed by any output call on the row's
                  c_site lines appears in any C probe block: the bytes the row
                  cites never surfaced. Literals are extracted from the src/
                  snapshot (R5g), format codes stripped (%s, $n, $p, ...),
                  and the longest fixed fragments matched.
  probe-refusal   Every C probe block (actor and peers) is empty or a refusal.

Expected refusals are not suspects: rows whose case_id, notes or scenario
header name the refusal (or quote the refusal line itself) get note=intended
and stay out of the suspect count. Screening is deliberately over-inclusive;
confirmed suspects are repaired by hand in a later step.

Unauditable rows are recorded in unauditable.tsv, never guessed (the brief's
two categories):
  - proof seeds that do not include seed 1 (the retained dump is seed 1);
  - no census dump for the scenario.

Refusal fragments were extended from the brief's starter list by grepping
src/act.*.c and src/spell_parser.c in the authoring session (R5g); each
extension cites its C exemplar inline.

Usage:
  python3 -I audit_hollow_greens.py \
      --repo /home/zach/wt/glm-dp1511 \
      --census ~/Archives/darkpawns/oracle-runs/2026-11-07/claude-rehearsal-1107

Stdlib only. Writes suspects.tsv and unauditable.tsv beside this file.
"""

import argparse
import csv
import io
import re
import subprocess
import sys
import tarfile
from pathlib import Path, PurePosixPath

# ---------------------------------------------------------------------------
# Refusal fragments. Every fragment is a fixed substring of a real C refusal
# string, grepped from src/ in the authoring session (R5g).
# ---------------------------------------------------------------------------
REFUSAL_FRAGMENTS = [
    # Brief starter list.
    "You don't see",                 # act.item.c do_get: "You don't see %s %s here."
    "You don't seem to have",        # act.item.c: "You don't seem to have %s %s."
    "You do not have",               # brief list
    "Huh?!?",                        # interpreter.c unknown command
    "Cannot find the target",        # spell_parser.c: "Cannot find the target of your spell!"
    "Nobody here by that name",
    "They don't seem to be here",
    "Your arms are already full",
    "You can't",                     # broad C refusal class
    "doesn't seem to be anything here",
    # Extensions grepped from src/ this session.
    "You do not see",                # "You do not see that here."
    "You don't seem to be",          # "You don't seem to be carrying anything." class
    "You do not seem",               # "You do not seem to have %s %s."
    "I see no",                      # "I see no %s there."
    "No such",                       # "No such player/object/mobile around."
    "There is no such",              # "There is no such player/creature."
    "That person isn't here",
    "Nothing seems to happen",       # spell_parser.c no-effect refusal
    "You cannot",                    # broad C refusal class
]

# Output calls whose quoted strings are player-facing text (grepped usage
# counts in src/act.*.c, spell_parser.c, magic.c, spells.c this session).
OUTPUT_CALL = re.compile(
    r"\b(act|stc|send_to_char|send_to_all|send_to_room|send_to_zone|page_string)\s*\("
)
# Log-only sinks: literals fed to mudlog/log are immortal wizline bytes, not
# the player-facing transcript, and must not anchor a claim.
LOG_CALL = re.compile(r"\b(mudlog|syslog|log)\s*\(")
BUFFER_CALL = re.compile(r"\b(sprintf|strcpy|strcat|strcpy_into)\s*\(\s*([A-Za-z_0-9\.>\-]+)\s*,")
# The mag_affects idiom (src/magic.c:908-909): output pointers assigned in the
# switch and sent by common code after it -- the assignment line is the
# literal's only appearance in the cited range.
ASSIGN_OUTPUT = re.compile(r"\bto_(?:vict|self|room|char)\s*=(?!=)")
STRING_LIT = re.compile(r'"((?:[^"\\]|\\.)*)"')
FORMAT_CODE = re.compile(r"\$[a-zA-Z]|%[-+ #0-9.]*[a-zA-Z]")

SITE_TOKEN = re.compile(r"^(?:(?:src/)?)([A-Za-z0-9_.\-]+\.[ch]):([0-9,\-]+)\s*$")
SOCIALS_TOKEN = re.compile(r"^lib/misc/socials(?::([0-9,\-]+))?\s*$")
MESSAGES_TOKEN = re.compile(r"^lib/misc/messages:([0-9,\-]+)\s*$")
SOCIALS_HEADER = re.compile(r"^([a-z]+) +\d+ +\d+\s*$")
# lib/misc/messages line classes that are not message text (load_messages:
# section headers `* ...`, the `M` marker, the attacktype number, `#` end).
MESSAGES_NONTEXT = re.compile(r"^(\*.*|M|#{1,3}|\d+)\s*$")
MAX_SITE_LINES = 200
# Presence window around a cited site: a row is not "claimed-absent" when the
# C neighborhood it cites demonstrably executed and printed (some rows cite a
# gate they pass through, or stop a line or two short of the claimed branch;
# the claimed bytes then sit just outside the citation). The window widens the
# *presence* check only; evidence is still attributed to the cited lines.
WINDOW_BEFORE = 6
WINDOW_AFTER = 12

MIN_FRAGMENT_LEN = 5

# Words ignored when deciding whether a case_id/note names a refusal. Kept
# minimal: distinctive refusal words ("target", "arms", "nobody") must survive,
# because case ids like cast.missing-target name the refusal through them.
STOPWORDS = {
    "the", "and", "for", "you", "your", "yours", "that", "this", "these",
    "those", "have", "has", "had", "here", "there", "them", "they", "with",
    "seem", "seems", "are", "not", "any", "can", "cannot", "cant", "don",
    "doesn", "didn", "won", "will", "into", "onto", "from", "was", "were",
    "when", "while", "what", "which", "who", "whom", "whose", "how", "why",
    "but", "all", "get", "got", "its", "his", "her", "him", "she", "such",
    "like", "thing", "person",
}

SIGNAL_RANK = {"claimed-absent": 0, "warmup-refusal": 1, "probe-refusal": 2}

MANIFEST_DIR = "docs/fidelity/depth"
SCENARIO_DIR = "cmd/dp-oracle-diff/scenarios"


def refusal_lines(text):
    """Return the lines of a block that match a refusal fragment."""
    hits = []
    for line in text.split("\n"):
        line = line.rstrip("\r")
        if not line:
            continue
        for frag in REFUSAL_FRAGMENTS:
            if frag in line:
                hits.append(line)
                break
    return hits


def fragments_of(literal):
    """Fixed fragments of a C string literal with format codes stripped and
    escaped newlines treated as fragment separators (normalized blocks keep
    multi-line literals on separate lines)."""
    frags = []
    for part in FORMAT_CODE.split(literal):
        for sub in re.split(r"\\r\\n|\\n|\\r", part):
            sub = sub.replace('\\"', '"').replace("\\\\", "\\")
            sub = sub.strip()
            if len(sub) >= MIN_FRAGMENT_LEN:
                frags.append(sub)
    return frags


def parse_site_token(token):
    m = SITE_TOKEN.match(token.strip())
    if not m:
        return None
    file, lines_spec = m.group(1), m.group(2)
    lines = []
    for piece in lines_spec.split(","):
        if "-" in piece:
            a, b = piece.split("-")
            a, b = int(a), int(b)
        else:
            a = b = int(piece)
        if a > b:
            a, b = b, a
        lines.extend(range(a, min(b, a + MAX_SITE_LINES - 1) + 1))
    return file, lines


class Snapshot:
    """One git-archive snapshot of the census commit."""

    def __init__(self, repo, ref):
        r = subprocess.run(
            ["git", "-C", str(repo), "archive", ref,
             MANIFEST_DIR, SCENARIO_DIR, "src", "lib/misc/socials",
             "lib/misc/messages"],
            capture_output=True, check=True)
        self.files = {}
        with tarfile.open(fileobj=io.BytesIO(r.stdout)) as tf:
            for m in tf.getmembers():
                if m.isfile():
                    self.files[m.name] = tf.extractfile(m).read()

    def manifest_paths(self):
        return sorted(n for n in self.files if n.startswith(MANIFEST_DIR + "/")
                      and n.endswith(".tsv"))

    def scenario_text(self, name):
        data = self.files.get(f"{SCENARIO_DIR}/{name}.txt")
        return data.decode("utf-8", errors="replace") if data is not None else None

    def src_file_lines(self, file):
        """Lines (1-based) of a src/ file from the snapshot."""
        data = self.files.get(f"src/{file}") or self.files.get(file)
        if data is None:
            return []
        return data.decode("utf-8", errors="replace").split("\n")

    def has_scenario(self, name):
        return f"{SCENARIO_DIR}/{name}.txt" in self.files


def print_literals_at_site(snapshot, file, lines, windowed=False):
    """String literals printed by the cited lines: literals from every line of
    the site range when any line in the range is an output call (buffer-built
    output via sprintf+send_to_char is the norm in this C tree). With
    windowed=True the range is widened by WINDOW_BEFORE/WINDOW_AFTER lines for
    the presence check. Log-fed buffers (sprintf straight into mudlog/log) are
    skipped: those bytes never reach the player transcript."""
    if windowed:
        start = max(1, min(lines) - WINDOW_BEFORE)
        end = max(lines) + WINDOW_AFTER
        lines = range(start, end)
    src_lines = snapshot.src_file_lines(file)
    selected = [src_lines[n - 1] for n in lines if 1 <= n <= len(src_lines)]
    if not selected or not any(OUTPUT_CALL.search(line) or ASSIGN_OUTPUT.search(line)
                               for line in selected):
        return []
    log_buffers = set()
    for line in selected:
        if LOG_CALL.search(line) and not OUTPUT_CALL.search(line):
            m = re.search(r"\(\s*([A-Za-z_0-9]+)\s*,", line)
            if m:
                log_buffers.add(m.group(1))
    literals = []
    for idx, line in enumerate(selected):
        if LOG_CALL.search(line) and not OUTPUT_CALL.search(line):
            continue
        m = BUFFER_CALL.search(line)
        if m and m.group(2) in log_buffers and not OUTPUT_CALL.search(line):
            continue
        for lit in STRING_LIT.finditer(line):
            literals.append((file, lines[0], lit.group(1)))
    return literals


def data_table_lines(snapshot, token):
    """Message fragments from a lib/misc data-table citation. Both tables are
    claim sources for rows whose bytes live in data, not in src/ literals; a
    cited line yields its whole entry/M-block, and every message line becomes
    a fragment."""
    m = MESSAGES_TOKEN.match(token.strip())
    if m:
        blocks = messages_block_lines(snapshot)
        data = snapshot.files.get("lib/misc/messages")
        if data is None:
            return []
        src_lines = data.decode("utf-8", errors="replace").split("\n")
        out, seen = [], set()

        def add(line_no, text):
            if line_no not in seen:
                seen.add(line_no)
                out.append((f"lib/misc/messages:{line_no}", line_no, text))

        # Plain range extraction (cited lines that are message text) union
        # block expansion (rows citing the M marker or an off-by-one line).
        for piece in m.group(1).split(","):
            if "-" in piece:
                a, b = piece.split("-")
                ns = range(int(a), int(b) + 1)
            else:
                ns = [int(piece)]
            for nn in ns:
                if 1 <= nn <= len(src_lines):
                    t = src_lines[nn - 1].strip()
                    if t and not MESSAGES_NONTEXT.match(t):
                        add(nn, t)
                for line_no, text in blocks.get(nn, []):
                    add(line_no, text)
        return out
    m = SOCIALS_TOKEN.match(token.strip())
    if m:
        entries = socials_entry_lines(snapshot)
        out, seen = [], set()
        if m.group(1):
            for piece in m.group(1).split(","):
                ns = [int(piece)] if "-" not in piece else list(
                    range(int(piece.split("-")[0]), int(piece.split("-")[1]) + 1))
                for nn in ns:
                    for line_no, text in entries.get(nn, []):
                        if line_no not in seen:
                            seen.add(line_no)
                            out.append((f"lib/misc/socials:{line_no}", line_no, text))
        return out
    return []


def row_fragments(snapshot, c_site, windowed=False):
    """All fixed fragments printed by the row's cited C sites (and, for data
    tables, the cited table lines)."""
    frags = []
    for token in (c_site or "").split(";"):
        site = parse_site_token(token)
        if site is None:
            for file, line_no, literal in data_table_lines(snapshot, token):
                for frag in fragments_of(literal):
                    frags.append((file, line_no, literal, frag))
            continue
        for file, line_no, literal in print_literals_at_site(snapshot, *site,
                                                             windowed=windowed):
            for frag in fragments_of(literal):
                frags.append((file, line_no, literal, frag))
    return frags


def parse_scenario(text):
    """Return (warmup, probe, header_comments) from a scenario file."""
    warmup, probe, comments = [], [], []
    section = None
    in_header = True
    for raw in text.split("\n"):
        line = raw.strip()
        if line.startswith("[") and line.endswith("]"):
            in_header = False
            name = line[1:-1].lower()
            if name == "warmup":
                section = warmup
            elif name == "probe":
                section = probe
            else:
                section = None
            continue
        if not line or line.startswith("#"):
            if in_header and line.startswith("#"):
                comments.append(line)
            continue
        if section is not None:
            section.append(line)
    return warmup, probe, comments


def parse_dump(path):
    """Return [(command, role, text)] from a census dump file.

    Block headers are `--- [command]` for peer-less scenarios and
    `--- [command [audience]]` otherwise (main.go:929-931); the primary
    audience is `actor`. Everything else is a named peer.
    """
    blocks = []
    command, role, lines = None, None, []
    started = False
    for raw in path.read_text(errors="replace").split("\n"):
        if raw.startswith("--- [") and raw.endswith("]"):
            if started:
                blocks.append((command, role, "\n".join(lines)))
            started = True
            inner = raw[len("--- ["):-1]
            role = None
            m = re.search(r"\s*\[([A-Za-z0-9_\-]+)\]$", inner)
            if m:
                role = m.group(1)
                inner = inner[: m.start()]
            command, lines = inner, []
            continue
        if started:
            lines.append(raw)
    if started:
        blocks.append((command, role, "\n".join(lines)))
    return blocks


def load_socials(snapshot):
    """{name: [(line_no, message)]} from lib/misc/socials. Header lines are
    `<name> <hide> <min_victim_position>`; message lines follow until the next
    header, with `#` marking an absent message (act.social.c:260-272)."""
    data = snapshot.files.get("lib/misc/socials")
    if data is None:
        return {}
    table = {}
    name = None
    for i, line in enumerate(data.decode("utf-8", errors="replace").split("\n"), 1):
        line = line.strip()
        if not line:
            continue
        h = SOCIALS_HEADER.match(line)
        if h:
            name = h.group(1)
            table[name] = []
            continue
        if name and line != "#":
            table[name].append((i, line))
    return table


_MESSAGES_BLOCKS = {}
_SOCIALS_ENTRIES = {}


def messages_block_lines(snapshot):
    """{line_no: [(line_no, text)]} mapping every line of each lib/misc/messages
    M-block (its `* section` header, the `M` marker, a_type, and trailing `#`/
    count lines) to the block's message lines. Rows cite the entry by its
    marker line, so one cited line must yield the whole block."""
    if _MESSAGES_BLOCKS:
        return _MESSAGES_BLOCKS
    data = snapshot.files.get("lib/misc/messages")
    if data is None:
        return {}
    src = data.decode("utf-8", errors="replace").split("\n")
    n = len(src)
    i = 0
    while i < n:
        if src[i].strip() == "M":
            start = i
            if start > 0 and src[start - 1].strip().startswith("*"):
                start -= 1
            j = i + 1
            while j < n and src[j].strip() != "M" and not src[j].strip().startswith("*"):
                j += 1
            msgs = []
            for k in range(i + 1, j):
                t = src[k].strip()
                if t and not MESSAGES_NONTEXT.match(t):
                    msgs.append((k + 1, t))
            for k in range(start, j):
                _MESSAGES_BLOCKS[k + 1] = msgs
            i = j
        else:
            i += 1
    return _MESSAGES_BLOCKS


def socials_entry_lines(snapshot):
    """{line_no: [(line_no, text)]} mapping every line of each lib/misc/socials
    entry (header included) to the entry's message lines."""
    if _SOCIALS_ENTRIES:
        return _SOCIALS_ENTRIES
    data = snapshot.files.get("lib/misc/socials")
    if data is None:
        return {}
    src = data.decode("utf-8", errors="replace").split("\n")
    n = len(src)
    i = 0
    while i < n:
        if SOCIALS_HEADER.match(src[i].strip()):
            j = i + 1
            while j < n and not SOCIALS_HEADER.match(src[j].strip()):
                j += 1
            msgs = [(k + 1, src[k].strip()) for k in range(i + 1, j)
                    if src[k].strip() and src[k].strip() != "#"]
            for k in range(i, j):
                _SOCIALS_ENTRIES[k + 1] = msgs
            i = j
        else:
            i += 1
    return _SOCIALS_ENTRIES


def is_distinctive(frag):
    return len(frag) >= 8 and " " in frag


def claimed_refusal_fragments(corpus):
    """Refusal fragments named (quoted) by a scenario's collective row text."""
    return {frag.lower() for frag in REFUSAL_FRAGMENTS if frag.lower() in corpus}


def is_actor(role):
    return role is None or role == "actor"


def block_is_refusal_or_empty(text):
    if not text.strip():
        return True
    hits = refusal_lines(text)
    nonempty = [l for l in text.split("\n") if l.strip()]
    return bool(nonempty) and len(hits) == len(nonempty)


def significant_words(text):
    return [w for w in re.findall(r"[a-z']{3,}", text.lower())
            if w not in STOPWORDS]


def intended(row, comments, evidence_refusals):
    """Does the row's own text name (or quote) the refusal?"""
    corpus = " ".join([
        (row.get("case_id") or "").replace("-", " ").replace("_", " ").replace(".", " "),
        row.get("notes") or "",
        " ".join(comments),
    ]).lower()
    for refusal in evidence_refusals:
        # Word match on the full refusal line: rows whose claim IS the refusal
        # describe it ("the exact sun-and-sand refusal") without quoting bytes.
        words = significant_words(refusal)
        if words:
            hits = sum(1 for w in words if w in corpus)
            if hits == len(words) or hits >= 3 or (
                    hits >= 2 and hits / len(words) >= 0.5):
                return True
        for frag in REFUSAL_FRAGMENTS:
            if frag.lower() not in refusal.lower():
                continue
            if frag.lower() in corpus:
                return True
            words = significant_words(frag)
            if not words:
                continue
            hits = sum(1 for w in words if w in corpus)
            if hits == len(words) or (
                    hits >= 2 and len(words) >= 2 and hits / len(words) >= 0.5):
                return True
        if len(refusal) >= 12 and refusal.lower() in corpus:
            return True
    return False


def scenario_proofs(proof):
    """[(name, seeds)] for every part of a proof column naming a scenario."""
    out = []
    for part in (proof or "").split(";"):
        part = part.strip()
        if not part:
            continue
        seeds = []
        name = part
        if "@" in part:
            name, seed_spec = part.split("@", 1)
            seeds = [s.strip() for s in seed_spec.split(",") if s.strip()]
        name = name.strip()
        if re.fullmatch(r"[a-z0-9][a-z0-9\-]*", name):
            out.append((name, seeds))
    return out


def main():
    ap = argparse.ArgumentParser(description="DP-1511 hollow-green audit")
    ap.add_argument("--repo", required=True, help="repository root (for git archive)")
    ap.add_argument("--census", required=True, help="rehearsal census run directory")
    ap.add_argument("--census-ref", default="b7fdd6ef58eb48ac407387ab62cdb0ed44a58863",
                    help="commit the census ran (default: rehearsal census HEAD)")
    args = ap.parse_args()

    census = Path(args.census)
    dump_dir = census / "census-dump"
    out_dir = Path(__file__).resolve().parent
    snap = Snapshot(args.repo, args.census_ref)
    socials = load_socials(snap)

    # Prepass: per-scenario row text, so a first probe that deliberately tests
    # a refusal (a sibling row's claim, quoted in its case_id/notes/comments)
    # is not misread as a setup failure.
    scen_cache = {}
    scen_corpus = {}

    def scenario_parts(name):
        if name not in scen_cache:
            scen_cache[name] = parse_scenario(snap.scenario_text(name))
        return scen_cache[name]

    for rel in snap.manifest_paths():
        text = snap.files[rel].decode("utf-8", errors="replace")
        lines = text.split("\n")
        header = lines[0].split("\t")
        idx = {name: i for i, name in enumerate(header)}
        for raw in lines[1:]:
            if not raw.strip():
                continue
            f = raw.split("\t")
            if len(f) < len(header):
                continue
            row = {k: (f[i] if i < len(f) else "") for k, i in idx.items()}
            if not row.get("status", "").startswith("oracle-green"):
                continue
            for scen_name, _seeds in scenario_proofs(row.get("proof", "")):
                if scen_name not in scen_corpus:
                    scen_corpus[scen_name] = ""
                scen_corpus[scen_name] += " " + (
                    row.get("case_id") or "").replace("-", " ").replace(".", " ")
                scen_corpus[scen_name] += " " + (row.get("notes") or "")
                scen_corpus[scen_name] += " " + " ".join(scenario_parts(scen_name)[2])
    scen_claimed = {name: claimed_refusal_fragments(c.lower())
                    for name, c in scen_corpus.items()}

    suspects, unauditable = [], []
    n_rows = n_pairs = 0

    for rel in snap.manifest_paths():
        text = snap.files[rel].decode("utf-8", errors="replace")
        lines = text.split("\n")
        header = lines[0].split("\t")
        idx = {name: i for i, name in enumerate(header)}
        for raw in lines[1:]:
            if not raw.strip():
                continue
            f = raw.split("\t")
            if len(f) < len(header):
                continue
            row = {k: (f[i] if i < len(f) else "") for k, i in idx.items()}
            if not row.get("status", "").startswith("oracle-green"):
                continue
            n_rows += 1
            proofs = scenario_proofs(row.get("proof", ""))
            if not proofs:
                continue
            for scen_name, seeds in proofs:
                n_pairs += 1
                reasons = []
                if not snap.has_scenario(scen_name):
                    reasons.append("scenario file not found at census commit")
                if seeds and "1" not in seeds:
                    reasons.append(f"proof seeds @{{{','.join(seeds)}}}; retained dump is seed 1")
                dump = None
                if not reasons and (not seeds or "1" in seeds):
                    dump_path = dump_dir / f"{scen_name}.txt"
                    if dump_path.exists():
                        dump = parse_dump(dump_path)
                    else:
                        reasons.append("no census dump for scenario")
                if reasons:
                    unauditable.append({
                        "manifest_file": rel,
                        "case_id": row["case_id"],
                        "scenario": scen_name,
                        "reason": "; ".join(reasons),
                    })
                    continue

                warmup, probe, comments = scenario_parts(scen_name)
                actor_blocks = [(c, t) for c, r, t in dump if is_actor(r)]
                all_blocks = [(c, r, t) for c, r, t in dump]

                signals, evidences, refusal_evidence = [], [], []

                # Signal 1: warmup-refusal — the first actor probe block (the
                # DP-1510 fingerprint: the first command that depends on the
                # setup refuses). Refusals in later sibling probes are often
                # the intended claim of other rows sharing the scenario, and a
                # first probe whose refusal any sibling row claims (quoted in
                # its text) is a deliberate branch test, not a setup failure.
                if warmup and actor_blocks:
                    first_cmd, first_text = actor_blocks[0]
                    hits = refusal_lines(first_text)
                    claimed = scen_claimed.get(scen_name, set())
                    hits = [h for h in hits
                            if not any(c in h.lower() for c in claimed)]
                    if hits:
                        signals.append("warmup-refusal")
                        evidences.append(f"{first_cmd}: {hits[0]}")
                        refusal_evidence.extend(hits)

                # Signal 2: claimed-absent. Cited fragments keep full
                # strictness; the widened window and the row's own social
                # entry (claims living in the lib/misc/socials data table,
                # not in src/ literals) may suppress only via distinctive
                # fragments -- generic neighbors like the bare "failed." must
                # not mask a hollow.
                frags = row_fragments(snap, row.get("c_site", ""))
                if frags and all_blocks:
                    block_text = "\n".join(t for _, _, t in all_blocks)
                    if all(fr[3] not in block_text for fr in frags):
                        window_frags = row_fragments(snap, row.get("c_site", ""),
                                                     windowed=True)
                        social_frags = []
                        entry = socials.get((row.get("command") or "").lower())
                        if entry:
                            for line_no, msg in entry:
                                for frag in fragments_of(msg):
                                    social_frags.append(
                                        ("lib/misc/socials", line_no, msg, frag))
                        suppressed = any(
                            is_distinctive(fr[3]) and fr[3] in block_text
                            for fr in window_frags + social_frags)
                        if not suppressed:
                            best = max(frags, key=lambda fr: len(fr[3]))
                            signals.append("claimed-absent")
                            evidences.append(
                                f"{best[0]}:{best[1]} prints \"{best[2]}\" "
                                f"-> fragment '{best[3]}' absent from every C block"
                            )

                # Signal 3: probe-refusal.
                if all_blocks and all(block_is_refusal_or_empty(t) for _, _, t in all_blocks):
                    c0, r0, t0 = all_blocks[0]
                    first = refusal_lines(t0)
                    signals.append("probe-refusal")
                    evidences.append(
                        f"all {len(all_blocks)} C blocks empty/refusal; "
                        f"first [{c0}" + (f" {r0}]" if r0 else "]")
                        + (f": {first[0]}" if first else ": empty")
                    )
                    refusal_evidence.extend(first)

                if not signals:
                    continue
                is_int = intended(row, comments, refusal_evidence)
                notes = []
                if is_int:
                    notes.append("intended")
                if warmup:
                    notes.append("warmup: " + "; ".join(warmup[:3]))
                suspects.append({
                    "manifest_file": rel,
                    "case_id": row["case_id"],
                    "status": row["status"],
                    "scenario": scen_name,
                    "signal": ",".join(sorted(set(signals), key=lambda s: SIGNAL_RANK[s])),
                    "evidence_line": " | ".join(evidences),
                    "note": "; ".join(notes) if notes else "",
                })

    def strength(s):
        sigs = s["signal"].split(",")
        return (-len(sigs), min(SIGNAL_RANK[x] for x in sigs),
                s["manifest_file"], s["case_id"])

    suspects.sort(key=strength)
    intended_n = sum(1 for s in suspects if "intended" in s["note"])
    with open(out_dir / "suspects.tsv", "w", newline="") as fh:
        w = csv.DictWriter(fh, fieldnames=[
            "manifest_file", "case_id", "status", "scenario", "signal",
            "evidence_line", "note"], delimiter="\t", lineterminator="\n")
        w.writeheader()
        w.writerows(suspects)
    unauditable.sort(key=lambda u: (u["manifest_file"], u["case_id"], u["scenario"]))
    with open(out_dir / "unauditable.tsv", "w", newline="") as fh:
        w = csv.DictWriter(fh, fieldnames=[
            "manifest_file", "case_id", "scenario", "reason"],
            delimiter="\t", lineterminator="\n")
        w.writeheader()
        w.writerows(unauditable)

    print(f"oracle-green rows: {n_rows}; row x scenario pairs: {n_pairs}")
    print(f"suspects: {len(suspects)} (intended: {intended_n}; "
          f"counted: {len(suspects) - intended_n})")
    print(f"unauditable: {len(unauditable)}")
    for sig in ("claimed-absent", "warmup-refusal", "probe-refusal"):
        n = sum(1 for s in suspects if sig in s["signal"].split(",") and "intended" not in s["note"])
        print(f"  {sig}: {n}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
