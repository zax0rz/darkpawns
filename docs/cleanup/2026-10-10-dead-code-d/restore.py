#!/usr/bin/env python3
"""PR-1 review fixes: restore tests + helpers Claude's review required.

Extracts named top-level blocks (with their doc comments) from origin/main
and appends them to the current files. Usage: restore.py <worktree>
"""
import re
import subprocess
import sys
from pathlib import Path

# file -> list of (kind, name) in origin/main; kind: func|type
RESTORE = {
    'pkg/testutil/helpers.go': [('func', 'NewTestPlayer')],
    'pkg/combat/engine_standup_test.go': [
        ('type', 'recordingRoller'),
        ('func', 'TestProcessCombatPair_MobStandupRoundDrawsFirst'),
        ('func', 'TestProcessCombatPair_MobAttacksOnStandupRound'),
        ('func', 'TestProcessCombatPair_PCStandup'),
        ('func', 'TestProcessCombatPair_PositionGateAWAKE'),
    ],
    'pkg/game/spec_cleric_test.go': [
        ('type', 'clericTestDraw'),
        ('func', 'scriptClericRolls'),
        ('func', 'TestSpecCleric_EntryGatesAndStand'),
        ('func', 'TestSpecCleric_Lspell12IsAnOffensiveNoOp'),
        ('func', 'TestSpecCleric_EarthquakeUsesNPCRoomMessage'),
        ('func', 'TestSpecCleric_CallLightningWeatherGate'),
        ('func', 'TestSpecCleric_BlindnessGateConsumesCDraw'),
    ],
    'pkg/olc/registry_test.go': [
        ('func', 'TestRegistryUsesTypedKeysAndStableOwners'),
        ('func', 'TestRegistryListSnapshotsClaimsAndAcquisitionTime'),
    ],
    'pkg/command/skill_commands.go': [
        ('func', 'CmdLearn'),
        ('func', 'CmdListSkills'),
    ],
    'pkg/command/skill_commands_test.go': [
        ('func', 'TestCmdLearn'),
    ],
}


def extract(lines: list, kind: str, name: str):
    pat = re.compile(r'^(?:' + kind + r')\s+' + re.escape(name) + r'[\s(]')
    starts = [i for i, l in enumerate(lines) if pat.match(l)]
    if not starts:
        raise SystemExit(f'NOT FOUND in origin/main: {kind} {name}')
    s = starts[0]
    # doc comment above
    ds = s
    while ds > 0 and lines[ds - 1].lstrip().startswith('//'):
        ds -= 1
    e = s
    if not lines[e].rstrip().endswith('}'):
        while e < len(lines) and lines[e] != '}':
            e += 1
    return lines[ds:e + 1]


def main() -> None:
    root = Path(sys.argv[1])
    for rel, items in RESTORE.items():
        p = root / rel
        origin = subprocess.run(['git', '-C', str(root), 'show', f'origin/main:{rel}'],
                                capture_output=True, text=True).stdout.splitlines()
        cur = p.read_text().splitlines()
        if cur and cur[-1] == '':
            cur = cur[:-1]
        added = []
        for kind, name in items:
            if any(re.match(r'^(?:func|type)\s+' + re.escape(name) + r'[\s(]', l)
                   for l in cur):
                print(f'{rel}: {name} already present, skipped')
                continue
            block = extract(origin, kind, name)
            cur += [''] + block
            added.append(name)
        p.write_text('\n'.join(cur) + '\n')
        print(f'{rel}: restored {len(added)}: {", ".join(added)}')


if __name__ == '__main__':
    main()
