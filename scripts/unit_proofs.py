"""Exact Go test declaration and execution evidence for unit-proven manifests
(unit-green and divergent-approved)."""
from __future__ import annotations

from collections import defaultdict
import json
import os
from pathlib import Path
import re
import subprocess
import tempfile

from fidelity_manifest import UNIT_PROOF_STATUSES

TOOL_ROOT = Path(__file__).resolve().parents[1]


def test_index(root):
    result = subprocess.run(["go", "run", "./scripts/fidelitytests", str(root)],
                            cwd=TOOL_ROOT, capture_output=True, text=True)
    if result.returncode:
        raise ValueError(f"Go test index failed: {result.stderr.strip()}")
    return json.loads(result.stdout)


def proof_symbols(proof):
    # Canonical subtest names use testing's whitespace-to-underscore spelling.
    return [s for s in re.split(r"[;,\s]+", proof.strip()) if s]


def resolve(symbol, index):
    package, separator, test = symbol.partition(":")
    if not separator:
        package, test = "", symbol
    root_name = test.split("/", 1)[0]
    declarations = [d for d in index["tests"] if d["name"] == root_name and
                    (not package or d["package"].removeprefix("./") == package.removeprefix("./"))]
    valid = [d for d in declarations if d["valid"]]
    if len(valid) > 1:
        return None, "ambiguous", "name occurs in multiple test declarations; qualify its package"
    if not valid:
        if declarations:
            return None, "not a test func", "declaration does not have a runnable testing.T signature"
        locations = sorted(path for path, text in index["files"].items() if root_name in text)
        if locations:
            return None, "matched only as a string", ", ".join(locations)
        return None, "missing", "no exact Go test declaration"
    return valid[0], "", ""


def execution_batches(resolved, root, out, timeout="10m"):
    packages = defaultdict(set)
    for declaration, name in resolved:
        packages[declaration["package"]].add(name.split("/", 1)[0])
    runs = {}
    for number, (package, names) in enumerate(sorted(packages.items()), 1):
        pattern = "^(" + "|".join(re.escape(name) for name in sorted(names)) + ")$"
        command = ["go", "test", "-json", "-count=1", "-timeout", timeout, "-run", pattern, package]
        log_path = out / f"package-{number:02d}.jsonl"
        with log_path.open("w") as log:
            result = subprocess.run(command, cwd=root, stdout=log, stderr=subprocess.STDOUT,
                                    env=dict(os.environ))
        events = []
        for line in log_path.read_text().splitlines():
            try:
                event = json.loads(line)
            except json.JSONDecodeError:
                continue
            events.append(event)
        runs[package] = {"status": result.returncode, "events": events, "command": command,
                         "log": log_path.name}
    return runs


def outcome(declaration, name, run):
    events = run["events"]
    # Pass/skip/fail terminal events, not matching names in stdout.
    terminal = {event["Test"]: event["Action"] for event in events
                if "Test" in event and event["Action"] in {"pass", "skip", "fail"}}
    ancestors = ["/".join(name.split("/")[:i]) for i in range(1, len(name.split("/")) + 1)]
    for ancestor in ancestors:
        if terminal.get(ancestor) == "skip":
            return "skipped", f"{ancestor} skipped"
        if terminal.get(ancestor) == "fail":
            return "failing", f"{ancestor} failed"
    if name not in terminal:
        if "fail" not in terminal.values() and any(e.get("Action") == "fail" and "Test" not in e for e in events):
            return "package failure", "package failed before proving this symbol"
        return "missing", "no terminal go test event for this exact name (build tags, naming or subtest mismatch)"
    children = {test: action for test, action in terminal.items() if test.startswith(name + "/")}
    skipped = sorted(test for test, action in children.items() if action == "skip")
    if skipped:
        return "skipped", "skipped descendant(s): " + ", ".join(skipped)
    if run["status"] and "fail" not in terminal.values():
        return "package failure", "package test command failed; a passing event alone does not prove a clean batch"
    subpath = name.partition("/")[2]
    if declaration["vacuous"] or (subpath and subpath in (declaration.get("vacuous_subtests") or [])):
        return "asserts nothing", "empty or literal-logging-only body; no checker/helper or panic-sensitive operation"
    return "PASS", "exact declaration and non-skipped terminal pass event"


def validate_unit_rows(rows, root=TOOL_ROOT, index=None):
    index = test_index(root) if index is None else index
    errors = []
    subclaims = []
    for row in rows:
        if row["status"] not in UNIT_PROOF_STATUSES:
            continue
        symbols = proof_symbols(row["proof"])
        location = f"{row['manifest']}:{row['line']} ({row['case_id']})"
        if row["status"] == "divergent-approved" and not re.search(r"\bDP-\d+\b", row.get("notes", "")):
            errors.append(f"{location}: divergent-approved row must cite its approving issue (DP-nnnn) in notes")
        if not symbols:
            errors.append(f"{location}: empty unit proof")
        for symbol in symbols:
            declaration, problem, detail = resolve(symbol, index)
            if problem:
                errors.append(f"{location}: {symbol}: {problem}: {detail}")
            elif "/" in (symbol.partition(":")[2] if ":" in symbol else symbol):
                # A literal Run alone can be unreachable. Verify the emitted runtime name.
                name = symbol.partition(":")[2] if ":" in symbol else symbol
                subclaims.append((row, symbol, declaration, name))
    if subclaims:
        with tempfile.TemporaryDirectory(prefix="dp-unit-subtests-") as directory:
            runs = execution_batches([(d, n) for _, _, d, n in subclaims], root, Path(directory))
            for row, symbol, declaration, name in subclaims:
                problem, detail = outcome(declaration, name, runs[declaration["package"]])
                if problem != "PASS":
                    errors.append(f"{row['manifest']}:{row['line']} ({row['case_id']}): {symbol}: {problem}: {detail}")
    return errors
