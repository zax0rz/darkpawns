import type { ReactNode } from 'react';

// Authored icon set. One geometric language, one stroke weight, square caps,
// to match the brutalist pawn mark rather than a rounded UI-kit look. Emoji
// were standing in here before, which gave every platform a different drawing
// and gave Dashboard and Decisions the same one.

export type IconName =
  | 'dashboard'
  | 'zones'
  | 'mobs'
  | 'objects'
  | 'agents'
  | 'operations'
  | 'terminal'
  | 'search'
  | 'refresh'
  | 'close'
  | 'check'
  | 'cross'
  | 'info'
  | 'pawn';

const PATHS: Record<IconName, ReactNode> = {
  // Panels of a status board.
  dashboard: (
    <>
      <rect x="2" y="2" width="5" height="5" />
      <rect x="9" y="2" width="5" height="5" />
      <rect x="2" y="9" width="5" height="5" />
      <rect x="9" y="9" width="5" height="5" />
    </>
  ),
  // A surveyed region with its marker.
  zones: (
    <>
      <path d="M2 4 L6 2 L10 4 L14 2 L14 12 L10 14 L6 12 L2 14 Z" />
      <path d="M6 2 L6 12" />
      <path d="M10 4 L10 14" />
    </>
  ),
  // A creature: horned silhouette, deliberately not a face.
  mobs: (
    <>
      <path d="M3 6 L3 3 L5 5" />
      <path d="M13 6 L13 3 L11 5" />
      <path d="M3 6 L8 2 L13 6 L13 11 L8 14 L3 11 Z" />
      <path d="M6 8 L6 9" />
      <path d="M10 8 L10 9" />
    </>
  ),
  // A crate in isometric.
  objects: (
    <>
      <path d="M8 2 L14 5 L8 8 L2 5 Z" />
      <path d="M2 5 L2 11 L8 14 L14 11 L14 5" />
      <path d="M8 8 L8 14" />
    </>
  ),
  // A node with its links: a process, not a robot.
  agents: (
    <>
      <rect x="5" y="5" width="6" height="6" />
      <path d="M8 2 L8 5" />
      <path d="M8 11 L8 14" />
      <path d="M2 8 L5 8" />
      <path d="M11 8 L14 8" />
    </>
  ),
  // Control sliders, not the universal gear.
  operations: (
    <>
      <path d="M2 4 L14 4" />
      <path d="M2 8 L14 8" />
      <path d="M2 12 L14 12" />
      <rect x="4" y="2.5" width="3" height="3" />
      <rect x="9" y="6.5" width="3" height="3" />
      <rect x="5" y="10.5" width="3" height="3" />
    </>
  ),
  // A prompt awaiting a command.
  terminal: (
    <>
      <rect x="2" y="3" width="12" height="10" />
      <path d="M4.5 6.5 L6.5 8 L4.5 9.5" />
      <path d="M8 10 L11.5 10" />
    </>
  ),
  // A sweep for findings.
  search: (
    <>
      <circle cx="7" cy="7" r="4.5" />
      <path d="M10.5 10.5 L14 14" />
    </>
  ),
  // Re-read from the source.
  refresh: (
    <>
      <path d="M13 8 A5 5 0 1 1 11.5 4.5" />
      <path d="M14 2 L14 5 L11 5" />
    </>
  ),
  close: (
    <>
      <path d="M3.5 3.5 L12.5 12.5" />
      <path d="M12.5 3.5 L3.5 12.5" />
    </>
  ),
  check: (
    <>
      <path d="M3 8.5 L6.5 12 L13 4" />
    </>
  ),
  cross: (
    <>
      <path d="M4 4 L12 12" />
      <path d="M12 4 L4 12" />
    </>
  ),
  info: (
    <>
      <circle cx="8" cy="8" r="6" />
      <path d="M8 7 L8 11" />
      <path d="M8 4.8 L8 5.4" />
    </>
  ),
  // The house mark. Canonical geometry, transplanted into the 16x16 grid by
  // transform rather than redrawn: a simplified pawn is a different mark, not
  // the same mark smaller. See DESIGN.md, The Wordmark Lockup Rule.
  pawn: (
    <g
      transform="translate(-4.6154 -1.4615) scale(0.18462)"
      fill="currentColor"
      stroke="none"
    >
      <circle cx="50" cy="23" r="12" />
      <rect x="37" y="38" width="26" height="5" />
      <polygon points="43,46 57,46 62,75 38,75" />
      <rect x="31" y="77" width="38" height="6" />
      <rect x="26" y="85" width="48" height="7" />
    </g>
  ),
};

interface IconProps {
  name: IconName;
  className?: string;
}

export function Icon({ name, className = 'h-4 w-4' }: IconProps) {
  return (
    <svg
      viewBox="0 0 16 16"
      className={className}
      fill="none"
      stroke="currentColor"
      strokeWidth={1.4}
      strokeLinecap="square"
      strokeLinejoin="miter"
      aria-hidden="true"
      focusable="false"
    >
      {PATHS[name]}
    </svg>
  );
}
