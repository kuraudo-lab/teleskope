# Public Demo Reports And Screenshot Plan

## Open the samples

The [demo catalog](../demo/index.html) links to all five requested areas. Clone
or download the repository and open `docs/demo/index.html`, or run:

```sh
python3 -m http.server 8090 --bind 127.0.0.1 --directory docs/demo
```

Open http://127.0.0.1:8090. GitHub's file viewer shows HTML source, not the
interactive report. No public hosting deployment is implied by these links.

| Area | Artifact | Walkthrough |
| --- | --- | --- |
| EKS | [Source report](../demo/source/index.html) | EKS menu → Overview, Compute, Add-ons; one managed nodegroup and the EBS CSI add-on. |
| Gateway | [Source report](../demo/source/index.html) | Kubernetes menu → Network; HTTPRoute `checkout` references Gateway `public` and Service `checkout-api`. Overview shows the workload relationships. |
| Storage | [Source](../demo/source/index.html) and [target](../demo/target/index.html) | Kubernetes menu → Storage; `orders-data` is Bound to `demo-orders-pv`, using `demo-block`. Source enables expansion; target disables it. |
| Hub | [Fleet sample](../demo/hub/index.html) | Two clusters with different Kubernetes versions and intentional partial EKS coverage. Links to cluster reports, fleet JSON, and Markdown. |
| AI analysis | [Output example](../demo/ai/index.html) | Hand-authored illustrative comparison, observed/inferred findings, evidence paths, limitations, JSON and Markdown downloads. No model was called. |

For full Hub interaction (filters, search, exports, cluster drill-down):

```sh
go run ./scripts/demo -serve
# Open http://127.0.0.1:8091; Ctrl-C to stop
```

This uses the production Hub handler with an in-memory synthetic fleet and a
fixed loopback listener. It never contacts AWS, Kubernetes, or an AI provider;
AI execution is disabled. The static fleet page is a portable summary, not the
full Hub application. Browser AI output is provided separately in the catalog.

## Reproduce and refresh

From the repository root, with the Go version required by `go.mod` installed:

```sh
go run ./scripts/demo
go test ./scripts/demo
# Optional isolated output directory:
go run ./scripts/demo -out /tmp/teleskope-demo
```

The generator embeds `scripts/demo/fixtures/{source,target,analysis}.json` and
uses the production report, Advisor, Hub, and analysis renderers/models. It does
not read kubeconfig, AWS profiles, local AI configuration, or the older recorded
EKS fixture. The first Go run may download build dependencies.

`docs/demo` is committed so readers need no Go installation. Source and target
each include self-contained HTML, snapshot JSON, Advisor JSON, and summary
Markdown. Hub includes fleet JSON and both remote-write envelopes. AI includes
structured JSON and Markdown. Generation overwrites these known output files;
use a separate output directory when experimenting. Do not edit generated files.

The fixed fixture timestamp is preserved. The generator replaces the Hub
Markdown renderer's wall-clock generation line with a labeled demo timestamp to
keep artifacts reproducible. Tests compare regenerated files byte-for-byte with
the committed outputs, validate catalog links, Gateway/storage references,
privacy constraints, and the seeded production Hub endpoints. Regenerate and
commit the artifact changes after report UI or model changes.

## Data provenance and limitations

Every input was authored for this demo; none came from a real cluster or model
conversation. Fictional cluster names are `source-demo` and `target-demo`, with
`checkout`, `platform`, and `observability` namespaces. Endpoint and image hosts
use the reserved `example.invalid` domain; the AWS account is `000000000000`.
No credentials, Secret values, ConfigMap contents, personal paths, customer
names, or real network addresses are included. Standard Kubernetes/AWS API names
and CSI driver identifiers remain intact so the product recognizes capabilities.
Automated marker checks supplement, rather than replace, review of fixture edits.

Kubernetes versions are illustrative fixture values, not a support matrix.
EKS upgrade insights are intentionally skipped, so upgrade readiness remains
unknown. Unrepresented collections must not be interpreted as complete evidence.
Gateway references do not prove traffic health, and Bound volumes do not prove
backup/restore readiness. The AI example illustrates the output format; its
inferences are distinct from inventory facts and deterministic Advisor output.

## Sharing

Use the catalog for README and walkthrough links, or publish the entire
`docs/demo` directory to a static host while preserving relative paths. The two
cluster HTML files can also be shared individually. Review modified fixtures
before publishing; no deployment or hosted URL is created by the generator.

## Screenshot Set

Prepare five images. Product Hunt can work with fewer, but five gives the page
room to tell a real story.

1. Static report overview with the cluster summary visible.
2. Topology view showing workloads and relationships.
3. Advisor evidence showing capability status and limitations.
4. Migration comparison showing source and target gaps.
5. Multi-cluster hub view showing fleet-level inventory.

Each screenshot should include enough real UI to be legible at Product Hunt
gallery size. Avoid terminal-only screenshots except for a small install or scan
step in the video.

## Demo Video Script

Target length: 45 to 75 seconds.

Opening:

> Before you migrate or upgrade a Kubernetes cluster, you need the shape of the
> system and the evidence behind it.

Middle:

> Teleskope scans EKS and Kubernetes inventory into local report artifacts. Open
> the static report to inspect topology, platform capabilities, resource
> details, and collection coverage. Then compare source and target reports to
> find migration gaps before the change window.

Close:

> Teleskope is open source and early. Try the sample report, run a scan, and
> tell us what cluster facts you need before your next change.

## Landing Page Sections

The first version can be very small:

- Product name and primary message.
- Sample report button.
- Install command and GitHub link.
- Three use cases: migration planning, upgrade review, cluster handoff.
- Three product views: report, advisor, compare.
- Data-handling note.
- Feedback CTA.

## Refresh Cadence

Review this file whenever a feature changes the launch story:

- After each release candidate.
- After report UI changes.
- After compare or advisor semantics change.
- Two weeks before Product Hunt launch.
- Two days before Product Hunt launch.

