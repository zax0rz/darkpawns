"""Self-tests for scripts/trace_path.py.

Run with: python3 -m pytest scripts/trace_path_test.py

Regression test for DP-810: trace_path.py used to hardcode a developer-only
absolute WORLD_DIR path. load_world() must work against any directory it's
given, with no dependence on a specific machine's filesystem layout.
"""

from pathlib import Path

import trace_path as tp

# Minimal .wld content in the same format as real files under lib/world/wld/:
# "#<vnum>\n<name>~\n<desc>~\n<flags>\nD<dir>\n<door desc>~\n<door keywords>~\n
#  <exit flags> <key> <to_room>\nS\n...$~"
WLD_CONTENT = """#100
Test Room~
  A small test room used for regression testing.
~
0 0 0 0 0 0
D0
a plain doorway~
door~
0 -1 101
S
#101
Second Test Room~
  Another small test room, reachable by going north from room 100.
~
0 0 0 0 0 0
S
$~
"""


def test_default_world_dir_points_at_repo_lib_world_wld():
    """The default WORLD_DIR must be derived from the script location, not hardcoded."""
    repo_root = Path(__file__).resolve().parent.parent
    assert tp.DEFAULT_WORLD_DIR == repo_root / "lib" / "world" / "wld"
    # Derived from __file__, not a hardcoded developer-specific absolute path.
    assert tp.DEFAULT_WORLD_DIR.is_relative_to(repo_root)


def test_load_world_from_temp_dir(tmp_path):
    """load_world() must load rooms from an arbitrary directory (hermetic, no dev path)."""
    (tmp_path / "100.wld").write_text(WLD_CONTENT)

    rooms = tp.load_world(str(tmp_path))

    assert set(rooms.keys()) == {100, 101}
    assert rooms[100]["name"] == "Test Room"
    assert rooms[101]["name"] == "Second Test Room"
    assert rooms[100]["exits"] == {0: 101}
    assert rooms[101]["exits"] == {}


def test_trace_follows_exit_between_rooms(tmp_path):
    (tmp_path / "100.wld").write_text(WLD_CONTENT)
    rooms = tp.load_world(str(tmp_path))

    end_vnum, result = tp.trace(rooms, 100, "n")

    assert end_vnum == 101
    assert result == "Second Test Room"


def test_trace_reports_no_exit(tmp_path):
    (tmp_path / "100.wld").write_text(WLD_CONTENT)
    rooms = tp.load_world(str(tmp_path))

    end_vnum, result = tp.trace(rooms, 100, "s")

    assert end_vnum == 100
    assert "NO EXIT" in result


def test_resolve_world_dir_honors_env_var(monkeypatch):
    monkeypatch.setattr(tp.sys, "argv", ["trace_path.py"])
    monkeypatch.setenv("DP_WORLD_DIR", "/some/override/path")
    assert tp.resolve_world_dir() == "/some/override/path"


def test_resolve_world_dir_honors_cli_arg(monkeypatch):
    monkeypatch.delenv("DP_WORLD_DIR", raising=False)
    monkeypatch.setattr(tp.sys, "argv", ["trace_path.py", "/cli/override/path"])
    assert tp.resolve_world_dir() == "/cli/override/path"


def test_resolve_world_dir_defaults_to_repo_layout(monkeypatch):
    monkeypatch.setattr(tp.sys, "argv", ["trace_path.py"])
    monkeypatch.delenv("DP_WORLD_DIR", raising=False)
    assert tp.resolve_world_dir() == str(tp.DEFAULT_WORLD_DIR)
