// Demo implementation of moth.admin.v1.AnalyticsService.
//
// Aurora Journal is seeded with ~120 days of daily rollups (gentle growth,
// weekend dips, noise) that stay order-of-magnitude consistent with the rest
// of the demo world: 22 end users (signup days come straight from PEOPLE),
// a few hundred cumulative logins, 5 active subscribers + 1 trial. Skylark
// is three days old: three tiny days of traffic, three events, and no
// subscription data at all, so the sparse/empty chart states render too.

import { handle } from "../transport";
import {
  daysAgo,
  invalidArgument,
  minutesAgo,
  now,
  randomId,
  ts,
  type Millis,
} from "../util";
import {
  CHURNED_ID,
  LIFETIME_ID,
  PEOPLE,
  PRODUCT_ANNUAL_ID,
  PRODUCT_LIFETIME_ID,
  PRODUCT_MONTHLY_ID,
  PROJECT_MAIN,
  PROJECT_SIDE,
  SUBSCRIBER_IDS,
  TRIAL_ID,
  demoId,
} from "../ids";
import { AnalyticsService, Granularity } from "../../gen/moth/admin/v1/analytics_pb";

// ---------- State slice ----------

// One pre-aggregated day of the daily_stats rollup, per project. Provider
// and platform counts split that day's logins so range breakdowns are just
// sums over the requested window.
interface AnalyticsDayRecord {
  date: string; // "YYYY-MM-DD", local calendar day
  signups: number;
  logins: number;
  dau: number;
  failures: number;
  password: number;
  google: number;
  apple: number;
  ios: number;
  android: number;
  web: number;
  other: number;
}

// One raw event of the activity feed.
interface AnalyticsEventRecord {
  id: string;
  type: string;
  userId: string; // empty for events without a subject (login failures)
  provider: string;
  platform: string;
  time: Millis;
}

// One product's share of one rolled-up month.
interface AnalyticsTierRecord {
  productId: string;
  revenueUsdMicros: number;
  newSubscribers: number;
  activeSubscribers: number;
}

// One month of the subscription rollup. All demo money is USD.
interface AnalyticsMonthRecord {
  period: string; // "YYYY-MM"
  revenueUsdMicros: number;
  activeSubscribers: number;
  newSubscribers: number;
  renewals: number;
  churned: number;
  trialsStarted: number;
  trialsConverted: number;
  appleUsdMicros: number;
  googleUsdMicros: number;
  stripeUsdMicros: number;
  tiers: AnalyticsTierRecord[];
}

export interface AnalyticsSlice {
  // Per project, oldest day first, contiguous, ending yesterday.
  analyticsDaily: Record<string, AnalyticsDayRecord[]>;
  // Per project, newest event first.
  analyticsEvents: Record<string, AnalyticsEventRecord[]>;
  // Per project, oldest month first, contiguous, ending the current month.
  analyticsMonthly: Record<string, AnalyticsMonthRecord[]>;
  // Live all-time user count backing the "Total users" tile.
  analyticsTotalUsers: Record<string, number>;
  // When the aggregate-and-prune job last ran (updated by RunRollup).
  analyticsLastRollupTime: Millis;
}

// ---------- Calendar helpers (local timezone, like the UI's lib/format) ----------

const DAY_RE = /^\d{4}-\d{2}-\d{2}$/;
const MONTH_RE = /^\d{4}-\d{2}$/;

function pad2(n: number): string {
  return String(n).padStart(2, "0");
}

function dayKeyOfDate(d: Date): string {
  return `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}`;
}

function monthKeyOfDate(d: Date): string {
  return `${d.getFullYear()}-${pad2(d.getMonth() + 1)}`;
}

function parseDayKey(s: string): Date {
  if (!DAY_RE.test(s)) {
    throw invalidArgument(`invalid date "${s}" (want YYYY-MM-DD)`);
  }
  const [y, m, d] = s.split("-").map(Number);
  return new Date(y, m - 1, d);
}

function checkMonthKey(s: string): string {
  if (!MONTH_RE.test(s)) {
    throw invalidArgument(`invalid period "${s}" (want YYYY-MM)`);
  }
  return s;
}

