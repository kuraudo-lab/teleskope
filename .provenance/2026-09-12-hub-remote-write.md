# Hub Remote Write Provenance

## Trigger

After landing the initial hub receiver, the next reviewable part is enabling existing `teleskope serve k8s` and `teleskope serve eks` collectors to publish their latest live view to a hub without blocking local collection or the local live UI.

## Scope

This record covers collector-side hub remote-write support: CLI flags and environment variables, a live-store publication hook, envelope construction from the current live response, and an asynchronous HTTP writer with retry behavior.

## Conversation Summary

The hub design keeps multi-cluster coordination out of collectors except for a narrow remote-write path. The local collector remains useful when the hub is unavailable. Each successful or partially successful live publication should enqueue the latest complete envelope for the hub, while failed or stale retained attempts should not create new hub revisions.

## Design Diff

No design file changes are included in this provenance commit. The implementation follows `docs/multi-cluster-hub.md` for `--hub-url`, environment configuration, asynchronous retry, best-effort local serve behavior, optional token-capable HTTP, and stable cluster identity.

## Decisions

- Use `TELESKOPE_HUB_URL` as the first-version hub URL environment variable and `TELESKOPE_HUB_TOKEN` for optional bearer-token writes.
- Add `--cluster-id` and `--cluster-name` to live serve commands so generic Kubernetes collectors can provide a stable fleet identity.
- Let live store expose an immutable `Response` and a non-blocking publish hook instead of importing the hub package into live.
- Keep only the newest pending envelope in the remote writer queue so a slow or unreachable hub does not accumulate stale publications.
- Retry failed writes in the background and log success/failure to the live server stream.

## Rejected Alternatives

- Blocking collection until the hub write succeeds was rejected because the design requires local serve to keep working when the hub is unavailable.
- Putting hub-specific types inside the live package was rejected to keep the live collector UI reusable without a hub dependency.
- Using a config file for hub URL/token was rejected because the design explicitly calls for flags and environment variables in the first version.

## Constraints

- Existing local `teleskope serve` behavior must remain available with no hub URL.
- Remote write must publish ready and partial live views only after a source has produced usable data.
- Hub authentication remains minimal bearer-token support; full auth is deferred.
- Tests should use `httptest` and avoid real cluster, AWS, or network dependencies.

## Evaluation Plan

- Run focused Go tests for `internal/hub`, `internal/live`, `internal/report`, and `internal/cli`.
- Verify the remote writer posts envelopes with bearer tokens.
- Verify retry publishes the newest pending envelope after an initial HTTP failure.
- Verify live publish hook fires for ready snapshots.
- Verify invalid hub URLs fail command validation before collection starts.

## Open Questions

- Whether future releases should persist failed remote-write queues across process restarts is deferred.

## Links

- Design files: `docs/multi-cluster-hub.md`
- Implementation files: `internal/hub/client.go`, `internal/live/live.go`, `internal/cli/serve.go`, related tests
