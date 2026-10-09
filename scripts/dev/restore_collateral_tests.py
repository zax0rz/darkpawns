#!/usr/bin/env python3
"""Restore tests deleted by a dead-code PR that do not call any function
the PR deleted (PR #1891 review rule, 2026-10-09).

A test may be deleted only if its body calls at least one non-test function
that the PR also deleted; everything else is restored verbatim from the base
commit, together with any test helpers it needs that went missing with it.

Usage:
    python3 scripts/dev/restore_collateral_tests.py BASE_REF [--apply]

Without --apply it only reports. Run from the PR's worktree with the PR's
commits checked out.
"""
import re
import subprocess
import sys

FUNC = re.compile(r"^func (\([^)]*\) )?([A-Za-z_][A-Za-z0-9_]*)\(")


def funcs_at(ref):
    """Map file path -> {func name -> (start line, end line)} at ref."""
    out = {}
    files = subprocess.run(
        ["git", "ls-tree", "-r", "--name-only", ref],
        capture_output=True, text=True, check=True).stdout.splitlines()
    for path in files:
        if not path.endswith(".go"):
            continue
        src = subprocess.run(
            ["git", "show", f"{ref}:{path}"], capture_output=True, text=True)
        if src.returncode != 0:
            continue
        lines = src.stdout.split("\n")
        fmap = {}
        for i, line in enumerate(lines):
            m = FUNC.match(line)
            if not m:
                continue
            j = i + 1
            while j < len(lines) and lines[j] != "}":
                j += 1
            fmap[m.group(2)] = (i, j)
        if fmap:
            out[path] = fmap
    return out


def body(path, name, ref):
    src = subprocess.run(["git", "show", f"{ref}:{path}"],
                         capture_output=True, text=True, check=True).stdout
    lines = src.split("\n")
    i, j = funcs_at_ref_cache[path][name]
    s = i
    while s - 1 >= 0 and lines[s - 1].startswith("//") and not lines[s - 1].startswith("// ---"):
        s -= 1
    return "\n".join(lines[s:j + 1])


def calls(text):
    """Identifiers called in the body: foo( and pkg.foo( / recv.foo( forms.
    Dotted calls are included conservatively: matching a deleted name by
    method name alone errs toward keeping a test deleted, never toward
    restoring one that calls deleted code."""
    out = set(re.findall(r"(?<![.\w])([a-zA-Z_][A-Za-z0-9_]*)\s*\(", text))
    out |= set(re.findall(r"\.([A-Za-z_][A-Za-z0-9_]*)\s*\(", text))
    return out


def main():
    base = sys.argv[1]
    apply = "--apply" in sys.argv
    head = "HEAD"

    global funcs_at_ref_cache
    base_funcs = funcs_at(base)
    funcs_at_ref_cache = base_funcs
    head_funcs = funcs_at(head)

    deleted_prod = {}   # name -> path (non-test)
    deleted_tests = {}  # (path, name)
    deleted_helpers = {}  # name -> path (non-Test test-file funcs)
    for path, fmap in base_funcs.items():
        for name in fmap:
            if name in head_funcs.get(path, {}):
                continue
            if path.endswith("_test.go"):
                if name.startswith("Test"):
                    deleted_tests[(path, name)] = True
                else:
                    deleted_helpers[name] = path
            else:
                deleted_prod[name] = path
    # helpers named in test files that survived under another file's name are
    # not "deleted functions" for the rule; only production deletions count.
    prod_names = set(deleted_prod)

    restore = []
    keep = []
    for (path, name) in sorted(deleted_tests):
        text = body(path, name, base)
        hit = prod_names & calls(text)
        if hit:
            keep.append((path, name, sorted(hit)))
        else:
            restore.append((path, name))

    # Transitively pull in deleted test helpers that restored tests call.
    need = {}
    changed = True
    restored_names = {n for _, n in restore}
    while changed:
        changed = False
        for (path, name) in restore:
            for c in calls(body(path, name, base)):
                if c in deleted_helpers and c not in need and c not in restored_names:
                    need[c] = deleted_helpers[c]
                    restored_names.add(c)
                    changed = True

    print(f"deleted production functions: {len(prod_names)}")
    print(f"deleted tests: {len(deleted_tests)}")
    print(f"  keep (calls a deleted function): {len(keep)}")
    for path, name, hit in keep:
        print(f"    {name} [{path}] -> {', '.join(hit)}")
    print(f"  restore (no deleted call): {len(restore)}")
    for path, name in restore:
        print(f"    {name} [{path}]")
    print(f"helpers restored alongside: {sorted(need)}")

    if not apply:
        return

    by_file = {}
    for path, name in restore:
        by_file.setdefault(path, []).append(name)
    for hname, hpath in need.items():
        by_file.setdefault(hpath, []).append(hname)
    for path, names in by_file.items():
        cur = open(path).read()
        for n in names:
            block = body(path, n, base)
            cur = cur + "\n" + block + "\n"
        open(path, "w").write(cur)
        print(f"restored {len(names)} into {path}")


if __name__ == "__main__":
    main()
