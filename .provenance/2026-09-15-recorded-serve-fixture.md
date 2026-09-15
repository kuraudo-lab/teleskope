# Recorded Serve Fixture Provenance

## Trigger

The user needs to validate the Teleskope web UI without manually spinning up an
EKS cluster for every integration acceptance run. They asked to determine
whether `serve` should be extended or whether an existing recorded-data path can
be used, then capture the currently deployed AWS EKS data, sanitize it, and keep
it in the repository for long-term use.

## Scope

This record covers the single-cluster `serve` workflow, the live HTTP snapshot
view, and repository fixture data for UI acceptance. It does not change
multi-cluster hub behavior, collector permissions, or browser-side data access.

## Conversation Summary

The existing `serve k8s` and `serve eks` commands only publish live collector
sources. The live UI already consumes a generic `/api/snapshot` envelope, so the
lowest-risk design is to add a recorded source mode under `serve` that loads an
existing `snapshot.json` or scan report directory and publishes it once through
the same live store.

## Design Diff

No prior design file existed specifically for recorded live fixtures. The README
should document the new `serve snapshot <path>` entry point and the checked-in
fixture generated from a real, sanitized EKS scan.

## Decisions

- Add `teleskope serve snapshot <path>` rather than overloading `serve k8s` or
  `serve eks`, so recorded UI acceptance is explicit.
- Reuse the existing live store and report UI instead of creating a separate
  mini server or offline-only report path.
- Split a recorded snapshot into `kubernetes` and `eks` source publications so
  source status, exports, advisor, and AI analysis continue to exercise the live
  UI contract.
- Generate the long-lived fixture from a real EKS scan, then recursively sanitize
  account IDs, ARNs, endpoints, cluster/context names, user names, node
  hostnames, IPs, VPC/subnet/security-group/instance/template/AMI IDs, and UUIDs.

## Rejected Alternatives

- Use `serve-hub` to load fixture data: rejected because hub is intentionally
  multi-cluster and remote-write oriented, while this acceptance workflow is
  single-cluster UI validation.
- Keep only offline HTML reports: rejected because the user asked to validate
  live web UI interactions, including `/api/snapshot` update behavior and live
  export/analyze endpoints.
- Commit raw EKS scan output: rejected because it contains account, ARN,
  endpoint, node hostname, network, and identity data.

## Constraints

- Browser requests must not initiate AWS or Kubernetes collection.
- The checked-in fixture must not contain real account IDs, hostnames, IPs, ARNs,
  domains, customer names, secret values, usernames, or local paths.
- Existing `serve k8s`, `serve eks`, `scan`, `compare`, and hub behavior should
  remain compatible.
- Terminal-visible operational logs should state recorded loading without
  logging bearer tokens or credentials.

## Evaluation Plan

- Run focused live/CLI tests for recorded source publication.
- Run full repository `make check` and `make build`.
- Run `git diff --check`.
- Start `serve snapshot` against the sanitized fixture and verify `/` plus
  `/api/snapshot` return live UI data with recorded source status.
- Search the committed fixture for the known real account, cluster name,
  endpoint token, EKS endpoint, node hostnames, private IP/CIDR values, and AWS
  resource IDs observed in the raw scan.

## Open Questions

None.

## Links

- Design files: README.md
- Implementation files: internal/cli/serve.go, internal/live/live.go,
  internal/cli/cli_test.go, scripts/sanitize-snapshot.py,
  testdata/recorded/eks-demo-snapshot.json
