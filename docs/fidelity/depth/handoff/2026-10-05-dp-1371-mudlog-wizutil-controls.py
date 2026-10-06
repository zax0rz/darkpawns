import pathlib, subprocess, sys
root = pathlib.Path(sys.argv[1]); root.mkdir(parents=True, exist_ok=True)
WS = 'pkg/session/wiz_system.go'
WP = 'pkg/session/wiz_player.go'
controls = [
 ('pardon-producer', WS,
  '\t\tgame.MudLog(fmt.Sprintf("(GC) %s pardoned by %s", target.player.Name, s.player.Name),\n\t\t\tgame.MudlogBrief, wizutilMudlogLevel(s.player), true)',
  '\t\t// omitted pardon producer',
  '^TestWizutilMudlogProducerBytesAndOrder$/^pardon$',
  '--- FAIL: TestWizutilMudlogProducerBytesAndOrder/pardon'),
 ('notitle-producer', WS,
  '\t\tgame.MudLog(fmt.Sprintf("(GC) Notitle %s for %s by %s.", onOff(newState), target.player.Name, s.player.Name),\n\t\t\tgame.MudlogNormal, wizutilMudlogLevel(s.player), true)',
  '\t\t// omitted notitle producer',
  '^TestWizutilMudlogProducerBytesAndOrder$/^notitle_both_directions_log_before_the_ack$',
  '--- FAIL: TestWizutilMudlogProducerBytesAndOrder/notitle_both_directions_log_before_the_ack'),
 ('squelch-producer', WS,
  '\t\tgame.MudLog(fmt.Sprintf("(GC) Squelch %s for %s by %s.", onOff(newState), target.player.Name, s.player.Name),\n\t\t\tgame.MudlogBrief, wizutilMudlogLevel(s.player), true)',
  '\t\t// omitted squelch producer',
  '^TestWizutilMudlogProducerBytesAndOrder$/^mute_both_directions_log_before_the_ack$',
  '--- FAIL: TestWizutilMudlogProducerBytesAndOrder/mute_both_directions_log_before_the_ack'),
 ('freeze-producer', WS,
  '\t\tgame.MudLog(fmt.Sprintf("(GC) %s frozen by %s.", target.player.Name, s.player.Name),\n\t\t\tgame.MudlogBrief, wizutilMudlogLevel(s.player), true)',
  '\t\t// omitted freeze producer',
  '^TestWizutilMudlogFreezeThawOrder$/^freeze_room_act_precedes_the_log$',
  '--- FAIL: TestWizutilMudlogFreezeThawOrder/freeze_room_act_precedes_the_log'),
 ('thaw-producer', WS,
  '\t\tgame.MudLog(fmt.Sprintf("(GC) %s un-frozen by %s.", target.player.Name, s.player.Name),\n\t\t\tgame.MudlogBrief, wizutilMudlogLevel(s.player), true)',
  '\t\t// omitted thaw producer',
  '^TestWizutilMudlogFreezeThawOrder$/^thaw_logs_before_every_message_and_before_flag_removal$',
  '--- FAIL: TestWizutilMudlogFreezeThawOrder/thaw_logs_before_every_message_and_before_flag_removal'),
 ('skillset-producer', WP,
  '\tgame.MudLog(fmt.Sprintf("%s changed %s\'s %s to %d.", s.player.Name, vict.Name, game.SkillCatalogName(skillNum), value),\n\t\tgame.MudlogBrief, -1, true)',
  '\t// omitted skillset producer',
  '^TestSkillsetMudlogIsFileOnly$',
  '--- FAIL: TestSkillsetMudlogIsFileOnly'),
 # Classifier controls: the proof must not be vacuous about type, threshold and
 # the file-only classification.
 ('pardon-wrong-type', WS,
  '\t\t\tgame.MudlogBrief, wizutilMudlogLevel(s.player), true)\n\tcase wizutilNotitle:',
  '\t\t\tgame.MudlogNormal, wizutilMudlogLevel(s.player), true)\n\tcase wizutilNotitle:',
  '^TestWizutilMudlogRecipientFilter$/^NRM_needs_normal_and_BRF_reaches_brief$',
  '--- FAIL: TestWizutilMudlogRecipientFilter/NRM_needs_normal_and_BRF_reaches_brief'),
 ('wizutil-wrong-threshold', WS,
  '\t\t\tgame.MudlogBrief, wizutilMudlogLevel(s.player), true)\n\tcase wizutilNotitle:',
  '\t\t\tgame.MudlogBrief, s.player.Level, true)\n\tcase wizutilNotitle:',
  '^TestWizutilMudlogRecipientFilter$/^actor_invisibility_raises_the_threshold$',
  '--- FAIL: TestWizutilMudlogRecipientFilter/actor_invisibility_raises_the_threshold'),
 ('skillset-broadcasts', WP,
  '\t\tgame.MudlogBrief, -1, true)',
  '\t\tgame.MudlogBrief, game.LVL_GOD, true)',
  '^TestSkillsetMudlogIsFileOnly$',
  '--- FAIL: TestSkillsetMudlogIsFileOnly'),
 # Ordering: C logs at modify.c:334, before SET_SKILL at :336. Moving the call
 # after the mutation must fail the log-time state probe.
 ('skillset-after-mutation', WP,
  '\tgame.MudLog(fmt.Sprintf("%s changed %s\'s %s to %d.", s.player.Name, vict.Name, game.SkillCatalogName(skillNum), value),\n\t\tgame.MudlogBrief, -1, true)\n\n\t// Step 10: SET_SKILL(vict, skill, value). Go stores skills by name string;\n\t// use the canonical spells[] display name (lowercased, matching how callers\n\t// key GetSkill/SetSkill — see spec_procs.go practice).\n\t// SkillStorageName carries C\'s catalog-to-key translations (DP-1342).\n\tcanonicalName := game.SkillStorageName(skillNum)\n\tvict.SetSkill(canonicalName, value)',
  '\t// Step 10: SET_SKILL(vict, skill, value). Go stores skills by name string;\n\t// use the canonical spells[] display name (lowercased, matching how callers\n\t// key GetSkill/SetSkill — see spec_procs.go practice).\n\t// SkillStorageName carries C\'s catalog-to-key translations (DP-1342).\n\tcanonicalName := game.SkillStorageName(skillNum)\n\tvict.SetSkill(canonicalName, value)\n\tgame.MudLog(fmt.Sprintf("%s changed %s\'s %s to %d.", s.player.Name, vict.Name, game.SkillCatalogName(skillNum), value),\n\t\tgame.MudlogBrief, -1, true)',
  '^TestSkillsetMudlogIsFileOnly$',
  '--- FAIL: TestSkillsetMudlogIsFileOnly'),
]
for name, path, before, after, testrun, marker in controls:
    p = pathlib.Path(path); original = p.read_text()
    if before not in original:
        sys.exit('missing mutation ' + name)
    cmd = ['go', 'test', './pkg/session', '-run', testrun, '-count=1']
    r = subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
    (root / (name + '-green.txt')).write_text(r.stdout)
    if r.returncode:
        sys.exit('initial green failed ' + name + '\n' + r.stdout)
    try:
        p.write_text(original.replace(before, after, 1))
        r = subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
        (root / (name + '-revert.txt')).write_text(r.stdout)
        if r.returncode == 0 or marker not in r.stdout or '[build failed]' in r.stdout:
            sys.exit('invalid red ' + name + '\n' + r.stdout)
    finally:
        p.write_text(original)
    r = subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
    (root / (name + '-restore.txt')).write_text(r.stdout)
    if r.returncode:
        sys.exit('restore failed ' + name + '\n' + r.stdout)
    print(name, '1 -> 0 -> 1', flush=True)
