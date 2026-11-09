#!/usr/bin/env python3
"""Retain actual C events from completed DP-1511 batch-4 runs, never a live log."""
import csv
import json
import pathlib

HERE = pathlib.Path(__file__).resolve().parent
ARCHIVE = pathlib.Path.home() / 'Archives/darkpawns/oracle-runs/2026-11-09'
SEEDS = json.loads((HERE / 'original-seeds.json').read_text())
NEEDLES = {
 'aid-failure-depth': ['You fumble and ruin the bandages.', 'Aidfail fumbles with some bandaging and drops it all over the place!'],
 'carve-depth': ["You don't have anything to carve with!", 'You carve up some meat from the the corpse of fish.', 'Carvegod carves up some meat from the the corpse of fish.', 'a piece of meat'],
 'clan-applicant-depth': ["You've applied to the clan!", 'You applied to Wolves'],
 'clan-depth': ['Clan renamed.', "You add to the clan's treasure.", 'Your clan is now private.', 'Clan deleted.'],
 'clan-member-depth': ['You are Leader (Rank 2) of Wolves', "You've been enrolled in the clan you chose!", "They're already in your clan.", "They're already in a clan.", 'You cannot kick out that person.', 'You cannot promote that person over your rank!'],
 'clan-plan-depth': ['This is the first plan.', 'It has two lines.', 'The replacement plan.', '(null)'],
 'clan-plan-mortal-depth': ['Enter the description, or plan for clan <<Wolves>>.', 'Mortal member plan.'],
 'fighter-wimpy-window-depth': ['You wimp out, and attempt to flee!', 'You are cornered and fail to retreat!', 'Hit points: 100(400)'],
 'idlist-depth': ['Huh?!?', 'Ok. Id list complete.'],
 'lifecycle-idle-rent-norent': ['You have been idle, and are pulled into a void.', '<CLOSE>', 'Nothing.', "Idleforcerent's tunic"],
 'lua-cast-spell-gates-depth': ['You can only cast this spell upon yourself!', 'You cannot cast this spell upon yourself!', "You can't cast this spell if you're not in a group!", 'You can only will this power upon yourself!', 'You cannot will this power upon yourself!', 'You cannot use this power if you are not in a group!', 'You dream about great magical powers.', 'You cannot concentrate while resting.', "You can't do this sitting!", "Impossible!  You can't concentrate enough!", "You can't do much of anything like this!"],
 'reagent-waterwalk-decoy-vnum-depth': ['You feel webbing between your toes.', 'a wand of invisibility'],
 'retaliation-flamestrike-save-depth': ['You call down a bolt of flame on him!', 'A bolt of flame shoots down from the heavens and engulfs an incubus!'],
 'spell-parser-item-output-gaps': ['You recite an important-looking scroll which dissolves.', 'Paper cut', 'Your the gem of control bathes you in a blinding glow!', 'Your the gem of control flares up with a blinding glow that surges toward Parserpeer!', "You can't use the gem of control like that.", 'It seems powerless.', 'Nothing seems to happen.', "Parsergod's a wand of invisibility flares up with a blinding glow that surges toward a loaf of bread!", 'A shimmering current leaps from a wand of invisibility toward a loaf of bread.', 'Parsergod bangs on a war gong, sending a resounding note through the air.', "You can't do this sitting!!", 'There is nothing to here to affect with an important-looking scroll.', "You can't do this sitting!", 'Parsergod quaffs a potion of lycanthropy, and lets out a lone, solitary howl...'],
 'spell-points-spoken-depth': ['You feel a lot better!', "PointsTarget's wounds glow and mend themselves.", 'You feel better.', 'You feel vitalized!', 'You feel invigorated!', 'You focus your mind on healing your body..', 'Your wounds mend themselves!', 'Your wounds mend beneath your hands!'],
 'violence-npc-switched-output-depth': ['You dodge!', 'A guard trainee dodges your attack!', 'You drag yourself to your feet.', 'A guard trainee scrambles to his feet!'],
 'wizard-file-output-depth': ['There is no such player.', 'Password changed to', 'Saved in file.', "Sorry, you can't do that."],
 'wizard-stat-clan-followers-depth': ['Clan : Wolves', 'Master is: <none>, Followers are: FoxtrotFollowXX, EchoFollowerXXX,', ' DeltaFollowerXX, CharlieFollower, BravoFollowerBB, AlphaFollowerAA'],
}

def checked_run(run):
    directory = ARCHIVE / run
    assert (directory / 'exit-code').exists(), ('unfinished', run)
    return directory


def dump(run, name):
    directory = checked_run(run)
    results = {a[1]: a[0] for line in (directory / 'results.tsv').read_text().splitlines() if len(a := line.split('\t')) >= 2}
    assert results[name] == 'PASS', (run, name, results[name])
    path = directory / 'census-dump' / (name + '.txt')
    lines = path.read_text().splitlines()
    assert lines[0] == 'normalized C oracle blocks:'
    return path, lines


