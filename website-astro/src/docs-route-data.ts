import { defineRouteMiddleware } from '@astrojs/starlight/route-data';
import { latestRelease } from './lib/docs-release.mjs';

export const onRequest = defineRouteMiddleware(({ locals, url }) => {
  const route = locals.starlightRoute;
  if (!url.pathname.startsWith('/docs/reference/api/')) return;
  const admin = url.pathname.includes('/api/admin/');
  // The OpenAPI plugin supplies the operation content; this supplies the same
  // required editorial metadata as a hand-authored reference page.
  const description = admin
    ? 'Look up the Dark Pawns admin API request and response contract. Sign in with a level 40 character and use the authenticated admin console session.'
    : 'Look up the Dark Pawns public HTTP API request and response contract for health checks, initial server setup, and currently active game listeners.';
  Object.assign(route.entry.data, { description, type: 'reference', audience: admin ? 'operators' : 'developers', verified: latestRelease });
  route.head = route.head.filter((entry) => entry.tag !== 'meta' || entry.attrs?.name !== 'description');
  route.head.push({ tag: 'meta', attrs: { name: 'description', content: description } });
  route.editUrl = new URL('https://github.com/zax0rz/darkpawns/edit/main/cmd/admin-openapi/main.go');
});
