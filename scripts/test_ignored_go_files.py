#!/usr/bin/env python3
"""Tests for scripts/check_ignored_go_files.py.

Run with the repository's script-test discovery:

    python3 -m unittest discover -s scripts -p 'test_*.py'

The point of these tests is the trap itself: a test file whose name ends in a
GOOS or GOARCH token is dropped from the build, so its tests never run and
nothing fails. ``test_arm_suffixed_test_file_is_reported`` builds a throwaway
module containing exactly that file and asserts the gate flags it.
"""

import os
import subprocess
import sys
import tempfile
import unittest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import check_ignored_go_files as guard  # noqa: E402  (path set above)

REPO_ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
GUARD = os.path.join(REPO_ROOT, "scripts", "check_ignored_go_files.py")


def write_module(root, files):
    for name, content in files.items():
        path = os.path.join(root, name)
        os.makedirs(os.path.dirname(path), exist_ok=True) if os.path.dirname(name) else None
        with open(path, "w", encoding="utf-8") as handle:
            handle.write(content)


TRAP_FILES = {
    "go.mod": "module example.com/armtrap\n\ngo 1.21\n",
    "trap.go": "package armtrap\n\nfunc Trap() int { return 1 }\n",
    "trap_arm_test.go": "package armtrap\n\nimport \"testing\"\n\nfunc TestArmOnly(t *testing.T) {}\n",
}


class ParsingTest(unittest.TestCase):
    def test_parses_concatenated_json_objects(self):
        stream = (
            '{"ImportPath": "example.com/m", "IgnoredGoFiles": ["a_windows.go"]}'
            '{"ImportPath": "example.com/m/sub"}'
            '{"ImportPath": "example.com/m/other", "IgnoredGoFiles": ["b_arm.go"]}'
        )
        self.assertEqual(
            guard.parse_go_list(stream),
            [("example.com/m", "a_windows.go"), ("example.com/m/other", "b_arm.go")],
        )

    def test_repo_relative_strips_the_module_prefix(self):
        self.assertEqual(
            guard.repo_relative("example.com/m/pkg/session", "x_windows.go", "example.com/m"),
            "pkg/session/x_windows.go",
        )

    def test_unexpected_ignores_allowlisted_paths(self):
        pairs = [("example.com/m/pkg/a", "x_windows.go"), ("example.com/m/pkg/b", "y_arm_test.go")]
        allowed = {"pkg/a/x_windows.go": "GOOS=windows"}
        self.assertEqual(guard.unexpected(pairs, "example.com/m", allowed), ["pkg/b/y_arm_test.go"])


class TrapTest(unittest.TestCase):
    """The failure this gate exists for: a *_arm_test.go that never compiles."""

    def test_arm_suffixed_test_file_is_reported(self):
        with tempfile.TemporaryDirectory() as root:
            write_module(root, TRAP_FILES)
            pairs = guard.collect_ignored(root)
            self.assertIn(
                ("example.com/armtrap", "trap_arm_test.go"),
                pairs,
                "Go did not ignore trap_arm_test.go; the trap this gate guards against may have changed",
            )
            self.assertEqual(guard.unexpected(pairs, "example.com/armtrap", allowed={}), ["trap_arm_test.go"])

    def test_cli_exits_one_and_names_the_file(self):
        with tempfile.TemporaryDirectory() as root:
            write_module(root, TRAP_FILES)
            result = subprocess.run(
                [sys.executable, GUARD, "--root", root],
                capture_output=True,
                text=True,
                cwd=REPO_ROOT,
            )
            self.assertEqual(result.returncode, 1, result.stderr)
            self.assertIn("trap_arm_test.go", result.stderr)
            self.assertIn("not allowlisted", result.stderr)

    def test_windows_file_passes_when_allowlisted(self):
        with tempfile.TemporaryDirectory() as root:
            write_module(root, {
                "go.mod": "module example.com/winonly\n\ngo 1.21\n",
                "win.go": "package winonly\n\nfunc Win() int { return 1 }\n",
                "extra_windows.go": "package winonly\n\nfunc Extra() int { return 2 }\n",
            })
            pairs = guard.collect_ignored(root)
            allowed = {"extra_windows.go": "GOOS=windows"}
            self.assertEqual(guard.unexpected(pairs, "example.com/winonly", allowed), [])


class RealTreeTest(unittest.TestCase):
    """The allowlist must describe this tree, or the gate rots silently."""

    def test_no_unallowlisted_ignored_files(self):
        module = guard.module_path(REPO_ROOT)
        pairs = guard.collect_ignored(REPO_ROOT)
        self.assertEqual(guard.unexpected(pairs, module), [])

    def test_every_allowlisted_path_exists(self):
        for path in guard.ALLOWED:
            self.assertTrue(os.path.exists(os.path.join(REPO_ROOT, path)), "%s is allowlisted but absent" % path)

    def test_cli_passes_on_this_tree(self):
        result = subprocess.run(
            [sys.executable, GUARD, "--root", REPO_ROOT],
            capture_output=True,
            text=True,
            cwd=REPO_ROOT,
        )
        self.assertEqual(result.returncode, 0, result.stderr)


if __name__ == "__main__":
    unittest.main()
