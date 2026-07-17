import type { StatTiles, SubscriptionTiles } from "../gen/moth/admin/v1/analytics_pb";

// Shared thresholds for the "login failures elevated" ops signal (project
// overview banner, analytics tab banner + tile, projects-list badge). The
// banner needs volume to be a signal, not noise: below this many attempts
// in 7 days a bad ratio is statistically meaningless.
export const FAILURE_MIN_ATTEMPTS = 20;
export const FAILURE_RATE_THRESHOLD = 0.9;

// loginAttempts7d is the tile block's total sign-in attempts over the last
// 7 completed days.
export function loginAttempts7d(tiles: StatTiles | undefined): number {
  return tiles ? Number(tiles.logins7d + tiles.loginFailures7d) : 0;
}

// failuresElevated reports whether the failure warning should show.
export function failuresElevated(tiles: StatTiles | undefined): boolean {
  return (
    tiles !== undefined &&
    loginAttempts7d(tiles) >= FAILURE_MIN_ATTEMPTS &&
    tiles.loginSuccessRate7d < FAILURE_RATE_THRESHOLD
  );
}

// Thresholds for the subscription "elevated churn" ops signal, mirroring the
// login-failure banner. Churn needs a subscriber base to be a signal, not
// noise: below this many active subscribers last month a bad ratio is
// statistically meaningless.
export const CHURN_MIN_ACTIVE = 20;
export const CHURN_RATE_THRESHOLD = 0.25;

// monthlyChurnRate is churned / previous-month active subscribers, 0..1.
export function monthlyChurnRate(tiles: SubscriptionTiles | undefined): number {
  if (!tiles) return 0;
  const prev = Number(tiles.activeSubscribersPrevious);
  return prev > 0 ? Number(tiles.churned) / prev : 0;
}

// churnElevated reports whether the elevated-churn / failed-renewal banner
// should show: enough of a base to matter, and a churn rate over threshold.
export function churnElevated(tiles: SubscriptionTiles | undefined): boolean {
  return (
    tiles !== undefined &&
    Number(tiles.activeSubscribersPrevious) >= CHURN_MIN_ACTIVE &&
    monthlyChurnRate(tiles) >= CHURN_RATE_THRESHOLD
  );
}
