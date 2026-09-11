# Web Gateway Table Rendering Provenance

## Trigger

The user reported that scan output already contains Gateway API data, but the web report still shows the Gateway sections as empty.

## Scope

This record covers the web report rendering path for GatewayClass, Gateway, and GatewayRoute tables in `internal/report/report.html`, plus regression coverage in `internal/report/report_test.go`.

## Conversation Summary

After Gateway API inventory collection was expanded and committed, the user verified that scan can collect Gateway-related data. The remaining issue was isolated to the web UI: the Network section included table elements for Gateway classes, Gateways, and Gateway routes, but those table elements were never populated by `renderTables()`.

The fix should bind the existing table DOM IDs to `k.gatewayClasses`, `k.gateways`, and `k.gatewayRoutes`, and strengthen tests so they check for render calls rather than only checking that table IDs exist in the HTML template.

## Design Diff

No standalone design document was changed. The intended change is captured here:

- Keep the existing Network section layout and table IDs.
- Add render bindings for GatewayClass, Gateway, and GatewayRoute tables.
- Reuse existing JavaScript format helpers such as `gatewayListeners`, `gatewayMatches`, `gatewayParents`, and `gatewayBackends`.
- Add HTML template regression checks for the render calls.

## Decisions

- Treat this as a web rendering bug, not a collector or inventory schema bug, because scan JSON already contains the data.
- Preserve the current UI table structure and only add the missing `table(...)` calls.
- Strengthen report tests to catch missing render bindings for table IDs.

## Rejected Alternatives

- Do not change the Gateway API collector again for this issue because the reported failure is in Web display after collection succeeds.
- Do not redesign the Network page layout; the immediate issue is that existing tables are unpopulated.

## Constraints

- Keep the fix small and focused on Web report rendering.
- Avoid changing JSON schema or collected data shape.
- Preserve existing namespace/resource filtering behavior.

## Evaluation Plan

- Run `go test ./internal/report -count=1`.
- Run the full Go test suite with `go test ./...`.
- Run `git diff --check` before committing implementation.

## Open Questions

- None.

## Links

- Design files: `.provenance/2026-09-11-web-gateway-table-rendering.md`
- Related issues/tasks: user-reported Web report Gateway tables showing empty despite scan data being present.