function addDaysD(d: Date, n: number): Date {
  const out = new Date(d);
  out.setDate(out.getDate() + n);
  return out;
}

function shiftMonthKey(key: string, delta: number): string {
  const [y, m] = key.split("-").map(Number);
  return monthKeyOfDate(new Date(y, m - 1 + delta, 1));
}

// usd renders a micros amount as the per-currency list the proto wants,
// omitting zero so empty months read as "no data" rather than "$0.00 USD".
function usd(micros: number): { currency: string; amountMicros: bigint }[] {
  return micros !== 0 ? [{ currency: "USD", amountMicros: BigInt(micros) }] : [];
}

// ---------- Seed: daily auth rollups ----------

// splitCount apportions `total` units across weighted buckets one unit at a
// time, so per-day provider/platform splits carry natural noise but always
// sum back to the day's login count.
function splitCount(total: number, weights: number[]): number[] {
  const out = new Array<number>(weights.length).fill(0);
  for (let u = 0; u < total; u++) {
    let r = Math.random();
    let i = 0;
    for (; i < weights.length - 1; i++) {
      r -= weights[i];
      if (r <= 0) break;
    }
    out[i]++;
  }
  return out;
}

// Aurora Journal: 120 days ending yesterday. Signups are exact — each PEOPLE
// entry contributes its one signup on its real day — while engagement grows
// with the user base, dips on weekends, gets a small Monday bump and carries
// multiplicative noise. Provider weights mirror the PEOPLE mix (9 password /
// 8 google / 5 apple).
function seedMainDaily(): AnalyticsDayRecord[] {
  const out: AnalyticsDayRecord[] = [];
  for (let offset = 120; offset >= 1; offset--) {
    const date = new Date(daysAgo(offset));
    const users = PEOPLE.filter((p) => p.signupDaysAgo >= offset).length;
    const signups = PEOPLE.filter((p) => p.signupDaysAgo === offset).length;
    const dow = date.getDay();
    const weekday = dow === 0 || dow === 6 ? 0.62 : dow === 1 ? 1.12 : 1;
    const noise = 0.75 + Math.random() * 0.5;
    let dau = Math.round(users * 0.24 * weekday * noise);
    dau = Math.min(users, Math.max(dau, signups > 0 ? 1 : 0));
    const logins = dau === 0 ? 0 : Math.max(dau, Math.round(dau * (1.15 + Math.random() * 0.55)));
    const r = Math.random();
    const failures = r < 0.05 ? 2 : r < 0.25 ? 1 : 0;
    const [password, google, apple] = splitCount(logins, [9 / 22, 8 / 22, 5 / 22]);
    const [ios, android, web, other] = splitCount(logins, [0.5, 0.3, 0.15, 0.05]);
    out.push({
      date: dayKeyOfDate(date),
      signups,
      logins,
      dau,
      failures,
      password,
      google,
      apple,
      ios,
      android,
      web,
      other,
    });
  }
  return out;
}

// Skylark: created three days ago — two signups, a handful of logins, no
// failures. Enough for the feed to breathe, not enough to look configured.
function seedSideDaily(): AnalyticsDayRecord[] {
  const rows = [
    { off: 3, signups: 1, logins: 1, dau: 1 },
    { off: 2, signups: 1, logins: 2, dau: 2 },
    { off: 1, signups: 0, logins: 1, dau: 1 },
  ];
  return rows.map((r) => ({
    date: dayKeyOfDate(new Date(daysAgo(r.off))),
    signups: r.signups,
    logins: r.logins,
    dau: r.dau,
    failures: 0,
    password: r.logins,
    google: 0,
    apple: 0,
    ios: r.logins,
    android: 0,
    web: 0,
    other: 0,
  }));
}

// ---------- Seed: recent events ----------

// Deterministic per-person platform, so the same user always shows up on the
// same device family across the feed.
const PERSON_PLATFORMS = ["ios", "android", "ios", "web", "ios", "android"];

function platformFor(idx: number): string {
  return PERSON_PLATFORMS[idx % PERSON_PLATFORMS.length];
}

