import { test, expect } from "@playwright/test";
import { watchFixture } from "../src/domain/node-watch.fixture";

for (const width of [1440, 390]) {
  test(`node watch production layout and controls at ${width}px`, async ({
    page,
  }) => {
    await page.setViewportSize({ width, height: 1000 });
    let snapshot = watchFixture(),
      revision = 10;
    await page.route("**/api/snapshot", async (route) => {
      const response = await route.fetch();
      const data = await response.json();
      await route.fulfill({
        json: { ...data, snapshot, revision },
        headers: { "Content-Type": "application/json" },
      });
    });
    await page.goto("/?page=eks-node-watch");
    await expect(page.locator("button.node")).toHaveCount(2);
    await expect(page.getByLabel("Namespace", { exact: true })).toBeDisabled();
    await expect(page.locator(".activity")).toContainText("Launching i-new");
    await expect(page.locator(".watch")).not.toContainText("SELECTED");
    await expect(page.locator(".node path").first()).toHaveAttribute("d", /Q/);
    for (const theme of ["dark", "light"]) {
      if ((await page.locator("html").getAttribute("data-theme")) !== theme)
        await page.locator("#themeToggle").click();
      expect(
        await page
          .locator(".watch")
          .evaluate((el) => el.scrollWidth <= el.clientWidth + 1),
      ).toBe(true);
    }
    const node = page.locator('[data-node="worker-a"]');
    await node.focus();
    await expect(page.getByRole("tooltip")).toContainText("ami-old");
    await node.press("Escape");
    await expect(page.getByRole("tooltip")).toHaveCount(0);
    await page.locator('[data-node="worker-b"]').click();
    await expect(page.locator(".detail")).toContainText("worker-b");
    snapshot = structuredClone(snapshot);
    snapshot.kubernetes!.nodes[1].ready = "True";
    revision++;
    await page.evaluate(() =>
      document.dispatchEvent(new Event("teleskope:refresh")),
    );
    await expect(page.locator(".detail .status-line")).toContainText(
      "Ready · 未 cordon",
    );
    snapshot.kubernetes!.nodes = [];
    revision++;
    await page.evaluate(() =>
      document.dispatchEvent(new Event("teleskope:refresh")),
    );
    await expect(page.locator(".inspector")).toContainText(
      "所选节点已离开当前快照",
    );
  });
}

test("upgrade insights remain available and link to node watch", async ({
  page,
}) => {
  await page.goto("/?page=eks-upgrades");
  await expect(page.locator("#eksInsightsTable")).toBeVisible();
  await page
    .getByRole("button", { name: "打开 Node watch", exact: true })
    .click();
  await expect(page).toHaveURL(/page=eks-node-watch/);
  await expect(page.locator("button.node")).toHaveCount(2);
});
