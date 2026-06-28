"""Self-tests for scripts/fidelity_data.py.

Run with: python3 -m pytest scripts/fidelity_data_test.py
"""

import re
from pathlib import Path

import fidelity_data as fd

REPO_ROOT = Path(__file__).resolve().parent.parent


def _find_go_symbol(go_file: Path, symbol: str) -> bool:
    """Return True if a Go file declares the given var/const symbol."""
    text = go_file.read_text(encoding="utf-8", errors="ignore")
    # Match "var Symbol" or "const Symbol" or "var (\nSymbol".
    patterns = [
        rf"\bvar\s+{re.escape(symbol)}\b",
        rf"\bconst\s+{re.escape(symbol)}\b",
    ]
    return any(re.search(p, text) for p in patterns)


def test_all_mapped_go_symbols_exist() -> None:
    """Every ARRAY_MAP entry with a non-empty go_name must exist in its Go file."""
    missing: list[str] = []
    for mapping in fd.ARRAY_MAP:
        if not mapping.go_name:
            continue
        go_file = REPO_ROOT / mapping.go_file
        if not go_file.exists():
            missing.append(f"{mapping.go_file} does not exist")
            continue
        if not _find_go_symbol(go_file, mapping.go_name):
            missing.append(f"{mapping.go_file}: {mapping.go_name} not found")
    assert not missing, "missing Go symbols:\n" + "\n".join(missing)


def test_check_all_loads_without_crashing() -> None:
    """check_all() must run and report structured results."""
    results = fd.check_all(REPO_ROOT)
    assert isinstance(results, dict)
    # Every mapped pair should have an entry.
    expected_keys = {
        f"{m.c_name} -> {m.go_name}"
        for m in fd.ARRAY_MAP
        if m.go_name
    }
    assert set(results.keys()) == expected_keys


def test_main_returns_int() -> None:
    """main() should return an integer exit code."""
    code = fd.main()
    assert isinstance(code, int)
