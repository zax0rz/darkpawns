import type { APIRoute, GetStaticPaths } from 'astro';
import { rooms, type RoomRecord } from '../../lib/rooms';

export const getStaticPaths = (() => rooms.map((room) => ({ params: { vnum: String(room.id) }, props: { room } }))) satisfies GetStaticPaths;

export const GET: APIRoute<{ room: RoomRecord }> = ({ props, site }) => {
  const { room } = props;
  return new Response(JSON.stringify({
    schemaVersion: 1,
    kind: 'room',
    canonicalUrl: new URL(`/rooms/${room.id}/`, site),
    source: 'Dark Pawns active world files',
    mapUrl: new URL(`/map/?room=${room.id}`, site),
    record: room,
  }, null, 2), {
    headers: { 'Content-Type': 'application/json; charset=utf-8', 'Content-Language': 'en' },
  });
};
