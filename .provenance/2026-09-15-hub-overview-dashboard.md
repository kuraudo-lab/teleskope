# Hub overview dashboard

Date: 2026-09-15
Issue: https://github.com/kuraudo-lab/teleskope/issues/21

## Intent

Turn the existing hub cluster list into a fleet-level operational dashboard that
can be accepted independently before starting fleet-wide search.

## Scope

- Extend hub fleet summaries with Kubernetes version and compact EKS risk counts.
- Render Kubernetes version distribution in the hub overview.
- Render EKS add-on and managed nodegroup risk counts in the overview and cluster table.
- Render source health counts across reported cluster sources.
- Add browser filters for region, provider, and state.

## Boundaries

- Keep hub storage memory-only.
- Do not implement fleet resource search from issue 23 in this block.
- Do not add provider credentials or hub-side cluster API calls.
- Keep EKS risk counts evidence-based and compact: count add-ons or nodegroups
  with non-active status or reported health issues.

## Verification

- Add focused tests for summary aggregation fields and embedded hub UI hooks.
- Run `go test ./internal/hub`.
- Run repository checks with temporary Go caches:
  `GOCACHE=/private/tmp/teleskope-go-cache GOMODCACHE=/private/tmp/teleskope-go-mod-cache make check`
- Run the same-cache `make build`.
- Run `git diff --check`.
