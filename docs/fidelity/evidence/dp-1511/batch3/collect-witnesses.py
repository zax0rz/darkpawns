#!/usr/bin/env python3
"""Retain actual event lines and outcomes from completed DP-1511 batch-3 runs."""
import csv
import json
import pathlib

HERE = pathlib.Path(__file__).resolve().parent
ARCHIVE = pathlib.Path.home() / 'Archives/darkpawns/oracle-runs/2026-11-09'
SEEDS = json.loads((HERE / 'original-seeds.json').read_text())
SPELLS = {
    'color-spray': ['Your spray of color misses ColorMissVictim -- what a dull world...',
        'ColorMissActor unsuccesfully tries to add color to your life!',
        'ColorMissActor splashes color everywhere except ColorMissVictim.'],
    'dispel-evil': ['DispelMissVictim is unaffected by your goodness!',
        'DispelMissActor thinks that he is really good, and tries to dispell you.',
        'DispelMissVictim laughs at DispelMissActor as he tries to dispell him!'],
    'dispel-good': ['GoodVictim is unaffected by your evil!',
        'GoodActor thinks that he is really evil, and tries to dispell you.',
        'GoodVictim laughs at GoodActor as he tries to dispell him!'],
    'energy-drain': ['You need more draining lessons...',
        'Luckily for you EnergyActor slept during draining-lessons!',
        'EnergyActor fails to drain EnergyVictim -- what a waste of energy!'],
}

HITS = {
    'color-spray': ['Your own private rainbow hits ColorMissVictim in the face -- WOWIE!!',
        'A sudden burst of colors sends you reeling!', 'ColorMissActor throws colors at ColorMissVictim who in vain tries to escape!'],
    'dispel-evil': ['DispelMissVictim shivers and suffers from his evilness!',
        'DispelMissActor makes your soul hurt and suffer!', "DispelMissActor makes DispelMissVictim's evil spirit shiver and suffer!"],
    'dispel-good': ['GoodVictim shivers and suffers from his goodness!',
        'GoodActor makes your soul hurt and suffer!', "GoodActor makes GoodVictim's good spirit shiver and suffer!"],
    'energy-drain': ['You drain EnergyVictim of some of his energy!',
        'You feel less energetic as EnergyActor drains you!', 'EnergyActor drains EnergyVictim -- what a waste of energy!'],
}

def dump(run, name):
    directory = ARCHIVE / run
    summary = directory / 'summary.txt'
    assert summary.exists() and 'verdict=CLEAN' in summary.read_text(), run
    path = directory / 'census-dump' / (name + '.txt')
    lines = path.read_text().splitlines()
    assert lines[0] == 'normalized C oracle blocks:'
    return path, lines


def pick(path, lines, needle):
    events = []
    block = ''
    seen = set()
    for i, line in enumerate(lines):
        if line.startswith('--- ['):
            block = line
        elif needle in line and block not in seen:
            seen.add(block)
            events.append({'line': line, 'dump': str(path), 'dump_line': i + 1, 'block': block})
    assert events, (path, needle)
    return events

