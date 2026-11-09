#!/usr/bin/env python3
"""Draw the preregistered round-three samples and one-byte output probes."""

import ast
import csv
import hashlib
import random
import string
from pathlib import Path


ROOT = Path(__file__).resolve().parents[5]
ROUND3 = ROOT / "docs/research/idiom-experiment/round3"
PACKAGES = {
    "session": {
        "seed": 20261106,
        "dir": ROUND3 / "population/session",
        "population_sha256": "3494855e2685510eee44cb1deeb435c0e0ee5260840b0e9d34469a7c9e602ecf",
        "literals_sha256": "7063b2f178f29e26075cb10e4d2d790edff3d84f84531f2365f514cdfad6ec32",
    },
    "spells": {
        "seed": 20261107,
        "dir": ROUND3 / "population/spells",
        "population_sha256": "bc5f237f34db3a5e5d57a77807446d60a9bc0438f0e1dace770f8c73db2b3fa0",
        "literals_sha256": "49d4f28b01e62959d3ff3aea3cdf0bef769e70b08d3abf10ba6fc316360b5e00",
    },
}
PREDICTED_SCENARIOS = {
    "session": {
        0: "cast-depth",
        1: "levels-depth,levels-arguments-depth",
        2: "flee-combat-forced,flee-no-exits",
        3: "pager-navigation",
        4: "qecho-depth",
        5: "send-depth",
        6: "transfer-depth",
        7: "lifecycle-idle-void,return-depth",
        9: "commands-depth",
        10: "qui-depth",
        11: "shutdown-depth",
        12: "sedit-session-depth,sedit-output-residuals-depth",
        13: "retreat-closed-door",
        14: "lines-depth",
        15: "recite-effect,recite-damage-spell",
        16: "sethunt-depth",
        17: "wizard-wiznet-depth",
        18: "qsay-depth",
        19: "skills-practice,practice-catalog-mage-cost",
        20: "date-depth",
        21: "mlist-depth",
        22: "group-depth",
        23: "dc-depth",
        24: "shutdown-depth",
        25: "sedit-session-depth,olc-save-depth",
        26: "snoop-depth,lifecycle-idle-force-rent-close",
        27: "wizard-stat-object-contents-depth",
        28: "toggle-cluster,god-toggles",
        29: "skillset-depth",
        31: "idlist-depth,idlist-gate-depth",
        32: "input-history-depth",
        35: "poofout-depth,poofin-depth",
        36: "snoop-depth",
        37: "group-depth,group-sleeping-send-to-char-depth",
        38: "teleport-depth,transfer-depth",
        39: "dig-depth,dig-permission-depth",
    },
    "spells": {
        0: "cast-depth,position-gates",
        1: "identify-character-depth",
        2: "spell-charm-save-npc-retaliation-depth,spell-charm-success-follow-audience-depth",
        3: "spell-points-spoken-depth,spellmatrix-world-potion-points-depth",
        4: "spell-points-spoken-depth,spellmatrix-mag-points-missing-depth",
        5: "cast-mental-lapse-spoken-depth",
        6: "will-vampirism-npc-silent-depth,spellmatrix-night-creatures-spoken-depth",
        7: "detect-poison-depth",
        8: "spellmatrix-mass-dominate-spoken-depth",
        9: "detect-poison-depth,identify-character-depth",
        10: "cast-locate-object-no-results-script-depth,cast-locate-object-no-locate-depth",
        11: "",
        13: "will-zen-meditation-depth",
        14: "spell-parser-scroll-fighting",
        15: "damage-neutral-spell-rescue,damage-message-color-depth",
        16: "teleport-depth,spellmatrix-shift-reality-spoken-depth",
        17: "spellmatrix-earthquake-spoken-depth,cast-meteor-swarm-success-depth",
        18: "mag-unaffects-room-output-depth,spellmatrix-mag-affects-spoken-depth",
        19: "spell-parser-scroll-fighting",
        20: "will-coc-room-audience-depth",
        21: "spellmatrix-create-food-staff-fanout-depth",
        22: "cast-meteor-swarm-success-depth,cast-meteor-swarm-peaceful-depth",
        23: "cast-create-water-depth",
        24: "cast-mirror-image-depth",
        25: "spellmatrix-mag-affects-spoken-depth,spellmatrix-mag-affects-batch-depth",
        26: "spellmatrix-hellfire-spoken-depth,cast-hellfire-area-depth",
        27: "enchant-armor-depth,spellmatrix-world-scroll-enchant-armor-depth",
        28: "identify-scroll-depth",
        29: "flesh-alter-depth,flesh-alter-gates",
        30: "spellmatrix-night-creatures-spoken-depth",
        31: "spell-parser-scroll-fighting",
        32: "mag-unaffects-room-output-depth",
        33: "cast-mindsight-watcher-depth,cast-mindsight-no-track",
        34: "spell-control-weather-invalid-depth,spellmatrix-control-weather-spoken-depth",
        35: "cast-divine-int-success-depth,cast-divine-int-neutral-depth,cast-divine-int-save-failure-depth",
        37: "summon-success-output-depth,summon-backfire-output-depth",
        38: "enchant-weapon-depth,spellmatrix-world-scroll-enchant-weapon-depth",
        39: "spellmatrix-mag-affects-spoken-depth,spellmatrix-mag-affects-batch-depth,egon-equipment-apply-spell",
    },
}


