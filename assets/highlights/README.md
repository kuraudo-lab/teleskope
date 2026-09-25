# README highlight captures

Captured on 2026-09-25 from the real embedded report and production Hub UI using
only the synthetic fixtures introduced by commit `5b7f09f`. No real scan, local
credential, personal path, or model output appears in these images. The source
checkout can be newer than the latest published binary; align both before PH.

| GIF | Sequence (3 seconds per frame) | Dimensions | Size |
| --- | --- | --- | --- |
| [Topology](topology.gif) | Search `orders` and focus its relationship → workload dependencies → PVC raw fields | 1280×720 | 549 KiB |
| [Evidence](evidence.gif) | Network/Gateway table → HTTPRoute raw fields → Storage table | 1280×720 | 507 KiB |
| [Fleet](fleet.gif) | Fleet overview → `orders` search → target Storage page | 1280×720 | 693 KiB |

These are paced keyframe walkthroughs, not continuous video recordings or a
measurement of UI response time. No synthetic UI, fabricated cursor, or AI image
editing is used. GIFs loop indefinitely, with no flashing or rapid transitions.
The README puts two behind expandable sections and links here for static viewing.

## Still images

- Topology: [focused relationship](frames/topology-01.jpg), [workload details](frames/topology-02.jpg), [PVC evidence](frames/topology-03.jpg).
- Evidence: [Gateway routes](frames/evidence-01.jpg), [HTTPRoute fields](frames/evidence-02.jpg), [storage configuration](frames/evidence-03.jpg).
- Fleet: [overview](frames/fleet-01.jpg), [search results](frames/fleet-02.jpg), [target cluster](frames/fleet-03.jpg).

## Refresh captures

1. Run `go run ./scripts/demo` and serve `docs/demo` on loopback port 8090.
2. Run `go run ./scripts/demo -serve` for the seeded read-only Hub on port 8091.
3. Capture browser content only, with the current UI and synthetic fixtures.
   Original report frames are 1600×900; Hub frames are 1280×720. Keep a 16:9
   viewport, let the UI settle, and retain each scene's three reviewed JPEGs in
   `frames/`. The browser screenshot API used here returned JPEG bytes.
4. On the source report, use Search `orders`, select the StatefulSet, close its
   drawer, and pan/zoom until both relationship endpoints are fully visible.
   Capture the graph, open the workload, then inspect its PVC. For evidence,
   use Kubernetes → Network, the HTTPRoute row, then Kubernetes → Storage.
5. In Hub, capture Fleet, search `orders`, follow `target-demo`, and select
   Kubernetes → Storage. The demo disables AI execution; do not trigger it.
6. Run `python3 scripts/build-highlight-gifs.py` with FFmpeg and FFprobe on PATH.
   The script uses a shared palette per GIF, Lanczos scaling, three-second holds,
   a nine-second loop, and a 1 MB file budget. It validates dimensions and duration.
7. Inspect all frames and loop playback; review the README embeds and update
   recorded sizes if they change. Refresh after any relevant report UI change.

These README GIFs are not yet PH gallery exports. PH has its own recommended
aspect ratio and submission preview; see the dated [readiness audit](../../docs/launch/ph-readiness-2026-09-25.md).
