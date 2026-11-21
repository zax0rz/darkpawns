# For every row citing src/<file>:<line>[-<end>], check:
#  resolve: the file exists and the lines are in range;
#  anchor: if the row quotes a C string (>=8 chars, in "..." or `...`, not a bare
#          identifier) that
#          occurs in that C file, is one occurrence inside the cited range?
import os, re, sys, glob, csv, collections
ROOT = sys.argv[1]; OUT = sys.argv[2]
src = {}
def lines(f):
    if f not in src:
        p = os.path.join(ROOT, "src", f)
        src[f] = open(p, errors="replace").read().split("\n") if os.path.exists(p) else None
    return src[f]
CITE = re.compile(r'(?:src/)?([a-z_0-9.-]+\.(?:c|h)):(\d+)(?:-(\d+))?')
ANCH = re.compile(r'"([^"\n]*)"|`([^`\n]*)`')
IDENT = re.compile(r'^[A-Za-z_][A-Za-z0-9_]*(\(\))?$')
def norm(s): return s.replace("\\r", "").replace("\\n", "").replace('\\"', '"').strip()
sources = sorted(glob.glob(f"{ROOT}/docs/fidelity/depth/*.tsv")) + sorted(glob.glob(f"{ROOT}/docs/research/test-audit/*/ledger.tsv")) + [f"{ROOT}/docs/research/EVIDENCE_LEDGER.tsv"]
stats = collections.Counter(); out = []
for path in sources:
    rel = os.path.relpath(path, ROOT)
    for ln, row in enumerate(open(path, errors="replace"), 1):
        cites = CITE.findall(row)
        if not cites: continue
        anchors = [norm(a or b) for a, b in ANCH.findall(row)]
        anchors = [a for a in anchors if not CITE.search(a) and len(a) >= 8 and not IDENT.match(a)]
        for f, a, b in cites:
            L = lines(f); s = int(a); e = int(b) if b else s
            if L is None: res = "no-such-file"; actual = ""
            elif s < 1 or e > len(L) or s > e: res = "out-of-range"; actual = f"file has {len(L)} lines"
            else:
                res = "resolves"; actual = ""
                hit_any = False
                for an in anchors:
                    occ = [i + 1 for i, t in enumerate(L) if an in t]
                    if not occ: continue
                    hit_any = True
                    if any(s - 2 <= o <= e + 2 for o in occ): res = "anchor-match"; actual = ""; break
                    res = "anchor-drift"; actual = f"{an[:50]!r} at {f}:{','.join(map(str, occ[:4]))}"
                if not hit_any and anchors and res == "resolves": res = "resolves-no-anchor-in-file"
            stats[res] += 1
            if res not in ("anchor-match", "resolves", "resolves-no-anchor-in-file"):
                out.append([rel, str(ln), f"{f}:{a}" + (f"-{b}" if b else ""), res, actual])
with open(OUT, "w") as o:
    w = csv.writer(o, delimiter="\t", lineterminator="\n"); w.writerow(["source", "row_line", "cite", "result", "detail"]); w.writerows(out)
print(dict(stats))
