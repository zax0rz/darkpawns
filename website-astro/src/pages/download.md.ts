import { markdownResponse } from '../lib/markdown';
import { releaseVersion, releaseURL, assetBase, unixInstall, windowsInstall, platforms, archiveURL, portableGuide, hostingGuide } from '../lib/download';

export function GET() {
  const downloads = platforms.flatMap((platform) => platform.architectures.map(({ arch, label }) =>
    `- [${platform.name}: ${label} (.${platform.format})](${archiveURL(platform.os, arch, platform.format)})`,
  )).join('\n');
  return markdownResponse(`
# A world of your own.

Download Dark Pawns, create your immortal, and keep the world running on your own machine.
[Release v${releaseVersion}](${releaseURL}). [Play here instead](https://darkpawns.org/play/).

## Install the game

macOS / Linux:

\`\`\`sh
${unixInstall}
\`\`\`

Uses curl, OpenSSL, and tar. No administrator rights or compiler needed.

Windows (PowerShell 5.1 or newer):

\`\`\`powershell
${windowsInstall}
\`\`\`

The installer checks the publisher signature and archive checksum, then opens the setup wizard.

## Make it yours

1. Choose a home for the world. Players, edits, and settings are kept apart from the release.
2. Choose the local web port and startup options. Review them, then choose Create and launch.
3. Open the printed address. The first character becomes the immortal: create it before inviting anyone else.

A new world listens on this machine only. Follow the [hosting guide](${hostingGuide}) before inviting other players.

## Download an archive

Each archive includes the launcher, server, and starter world. Downloads are hosted on GitHub.

${downloads}

For Macs, check About This Mac for Apple silicon or Intel. On Linux, run \`uname -m\` (x86_64 or aarch64). On Windows, check Settings → System → About → System type.

[Verify the archive against the signed release manifest](${portableGuide}#manual-updates) before opening it.
[Manifest](${assetBase}/release-manifest.txt) · [Signature](${assetBase}/release-manifest.sig).

Extract the archive and run \`darkpawns setup\`, or \`.\\darkpawns.exe setup\` in PowerShell. Use \`setup --plain\` for screen readers.

## Keep the world running

\`\`\`sh
darkpawns status
darkpawns start
darkpawns stop
darkpawns doctor
\`\`\`

Keep a backup of your instance, which holds players and world edits.
[Portable release guide](${portableGuide}) · [Hosting guide](${hostingGuide}).

## If your system blocks it

Releases carry a publisher signature checked by the installer. They are not Apple-notarized or Windows Authenticode-signed. A browser download may trigger an operating-system warning.

[Apple's guide to opening apps safely](https://support.apple.com/102445) · [Microsoft SmartScreen guidance](https://learn.microsoft.com/windows/security/operating-system-security/virus-and-threat-protection/microsoft-defender-smartscreen/).

On Windows, run \`.\\darkpawns.exe setup\` from PowerShell rather than double-clicking the executable. See the [installation notes](https://github.com/zax0rz/darkpawns#install) for observed platform warnings.
`, { title: 'Download & Host Dark Pawns', textKind: 'original', voiceLayer: 'mythic-admin' }, '/download/');
}
