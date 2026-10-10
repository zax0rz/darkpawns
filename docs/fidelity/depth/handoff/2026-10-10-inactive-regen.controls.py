#!/usr/bin/env python3
"""Prove inactive regeneration and active controls with temporary gate mutations."""

from pathlib import Path
import subprocess
import sys


ROOT = Path(__file__).resolve().parents[4]
SOURCE = ROOT / "pkg/game/limits_condition.go"
TEST = "TestPointUpdateInactiveRegen"
COMMAND = ["go", "test", "./pkg/game", "-run", f"^{TEST}$", "-count=1", "-v"]
GATE = "// C: limits.c:495-501 gates all three player resource gains on\n\t\t\t// PRF_INACTIVE; poison/cutthroat damage below remains outside it.\n\t\t\tif !inactive {"


def run(
    label: str,
    *,
    expected_failures: tuple[str, ...] = (),
    expected_passes: tuple[str, ...] = (),
) -> None:
    result = subprocess.run(COMMAND, cwd=ROOT, text=True, capture_output=True)
    output = result.stdout + result.stderr
    for assertion in expected_failures:
        if assertion not in output:
            raise SystemExit(f"{label}: missing expected assertion {assertion!r}\n{output}")
    for test in expected_passes:
        if test not in output:
            raise SystemExit(f"{label}: missing expected passing case {test!r}\n{output}")
    if result.returncode == 0 and expected_failures:
        raise SystemExit(f"{label}: expected assertion failures, but test passed\n{output}")
    if result.returncode != 0 and not expected_failures:
        raise SystemExit(f"{label}: test failed\n{output}")
    print(f"{label}: {'expected assertion failures' if expected_failures else 'PASS'}")


def main() -> int:
    original = SOURCE.read_text()
    if original.count(GATE) != 1:
        raise SystemExit("expected exactly one inactive regen gate; source was not changed")

    run(
        "fixed source",
        expected_passes=("--- PASS: TestPointUpdateInactiveRegen/active_resources_regenerate",),
    )
    mutations = (
        (
            "gate removed",
            GATE.removesuffix("if !inactive {") + "if true {",
            (
                "inactive vitals changed:",
                "inactive poisoned HP=",
                "inactive cutthroat HP=",
            ),
            ("--- PASS: TestPointUpdateInactiveRegen/active_resources_regenerate",),
        ),
        (
            "gate inverted",
            GATE.removesuffix("if !inactive {") + "if inactive {",
            ("active vitals did not all regenerate:",),
            (),
        ),
    )
    try:
        for label, mutated_gate, expected_failures, expected_passes in mutations:
            SOURCE.write_text(original.replace(GATE, mutated_gate, 1))
            run(label, expected_failures=expected_failures, expected_passes=expected_passes)
            SOURCE.write_text(original)
    finally:
        SOURCE.write_text(original)
    run(
        "restored source",
        expected_passes=("--- PASS: TestPointUpdateInactiveRegen/active_resources_regenerate",),
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())
