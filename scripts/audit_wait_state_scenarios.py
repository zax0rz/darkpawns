#!/usr/bin/env python3
"""Audit dp-oracle-diff scenarios for un-pulsed WAIT_STATE commands (DP-1531).

Derives the wait-setting command list directly from C (`WAIT_STATE(` in src/*.c
and `cmd_info[]` in src/interpreter.c), then scans every scenario in
cmd/dp-oracle-diff/scenarios/*.txt for cases where an actor (primary or a
directed `send:<peer>`) executes a wait-setting command followed by further
probe steps for that actor before any `~dpclock pulse`.
"""

from __future__ import annotations

import argparse
import glob
import json
import os
import re
import sys
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parent.parent

# Helper functions in src/ that call WAIT_STATE(ch, ...) on behalf of ACMD(s).
HELPER_CALLERS = {
    "perform_drop_gold": ["do_drop"],
    "mag_objectmagic": ["do_use"],
    "ambush_event": ["do_ambush"],
}


def find_wait_functions(src_dir: Path) -> dict[str, int]:
    """Parse src/*.c for WAIT_STATE( calls and map C functions to max wait pulses."""
    pulse_violence = 20  # 2 RL_SEC * 10 PASSES_PER_SEC in src/structs.h
    fn_waits: dict[str, int] = {}

    def eval_wait_expr(expr: str) -> int:
        expr = expr.replace("PULSE_VIOLENCE", str(pulse_violence))
        m = re.match(r"^\s*(\d+)\s*([+*]\s*\d+)?\s*$", expr)
        if m:
            base = int(m.group(1))
            if m.group(2):
                op_str = m.group(2).strip()
                op, rhs = op_str[0], int(op_str[1:].strip())
                return base + rhs if op == "+" else base * rhs
            return base
        if "*" in expr:
            parts = [p.strip() for p in expr.split("*")]
            if len(parts) == 2 and parts[0].isdigit() and parts[1].isdigit():
                return int(parts[0]) * int(parts[1])
        return pulse_violence * 3

    for path in sorted(src_dir.glob("*.c")):
        lines = path.read_text(errors="replace").splitlines()
        cur_fn = None
        for idx, line in enumerate(lines):
            m_acmd = re.match(r"^ACMD\((\w+)\)", line)
            if m_acmd:
                cur_fn = m_acmd.group(1)
            elif (
                re.match(r"^[A-Za-z_][A-Za-z0-9_*\s]+\(", line)
                and not line.startswith((" ", "\t"))
                and ";" not in line
            ):
                m_fn = re.match(r"^.*?\b(\w+)\s*\(", line)
                if m_fn:
                    cur_fn = m_fn.group(1)
            elif (
                idx > 0
                and re.match(r"^[a-z_][a-z0-9_]*\(", line)
                and not line.startswith((" ", "\t"))
                and ";" not in line
            ):
                cur_fn = line.split("(", 1)[0].strip()

            m_wait = re.search(r"WAIT_STATE\(\s*(\w+)\s*,\s*([^)]+)\)", line)
            if m_wait and cur_fn:
                target_var = m_wait.group(1)
                wait_pulses = eval_wait_expr(m_wait.group(2))
                if target_var == "ch":
                    for owner in HELPER_CALLERS.get(cur_fn, [cur_fn]):
                        fn_waits[owner] = max(fn_waits.get(owner, 0), wait_pulses)
    return fn_waits


def parse_cmd_info(interpreter_c: Path) -> list[tuple[str, str, str]]:
    """Parse cmd_info[] in src/interpreter.c in exact dispatch order."""
    text = interpreter_c.read_text(errors="replace")
    m = re.search(r"const struct command_info cmd_info\[\]\s*=\s*\{(.*?)\};", text, re.S)
    if not m:
        raise RuntimeError(f"Could not find cmd_info[] in {interpreter_c}")
    entries = re.findall(
        r'\{\s*"([^"]+)"\s*,\s*[^,]+,\s*(\w+)\s*,\s*[^,]+,\s*([^}]+?)\s*\}',
        m.group(1),
    )
    return [(cmd, fn, scmd.strip()) for cmd, fn, scmd in entries]


def resolve_cmd(
    word: str, cmd_info: list[tuple[str, str, str]]
) -> tuple[str, str, str] | None:
    w = word.lower()
    if not w:
        return None
    for cmd, fn, scmd in cmd_info:
        if cmd == "RESERVED":
            continue
        if cmd.startswith(w):
            return (cmd, fn, scmd)
    return None


