<p align="center">
  <img src="assets/teleskope-icon-128.png" alt="Teleskope icon" width="128" height="128">
</p>

<h1 align="center">Teleskope</h1>

<p align="center">
  EKS and Kubernetes inventory for people who need to understand a cluster before they change it.
</p>

<p align="center">
  <a href="https://github.com/kuraudo-lab/teleskope/actions/workflows/ci.yml"><img alt="CI" src="https://github.com/kuraudo-lab/teleskope/actions/workflows/ci.yml/badge.svg"></a>
  <a href="https://github.com/kuraudo-lab/teleskope/releases"><img alt="Latest release" src="https://img.shields.io/github/v/release/kuraudo-lab/teleskope?sort=semver"></a>
  <a href="https://github.com/kuraudo-lab/teleskope/releases"><img alt="Platforms" src="https://img.shields.io/badge/platform-linux%20%7C%20macOS%20%7C%20windows-0f766e"></a>
</p>

Teleskope collects EKS metadata, Kubernetes API inventory, platform component
configuration, and workload topology into reviewable reports. It is built for
the moment before a migration, upgrade, incident review, or platform cleanup:
you need the shape of the system, the raw evidence behind it, and a local view
you can share with teammates without granting cluster access.

## Quick install

Install the latest stable release with one command.

Linux/macOS:

```sh
curl -fsSL https://raw.githubusercontent.com/kuraudo-lab/teleskope/main/scripts/install.sh | sh
```

Windows PowerShell:

```powershell
irm https://raw.githubusercontent.com/kuraudo-lab/teleskope/main/scripts/install.ps1 | iex
```

By default, Unix installs to `$HOME/.local/bin` and Windows installs to
`$env:LOCALAPPDATA\Microsoft\WindowsApps`. Set `INSTALL_DIR` to choose another
directory.

