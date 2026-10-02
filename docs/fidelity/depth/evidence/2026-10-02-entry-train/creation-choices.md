# entry.creation-choices

R1/R5g: src/interpreter.c:1721,1841-1858,1988-2105,1674-1690; src/class.c:122-166; src/constants.c:196-220. Confirmation, color, sex, race, class and hometown now use the first input byte after leading C whitespace. Race help retains its second-byte selector. Unknown help no longer adds an invented leading CRLF. All seven race selectors, five common base classes, human-only ninja, remort denials, race-help arms and invalid choices are covered.

The normalized paired vehicle entry-creation-choices-matrix passed (1/1 CLEAN) in dp-1371-entry-creation-choices. C output was inspected: every help arm prints its text/menu; non-human ninja and remort inputs refuse before the thief/hometown path. It is not raw ANSI help proof or exact RNG evidence. Real WebSocket JSON also reaches each first-byte stage, including human ninja and help suffixes.

R5h triples [0,1,0]: creation-choices (restore whole-string choices), race-choice (restore whole-string race), race-help (restore invented CRLF). All fail assertions. Replay revert_proofs.py; archive dp-1371-entry-train-proofs. Required gates passed in creation-choice-gates.

Other readers: charColor drives entry ANSI expansion and both saved PRF_COLOR bits; sex/race/class/hometown feed body initialization, roll_real_abils, class restrictions, starting skills/gear and newcomer relocation. No persistence is written until stats acceptance. Browser options remain presentation metadata; JSON free input and terminal input share the same handler. Stats accept/reroll is a separate case, and name lookup/security holds are unchanged.