// Aurora Journal's feed: ~30 events over the last three days referencing
// real PEOPLE — logins and session refreshes mostly, plus the two newest
// signups (Emma 1d ago, Omar 3d ago, matching their signupDaysAgo), a couple
// of anonymous failed sign-ins, one password reset, one email verification
// and one identity link. Newest first.
function seedMainEvents(): AnalyticsEventRecord[] {
  const out: AnalyticsEventRecord[] = [];
  let n = 0;
  const ev = (
    minsAgo: number,
    type: string,
    personIdx: number | null,
    provider?: string,
    platform?: string,
  ): void => {
    n += 1;
    const p = personIdx === null ? null : PEOPLE[personIdx];
    const loginLike = type === "user.login" || type === "user.signup";
    out.push({
      id: demoId("evnt", n),
      type,
      userId: p ? p.id : "",
      provider: provider ?? (p && loginLike ? p.provider : ""),
      platform: platform ?? (personIdx !== null ? platformFor(personIdx) : ""),
      time: minutesAgo(minsAgo),
    });
  };

  ev(25, "user.login", 21); // Emma
  ev(41, "token.refresh", 0); // Maya
  ev(68, "user.login", 20); // Omar
  ev(95, "user.login", 17); // Aisha
  ev(132, "user.login_failed", null, "password", "web");
  ev(170, "user.login", 7); // Priya
  ev(214, "token.refresh", 10); // Liam
  ev(260, "user.login", 15); // Ingrid
  ev(305, "user.login", 16); // Marco
  ev(356, "email.verified", 21); // Emma
  ev(402, "user.login", 18); // Sven
  ev(450, "user.login", 4); // Yuki
  ev(495, "token.refresh", 2); // Amara
  ev(540, "user.login", 19); // Lucia
  ev(600, "user.login_failed", null, "password", "ios");
  ev(660, "user.login", 9); // Hannah
  ev(720, "user.login", 0); // Maya
  ev(800, "user.login", 5); // Claire, right after her reset
  ev(815, "password.reset_completed", 5, "password", "web");
  ev(900, "user.login", 8); // Diego
  ev(980, "token.refresh", 7); // Priya
  ev(1440, "user.signup", 21); // Emma joined a day ago
  ev(1560, "user.login", 11); // Zofia
  ev(1650, "user.login", 12); // Arthur
  ev(1800, "identity.linked", 16, "google"); // Marco linked Google
  ev(1900, "user.login", 14); // Felix
  ev(2100, "token.refresh", 15); // Ingrid
  ev(2600, "user.login", 20); // Omar
  ev(2900, "user.login", 13); // Nadia
  ev(3400, "user.login", 1); // Jonas
  ev(3900, "user.login", 6); // Mikkel
  ev(4320, "user.signup", 20); // Omar joined three days ago

  return out.sort((a, b) => b.time - a.time);
}

// Skylark's founder poking at a fresh project. The user ids are demo-minted;
// nothing in the analytics UI joins on them.
function seedSideEvents(): AnalyticsEventRecord[] {
  const founder = demoId("user", 23);
  const tester = demoId("user", 24);
  return [
    {
      id: demoId("evnt", 101),
      type: "user.login",
      userId: founder,
      provider: "password",
      platform: "ios",
      time: minutesAgo(1 * 1440 + 130),
    },
    {
      id: demoId("evnt", 102),
      type: "user.signup",
      userId: tester,
      provider: "password",
      platform: "ios",
      time: minutesAgo(2 * 1440 + 45),
    },
    {
      id: demoId("evnt", 103),
      type: "user.signup",
      userId: founder,
      provider: "password",
      platform: "ios",
      time: minutesAgo(3 * 1440 + 20),
    },
  ];
}

// ---------- Seed: monthly subscription rollups ----------

const PRICE_MONTHLY = 4_990_000; // $4.99
const PRICE_ANNUAL = 39_990_000; // $39.99
const PRICE_LIFETIME = 99_990_000; // $99.99

type StoreName = "apple" | "google" | "stripe";

