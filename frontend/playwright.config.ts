import { defineConfig, devices } from "@playwright/test";

const baseURL = process.env.KINGSWAY_E2E_BASE_URL ?? "http://127.0.0.1:3000";

export default defineConfig({
  expect: {
    timeout: 10_000,
  },
  forbidOnly: Boolean(process.env.CI),
  fullyParallel: false,
  reporter: [["list"]],
  retries: 0,
  testDir: "./e2e",
  timeout: 60_000,
  use: {
    baseURL,
    screenshot: "only-on-failure",
    trace: "retain-on-failure",
    video: "retain-on-failure",
  },
  projects: [
    {
      name: "chromium-system",
      use: {
        ...devices["Desktop Chrome"],
        channel: process.env.KINGSWAY_E2E_BROWSER_CHANNEL ?? "chrome",
      },
    },
  ],
});
