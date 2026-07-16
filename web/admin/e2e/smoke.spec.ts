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

test("theme editor saves a primary color and persists it", async ({ page }) => {
  await page.goto("/admin/login");
  await page.getByLabel("Email").fill(admin.email);
  await page.getByLabel("Password").fill(admin.password);
  await page.getByRole("button", { name: "Sign in" }).click();

  await page.getByText("Birdwatch").first().click();
  await page.getByRole("tab", { name: "Design" }).click();

  // Fresh project → built-in default theme in the editor.
  const primary = page.getByLabel("Primary", { exact: true });
  await expect(primary).toHaveValue("#6750A4");

  await primary.fill("#0B57D0");
  await page.getByRole("button", { name: "Save theme" }).click();
  await expect(page.getByText("Saved.")).toBeVisible();

  // Survives a full reload — the theme really landed on the project.
  await page.reload();
  await expect(page.getByLabel("Primary", { exact: true })).toHaveValue("#0B57D0");

  // The save shows up as the current revision.
  await expect(page.getByText("Current", { exact: true })).toBeVisible();
});

// Preview honesty (plan/06): the three reference themes shared with the
// Flutter golden suite (sdk/flutter/test/golden), as UpdateTheme payloads.
const referenceThemes: Record<string, object> = {
  default: {
    colors: {
      primary: "#6750A4",
      onPrimary: "#FFFFFF",
      background: "#FFFBFE",
      onBackground: "#1C1B1F",
      surface: "#FFFBFE",
      onSurface: "#1C1B1F",
      error: "#B3261E",
      onError: "#FFFFFF",
    },
    typography: { fontFamily: "Inter", scale: 1.0 },
    spacing: { unit: 8 },
    shape: { cornerRadius: 12 },
    legal: {},
  },
  ocean: {
    colors: {
      primary: "#0B6E99",
      onPrimary: "#FFFFFF",
      background: "#F7FAFC",
      onBackground: "#102A43",
      surface: "#FFFFFF",
      onSurface: "#102A43",
      error: "#B00020",
      onError: "#FFFFFF",
    },
    typography: { fontFamily: "Inter", scale: 0.9 },
    spacing: { unit: 6 },
    shape: { cornerRadius: 2 },
    legal: { termsUrl: "https://example.com/terms", privacyUrl: "https://example.com/privacy" },
  },
  sunset: {
    colors: {
      primary: "#C8481F",
      onPrimary: "#FFFFFF",
      background: "#FFF8F2",
      onBackground: "#33201A",
      surface: "#FFFFFF",
      onSurface: "#33201A",
      error: "#8C1D18",
      onError: "#FFFFFF",
    },
    darkColors: {
      primary: "#FFB59D",
      onPrimary: "#3B0900",
      background: "#1F1410",
      onBackground: "#F5E4DD",
      surface: "#2A1B15",
      onSurface: "#F5E4DD",
      error: "#FFB4AB",
      onError: "#3B0900",
    },
    typography: { fontFamily: "Inter", scale: 1.1 },
    spacing: { unit: 10 },
    shape: { cornerRadius: 28 },
    legal: { privacyUrl: "https://example.com/privacy" },
  },
};

test("preview honesty: capture the reference-theme previews for golden review", async ({
  page,
}) => {
  // Renders the live preview for each reference theme (light and dark) and
  // saves screenshots into e2e/preview/ (untracked), for a side-by-side
  // review against the committed Flutter goldens in
  // sdk/flutter/test/golden/goldens — see `make preview-goldens`.
  await page.goto("/admin/login");
  await page.getByLabel("Email").fill(admin.email);
  await page.getByLabel("Password").fill(admin.password);
  await page.getByRole("button", { name: "Sign in" }).click();
  await expect(page.getByRole("heading", { name: "Projects" })).toBeVisible();

  // The session cookie is shared with page.request, so the connect JSON
  // endpoints can install each reference theme directly.
  const list = await page.request.post("/moth.admin.v1.ProjectService/ListProjects", {
    data: {},
    headers: { "Content-Type": "application/json" },
  });
  expect(list.ok()).toBeTruthy();
  const projects = ((await list.json()) as { projects: { id: string; name: string }[] })
    .projects;
  const projectId = projects.find((p) => p.name === "Birdwatch")!.id;

  for (const [name, theme] of Object.entries(referenceThemes)) {
    const update = await page.request.post("/moth.admin.v1.ThemeService/UpdateTheme", {
      data: { projectId, theme },
      headers: { "Content-Type": "application/json" },
    });
    expect(update.ok()).toBeTruthy();

    await page.goto("/admin");
    await page.getByText("Birdwatch").first().click();
    await page.getByRole("tab", { name: "Design" }).click();
    const preview = page.locator(".phone");
    await expect(preview).toBeVisible();

    await page.getByRole("button", { name: "Light" }).click();
    await preview.screenshot({ path: `e2e/preview/login_${name}_light.png` });
    await page.getByRole("button", { name: "Dark" }).click();
    await preview.screenshot({ path: `e2e/preview/login_${name}_dark.png` });
  }
});

test("contrast validation warns and blocks an illegible palette", async ({ page }) => {
  await page.goto("/admin/login");
  await page.getByLabel("Email").fill(admin.email);
  await page.getByLabel("Password").fill(admin.password);
  await page.getByRole("button", { name: "Sign in" }).click();

  await page.getByText("Birdwatch").first().click();
  await page.getByRole("tab", { name: "Design" }).click();

  // Primary == on-primary (white on white, from the previous scenario) is
  // unreadable: the pair badge flips to a failure and save is blocked.
  await page.getByLabel("Primary", { exact: true }).fill("#FFFFFF");
  await expect(page.getByText("fails AA").first()).toBeVisible();
  await expect(
    page.getByText("Fix the failing contrast pairs before saving", { exact: false }),
  ).toBeVisible();
  await expect(page.getByRole("button", { name: "Save theme" })).toBeDisabled();
});
