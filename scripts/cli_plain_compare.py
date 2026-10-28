#!/usr/bin/env python3
"""Compare legacy launcher stdout, stderr and exit codes without a terminal.

Build the base and candidate launchers first. The shared disposable executable
path keeps doctor/recovery paths stable across binaries. No world is started.
"""
import argparse
import json
import os
from pathlib import Path
import shutil
import subprocess


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("before", type=Path)
    parser.add_argument("after", type=Path)
    parser.add_argument("--evidence", type=Path, default=Path(".local/cli-evidence/plain"))
    args = parser.parse_args()
    output = args.evidence.resolve()
    output.mkdir(parents=True, exist_ok=True)
    instance = output / "instance"
    instance.mkdir(exist_ok=True, mode=0o700)
    config = instance / "instance.json"
    config.write_text(json.dumps({"format": 1, "http_port": 45319, "jwt_secret": "00" * 32}))
    config.chmod(0o600)
    anchor = output / "darkpawns"
    commands = [
        ["status"], ["status", "--json"], ["doctor"], ["doctor", "--json"],
        ["update", "--check"],
        ["update", "--check", "--archive", str(output / "missing.tar.gz"), "--sha256", "00" * 32],
        ["setup", "--plain", "--instance", str(output / "missing-world")],
    ]
    report = []
    for number, command in enumerate(commands):
        results = []
        for label, binary in [("before", args.before), ("after", args.after)]:
            replacement = output / "replacement"
            shutil.copy2(binary, replacement)
            os.replace(replacement, anchor)
            result = subprocess.run([str(anchor), "--instance", str(instance), *command],
                                    input=b"n\n", capture_output=True, timeout=15, check=False)
            results.append((result.returncode, result.stdout, result.stderr))
            (output / f"{number}-{label}.stdout").write_bytes(result.stdout)
            (output / f"{number}-{label}.stderr").write_bytes(result.stderr)
        if results[0] != results[1]:
            raise SystemExit(f"plain output differs: {command!r}")
        report.append({"command": command, "same_exit_stdout_stderr": True})
    (output / "comparison.json").write_text(json.dumps(report, indent=2) + "\n")
    print(f"plain comparison: {len(report)} exact exit/stdout/stderr matches")


if __name__ == "__main__":
    main()
