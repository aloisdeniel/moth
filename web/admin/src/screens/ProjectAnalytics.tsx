import { useMutation, useQuery } from "@connectrpc/connect-query";
import { useState } from "react";

import { errorMessage, invalidate } from "../api";
import { BarBreakdown, LineChart, StatTile } from "../components/charts";
import { ErrorNote, Loading } from "../components/ui";
import type {
  CurrencyAmount,
  Event,
  SubscriptionMonthlyStat,
  SubscriptionTiles,
} from "../gen/moth/admin/v1/analytics_pb";
import { AnalyticsService, Granularity } from "../gen/moth/admin/v1/analytics_pb";
import type { Project } from "../gen/moth/admin/v1/project_pb";
import { ProductService } from "../gen/moth/admin/v1/product_pb";
import { formatMoney } from "../lib/billing";
import { churnElevated, failuresElevated, loginAttempts7d, monthlyChurnRate } from "../lib/failures";
import { dayAgo, formatRelative, monthAgo, monthKey } from "../lib/format";

const RANGES = [7, 30, 90] as const;
type Range = (typeof RANGES)[number];

const MONTH_RANGES = [3, 6, 12] as const;
type MonthRange = (typeof MONTH_RANGES)[number];

const DAU_TOOLTIP =
  "Approximation: distinct users with a sign-in or session refresh that day. " +
  "Users whose session stayed idle all day are not counted.";

export function ProjectAnalytics({ project }: { project: Project }) {
  const [range, setRange] = useState<Range>(30);
  // The rollup only materializes completed days, so the range ends at
  // yesterday — requesting today would append a fabricated all-zero point
  // to every chart.
  const fromDate = dayAgo(range);
  const toDate = dayAgo(1);

  const stats = useQuery(AnalyticsService.method.getStats, {
    projectId: project.id,
    fromDate,
    toDate,
    granularity: Granularity.DAY,
  });
  const events = useQuery(AnalyticsService.method.listRecentEvents, {
    projectId: project.id,
    limit: 50,
  });

  const rollup = useMutation(AnalyticsService.method.runRollup, {
    onSuccess: () => {
      invalidate(AnalyticsService.method.getStats, AnalyticsService.method.listRecentEvents);
    },
  });

  if (stats.isPending) {
    return <Loading />;
  }
  if (stats.isError) {
    return <ErrorNote message={errorMessage(stats.error)} />;
  }

  const tiles = stats.data.tiles;
  const series = stats.data.series;
  const labels = series.map((d) => d.date);
  const attempts7d = loginAttempts7d(tiles);
  const elevated = failuresElevated(tiles);

  const hasTraffic =
    series.some((d) => d.signups + d.logins + d.dau + d.failures > 0n) ||
    (events.data?.events.length ?? 0) > 0;

  const csvHref =
    `/admin/export/stats.csv?project_id=${encodeURIComponent(project.id)}` +
    `&from=${fromDate}&to=${toDate}`;

  return (
    <div className="stack-24">
      <div className="row-16">
        <div className="seg" role="group" aria-label="Date range">
          {RANGES.map((r) => (
            <button
              key={r}
              type="button"
              className="seg__btn"
              aria-pressed={range === r}
              onClick={() => setRange(r)}
            >
              {r}d
            </button>
          ))}
        </div>
        <span className="topbar__spacer" />
        <a className="caption" href={csvHref} download>
          Download CSV
        </a>
        <button
          type="button"
          className="btn btn--ghost btn--compact"
          disabled={rollup.isPending}
          onClick={() => rollup.mutate({ projectId: project.id })}
        >
          {rollup.isPending ? "Refreshing…" : "Refresh data"}
        </button>
      </div>

      {rollup.isError && <ErrorNote message={errorMessage(rollup.error)} />}

      {elevated && tiles && (
        <div className="banner banner--warning" role="status">
          Login failures elevated — {tiles.loginFailures7d.toString()} of{" "}
          {attempts7d} sign-in attempts failed over the last 7 days. Check the
          project's provider configuration.
        </div>
      )}

      {tiles && (
        <div className="stat-tiles">
          <StatTile
            label="Total users"
            value={tiles.totalUsers.toString()}
            hint="all time"
          />
          <StatTile
            label="New users"
            value={tiles.newUsers7d.toString()}
            delta={{
              current: Number(tiles.newUsers7d),
              previous: Number(tiles.newUsersPrevious7d),
            }}
            hint="last 7 days"
          />
          <StatTile
            label="Daily active users"
            value={tiles.latestDau.toString()}
            hint={tiles.latestDauDate ? `on ${tiles.latestDauDate}` : "no data yet"}
            title={DAU_TOOLTIP}
          />
          <StatTile
            label="Login success rate"
            value={attempts7d > 0 ? `${(tiles.loginSuccessRate7d * 100).toFixed(1)}%` : "—"}
            hint={
              attempts7d > 0
                ? `${tiles.logins7d.toString()} of ${attempts7d} attempts, 7 days`
                : "no attempts in 7 days"
            }
            warning={elevated}
          />
        </div>
      )}

      {!hasTraffic ? (
        <div className="card empty">
          <p className="body-strong">No activity yet</p>
          <p className="caption">
            Charts appear here once your app starts signing users up and in.
            Events are captured automatically — no extra SDK calls needed.
          </p>
        </div>
      ) : (
        <>
          <div className="chart-grid">
            <section className="card chart-card">
              <span className="chart-card__title">Signups per day</span>
              <LineChart
                labels={labels}
                series={[{ label: "Signups", values: series.map((d) => Number(d.signups)) }]}
              />
            </section>
            <section className="card chart-card">
              <span className="chart-card__title">Logins per day</span>
              <LineChart
                labels={labels}
                series={[{ label: "Logins", values: series.map((d) => Number(d.logins)) }]}
              />
            </section>
            <section className="card chart-card" title={DAU_TOOLTIP}>
              <span className="chart-card__title">Daily active users</span>
              <LineChart
                labels={labels}
                series={[{ label: "Active users", values: series.map((d) => Number(d.dau)) }]}
              />
            </section>
          </div>

          <div className="chart-grid">
            <section className="card chart-card">
              <span className="chart-card__title">Sign-in method · logins over {range}d</span>
              <BarBreakdown
                items={[
                  { label: "Password", value: Number(stats.data.providers?.password ?? 0n) },
                  { label: "Google", value: Number(stats.data.providers?.google ?? 0n) },
                  { label: "Apple", value: Number(stats.data.providers?.apple ?? 0n) },
                ]}
              />
            </section>
            <section className="card chart-card">
              <span className="chart-card__title">Platform · logins over {range}d</span>
              <BarBreakdown
                items={[
                  { label: "iOS", value: Number(stats.data.platforms?.ios ?? 0n) },
                  { label: "Android", value: Number(stats.data.platforms?.android ?? 0n) },
                  { label: "Web", value: Number(stats.data.platforms?.web ?? 0n) },
                  { label: "Other", value: Number(stats.data.platforms?.other ?? 0n) },
                ]}
              />
            </section>
          </div>

          <section className="card card--pad stack-12">
            <h3 className="card__title">Recent activity</h3>
            {events.isPending && <Loading />}
            {events.isError && <ErrorNote message={errorMessage(events.error)} />}
            {events.data &&
              (events.data.events.length === 0 ? (
                <p className="caption">No events recorded yet.</p>
              ) : (
                <div>
                  {events.data.events.map((e) => (
                    <FeedRow key={e.id} event={e} />
                  ))}
                </div>
              ))}
          </section>
        </>
      )}

      <SubscriptionSection project={project} />
    </div>
  );
}

