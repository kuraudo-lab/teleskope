import { expect, it } from "vitest";
import {
  topologyCondensedHierarchy,
  topologyOverviewLayout,
} from "./topology-layout.js";
it("condenses cycles and preserves stable layout across inventory order", () => {
  const nodes = [
    { id: "a", name: "a", kind: "service" },
    { id: "b", name: "b", kind: "workload" },
    { id: "c", name: "c", kind: "workload" },
    { id: "isolated", name: "isolated", kind: "workload" },
  ];
  const edges = [
    { id: "ab", source: "a", target: "b" },
    { id: "ba", source: "b", target: "a" },
    { id: "bc", source: "b", target: "c" },
  ];
  const hierarchy = topologyCondensedHierarchy(nodes, edges);
  expect(hierarchy.componentOf.get("a")).toBe(hierarchy.componentOf.get("b"));
  const first = topologyOverviewLayout(nodes, edges, 900, 120, 92),
    second = topologyOverviewLayout(
      [...nodes].reverse(),
      [...edges].reverse(),
      900,
      120,
      92,
    );
  expect(first.size).toBe(4);
  for (const [id, node] of first)
    expect([node.x, node.y]).toEqual([second.get(id).x, second.get(id).y]);
});
