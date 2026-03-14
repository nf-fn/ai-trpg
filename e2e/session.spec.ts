import { test, expect } from "@playwright/test";

test.describe("AI TRPG", () => {
  test("setup screen loads with rules and scenarios", async ({ page }) => {
    await page.goto("/");

    // Setup screen should be visible
    await expect(page.locator("#setup-screen")).toBeVisible();
    await expect(page.locator("#session-screen")).toBeHidden();

    // Title
    await expect(page.locator("h1")).toHaveText("AI TRPG");

    // Rule and scenario dropdowns should have options
    const ruleOptions = page.locator("#rule-select option");
    await expect(ruleOptions).not.toHaveCount(0);

    const scenarioOptions = page.locator("#scenario-select option");
    await expect(scenarioOptions).not.toHaveCount(0);
  });

  test("API returns rules list", async ({ request }) => {
    const response = await request.get("/api/rules");
    expect(response.ok()).toBeTruthy();

    const rules = await response.json();
    expect(Array.isArray(rules)).toBeTruthy();
    expect(rules.length).toBeGreaterThan(0);
    expect(rules[0]).toHaveProperty("name");
    expect(rules[0]).toHaveProperty("description");
  });

  test("API returns scenarios list", async ({ request }) => {
    const response = await request.get("/api/scenarios");
    expect(response.ok()).toBeTruthy();

    const scenarios = await response.json();
    expect(Array.isArray(scenarios)).toBeTruthy();
    expect(scenarios.length).toBeGreaterThan(0);
    expect(scenarios[0]).toHaveProperty("name");
    expect(scenarios[0]).toHaveProperty("description");
  });

  test("start session and send text message", async ({ page }) => {
    await page.goto("/");

    // Start session
    await page.click("#start-btn");

    // Session screen should appear
    await expect(page.locator("#session-screen")).toBeVisible({ timeout: 5000 });
    await expect(page.locator("#setup-screen")).toBeHidden();

    // Wait for WebSocket connection
    await expect(page.locator("#connection-status")).toHaveText("接続中", {
      timeout: 5000,
    });

    // Wait for GM's initial response
    await expect(page.locator(".message.gm")).toBeVisible({ timeout: 30000 });

    // Send a text message
    await page.fill("#text-input", "洞窟に入る");
    await page.click("#send-btn");

    // Player message should appear
    const playerMessages = page.locator(".message.player");
    await expect(playerMessages.last()).toContainText("洞窟に入る");

    // Wait for GM response
    const gmMessages = page.locator(".message.gm");
    await expect(gmMessages).toHaveCount(2, { timeout: 30000 });
  });

  test("text input via Enter key", async ({ page }) => {
    await page.goto("/");
    await page.click("#start-btn");
    await expect(page.locator("#session-screen")).toBeVisible({ timeout: 5000 });
    await expect(page.locator(".message.gm")).toBeVisible({ timeout: 30000 });

    // Type and press Enter
    await page.fill("#text-input", "周囲を見渡す");
    await page.press("#text-input", "Enter");

    // Player message should appear
    await expect(page.locator(".message.player").last()).toContainText(
      "周囲を見渡す"
    );
  });
});
