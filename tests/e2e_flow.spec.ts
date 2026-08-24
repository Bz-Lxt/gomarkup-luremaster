import { test, expect } from "@playwright/test";

const WEB = process.env.WEB_BASE || "http://localhost:29681";

test("login atlas catch loop", async ({ page }) => {
  await page.goto(WEB + "/");
  await page.getByLabel("邮箱").fill("hunter@lure.local");
  await page.getByLabel("密码").fill("LureHunt@2026");
  await page.getByRole("button", { name: "进入海图" }).click();
  await expect(page).toHaveURL(/atlas/);
  await expect(page.getByText(/标点|海图|结构/)).toBeVisible({ timeout: 15000 });
  await page.getByRole("link", { name: /战报/ }).click();
  await expect(page).toHaveURL(/catches/);
});
