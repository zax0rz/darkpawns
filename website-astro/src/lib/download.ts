import { latestRelease } from './docs-release.mjs';

// Published release assets. The version is the newest docs/releases/vX.Y.Z.md,
// the same source the docs use, so publishing a release's notes moves this page
// with it. `make deploy-site` checks every asset link resolves before syncing.
export const releaseVersion = latestRelease.replace(/^v/, '');
export const releaseURL = `https://github.com/zax0rz/darkpawns/releases/tag/v${releaseVersion}`;
export const assetBase = `https://github.com/zax0rz/darkpawns/releases/download/v${releaseVersion}`;
export const unixInstall = 'curl -fsSL https://darkpawns.org/install.sh | bash';
// Keep this invocation in step with README.md: use a temporary file and clean it up.
export const windowsInstall = "$installer = Join-Path ([IO.Path]::GetTempPath()) ('darkpawns-' + [Guid]::NewGuid().ToString('N') + '.ps1'); try { Invoke-WebRequest -UseBasicParsing 'https://github.com/zax0rz/darkpawns/releases/latest/download/install.ps1' -OutFile $installer; & powershell.exe -NoProfile -ExecutionPolicy Bypass -File $installer; if ($LASTEXITCODE -ne 0) { throw 'Installer did not complete.' } } finally { Remove-Item -LiteralPath $installer -ErrorAction SilentlyContinue }";
export const platforms = [
  { name: 'macOS', os: 'darwin', format: 'tar.gz', architectures: [{ arch: 'arm64', label: 'Apple silicon' }, { arch: 'amd64', label: 'Intel' }] },
  { name: 'Linux', os: 'linux', format: 'tar.gz', architectures: [{ arch: 'amd64', label: 'Intel / AMD (x86-64)' }, { arch: 'arm64', label: 'ARM64' }] },
  { name: 'Windows', os: 'windows', format: 'zip', architectures: [{ arch: 'amd64', label: 'Intel / AMD (x86-64)' }, { arch: 'arm64', label: 'ARM64' }] },
];
export const archiveURL = (os: string, arch: string, format: string) =>
  `${assetBase}/darkpawns_${releaseVersion}_${os}_${arch}.${format}`;
export const portableGuide = 'https://github.com/zax0rz/darkpawns/blob/main/docs/portable-release.md';
export const hostingGuide = 'https://github.com/zax0rz/darkpawns/blob/main/docs/portable-hosting.md';
