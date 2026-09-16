# Topology View Redesign Provenance

Date: 2026-09-16

## Problem

The previous topology view placed every entity as a colored circle on concentric
rings. In clusters with more entities, labels and links overlapped, making the
view visually busy but operationally hard to use. Entity type was also encoded
mostly by color, with limited explanatory legend support.

## Direction

Redesign topology as a dense operational map:

- Group resources into semantic lanes such as entry, service, workload, runtime,
  and artifacts.
- Render entities as fixed-size cards with resource icons, labels, metadata, and
  health/status dots.
- Hide empty lanes and group overflow in high-cardinality views.
- Preserve existing report-wide capabilities: theme switching, export, live
  events, pan/zoom, filters, and click-through details.

## Acceptance

- The overview topology remains interactive in the existing embedded report UI.
- Recorded snapshot UI renders non-empty topology lanes with icons and links.
- Theme, export, and recent event surfaces continue to work.
- Repository checks and build pass.
