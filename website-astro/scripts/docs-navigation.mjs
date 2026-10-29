import { readdirSync } from 'node:fs';
import docsRedirects from './docs-redirects.json' with { type: 'json' };
export { docsRedirects };
const versions = readdirSync(new URL('../../docs/releases/', import.meta.url)).filter((f) => /^v\d+\.\d+\.\d+\.md$/.test(f)).sort((a,b) => b.localeCompare(a, undefined, { numeric: true }));
export const latestRelease = versions[0].slice(0, -3);
const link = (label, slug) => ({ label, link: '/docs/' + slug + '/' });
export function sidebar(apiGroups = []) {
  return [
    { label: 'Play', collapsed: false, items: [link('Your first hour','play/first-hour'), link('Connect a client','play/connect-client')] },
    { label: 'Host', collapsed: false, items: [link('Install Dark Pawns','host/install'), link('Your first login','host/first-login'),link('The admin console','host/admin-console'),link('Dashboard and logs','host/dashboard-and-logs'),link('Player management','host/player-management'),link('Notifications','host/notifications'),link('Message of the day','host/motd-editor'),link('Configuration','host/admin-configuration'),link('Update your server','host/update-server'),link("Players can't connect",'host/players-cannot-connect')] },
    { label: 'Build', items: [link('Run a local server','build/local-server')] },
    { label: 'Reference', items: [link('Glossary','reference/glossary'),link('Commands','reference/commands'), ...apiGroups] },
    { label: 'Project', items: [link('Architecture','project/architecture'),link('Contributing','project/contributing'),link('Port fidelity','project/port-fidelity'),link('Release announcements','project/release-announcements')] },
    { label: 'Releases', items: [link('All releases','releases'), ...versions.map((f) => link(f.slice(0,-3), 'releases/'+f.slice(0,-3).replaceAll('.','-')))] },
  ];
}
