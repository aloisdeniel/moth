import { defineConfig } from "@playwright/test";

// Integration test for the React SDK against the real moth binary
// (bin/moth, built with `make build`): the global setup spawns the server
// on a fresh data dir, a local Stripe API double, and the example app via
// Vite — see e2e/global-setup.ts. Run with `make sdk-react-e2e`.
export default defineConfig({
  testDir: "./e2e",
  globalSetup: "./e2e/global-setup.ts",
  globalTeardown: "./e2e/global-teardown.ts",
  timeout: 60_000, // the refresh scenario waits out a 5s token TTL; checkout polls
  retries: 0,
  workers: 1, // one shared server instance; scenarios build on each other
  use: {
    baseURL: "http://127.0.0.1:8991",
    trace: "retain-on-failure",
  },
  reporter: process.env.CI ? "github" : "list",
});
