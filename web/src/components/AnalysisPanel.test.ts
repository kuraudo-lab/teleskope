// @vitest-environment happy-dom
import { mount } from "@vue/test-utils";
import { it, expect, vi } from "vitest";
import AnalysisPanel from "./AnalysisPanel.vue";
it("renders sections, recommendation, evidence and metadata and restores a saved result", async () => {
  const request = {
    revision: 1,
    scope: {
      pageId: "nodes",
      namespace: "app",
      resourceType: "nodes",
      selectedRefs: [],
    },
  };
  const result = {
    summary: "Review nodes",
    model: "test-model",
    sections: [
      {
        items: [
          {
            summary: "Fact",
            basis: "observed",
            detail: "Observed detail",
            recommendation: "Check capacity",
            evidence: ["source evidence"],
          },
          { summary: "Inference", basis: "inference" },
        ],
      },
    ],
    limitations: ["No traffic evidence"],
  };
  const saved = { result: null, key: "" };
  const analyze = vi.fn().mockResolvedValue(result);
  const wrapper = mount(AnalysisPanel, {
    props: {
      page: "nodes",
      request,
      available: true,
      busy: false,
      analyze,
      saved,
    },
  });
  await wrapper.find("button").trigger("click");
  await new Promise((r) => setTimeout(r, 0));
  expect(analyze).toHaveBeenCalledWith(request);
  for (const text of [
    "Review nodes",
    "test-model",
    "Check capacity",
    "source evidence",
    "No traffic evidence",
  ])
    expect(wrapper.text()).toContain(text);
  expect(wrapper.find("button").attributes("disabled")).toBeDefined();
  wrapper.unmount();
  const again = mount(AnalysisPanel, {
    props: {
      page: "nodes",
      request,
      available: true,
      busy: false,
      analyze,
      saved,
    },
  });
  expect(again.text()).toContain("Review nodes");
  again.unmount();
});
