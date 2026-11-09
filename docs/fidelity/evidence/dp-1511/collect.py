#!/usr/bin/env python3
"""Extract reviewed refusal literals from src; line selections are audited inputs."""
import json
import pathlib
import re
root = pathlib.Path(__file__).resolve().parents[4]
# Each site was read from src in this session. Duplicate text retains its first
# citation; additional occurrences remain visible in the site inventory.
sites = {
'act.item.c': '''56 90 92 96 105 109 111 118 122 142 146 162 167 172 203 228 233 243 259 262 299 309 319 335 338 358 360 372 384 418 421 436 438 488 583 596 610 620 632 676 682 687 692 715 725 740 776 791 815 819 1434 1435 1436 1437 1438 1439 1440 1441 1442 1443 1444 1445 1446 1447 1448 1449 1450 1451 1452 1453 1454 1455 1461 1482 1488 1494 1500 1508 1575 1595 1602 1618 1624 1639 1647 1668 1692 1705 1720 1722 1741 1756 1761 1774 1783''',
'olc.c':'113 157 243',
'act.other.c':'''116 118 126 127 130 210 645 701 714 722 953 959 966 1196 1408 1418 1426 1431 1483 1552 1556 1563 1567 1571 1573 1575 1577 1579 1581 1734 1739 1744''',
'act.movement.c':'''703 718 721 121 129 150 152 157 159 174 176 182 188 320 328 333 378 391 401 409 421 607 618 621 624 627 630 633 745 748 756 759 790 796 825 828 849 855 857 859 870 874 900 905 909 914 920''',
'config.c':'93 94',
'shop.c':'520 527 918 920',
'boards.c':'221 225 229 241 254 290 309 334 338 342 349 354 358 385 389 394 400 406 414 419',
'handler.c':'705 715',
'modify.c':'309 316 321 325 329',
'act.wizard.c':'''1123 232 1005 1011 1016 1020 1025 1048 1053 1069 1144 1146 1150 1152 1160 1307 1363 1378 1518 1523 1527 1668 1681 2091 2095 2098 2141 2145 2693 2698 2709 2818 2845 2859 2881 2899 2936 2958 2975 2982 3007 3048 3093 3250 3287 3298''',
'spell_parser.c':'''841 847 856 859 862 865 868 875 881 883 889 891 897 899 939 945 948 980 1040 1045 1056 1059 1066 1068 1074 1076 1101 419 424 432 441 444 589 641 663 668 676''',
'house.c':'394 406 410 421 426 436 447 489 504 525',
'clan.c':'''136 141 147 170 176 184 189 193 197 201 254 258 299 305 317 353 359 363 369 373 422 426 479 482 529 532 535 555 587 619 658 695 699 707 711 929 932 935 943 946 949 959 970 980 1025 1030 1038 1052 1102 1107 1114 1119 1124 1191 1195 1203 1209 1213 1309 1314 1321 1369 1374 1381''',
'act.offensive.c':'109 111 118 129 146 150 152 154 429 436 446 452 457 463 469 1463 1469 1475 1481 1490 1496',
'act.informative.c':'923 927 933 939 1014 1074 1107 1111 1118 2486',
'ban.c':'221 233',
'interpreter.c':'1011 1016',
'objsave.c':'1038 1094 1099 1112 1142 1146 1150',
'act.comm.c':'769 775 780 908 914 916 918 920 922 924 928 1039 1044 1049 1060 1068 1073 1079 1091 1093 1095',
}
rows=[]
for filename, numbers in sites.items():
    lines=(root/'src'/filename).read_text().splitlines(True)
    for number in map(int,numbers.split()):
        tail=''.join(lines[number-1:])
        if not re.match(r'\s*(?:.*?send_to_char|stc|act|sprintf|char \*|")',lines[number-1]):
            print('INVALID SITE',filename,number)
            continue
        if '"' not in lines[number-1]:
            raise ValueError((filename, number, 'site must start a literal'))
        # collect adjacent C string literals in this statement (not subsequent arguments).
        match=re.search(r'"(?:[^"\\]|\\.)*"(?:\s*"(?:[^"\\]|\\.)*")*',tail)
        if not match:
            print('NO LITERAL',filename,number)
            continue
        literal=''.join(json.loads(x) for x in re.findall(r'"(?:[^"\\]|\\.)*"',match[0]))
        rows.extend((f'src/{filename}:{number}',line.strip()) for line in literal.splitlines() if line.strip())
