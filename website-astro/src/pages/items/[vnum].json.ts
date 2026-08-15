import type { APIRoute, GetStaticPaths } from 'astro';
import { items, type Item } from '../../lib/database';

export const getStaticPaths = (() => items.map((item) => ({ params: { vnum: String(item.v) }, props: { item } }))) satisfies GetStaticPaths;

export const GET: APIRoute<{ item: Item }> = ({ props, site }) => {
  const { item } = props;
  return new Response(JSON.stringify({
    schemaVersion: 1,
    kind: 'item',
    canonicalUrl: new URL(`/items/${item.v}/`, site),
    source: 'Dark Pawns active world files',
    record: item,
  }, null, 2), {
    headers: { 'Content-Type': 'application/json; charset=utf-8', 'Content-Language': 'en' },
  });
};