// One subscriber's billing timeline, in days-ago. startDaysAgo is the first
// payment (null while still trialing); recurring plans repay every
// cadenceDays until churned; cadence null is a one-time purchase.
interface SubSpec {
  userId: string;
  productId: string;
  store: StoreName;
  priceMicros: number;
  cadenceDays: number | null;
  startDaysAgo: number | null;
  churnDaysAgo?: number;
  trialStartDaysAgo?: number;
}

// The subscription story, agreeing with ids.ts: Maya + Yuki on annual,
// Priya + Liam + Ingrid on monthly (Ingrid converted from a trial), Claire
// churned two months back, Amara bought lifetime, Omar is mid-trial.
const SUB_SPECS: SubSpec[] = [
  { userId: SUBSCRIBER_IDS[0], productId: PRODUCT_ANNUAL_ID, store: "apple", priceMicros: PRICE_ANNUAL, cadenceDays: 365, startDaysAgo: 155 },
  { userId: SUBSCRIBER_IDS[1], productId: PRODUCT_ANNUAL_ID, store: "google", priceMicros: PRICE_ANNUAL, cadenceDays: 365, startDaysAgo: 135 },
  { userId: SUBSCRIBER_IDS[2], productId: PRODUCT_MONTHLY_ID, store: "google", priceMicros: PRICE_MONTHLY, cadenceDays: 30, startDaysAgo: 100 },
  { userId: SUBSCRIBER_IDS[3], productId: PRODUCT_MONTHLY_ID, store: "apple", priceMicros: PRICE_MONTHLY, cadenceDays: 30, startDaysAgo: 70 },
  { userId: SUBSCRIBER_IDS[4], productId: PRODUCT_MONTHLY_ID, store: "google", priceMicros: PRICE_MONTHLY, cadenceDays: 30, startDaysAgo: 17, trialStartDaysAgo: 24 },
  { userId: CHURNED_ID, productId: PRODUCT_MONTHLY_ID, store: "stripe", priceMicros: PRICE_MONTHLY, cadenceDays: 30, startDaysAgo: 125, churnDaysAgo: 55 },
  { userId: LIFETIME_ID, productId: PRODUCT_LIFETIME_ID, store: "apple", priceMicros: PRICE_LIFETIME, cadenceDays: null, startDaysAgo: 45 },
  { userId: TRIAL_ID, productId: PRODUCT_MONTHLY_ID, store: "apple", priceMicros: PRICE_MONTHLY, cadenceDays: 30, startDaysAgo: null, trialStartDaysAgo: 2 },
];

function tierAt(rec: AnalyticsMonthRecord, productId: string): AnalyticsTierRecord {
  let t = rec.tiers.find((x) => x.productId === productId);
  if (!t) {
    t = { productId, revenueUsdMicros: 0, newSubscribers: 0, activeSubscribers: 0 };
    rec.tiers.push(t);
  }
  return t;
}

