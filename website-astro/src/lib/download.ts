// Published release assets. Update the version and verify the asset list together.
export const releaseVersion = '0.1.4';
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
