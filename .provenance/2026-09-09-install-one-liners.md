# Install One Liners Provenance

## Trigger

The user asked to update the README with one-line installation commands for
different platforms, review the approach before writing files, then implement
the scripts and actually test them while cleaning up local traces.

## Scope

This record covers user-facing installation documentation and helper scripts:

- `README.md`
- `scripts/install.sh`
- `scripts/install.ps1`

It does not change the release archive format, GitHub Actions release workflow,
or binary build configuration.

## Conversation Summary

The accepted design keeps README commands short and delegates the platform
selection, checksum verification, extraction, and installation details to
script files in the repository. The Unix one-liner fetches `scripts/install.sh`
from `main` and pipes it to `sh`. The Windows one-liner fetches
`scripts/install.ps1` and evaluates it in PowerShell.

The scripts install the latest stable GitHub Release. Unix targets Linux and
macOS, detects `amd64` and `arm64`, downloads the matching `.tar.gz` archive and
`checksums.txt`, verifies SHA-256, and installs to `$HOME/.local/bin` unless
`INSTALL_DIR` is set. Windows detects `amd64` and `arm64`, downloads the matching
`.zip` archive and `checksums.txt`, verifies SHA-256 with PowerShell, and
installs to `$env:LOCALAPPDATA\Microsoft\WindowsApps` unless `INSTALL_DIR` is
set.

Testing found macOS curl could hit `curl: (16) Error in the HTTP2 framing layer`
when downloading GitHub release assets. The Unix script therefore forces
HTTP/1.1 and retries downloads. The user provided local proxy variables for the
test run. With those proxy settings, the Unix installer downloaded the latest
release, installed Teleskope 0.5.0 into a temporary install directory, and
`teleskope --version` returned `teleskope 0.5.0`. The temporary install directory
and installer temp directories were removed.

## Design Diff

`README.md` should replace the manual-only binary download section with quick
install commands for Linux/macOS and Windows PowerShell, then keep manual
download and checksum guidance below those commands.

`scripts/install.sh` should:

- resolve the latest release tag from GitHub
- map `uname -s` to `linux` or `darwin`
- map `uname -m` to `amd64` or `arm64`
- download the expected release archive and `checksums.txt`
- verify SHA-256 with `sha256sum` or `shasum`
- extract only `teleskope`
- install into `INSTALL_DIR` or `$HOME/.local/bin`
- clean its temporary directory on exit

`scripts/install.ps1` should:

- query the latest release from GitHub's API
- detect native Windows architecture, including ARM64 from `PROCESSOR_ARCHITEW6432`
- download the expected `.zip` archive and `checksums.txt`
- verify SHA-256 with `Get-FileHash`
- extract `teleskope.exe`
- install into `INSTALL_DIR` or the user's WindowsApps directory
- clean its temporary directory in a `finally` block

## Decisions

- Keep the README one-liners short so users can copy the install command without
  reading a large inline script.
- Install into user-writable directories by default to avoid requiring sudo or
  administrator privileges.
- Verify checksums before installing, matching the release pipeline's package
  integrity contract.
- Use the latest stable release by default because the release pipeline already
  rejects prerelease tags.
- Force HTTP/1.1 in the Unix curl script because the local real download test
  reproduced HTTP/2 framing errors against GitHub release assets.
- Leave PowerShell runtime testing for a Windows environment because this macOS
  machine does not have `pwsh` installed.

## Rejected Alternatives

- Do not paste long install scripts directly into README. That would make the
  documentation harder to review and maintain.
- Do not require `jq` for install commands. JSON parsing is avoided in the Unix
  path by resolving the latest release redirect.
- Do not install into `/usr/local/bin` by default. That often requires elevated
  privileges and is a poor default for a one-liner.
- Do not skip checksum verification in the convenience installer.

## Constraints

- Release assets are named `teleskope_${version}_${os}_${arch}.tar.gz` for Unix
  targets and `teleskope_${version}_windows_${arch}.zip` for Windows.
- Windows archives contain `teleskope.exe`; Unix archives contain `teleskope`.
- The installer must support `amd64` and `arm64` only, matching the release
  pipeline targets.
- Temporary test directories must be removed after validation.
- The README command that fetches from `main` only works after these scripts are
  merged or pushed to that branch.

## Evaluation Plan

- Run `sh -n scripts/install.sh`.
- Run `git diff --check`.
- Test Unix installation against the latest release with proxy variables:
  `http_proxy`, `https_proxy`, and `all_proxy`.
- Install into `/tmp/teleskope-install-test`, run
  `/tmp/teleskope-install-test/teleskope --version`, and confirm the installed
  version matches the latest release.
- Confirm `/tmp/teleskope-install-test` and `/tmp/teleskope-install.*` do not
  remain after the test.
- Review `scripts/install.ps1` manually on this machine; run it later in a
  Windows or PowerShell-capable environment.

## Open Questions

- Whether to add CI coverage for the Windows installer script in a future
  workflow.

## Links

- Design files: `README.md`
- Related issues/tasks: current Codex task