// seedMainMonthly replays SUB_SPECS into per-month buckets: payments become
// revenue + store/tier splits, first payments count as new subscribers,
// later ones as renewals; trials and churn land in their own months. Active
// subscribers per month is the distinct count of live recurring subs (the
// headline 5), lifetime counts only inside its own tier.
function seedMainMonthly(): AnalyticsMonthRecord[] {
  const months = new Map<string, AnalyticsMonthRecord>();
  const monthOf = (d: number): string => monthKeyOfDate(new Date(daysAgo(d)));
  const at = (key: string): AnalyticsMonthRecord => {
    let m = months.get(key);
    if (!m) {
      m = {
        period: key,
        revenueUsdMicros: 0,
        activeSubscribers: 0,
        newSubscribers: 0,
        renewals: 0,
        churned: 0,
        trialsStarted: 0,
        trialsConverted: 0,
        appleUsdMicros: 0,
        googleUsdMicros: 0,
        stripeUsdMicros: 0,
        tiers: [],
      };
      months.set(key, m);
    }
    return m;
  };

  for (const s of SUB_SPECS) {
    if (s.trialStartDaysAgo !== undefined) {
      at(monthOf(s.trialStartDaysAgo)).trialsStarted += 1;
    }
    if (s.startDaysAgo === null) continue; // still trialing, no payments yet
    if (s.trialStartDaysAgo !== undefined) {
      at(monthOf(s.startDaysAgo)).trialsConverted += 1;
    }
    let d = s.startDaysAgo;
    while (d >= 0 && (s.churnDaysAgo === undefined || d > s.churnDaysAgo)) {
      const rec = at(monthOf(d));
      rec.revenueUsdMicros += s.priceMicros;
      if (s.store === "apple") rec.appleUsdMicros += s.priceMicros;
      else if (s.store === "google") rec.googleUsdMicros += s.priceMicros;
      else rec.stripeUsdMicros += s.priceMicros;
      const tier = tierAt(rec, s.productId);
      tier.revenueUsdMicros += s.priceMicros;
      if (d === s.startDaysAgo) {
        rec.newSubscribers += 1;
        tier.newSubscribers += 1;
      } else {
        rec.renewals += 1;
      }
      if (s.cadenceDays === null) break; // one-time purchase
      d -= s.cadenceDays;
    }
    if (s.churnDaysAgo !== undefined) {
      at(monthOf(s.churnDaysAgo)).churned += 1;
    }
  }

  // Make the run of months contiguous up to the current month, then fill
  // the non-additive distinct-active counts per month.
  const currentKey = monthKeyOfDate(new Date());
  const firstKey = [...months.keys()].sort()[0] ?? currentKey;
  let cursor = firstKey;
  for (let i = 0; i < 24; i++) {
    at(cursor);
    if (cursor === currentKey) break;
    cursor = shiftMonthKey(cursor, 1);
  }

  for (const [key, rec] of months) {
    for (const s of SUB_SPECS) {
      if (s.startDaysAgo === null) continue;
      if (monthOf(s.startDaysAgo) > key) continue;
      if (s.cadenceDays === null) {
        // Lifetime owner: active within its tier, not a recurring subscriber.
        tierAt(rec, s.productId).activeSubscribers += 1;
        continue;
      }
      if (s.churnDaysAgo !== undefined && monthOf(s.churnDaysAgo) < key) continue;
      rec.activeSubscribers += 1;
      tierAt(rec, s.productId).activeSubscribers += 1;
    }
  }

  return [...months.values()].sort((a, b) => (a.period < b.period ? -1 : 1));
}

// ---------- Seed ----------

export function seedAnalytics(): AnalyticsSlice {
  return {
    analyticsDaily: {
      [PROJECT_MAIN.id]: seedMainDaily(),
      [PROJECT_SIDE.id]: seedSideDaily(),
    },
    analyticsEvents: {
      [PROJECT_MAIN.id]: seedMainEvents(),
      [PROJECT_SIDE.id]: seedSideEvents(),
    },
    analyticsMonthly: {
      [PROJECT_MAIN.id]: seedMainMonthly(),
      [PROJECT_SIDE.id]: [],
    },
    analyticsTotalUsers: {
      [PROJECT_MAIN.id]: PEOPLE.length,
      [PROJECT_SIDE.id]: 2,
    },
    // "The nightly job last ran early this morning."
    analyticsLastRollupTime: minutesAgo(190),
  };
}

// ---------- Handlers ----------

function sumDays(list: AnalyticsDayRecord[], f: (r: AnalyticsDayRecord) => number): number {
  return list.reduce((acc, r) => acc + f(r), 0);
}