def pick(path, lines, needle):
    events = []
    block = ''
    for i, line in enumerate(lines):
        if line.startswith('--- ['):
            block = line
        elif needle in line:
            events.append({'line': line, 'dump': str(path), 'dump_line': i + 1, 'block': block})
    assert events, (path, needle)
    return events

before_rows = []
for name, seeds in SEEDS.items():
    for seed in seeds:
        directory = checked_run(f'sol-dp1511-b4-before-seed{seed}')
        results = {a[1]: a[0] for line in (directory / 'results.tsv').read_text().splitlines() if len(a := line.split('\t')) >= 2}
        assert results[name] == 'FAIL', (name, seed, results[name])
        paths = list(directory.glob(f'initial-logs/*/{name}.attempt1.log'))
        assert len(paths) == 1, paths
        refusal = next(line for line in paths[0].read_text().splitlines() if 'setup-refused:' in line)
        before_rows.append([name, seed, refusal, str(paths[0])])

records = []
outcomes = []
for name, needles in NEEDLES.items():
    run = 'sol-dp1511-b4-final'
    path, lines = dump(run, name)
    events = [event for needle in needles for event in pick(path, lines, needle)]
    records.append({'scenario': name, 'verdict': 'Repaired', 'run': run, 'events': events})
    for seed in SEEDS[name]:
        seed_run = run if seed == 1 else f'sol-dp1511-b4-after-seed{seed}'
        if seed != 1 and name in ['retaliation-flamestrike-save-depth', 'wizard-file-output-depth', 'spell-points-spoken-depth']:
            seed_run = f'sol-dp1511-b4-witness-repair-seed{seed}'
        seed_path, seed_lines = dump(seed_run, name)
        text = '\n'.join(seed_lines)
        if name == 'retaliation-flamestrike-save-depth':
            outcome = 'applied' if 'You call down a bolt of flame on him!' in text else 'saved'
            assert outcome != 'saved' or 'Nothing seems to happen.' in text
            assert 'An incubus gives you a pound.' in text or 'missed the joke that an incubus' in text
        else:
            outcome = 'witnessed'
            assert all(needle in text for needle in needles), (name, seed)
        outcomes.append([name, seed, outcome, str(seed_path)])

# The repaired overflow comparison is intentionally red and outside the corpus.
red = checked_run('sol-dp1511-b4-witness-repair-seed1')
path = red / 'census-dump/clan-plan-overflow.txt'
lines = path.read_text().splitlines()
records.append({'scenario': 'clan-plan-overflow', 'verdict': 'Go disagrees with C',
                'run': red.name, 'events': pick(path, lines, 'String too long.  Last line skipped.')})

warmups = []
for name, needle in [('lifecycle-idle-rent-norent', 'You wear a testing cloak around your body.'),
                     ('reagent-waterwalk-decoy-vnum-depth', '1) Namelist : frog leg decoy'),
                     ('violence-npc-switched-output-depth', 'Pos: Sitting')]:
    directory = checked_run('sol-dp1511-b4-final')
    paths = list(directory.glob(f'initial-logs/*/{name}.capture1/c-warmup.txt'))
    assert len(paths) == 1, paths
    path = paths[0]
    warmups += [{'scenario': name, **event} for event in pick(path, path.read_text().splitlines(), needle)]

# R5j: absence checks are paired with actual inventory/roundtrip output above.
_, lines = dump('sol-dp1511-b4-final', 'reagent-waterwalk-decoy-vnum-depth')
assert not any('A frog leg turns into dust' in line for line in lines)
assert sum('a wand of invisibility' in line for line in lines) == 2
_, lines = dump('sol-dp1511-b4-final', 'lifecycle-idle-rent-norent')
assert not any('testing cloak' in line or 'piece of fruit' in line for line in lines)
(HERE / 'witnesses.json').write_text(json.dumps({'cases': sorted(records, key=lambda r: r['scenario']), 'warmups': warmups}, indent=2) + '\n')
for filename, header, rows in [
 ('before-refusals.tsv', ['scenario', 'seed', 'setup_refusal', 'attempt_log'], before_rows),
 ('seed-outcomes.tsv', ['scenario', 'seed', 'outcome', 'C_dump'], outcomes),
 ('triage.tsv', ['scenario', 'verdict', 'original_seeds', 'C_witness', 'dump', 'dump_line'],
  [[r['scenario'], r['verdict'], ','.join(map(str, SEEDS[r['scenario']])), r['events'][0]['line'], r['events'][0]['dump'], r['events'][0]['dump_line']] for r in sorted(records, key=lambda r: r['scenario'])]),
]:
    with (HERE / filename).open('w') as out:
        writer = csv.writer(out, delimiter='\t', lineterminator='\n')
        writer.writerow(header)
        writer.writerows(rows)
print(f'{len(records)} cases; {sum(len(r["events"]) for r in records)} real C event lines; {len(warmups)} setup event lines')
