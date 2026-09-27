# Scope Reborn Walkthrough

This walkthrough is the product acceptance path for the Scope-inspired
topology preview. Every required step uses only de-identified recorded or
generated evidence and requires no Kubernetes, AWS or external-provider
credentials.

## 1. Verify the release contract

```sh
make scope-release-check
```

Expected: `VALID`, four ordered slices, Popeye issue #40 identified as the
product graduation dependency, external evidence marked optional, and the four
rejected security capabilities shown as disabled. This checks the contract; it
does not claim that #40 is complete.

## 2. Walk the recorded investigation loop

```sh
go run ./cmd/teleskope serve snapshot testdata/recorded/eks-demo-snapshot.json --listen 127.0.0.1:8081
```

Open `http://127.0.0.1:8081` and complete this path:

1. Use Search or a finding link to select a workload or Service.
2. Open Topology and confirm the selected object is preserved.
3. Switch Application, Workload, Runtime and Infrastructure views; use a
   relative in Inspector to cross into the required view.
4. Inspect scoped findings, metrics, metadata and raw evidence. Confirm every
   runtime/metric section names its source and that absent observed connections
   are `unavailable`, not inferred from Service selectors.
5. Copy the URL, reload it, and confirm view, mode, filters, focus and selection
   are restored.
6. Repeat selection in table mode and with keyboard only. Toggle light/dark
   theme and check focus remains visible.

## 3. Observe live continuity

```sh
make topology-live-demo
```

Open `http://127.0.0.1:8093`. The local harness loads the de-identified Recorded
fixture, selects `aws-node`, changes a different workload and publishes it
through the real live Store and `/api/topology` endpoint. It then publishes 17
bounded revisions so the browser baseline expires. The visible
`Topology live continuity evidence` panel must report:

- `delta.responseKind: "delta"` and `delta.accepted: true`;
- `selectionStable: true` and `layoutStable: true`;
- `revisionGap.responseKind: "full"` and `fullReset: true`;
- `nodeProbe: false`.

This is a deterministic local continuity demo, not simulated observed-network
evidence. Production live and hub still use the existing Kubernetes watch/poll
and remote-write boundaries; do not install a node probe or add topology-only
API polling.

## 4. Exercise scale degradation

```sh
make topology-scale-demo SCALE_PROFILE=large
```

Open `http://127.0.0.1:8092`. The visible evidence panel must report exactly
1,500 nodes, 2,800 edges and `table` mode after five painted renders. The live
status explains the 400-node/800-edge SVG limits and how to narrow the view.
Repeat with `small` and `medium`; both use graph mode. Verify keyboard access,
375-pixel layout, 200% zoom, reduced motion and a screen-reader traversal.

## 5. Review the security boundary

The walkthrough passes only if all answers are “no”:

- Was a privileged per-node Probe or DaemonSet installed?
- Did browser JavaScript receive cluster or provider credentials?
- Was declared or inferred topology presented as observed traffic?
- Were terminal, delete, scale or lifecycle write controls exposed?
- Did hub data cross cluster identity or coverage boundaries?

Record browser version, machine, commit and viewport with the timing and heap
samples required by `docs/topology-scale-performance-accessibility.md`. Passing
this walkthrough makes the topology slices demo-ready; the **Scope reborn**
product label still waits for the Popeye graduation gate in issue #40.