records = []
spell_outcomes = []
sleep_outcomes = []
vampire_outcomes = []
cylinder_outcomes = []
for name, seeds in SEEDS.items():
    run = 'sol-dp1511-b3-final'
    needles = []
    if 'skill-message-miss' in name:
        spell = name.removesuffix('-skill-message-miss-depth')
        if spell in ['dispel-evil', 'dispel-good']:
            run = 'sol-dp1511-b3-after-seed3'
        needles = SPELLS[spell] + ['You recite', 'sanctuary=1']
    elif name.startswith('item-sleep-'):
        needles = ['You quaff some northern devil ginseng powder.',
            'Pulling a bit of sand' if name.endswith('-sand') else 'You attempt the spell without the components...',
            'Your spell fails to affect them because you are not an Outlaw!' if '-refusal-' in name else 'You feel very sleepy...']
    elif name.startswith('vampire-consume-'):
        needles = ['Lothar the Vampire Lord sinks his fangs into your neck!', 'You drink the beer.', 'You eat a loaf of bread.']
        if not name.endswith('-day'):
            needles += ['The vampirism in your body is not satiated by mere beer...',
                        'The vampirism in your body is not satiated by mere food...']
    elif name == 'appraise-depth':
        needles = ["You estimate it's worth"]
    elif name == 'spec-proc-elements-load-cylinders':
        needles = ['A green cylinder of light extends upwards from the pillar.',
                   'The green cylinder of light slowly sinks back into the pillar.',
                   'You get an elemental talisman of earth.']
    elif name == 'object-door-messages':
        needles = ['Dooractor opens the door.', 'Dooractor closes the door.']
    elif name == 'ban-depth':
        needles = ['No sites are banned.', 'Site banned.']
    elif name == 'ban-unban-success-depth':
        needles = ['Site banned.', 'Site unbanned.']
    elif name == 'act-messages':
        needles = ["You whisper to Actvictim, 'quiet words'", "Actactor whispers to you, 'quiet words'",
                   'Actactor whispers something to Actvictim.', 'You recite a backpack which dissolves.',
                   'Actactor recites a backpack.']
    elif name == 'afk-depth':
        needles = ['Afkdepth goes AFK...', 'Afkdepth returns from some repulsive act...']
    path, lines = dump(run, name)
    events = [event for needle in needles for event in pick(path, lines, needle)]
    verdict = 'Refusal intended' if name.startswith('ban-') else 'Go disagrees with C (destination bytes blocked)' if name == 'object-door-messages' else 'Repaired'
    records.append({'scenario': name, 'verdict': verdict, 'run': run, 'events': events})
    for seed in seeds:
        seed_run = 'sol-dp1511-b3-final' if seed == 1 else f'sol-dp1511-b3-after-seed{seed}'
        if name == 'spec-proc-elements-load-cylinders' and seed != 1:
            seed_run = f'sol-dp1511-b3-witness-repair-seed{seed}'
        seed_path, seed_lines = dump(seed_run, name)
        text = '\n'.join(seed_lines)
        if 'skill-message-miss' in name:
            spell = name.removesuffix('-skill-message-miss-depth')
            miss = SPELLS[spell][0] in text
            if miss:
                assert all(line in text for line in SPELLS[spell]), (name, seed, 'incomplete miss triplet')
            assert miss or all(line in text for line in HITS[spell]), (name, seed, 'incomplete hit triplet')
            assert 'You recite' in text and 'sanctuary=1' in text
            spell_outcomes.append([name, seed, 'miss' if miss else 'hit', str(seed_path)])
        elif name.startswith('item-sleep-'):
            outcome = ('refusal' if 'Your spell fails to affect them because you are not an Outlaw!' in text
                       else 'sleep' if 'You feel very sleepy...' in text else 'saved')
            assert outcome != 'saved' or 'shakes his head wearily' in text
            assert 'You quaff some northern devil ginseng powder.' in text
            assert ('Pulling a bit of sand' in text) == name.endswith('-sand')
            sleep_outcomes.append([name, seed, outcome, 'sand' if name.endswith('-sand') else 'bare', str(seed_path)])
        elif name.startswith('vampire-consume-'):
            warnings = text.count('The vampirism in your body is not satiated')
            assert warnings == (0 if name.endswith('-day') else 4), (name, seed, warnings)
            assert 'sinks his fangs into your neck!' in text and 'You drink the beer.' in text and 'You eat a loaf of bread.' in text
            vampire_outcomes.append([name, seed, warnings, str(seed_path)])

        elif name == 'spec-proc-elements-load-cylinders':
            assert 'You get an elemental talisman of earth.' in text
            assert text.count('A green cylinder of light extends upwards from the pillar.') == 2
            assert text.count('The green cylinder of light slowly sinks back into the pillar.') == 2
            cylinder_outcomes.append([name, seed, 'create-and-remove', str(seed_path)])

warmups = []
for name in ['ban-depth', 'ban-unban-success-depth']:
    directory = ARCHIVE / 'sol-dp1511-b3-final'
    for engine, needle in [('c', 'Site unbanned.'), ('go', 'That site is not currently banned.')]:
        paths = list(directory.glob(f'initial-logs/*/{name}.capture1/{engine}-warmup.txt'))
        assert len(paths) == 1
        path = paths[0]
        lines = path.read_text().splitlines()
        index = lines.index(needle)
        warmups.append({'scenario': name, 'engine': engine, 'line': needle, 'dump': str(path), 'dump_line': index + 1})
(HERE / 'witnesses.json').write_text(json.dumps({'cases': records, 'intentional_warmups': warmups}, indent=2) + '\n')
for filename, header, rows in [
    ('spell-outcomes.tsv', ['scenario', 'seed', 'outcome', 'C_dump'], spell_outcomes),
    ('sleep-outcomes.tsv', ['scenario', 'seed', 'outcome', 'reagent', 'C_dump'], sleep_outcomes),
    ('vampire-outcomes.tsv', ['scenario', 'seed', 'night_warning_count', 'C_dump'], vampire_outcomes),
    ('cylinder-outcomes.tsv', ['scenario', 'seed', 'outcome', 'C_dump'], cylinder_outcomes),
    ('triage.tsv', ['scenario', 'verdict', 'original_seeds', 'C_witness', 'dump', 'dump_line'], [
        [r['scenario'], r['verdict'], ','.join(map(str, SEEDS[r['scenario']])),
         r['events'][0]['line'], r['events'][0]['dump'], r['events'][0]['dump_line']] for r in records]),
]:
    with (HERE / filename).open('w') as out:
        writer = csv.writer(out, delimiter='\t', lineterminator='\n')
        writer.writerow(header)
        writer.writerows(rows)
print(f'{len(records)} cases; {sum(len(r["events"]) for r in records)} real C event lines; '
      f'{len(warmups)} intentional C/Go warmup lines')
