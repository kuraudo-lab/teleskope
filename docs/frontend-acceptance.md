# Frontend migration acceptance evidence

Tracking: [#50](https://github.com/kuraudo-lab/teleskope/issues/50). Implementation and acceptance fixes are complete. Current acceptance/merge status is tracked in [PR #62](https://github.com/kuraudo-lab/teleskope/pull/62).

## Verified locally on 2026-10-02

- Clean locked dependency install, TypeScript check and production build passed. Vue and all assets are bundled: JS 209.74 kB (gzip 89.16 kB), CSS 50.65 kB (gzip 10.14 kB).
- Eight frontend tests passed: publication cancellation/stale analysis, topology delta protocol, graph layout, workload template relationships/escaping, and analysis rendering.
- `make check build` passed, including all Go tests, vet and fixture sanitization checks. Four release archive validation tests passed.
- Sixteen Chrome browser scenarios passed against the actual embedded binary, standalone HTML and synthetic production Hub/live endpoints. Coverage includes every resource page, namespace retention, search/details, theme, graph/table URL state, keyboard graph selection and zoom, delta/gap recovery, Hub filters/search/export/drilldown, and scoped AI result caching. Acceptance regressions also cover stable topology type colors during drilldown/focus/hover in both themes, plus visible and clickable EKS/Kubernetes dropdowns at 1440, 1160 and 390 pixels. Browser cases assert no uncaught page errors where applicable.
- Standalone HTML loaded with HTTP/HTTPS requests blocked and exported JSON successfully. It uses exactly the same bundle and components as served reports.
- The built binary served an archived snapshot from an isolated directory with empty PATH and nonexistent AWS/Kubernetes configuration, without frontend files. Original collection time and inventory survived export.
- Light topology and dark workload drawer screenshots were inspected. Acceptance server: `./bin/teleskope serve snapshot testdata/recorded/eks-demo-snapshot.json --listen 127.0.0.1:8093`.

## Topology measurements

macOS, installed headless Chrome 154, viewport 1440×1000; five render samples per fixture. These measure the existing renderer harness, not total initial page/network load. Graph mode was retained; selection and zoom were exercised.

| Fixture | Nodes | Relationships | Median | Worst | Median budget |
| --- | ---: | ---: | ---: | ---: | ---: |
| Small | 100 | 180 | 33.2 ms | 33.3 ms | 500 ms |
| Medium | 400 | 800 | 33.5 ms | 82.8 ms | 1500 ms |
| Large | 1500 | 2800 | 133.3 ms | 161.2 ms | 4000 ms |

Reproduce with `cd web && PLAYWRIGHT_CHANNEL=chrome npm run test:e2e` after `make build`. Default browser is Playwright Chromium. `web/test-results/results.json` includes the measurement attachments; artifacts are intentionally untracked.

## Release gates and boundaries

CI rebuilds committed assets and rejects drift, runs frontend/Go/browser tests, creates all six platform archives and smoke-tests each native binary including snapshot serving. Local macOS success does not itself prove the other operating systems; their status is available on the PR checks.

Fixtures are synthetic or de-identified. Scoped AI uses a mocked provider response; no new real-cluster or external-model verification is claimed. Replay recomputes Advisor and topology using the installed version, while preserving snapshot evidence and collection time. It cannot reconstruct observations, event history or AI artifacts absent from the archive.

`serve snapshot` is the standard rich archive entry. Self-contained HTML is retained because it adds packaging only. There is one application source, no separate legacy live script or Hub HTML application. Existing graph algorithms and escaped cell/evidence formatters remain isolated JavaScript modules; application/component state and transport are TypeScript.
