# Report Page Shell Redesign Provenance

Date: 2026-09-16

## Problem

After the topology redesign, the rest of the report still used the older visual
language: softer card-heavy spacing, a hero-like header, rounded panel chrome,
and less consistent dense operational styling. The user asked to bring the whole
page, not only topology, in line with the mock page.

## Direction

Apply the mock's dense operations-console style across the embedded report UI:

- Keep the existing embedded single-file report and vanilla JavaScript model.
- Preserve navigation, filters, theme switching, export, live events, AI
  analysis, drawer details, and table rendering behavior.
- Tighten the layout with a darker shell, compact header/action bar, denser
  cards, smaller panel radii, and more scannable tables.
- Keep light and dark themes readable, with visible focus states and responsive
  behavior.

## Acceptance

- The recorded snapshot live UI renders overview, tables, topology, drawer, and
  recent events using the same visual direction.
- Existing report tests, full checks, and build pass.
- Browser validation shows no console errors for the recorded snapshot UI.
