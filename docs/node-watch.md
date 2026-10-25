# EKS Node watch

Open **EKS → Node watch**, or use the link on **EKS → Upgrades**. The view uses the
same snapshot publication as the other pages; there is no separate backend or
simulated production dataset. Top-bar pause, themes and exports remain available.

- Each rounded hexagon is one Kubernetes Node. The large number is non-DaemonSet
  Pods; the small number is DaemonSet-owned Pods. Only bound, non-terminal Pods
  count, including those with a deletion timestamp. All namespaces are included.
- Green means Ready and not cordoned, amber means Ready and cordoned, red means
  NotReady, grey means unknown Readiness. Cordon and taints are also shown in
  details; green does not assert that a particular Pod can schedule.
- Hover or keyboard-focus for node metadata; click to pin details. Selection
  follows the Node UID across publications. If that UID disappears, the panel
  explicitly reports its absence instead of displaying stale facts as current.
- Target comparison uses the **instance's** ASG launch template, not the group's
  configured target substituted for every instance. Symbolic `$Latest` / `$Default`
  and old snapshots without instance-template provenance remain unknown. The
  denominator is all observed Kubernetes nodes, not desired ASG capacity.
- Pending counts include only unbound Pending Pods. Terminating counts include
  non-terminal Pods with deletion timestamps. Workload readiness and PDBs are
  cluster-wide; PDB zero allowance is a risk signal, not proof of blocked eviction.
- EC2 instances not joined to a Kubernetes Node are listed separately. This may
  indicate startup, shutdown or different AWS/Kubernetes observation times.

## Collection and permissions

The existing read-only EC2 and Auto Scaling collection now preserves EC2 launch
time and per-instance launch-template/lifecycle facts. Kubernetes scans and watch
publications preserve Node creation and Pod deletion timestamps.

Add `autoscaling:DescribeScalingActivities` to the collector's read-only permissions
for ASG activity history. Requests are scoped to the ASGs referenced by managed
nodegroups, never an unfiltered account-wide query. Each poll requests up to 100
recent/in-progress activities per group. A continuation token marks coverage as
partial with an explicit omitted-history reason. An access denial does not discard
other infrastructure facts. This view does not collect instance refresh percentage
or Kubernetes eviction events and does not infer either from activity descriptions.

AWS and Kubernetes have separate collection clocks. Live mode follows existing
collection intervals and the UI publication poll; it is not a guarantee of instant
AWS updates. Recorded and offline reports are labeled as historical. Missing data,
partial coverage, stale sources and connection errors remain visible.

Old snapshots still open. Missing creation/launch timestamps, ASG activities and
verified instance templates show as unavailable; no historical facts are invented.

## Validation

Domain tests cover Pod accounting, unknown/partial data, provider-ID joins and
legacy/symbolic templates. Component tests cover selection refresh/removal, search,
keyboard details and recorded/paused labels. Collector tests cover scoped activity
requests, truncation, access denial, actual template provenance and timestamps.

AWS references:
- https://docs.aws.amazon.com/autoscaling/ec2/APIReference/API_DescribeScalingActivities.html
- https://docs.aws.amazon.com/autoscaling/ec2/APIReference/API_Instance.html

## Local mock demo and UI validation

Use `make node-watch-demo` to build and start the shared, explicitly synthetic
local UI demo (no credentials). The equivalent manual commands are:

```sh
python3 scripts/node-watch-fixture.py
bin/teleskope serve snapshot testdata/synthetic/node-watch.json --listen 127.0.0.1:8768
```

Open `http://127.0.0.1:8768/?page=eks-node-watch`. The cluster title is
**MOCK · EKS upgrade demo**. This is the production UI and snapshot API backed by
mock data, not the standalone HTML design mock. The recorded label means the
scenario is static; it does not imply observations from a real cluster.

The same JSON fixture is used by browser validation. It includes 11 nodes across
two nodegroups, Ready/Cordoned/NotReady/Unknown states, 115 non-DS and 61 DS Pods,
4 unbound Pending Pods, 2 terminating Pods, old/new instance templates, launch
and registration times, three ASG activities and a zero-budget PDB risk signal.
Regenerate the committed JSON with the script whenever the scenario changes.