// ---------- Subscriptions (milestone 14) ----------

// amountFor returns the micros recorded for one currency in a per-currency
// list (0 when absent). Money is always read per currency — never blended.
function amountFor(list: CurrencyAmount[], currency: string): bigint {
  return list.find((a) => a.currency === currency)?.amountMicros ?? 0n;
}

// collectCurrencies gathers the distinct ISO codes present anywhere in the
// response, so the selector only offers currencies that actually have data.
function collectCurrencies(
  tiles: SubscriptionTiles | undefined,
  series: SubscriptionMonthlyStat[],
): string[] {
  const set = new Set<string>();
  const add = (list: CurrencyAmount[]) =>
    list.forEach((a) => {
      if (a.currency && a.amountMicros !== 0n) set.add(a.currency);
    });
  if (tiles) {
    add(tiles.revenueThisMonth);
    add(tiles.revenuePreviousMonth);
  }
  series.forEach((m) => add(m.revenue));
  return [...set].sort();
}

// primaryCurrency picks the default currency to show first: the one with the
// largest revenue, preferring the latest (current) month and falling back to
// the whole series. Defaulting to the alphabetically-first currency would bury
// the headline revenue of the dominant currency (e.g. show a 5% EUR slice
// before 95% USD just because "EUR" < "USD"). The selector still lists every
// currency, sorted, so the user can switch. Only positive amounts count toward
// "largest" so a net-negative refund month does not win the default.
function primaryCurrency(
  tiles: SubscriptionTiles | undefined,
  series: SubscriptionMonthlyStat[],
  currencies: string[],
): string {
  if (currencies.length === 0) return "";
  const totals = new Map<string, bigint>();
  const add = (list: CurrencyAmount[]) =>
    list.forEach((a) => {
      if (a.currency && a.amountMicros > 0n) {
        totals.set(a.currency, (totals.get(a.currency) ?? 0n) + a.amountMicros);
      }
    });
  if (tiles) add(tiles.revenueThisMonth);
  if (totals.size === 0) series.forEach((m) => add(m.revenue));
  let best = currencies[0];
  let bestValue = -1n;
  for (const c of currencies) {
    const v = totals.get(c) ?? 0n;
    if (v > bestValue) {
      bestValue = v;
      best = c;
    }
  }
  return best;
}

