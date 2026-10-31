import { getCollection } from 'astro:content';
import config from 'virtual:starlight/user-config';

type Page = { id: string; title: string; description: string; markdown: string };
type SidebarItem = string | { label?: string; link?: string; slug?: string; items?: SidebarItem[]; autogenerate?: { directory: string } };
const normal = (value: string) => value.replace(/^\/+|\/+$/g, '').replace(/(^|\/)index$/, '').replace(/^docs\/?/, '');
const pathFor = (id: string) => id ? '/docs/'+id+'/' : '/docs/';

// The same collections that build authored, generated and API documentation
// supply the titles and descriptions. The resolved Starlight sidebar owns order.
export async function documentationSections() {
  const collections = await Promise.all([getCollection('docs'), getCollection('generatedDocs'), getCollection('docsApiCatalogue')]);
  const pages = new Map<string, Page>();
  for (const entry of collections.flat()) {
    if ('draft' in entry.data && entry.data.draft) continue;
    const id = normal(entry.id);
    pages.set(id, { id, title: entry.data.title, description: entry.data.description, markdown: pathFor(id).replace(/\/$/, '')+'.md' });
  }
  const visited = new Set<string>();
  const alphabetical = (a: Page,b: Page) => a.title.localeCompare(b.title,'en');
  const take = (id: string, list: Page[]) => { const page = pages.get(normal(id)); if(page && !visited.has(page.id)){visited.add(page.id);list.push(page);} };
  function walk(items: SidebarItem[], list: Page[]) {
    for(const item of items){
      if(typeof item === 'string'){take(item,list);continue;}
      if(item.link?.startsWith('/docs')) take(item.link,list);
      if(item.slug) take(item.slug,list);
      if(item.items) walk(item.items,list);
      if(item.autogenerate){const prefix=normal(item.autogenerate.directory);for(const p of [...pages.values()].filter(p=>p.id===prefix||p.id.startsWith(prefix+'/')).sort(alphabetical))take(p.id,list);}
    }
  }
  const sections: {title: string; pages: Page[]}[]=[];
  for(const item of (config.sidebar ?? []) as SidebarItem[]){
    const list: Page[]=[];
    const label=typeof item==='string' ? 'Overview' : item.label ?? 'Overview';
    if(typeof item!=='string' && item.items){
      const firstLink=JSON.stringify(item.items).match(/\/docs\/([^/"\\]+)/)?.[1];
      if(firstLink)take(firstLink,list);
    }
    walk([item],list);
    const prefix=list.find(p=>p.id)?.id.split('/')[0];
    // New content still appears if an editor hasn't added its individual link
    // yet. It belongs to the sidebar's existing section, never a second list.
    if(prefix)for(const p of [...pages.values()].filter(p=>p.id===prefix||p.id.startsWith(prefix+'/')).sort(alphabetical))take(p.id,list);
    if(list.length)sections.push({title:label,pages:list});
  }
  const remaining=[...pages.values()].filter(p=>!visited.has(p.id)).sort(alphabetical);
  if(remaining.length)sections.push({title:'More documentation',pages:remaining});
  return sections;
}
