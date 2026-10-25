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
  await page.goto("http://127.0.0.1:18946/?page=eks-upgrades");
  await expect(page.locator("#eksInsightsTable")).toBeVisible();
  await page
    .getByRole("button", { name: "打开 Node watch", exact: true })
    .click();
  await expect(page).toHaveURL(/page=eks-node-watch/);
  await expect(page.locator("button.node")).toHaveCount(11);
});

for (const width of [1440, 390]) {
  test(`shared local mock demo fixture at ${width}px`, async ({ page }) => {
    await page.setViewportSize({ width, height: 1000 });
    await page.goto("http://127.0.0.1:18946/?page=eks-node-watch");
    await expect(page.locator("button.node")).toHaveCount(11);
    await expect(page.locator("#title")).toContainText("MOCK");
    await expect(page.locator(".stats")).toContainText("115 / 61");
    await expect(page.locator(".stats")).toContainText("4 / 2");
    await expect(page.locator(".event")).toHaveCount(3);
    await expect(page.locator(".pdb-note")).toContainText(
      "1 个 PDB 当前无可用驱逐预算",
    );
    await expect(page.locator('[data-node="a03"]')).toContainText("Cordoned");
    await expect(page.locator('[data-node="b02"]')).toContainText("NotReady");
    await expect(page.locator('[data-node="s03"]')).toContainText("Unknown");
    await page.locator('[data-node="a03"]').click();
    await expect(page.locator(".detail")).toContainText("ami-demo-v17");
    await expect(page.locator(".detail")).toContainText("2 个 Pod 正在删除");
    await page.getByLabel("筛选节点组").selectOption("system-workers");
    await expect(page.locator("button.node")).toHaveCount(3);
    await expect(page.locator(".event")).toHaveCount(0);
    expect(
      await page
        .locator(".watch")
        .evaluate((el) => el.scrollWidth <= el.clientWidth + 1),
    ).toBe(true);
  });
}
