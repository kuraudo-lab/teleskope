import { describe, it, expect } from "vitest";
import { applyTopologyUpdate } from "./topology-update";
import type { Graph, TopologyUpdate } from "./types";
const graph: Graph = {
  clusterId: "one",
  revision: 1,
  nodes: [{ id: "a" }],
  edges: [],
};
const delta: TopologyUpdate = {
  schemaVersion: "teleskope.io/topology-update/v1alpha1",
  kind: "delta",
  clusterId: "one",
  baseRevision: 1,
  revision: 2,
  upsertNodes: [{ id: "b" }],
  upsertEdges: [{ id: "edge", source: "a", target: "b" }],
};
describe("topology revision protocol", () => {
  it("merges without mutating input and tolerates replay", () => {
    const next = applyTopologyUpdate(graph, delta)!;
    expect(next.nodes).toHaveLength(2);
    expect(graph.nodes).toHaveLength(1);
    expect(applyTopologyUpdate(next, delta)).toBe(next);
  });
  it("rejects gaps, cluster changes and dangling edges", () => {
    expect(
      applyTopologyUpdate(graph, { ...delta, baseRevision: 0 }),
    ).toBeNull();
    expect(
      applyTopologyUpdate(graph, { ...delta, clusterId: "two" }),
    ).toBeNull();
    expect(
      applyTopologyUpdate(graph, { ...delta, upsertNodes: [] }),
    ).toBeNull();
  });
});
