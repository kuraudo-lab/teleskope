# Embedded UI Runtime Provenance

## Trigger

The user asked to start work on GitHub issue #4, "Refactor embedded UI
runtime", after accepting the scoped analysis request work. The issue focuses
on preserving Teleskope's single-binary embedded UI delivery while reducing
string replacement and global script coupling as the UI grows.

After reviewing and accepting the implementation, the user asked to record
provenance and then create the implementation commit, with the implementation
commit explicitly closing issue #4 if the work is complete.

## Scope

This record covers the embedded report UI runtime used by:

- `internal/report`
- live `serve` report rendering
- hub cluster drilldown rendering

The behavior covered is offline report rendering, live-mode body attributes,
boot and endpoint configuration, script/style injection, script-data escaping,
and tests that prove hub drilldown still receives cluster-scoped endpoints.

## Conversation Summary

Issue #4 was groomed as an architecture refactor for the embedded UI. The
existing implementation kept the UI in the Go binary through `go:embed`, which
remains the preferred delivery model, but the rendering code had accumulated
one-off replacements for snapshot JSON, advisor JSON, markdown text, live body
markers, live CSS, injected JavaScript, and endpoint rewrites.

The accepted direction was to keep the single-binary model and avoid a frontend
build chain while introducing one internal renderer for the shared shell.
Offline reports, live serve, and hub drilldown should all route through the
same interface. Live and hub-specific endpoint paths should be represented as a
boot configuration JSON blob, rather than rewriting hard-coded strings inside
the live script.

## Design Diff

- Add `internal/report/ui_runtime.go` with `RenderUI`, `UIMode`,
  `UIEndpoints`, and `UIRenderOptions`.
- Move boot config rendering, mode-specific body opening, payload selection,
  style injection, script injection, and script text escaping into the shared
  runtime.
- Route offline `HTML` rendering through `RenderUI` with snapshot/advisor/
  markdown payloads.
- Route `LiveHTMLWithOptions` through `RenderUI` with live mode, embedded live
  CSS, embedded live JavaScript, and endpoint config.
- Add a `boot-config` script tag to the embedded HTML template and make export
  behavior read endpoint paths from that config.
- Make `live.js` read snapshot and analysis endpoints from boot config instead
  of relying on Go-side JavaScript string replacement.
- Extend report and hub tests for endpoint config, script escaping, live body
  marker behavior, and hub drilldown boot config.

## Decisions

- Keep `go:embed` in `internal/report/html.go` so report templates and live
  JavaScript remain compiled into the Teleskope binary.
- Keep the UI runtime inside `internal/report` because the report shell is the
  shared asset used by offline, live, and hub drilldown pages.
- Use a JSON boot config script tag for endpoint paths. This makes endpoint
  configuration explicit data instead of source-code rewriting.
- Preserve the existing exported `HTML`, `LiveHTML`, and `LiveHTMLWithOptions`
  functions so callers do not need to change.
- Keep hub fleet HTML separate for now. Issue #4 acceptance is about offline
  report, live serve, and hub drilldown sharing the report UI renderer.
- Escape `</` in script data and injected scripts so embedded JSON, markdown,
  and scripts cannot accidentally terminate a script tag.

## Rejected Alternatives

- Do not add a frontend build chain in this issue; it would conflict with the
  explicit single-binary constraint and is larger than the current refactor.
- Do not continue replacing endpoint literals in JavaScript; it couples Go
  rendering to exact source snippets and becomes brittle as the UI grows.
- Do not split live and hub drilldown into separate renderers; that would make
  future page-level analysis and endpoint behavior drift.
- Do not move all hub fleet UI into the report runtime in this slice; it is a
  different page shell and can be considered separately.

## Constraints

- Single-binary embedded UI delivery must remain intact.
- Offline report, live serve, and hub drilldown must render through one report
  UI interface.
- Existing HTML output and endpoint behavior should remain compatible.
- Browser-side provider credentials remain out of scope and must not be
  introduced.
- Tests should validate runtime behavior without requiring a browser or network
  service.

## Evaluation Plan

- Add report tests proving custom live endpoint paths are rendered into boot
  config and are not inlined by JavaScript source rewriting.
- Add report tests proving script data and injected scripts escape `</script>`.
- Add report tests proving live body markers come from render mode and are not
  present in offline HTML.
- Add hub tests proving cluster drilldown HTML includes cluster-scoped boot
  config endpoints.
- Run full repository validation through `make check`, `make build`, and
  `git diff --check`.

Validation was performed after implementation using:

- `GOCACHE=/private/tmp/teleskope-go-cache GOMODCACHE=/private/tmp/teleskope-go-mod-cache make check`
- `GOCACHE=/private/tmp/teleskope-go-cache GOMODCACHE=/private/tmp/teleskope-go-mod-cache make build`
- `git diff --check`

## Open Questions

- Whether the hub fleet shell should later use a sibling UI runtime.
- Whether future UI growth should move live styles out of `html.go` into an
  embedded CSS asset while still keeping single-binary delivery.
- Whether a later frontend refactor should introduce typed client boot config
  validation in JavaScript.

## Links

- Design files: `docs/architecture.md`
- Related issues/tasks: https://github.com/kuraudo-lab/teleskope/issues/4
