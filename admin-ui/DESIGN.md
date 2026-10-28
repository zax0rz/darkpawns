---
name: Dark Pawns Admin
description: Haunted Paperback controls for operators and world builders.
colors:
  paper: "#EFE7D6"
  paper-deep: "#E5DAC1"
  ink: "#1A1614"
  ink-muted: "#56504A"
  oxblood: "#A8201A"
  oxblood-deep: "#7A1812"
  registration: "#0F0F0F"
  online: "#3A8A3A"
typography:
  headline:
    fontFamily: "DM Serif Display, Georgia, serif"
    fontSize: "1.5rem"
    fontWeight: 400
    lineHeight: "2rem"
    letterSpacing: "0.01em"
  title:
    fontFamily: "DM Serif Display, Georgia, serif"
    fontSize: "1.25rem"
    fontWeight: 400
    lineHeight: "1.75rem"
  body:
    fontFamily: "Source Serif 4, Georgia, serif"
    fontSize: "1rem"
    lineHeight: "1.5rem"
  control:
    fontFamily: "Source Serif 4, Georgia, serif"
    fontSize: "0.875rem"
    lineHeight: "1.25rem"
  mono:
    fontFamily: "JetBrains Mono, Fira Code, monospace"
    fontSize: "0.75rem"
    lineHeight: "1rem"
rounded:
  sharp: "0"
  micro: "2px"
spacing:
  xs: "0.25rem"
  sm: "0.5rem"
  md: "1rem"
  lg: "1.5rem"
  xl: "2rem"
components:
  button-primary:
    backgroundColor: "{colors.oxblood}"
    textColor: "{colors.paper}"
    rounded: "{rounded.micro}"
    height: "2.25rem"
    padding: "0.5rem 1rem"
  button-outline:
    backgroundColor: "{colors.paper}"
    textColor: "{colors.ink}"
    rounded: "{rounded.micro}"
  button-secondary:
    backgroundColor: "{colors.paper-deep}"
    textColor: "{colors.ink}"
  button-ghost:
    backgroundColor: "transparent"
  button-link:
    textColor: "{colors.oxblood}"
  input:
    backgroundColor: "{colors.paper}"
    rounded: "{rounded.micro}"
    height: "2.25rem"
    padding: "0.25rem 0.75rem"
  card:
    backgroundColor: "{colors.paper-deep}"
    textColor: "{colors.ink}"
    rounded: "{rounded.micro}"
  navigation:
    textColor: "{colors.ink}"
    rounded: "{rounded.sharp}"
    padding: "0.5rem 0.875rem"
---

# Admin console design

## Overview

**Creative North Star: "The Haunted Paperback"**

The console extends [Haunted Paperback](../website-astro/DESIGN.md): wordmark at
top left, static desktop sidebar, main working area, and one light theme. The
operator's job is to inspect records, edit safely, and understand save and
validation states. Use [brand voice Layer 1](../docs/brand-voice.md): direct,
technical, specific.

**Key Characteristics:**
- Cream surfaces, charcoal rules, sparse oxblood.
- Flat controls, serif reading, machine-only monospace.
- Existing permissions and save boundaries remain visible.

## Colors

Paper carries the working area and dialogs; Paper-deep carries sidebar, table
headings, cards and toasts. Ink carries text and rules; muted ink supports
metadata. Oxblood marks primary actions, selection, focus and errors. Deep
oxblood carries primary and destructive hover states; outlined secondary actions
invert to ink with paper text on hover. Green means live status or success, never decoration.
Registration is the inherited black-plate token, not a new surface color.

**The One Oxblood Rule.** Keep the accent sparse; status also needs words or icons.

## Typography

DM Serif Display carries headings, Source Serif 4 reading and form copy, and
JetBrains Mono identifiers, commands, timestamps and measurements. Fonts are
served locally. Console type uses a fixed rem scale: page headings generally
1.5rem, subsection titles 1.25rem, reading copy 1rem, controls 0.875rem, machine
metadata 0.75rem. The development preview uses a larger 1.875rem heading.

## Layout

Keep the existing sidebar and working-area composition. At 1024px and above the
sidebar is 16rem wide; below it, navigation opens in a Radix dialog drawer.
Below 768px a bottom navigation bar remains available. The toolbar is 3rem high;
working-area padding changes from 1rem to 1.5rem at 768px. Wide tables scroll
within the working area. Role-filtered groups and primary save placement remain.

## Elevation & Depth

**The Flat-Ink Rule.** Separate surfaces with ink rules and the two paper tones.
No shadows, gradients, glass or blur. Modal scrims use ink at 45% opacity.
State transitions are brief; reduced-motion preferences shorten them.

