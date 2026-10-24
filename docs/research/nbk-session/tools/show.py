import json,sys
for e in json.load(open(sys.argv[1])):
    if 'in' in e: print(f"\n>>> {e['in']!r}")
    else: print(e['out'].replace('\x1b','\\e'), end='')
