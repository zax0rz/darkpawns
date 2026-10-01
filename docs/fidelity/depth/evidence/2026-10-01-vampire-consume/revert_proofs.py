#!/usr/bin/env python3
"""Assertion-only 0/1/0 proofs for the already implemented vampire gates."""
import hashlib
import subprocess
from pathlib import Path
source = Path("pkg/game/item_consumable.go")
original = source.read_text()
out = Path("/home/zach/Archives/darkpawns/oracle-runs/2026-10-01/dp-1371-p4-vampire-proofs")
out.mkdir(parents=True, exist_ok=True)
mutations = [
 ("drink-plr", "ch.HasPLRFlag(PlrVampire) && liqIndex", "ch.IsAffected(affVampire) && liqIndex"),
 ("drink-blood", "liqIndex != LiqBlood", "true"),
 ("drink-sunset", "(GetSunlight() == SunSet || GetSunlight() == SunDark)", "(GetSunlight() == SunDark)"),
 ("drink-dark", "(GetSunlight() == SunSet || GetSunlight() == SunDark)", "(GetSunlight() == SunSet)"),
 ("drink-day", "(GetSunlight() == SunSet || GetSunlight() == SunDark)", "true"),
 ("drink-gate", "ch.HasPLRFlag(PlrVampire) && liqIndex != LiqBlood", "false && liqIndex != LiqBlood"),
 ("drink-drunk", "GainCondition(ch, CondDrunk, (drunkAff*amount)/4)", "GainCondition(ch, CondDrunk, (drunkAff*amount)/400)"),
 ("drink-volume", "item.SetValue(1, item.GetValue(1)-amount)", "item.SetValue(1, item.GetValue(1))"),
 ("drink-weight", "item.SetWeight(item.GetWeight() - weight)", "item.SetWeight(item.GetWeight())"),
 ("drink-message", "The vampirism in your body is not satiated by mere %s...", "The vampirism in your body is satiated by mere %s..."),
 ("eat-plr", "ch.HasPLRFlag(PlrVampire) && (GetSunlight()", "ch.IsAffected(affVampire) && (GetSunlight()"),
 ("eat-sunset", "ch.HasPLRFlag(PlrVampire) && (GetSunlight() == SunSet || GetSunlight() == SunDark)", "ch.HasPLRFlag(PlrVampire) && (GetSunlight() == SunDark)"),
 ("eat-dark", "ch.HasPLRFlag(PlrVampire) && (GetSunlight() == SunSet || GetSunlight() == SunDark)", "ch.HasPLRFlag(PlrVampire) && (GetSunlight() == SunSet)"),
 ("eat-day", "ch.HasPLRFlag(PlrVampire) && (GetSunlight() == SunSet || GetSunlight() == SunDark)", "ch.HasPLRFlag(PlrVampire)"),
 ("eat-gate", "ch.HasPLRFlag(PlrVampire) && (GetSunlight() == SunSet || GetSunlight() == SunDark)", "false"),
 ("eat-message", "The vampirism in your body is not satiated by mere food...", "The vampirism in your body is satiated by mere food..."),
 ("eat-extraction", "ch.Inventory.RemoveItem(item)\n\t\tw.ExtractObject(item, ch.GetRoomVNum())", "// mutation: retain eaten food"),
 ("taste-consumption", "newVal := item.GetValue(0) - 1", "newVal := item.GetValue(0)"),
]
rows = ["mutation\tbefore\tmutant\tafter\tfailure"]
def run(name,stage):
 r = subprocess.run(["go","test","./pkg/game","-run","^TestVampireConsumableDepth$","-count=1"], stdout=subprocess.PIPE,stderr=subprocess.STDOUT,text=True)
 (out / f"{name}-{stage}.log").write_text(r.stdout)
 return r
try:
 for name, old, new in mutations:
  assert old in original, name
  source.write_text(original)
  before=run(name,"before")
  source.write_text(original.replace(old,new,1))
  mutant=run(name,"mutant")
  source.write_text(original)
  after=run(name,"after")
  assert before.returncode==0 and mutant.returncode==1 and after.returncode==0, name
  assert "--- FAIL: TestVampireConsumableDepth" in mutant.stdout and "build failed" not in mutant.stdout, name
  rows.append(f"{name}\t0\t1\t0\tassertion")
  print(name,"0 -> 1 -> 0",flush=True)
finally:
 source.write_text(original)
Path("docs/fidelity/depth/evidence/2026-10-01-vampire-consume/revert-triples.tsv").write_text("\n".join(rows)+"\n")
(out/"source.sha256").write_text(hashlib.sha256(source.read_bytes()).hexdigest()+"  pkg/game/item_consumable.go\n")