## Shapes

Corners are square or at most 2px. Rules are generally 1px; table headings use
a 2px lower rule. Focus uses an oxblood outline rather than a glow.

## Components

Source-owned shadcn/ui components use Radix for keyboard navigation, focus and
portalled overlays, restyled with the paperback tokens.

- **Button:** explicit action labels; oxblood primary/destructive, ruled outline,
  paper-deep secondary, ghost navigation, and oxblood links. Shared default height
  is 2.25rem. Disabled and pending states stay recognizable.
- **Switch:** rectangular track and thumb, ink rule, oxblood checked state, visible
  label. Checkboxes remain for independent selections.
- **Input and Select:** paper-filled inputs and native selects; the Radix trigger
  inherits its containing surface. Search fields may explicitly remain transparent.
  Ink borders, 2px corners,
  oxblood focus outline. Errors sit beside fields. Native selects retain dense
  schema-generated form semantics.
- **Table:** semantic headings, paper-deep header, ink row dividers, tabular
  measurements. No zebra striping; hover adds a faint oxblood wash.
- **Dialog and AlertDialog:** paper panel, ink rule and restrained scrim; explicit
  title, description, cancellation and focus. Reserve for protected focus or
  consequential confirmation. Native unsaved-navigation browser confirmations
  remain to preserve existing behavior.
- **Tabs:** ruled navigation, oxblood selection and keyboard arrow navigation.
  Preserve existing content and save boundaries.
- **Toast:** paper-deep, status icon, readable message, labeled dismissal; errors
  use oxblood. No shadow. Status never depends on color alone.
- **Tooltip and DropdownMenu:** portalled paper panels, ink rules; supplement
  accessible labels rather than replace them.
- **Card:** flat ruled grouping for a distinct working unit. Avoid nesting.
- **Chart:** Recharts through shadcn's container, paper tooltip, serif labels.
  Oxblood, ink and muted-ink series; explicit legends and dash patterns distinguish
  series. Dashboard and activity charts use recorded telemetry: latest online and
  observed peak, daily logins and accepted characters, and selectable activity
  counts. Unobserved intervals remain gaps; adjacent aggregated observations do
  not prove continuous coverage within an interval. State sample coverage beside
  charts and in detail tables; partial periods are never presented as full-period
  totals. Preserve explicit first-observation, unknown and unavailable states.
  Operational charts have no synthetic fallback. Development fixtures remain
  explicitly synthetic and separate from operational claims.
- **Operational drilldowns and logs:** linked measurements open operations or
  UTC activity detail; error and warning detail links open filtered retained logs.
  Keep filters, live/paused/error status and raw .log/CSV actions visible above
  the dense table. Downloads apply the filters to the retained ring of at most
  1,000 entries, not a full-file archive; live arrivals can change the export
  between viewing and downloading. Mudlog type and immortal visibility remain
  distinct from server severity, with the caller's access ceiling preserved.
  Retention and dropped-event notices explain why historical totals can exceed
  the available log rows.
- **Icons:** Lucide navigation icons use a consistent 1.5 stroke, usually 16px.
  Restrained linework stays legible beside dense serif copy and reads as ink.
  Phosphor offers more weight variants; one quiet stroke suits this console better.
  Shadcn control icons retain their component defaults. The canonical pawn and
  wordmark remain unchanged brand assets.
- **Data boundary:** JSON calls use the generated OpenAPI schema with openapi-fetch;
  Prometheus and SSE retain their separate transports. Preserve cookie/CSRF
  headers, authorization, revisions, ETags and editing semantics.

## Do's and Don'ts

- **Do** keep labels explicit and validation, errors and unsaved state visible.
- **Do** verify routed pages, narrow layouts, focus, overlays and save boundaries.
- **Do** keep screenshots as local evidence; this foundation ships no raster assets.
- **Don't** introduce a second theme or decorative status colors.
- **Don't** turn synthetic chart fixtures into operational claims.


## Character activity detail

The activity page keeps the dashboard shell and selected 24-hour/7-day window.
For logins, accepted creations and player deaths, show the named evidence before
the aggregate chart. Search by character name; page older records with a stable
cursor. Keep recording start, visibility, missing details and retention visible.
Death rows show room name/vnum and a supplied cause; unknown stays unknown.

Desktop uses the shared ruled table. Below 768px each record stacks its name,
class, time, room and cause so the operator never needs to pan sideways to learn
what killed a character. The chart stays below the records. The coverage table
is a native disclosure, and release/uptime has its own selected detail view.
Use the existing flat buttons, inputs, charts and focus treatment. Product copy
follows brand-voice §8.5.
