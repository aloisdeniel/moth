import { useMutation, useQuery } from "@connectrpc/connect-query";
import { useState } from "react";

import { errorMessage, invalidate } from "../api";
import { BarBreakdown, LineChart, StatTile } from "../components/charts";
import { ErrorNote, Loading } from "../components/ui";
import type { Event } from "../gen/moth/admin/v1/analytics_pb";
import { AnalyticsService, Granularity } from "../gen/moth/admin/v1/analytics_pb";
import type { Project } from "../gen/moth/admin/v1/project_pb";
import { failuresElevated, loginAttempts7d } from "../lib/failures";
import { dayAgo, formatRelative } from "../lib/format";

const RANGES = [7, 30, 90] as const;
type Range = (typeof RANGES)[number];

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
    </div>
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
