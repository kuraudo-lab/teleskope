# EKS node upgrade watch provenance

## Trigger and accepted design
The user requested an EKS node-upgrade observation view and accepted
`docs/mockups/eks-node-upgrade.html`, then requested pixel-level integration into
the real Teleskope system. Six-sided node tiles must have soft rounded corners;
selection uses the border only, never a SELECTED caption.

## Decisions and scope
- Add EKS > Node watch beside existing Upgrades, with a link from Upgrades.
  Preserve existing upgrade insights, top navigation, themes, exports and pause.
- Reproduce the accepted metrics/map/inspector/ASG activity/workload layout using
  scoped Vue styles and existing tokens. No simulation controls or mock facts in
  production. Use real snapshot publications in live, recorded and offline modes.
- Count bound non-terminal Pods per node, split by DaemonSet ownership. Track
  unbound Pending and terminating Pods separately. Counts use whole-cluster scope.
- Distinguish Ready, NotReady, Unknown, cordon and scheduling taints. Never infer
  drain completion or actual PDB eviction blockage from counts alone.
- Join Kubernetes nodes to EC2 by provider instance ID, with name fallback only
  when provider ID is absent. Preserve unknown and collection coverage states.
- Add Node creation time, Pod deletion time, EC2 launch time, per-ASG-instance
  template and lifecycle facts, and recent ASG activities. Do not treat the ASG
  target launch template as the version used by every instance. Legacy snapshots
  without instance-template provenance must not count as verified target matches.
- Scope ASG activity requests to nodegroup ASGs; bound history to 100 activities
  per group and mark truncation/denial as partial coverage. No mutation APIs.
- Show source timestamps, recorded/offline/paused/error states, and missing data.
  PDB zero allowance is a risk indicator, not proof of an active blocked drain.

## Evaluation
Focused Go collector mapping/coverage tests; domain tests for ownership, missing
coverage, target comparison, lifecycle and joins; component tests for filtering,
selection refresh/removal, keyboard details and rounded tiles. Build embedded
assets, full make check/build, asset check and real browser verification against
recorded data, plus explicitly synthetic fixture coverage where required.

## Constraints
No cluster mutation, no fake production data, no unrelated docs/research edits.
This is a structured conversation summary. AWS activity permissions may need
adding to an operator's read-only policy; denial must remain visible.

## Sources
- https://docs.aws.amazon.com/autoscaling/ec2/APIReference/API_DescribeScalingActivities.html
- https://docs.aws.amazon.com/autoscaling/ec2/APIReference/API_Instance.html
