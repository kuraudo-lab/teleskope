import { test, expect } from "@playwright/test";

for (const theme of ["light", "dark"]) {
  test(`topology type colors survive drilldown, focus and hover in ${theme}`, async ({
    page,
  }) => {
    await page.addInitScript(
      (value) => localStorage.setItem("teleskope.theme", value),
      theme,
    );
    await page.goto("/");
    await expect(
      page.locator("#topology .node.kind-workload").first(),
    ).toBeVisible();
    const colors = await page
      .locator("#topology .node")
      .evaluateAll((nodes) =>
        Object.fromEntries(
          nodes.map((node) => [
            Array.from(node.classList).find((c) => c.startsWith("kind-"))!,
            getComputedStyle(node.querySelector(".node-shape")!).stroke,
          ]),
        ),
      );
    // Pod and infrastructure types first appear in deeper relationship views.
    const palette = await page.locator("html").evaluate((el) => {
      const style = getComputedStyle(el);
      return {
        pod: style.getPropertyValue("--amber").trim(),
        infrastructure: style.getPropertyValue("--cyan").trim(),
      };
    });
    const expected = await page.evaluate(
      ({ colors, palette }) => {
        const probe = document.createElement("div");
        document.body.append(probe);
        for (const [kind, color] of Object.entries(palette)) {
          probe.style.color = color;
          colors["kind-" + kind] = getComputedStyle(probe).color;
        }
        probe.remove();
        return colors;
      },
      { colors, palette },
    );
    for (const kind of ["workload", "pod", "infrastructure"]) {
      const node =
        kind === "workload"
          ? page
              .locator("#topology .node.kind-workload")
              .filter({
                has: page.locator(".label", { hasText: /^aws-node$/ }),
              })
          : page.locator(`#topology .node.kind-${kind}`).first();
      await expect(node).toBeVisible();
      const id = await node.getAttribute("data-node-id");
      await node.focus();
      // Keyboard focus itself must not replace the semantic color.
      await expect(node.locator(".node-shape")).toHaveCSS(
        "stroke",
        expected["kind-" + kind],
      );
      await page.keyboard.press("Enter");
      await expect(page.locator("#topology .node.selected")).toHaveAttribute(
        "data-node-id",
        id!,
      );
      const actual = await page
        .locator("#topology .node")
        .evaluateAll((nodes) =>
          nodes.map((node) => ({
            kind: Array.from(node.classList).find((c) =>
              c.startsWith("kind-"),
            )!,
            stroke: getComputedStyle(node.querySelector(".node-shape")!).stroke,
          })),
        );
      for (const item of actual)
        expect(item.stroke, item.kind).toBe(expected[item.kind]);
      await page.locator("#topology .node.selected").hover();
      for (const item of await page
        .locator("#topology .node")
        .evaluateAll((nodes) =>
          nodes.map((node) => ({
            kind: Array.from(node.classList).find((c) =>
              c.startsWith("kind-"),
            )!,
            stroke: getComputedStyle(node.querySelector(".node-shape")!).stroke,
          })),
        ))
        expect(item.stroke, "hover " + item.kind).toBe(expected[item.kind]);
      await page.mouse.move(0, 0);
    }
  });
}
