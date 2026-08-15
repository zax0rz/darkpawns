import type { APIRoute, GetStaticPaths } from 'astro';
import { zonesSorted, type ZoneRecord } from '../../lib/zones';

export const getStaticPaths = (() => zonesSorted.map((zone) => ({ params: { id: String(zone.id) }, props: { zone } }))) satisfies GetStaticPaths;

export const GET: APIRoute<{ zone: ZoneRecord }> = ({ props, site }) => {
  const { zone } = props;
  return new Response(JSON.stringify({
    schemaVersion: 1,
    kind: 'zone',
    canonicalUrl: new URL(`/zones/${zone.id}/`, site),
    source: 'Dark Pawns active world files',
    mapUrl: new URL(`/map/?zone=${zone.id}`, site),
    record: {
      id: zone.id,
      name: zone.name,
      top: zone.top,
      roomCount: zone.roomCount,
      connectedZoneIds: zone.connectedZoneIds,
      rooms: zone.rooms,
      mobs: zone.mobs,
      items: zone.items,
      shops: zone.shops,
    },
  }, null, 2), {
    headers: { 'Content-Type': 'application/json; charset=utf-8', 'Content-Language': 'en' },
  });
};
