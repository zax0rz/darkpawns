#!/usr/bin/env python3
"""Build v2/attribution.tsv from v1's table plus a match_strength column.

match_strength:
  mistake-line     the matched line is part of the mistake itself -- the wrong
                   message, condition or invented logic that the fix changed.
  function-created the matched line is a signature, declaration or call inside
                   the function the mistake lives in, but not the wrong line.
  generic          a line common enough to be written independently, or a line
                   from a different function than the mistake.

Retry overrides: for each v1 match that was not mistake-line, a second search
was run for a write containing the mistake's own line (a line the fix removed).
Four rows found one (G006, S068, S078, S083); the rest did not.
"""
import csv, json, os, re

V1 = '/home/zach/dp-research/attribution/attribution.tsv'
OUT = '/home/zach/dp-research/attribution/v2/attribution.tsv'
CORPORA = ['/tmp/attr/nuc_extract.jsonl', '/tmp/attr/macmini_extract.jsonl',
           '/tmp/attr/macmini_orphan.jsonl']

STRENGTH = {
 # mistake-line: the fix changed this exact line and it is the mistake
 'G044': 'mistake-line', 'G077': 'mistake-line', 'S042': 'mistake-line',
 'S057': 'mistake-line', 'S068': 'mistake-line', 'S078': 'mistake-line',
 'S083': 'mistake-line',
 # function-created: the function/locus, not the wrong line
 'G006': 'function-created', 'G013': 'function-created', 'G032': 'function-created',
 'G046': 'function-created', 'G084': 'function-created', 'S001': 'function-created',
 'S029': 'function-created', 'S033': 'function-created', 'S054': 'function-created',
 # generic: a different function, or a line written independently
 'G067': 'generic', 'S028': 'generic',
}

# id -> (session_id, tool_call_utc, model, matched_line) found by the retry pass
RETRY = {
 'G006': ('9927d65c-a28e-4d67-964c-58cd63f39653', '2026-04-26T20:19:34Z',
          'glm-5-turbo', 'castIdentify(level, ch, cvict, ovict)'),
 'S068': ('8e4697db-74af-409e-bd59-293e5cc5514c', '2026-04-25T00:06:30Z',
          'deepseek-chat', 'item, ok := p.Equipment.GetItemInSlot(SlotLight)'),
 'S078': ('9927d65c-a28e-4d67-964c-58cd63f39653', '2026-04-26T20:21:07Z',
          'glm-5-turbo', 'sendToCaster(ch, "You feel informed:\\r\\n")'),
 'S083': ('8e4697db-74af-409e-bd59-293e5cc5514c', '2026-04-25T00:12:20Z',
          'deepseek-chat', 'item.Prototype.Values[1]--'),
}

RETRY_NOTE = ('retried for the mistake\'s own line and kept the stronger match'
              if True else '')


def norm(s):
    return re.sub(r'\s+', ' ', s).strip()


def find_write(sid, utc, line):
    want = norm(line).lower()
    for p in CORPORA:
        if not os.path.exists(p):
            continue
        for raw in open(p):
            r = json.loads(raw)
            if r['session_id'] != sid or r['tool_call_utc'] != utc:
                continue
            if want in norm(r['code']).lower():
                return r
    return None


def main():
    rows = list(csv.DictReader(open(V1), delimiter='\t'))
    cols = list(rows[0].keys()) + ['match_strength']
    out = []
    for r in rows:
        rec = dict(r)
        rid = r['id']
        if rid in RETRY:
            sid, utc, model, line = RETRY[rid]
            w = find_write(sid, utc, line)
            if w is None:
                raise SystemExit('retry write not found for ' + rid)
            rec.update({'session_id': sid, 'transcript_file': w['transcript_file'],
                        'tool_call_utc': utc, 'model': model,
                        'match_type': 'verbatim' if line in w['code'] else 'whitespace',
                        'matched_line': line})
            old = r['notes']
            rec['notes'] = (old + ' ' if old else '') + RETRY_NOTE
        rec['match_strength'] = STRENGTH.get(rid, '')
        out.append(rec)
    with open(OUT, 'w') as fh:
        fh.write('\t'.join(cols) + '\n')
        for rec in out:
            fh.write('\t'.join(str(rec[c]).replace('\t', ' ').replace('\n', ' ')
                               for c in cols) + '\n')
    m = [r for r in out if r['status'] == 'matched']
    print('rows', len(out), 'matched', len(m))
    import collections
    print(collections.Counter(r['match_strength'] for r in m))


if __name__ == '__main__':
    main()