Manual downloads are available from
[GitHub Releases](https://github.com/kuraudo-lab/teleskope/releases).
Packages cover Linux, macOS (`darwin`), and Windows on `amd64` and `arm64`.
Choose `arm64` for Apple Silicon and `amd64` for Intel/AMD x64 machines.
Linux/macOS archives use `.tar.gz`; Windows archives use `.zip` and contain
`teleskope.exe`.

Install scripts and release workflows verify the archive's SHA-256 against
`checksums.txt` before installing. To verify manually, use `sha256sum` on Linux,
`shasum -a 256` on macOS, or `Get-FileHash -Algorithm SHA256` in PowerShell.
Confirm the installation with `teleskope --version`. The initial packages do
not include Developer ID notarization or Windows Authenticode signatures.

## Quick start

Collect an EKS and Kubernetes inventory:

```sh
teleskope scan eks --cluster my-cluster
```

Use AWS profile and region overrides when needed:

```sh
teleskope scan eks --cluster my-cluster --profile prod --region ap-northeast-1
```

Collect Kubernetes-only inventory from your current kubeconfig:

```sh
teleskope scan k8s --kube-context prod
```

The default output is a timestamped report directory under the current working
directory. Use `--output-dir` to choose a parent directory, `--output human` for
a terminal summary, or `--output json` for the full snapshot on stdout.

## What you get

Each report keeps raw evidence and a human-readable view side by side:

| Artifact | Purpose |
| --- | --- |
| `snapshot.json` | Complete inventory snapshot for automation and later analysis |
| `eks.json` | AWS-side EKS metadata, add-ons, node groups, access entries, and Pod Identity associations |
| `kubernetes.json` | Kubernetes API resources, workloads, networking, storage, RBAC, and platform objects |
| `advisor.json` | Evidence-backed cluster capability assessment |
| `coverage.json` | Collection coverage and errors, so partial inventory is explicit |
| `summary.md` | Markdown review notes for quick scanning and sharing |
| `index.html` | Interactive static report with filters, topology, and resource details |

Scan progress is written to stderr while the final report path is written to
stdout. Kubernetes Secret values are not collected; Secrets are recorded as
metadata-only objects.

## Cluster advisor

The **Advisor** tab summarizes Ingress and Gateway classes, storage RWX binding
evidence, volume expansion configuration, and custom API declarations. Expand
each result to see evidence and limitations. It distinguishes unknown capability
from unsupported configuration and marks retained live data as stale. See
[Advisor semantics](docs/advisor.md) for the first version's scope.

## Migration comparison

Compare two scan reports to plan migration from a source cluster to a target
cluster:

```sh
teleskope compare --source ./source-scan --target ./target-scan --output markdown
```

`--source` and `--target` accept either a scan report directory or a
`snapshot.json` file. The comparison is deterministic and does not call an LLM.
It highlights Kubernetes version differences, advisor capability gaps, missing
or changed API resources, CRDs, networking classes, storage classes, CSI drivers,
runtime classes, node platforms, workload references, and incomplete collection
coverage. Supported outputs are `human`, `markdown`, and `json`.

## Optional LLM analysis

LLM-backed analysis is opt-in and runs as a post-processing step against existing
scan results. It uses an OpenAI-compatible `/v1/chat/completions` endpoint and
keeps `scan`, `compare`, `advisory`, `serve`, and exports deterministic when the
LLM is not used.

Create `$HOME/.teleskope/config.yaml`:

```yaml
llm:
  provider: openai-compatible
  base_url: https://api.openai.com/v1
  api_key_env: OPENAI_API_KEY
  model: <model-name>
  timeout: 5m
  # Set to false for providers that reject OpenAI JSON mode or return plain text, such as some gpt-oss-120b deployments.
  json_mode: true
```

`api_key` can be used instead of `api_key_env` for local-only setups. Set
`json_mode: false` when an OpenAI-compatible provider rejects `response_format`
or returns plain text instead of JSON; Teleskope will preserve that response as
plain text analysis. Analyze a single scan or a migration comparison with:

```sh
teleskope analyze scan ./scan-report --output markdown
teleskope analyze compare --source ./source-scan --target ./target-scan --output markdown
```

Use `--output context` to inspect the trimmed model context without calling the
provider. Use `--config` to point at a different config file. `--web-search`
records that web-search-backed conclusions are allowed in the prompt, but the
OpenAI-compatible adapter does not expose a provider-specific search tool in the
first implementation.

In `teleskope serve`, the live web UI shows an **Analyze** button after the first
snapshot is available. Clicking it runs the same scan analysis from the local
server process using `$HOME/.teleskope/config.yaml`; browser refreshes never call
the provider automatically and the browser never receives the provider API key.

For a local web workflow, start the advisory UI and choose the two
`snapshot.json` files in the browser:

```sh
teleskope advisory
```

The default listener is `127.0.0.1:8080`; use `--listen` to change it. The
browser uploads the two snapshots to the local server, which runs the same
deterministic comparison module used by `teleskope compare`.

## Live inventory

Run a local web server with the same embedded UI as the offline report:

```sh
teleskope serve k8s --kube-context prod
teleskope serve k8s --kubeconfig ./config --interval 5m --timeout 5m
teleskope serve eks --cluster my-cluster --kube-context prod --aws-interval 15m
teleskope serve eks --cluster my-cluster --skip-kubernetes
```

Open [localhost:8080](http://localhost:8080). The default listener is
`127.0.0.1:8080`; use `--listen` to change it. There is no built-in HTTP
authentication. Inventory can include ConfigMap contents and infrastructure
details; keep the listener local or put it behind an authenticated access
boundary.

Each source scans immediately, then waits after its previous scan finishes:
Kubernetes defaults to 5 minutes, AWS to 15 minutes, with up to 10% added
jitter. `--timeout` applies independently to each attempt. Slow scans never
overlap. Clients and credentials are reused between scans; Kubernetes and AWS
failures do not stop the web server or the other source. For EKS, select a
kubeconfig context that points to the same cluster as `--cluster`.

All browsers read one in-memory view. The page checks for updates every 5
seconds using conditional HTTP requests and preserves its filters, active
section, topology pan/zoom, scroll positions, and selected details. “Pause page
updates” pauses that browser only. Ctrl-C or SIGTERM stops the server and
collection.

The status panel distinguishes loading, partial coverage, failed initial
collection, and stale retained data. Collection details show errors and the next
attempt time. Initial partial inventory is usable. If a later attempt fails, or
a previously observed resource cannot be refreshed, the **entire previous result
for that source** is retained and marked stale. A successful empty list removes
old objects. AWS and Kubernetes publication times are independent; the combined
snapshot is not a transactional cluster-wide observation.

`GET /api/snapshot` returns an envelope with `revision`, `snapshot` (null until
the first usable result), and per-source `sources` statuses. The revision changes
when source data is published; ETags also change for status-only updates. HTTP
requests never trigger scans. Live state is memory-only and is rebuilt on
restart. Existing `scan` commands and offline artifacts remain available.

This first live version supports polling. Watch/informer mode and packaged
in-cluster deployment are subsequent steps.

## Releasing

After merging the desired changes into `main`, create and push a stable tag:

```sh
git switch main
git pull --ff-only
git tag -a v0.1.0 -m "Release v0.1.0"
git push origin v0.1.0
```

Only `vMAJOR.MINOR.PATCH` is supported; prerelease tags are rejected. The tag's
commit must be reachable from `main`. GitHub Actions runs tests and vet, builds
six archives with GoReleaser 2.18.1 and CGO disabled, and tests each archive on a
native runner. It then uploads a draft release, downloads and verifies every
attachment, and automatically publishes it as the latest release. Report
smoke tests use a missing kubeconfig and need no cloud credentials.

If publication fails, fix the cause and rerun the workflow within the three-day
artifact retention period, or rerun all jobs to rebuild. An existing draft can
be repaired; an already-published release cannot be overwritten by this workflow.
Use a new version for corrections. Publish stable tags in increasing version order.

The workflows use standard GitHub-hosted runners and GitHub Releases storage.
No self-hosted runner, external storage, PAT, or AWS credentials are required.
The publish job requests `contents: write` using the built-in `GITHUB_TOKEN`;
all build and test jobs have read-only repository permissions. Repository or
organization policies must allow these Actions and hosted runners.

PRs and pushes to `main` run the same build and smoke tests in local snapshot
mode, without creating a release. To check packaging locally with GoReleaser:

```sh
goreleaser check
goreleaser release --snapshot --clean --skip=publish
```

## Development

Requires Go 1.26.0 or later. Make is optional.

```sh
make build
./bin/teleskope --help
./bin/teleskope --version
make check
```

Without Make:

```sh
go build -o bin/teleskope ./cmd/teleskope
go test ./...
go vet ./...
```

Set a build version with `make build VERSION=0.1.0`.

## Layout

```text
cmd/teleskope/  Executable entry point
assets/         Project visual assets such as the Teleskope PNG icon
internal/awseks/     AWS-side EKS collector
internal/buildinfo/  Build metadata
internal/cli/        Internal command-line implementation
internal/inventory/  Snapshot data model
internal/k8s/        Kubernetes API collector
internal/report/     Timestamped report artifacts
internal/render/     Output renderers
docs/                Architecture and design notes
go.mod               Module definition
Makefile             Local development commands
```

Internal packages are added as their functionality is implemented. See
[architecture](docs/architecture.md) for intended module boundaries.
