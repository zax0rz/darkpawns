import json,sys
# usage: login.py name pw out.json cmd:wait ...
name,pw,out=sys.argv[1:4]
steps=[[None,2],[name,1],[pw,2],["",1],["1",2]]
for a in sys.argv[4:]:
    c,_,w=a.rpartition(':'); steps.append([c,float(w)])
json.dump(steps,open(out,'w'))
