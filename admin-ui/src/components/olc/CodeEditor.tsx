import { useEffect, useRef } from 'react';
import { EditorState } from '@codemirror/state';
import { EditorView, keymap, lineNumbers, highlightActiveLineGutter, drawSelection } from '@codemirror/view';
import { defaultKeymap, history, historyKeymap } from '@codemirror/commands';
import { StreamLanguage } from '@codemirror/language';
import { lua } from '@codemirror/legacy-modes/mode/lua';

interface CodeEditorProps {
  value: string;
  language: 'lua' | 'text';
  readOnly?: boolean;
  label: string;
  onChange: (value: string) => void;
}

const luaLanguage = StreamLanguage.define(lua);

const paperTheme = EditorView.theme({
  '&': { minHeight: '34rem', backgroundColor: 'var(--color-paper)', color: 'var(--color-ink)' },
  '.cm-content': { caretColor: 'var(--color-accent)' },
  '.cm-scroller': { fontFamily: '"JetBrains Mono", monospace', fontSize: '13px', lineHeight: '1.65' },
  '.cm-gutters': { backgroundColor: 'var(--color-paper-deep)', color: 'var(--color-ink-muted)', borderRight: '1px solid var(--color-rule)' },
  '.cm-activeLineGutter': { backgroundColor: 'color-mix(in srgb, var(--color-accent) 8%, transparent)' },
  '&.cm-focused': { outline: '2px solid var(--color-accent)', outlineOffset: '-2px' },
});

// CodeEditor is CodeMirror 6 with only the legacy Lua stream mode: no
// language server, no autocomplete, no kitchen sink. The editor owns its
// document while typing; a value change it did not originate (a reload, a
// conflict resolution) replaces the document wholesale.
export function CodeEditor({ value, language, readOnly = false, label, onChange }: CodeEditorProps) {
  const host = useRef<HTMLDivElement>(null);
  const view = useRef<EditorView | null>(null);
  const latestValue = useRef(value);
  const onChangeRef = useRef(onChange);

  // Declared before the mount effect so a remount (readOnly flipped after a
  // listing loads) opens on the current text, not the first render's.
  useEffect(() => {
    latestValue.current = value;
    onChangeRef.current = onChange;
  });

  useEffect(() => {
    if (!host.current) return;
    const editor = new EditorView({
      parent: host.current,
      state: EditorState.create({
        doc: latestValue.current,
        extensions: [
          lineNumbers(),
          highlightActiveLineGutter(),
          drawSelection(),
          history(),
          keymap.of([...defaultKeymap, ...historyKeymap]),
          EditorState.readOnly.of(readOnly),
          EditorView.editable.of(!readOnly),
          EditorView.contentAttributes.of({ 'aria-label': label }),
          EditorView.lineWrapping,
          ...(language === 'lua' ? [luaLanguage] : []),
          EditorView.updateListener.of((update) => {
            if (update.docChanged) onChangeRef.current(update.state.doc.toString());
          }),
          paperTheme,
        ],
      }),
    });
    view.current = editor;
    return () => {
      editor.destroy();
      view.current = null;
    };
  }, [language, readOnly, label]);

  useEffect(() => {
    const editor = view.current;
    if (!editor) return;
    const current = editor.state.doc.toString();
    if (current !== value) {
      editor.dispatch({ changes: { from: 0, to: current.length, insert: value } });
    }
  }, [value]);

  return <div ref={host} className="overflow-hidden border border-rule" />;
}
