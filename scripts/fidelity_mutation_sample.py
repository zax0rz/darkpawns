#!/usr/bin/env python3
"""Pre-register a deterministic stratified sample of runnable unit-proof symbols."""
from collections import defaultdict
import argparse
import csv
from datetime import datetime, timezone
import hashlib
import json
from pathlib import Path
import subprocess

from fidelity_manifest import load_rows

SEED = 20260930
# Classification depends only on manifest metadata, never on test bodies.
GROUPS = {
    "entry-persistence": "entry idle lifecycle persist save rent crash password",
    "combat": "combat damage fight hit kill flee assist rescue shoot whirlwind dragon smackheads backstab bash kick disarm circle strike retreat",
    "skills-spells": "skill spell practice cast quaff recite use brew forge sharpen enchant trance meditate morph concentrate",
    "objects-economy": "object get put drop give equip wear wield remove junk sacrifice shop buy sell steal gold bank auc loot eat drink fill light door open close lock unlock pick",
    "wizard-admin": "wizard show stat set advance restore load purge force snoop switch return olc reload dns ban zedit medit oedit tedit vnum mlist olist rlist mstat",
    "scripts-specials": "lua scripting script spec-procs spec special mobile mob",
    "communication": "say tell whisper ask shout gossip gtell gsay channel board write mail note report chat act social emote insult group follow order shadow",
    "world-utilities": "",
}

def rank(seed, value):
    return hashlib.sha256(f"{seed}:{value}".encode()).hexdigest()

def category(row):
    stem = Path(row["manifest"]).stem
    for name, words in GROUPS.items():
        if any(stem == w or stem.startswith(w + "-") for w in words.split()):
            return name
    return "world-utilities"

def sample(frame, size, seed):
    strata = defaultdict(list)
    for item in frame:
        strata[(item["package"], item["category"])].append(item)
    if not len(strata) <= size <= len(frame):
        raise ValueError("sample size must cover each nonempty stratum and fit the frame")
    quotas = {key: 1 for key in strata}
    # Hamilton apportionment of remaining slots, proportional to each
    # stratum's unsampled population; minimum-one coverage is kept explicit.
    slots = size - len(strata)
    remaining = sum(len(v) - 1 for v in strata.values())
    remainders = {}
    if remaining:
        for key, members in strata.items():
            count, rem = divmod(slots * (len(members) - 1), remaining)
            quotas[key] += count
            remainders[key] = rem
        for key in sorted(strata, key=lambda k: (-remainders[k], rank(seed, repr(k))))[:size - sum(quotas.values())]:
            quotas[key] += 1
    selected = []
    for key in sorted(strata):
        candidates = sorted(strata[key], key=lambda x: (rank(seed, x["symbol"]), x["symbol"]))
        selected.extend(candidates[:quotas[key]])
    selected.sort(key=lambda x: (x["package"], x["category"], x["symbol"]))
    return selected, [{"package": k[0], "category": k[1], "population": len(strata[k]), "sample": quotas[k]} for k in sorted(strata)]

def compact_selection(payload, frame_sha256):
    return {
        "seed": payload["seed"], "frame_sha256": frame_sha256,
        "head": payload["head"], "created_utc": payload["created_utc"],
        "selected": [
            {"id": c["id"], "symbol": c["symbol"], "package": c["package"],
             "anchor": {k: c["anchor"][k] for k in ["manifest", "line", "case_id"]}}
            for c in payload["selected"]
        ],
    }


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--root", type=Path, required=True)
    p.add_argument("--baseline", type=Path, required=True)
    p.add_argument("--out", type=Path, required=True)
    p.add_argument("--seed", type=int, default=SEED)
    p.add_argument("--size", type=int, default=100)
    p.add_argument("--created-utc", help="original lock timestamp for byte-identical regeneration")
    p.add_argument("--selection-out", type=Path, help="optional compact selection and full-frame checksum")
    a = p.parse_args()
    if a.out.exists() or (a.selection_out and a.selection_out.exists()):
        p.error("refusing to overwrite a locked sampling frame")
    claims = list(csv.DictReader((a.baseline / "unit-results.tsv").open(), delimiter="\t"))
    if any(c["problem"] != "PASS" for c in claims):
        p.error("baseline contains unproven claims; resolve sampling eligibility explicitly")
    index = json.loads((a.baseline / "test-index.json").read_text())
    declarations = {(d["file"], str(d["line"])): d for d in index["tests"]}
    grouped = defaultdict(list)
    rows = {(r["manifest"], r["line"], r["case_id"]): r for r in load_rows(a.root / "docs/fidelity/depth", a.root)}
    for claim in claims:
        grouped[claim["symbol"]].append(rows[(claim["manifest"], claim["line"], claim["case_id"])])
    frame = []
    for symbol, references in sorted(grouped.items()):
        references.sort(key=lambda r: (rank(a.seed, symbol + ":" + r["case_id"]), r["case_id"]))
        claim = next(c for c in claims if c["symbol"] == symbol)
        d = declarations[(claim["test_file"], str(claim["test_line"]))]
        frame.append({"symbol": symbol, "package": d["package"], "file": d["file"], "line": d["line"], "category": category(references[0]), "anchor": references[0], "references": references})
    chosen, strata = sample(frame, a.size, a.seed)
    for number, item in enumerate(chosen, 1):
        item["id"] = f"M{number:03d}"
    head = subprocess.check_output(["git", "-C", str(a.root), "rev-parse", "HEAD"], text=True).strip()
    payload = {"created_utc": a.created_utc or datetime.now(timezone.utc).isoformat(), "head": head, "seed": a.seed, "method": "one per package/category stratum; Hamilton proportional remainder allocation; SHA256(seed:symbol) ranks within stratum", "row_claims": len(claims), "frame_size": len(frame), "sample_size": len(chosen), "groups": GROUPS, "strata": strata, "frame": frame, "selected": chosen}
    a.out.parent.mkdir(parents=True, exist_ok=True)
    a.out.write_text(json.dumps(payload, indent=2, sort_keys=True) + "\n")
    if a.selection_out:
        a.selection_out.parent.mkdir(parents=True, exist_ok=True)
        a.selection_out.write_text(json.dumps(compact_selection(payload, hashlib.sha256(a.out.read_bytes()).hexdigest()), indent=2, sort_keys=True) + "\n")
    print(f"locked: {a.out} sha256={hashlib.sha256(a.out.read_bytes()).hexdigest()} population={len(frame)} selected={len(chosen)} strata={len(strata)}")

if __name__ == "__main__":
    main()
