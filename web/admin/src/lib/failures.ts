import type { StatTiles } from "../gen/moth/admin/v1/analytics_pb";

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
