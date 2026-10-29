import { execFileSync } from 'node:child_process';
import { mkdtempSync, writeFileSync, rmSync, mkdirSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
const dir = mkdtempSync(join(tmpdir(), 'darkpawns-api-'));
try {
  // Export the same Huma registrations served at /api/openapi.json, offline.
  // An explicit URL can instead verify a running server's published contract.
  const source = process.env.ADMIN_OPENAPI_URL;
  const schema = source ? await (await fetch(source)).text() : execFileSync('go', ['run', './cmd/admin-openapi'], { cwd: '..', encoding: 'utf8' });
  mkdirSync('../website-astro/.generated/api', { recursive: true });
  writeFileSync('../website-astro/.generated/api/openapi.json', schema);
  const input = join(dir, 'openapi.json');
  writeFileSync(input, schema);
  execFileSync('npx', ['--no-install', 'openapi-typescript', input, '-o', 'src/api/schema.d.ts'], { stdio: 'inherit' });
} finally { rmSync(dir, { recursive: true, force: true }); }
