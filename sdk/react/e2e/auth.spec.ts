import { expect, test, type Page } from "@playwright/test";

// The auth story of the example app (Stripe-enabled project, 5-second
// access-token TTL): signup opens a session, a full page reload restores
// it, sign-out lands back on the login screen, and an access token that
// expires mid-session refreshes transparently on the next authed call.
//
// Scenarios run in order (workers: 1) and build on each other: the user
// signed up in the first test signs in again in the second.

const EMAIL = "jane@example.com";
const PASSWORD = "password-123";

async function signIn(page: Page, email: string, password: string) {
  await page.locator("#moth-email").fill(email);
  await page.locator("#moth-password").fill(password);
  await page.locator('[data-moth="submit"]').click();
}

test("signup → login → reload keeps session → sign out", async ({ page }) => {
  await page.goto("/");

  // Signed out: the SDK's login screen renders (email/password form).
  await expect(page.locator("#moth-email")).toBeVisible();

  // Switch to sign-up mode (public sign-up is open on this project).
  await page.locator('[data-moth="toggle-mode"]').click();
  await signIn(page, EMAIL, PASSWORD);

  // No email verification required: sign-up opens a session immediately
  // and the provider swaps in the app.
  await expect(page.getByRole("heading", { name: `Hello ${EMAIL}` })).toBeVisible();
  await expect(page.getByText("free tier")).toBeVisible();

  // A full page reload restores the persisted session (no login screen).
  await page.reload();
  await expect(page.getByRole("heading", { name: `Hello ${EMAIL}` })).toBeVisible();

  // Sign out drops back to the login screen and clears the session…
  await page.getByRole("button", { name: "Sign out" }).click();
  await expect(page.locator("#moth-email")).toBeVisible();
  await page.reload();
  await expect(page.locator("#moth-email")).toBeVisible();

  // …and signing in again with the same credentials works.
  await signIn(page, EMAIL, PASSWORD);
  await expect(page.getByRole("heading", { name: `Hello ${EMAIL}` })).toBeVisible();
});

test("access-token expiry mid-session refreshes transparently", async ({ page }) => {
  await page.goto("/");
  await expect(page.locator("#moth-email")).toBeVisible();
  await signIn(page, EMAIL, PASSWORD);
  await expect(page.getByRole("heading", { name: `Hello ${EMAIL}` })).toBeVisible();

  // The project mints 5-second access tokens; wait past expiry, then make
  // an authenticated call ("Check access" fetches the customer info with a
  // Bearer token). The SDK must refresh under the hood: the call succeeds
  // and no login screen ever appears.
  await page.waitForTimeout(6_500);
  await page.getByRole("button", { name: "Check access" }).click();
  await expect(page.getByTestId("access-status")).toHaveText("Access: free tier");
  await expect(page.getByRole("heading", { name: `Hello ${EMAIL}` })).toBeVisible();
  await expect(page.locator("#moth-email")).not.toBeVisible();
});
