#!/usr/bin/env python3
"""
Tests for scripts/wire_scripts.py.

The key regression: add_script_to_mob must insert the Script: line inside the
*correct* mob block (bounded from #<vnum> to the next #<number>/EOF), never
into a neighbouring block. The old non-greedy DOTALL regex could cross block
boundaries and corrupt an adjacent mob.
"""

import os
import sys
import tempfile

import pytest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import wire_scripts  # noqa: E402


# A minimal two-mob .mob file. Mob #1 has a normal E terminator; mob #2 is the
# target. Both have proper terminators so the happy path is covered.
TWO_MOB_CONTENT = """\
#1
alpha~
alpha~
An alpha mob.
~
It looks ordinary.
\
16393 0 0 0 0 0 0 0 1000 E
26 -6 -16 26d5+322 17d4+17
E
#2
beta~
beta~
A beta mob.
~
It looks ordinary.
\
8 0 0 0 0 0 0 0 0 E
10 10 0 10d5+110 5d4+5
E
"""


def _write_mob(tmpdir, content, name="test.mob"):
    path = os.path.join(tmpdir, name)
    with open(path, "w") as f:
        f.write(content)
    return path


def test_script_lands_in_target_block():
    """The Script line must appear inside mob #2, not mob #1."""
    with tempfile.TemporaryDirectory() as d:
        path = _write_mob(d, TWO_MOB_CONTENT)
        ok, msg = wire_scripts.add_script_to_mob(path, 2, "beta.lua")
        assert ok, msg

        with open(path) as f:
            result = f.read()

        # Mob #2 is the last block, so it runs from "#2\n" to EOF.
        block2_start = result.index("#2\n")
        block2 = result[block2_start:]
        assert "Script: beta.lua" in block2, (
            f"Script line not in mob #2 block; got:\n{block2}"
        )

        # Mob #1's block must NOT have gained a Script line.
        block1_start = result.index("#1\n")
        block1 = result[block1_start:result.index("#2\n")]
        assert "Script:" not in block1, (
            f"Script line leaked into mob #1 block; got:\n{block1}"
        )


def test_bitmask_appended():
    """When get_trigger_bitmask returns >0, it is appended to the Script line."""
    with tempfile.TemporaryDirectory() as d:
        path = _write_mob(d, TWO_MOB_CONTENT)
        # 'fighter' in the script name yields bitmask 256 (see
        # get_trigger_bitmask fallback).
        ok, _ = wire_scripts.add_script_to_mob(path, 2, "fighter.lua")
        assert ok

        with open(path) as f:
            result = f.read()
        assert "Script: fighter.lua 256" in result, (
            f"expected bitmask 256 appended; got:\n{result}"
        )


def test_idempotent_when_already_wired():
    """Re-wiring a correctly-wired mob reports no change and leaves content."""
    with tempfile.TemporaryDirectory() as d:
        path = _write_mob(d, TWO_MOB_CONTENT)
        wire_scripts.add_script_to_mob(path, 2, "beta.lua")
        with open(path) as f:
            after_first = f.read()

        ok, msg = wire_scripts.add_script_to_mob(path, 2, "beta.lua")
        assert not ok, f"expected no-op on second wire, got ok=True ({msg})"
        assert "already" in msg.lower()

        with open(path) as f:
            after_second = f.read()
        assert after_first == after_second, "content changed on idempotent re-wire"


def test_mob_not_found():
    """A missing vnum reports not-found without writing."""
    with tempfile.TemporaryDirectory() as d:
        path = _write_mob(d, TWO_MOB_CONTENT)
        ok, msg = wire_scripts.add_script_to_mob(path, 999, "nope.lua")
        assert not ok
        assert "not found" in msg.lower()


# The dangerous case: a target mob block whose own E terminator is missing.
# The OLD non-greedy regex `#2\n.*?\nE\n` would run past #2 into #3 and steal
# #3's E, inserting the Script line into mob #3's block. The bounded version
# confines the insertion to #2's block (which then gets an E appended so the
# file stays parseable on subsequent runs).
MISSING_E_CONTENT = """\
#1
alpha~
An alpha mob.
~
8 0 0 0 0 0 0 0 0 E
E
#2
beta~
A beta mob missing its own E terminator.
~
8 0 0 0 0 0 0 0 0 E
#3
gamma~
A gamma mob.
~
8 0 0 0 0 0 0 0 0 E
E
"""


def test_does_not_cross_into_next_block_when_e_missing():
    """Even without its own E, the Script must stay in #2, not bleed into #3."""
    with tempfile.TemporaryDirectory() as d:
        path = _write_mob(d, MISSING_E_CONTENT)
        ok, _ = wire_scripts.add_script_to_mob(path, 2, "beta.lua")

        # With no E terminator inside #2's block, the function should report
        # the structural problem rather than silently corrupting #3.
        if ok:
            # If it did wire, the Script line must be in #2's bounded region
            # (between #2 and #3), NOT in #3's block.
            with open(path) as f:
                result = f.read()
            b2 = result[result.index("#2\n"):result.index("#3\n")]
            b3 = result[result.index("#3\n"):]
            assert "Script: beta.lua" in b2
            assert "Script: beta.lua" not in b3, "Script bled into mob #3!"
        else:
            # Acceptable: refuse to wire a malformed block. The key invariant
            # is that #3 is never corrupted either way.
            with open(path) as f:
                result = f.read()
            assert "Script:" not in result, (
                "malformed block should not add a Script anywhere"
            )


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
