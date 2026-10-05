#!/usr/bin/env python3
"""Replay compiled step-1 identity revert triples in a disposable checkout."""
import argparse
import datetime
import pathlib
import subprocess

parser = argparse.ArgumentParser()
parser.add_argument('--output', type=pathlib.Path, required=True)
args = parser.parse_args()
args.output.mkdir(parents=True, exist_ok=True)
engine = pathlib.Path('pkg/combat/engine.go')
original = engine.read_text()
command = ['go', 'test', './pkg/game', './pkg/combat', '-run',
           '^(TestCombatBodyIdentityDuplicateEnrollment|TestCombatBodyParriedIsolation)$', '-count=1']


def run(label, want_failure):
    result = subprocess.run(command, text=True, stdout=subprocess.PIPE,
                            stderr=subprocess.STDOUT, check=False)
    (args.output / (label + '.txt')).write_text(result.stdout)
    if want_failure:
        assert result.returncode != 0 and '--- FAIL:' in result.stdout, result.stdout
        assert '[build failed]' not in result.stdout, result.stdout
    else:
        assert result.returncode == 0, result.stdout
    print(label, result.returncode, flush=True)


controls = {
    'attacker-name': [('if k.Attacker == attacker {',
                       'if k.Attacker.GetName() == attacker.GetName() {')],
    'member-name': [('existing != nil && existing == fighter',
                     'existing != nil && existing.GetName() == fighter.GetName()')],
    'round-name': [('make(map[Combatant]bool, len(fighters))',
                    'make(map[string]bool, len(fighters))'),
                   ('seen[fighter]', 'seen[fighter.GetName()]')],
    'target-name': [('return fighter.GetFightingBody()',
                     'target := fighter.GetFightingBody(); if target == nil {return nil}; '
                     'for _,candidate := range ce.combatOrder {'
                     'if candidate.GetName() == target.GetName() {return candidate}}; return nil')],
    'query-name': [('target := body.GetFightingBody()\n\treturn target, target != nil',
                    'target := body.GetFightingBody(); if target == nil {return nil,false}; '
                    'ce.mu.RLock(); defer ce.mu.RUnlock(); '
                    'for _,candidate := range ce.combatOrder {'
                    'if candidate.GetName() == target.GetName() {return candidate,true}}; return nil,false')],
    'bare-id': [('if k.Attacker == attacker {',
                'if k.Attacker.(interface{GetID() int}).GetID() == '
                'attacker.(interface{GetID() int}).GetID() {')],
    'parried-name': [('defenseAction := ce.parried[body]',
                     'var defenseAction string; for candidate,value := range ce.parried {'
                     'if candidate.GetName() == body.GetName() {defenseAction=value; break}}')],
}
(args.output / 'command.txt').write_text(' '.join(command) + '\n')
(args.output / 'HEAD.txt').write_text(subprocess.check_output(['git', 'rev-parse', 'HEAD'], text=True))
(args.output / 'started.txt').write_text(datetime.datetime.now(datetime.timezone.utc).isoformat())
try:
    run('green-before', False)
    for label, edits in controls.items():
        mutated = original
        for before, after in edits:
            assert before in mutated, (label, before)
            mutated = mutated.replace(before, after)
        engine.write_text(mutated)
        run(label + '-revert', True)
        engine.write_text(original)
        run(label + '-restore', False)
finally:
    engine.write_text(original)
