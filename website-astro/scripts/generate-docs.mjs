import { execFileSync } from 'node:child_process';
import { readdirSync, readFileSync, writeFileSync, mkdirSync, rmSync, existsSync, cpSync } from 'node:fs';
import { join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { latestRelease } from './docs-navigation.mjs';
import { commandGroups } from './command-taxonomy.mjs';
const root = fileURLToPath(new URL('../../', import.meta.url));
const web = join(root, 'website-astro');
mkdirSync(join(web,'.generated'),{recursive:true});
writeFileSync(join(web,'.generated/release.json'), JSON.stringify({ version: latestRelease })+'\n');
const out = join(web, '.generated/docs');
rmSync(out, { recursive: true, force: true }); mkdirSync(out, { recursive: true });
const write = (slug, title, description, body, source, type = 'reference', audience = 'developers') => {
  const path = join(out, slug + '.md'); mkdirSync(resolve(path, '..'), { recursive: true });
  writeFileSync(path, `---\ntitle: ${JSON.stringify(title)}\ndescription: ${JSON.stringify(description)}\ntype: ${type}\naudience: ${audience}\nverified: ${latestRelease}\nsourcePath: ${JSON.stringify(source)}\n---\n\n${body.trim()}\n`);
};
const releases = readdirSync(join(root, 'docs/releases')).filter((f) => /^v\d+\.\d+\.\d+\.md$/.test(f)).sort((a,b) => b.localeCompare(a, undefined, {numeric:true}));
for (const file of releases) {
  const version = file.slice(0,-3);
  write('releases/'+version.replaceAll('.','-'), 'Dark Pawns '+version, `Changes shipped in Dark Pawns ${version}, reproduced from the release notes used by the release workflow. Read fixes, operator changes and install details.`, readFileSync(join(root,'docs/releases',file),'utf8'), 'docs/releases/'+file, 'release');
}
write('releases', 'Releases', 'Read the Dark Pawns release notes, newest first. Each page comes from the same source file used to publish the signed server release and its announcement.', releases.map((f) => `- [${f.slice(0,-3)}](/docs/releases/${f.slice(0,-3).replaceAll('.','-')}/)`).join('\n'), 'docs/releases/');
// Group the original lib/text/help records. The taxonomy keeps archive
// metadata for classification, but every body comes from raw game source.
// Code blocks protect source indentation, punctuation, and line breaks.
const helpGroups = commandGroups();
const groupLinks = [];
const fenceFor = (body) => {
  const longest = Math.max(0, ...[...body.matchAll(/`+/g)].map(([fence]) => fence.length));
  return '`'.repeat(Math.max(3, longest + 1));
};
for (const group of helpGroups) {
  const sections = group.topics.map((topic) => {
    const fence = fenceFor(topic.body);
    return `<a id="${topic.anchor}"></a>\n## ${topic.title}\n\n**Verbatim in-game help.**\n\n${fence}\n${topic.body}${topic.body.endsWith('\n') ? '' : '\n'}${fence}`;
  });
  const intro = "Look up a command or topic below. The help text keeps the game's wording, including its examples and warnings.";
  const description = `Look up Dark Pawns ${group.title.toLowerCase()} commands and topics in the game's original help text, with the wording, examples and warnings kept intact.`;
  const audience = group.slug === 'immortal-commands' ? 'operators' : 'players';
  write(`reference/commands/${group.slug}`, group.title, description, `${intro}\n\n${sections.join('\n\n')}`, 'lib/text/help/', 'reference', audience);
  groupLinks.push(`- [${group.title}](/docs/reference/commands/${group.slug}/) (${group.topics.length} topics)`);
}
write('reference/commands', 'Commands', "It's the Dark Pawns command index, with topics grouped by task. Every page keeps the original help text and gives each topic a stable anchor.", groupLinks.join('\n'), 'lib/text/help/commands.hlp', 'reference', 'players');
const apiOut = join(web,'.generated/api'); mkdirSync(apiOut,{recursive:true});
const spec = JSON.parse(execFileSync('go',['run','./cmd/admin-openapi'],{cwd:root,encoding:'utf8',maxBuffer:16*1024*1024}));
writeFileSync(join(apiOut,'openapi.json'),JSON.stringify(spec,null,2)+'\n');
const apiCatalogue = [];
for (const [kind, pred] of [['public',p => !p.startsWith('/admin')], ['admin',p => p.startsWith('/admin')]]) {
  const part = {...spec, info:{...spec.info,title:`Dark Pawns ${kind} API`,description:kind === 'admin' ? 'Sign in through POST /admin/login with a level 40 implementor character. The admin console and API share the authenticated session. Read each operation for its request and response contract.' : 'Unauthenticated HTTP endpoints for server health, first-character setup status, and observed game listeners. For a playable session, see the client connection guide.'},paths:Object.fromEntries(Object.entries(spec.paths).filter(([p]) => pred(p)))};
  // Keep display titles concise while preserving the full registered summary.
  for (const path of Object.values(part.paths)) for (const operation of Object.values(path)) {
    if (operation && typeof operation === 'object' && operation.summary?.length > 59) {
      operation.description = operation.summary + '\n\n' + (operation.description ?? '');
      operation.summary = operation.operationId.replaceAll('-', ' ');
    }
  }
  const apiDescription = kind === 'admin'
    ? 'Look up the Dark Pawns admin API request and response contract. Sign in with a level 40 character and use the authenticated admin console session.'
    : 'Look up the Dark Pawns public HTTP API request and response contract for health checks, initial server setup, and currently active game listeners.';
  const addApi = (slug, title) => apiCatalogue.push({ id: slug, title, description: apiDescription, type: 'reference', audience: kind === 'admin' ? 'operators' : 'developers', verified: latestRelease, sourcePath: 'cmd/admin-openapi/main.go' });
  addApi('reference/api/'+kind, part.info.title);
  for (const item of Object.values(part.paths)) for (const [method, operation] of Object.entries(item)) {
    if (['get','put','post','delete','options','head','patch','trace'].includes(method) && operation.operationId) addApi('reference/api/'+kind+'/operations/'+operation.operationId, operation.summary ?? operation.operationId);
  }
  writeFileSync(join(apiOut,kind+'.json'),JSON.stringify(part,null,2)+'\n');
}
// One canonical output tree serves the README and the site.
const media = join(root,'docs/images/generated'); const publicMedia = join(root,'website/static/media/docs');
if(existsSync(media)){ rmSync(publicMedia,{recursive:true,force:true}); mkdirSync(publicMedia,{recursive:true}); cpSync(media,publicMedia,{recursive:true}); }
console.log(`docs: ${releases.length} release pages, ${helpGroups.reduce((count, group) => count + group.topics.length, 0)} verbatim help topics in ${helpGroups.length} command groups, offline public/admin OpenAPI export`);

writeFileSync(join(apiOut,'catalogue.json'),JSON.stringify(apiCatalogue,null,2)+'\n');
