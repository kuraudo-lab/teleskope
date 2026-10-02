import { test, expect } from "@playwright/test";
for (const width of [1440, 1160, 390]) {
  test(`navigation dropdowns are visible and clickable at ${width}px`, async ({
    page,
  }) => {
    await page.setViewportSize({ width, height: 1000 });
    await page.goto("/");
    for (const [group, item, section] of [
      ["EKS", "Add-ons", "eks-addons"],
      ["Kubernetes", "Storage", "storage"],
    ]) {
      await page.getByRole("button", { name: group, exact: true }).click();
      const menu = page.getByRole("menu", { name: group, exact: true });
      await expect(menu).toBeVisible();
      for (const option of await menu.getByRole("menuitem").all()) {
        expect(
          await option.evaluate((el) => {
            const r = el.getBoundingClientRect();
            return (
              r.left >= 0 &&
              r.right <= innerWidth &&
              el.contains(
                document.elementFromPoint(
                  r.x + r.width / 2,
                  r.y + r.height / 2,
                ),
              )
            );
          }),
          `${group} item must be on screen and receive pointer hits`,
        ).toBe(true);
      }
      await menu.getByRole("menuitem", { name: item, exact: true }).click();
      await expect(page.locator(`[data-section="${section}"]`)).toBeVisible();
      await expect(menu).toBeHidden();
    }
  });
}
