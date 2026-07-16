import { readFileSync } from "node:fs";

import { expect, test } from "@playwright/test";

import { STATE_FILE } from "./global-setup";

// The milestone's browser-only acceptance flow: first-run admin creation →
// log in → create a project → see its keys. Scenarios run in order against
// one server instance.
test.describe.configure({ mode: "serial" });

const admin = { email: "ops@example.com", password: "playwright-pass-1" };

function setupToken(): string {
  const state = JSON.parse(readFileSync(STATE_FILE, "utf-8")) as { setupToken: string };
  return state.setupToken;
}

test("first-run setup creates the initial admin", async ({ page }) => {
  // The console prints /admin?setup=TOKEN; the SPA turns it into the
  // setup screen with the token prefilled.
  await page.goto(`/admin?setup=${setupToken()}`);
  await expect(page.getByRole("heading", { name: "Create the first admin" })).toBeVisible();

  await page.getByLabel("Email").fill(admin.email);
  await page.getByLabel("Password", { exact: false }).fill(admin.password);
  await page.getByRole("button", { name: "Create admin account" }).click();

  // Lands signed-in on the (empty) projects screen.
  await expect(page.getByRole("heading", { name: "Projects" })).toBeVisible();
  await expect(page.getByText("No projects yet")).toBeVisible();
});

test("log out and back in", async ({ page }) => {
  await page.goto("/admin");
  // No cookie in this fresh browser context → login screen.
  await expect(page.getByRole("heading", { name: "Sign in to moth" })).toBeVisible();

  await page.getByLabel("Email").fill(admin.email);
  await page.getByLabel("Password").fill(admin.password);
  await page.getByRole("button", { name: "Sign in" }).click();
  await expect(page.getByRole("heading", { name: "Projects" })).toBeVisible();

  await page.getByRole("button", { name: "Log out" }).click();
  await expect(page.getByRole("heading", { name: "Sign in to moth" })).toBeVisible();
});

test("create a project and see its keys", async ({ page }) => {
  await page.goto("/admin/login");
  await page.getByLabel("Email").fill(admin.email);
  await page.getByLabel("Password").fill(admin.password);
  await page.getByRole("button", { name: "Sign in" }).click();

  await page.getByRole("button", { name: "Create project" }).click();
  await page.getByLabel("Name").fill("Birdwatch");
  await expect(page.getByText("Slug: birdwatch")).toBeVisible();
  await page.getByRole("dialog").getByRole("button", { name: "Create project" }).click();

  // The creation dialog shows both keys; the secret exactly once.
  await expect(page.getByText("Birdwatch is ready")).toBeVisible();
  await expect(page.getByText(/pk_[A-Za-z0-9_-]+/)).toBeVisible();
  await expect(page.getByText(/sk_[A-Za-z0-9_-]+/)).toBeVisible();
  await expect(page.getByText("You won't see this key again", { exact: false })).toBeVisible();

  await page.getByRole("button", { name: "Open project" }).click();

  // Project overview: publishable key visible, secret masked, JWKS URL.
  await expect(page.getByRole("heading", { name: "Birdwatch" })).toBeVisible();
  await expect(page.getByText(/pk_[A-Za-z0-9_-]+/)).toBeVisible();
  await expect(page.getByText("sk_••••", { exact: false })).toBeVisible();
  await expect(page.getByText(/\/p\/birdwatch\/\.well-known\/jwks\.json/)).toBeVisible();
});

test("setup instructions render real values", async ({ page }) => {
  await page.goto("/admin/login");
  await page.getByLabel("Email").fill(admin.email);
  await page.getByLabel("Password").fill(admin.password);
  await page.getByRole("button", { name: "Sign in" }).click();

  await page.getByText("Birdwatch").first().click();
  await page.getByRole("tab", { name: "Setup" }).click();

  await expect(page.getByText("hosted: http://127.0.0.1:8990/pub")).toBeVisible();
  await expect(page.getByText(/publishableKey: 'pk_[A-Za-z0-9_-]+'/)).toBeVisible();
  await expect(
    page.getByText("http://127.0.0.1:8990/p/birdwatch/.well-known/jwks.json").first(),
  ).toBeVisible();
});

test("enable Google sign-in and see it persist", async ({ page }) => {
  await page.goto("/admin/login");
  await page.getByLabel("Email").fill(admin.email);
  await page.getByLabel("Password").fill(admin.password);
  await page.getByRole("button", { name: "Sign in" }).click();

  await page.getByText("Birdwatch").first().click();
  await page.getByRole("tab", { name: "Providers" }).click();

  // The guide interpolates the instance base URL into the redirect URIs.
  await expect(
    page.getByText("http://127.0.0.1:8990/oauth/google/callback").first(),
  ).toBeVisible();
  await expect(
    page.getByText("http://127.0.0.1:8990/oauth/apple/callback").first(),
  ).toBeVisible();

  await page.getByLabel("Enable Sign in with Google").check();
  await page
    .getByLabel("Web client ID")
    .fill("1234567890-birdwatch.apps.googleusercontent.com");
  await page.getByRole("button", { name: "Save providers" }).click();
  await expect(page.getByText("Saved.")).toBeVisible();

  // Survives a full reload — the config really landed in project settings.
  await page.reload();
  await expect(page.getByLabel("Enable Sign in with Google")).toBeChecked();
  await expect(page.getByLabel("Web client ID")).toHaveValue(
    "1234567890-birdwatch.apps.googleusercontent.com",
  );
});