// SubscriptionSection is the milestone-14 revenue dashboard: headline is
// total revenue per month, per currency (never a blended total). All numbers
// come from the pre-aggregated monthly rollup via GetSubscriptionStats.
function SubscriptionSection({ project }: { project: Project }) {
  const [months, setMonths] = useState<MonthRange>(6);
  const [pickedCurrency, setPickedCurrency] = useState<string>("");

  // The monthly rollup materializes completed months up to the current one;
  // the server zero-fills the range so charts stay contiguous.
  const fromPeriod = monthAgo(months - 1);
  const toPeriod = monthKey(new Date());

  const sub = useQuery(AnalyticsService.method.getSubscriptionStats, {
    projectId: project.id,
    fromPeriod,
    toPeriod,
  });
  const products = useQuery(ProductService.method.listProducts, { projectId: project.id });

  const productName = (id: string): string => {
    if (!id) return "Other";
    const p = products.data?.products.find((p) => p.id === id);
    return p ? p.displayName || p.identifier : id.slice(0, 8);
  };

  if (sub.isPending) {
    return (
      <section className="card card--pad">
        <h3 className="card__title">Subscriptions</h3>
        <Loading />
      </section>
    );
  }
  if (sub.isError) {
    return (
      <section className="card card--pad stack-12">
        <h3 className="card__title">Subscriptions</h3>
        <ErrorNote message={errorMessage(sub.error)} />
      </section>
    );
  }

  const tiles = sub.data.tiles;
  const series = sub.data.series;
  const tiers = sub.data.tiers;
  const stores = sub.data.stores;

  const currencies = collectCurrencies(tiles, series);
  const currency = currencies.includes(pickedCurrency)
    ? pickedCurrency
    : primaryCurrency(tiles, series, currencies);
  const elevated = churnElevated(tiles);

  const hasSubs =
    (tiles?.latestPeriod ?? "") !== "" ||
    series.some(
      (m) =>
        m.revenue.length > 0 ||
        m.activeSubscribers + m.newSubscribers + m.renewals + m.churned + m.trialsStarted > 0n,
    );

  const csvHref =
    `/admin/export/subscriptions.csv?project_id=${encodeURIComponent(project.id)}` +
    `&from=${fromPeriod}&to=${toPeriod}`;

  const header = (
    <div className="row-16">
      <h2 className="title-3">Subscriptions</h2>
      <span className="topbar__spacer" />
      <div className="seg" role="group" aria-label="Month range">
        {MONTH_RANGES.map((r) => (
          <button
            key={r}
            type="button"
            className="seg__btn"
            aria-pressed={months === r}
            onClick={() => setMonths(r)}
          >
            {r}m
          </button>
        ))}
      </div>
    </div>
  );

  if (!hasSubs) {
    return (
      <section className="stack-12">
        {header}
        <div className="card empty">
          <p className="body-strong">No subscriptions yet</p>
          <p className="caption">
            Revenue, active subscribers and churn appear here once this app
            starts selling subscriptions. Store events are captured
            automatically — no extra SDK calls needed.
          </p>
        </div>
      </section>
    );
  }

  // Axis ticks drop the cents; tiles and tooltips keep them.
  const axisMoney = (v: number) => formatMoney(BigInt(Math.round(v * 1_000_000)), currency, 0);
  const revenueValues = series.map((m) => Number(amountFor(m.revenue, currency)) / 1_000_000);
  const labels = series.map((m) => m.period);

  const revenueThisMonth = tiles ? amountFor(tiles.revenueThisMonth, currency) : 0n;
  const revenuePrevMonth = tiles ? amountFor(tiles.revenuePreviousMonth, currency) : 0n;

  return (
    <section className="stack-24">
      <div className="row-16">
        <h2 className="title-3">Subscriptions</h2>
        <span className="topbar__spacer" />
        {currencies.length > 1 && (
          <div className="seg" role="group" aria-label="Currency">
            {currencies.map((c) => (
              <button
                key={c}
                type="button"
                className="seg__btn"
                aria-pressed={currency === c}
                onClick={() => setPickedCurrency(c)}
              >
                {c}
              </button>
            ))}
          </div>
        )}
        <div className="seg" role="group" aria-label="Month range">
          {MONTH_RANGES.map((r) => (
            <button
              key={r}
              type="button"
              className="seg__btn"
              aria-pressed={months === r}
              onClick={() => setMonths(r)}
            >
              {r}m
            </button>
          ))}
        </div>
        <a className="caption" href={csvHref} download>
          Download CSV
        </a>
      </div>

      {elevated && tiles && (
        <div className="banner banner--warning" role="status">
          Elevated churn — {tiles.churned.toString()} of{" "}
          {tiles.activeSubscribersPrevious.toString()} subscribers churned this
          month ({Math.round(monthlyChurnRate(tiles) * 100)}%). Check for a spike
          in billing failures or a misconfigured store key.
        </div>
      )}

      <div className="stat-tiles">
        <StatTile
          label="Active subscribers"
          value={tiles ? tiles.activeSubscribers.toString() : "0"}
          delta={
            tiles
              ? {
                  current: Number(tiles.activeSubscribers),
                  previous: Number(tiles.activeSubscribersPrevious),
                }
              : undefined
          }
          hint={tiles?.latestPeriod ? `as of ${tiles.latestPeriod}` : "this month"}
        />
        <StatTile
          label="Revenue this month"
          value={formatMoney(revenueThisMonth, currency)}
          delta={{ current: Number(revenueThisMonth), previous: Number(revenuePrevMonth) }}
          hint={currency ? `${currency} · store-reported gross` : "store-reported gross"}
          title="Store-reported gross list revenue, net of refunds. Not net of store commission or tax; never blended across currencies."
        />
        <StatTile
          label="Trial-to-paid"
          value={
            tiles && tiles.trialsStarted > 0n
              ? `${(tiles.trialConversionRate * 100).toFixed(0)}%`
              : "—"
          }
          hint={
            tiles && tiles.trialsStarted > 0n
              ? `${tiles.trialsConverted.toString()} of ${tiles.trialsStarted.toString()} trials`
              : "no trials this month"
          }
        />
        <StatTile
          label="New vs churned"
          value={tiles ? `+${tiles.newSubscribers.toString()} / −${tiles.churned.toString()}` : "—"}
          hint="this month"
          warning={elevated}
        />
      </div>

      <section className="card chart-card">
        <span className="chart-card__title">
          Revenue per month{currency ? ` · ${currency}` : ""} · store-reported gross
        </span>
        <LineChart
          labels={labels}
          series={[{ label: "Revenue", values: revenueValues }]}
          formatValue={axisMoney}
        />
      </section>

      <div className="chart-grid">
        <section className="card chart-card">
          <span className="chart-card__title">Active subscribers by tier · {months}m</span>
          {tiers.length === 0 ? (
            <p className="caption">No tier data in this range.</p>
          ) : (
            <BarBreakdown
              items={tiers.map((t) => ({
                label: productName(t.productId),
                value: Number(t.activeSubscribers),
              }))}
            />
          )}
        </section>
        <section className="card chart-card">
          <span className="chart-card__title">
            Revenue by store{currency ? ` · ${currency}` : ""} · {months}m
          </span>
          <BarBreakdown
            items={[
              {
                label: "Apple",
                value: Math.round(Number(amountFor(stores?.apple ?? [], currency)) / 1_000_000),
              },
              {
                label: "Google",
                value: Math.round(Number(amountFor(stores?.google ?? [], currency)) / 1_000_000),
              },
              {
                // Stripe money is the web channel in the store dimension.
                label: "Web",
                value: Math.round(Number(amountFor(stores?.stripe ?? [], currency)) / 1_000_000),
              },
            ]}
          />
        </section>
      </div>
    </section>
  );
}

