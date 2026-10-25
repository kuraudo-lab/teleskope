# Node watch local mock validation

The user requests mock data for local UI validation and demonstration. Keep the
real Vue page and Go snapshot server; supply an explicitly synthetic snapshot
shared by the browser tests and local preview. Cover two nodegroups, rounded node
states, cordon, unknown readiness, terminating/pending Pods, per-instance templates,
ASG activities and PDB risk. Do not connect to AWS or Kubernetes. Preserve the
previous standalone mock as a design artifact. Verify the served snapshot API,
browser interactions and responsive layout with the shared fixture.

Real Go-server validation exposed a PDB mapping bug: read `podDisruptionBudgetDetails`,
not the name-only `podDisruptionBudgets` reference list. Include the correction
and cover PDB risk after Go serialization in the shared mock browser tests.

Mobile browser validation also exposed focus-triggered auto-scroll dismissing
the tooltip. Keep keyboard-focused node details positioned through scrolling.
