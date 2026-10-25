// @vitest-environment happy-dom
import { mount } from "@vue/test-utils";
import { it, expect } from "vitest";
import NodeWatch from "./NodeWatch.vue";
import { watchFixture } from "../domain/node-watch.fixture";
it("preserves selection across publications and explicitly retires removed nodes", async () => {
  const snapshot = watchFixture();
  const wrapper = mount(NodeWatch, {
    props: {
      snapshot,
      sources: { kubernetes: { mode: "recorded", state: "ready" } },
      mode: "live",
      paused: false,
      loading: false,
      error: "",
    },
  });
  expect(wrapper.text()).toContain("Recorded");
  expect(wrapper.text()).not.toContain("SELECTED");
  expect(wrapper.findAll("button.node")).toHaveLength(2);
  await wrapper.find('[data-node="worker-b"]').trigger("click");
  expect(wrapper.find(".detail").text()).toContain("worker-b");
  const updated = structuredClone(snapshot);
  updated.kubernetes!.nodes[1].ready = "True";
  await wrapper.setProps({ snapshot: updated });
  expect(
    wrapper.find('[data-node="worker-b"]').attributes("aria-pressed"),
  ).toBe("true");
  expect(wrapper.find(".detail").text()).toContain("Ready · Uncordoned");
  updated.kubernetes!.nodes = [];
  await wrapper.setProps({ snapshot: structuredClone(updated) });
  expect(wrapper.text()).toContain(
    "The selected node is no longer in this snapshot",
  );
  wrapper.unmount();
});
it("supports search, keyboard details, empty results and honest activity coverage", async () => {
  const wrapper = mount(NodeWatch, {
    props: {
      snapshot: watchFixture(),
      sources: {},
      mode: "offline",
      paused: true,
      loading: false,
      error: "",
    },
  });
  const node = wrapper.find('[data-node="worker-a"]');
  await node.trigger("focus");
  expect(wrapper.find('[role="tooltip"]').text()).toContain("ami-old");
  await node.trigger("keydown", { key: "Escape" });
  expect(wrapper.find('[role="tooltip"]').exists()).toBe(false);
  await wrapper.find("input").setValue("not-present");
  expect(wrapper.findAll("button.node")).toHaveLength(0);
  expect(wrapper.text()).toContain("No matching nodes");
  expect(wrapper.text()).toContain("Launching i-new");
  expect(wrapper.text()).toContain("Paused");
  wrapper.unmount();
});
