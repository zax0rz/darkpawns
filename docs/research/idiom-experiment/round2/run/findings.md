# Census blind spots

- Probe 0 (`findDoor`, `door-basic` prediction): the full census was clean after rechecking an unrelated infra row; changing the selected “I really don’t see how…” output byte was not detected.
- Probe 2 (`specNewbieZoneEntranceActor`, no scenario predicted): the full census was clean after rechecking an unrelated infra row; changing the selected private entry refusal byte was not detected.
- Probe 3 (`specNinelives`, no scenario predicted): the full census was clean; changing the selected revival output format byte was not detected.
- Probe 4 (`CmdSerpentKick`, `serpent-depth` prediction): the full census was clean after rechecking an unrelated infra row; changing the selected “Kick who?” response was not detected.
- Probe 8 (`specShopKeeperActor`, no scenario predicted): the full census was clean; changing the selected shopkeeper shout template byte was not detected.
- Probe 10 (`World.CastGateSpell`, no scenario predicted): the full census was clean; changing the selected gate-spell fallback message byte was not detected.
- Probe 11 (`World.AffectUpdate`, no scenario predicted): the full census was clean; changing the selected wear-off message format byte was not detected.
- Probe 15 (`weatherChangeLocked`, `weather-sky-transitions` prediction): the full census was clean; changing the selected lightning message byte was not detected.
- Probe 16 (`World.MpSound`, no scenario predicted): the full census was clean; changing the selected emote string byte was not detected.
- Probe 18 (`BanManager.DoUnban`, no scenario predicted): the full census was clean; changing the selected unban acknowledgement byte was not detected.
- Probe 19 (`AdminCommands.cmdWarn`, no scenario predicted): the full census was clean after rechecking an unrelated infra row; changing the selected offline-player message byte was not detected.
- Probe 21 (`World.DoReply`, no scenario predicted): the full census was clean; changing the selected linkless-recipient message byte was not detected.
- Probe 23 (`CmdSkills`, no scenario predicted): the first full census had an unrelated infra row and its recheck failed on that same unrelated scenario; the required full retry was clean. The selected skills header mutation was not detected.
- Probe 27 (`mobDisarm`, no scenario predicted): the full census was clean after rechecking an unrelated infra row; changing the selected disarm failure message byte was not detected.
- Probe 37 (`executeShoot`, `shoot-mob-hit-depth` prediction): the full census was clean after rechecking two unrelated infra rows; changing the selected “You decide to go investigate…” output byte was not detected.
- Probe 38 (`specMirror`, no scenario predicted): the full census was clean; changing the selected appearance message format byte was not detected.
- Probe 39 (`World.doTransform`, no scenario predicted): the full census was clean; changing the selected werewolf transformation act byte was not detected.
