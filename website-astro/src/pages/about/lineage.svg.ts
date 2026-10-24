import { lineageSVG } from '../../lib/lineage';

export function GET() {
  return new Response(lineageSVG(true), { headers: { 'Content-Type': 'image/svg+xml; charset=utf-8' } });
}
