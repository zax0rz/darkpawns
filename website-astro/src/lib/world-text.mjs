// World-file prose is hard-wrapped near 80 columns for a telnet screen, and a
// line that opens with spaces starts a new paragraph. Rendering it with
// `white-space: pre-line` keeps the 80-column breaks, so in a narrower column
// every line wraps early and leaves a stray short line behind it.
//
// reflowWorldText joins each paragraph's lines into one line of prose. Text
// drawn with box characters (the maze maps in rooms 9900 and 18366) is left
// exactly as written, because its line breaks are the drawing.

const DRAWING = /\*--|--\*|\|\s{2,}\|/;

/** @param {string | undefined | null} text */
export function reflowWorldText(text) {
  const source = (text ?? '').replace(/\r\n?/g, '\n').replace(/\s+$/, '');
  if (!source) return { drawing: false, paragraphs: [] };
  if (DRAWING.test(source)) return { drawing: true, paragraphs: [source] };

  const paragraphs = [];
  let current = [];
  const flush = () => {
    if (current.length) paragraphs.push(current.join(' ').replace(/\s+/g, ' ').trim());
    current = [];
  };
  for (const line of source.split('\n')) {
    if (!line.trim()) {
      flush();
      continue;
    }
    if (/^\s/.test(line)) flush();
    current.push(line.trim());
  }
  flush();
  return { drawing: false, paragraphs };
}