# Shop refusal text passes through do_tell/do_say. The initial %s is
# the recipient argument, removed by do_tell's half_chop, not displayed.
shop = (root/'src/shop.c').read_text().splitlines(True)
for number in [487, 494, 560, 562, 564, 566, 568]:
    match=re.search(r'"(?:[^"\\]|\\.)*"', ''.join(shop[number-1:]))
    text=json.loads(match[0])
    assert text.startswith('%s '), (number,text)
    rows.append((f'src/shop.c:{number}; src/act.comm.c:876', "$n tells you, '"+text[3:]+"'"))
header=(root/'src/shop.h').read_text().splitlines()
for number in [190,191,192,194,195,196]:
    text=json.loads(re.search(r'".*"',header[number-1])[0])
    wrapper = "$n tells you, '"
    if number in [190,191,192,194]:
        speech = {'!':'exclaims','?':'asks','.':'states'}.get(text[-1],'says')
        wrapper = "$n " + speech + ", '"
    rows.append((f'src/shop.h:{number}; src/shop.c:74-138; src/act.comm.c:787-801,876',wrapper+text+"'"))
path=root/'docs/fidelity/evidence/dp-1511/refusal-sites.tsv'
path.write_text('c_site\ttemplate\n'+''.join(f'{c}\t{json.dumps(t,ensure_ascii=False)}\n' for c,t in rows))
print('extracted',len(rows),'sites to',path)

