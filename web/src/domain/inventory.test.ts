import { expect, it } from "vitest";
import { inventoryModel } from "./inventory.js";
it("keeps zero replica template dependencies and escapes untrusted names", () => {
  const w = {
    kind: "Deployment",
    namespace: "app",
    name: "<script>alert(1)</script>",
    replicas: 0,
    serviceAccountName: "web-sa",
    containers: [
      {
        image: "repo/web:v1",
        envConfigRefs: [
          { kind: "ConfigMap", namespace: "app", name: "config" },
        ],
      },
    ],
    volumes: [{ name: "data", persistentVolumeClaim: "claim" }],
    selector: { app: "web" },
  };
  const m = inventoryModel({
    kubernetes: {
      workloads: [w],
      services: [
        {
          kind: "Service",
          namespace: "app",
          name: "web",
          selector: { app: "web" },
        },
      ],
    },
  });
  const detail = m.workloadDetailHTML(w);
  for (const text of [
    "repo/web:v1",
    "claim",
    "config",
    "web-sa",
    "service/app/web",
    "0/0",
  ])
    expect(detail).toContain(text);
  expect(detail).not.toContain("<script>");
  expect(m.tables.workloadsTable.rows).toHaveLength(1);
});
it("preserves missing namespace filter rather than widening data scope", () => {
  const m = inventoryModel(
    {
      kubernetes: {
        workloads: [
          { name: "one", namespace: "app" },
          { name: "two", namespace: "other" },
        ],
      },
    },
    {},
    "deleted",
  );
  expect(m.tables.workloadsTable.rows).toHaveLength(0);
});
