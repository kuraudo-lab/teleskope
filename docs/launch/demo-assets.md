# Demo And Screenshot Plan

Use this plan to keep launch visuals aligned with the product as it changes.

## Sample Dataset

The public demo should use data that feels like a real platform without exposing
any real infrastructure.

Recommended shape:

- Two clusters: `source-prod` and `target-prod`.
- Three namespaces: `checkout`, `platform`, and `observability`.
- A mix of Deployments, Services, Ingress or Gateway resources, ConfigMaps,
  StorageClasses, PVCs, CRDs, and RBAC objects.
- One or two intentional migration gaps, such as a missing IngressClass, changed
  Kubernetes version, missing CRD, or storage capability difference.
- A partial collection example that demonstrates coverage without making the
  product look broken.

Avoid:

- Real account IDs, hostnames, IP ranges, ARNs, domain names, service names, or
  customer names.
- Secret values.
- Screenshots that reveal local paths, usernames, cloud account details, or
  kubeconfig names.

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

