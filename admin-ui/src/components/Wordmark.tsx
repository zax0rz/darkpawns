// The one lockup. This mirrors website-astro/src/components/Header.astro exactly:
// an ink pawn, "DARK" in ink, "PAWNS" in oxblood, stacked on two lines in the
// display serif at line-height 0.82. Geometry and colour split are specified in
// DESIGN.md under The Wordmark Lockup Rule and checked by
// scripts/check_wordmark.py, so do not redraw either half here.
//
// The pawn's five shapes and its 24 8 52 88 viewBox are canonical. A rescaled or
// simplified redraw is a different mark, not the same mark smaller.

interface WordmarkProps {
  /** Pawn height. The site header uses 3.2rem; smaller chrome scales down. */
  pawnClass?: string;
  /** Lettering size. The site header uses 2rem. */
  textClass?: string;
  className?: string;
}

export function Wordmark({
  pawnClass = 'h-[3.2rem]',
  textClass = 'text-[2rem]',
  className = '',
}: WordmarkProps) {
  return (
    <span className={`inline-flex items-center gap-2 text-ink ${className}`}>
      <svg
        viewBox="24 8 52 88"
        className={`${pawnClass} w-auto shrink-0 text-ink`}
        aria-hidden="true"
        focusable="false"
      >
        <g fill="currentColor">
          <circle cx="50" cy="23" r="12" />
          <rect x="37" y="38" width="26" height="5" />
          <polygon points="43,46 57,46 62,75 38,75" />
          <rect x="31" y="77" width="38" height="6" />
          <rect x="26" y="85" width="48" height="7" />
        </g>
      </svg>
      <span
        // text-left is pinned, not inherited: the two lines share a left edge
        // wherever the lockup is placed. Dropped into a centred container the
        // words would centre against each other, which is a different mark.
        className={`font-display uppercase leading-[0.82] tracking-[0.01em] text-left ${textClass}`}
      >
        DARK
        <br />
        <span className="text-accent">PAWNS</span>
      </span>
    </span>
  );
}
