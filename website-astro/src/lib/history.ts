export interface HistoryEra {
  readonly year: string;
  readonly title: string;
  readonly text: string;
  readonly source: string;
  readonly sourceHref?: string;
}

// Canonical Dark Pawns chronology — single source of truth shared by `/` and `/about/`.
// Every date and claim traces to a primary source (Sourced-or-Silent Rule).
export const HISTORY_ERAS: readonly HistoryEra[] = [
  {
    year: '1994',
    title: 'Dark Pawns begins',
    text: 'Serapis (Derek Karnes) and Tracer (C. Jackson) open Dark Pawns on CircleMUD 1.7 at knight.ufp.org in September.',
    source: 'timeline.md · Sept 1994',
    sourceHref: '/archive/history/',
  },
  {
    year: '1995',
    title: 'The first major change',
    text: 'The base code moves to CircleMUD 2.2, and the game becomes Dark Pawns 1.0.',
    source: 'timeline.md · 1995',
    sourceHref: '/archive/history/',
  },
  {
    year: '1996',
    title: 'Coders and builders arrive',
    text: 'Arak and Orodreth join as coders in February; Frontline follows in March while Oddity, Nimos, and Aralius build the world.',
    source: 'timeline.md · Feb–Mar 1996',
    sourceHref: '/archive/history/',
  },
  {
    year: '1998',
    title: 'Dark Pawns 2.2',
    text: 'Frontline (R.E. Paret) and Orodreth (S. Thompson) become co-implementors in May and ship the largest update in the game\u2019s history that December.',
    source: 'timeline.md · May & Dec 1998',
    sourceHref: '/archive/history/',
  },
  {
    year: '2001',
    title: 'The break-in',
    text: 'An intruder wipes talon.wolfpaw.net. Orodreth\u2019s backups save most of the world, player data and much of 3.0 are lost, and darkpawns.com is registered.',
    source: 'timeline.md · Feb 2001',
    sourceHref: '/archive/history/',
  },
  {
    year: '2010',
    title: 'The quiet years',
    text: 'The public server closes. Frontline later publishes the surviving C source and world files on GitHub in the state they held at closure.',
    source: 'rparet/darkpawns README',
    sourceHref: 'https://github.com/rparet/darkpawns/blob/master/README.md',
  },
  {
    year: '2019',
    title: 'DPReturns',
    text: 'Former players reopen the C codebase at dpreturns.com (now darkpawns.net), started by Slayer, expanded over the pandemic years, and later maintained by Vintage.',
    source: 'r/MUD · Oct 2019',
    sourceHref: '/about/dpreturns/',
  },
  {
    year: '2026',
    title: 'The Go restoration',
    text: 'A new open-source Go server loads the original world and verifies its player-facing behavior byte for byte against the C oracle.',
    source: 'C-oracle · Go port',
    sourceHref: '/docs/project/port-fidelity/',
  },
];
