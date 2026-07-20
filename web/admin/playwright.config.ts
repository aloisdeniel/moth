import { defineConfig } from "@playwright/test";

// The smoke test drives the real moth binary (bin/moth, built with `make
// build`) with the SPA embedded — exactly what a user runs.
export default defineConfig({
  testDir: "./e2e",
  globalSetup: "./e2e/global-setup.ts",
  globalTeardown: "./e2e/global-teardown.ts",
  timeout: 30_000,
  retries: 0,
  workers: 1, // one shared server instance; scenarios build on each other
  use: {
    baseURL: "http://127.0.0.1:8990",
    trace: "retain-on-failure",
  },
  reporter: process.env.CI ? "github" : "list",
});