def is_wait_cmd(
    resolved: tuple[str, str, str] | None,
    full_line: str,
    fn_waits: dict[str, int],
) -> int:
    """Return >0 wait pulses if the command can set WAIT_STATE(ch, ...), else 0."""
    if not resolved:
        return 0
    _cmd, fn, scmd = resolved
    if fn not in fn_waits:
        return 0
    if fn == "do_gen_tog":
        return fn_waits[fn] if "SCMD_NOSUMMON" in scmd else 0
    if fn == "do_drop":
        if scmd not in ("SCMD_DROP", "SCMD_DONATE"):
            return 0
        # perform_drop_gold only runs for coins/gold
        return fn_waits[fn] if re.search(r"\b(coin|coins|gold)\b", full_line, re.I) else 0
    if fn == "do_use":
        # Bare q/quaff/recite/use without an argument returns before mag_objectmagic.
        parts = full_line.strip().split()
        return fn_waits[fn] if len(parts) >= 2 else 0
    return fn_waits[fn]


def scan_scenarios(
    scenarios_dir: Path,
    cmd_info: list[tuple[str, str, str]],
    fn_waits: dict[str, int],
) -> list[dict]:
    results = []
    for path in sorted(scenarios_dir.glob("*.txt")):
        name = path.stem
        lines = path.read_text(errors="replace").splitlines()
        section = None
        waiting: dict[str, dict] = {}
        probe_hits = []
        warmup_hits = []

        for lineno, raw in enumerate(lines, 1):
            s = raw.strip()
            if not s or s.startswith("#"):
                continue
            if s.startswith("[") and s.endswith("]"):
                section = s
                continue
            if section not in ("[warmup]", "[probe]") and not (
                section and section.startswith("[probe:")
            ):
                continue
            if s.startswith("~dpclock pulse"):
                waiting.clear()
                continue

            actor = "primary"
            cmd_line = s
            if s.startswith("send:"):
                rest = s[len("send:") :]
                parts = rest.split(" ", 1)
                actor = parts[0]
                cmd_line = parts[1] if len(parts) > 1 else ""

            if section != "[warmup]" and actor in waiting:
                prev = waiting[actor]
                hit = {
                    "actor": actor,
                    "wait_line": prev["line"],
                    "wait_cmd": prev["cmd"],
                    "wait_step": prev["step"],
                    "wait_pulses": prev["pulses"],
                    "wait_section": prev["section"],
                    "probe_line": lineno,
                    "probe_step": s,
                }
                if prev["section"] == "[warmup]":
                    warmup_hits.append(hit)
                    del waiting[actor]
                else:
                    probe_hits.append(hit)
                    continue

            tokens = cmd_line.strip().split()
            first_word = tokens[0] if tokens else ""
            if not first_word or first_word.startswith(("'", '"', ";", ":")):
                continue
            resolved = resolve_cmd(first_word, cmd_info)
            wait_pulses = is_wait_cmd(resolved, cmd_line, fn_waits)
            if wait_pulses > 0:
                waiting[actor] = {
                    "line": lineno,
                    "cmd": resolved[0],
                    "step": s,
                    "pulses": wait_pulses,
                    "section": section,
                }

        if probe_hits or warmup_hits:
            results.append(
                {
                    "name": name,
                    "probe_hits": probe_hits,
                    "warmup_hits": warmup_hits,
                }
            )
    return results


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--json", action="store_true", help="Output JSON")
    parser.add_argument(
        "--names", action="store_true", help="Output comma-separated scenario names"
    )
    args = parser.parse_args()

    fn_waits = find_wait_functions(REPO_ROOT / "src")
    cmd_info = parse_cmd_info(REPO_ROOT / "src" / "interpreter.c")
    results = scan_scenarios(
        REPO_ROOT / "cmd" / "dp-oracle-diff" / "scenarios", cmd_info, fn_waits
    )

    if args.json:
        print(json.dumps({"fn_waits": fn_waits, "flagged": results}, indent=2))
        return 0
    if args.names:
        print(",".join(r["name"] for r in results))
        return 0

    probe_only = [r for r in results if r["probe_hits"]]
    warmup_only = [r for r in results if r["warmup_hits"] and not r["probe_hits"]]
    print(f"Total flagged scenarios: {len(results)}")
    print(f"  probe->probe: {len(probe_only)}")
    print(f"  warmup->probe only: {len(warmup_only)}")
    for r in results:
        if r["probe_hits"]:
            h = r["probe_hits"][0]
            print(
                f"  [probe] {r['name']}: L{h['wait_line']} ({h['wait_cmd']}, {h['wait_pulses']}p) -> L{h['probe_line']} {h['probe_step']!r} (+{len(r['probe_hits'])-1} more)"
            )
        if r["warmup_hits"]:
            h = r["warmup_hits"][0]
            print(
                f"  [warmup->probe] {r['name']}: L{h['wait_line']} ({h['wait_cmd']}, {h['wait_pulses']}p) -> L{h['probe_line']} {h['probe_step']!r}"
            )
    return 0


if __name__ == "__main__":
    sys.exit(main())
