#!/usr/bin/env python3
"""Extract assistant file-writes to darkpawns paths from OpenClaw transcripts.

Privacy rule (brief): this script emits ONLY metadata and the written code.
It never emits user messages, assistant prose, thinking, tool results, or any
other tool's arguments.

Per emitted record:
  session_id       UUID of the session (from the transcript file name)
  transcript_file  file name relative to the archive root (or 'mac-mini:'+path)
  tool_call_utc    the message record's timestamp, normalised to UTC ISO-8601
  model            the model on THAT assistant message (message.model)
  target_path      the file path the tool call wrote or edited
  tool             'write' or 'edit'
  code             write: arguments.content; edit: the newText(s), joined
  call_id          the tool call id (used for de-duplication only)

Usage:
  extract.py --root <dir> --out <jsonl> [--limit-files N]
  extract.py --schema           # key names only, no values (never used now)

The archive holds full-file `.bak`/`.reset`/`.deleted` copies, so the same tool
call appears in several files; records are de-duplicated on (session_id, call_id).
"""
import argparse, collections, json, os, sys, datetime

DARK = 'darkpawns'
WRITE_TOOLS = {'write', 'edit'}


def to_utc(ts):
    if ts is None:
        return ''
    if isinstance(ts, (int, float)):
        return datetime.datetime.fromtimestamp(ts / 1000.0, datetime.timezone.utc) \
            .strftime('%Y-%m-%dT%H:%M:%SZ')
    s = str(ts).strip()
    try:
        dt = datetime.datetime.fromisoformat(s.replace('Z', '+00:00'))
        if dt.tzinfo is None:
            dt = dt.replace(tzinfo=datetime.timezone.utc)
        return dt.astimezone(datetime.timezone.utc).strftime('%Y-%m-%dT%H:%M:%SZ')
    except Exception:
        return s


def session_uuid(fname):
    base = os.path.basename(fname)
    for sep in ('.jsonl',):
        if sep in base:
            base = base.split(sep)[0]
            break
    return base


def emit_from_file(path, relname, sink, seen):
    """Yield records from one transcript file. Prefilter keeps it fast and
    never parses a line that cannot contain a darkpawns file write."""
    n = 0
    with open(path, 'r', encoding='utf-8', errors='replace') as fh:
        for line in fh:
            if 'darkpawns' not in line or 'toolCall' not in line:
                continue
            line = line.strip()
            if not line:
                continue
            try:
                obj = json.loads(line)
            except Exception:
                continue
            if obj.get('type') != 'message':
                continue
            msg = obj.get('message') or {}
            if msg.get('role') != 'assistant':
                continue
            model = msg.get('model') or ''
            ts = to_utc(obj.get('timestamp') or msg.get('timestamp'))
            sid = session_uuid(path)
            for c in msg.get('content') or []:
                if not isinstance(c, dict) or c.get('type') != 'toolCall':
                    continue
                if c.get('name') not in WRITE_TOOLS:
                    continue
                args = c.get('arguments') or {}
                target = args.get('path') or ''
                if DARK not in target:
                    continue
                if c.get('name') == 'write':
                    code = args.get('content') or ''
                else:
                    parts = []
                    for e in args.get('edits') or []:
                        if isinstance(e, dict) and e.get('newText') is not None:
                            parts.append(e['newText'])
                        elif isinstance(e, str):
                            # Some edit calls carry bare strings in `edits`.
                            parts.append(e)
                    code = '\n'.join(parts)
                key = (sid, c.get('id') or '')
                if key in seen:
                    continue
                seen.add(key)
                sink.write(json.dumps({
                    'session_id': sid,
                    'transcript_file': relname,
                    'tool_call_utc': ts,
                    'model': model,
                    'target_path': target,
                    'tool': c.get('name'),
                    'code': code,
                    'call_id': c.get('id') or '',
                }) + '\n')
                n += 1
    return n


def emit_claude_md(path, relname, sink, seen):
    """Emit (session, timestamp, tool) for tool calls that read CLAUDE.md.
    Metadata only; no arguments, no content."""
    n = 0
    with open(path, 'r', encoding='utf-8', errors='replace') as fh:
        for line in fh:
            if 'CLAUDE.md' not in line or 'toolCall' not in line:
                continue
            try:
                obj = json.loads(line)
            except Exception:
                continue
            if obj.get('type') != 'message':
                continue
            msg = obj.get('message') or {}
            if msg.get('role') != 'assistant':
                continue
            sid = session_uuid(path)
            for c in msg.get('content') or []:
                if not isinstance(c, dict) or c.get('type') != 'toolCall':
                    continue
                if c.get('name') not in ('read', 'exec'):
                    continue
                args = c.get('arguments') or {}
                blob = (args.get('path') or '') + ' ' + (args.get('command') or '')
                if 'CLAUDE.md' not in blob:
                    continue
                key = (sid, c.get('id') or '')
                if key in seen:
                    continue
                seen.add(key)
                sink.write(json.dumps({
                    'session_id': sid, 'transcript_file': relname,
                    'tool_call_utc': to_utc(obj.get('timestamp')),
                    'tool': c.get('name'), 'call_id': c.get('id') or '',
                }) + '\n')
                n += 1
    return n


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--root', required=True)
    ap.add_argument('--out', required=True)
    ap.add_argument('--prefix', default='', help="prefix for transcript_file, e.g. 'mac-mini:'")
    ap.add_argument('--limit-files', type=int, default=0)
    ap.add_argument('--claude-md', action='store_true',
                    help='emit CLAUDE.md reads instead of file writes')
    args = ap.parse_args()
    emit = emit_claude_md if args.claude_md else emit_from_file

    seen = set()
    files = 0
    total = 0
    sink = sys.stdout if args.out == '-' else open(args.out, 'w')
    try:
        for dirpath, _, names in os.walk(args.root):
            for nm in sorted(names):
                if '.jsonl' not in nm:
                    continue
                files += 1
                if args.limit_files and files > args.limit_files:
                    break
                full = os.path.join(dirpath, nm)
                rel = args.prefix + os.path.relpath(full, args.root)
                try:
                    total += emit(full, rel, sink, seen)
                except Exception as e:
                    print('ERR', rel, e, file=sys.stderr)
            if args.limit_files and files > args.limit_files:
                break
    finally:
        if sink is not sys.stdout:
            sink.close()
    print(f'files={files} writes={total}', file=sys.stderr)


if __name__ == '__main__':
    main()
