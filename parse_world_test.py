#!/usr/bin/env python3
"""Regression tests for the Dark Pawns world parser."""

import io
from parse_world import WorldParser


def _make_parser() -> WorldParser:
    return WorldParser("/tmp/nonexistent_world")


def test_parse_exit_multiline_description_and_keywords():
    parser = _make_parser()
    lines = [
        "D0\n",
        "A heavy iron door blocks your way.\n",
        "It looks very sturdy.~\n",
        "iron door\n",
        "sturdy door~\n",
        "1 123 100\n",
    ]
    exit_data, new_i = parser._parse_exit(lines, 0, "north")
    assert exit_data is not None
    assert exit_data["direction"] == "north"
    assert exit_data["to_room"] == 100
    assert exit_data["door_state"] == 1
    assert exit_data["key"] == 123
    assert exit_data["description"] == "A heavy iron door blocks your way.\nIt looks very sturdy."
    assert exit_data["keywords"] == "iron door\nsturdy door"
    assert new_i == 6


def test_parse_exit_single_line():
    parser = _make_parser()
    lines = [
        "D1\n",
        "The eastern path.~\n",
        "path east~\n",
        "0 -1 200\n",
    ]
    exit_data, new_i = parser._parse_exit(lines, 0, "east")
    assert exit_data is not None
    assert exit_data["description"] == "The eastern path."
    assert exit_data["keywords"] == "path east"
    assert exit_data["to_room"] == 200
    assert new_i == 4


if __name__ == "__main__":
    test_parse_exit_multiline_description_and_keywords()
    test_parse_exit_single_line()
    print("parse_world tests passed")
