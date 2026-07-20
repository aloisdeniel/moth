import { seedAnalytics } from "./services/analytics";
import type { AnalyticsSlice } from "./services/analytics";
import { seedContent } from "./services/content";
import type { ContentSlice } from "./services/content";
import { seedInstance } from "./services/instance";
import type { InstanceSlice } from "./services/instance";
import { seedMonetization } from "./services/monetization";
import type { MonetizationSlice } from "./services/monetization";
import { seedProjects } from "./services/projects";
import type { ProjectsSlice } from "./services/projects";
import { seedUsers } from "./services/users";
import type { UsersSlice } from "./services/users";

// Demo state: one JSON document in localStorage, composed of one slice per
// service area. First load (or a seed-version bump after a redeploy) plants
// the example world; afterwards the visitor's own edits persist locally.

// Bump when the seed shape or content changes so returning visitors reseed.
export const SEED_VERSION = 1;

const STORAGE_KEY = "moth-demo-state";

export type DemoState = { demoVersion: number } & InstanceSlice &
  ProjectsSlice &
  UsersSlice &
  ContentSlice &
  MonetizationSlice &
  AnalyticsSlice;

export function seed(): DemoState {
  return {
    demoVersion: SEED_VERSION,
    ...seedInstance(),
    ...seedProjects(),
    ...seedUsers(),
    ...seedContent(),
    ...seedMonetization(),
    ...seedAnalytics(),
  };
}

let cached: DemoState | null = null;

export function loadState(): DemoState {
  if (cached) {
    return cached;
  }
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (raw) {
      const parsed = JSON.parse(raw) as DemoState;
      if (parsed.demoVersion === SEED_VERSION) {
        cached = parsed;
        return cached;
      }
    }
  } catch {
    // Corrupted or inaccessible storage: fall through to a fresh seed.
  }
  cached = seed();
  saveState(cached);
  return cached;
}

export function saveState(state: DemoState): void {
  cached = state;
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
  } catch {
    // Storage full or blocked: the demo still works, it just won't persist.
  }
}

export function resetState(): void {
  cached = null;
  try {
    localStorage.removeItem(STORAGE_KEY);
  } catch {
    // Ignore; reload will reseed anyway.
  }
}