export function registerAnalytics(): void {
  handle(AnalyticsService.method.getStats, (state: AnalyticsSlice, req) => {
    if (req.projectId === "") {
      throw invalidArgument("project_id is required");
    }
    if (req.granularity !== Granularity.UNSPECIFIED && req.granularity !== Granularity.DAY) {
      throw invalidArgument("only daily granularity is rolled up");
    }
    const days = state.analyticsDaily[req.projectId] ?? [];
    const byDate = new Map(days.map((d) => [d.date, d]));

    // The rollup materializes completed days only, so an omitted range
    // defaults to the last 30 days ending yesterday (what the overview's
    // failure banner asks for).
    const to = req.toDate !== "" ? parseDayKey(req.toDate) : new Date(daysAgo(1));
    const from = req.fromDate !== "" ? parseDayKey(req.fromDate) : addDaysD(to, -29);
    if (from.getTime() > to.getTime()) {
      throw invalidArgument("from_date is after to_date");
    }

    const series = [];
    const providers = { password: 0, google: 0, apple: 0 };
    const platforms = { ios: 0, android: 0, web: 0, other: 0 };
    for (let d = from, i = 0; d.getTime() <= to.getTime() && i < 400; d = addDaysD(d, 1), i++) {
      const key = dayKeyOfDate(d);
      const rec = byDate.get(key);
      series.push({
        date: key,
        signups: BigInt(rec?.signups ?? 0),
        logins: BigInt(rec?.logins ?? 0),
        dau: BigInt(rec?.dau ?? 0),
        failures: BigInt(rec?.failures ?? 0),
      });
      if (rec) {
        providers.password += rec.password;
        providers.google += rec.google;
        providers.apple += rec.apple;
        platforms.ios += rec.ios;
        platforms.android += rec.android;
        platforms.web += rec.web;
        platforms.other += rec.other;
      }
    }

    // The tile block always describes the last 7 completed days, regardless
    // of the requested chart range.
    const last7 = days.slice(-7);
    const prev7 = days.slice(-14, -7);
    const latest = days.length > 0 ? days[days.length - 1] : undefined;
    const logins7 = sumDays(last7, (r) => r.logins);
    const failures7 = sumDays(last7, (r) => r.failures);
    const attempts7 = logins7 + failures7;

    return {
      tiles: {
        totalUsers: BigInt(state.analyticsTotalUsers[req.projectId] ?? 0),
        newUsers7d: BigInt(sumDays(last7, (r) => r.signups)),
        newUsersPrevious7d: BigInt(sumDays(prev7, (r) => r.signups)),
        latestDau: BigInt(latest?.dau ?? 0),
        latestDauDate: latest?.date ?? "",
        logins7d: BigInt(logins7),
        loginFailures7d: BigInt(failures7),
        loginSuccessRate7d: attempts7 > 0 ? logins7 / attempts7 : 0,
      },
      series,
      providers: {
        password: BigInt(providers.password),
        google: BigInt(providers.google),
        apple: BigInt(providers.apple),
      },
      platforms: {
        ios: BigInt(platforms.ios),
        android: BigInt(platforms.android),
        web: BigInt(platforms.web),
        other: BigInt(platforms.other),
      },
    };
  });

  handle(AnalyticsService.method.listRecentEvents, (state: AnalyticsSlice, req) => {
    if (req.projectId === "") {
      throw invalidArgument("project_id is required");
    }
    const limit = req.limit > 0 ? Math.min(req.limit, 50) : 50;
    const events = (state.analyticsEvents[req.projectId] ?? [])
      .slice(0, limit)
      .map((e) => ({
        id: e.id,
        type: e.type,
        userId: e.userId,
        provider: e.provider,
        platform: e.platform,
        createTime: ts(e.time),
      }));
    return { events };
  });

  handle(AnalyticsService.method.runRollup, (state: AnalyticsSlice, req) => {
    const finished = now();
    state.analyticsLastRollupTime = finished;
    const projectIds =
      req.projectId !== "" ? [req.projectId] : Object.keys(state.analyticsDaily);
    let daysProcessed = 0;
    for (const id of projectIds) {
      daysProcessed += (state.analyticsDaily[id] ?? []).length;
    }
    return {
      runId: randomId(),
      startTime: ts(finished - 1400),
      finishTime: ts(finished),
      daysProcessed,
      // Re-rolling is idempotent and the demo retains everything.
      eventsPruned: BigInt(0),
    };
  });

  handle(AnalyticsService.method.getSubscriptionStats, (state: AnalyticsSlice, req) => {
    if (req.projectId === "") {
      throw invalidArgument("project_id is required");
    }
    const stored = state.analyticsMonthly[req.projectId] ?? [];
    const byPeriod = new Map(stored.map((m) => [m.period, m]));

    const toKey = req.toPeriod !== "" ? checkMonthKey(req.toPeriod) : monthKeyOfDate(new Date());
    const fromKey = req.fromPeriod !== "" ? checkMonthKey(req.fromPeriod) : shiftMonthKey(toKey, -5);
    if (fromKey > toKey) {
      throw invalidArgument("from_period is after to_period");
    }

    // Zero-filled monthly series over the inclusive range.
    const series = [];
    for (let k = fromKey, i = 0; k <= toKey && i < 60; k = shiftMonthKey(k, 1), i++) {
      const m = byPeriod.get(k);
      series.push({
        period: k,
        revenue: usd(m?.revenueUsdMicros ?? 0),
        activeSubscribers: BigInt(m?.activeSubscribers ?? 0),
        newSubscribers: BigInt(m?.newSubscribers ?? 0),
        renewals: BigInt(m?.renewals ?? 0),
        churned: BigInt(m?.churned ?? 0),
        trialsStarted: BigInt(m?.trialsStarted ?? 0),
        trialsConverted: BigInt(m?.trialsConverted ?? 0),
      });
    }

    // Headline tiles: the latest rolled-up month at or before to_period,
    // against the month before it.
    const latest = stored.filter((m) => m.period <= toKey).pop();
    const prev = latest ? byPeriod.get(shiftMonthKey(latest.period, -1)) : undefined;
    const tiles = {
      latestPeriod: latest?.period ?? "",
      revenueThisMonth: usd(latest?.revenueUsdMicros ?? 0),
      revenuePreviousMonth: usd(prev?.revenueUsdMicros ?? 0),
      activeSubscribers: BigInt(latest?.activeSubscribers ?? 0),
      activeSubscribersPrevious: BigInt(prev?.activeSubscribers ?? 0),
      newSubscribers: BigInt(latest?.newSubscribers ?? 0),
      churned: BigInt(latest?.churned ?? 0),
      trialsStarted: BigInt(latest?.trialsStarted ?? 0),
      trialsConverted: BigInt(latest?.trialsConverted ?? 0),
      trialConversionRate:
        latest && latest.trialsStarted > 0 ? latest.trialsConverted / latest.trialsStarted : 0,
    };

    // Per-tier: revenue and new subscribers sum over the range; active is the
    // non-additive distinct count of the latest month in range.
    const tierAgg = new Map<string, { revenue: number; added: number; active: number }>();
    let latestInRange: AnalyticsMonthRecord | undefined;
    const storeTotals = { apple: 0, google: 0, stripe: 0 };
    for (const m of stored) {
      if (m.period < fromKey || m.period > toKey) continue;
      latestInRange = m; // stored is ascending
      storeTotals.apple += m.appleUsdMicros;
      storeTotals.google += m.googleUsdMicros;
      storeTotals.stripe += m.stripeUsdMicros;
      for (const t of m.tiers) {
        const agg = tierAgg.get(t.productId) ?? { revenue: 0, added: 0, active: 0 };
        agg.revenue += t.revenueUsdMicros;
        agg.added += t.newSubscribers;
        tierAgg.set(t.productId, agg);
      }
    }
    if (latestInRange) {
      for (const t of latestInRange.tiers) {
        const agg = tierAgg.get(t.productId);
        if (agg) agg.active = t.activeSubscribers;
      }
    }
    const tierOrder = [PRODUCT_MONTHLY_ID, PRODUCT_ANNUAL_ID, PRODUCT_LIFETIME_ID];
    const orderIdx = (id: string): number => {
      const i = tierOrder.indexOf(id);
      return i === -1 ? tierOrder.length : i;
    };
    const tiers = [...tierAgg.entries()]
      .filter(([, v]) => v.revenue !== 0 || v.added !== 0 || v.active !== 0)
      .sort((a, b) => orderIdx(a[0]) - orderIdx(b[0]))
      .map(([productId, v]) => ({
        productId,
        revenue: usd(v.revenue),
        newSubscribers: BigInt(v.added),
        activeSubscribers: BigInt(v.active),
      }));

    return {
      tiles,
      series,
      tiers,
      stores: {
        apple: usd(storeTotals.apple),
        google: usd(storeTotals.google),
        stripe: usd(storeTotals.stripe),
      },
    };
  });
}
