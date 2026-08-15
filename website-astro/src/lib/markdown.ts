type MarkdownMetadata = Record<string, string | number | boolean | Date | undefined>;

const yamlValue = (value: string | number | boolean | Date): string => {
  if (value instanceof Date) return JSON.stringify(value.toISOString());
  if (typeof value === 'string') return JSON.stringify(value);
  return String(value);
};

export function markdownResponse(body: string, metadata: MarkdownMetadata, canonicalPath: string): Response {
  const fields: MarkdownMetadata = { ...metadata, canonical: `https://darkpawns.labz0rz.com${canonicalPath}` };
  const frontmatter = Object.entries(fields)
    .flatMap(([key, value]) => value === undefined ? [] : [`${key}: ${yamlValue(value)}`])
    .join('\n');

  return new Response(`---\n${frontmatter}\n---\n\n${body.trim()}\n`, {
    headers: {
      'Content-Type': 'text/markdown; charset=utf-8',
      'Content-Language': 'en',
      Link: `<https://darkpawns.labz0rz.com${canonicalPath}>; rel="canonical"; type="text/html"`,
    },
  });
}