# Source inventory retains unreachable strings, but they are not gate entries:
# perform_wear refuses WEAR_LIGHT before the array and advances paired slots
# past sentinel entries; its second wield-bit test repeats its first gate.
# do_stand's sleeping arm is blocked by cmd_info minimum POS_RESTING.
excluded = {
    "That's private property -- no trespassing!": "src/act.wizard.c:229-236 is commented out",
    "You're not godly enough for that room.": "home's find_target_room already returns on the identical GODROOM level condition (src/act.wizard.c:219-222,3277-3288; src/structs.h:612-613)",
    "Sorry, the board isn't working.": "requires a NULL counted heading; normal writes allocate it before counting (src/boards.c:249-271), loader resets zero heading length and otherwise always allocates (src/boards.c:503-514); CREATE aborts on allocation failure (src/utils.h:173-177)",
    "That message appears to be screwed up.": "same NULL counted-heading invariant in Board_display_msg/Board_remove_msg (src/boards.c:353-355,393-395); source inventory retained",
    "$n tells you, 'Something screwy only gave you %d.'": "unreachable final else: every while-exit condition is handled by prior partial-buy arms (src/shop.c:532-569); same_obj is checked before this chain",
    "You're in no position to go ANYWHERE!": "NPC direction path is standing before do_simple_move: do_move registration plus follower guard (src/interpreter.c:314-320; src/act.movement.c:344-349,358-365). Other callers such as flee are outside inventoried direction verbs",
    "You can't see anything but stars!": "look registration POS_RESTING preempts its internal below-SLEEPING position arm (src/interpreter.c:532; src/act.informative.c:1106-1108). No hollow warmup incident justifies expanding position refusals (brief item4)",
    "Deposit how much?": "src/clan.c:889-893 returns on empty input before inner bank prompt; immortal first token cannot be empty for a resolved clan",
    "Withdraw how much?": "same initial-empty bank guard at src/clan.c:889-893",
    "Set it to how much?": "src/clan.c:996-998 returns on empty input before inner money prompt",
    "Set how many ranks?": "src/clan.c:1073-1075 returns on empty input before inner ranks prompt",
    "Set to which level?": "src/clan.c:1280-1282 returns on empty input before inner application prompt",
    "Set the privilege to which rank?": "src/clan.c:1341-1343 returns on empty input before inner privilege prompt",
    "You can only quaff potions.": "SCMD_QUAFF branch src/act.other.c:951-955; inventoried do_use verb is recite (SCMD_RECITE)",
    "You can't seem to figure out how to use it.": "SCMD_USE branch src/act.other.c:963-970 is outside inventoried recite command path",
    "Try holding it.(?)": "second line of same SCMD_USE-only refusal src/act.other.c:966-967",
    "You don't seem to have the proper key.": "lock/unlock-only guard src/act.movement.c:631-633; inventoried door verbs are open/close",
    "You can't do that!": "do_invis NPC arm is preempted by immortal-command NPC gate (src/interpreter.c:519,921-922); no direct do_invis callers",
    "You cannot change that.": "src/act.wizard.c:2981-2984 is commented out",
    "That can only be done to a beast!": "src/act.wizard.c:2542-2604 has only PC/BOTH entries, no NPC-only field",
    "Bad clan banking call, please report to a God.": "do_clan passes only CB_WITHDRAW/CB_DEPOSIT (src/clan.c:1564,1569)",
    "Problem in command, please report.": "fixed dispatch: src/clan.c:1525,1530,1564,1569 cannot reach bank/money default modes",
    "You have to wake up first.": "do_sit/do_rest minimum POS_RESTING excludes sleeping before their handlers (src/interpreter.c:633,703)",
    "Already wearing TAKE object - Tell a god.": "src/act.item.c:1459-1470; WEAR_LIGHT=ITEM_WEAR_TAKE=0 (src/structs.h:390,446)",
    "YOU SHOULD NEVER SEE THIS MESSAGE.  PLEASE REPORT.": "src/act.item.c:1465-1470 skips occupied first paired slots",
    "You can't wield that.": "src/act.item.c:1459-1463 already checks the same wear bit before src/act.item.c:1480",
    "You have to wake up first!": "src/interpreter.c:733 (stand minimum POS_RESTING)",
}
# The legacy rent receptionists have declarations but no special assignment
# anywhere in src (src/spec_assign.c:89-90; src/objsave.c:1199-1207).
# The registered rent command is do_not_here (src/interpreter.c:646).
for cite, text in rows:
    if cite.startswith('src/objsave.c:'):
        excluded[text] = "unassigned receptionist/cryogenicist path; src/spec_assign.c:89-90 declarations only, src/objsave.c:1199-1207 wrappers only; rent dispatch src/interpreter.c:646"
(root/'docs/fidelity/evidence/dp-1511/excluded-refusals.json').write_text(json.dumps(excluded,indent=2)+'\n')
# Keep the original exact table byte-for-byte; generate the expanded inventory.
original = (root/'internal/oraclediff/setuprefusal.go').read_text()
seen = {json.loads(x) for x in re.findall(r'\{("(?:[^"\\]|\\.)*")', original)}
exact, patterns = [], []
for cite, text in rows:
    if text in seen or text in excluded:
        continue
    seen.add(text)
    (patterns if '%' in text or '$' in text else exact).append((text, cite))
output = 'package oraclediff\n\n// DP-1511: generated by docs/fidelity/evidence/dp-1511/collect.py.\n// Existing DP-1456 exact entries stay intact.\nvar setupRefusalAdditionalLines = []setupRefusal{\n'
for text, cite in exact:
    output += f'\t{{{json.dumps(text)}, {json.dumps(cite)}}},\n'
output += '}\n\nvar setupRefusalPatterns = []setupRefusalPattern{\n'
for text, cite in patterns:
    output += f'\trefusalPattern({json.dumps(text)}, {json.dumps(cite)}),\n'
output += '}\n'
(root/'internal/oraclediff/setuprefusal_inventory.go').write_text(output)
print(len(exact), 'additional exact entries;', len(patterns), 'patterns')
