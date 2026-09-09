# README Product Preview Provenance

## Trigger

The user asked to make the README more vivid, add the product icon, and provide
a preview before accepting the change. After reviewing the generated preview,
the user approved landing the README update.

## Scope

This record covers the user-facing README presentation:

- `README.md`

It does not change the CLI, release pipeline, install scripts, report rendering,
or project assets.

## Conversation Summary

The README previously opened with a plain heading and engineering-oriented
status text. The accepted direction was to make the first screen feel more like
a product page while staying compatible with GitHub-flavored Markdown. The new
opening places the existing Teleskope icon above a centered title, adds a concise
tagline, and includes status badges for CI, latest release, and supported
platforms.

The README structure was also tightened so a new reader sees installation,
first commands, report output, and live inventory before the deeper release and
development notes. A local preview HTML file was generated from GitHub Markdown
rendering and checked for the icon, badges, quick install section, and report
artifact table.

## Design Diff

`README.md` should:

- show `assets/teleskope-icon-128.png` at the top
- center the product name and tagline using GitHub-compatible HTML
- show badges for CI, latest release, and supported platforms
- replace the old status-first opening with a product-oriented overview
- move install instructions near the top
- add a `Quick start` section with common scan commands
- add a `What you get` table for generated report artifacts
- rename `Live web inventory` to `Live inventory`
- keep release, development, and layout details after the reader-facing sections

## Decisions

- Use the existing repository icon instead of generating a new visual asset.
- Keep the README as plain GitHub Markdown plus minimal allowed HTML for
  centering, because GitHub strips most custom styling.
- Put installation and first scan commands before implementation status so the
  page serves new users first.
- Use a table for report artifacts because the values are easier to compare than
  a dense paragraph.
- Keep deeper operational caveats for live inventory because they matter for
  trust and security boundaries.

## Rejected Alternatives

- Do not build a custom landing page outside README for this step. The user's
  request was specifically to improve README presentation.
- Do not add new generated graphics. The repository already has product icon
  assets that match the project.
- Do not use complex HTML/CSS in README. GitHub's renderer would strip or alter
  much of it.
- Do not remove release and development notes; they remain useful for
  contributors.

## Constraints

- GitHub README rendering supports limited HTML and remote badges.
- The icon path must stay relative to the repository so it renders on GitHub.
- The README should remain readable as Markdown source.
- Preview was generated as `/tmp/teleskope-readme-preview.html`; browser
  security policy blocked opening that file through the in-app browser, so the
  rendered HTML structure was verified directly.

## Evaluation Plan

- Generate a GitHub Markdown preview HTML for `README.md`.
- Verify the rendered preview contains the icon, badges, `Quick install`, and
  `What you get` table.
- Run `git diff --check`.
- Review the rendered preview in the Codex/app preview surface when available.

## Open Questions

- None.

## Links

- Design files: `README.md`
- Related issues/tasks: current Codex task
