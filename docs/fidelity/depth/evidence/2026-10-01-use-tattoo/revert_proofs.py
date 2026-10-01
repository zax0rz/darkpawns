#!/usr/bin/env python3
"""Run assertion-only 0 -> 1 -> 0 tattoo proofs in an isolated worktree."""
import pathlib
import subprocess
import sys

source, work, evidence = map(pathlib.Path, sys.argv[1:4])
files = ["pkg/game/other_economy.go", "pkg/spells/call_magic.go",
         "pkg/game/use_tattoo_depth_test.go", "pkg/session/use_tattoo_depth_test.go",
         "pkg/game/limits_condition.go"]
originals = {f: (source / f).read_text() for f in files}
for f, text in originals.items():
    (work / f).write_text(text)
evidence.mkdir(parents=True, exist_ok=True)
for f, text in originals.items():
    (evidence / pathlib.Path(f).name).write_text(text)

game = "pkg/game/other_economy.go"
magic = "pkg/spells/call_magic.go"
mutants = [
 ("blind", game, "!observer.IsAffected(affBlind)", "true"),
 ("dark", game, "!w.IsRoomDark(observer.GetRoom())", "true"),
 ("infravision", game, "observer.IsAffected(affInfravision)", "false"),
 ("holylight", game, "return observer.GetHolyLight() || (lightOK && invisOK)", "return lightOK && invisOK"),
 ("immortal-visibility", game, "return observer.GetHolyLight() || (lightOK && invisOK)", "return observer.GetLevel() >= combat.LVL_IMMORT || observer.GetHolyLight() || (lightOK && invisOK)"),
 ("wizinvis", game, "if observer.GetLevel() < caster.GetInvisLevel() {", "if false {"),
 ("hide-extension", game, "if observer.GetLevel() < caster.GetInvisLevel() {", "if caster.IsAffected(affHide) { return false }; if observer.GetLevel() < caster.GetInvisLevel() {"),
 ("nomagic-pers", game, 'name = "someone"', 'name = ch.Name'),

 ("eye-invisible-audience", game, "w.tattooCanSee(player, w.caster)", "true"),
 ("eye-sleeping-audience", game, "sendOk(player, false) && w.tattooCanSee(player, w.caster)", "w.tattooCanSee(player, w.caster)"),
 ("held-boundary", game, "!isASCIIAlpha(names[n])", "names[n] == ' '"),
 ("hourly-expiry", "pkg/game/limits_condition.go", "p.TatTimer--", "// omit decrement"),
 ("dispatch", game, 'if held == nil && strings.EqualFold(itemArg, "tattoo") {', 'if false {'),
 ("cooldown-gate", game, "if ch.TatTimer != 0 {", "if false {"),
 ("negative-cooldown", game, "if ch.TatTimer != 0 {", "if ch.TatTimer > 0 {"),
 ("singular", game, 'suffix = ""', 'suffix = "s"'),
 ("timer", game, "ch.TatTimer = 24", "ch.TatTimer = 23"),
 ("none-timer", game, 'ch.SendMessage("You don\'t have a tattoo.\\r\\n")', 'ch.SendMessage("You don\'t have a tattoo.\\r\\n")\nreturn true'),
 ("unsupported-timer", game, 'ch.SendMessage("Your tattoo can\'t be \'use\'d.\\r\\n")', 'ch.TatTimer = 24\nch.SendMessage("Your tattoo can\'t be \'use\'d.\\r\\n")'),
 ("charm-flag", game, "Flags:     engine.AFFCharm,", "Flags:     1 << 3,"),
 ("charm-duration", game, "Duration:  20,", "Duration:  19,"),
 ("quiet-follow", game, "AddFollowerQuietMob(mob, ch)", "// omit quiet follower"),
 ("skull-room-audience", game, 'w.tattooRoomAct(ch, mob, "$n\'s tattoo glows brightly for a second, and $N appears!")', 'Act(w, false, ch, mob, nil, nil, "$n\'s tattoo glows brightly for a second, and $N appears!", "", ToChar)'),
 ("skull-visibility", game, "ok && w.tattooCanSee(player, ch)", "ok"),
 ("eye-mapping", game, "w.castTattoo(ch, spells.SpellGreatPercept)", "w.castTattoo(ch, spells.SpellBless)"),
 ("ship-mapping", game, "w.castTattoo(ch, spells.SpellChangeDensity)", "w.castTattoo(ch, spells.SpellBless)"),
 ("angel-mapping", game, "w.castTattoo(ch, spells.SpellBless)", "w.castTattoo(ch, spells.SpellGreatPercept)"),
 ("effective-level", magic, "spellNum, 12, CastWand, world, true, true", "spellNum, getLevel(caster), CastWand, world, true, true"),
 ("extra-draw", magic, "spellNum, 12, CastWand, world, true, true", "spellNum, dprng.Number(12, 12), CastWand, world, true, true"),
 ("sitting", game, "if ch.GetPosition() == combat.PosSitting {", "if false {"),
 ("nomagic-bit", game, "room.HasFlag(tattooRoomNoMagic)", "room.HasFlag(20)"),
 ("immortal-exemption", game, "ch.GetLevel() < combat.LVL_IMMORT", "ch.GetLevel() <= combat.LVL_IMMORT"),
 ("nomagic-actor", game, 'ch.SendMessage("Your magic fizzles out and dies.\\r\\n")', 'ch.SendMessage("A magical force prevents you from casting here.\\r\\n")'),
 ("nomagic-psi", game, 'ch.SendMessage("Your will fades, disturbed by an unseen force.\\r\\n")', 'ch.SendMessage("Your magic fizzles out and dies.\\r\\n")'),
]

def run(name, want):
    result = subprocess.run(["go", "test", "./pkg/game", "./pkg/session",
                             "-run", "^Test(UseTattoo|CmdUseTattoo)", "-count=1"],
                            cwd=work, text=True, stdout=subprocess.PIPE,
                            stderr=subprocess.STDOUT, timeout=120)
    (evidence / (name + ".txt")).write_text(result.stdout)
    if result.returncode != want or "[build failed]" in result.stdout:
        raise RuntimeError(f"{name}: exit {result.returncode}, wanted {want}\n{result.stdout}")
    if want and "--- FAIL: Test" not in result.stdout:
        raise RuntimeError(f"{name}: no assertion failure")
    return result.returncode

rows = ["mutation\tbefore\tmutant\trestored"]
try:
    for file in (game, magic):
        baseline = subprocess.check_output(["git", "show", "HEAD:" + file], cwd=work, text=True)
        (work / file).write_text(baseline)
    run("origin-main-final-matrix", 1)
    for file in (game, magic):
        (work / file).write_text(originals[file])
    for name, file, old, new in mutants:
        before = run(name + "-before", 0)
        assert originals[file].count(old) == 1, (name, originals[file].count(old))
        (work / file).write_text(originals[file].replace(old, new, 1))
        mutant = run(name + "-mutant", 1)
        (work / file).write_text(originals[file])
        restored = run(name + "-restored", 0)
        rows.append(f"{name}\t{before}\t{mutant}\t{restored}")
        print(rows[-1], flush=True)
finally:
    for file, text in originals.items():
        (work / file).write_text(text)
    (evidence / "revert-results.tsv").write_text("\n".join(rows) + "\n")
