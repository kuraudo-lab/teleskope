import { describe, it, expect } from "vitest";
import { nodeWatch, collectionStatus } from "./node-watch";
import { watchFixture } from "./node-watch.fixture";
describe("node upgrade observation", () => {
  it("counts all namespaces, direct DS ownership, active bound pods and pending separately", () => {
    const view = nodeWatch(watchFixture());
    expect(view.nodes[0]).toMatchObject({
      nonDS: 2,
      ds: 1,
      terminating: 1,
      state: "cordon",
      target: "different",
    });
    expect(view.nodes[1]).toMatchObject({
      nonDS: 1,
      ds: 0,
      state: "notready",
      target: "match",
    });
    expect(view.pending).toBe(1);
    expect(view.terminating).toBe(1);
    expect(view.pdbs.map((p) => p.name)).toEqual(["payments"]);
    expect(view.workloads.find((w) => w.name === "scaled-down")?.desired).toBe(
      0,
    );
    expect(view.unregistered.map((i) => i.instanceId)).toEqual(["i-new"]);
  });
  it("does not assume absent pod data means zero or missing readiness means Ready", () => {
    const view = nodeWatch({
      kubernetes: { nodes: [{ name: "n", unschedulable: true }] },
    });
    expect(view.nodes[0]).toMatchObject({
      nonDS: null,
      ds: null,
      state: "unknown",
      target: "unknown",
    });
    expect(view.pending).toBeNull();
    expect(
      nodeWatch({
        kubernetes: { nodes: [{ name: "n" }], pods: [] },
        coverage: [{ resource: "Pods", status: "partial" }],
      }).nodes[0].nonDS,
    ).toBeNull();
  });
  it("does not trust old group-derived templates or symbolic target versions", () => {
    const s = watchFixture();
    delete s.eks!.infrastructure.instances[0].launchTemplateSource;
    expect(nodeWatch(s).nodes[0].target).toBe("unknown");
    expect(nodeWatch(s).nodes[0].template).toBe("未采集实例模板");
    s.eks!.infrastructure.autoScalingGroups[0].launchTemplateVersion =
      "$Latest";
    expect(nodeWatch(s).nodes[1].target).toBe("unknown");
  });
  it("never matches a conflicting provider ID using the same DNS name", () => {
    const s = watchFixture();
    s.eks!.infrastructure.instances[0].kubernetesNodeName = "worker-a";
    s.kubernetes!.nodes[0].providerId = "aws:///zone/i-other";
    expect(nodeWatch(s).nodes[0].instance).toBeUndefined();
    delete s.kubernetes!.nodes[0].providerId;
    expect(nodeWatch(s).nodes[0].instance?.instanceId).toBe("i-a");
  });
  it("retains partial coverage even when another record is complete", () => {
    expect(
      collectionStatus(
        {
          coverage: [
            { resource: "Pods", status: "complete" },
            { resource: "Pods", status: "partial", reason: "forbidden" },
          ],
        },
        "Pods",
      ).status,
    ).toBe("partial");
  });
});
