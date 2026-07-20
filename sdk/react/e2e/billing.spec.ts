import { expect, test, type Page } from "@playwright/test";
import { APP_NOSTRIPE_URL } from "./global-setup";

// The billing loop against the Stripe test double: a free user hits the
// gated pro page, sees the themed paywall, completes a (double-hosted)
// checkout whose signed checkout.session.completed webhook lands in moth,
// returns, and the page unlocks without a manual reload. manageBilling
// reaches the (double-hosted) Billing Portal. A second project with no
// Stripe configuration runs the same auth story with gates never blocking.
//
// Scenarios run in order (workers: 1): the buyer signed up in the first
// test manages billing in the second.

const BUYER_EMAIL = "buyer@example.com";
const PASSWORD = "password-123";

async function signIn(page: Page, email: string, password: string) {
  await page.locator("#moth-email").fill(email);
  await page.locator("#moth-password").fill(password);
  await page.locator('[data-moth="submit"]').click();
}

test("free user: paywall → test checkout → page unlocks without reload", async ({ page }) => {
  await page.goto("/");
  await page.locator('[data-moth="toggle-mode"]').click();
  await signIn(page, BUYER_EMAIL, PASSWORD);
  await expect(page.getByRole("heading", { name: `Hello ${BUYER_EMAIL}` })).toBeVisible();
  await expect(page.getByText("free tier")).toBeVisible();

  // The gated page: no "pro" entitlement, and the offering sells one — the
  // paywall renders with the tier card and price metadata.
  await page.getByRole("button", { name: "Pro area" }).click();
  await expect(page.locator('[data-moth="headline"]')).toBeVisible();
  const tier = page.locator('[data-moth="tier-monthly"]');
  await expect(tier).toBeVisible();
  await expect(tier).toContainText("Monthly");
  await expect(tier).toContainText("$9.99 / month");
  await expect(page.getByRole("heading", { name: "Pro area" })).not.toBeVisible();

  // Purchase: CreateCheckoutSession → redirect to the double's hosted
  // checkout, which completes immediately (fires the HMAC-signed
  // checkout.session.completed webhook at moth) and bounces back to the
  // success URL with the SDK's moth_checkout=success marker.
  await page.locator('[data-moth="purchase"]').click();

  // Back in the app the SDK strips the marker and re-fetches the customer
  // info (short poll for webhook latency): the entitlement flips without
  // any manual reload…
  await expect(page.getByRole("heading", { name: `Hello ${BUYER_EMAIL}` })).toBeVisible();
  await expect(page.getByText("pro", { exact: true })).toBeVisible({ timeout: 15_000 });

  // …and the gated page is unlocked (no paywall).
  await page.getByRole("button", { name: "Pro area" }).click();
  await expect(page.getByRole("heading", { name: "Pro area" })).toBeVisible();
  await expect(page.locator('[data-moth="headline"]')).not.toBeVisible();
});

test("manageBilling reaches the Billing Portal", async ({ page }) => {
  await page.goto("/");
  await signIn(page, BUYER_EMAIL, PASSWORD);
  await expect(page.getByRole("heading", { name: `Hello ${BUYER_EMAIL}` })).toBeVisible();

  // CreateBillingPortalSession resolves the buyer's Stripe customer and
  // redirects to the portal URL — the double's stub landing page.
  await page.getByRole("button", { name: "Manage billing" }).click();
  await page.waitForURL(/127\.0\.0\.1:8993\/portal/);
  await expect(page.getByText("Stripe billing portal (test double)")).toBeVisible();

  // The portal's return link lands back in the signed-in app.
  await page.getByRole("link", { name: "Return to app" }).click();
  await expect(page.getByRole("heading", { name: `Hello ${BUYER_EMAIL}` })).toBeVisible();
});

test("project with no Stripe config: gates never block, auth story intact", async ({ page }) => {
  const email = "nostripe@example.com";
  await page.goto(APP_NOSTRIPE_URL);
  await expect(page.locator("#moth-email")).toBeVisible();
  await page.locator('[data-moth="toggle-mode"]').click();
  await signIn(page, email, PASSWORD);
  await expect(page.getByRole("heading", { name: `Hello ${email}` })).toBeVisible();

  // The gated page falls through: nothing in the offering sells "pro", so
  // the gate opens instead of dead-ending a project that sells nothing.
  await page.getByRole("button", { name: "Pro area" }).click();
  await expect(page.getByRole("heading", { name: "Pro area" })).toBeVisible();
  await expect(page.locator('[data-moth="headline"]')).not.toBeVisible();

  // The rest of the auth story is unaffected: reload restores, sign out
  // returns to the login screen.
  await page.reload();
  await expect(page.getByRole("heading", { name: `Hello ${email}` })).toBeVisible();
  await page.getByRole("button", { name: "Sign out" }).click();
  await expect(page.locator("#moth-email")).toBeVisible();
});
