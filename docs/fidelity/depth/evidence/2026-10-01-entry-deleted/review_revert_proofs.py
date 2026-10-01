#!/usr/bin/env python3
"""Run assertion-only review repair triples in a disposable proof checkout."""
from pathlib import Path
import shutil, subprocess, sys
source=Path(__file__).resolve().parents[5]
proof=Path(sys.argv[1])
logs=Path(sys.argv[2]); logs.mkdir(parents=True,exist_ok=True)
files=subprocess.check_output(["git","diff","origin/main","--name-only"],cwd=source,text=True).splitlines()
files += ["pkg/admin/deleted_record_test.go"]
for name in set(files):
 if name.endswith(".go"):
  shutil.copyfile(source/name,proof/name)
admin=["pkg/admin/"+name for name in ["login.go","olc.go","olc_new_zone.go","fileedit.go"]]
mutations=[("admin-readers",admin,"./pkg/admin","TestDeletedAdmin"),
 ("clan-clear",["pkg/session/menu.go"],"./pkg/session","TestEntryDeletedMenuClanAndAliases"),
 ("alias-delete",["pkg/session/menu.go"],"./pkg/session","TestEntryDeletedMenuClanAndAliases")]
rows=["mutation\tbefore\tmutant\tafter\tfailure"]
for name,paths,package,test in mutations:
 saved={p:(proof/p).read_text() for p in paths}
 statuses=[]
 try:
  for stage in ["before","mutant","after"]:
   for p,s in saved.items():
    if stage=="mutant":
     if name=="admin-readers":
      s=subprocess.check_output(["git","show","a6b429f88:"+p],cwd=source,text=True)
     elif name=="clan-clear":
      assert "clearClan = index > 0" in s
      s=s.replace("clearClan = index > 0","clearClan = index > 0 && false")
     else:
      assert "game.DeleteAliases(name)" in s
      s=s.replace("game.DeleteAliases(name)",'game.DeleteAliases("")')
    (proof/p).write_text(s)
   out=subprocess.run(["go","test",package,"-run",test,"-count=1"],cwd=proof,capture_output=True,text=True)
   text=out.stdout+out.stderr
   (logs/(name+"-"+stage+".log")).write_text(text)
   expected=1 if stage=="mutant" else 0
   assert out.returncode==expected,(name,stage,out.returncode,text)
   if expected: assert "--- FAIL: Test" in text and "[build failed]" not in text
   statuses.append(str(out.returncode))
 finally:
  for p,s in saved.items(): (proof/p).write_text(s)
 rows.append(name+"\t"+"\t".join(statuses)+"\tassertion")
 print(name,*statuses,flush=True)
(source/"docs/fidelity/depth/evidence/2026-10-01-entry-deleted/review-revert-triples.tsv").write_text("\n".join(rows)+"\n")
