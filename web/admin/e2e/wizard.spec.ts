import { expect, test, type Page } from "@playwright/test";

// Milestone 22 acceptance flows: the adaptive creation wizard against the
// real binary. Runs after smoke.spec.ts (same server instance, same admin).
test.describe.configure({ mode: "serial" });

const admin = { email: "ops@example.com", password: "playwright-pass-1" };

async function login(page: Page) {
  await page.goto("/admin/login");
  await page.getByLabel("Email").fill(admin.email);
  await page.getByLabel("Password").fill(admin.password);
  await page.getByRole("button", { name: "Sign in" }).click();
  await expect(page.getByRole("heading", { name: "Projects" })).toBeVisible();
}

// listProjectCount reads ListProjects through the connect JSON endpoint
// (session cookie shared with page.request) — the source of truth the
// abandon test asserts against.
async function listProjectCount(page: Page): Promise<number> {
  const resp = await page.request.post("/moth.admin.v1.ProjectService/ListProjects", {
    data: {},
    headers: { "Content-Type": "application/json" },
  });
  expect(resp.ok()).toBeTruthy();
  const body = (await resp.json()) as { projects?: unknown[] };
  return body.projects?.length ?? 0;
}

test("a web-only free no-push project is created in two steps", async ({ page }) => {
  await login(page);
  await page.getByRole("button", { name: "Create project" }).click();

  // Step 1 — basics: web only.
  await page.getByLabel(/^Name/).fill("Webly");
  await expect(page.getByLabel(/^Slug/)).toHaveValue("webly");
  await page.getByLabel("Web", { exact: true }).check();
  await page.getByRole("button", { name: "Continue" }).click();

  // Step 2 — sign-in. Enabling Google inlines the same credential fields
  // as the Providers tab, minus the native ones a web-only app never needs.
  await page.getByLabel("Enable Sign in with Google").check();
  await expect(page.getByLabel("Web client ID")).toBeVisible();
  await expect(page.getByLabel("iOS client ID")).toHaveCount(0);
  await expect(page.getByLabel("Android client ID")).toHaveCount(0);

  // Defer the credentials — first-class, with the CLI hint.
  await page.getByRole("button", { name: "Configure later" }).click();
  await expect(page.getByText("moth setup google --project webly")).toBeVisible();

  // Straight to review: monetization and push default to "no", so the
  // wizard never rendered store, push or native-provider fields.
  await page.getByRole("button", { name: "Skip to review" }).click();
  await expect(page.getByText("no — free app")).toBeVisible();
  await page.getByRole("button", { name: "Create project" }).click();

  // Keys shown once, then the tailored setup tab.
  await expect(page.getByRole("heading", { name: "Webly is ready" })).toBeVisible();
  await expect(page.getByText(/pk_[A-Za-z0-9_-]+/)).toBeVisible();
  await expect(page.getByText(/sk_[A-Za-z0-9_-]+/)).toBeVisible();
  await expect(page.getByText("You won't see this key again", { exact: false })).toBeVisible();
  // Deferring Google writes the provider disabled with no credentials, so
  // every follow-up write lands: the keys screen shows NO failure entries.
  await expect(page.getByText("Some setup did not land")).toHaveCount(0);
  await page.getByRole("button", { name: "Continue to setup" }).click();

  // Web-only setup: the npm/React path only — no Flutter toggle, no
  // pubspec, and the monetization/push sections are gone (profile says no).
  await expect(page.getByText("@moth:registry=http://127.0.0.1:8990/npm")).toBeVisible();
  await expect(page.getByRole("button", { name: "Flutter" })).toHaveCount(0);
  await expect(page.getByText("pubspec.yaml")).toHaveCount(0);
  await expect(page.getByRole("heading", { name: /Monetize/ })).toHaveCount(0);
  await expect(page.getByRole("heading", { name: /Push notifications/ })).toHaveCount(0);
});

test("abandoning the wizard mid-flow creates nothing", async ({ page }) => {
  await login(page);
  const before = await listProjectCount(page);

  await page.getByRole("button", { name: "Create project" }).click();
  await page.getByLabel(/^Name/).fill("Abandoned App");
  await page.getByLabel("iOS").check();
  await page.getByRole("button", { name: "Continue" }).click();
  await expect(page.getByLabel("Open sign-up")).toBeChecked();

  // Walk away mid-flow — all wizard state is client-side.
  await page.getByRole("link", { name: "Projects" }).click();
  await expect(page.getByRole("heading", { name: "Projects" })).toBeVisible();
  await expect(page.getByText("Abandoned App")).toHaveCount(0);

  // ListProjects is unchanged: no project, no draft, nothing to clean up.
  expect(await listProjectCount(page)).toBe(before);
});
