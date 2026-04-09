import { test, expect } from "@playwright/test";
import { captureJourneyCheckpoint } from "@tests/e2e/utils/journey";
import { logPageState } from "@tests/e2e/utils/pageStateLogger";

test.describe("Journey: auth landing", () => {
  test.beforeEach(async ({ page }) => {
    await page.context().clearCookies();
  });

  test("Fresh auth landing is clean @journey", async ({ page }) => {
    await page.goto("/", { waitUntil: "domcontentloaded" });
    await expect
      .poll(() => page.url(), { timeout: 60000 })
      .toMatch(/\/auth\/(login|signup)(\?.*)?$/);
    await expect
      .poll(async () => (await page.locator("body").innerText()).trim(), {
        timeout: 60000,
      })
      .toMatch(
        /Create account|Create Account|Already have an account|New to Onyx\?|Sign In/i
      );
    await page.waitForTimeout(1000);

    const loggedOutModal = page.getByText("You Have Been Logged Out", {
      exact: true,
    });
    console.log(
      `[journey-auth-landing] ${JSON.stringify({
        url: page.url(),
        loggedOutModalVisible: (await loggedOutModal.count()) > 0,
      })}`
    );

    await logPageState(page, "journey auth landing");
    await captureJourneyCheckpoint(page, "auth-landing");
    await expect(loggedOutModal).toHaveCount(0);

    await expect(page.locator("body")).toContainText(
      /New to Onyx\?|Create an Account|Sign In/
    );
  });
});