def read_tsv(path):
    with path.open(newline="", encoding="utf-8") as stream:
        return list(csv.DictReader(stream, delimiter="\t", quoting=csv.QUOTE_NONE))


def sha256(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def unescape_tsv(value):
    """Reverse population.go's backslash-first TSV escaping."""
    out = []
    i = 0
    while i < len(value):
        if value[i] != "\\" or i + 1 == len(value):
            out.append(value[i])
            i += 1
            continue
        i += 1
        esc = value[i]
        out.append({"\\": "\\", "t": "\t", "r": "\r", "n": "\n"}.get(esc, "\\" + esc))
        i += 1
    return "".join(out)


def go_unquote(token):
    if token.startswith("`") and token.endswith("`"):
        return token[1:-1]
    return ast.literal_eval(token)


def escape_token_segment(segment):
    """Return the decoded value of one Go string-token segment."""
    return go_unquote('"' + segment + '"')


def encoded_letter_offset(raw_token, desired):
    """Find one source byte that changes the first ASCII letter to desired."""
    if raw_token.startswith("`"):
        for rel, char in enumerate(raw_token[1:-1], start=1):
            if char in string.ascii_letters:
                byte_rel = len(raw_token[:rel].encode("utf-8"))
                return byte_rel, char, desired
        raise ValueError(f"no ASCII letter in {raw_token!r}")

    i = 1
    end = len(raw_token) - 1
    while i < end:
        char = raw_token[i]
        if char != "\\":
            if char in string.ascii_letters:
                byte_rel = len(raw_token[:i].encode("utf-8"))
                return byte_rel, char, desired
            i += 1
            continue

        esc = raw_token[i + 1]
        if esc in "x":
            stop = i + 4
        elif esc == "u":
            stop = i + 6
        elif esc == "U":
            stop = i + 10
        elif esc in "01234567":
            stop = i + 2
            while stop < min(i + 4, end) and raw_token[stop] in "01234567":
                stop += 1
        else:
            stop = i + 2
        segment = raw_token[i:stop]
        decoded = escape_token_segment(segment)
        letter_index = next((j for j, c in enumerate(decoded) if c in string.ascii_letters), None)
        if letter_index is not None:
            for rel in range(i + 1, stop):
                for replacement in string.printable:
                    candidate = raw_token[:rel] + replacement + raw_token[rel + 1 :]
                    try:
                        if go_unquote(candidate) == decoded[:letter_index] + desired + decoded[letter_index + 1 :]:
                            byte_rel = len(raw_token[:rel].encode("utf-8"))
                            return byte_rel, raw_token[rel], replacement
                    except (SyntaxError, ValueError):
                        continue
            raise ValueError(f"cannot one-byte mutate escaped letter in {raw_token!r}")
        i = stop
    raise ValueError(f"no ASCII letter in {raw_token!r}")


def first_letter(value):
    return next((char for char in value if char in string.ascii_letters), None)


def display_value(value):
    return value.encode("unicode_escape").decode("ascii")


def write_tsv(path, fieldnames, rows):
    with path.open("w", newline="", encoding="utf-8") as stream:
        writer = csv.DictWriter(stream, fieldnames=fieldnames, delimiter="\t", lineterminator="\n")
        writer.writeheader()
        writer.writerows(rows)


def main():
    scenario_names = {path.stem for path in (ROOT / "cmd/dp-oracle-diff/scenarios").glob("*.txt")}
    for package, predictions in PREDICTED_SCENARIOS.items():
        for i, values in predictions.items():
            missing = [name for name in values.split(",") if name and name not in scenario_names]
            if missing:
                raise SystemExit(f"unknown predicted scenarios for {package} probe {i}: {missing}")

    samples = []
    probes = []
    for package, config in PACKAGES.items():
        pop_path = config["dir"] / "population.tsv"
        literals_path = config["dir"] / "population-literals.tsv"
        if sha256(pop_path) != config["population_sha256"]:
            raise SystemExit(f"population hash changed: {pop_path}")
        if sha256(literals_path) != config["literals_sha256"]:
            raise SystemExit(f"literal hash changed: {literals_path}")

        population = read_tsv(pop_path)
        if len(population) < 40:
            raise SystemExit(f"{package} population has only {len(population)} rows")
        literal_rows = read_tsv(literals_path)
        by_function = {}
        for item in literal_rows:
            item["raw_token"] = unescape_tsv(item["literal"])
            item["offset"] = int(item["offset"])
            by_function.setdefault((item["func"], item["file"]), []).append(item)
        for items in by_function.values():
            items.sort(key=lambda item: item["offset"])

        selected = random.Random(config["seed"]).sample(population, 40)
        for i, function in enumerate(selected):
            function_key = (function["func"], function["file"])
            function_literals = by_function[function_key]
            probe = random.Random(config["seed"] + i).choice(function_literals)
            value = go_unquote(probe["raw_token"])
            first = first_letter(value)
            if first is None:
                raise SystemExit(f"selected literal has no ASCII letter: {probe}")
            replacement = "Z" if first == "Q" else "Q"
            relative, old_byte, new_byte = encoded_letter_offset(probe["raw_token"], replacement)
            function_line = int(function["line"])
            source_offset = int(probe["offset"]) + relative
            source = (ROOT / probe["file"]).read_bytes()
            token_bytes = probe["raw_token"].encode("utf-8")
            if source[int(probe["offset"]) : int(probe["offset"]) + len(token_bytes)] != token_bytes:
                raise SystemExit(f"literal offset/token mismatch: {probe}")
            if source[source_offset : source_offset + 1] != old_byte.encode("ascii"):
                raise SystemExit(f"mutation byte mismatch at {source_offset}: {probe}")
            samples.append(
                {
                    "package": package,
                    "seed": config["seed"],
                    "i": i,
                    "func": function["func"],
                    "file": function["file"],
                    "function_line": function_line,
                    "n_literals": function["n_literals"],
                }
            )
            probes.append(
                {
                    "package": package,
                    "seed": config["seed"],
                    "i": i,
                    "func": function["func"],
                    "file": function["file"],
                    "function_line": function_line,
                    "literal_line": probe["line"],
                    "literal": probe["raw_token"],
                    "literal_offset": probe["offset"],
                    "mutation_offset": source_offset,
                    "old_byte": old_byte,
                    "new_byte": new_byte,
                    "mutated_value": display_value(value.replace(first, replacement, 1)),
                    "predicted_scenarios": PREDICTED_SCENARIOS.get(package, {}).get(i, ""),
                    "census_dir": "",
                    "verdict": "",
                    "failed_scenarios": "",
                    "caught": "",
                    "first_pass_verdict": "",
                    "c_cite": "",
                    "notes": "",
                }
            )

    sample_fields = ["package", "seed", "i", "func", "file", "function_line", "n_literals"]
    probe_fields = [
        "package", "seed", "i", "func", "file", "function_line", "literal_line", "literal",
        "literal_offset", "mutation_offset", "old_byte", "new_byte", "mutated_value",
        "predicted_scenarios", "census_dir", "verdict", "failed_scenarios", "caught",
        "first_pass_verdict", "c_cite", "notes",
    ]
    write_tsv(ROUND3 / "run/sample.tsv", sample_fields, samples)
    write_tsv(ROUND3 / "run/probes.tsv", probe_fields, probes)
    print(f"wrote {len(samples)} sampled functions and {len(probes)} probes")


if __name__ == "__main__":
    main()
