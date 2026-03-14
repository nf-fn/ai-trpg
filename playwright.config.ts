import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./e2e",
  timeout: 60_000,
  retries: 0,
  use: {
    baseURL: "http://localhost:8080",
    trace: "on-first-retry",
  },
  webServer: {
    command: "go run ./cmd/server/",
    url: "http://localhost:8080",
    reuseExistingServer: !process.env.CI,
    timeout: 10_000,
  },
});
