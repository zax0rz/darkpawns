---
version: 1
slug: "admin-ui-src-pages-motdpage-tsx"
primary_target: "admin-ui/src/pages/MotdPage.tsx"
related_targets: ["admin-ui/src/components/MotdPreview.tsx"]
---

# MOTD editor

Mode: Operate. Extend the established admin console; follow brand-voice §8.5.
Primary target: admin-ui/src/pages/MotdPage.tsx. Route: /admin/staff/motd.

## Direction contract

THESIS: Welcome-message editing is a focused writing task with an honest terminal preview and a safe save boundary.
OWN-WORLD: Inherit the cream paper, charcoal rules, serif headings and oxblood controls. The terminal uses the existing client's dark screen because it previews player output; it is not a second console theme.
STORY: Find MOTD in the sidebar or dashboard. Choose mortal or immortal text, edit, inspect colours and column wrapping, then save. Drafts survive tab switches. A stale revision preserves the draft and explains the conflict.
FIRST VIEWPORT: Heading and short purpose, two document tabs, audience and save controls, then editor left / terminal right on desktop. Stack on mobile; keep terminal width selectable and scroll inside it. Snippets sit below the editor. Discord insertion asks for the operator's invite.
FORM: Precisely specified local extension of the existing file editor. No concept roll or comp is required; the user confirmed tabs and the desktop/mobile arrangement. seed: none (narrow extension).
FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance

## Quality bar

Both messages use the existing authenticated tedit/fileedit route, ETag preconditions, live refresh and audit trail. Enforce C's tedit command and field permission levels. Preview game colour markers and hard terminal wrapping; never rewrite saved text for preview. Show loading, read-only, unsaved, pending, over-limit, error and conflict states. Verify real-handler saves/conflicts, desktop and mobile screenshots, build/lint and sequential Go gates. No shipping rasters or visual-system changes.
