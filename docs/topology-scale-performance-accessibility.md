# Topology Scale, Performance, And Accessibility Gates

These gates define the supported Scope-inspired investigation experience. They
are acceptance targets for the embedded report, live server, and hub drilldown;
they are not claims about every browser or optional telemetry backend.

## Deterministic scale fixtures

The fixture builder in `internal/topology/scale_fixture.go` defines the
canonical shapes used by tests and benchmarks:

| Profile | Nodes | Edges | Intended path |
| --- | ---: | ---: | --- |
| Small | 100 | 180 | Full SVG graph and Inspector |
| Medium | 400 | 800 | Maximum full SVG graph budget |
| Large | 1,500 | 2,800 | Relation table and filtered-neighborhood exploration |

Fixtures use stable IDs, typed edges, deterministic timestamps, and explicit
unavailable runtime-connection coverage. A measured update changes ten
existing nodes while retaining stable identities. A large fixture remains
complete graph evidence even though its unfiltered UI defaults to the table.

Run `make topology-scale-demo SCALE_PROFILE=large`, then open
`http://127.0.0.1:8092`. The command injects the exact canonical graph through
the shared offline `RenderUI` path. The page selects the Runtime view, performs
five painted renders, and exposes the result visibly in
`#topologyScaleEvidence` and programmatically as
`window.__teleskopeScaleEvidence`. Repeat with `small`, `medium`, and `large`;
the Large page must report `mode: table` and the complexity remedy. Use
`go run ./scripts/topology-scale -profile medium -output /tmp/topology-medium.html`
to retain a self-contained artifact instead of serving it.

## UI budgets

Measure a production build in a current desktop Chromium browser on a machine
with at least four logical cores. Run five warm samples and report the median
and worst sample together with browser version, machine, commit, fixture, and
viewport. The topology surface exposes its last sample as `data-render-ms`,
`data-render-nodes`, `data-render-edges`, and `data-render-mode`.

| Operation | Small | Medium | Large/degraded |
| --- | ---: | ---: | ---: |
| Initial usable render | median <= 500 ms, worst <= 1 s | median <= 1.5 s, worst <= 3 s | table median <= 2 s, worst <= 4 s |
| Accepted incremental revision | median <= 250 ms | median <= 500 ms | table median <= 750 ms |
| Search/filter response | <= 100 ms | <= 200 ms | <= 300 ms |
| Node/edge selection response | <= 100 ms | <= 150 ms | <= 200 ms |
| Unchanged-node lane/rank movement | 0 positions | 0 positions | 0 positions |

The SVG graph accepts at most 400 visible nodes and 800 visible edges per
semantic view. Above either limit Teleskope selects the relation table, retains
the complete evidence and URL/filter state, and announces the observed counts,
limits, and narrowing remedy. Filtering below both limits makes graph mode
available again. No data is removed from JSON or Markdown export.

Browser JavaScript heap should remain below 150 MiB for Small, 300 MiB for
Medium, and 350 MiB for Large in table mode after five accepted revisions.
Growth after the first accepted revision should remain below 10% when the same
revision is repeatedly rendered. These memory figures require browser tooling;
they are recorded evidence, not wall-clock unit-test assertions.

Reference browser sample on 2026-09-27, Chromium 154 on Apple M4, 840x1175
viewport, five painted renders from `scripts/topology-scale`:

| Profile | Mode | Median | Worst | Heap |
| --- | --- | ---: | ---: | ---: |
| Small | graph | 22.0 ms | 34.2 ms | 23.4 MiB |
| Medium | graph | 83.4 ms | 125.2 ms | 33.8 MiB |
| Large | table fallback | 411.4 ms | 434.2 ms | 42.6 MiB |

The Large evidence also recorded the exact 1,500-object/2,800-relationship
fallback message. Values are a reproducible reference sample, not a
cross-machine guarantee.

## Server and provider budgets

`make topology-bench` measures full diff and delta apply with the canonical
fixtures. On the same reference class of four-core machine, Medium diff/apply
should each stay below 50 ms and 16 MiB allocated per operation; Large should
stay below 250 ms and 64 MiB. CI records the values and investigates a greater
than 20% median regression before release instead of using flaky timing
assertions.

For a single 1,500-node cluster updated once per second:

- live server steady state: <= 1 CPU core and <= 256 MiB RSS;
- hub incremental processing per active cluster: <= 0.25 CPU core and <= 128
  MiB additional RSS, with the existing 16-revision bounded history;
- Kubernetes/EKS collection: <= 0.5 CPU core and <= 256 MiB RSS, with no
  increased API polling solely for topology animation;
- each optional external evidence provider publishes and is tested against its
  own CPU, memory, sample/window, and backpressure budget before graduation.

Missing provider evidence remains `unavailable`. A privileged node Probe or
DaemonSet is not a supported way to meet any budget.

Reference sample on 2026-09-27, Apple M4 arm64, from `make topology-bench`:

| Profile | Diff | Diff bytes/op | Apply | Apply bytes/op |
| --- | ---: | ---: | ---: | ---: |
| Small | 1.73 ms | 1.25 MiB | 0.77 ms | 0.55 MiB |
| Medium | 6.95 ms | 6.01 MiB | 3.43 ms | 2.83 MiB |
| Large | 25.80 ms | 25.55 MiB | 12.28 ms | 12.06 MiB |

This is a reproducible baseline, not a cross-machine guarantee.

## Accessibility gates

- Every node and edge is keyboard focusable and selectable with Enter/Space.
  Table and Inspector relatives use native buttons and table semantics.
- Zoom in, zoom out, and reset are named buttons; pointer wheel and drag are
  enhancements, not the only controls. Zoom remains bounded to 35%-400%.
- Selected and focused objects have non-color SVG treatment. Edge type is
  available in accessible names and Inspector text, not only stroke color.
- Search, view, mode, health, system, and state controls have programmatic
  labels. Complexity fallback is an `aria-live` status message.
- `prefers-reduced-motion: reduce` suppresses decorative animation and reduces
  transitions without hiding state changes.
- Dark and light themes target WCAG 2.2 AA contrast: 4.5:1 for ordinary text,
  3:1 for large text and meaningful non-text UI boundaries. Automated contrast
  checks cover ordinary, muted, warning, and disabled text plus focus and
  selected boundaries in both themes; browser review covers composed states.
- At 375 CSS pixels the document has no horizontal page overflow. Wide relation
  tables scroll inside their workspace and remain navigable at 200% zoom.

Screen-reader acceptance traverses controls, graph nodes and edges,
selected-object summary, relatives, findings, metrics, raw-evidence disclosure,
and complexity status in that order. The graph is not the sole representation:
table mode is always available and becomes the deterministic high-complexity
fallback.

## Release evidence

A release candidate records:

1. `make check`, `make build`, and `git diff --check`;
2. `make topology-bench` output;
3. five-sample browser timing and heap results from `make topology-scale-demo`
   for all three profiles;
4. keyboard-only, reduced-motion, 375-pixel, 200%-zoom, light/dark contrast,
   and one screen-reader walkthrough;
5. a large-view screenshot showing the announced table fallback and a filtered
   graph below the limit.

Failure of a budget blocks the Scope-reborn release slice or requires an
explicitly documented smaller supported limit. It never silently weakens
evidence, enables a node probe, or relabels declared data as observed runtime
telemetry.