// Humanized copy for the activity feed: "Sign-in with Google · iOS · 2h ago".

const EVENT_LABELS: Record<string, string> = {
  "user.signup": "Sign-up",
  "user.login": "Sign-in",
  "user.login_failed": "Failed sign-in",
  "token.refresh": "Session refresh",
  "password.reset_completed": "Password reset",
  "email.verified": "Email verified",
  "user.deleted": "Account deleted",
  "identity.linked": "Identity linked",
};

const PROVIDER_LABELS: Record<string, string> = {
  password: "password",
  google: "Google",
  apple: "Apple",
};

const PLATFORM_LABELS: Record<string, string> = {
  ios: "iOS",
  android: "Android",
  web: "Web",
};

function FeedRow({ event }: { event: Event }) {
  const what = EVENT_LABELS[event.type] ?? event.type;
  const provider = PROVIDER_LABELS[event.provider];
  const platform = PLATFORM_LABELS[event.platform];
  const failed = event.type === "user.login_failed";
  return (
    <div className="feed__row">
      <span>
        <span className={failed ? "text-danger" : "feed__what"}>
          {what}
          {provider ? ` with ${provider}` : ""}
        </span>
        {platform ? <span className="feed__meta"> · {platform}</span> : null}
      </span>
      <span className="feed__time">{formatRelative(event.createTime)}</span>
    </div>
  );
}
