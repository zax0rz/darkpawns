import type { APIRoute, GetStaticPaths } from 'astro';
import { mobs, type Mob } from '../../lib/database';

export const getStaticPaths = (() => mobs.map((mob) => ({ params: { vnum: String(mob.v) }, props: { mob } }))) satisfies GetStaticPaths;

export const GET: APIRoute<{ mob: Mob }> = ({ props, site }) => {
  const { mob } = props;
  return new Response(JSON.stringify({
    schemaVersion: 1,
    kind: 'mob',
    canonicalUrl: new URL(`/mobs/${mob.v}/`, site),
    source: 'Dark Pawns active world files',
    record: mob,
  }, null, 2), {
    headers: { 'Content-Type': 'application/json; charset=utf-8', 'Content-Language': 'en' },
  });
};
