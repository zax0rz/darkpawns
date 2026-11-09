#!/usr/bin/env python3
"""Extract actual C witnesses only from the named completed clean runs."""
import csv
import json
import pathlib

HERE = pathlib.Path(__file__).resolve().parent
ARCHIVE = pathlib.Path.home() / 'Archives/darkpawns/oracle-runs/2026-11-09'
SEEDS = json.loads((HERE / 'original-seeds.json').read_text())
SELECTORS = {
    'magic-surface-callsite-depth': [
        'The gods protect Casttarget.', 'You create a Magic Mushroom.',
        'Castgod creates a Magic Mushroom.', "Casttarget's wounds glow and mend themselves.",
        'You feel better.', 'As you quickly mumble the incantation',
        'Suddenly, a cloud of thick, acrid black smoke appears',
        'There is no corpse!', 'The corpse starts to twitch, then stands'],
    'mudlog-wizutil-depth': ['[ (GC) Wizvictim pardoned by Wizactor ]',
        '[ (GC) Notitle ON', '[ (GC) Notitle OFF', '[ (GC) Squelch ON',
        '[ (GC) Squelch OFF', '[ (GC) Wizvictim frozen', '[ (GC) Wizvictim un-frozen'],
    'wizutil-thaw-higher-freeze-depth': ['Sorry, a level 40 God froze Thawvictim'],
    'spell-charm-save-npc-retaliation-depth': ['Your victim resists!',
        'A guard trainee seems to be having trouble getting his body to fight you!!'],
    'summon-high-level-victim-depth': ['You failed.', 'Castgod just tried to summon you to:', 'He failed.'],
    'spec-proc-breed-killer-growl': ["You hear a low growl in the back of a nightbreed hunter's throat."],
    'spec-proc-cityguard-pissed': ["A cityguard says, 'You just pissed me off, a large snake!'"],
    'spec-proc-elements-guardian-mumble': ['A guardian spirit mumbles softly and you begin to scream,',
        'A guardian spirit mumbles softly and ElemPeer begins screaming loudly'],
    'spec-proc-mindflayer': ['wrapping around your head!', "wrapping around Mindactor's head!"],
    'spec-proc-mindflayer-soul-leech': ['wrapping around your head!', "wrapping around Mindactor's head!"],
    'spec-proc-snake-bite': ['A large snake bites you!', 'A large snake bites SnakeGod!'],
    'spec-proc-take-to-jail-pissed': ["Hans the gateguard says, 'You just pissed me off, GuardAlly!'"],
    'spec-proc-troll': ["Targ's wounds glow brightly for a moment, then disappear!"],
    'spec-proc-troll-combat': ["Targ's wounds glow brightly for a moment, then disappear!"],
    'spec-proc-troll-regen': [],  # Two phase-specific witnesses below.
    'spec-proc-werewolf': ['The werewolf looks up and lets out a long, fierce howl.',
        'The werewolf tears into your leg with his huge fangs!',
        "The werewolf rips apart WolfVictim's leg with his fangs!", 'You hear a loud howling in the distance.'],
    'spec-proc-werewolf-acts': ['The werewolf looks up and lets out a long, fierce howl.',
        'The werewolf tears into your leg with his huge fangs!',
        "The werewolf rips apart WolfVictim's leg with his fangs!", 'You hear a loud howling in the distance.'],
}

def extract(name, run, needles, seed=1):
    directory = ARCHIVE / run
    summary = directory / 'summary.txt'
    assert summary.exists() and 'verdict=CLEAN' in summary.read_text(), f'not completed clean: {run}'
    dump = directory / 'census-dump' / (name + '.txt')
    lines = dump.read_text().splitlines()
    assert lines[0] == 'normalized C oracle blocks:'
    indices = [next(i for i, line in enumerate(lines) if needle in line) for needle in needles]
    if name == 'spec-proc-troll-regen':
        hit = next(i for i, line in enumerate(lines) if line.startswith('--- [hit targ'))
        indices = [next(i for i, line in enumerate(lines) if i < hit and 'wounds glow' in line),
                   next(i for i, line in enumerate(lines) if i > hit and 'wounds glow' in line)]
    events = []
    for i in indices:
        block = next(line for line in reversed(lines[:i]) if line.startswith('--- ['))
        events.append({'line': lines[i], 'dump': str(dump), 'dump_line': i + 1, 'block': block})
    return {'scenario': name, 'seed': seed, 'run': run, 'events': events}

records = []
for name in SEEDS:
    run = 'sol-dp1511-b2-final-draws'
    if name == 'magic-surface-callsite-depth':
        run = 'sol-dp1511-b2-magic-audience-seed1'
    elif name == 'spec-proc-troll-regen':
        run = 'sol-dp1511-b2-troll-idle-witness'
    records.append(extract(name, run, SELECTORS[name]))
for seed in [2, 3, 5, 8]:
    records.append(extract('spec-proc-werewolf', f'sol-dp1511-b2-witness-repair-seed{seed}',
        SELECTORS['spec-proc-werewolf'], seed))
    records.append(extract('magic-surface-callsite-depth', f'sol-dp1511-b2-magic-audience-seed{seed}',
        SELECTORS['magic-surface-callsite-depth'], seed))
(HERE / 'witnesses.json').write_text(json.dumps(records, indent=2) + '\n')
with (HERE / 'triage.tsv').open('w') as out:
    writer = csv.writer(out, delimiter='\t', lineterminator='\n')
    writer.writerow(['scenario', 'verdict', 'original_seeds', 'C_witness', 'dump', 'dump_line'])
    for record in records[:len(SEEDS)]:
        event = record['events'][0]
        writer.writerow([record['scenario'], 'Repaired', ','.join(map(str, SEEDS[record['scenario']])),
                         event['line'], event['dump'], event['dump_line']])
print(f'{len(SEEDS)} scenarios; {len(records)} scenario/seed witness sets; '
      f'{sum(len(row["events"]) for row in records)} actual C lines')
