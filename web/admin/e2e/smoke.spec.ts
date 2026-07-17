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

  // The CLI path sits next to the manual walkthrough, with this project's
  // real slug and the instance URL baked in.
  await expect(page.getByText("moth login http://127.0.0.1:8990")).toBeVisible();
  await expect(page.getByText("moth setup google --project birdwatch")).toBeVisible();
  await expect(page.getByText("moth setup apple --project birdwatch")).toBeVisible();
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

test("monetization: create an entitlement and a product, and show the store URLs", async ({
  page,
}) => {
  await page.goto("/admin/login");
  await page.getByLabel("Email").fill(admin.email);
  await page.getByLabel("Password").fill(admin.password);
  await page.getByRole("button", { name: "Sign in" }).click();

  await page.getByText("Birdwatch").first().click();
  await page.getByRole("tab", { name: "Monetization" }).click();

  // The store-notification URLs interpolate the instance base URL and slug.
  await expect(
    page.getByText("http://127.0.0.1:8990/billing/apple/notifications/birdwatch"),
  ).toBeVisible();
  await expect(
    page.getByText("http://127.0.0.1:8990/billing/google/rtdn/birdwatch"),
  ).toBeVisible();

  // Create an entitlement (dialog-scoped: the product dialog also has an
  // "Identifier" field).
  await page.getByRole("button", { name: "Add entitlement" }).click();
  const entDialog = page.getByRole("dialog");
  await entDialog.getByLabel("Identifier").fill("pro");
  await entDialog.getByLabel("Display name").fill("Pro");
  await entDialog.getByRole("button", { name: "Add entitlement" }).click();
  await expect(page.getByText("Pro", { exact: true })).toBeVisible();

  // Create a product that grants it.
  await page.getByRole("button", { name: "Add product" }).click();
  const prodDialog = page.getByRole("dialog");
  await prodDialog.getByLabel("Identifier").fill("monthly");
  await prodDialog.getByLabel("Display name").fill("Monthly Pro");
  await prodDialog.getByLabel("Price").fill("9.99");
  await prodDialog.getByRole("checkbox").first().check();
  await prodDialog.getByRole("button", { name: "Add product" }).click();
  // The product now appears in both the Products card and the default-offering
  // (paywall) card, so match the first occurrence.
  await expect(page.getByText("Monthly Pro").first()).toBeVisible();

  // Both survive a full reload — they really landed in the project catalog.
  await page.reload();
  await expect(page.getByText("Pro", { exact: true })).toBeVisible();
  await expect(page.getByText("Monthly Pro").first()).toBeVisible();
});

