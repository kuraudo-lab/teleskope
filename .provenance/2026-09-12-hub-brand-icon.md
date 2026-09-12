# Hub Brand Icon Provenance

## Trigger

The user asked for the hub web page's top-left icon to use the Teleskope icon instead of the placeholder gradient mark.

## Scope

This record covers the hub fleet page brand mark rendered by `internal/hub/html.go`.

## Conversation Summary

The hub page currently uses a styled empty logo block while the project already ships Teleskope PNG assets. The accepted direction is to replace the placeholder mark with the real Teleskope icon in the embedded hub UI.

## Design Diff

No design file changes are included in this provenance commit. The implementation should preserve the existing hub layout and only swap the brand mark asset.

## Decisions

- Use the existing `assets/teleskope-icon-128.png` artwork as the source icon.
- Embed the icon into the hub package so `teleskope serve-hub` remains a self-contained binary and does not depend on the current working directory.
- Keep the 42px rounded logo slot so the surrounding sidebar spacing stays aligned with the current hub UI.

## Rejected Alternatives

- Referencing `/assets/teleskope-icon-128.png` from HTML was rejected because `serve-hub` does not serve static assets from the repository.
- Keeping the gradient placeholder was rejected because it does not match the user's requested brand icon.

## Constraints

- The hub UI must remain self-contained with no frontend build step.
- The visual change should not alter hub API behavior.
- Tests should verify the HTML contains the embedded icon data URL.

## Evaluation Plan

- Add or update a focused hub test for the branded icon markup.
- Run focused Go tests for `internal/hub`.
- Run the full project check after committing the implementation.

## Open Questions

- None.

## Links

- Design files: `docs/multi-cluster-hub.md`
- Implementation files: `internal/hub/html.go`, `internal/hub/teleskope-icon-128.png`, `internal/hub/store_test.go`