test("monetization: store-connection panel renders and a push dry-run opens", async ({
  page,
}) => {
  await page.goto("/admin/login");
  await page.getByLabel("Email").fill(admin.email);
  await page.getByLabel("Password").fill(admin.password);
  await page.getByRole("button", { name: "Sign in" }).click();

  await page.getByText("Birdwatch").first().click();
  await page.getByRole("tab", { name: "Monetization" }).click();

  // The store-connection panel renders one card per store with its
  // credential/notification state.
  await expect(page.getByRole("heading", { name: "Store connection" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Apple — App Store Connect" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Google — Google Play" })).toBeVisible();

  // "Push to Apple" runs a dry-run and opens the plan dialog. No store call
  // happens without credentials — the server returns an empty/guided plan.
  await page.getByRole("button", { name: "Push to Apple" }).click();
  await expect(page.getByText("Push catalog to Apple")).toBeVisible();
  await page.getByRole("dialog").getByRole("button", { name: "Cancel" }).click();
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

test("paywall editor saves a headline and the preview reflects it", async ({ page }) => {
  await page.goto("/admin/login");
  await page.getByLabel("Email").fill(admin.email);
  await page.getByLabel("Password").fill(admin.password);
  await page.getByRole("button", { name: "Sign in" }).click();

  await page.getByText("Birdwatch").first().click();
  await page.getByRole("tab", { name: "Design" }).click();

  // Switch from the login theme editor to the paywall editor (same Design
  // tab); the section toggle keeps a single live preview on screen.
  await page.getByRole("button", { name: "Paywall" }).click();

  const headline = page.getByLabel("Headline");
  await headline.fill("Go Pro today");

  // The live preview reflects the unsaved headline immediately.
  await expect(page.locator(".mothpw__headline")).toHaveText("Go Pro today");

  await page.getByRole("button", { name: "Save paywall" }).click();
  await expect(page.getByText("Saved.")).toBeVisible();

  // Survives a full reload — the config really landed on the project.
  await page.reload();
  await page.getByRole("button", { name: "Paywall" }).click();
  await expect(page.getByLabel("Headline")).toHaveValue("Go Pro today");
  await expect(page.locator(".mothpw__headline")).toHaveText("Go Pro today");
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

test("personal access tokens: create shows the plaintext once, list, revoke", async ({
  page,
}) => {
  await page.goto("/admin/login");
  await page.getByLabel("Email").fill(admin.email);
  await page.getByLabel("Password").fill(admin.password);
  await page.getByRole("button", { name: "Sign in" }).click();
  await expect(page.getByRole("heading", { name: "Projects" })).toBeVisible();

  await page.getByRole("link", { name: "Instance settings" }).click();
  await expect(page.getByRole("heading", { name: "Instance settings" })).toBeVisible();

  // The CLI helper interpolates the real instance base URL.
  await expect(page.getByText("moth login http://127.0.0.1:8990")).toBeVisible();

  // Create a token; the dialog shows the moth_pat_ plaintext exactly once.
  // (Dialog-scoped: the SMTP card's "Username (optional)" also matches "Name".)
  await page.getByRole("button", { name: "Create token" }).click();
  await page.getByRole("dialog").getByLabel("Name").fill("laptop");
  await page.getByRole("dialog").getByRole("button", { name: "Create token" }).click();
  await expect(page.getByText(/moth_pat_[A-Za-z0-9_-]+/)).toBeVisible();
  await expect(
    page.getByText("You won't see this token again", { exact: false }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Done" }).click();

  // Listed as active; the plaintext is gone for good.
  const row = page.getByRole("row", { name: /laptop/ });
  await expect(row).toBeVisible();
  await expect(row.getByText("Active")).toBeVisible();
  await expect(page.getByText(/moth_pat_[A-Za-z0-9_-]+/)).toBeHidden();

  // Revoke flips the state and removes the revoke action.
  await row.getByRole("button", { name: "Revoke" }).click();
  await page.getByRole("dialog").getByRole("button", { name: "Revoke token" }).click();
  await expect(row.getByText("Revoked")).toBeVisible();
  await expect(row.getByRole("button", { name: "Revoke" })).toBeHidden();
});

test("analytics tab renders the empty state for a zero-traffic project", async ({ page }) => {
  await page.goto("/admin/login");
  await page.getByLabel("Email").fill(admin.email);
  await page.getByLabel("Password").fill(admin.password);
  await page.getByRole("button", { name: "Sign in" }).click();

  await page.getByText("Birdwatch").first().click();
  await page.getByRole("tab", { name: "Analytics" }).click();

  // Zero traffic → stat tiles of zeros plus a friendly empty state, never
  // broken charts. (Seeded-data correctness is covered by the Go tests.)
  await expect(page.getByText("Total users")).toBeVisible();
  await expect(page.getByText("Login success rate")).toBeVisible();
  await expect(page.getByText("No activity yet")).toBeVisible();

  // The subscriptions revenue dashboard (milestone 14) renders its own
  // friendly empty state for a project that has sold nothing.
  await expect(page.getByRole("heading", { name: "Subscriptions" })).toBeVisible();
  await expect(page.getByText("No subscriptions yet")).toBeVisible();
});

test("audit log viewer renders with filters and log card", async ({ page }) => {
  await page.goto("/admin/login");
  await page.getByLabel("Email").fill(admin.email);
  await page.getByLabel("Password").fill(admin.password);
  await page.getByRole("button", { name: "Sign in" }).click();
  await expect(page.getByRole("heading", { name: "Projects" })).toBeVisible();

  await page.getByRole("link", { name: "Audit" }).click();
  await expect(page.getByRole("heading", { name: "Audit", exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Apply filters" })).toBeVisible();

  // The log card renders whether the earlier scenarios produced entries or
  // the filters match nothing (empty state) — either way it is present.
  await expect(page.getByRole("heading", { name: "Audit log" })).toBeVisible();
});

test("abuse controls: an allowed email domain saves and persists", async ({ page }) => {
  await page.goto("/admin/login");
  await page.getByLabel("Email").fill(admin.email);
  await page.getByLabel("Password").fill(admin.password);
  await page.getByRole("button", { name: "Sign in" }).click();

  await page.getByText("Birdwatch").first().click();
  await page.getByRole("tab", { name: "Settings" }).click();

  const domain = "e2e-allowed.example";
  const allow = page.getByLabel("Allowed email domains");
  await allow.fill(domain);
  await allow.press("Enter");
  await page.getByRole("button", { name: "Save settings" }).click();
  await expect(page.getByText("Saved.")).toBeVisible();

  // Survives a full reload — the URL keeps the Settings tab active.
  await page.reload();
  await expect(page.getByText(domain)).toBeVisible();
});
