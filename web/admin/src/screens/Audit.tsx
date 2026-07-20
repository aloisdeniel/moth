import { useQuery } from "@connectrpc/connect-query";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import { useState } from "react";

import { errorMessage } from "../api";
import { ErrorNote, Field, Loading } from "../components/ui";
import { AuditService } from "../gen/moth/admin/v1/audit_pb";
import type { AuditEntry, ListAuditLogRequest } from "../gen/moth/admin/v1/audit_pb";
import { ProjectService } from "../gen/moth/admin/v1/project_pb";
import { formatDateTime, formatRelative } from "../lib/format";

const PAGE_SIZE = 50;

// Committed filter set. Kept separate from the draft inputs so typing does
// not fire a request on every keystroke — the query only re-runs on Apply.
interface Filters {
  projectId: string;
  actorId: string;
  action: string;
  from: string; // "YYYY-MM-DD" or ""
  to: string; // "YYYY-MM-DD" or ""
}

const EMPTY: Filters = { projectId: "", actorId: "", action: "", from: "", to: "" };

// The request init shape connect-query accepts (no generated $typeName).
type AuditRequestInit = Omit<Partial<ListAuditLogRequest>, "$typeName">;

// Audit is the instance-level viewer for the append-only admin audit log.
export function Audit() {
  const [draft, setDraft] = useState<Filters>(EMPTY);
  const [applied, setApplied] = useState<Filters>(EMPTY);
  const projects = useQuery(ProjectService.method.listProjects);

  const csvHref = auditCsvHref(applied);
  const appliedKey = JSON.stringify(applied);

  return (
    <main className="page">
      <h1>Audit</h1>
      <p className="caption" style={{ maxWidth: 640 }}>
        Every admin action — through a browser session or a personal access
        token — and security-relevant server event, newest first. Records are
        append-only and outlive the projects they reference.
      </p>

      <form
        className="card card--pad stack-16"
        onSubmit={(e) => {
          e.preventDefault();
          setApplied(draft);
        }}
      >
        <div className="row-16" style={{ flexWrap: "wrap", alignItems: "flex-end" }}>
          <Field label="Project">
            <select
              className="input"
              value={draft.projectId}
              onChange={(e) => setDraft({ ...draft, projectId: e.target.value })}
            >
              <option value="">All projects</option>
              {projects.data?.projects.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name}
                </option>
              ))}
            </select>
          </Field>
          <Field label="Action">
            <input
              className="input input--mono"
              value={draft.action}
              placeholder="signing_key.rotate"
              spellCheck={false}
              onChange={(e) => setDraft({ ...draft, action: e.target.value })}
            />
          </Field>
          <Field label="Actor id">
            <input
              className="input input--mono"
              value={draft.actorId}
              placeholder="admin id"
              spellCheck={false}
              onChange={(e) => setDraft({ ...draft, actorId: e.target.value })}
            />
          </Field>
          <Field label="From">
            <input
              className="input"
              type="date"
              value={draft.from}
              onChange={(e) => setDraft({ ...draft, from: e.target.value })}
            />
          </Field>
          <Field label="To">
            <input
              className="input"
              type="date"
              value={draft.to}
              onChange={(e) => setDraft({ ...draft, to: e.target.value })}
            />
          </Field>
        </div>
        <div className="row-12">
          <button type="submit" className="btn btn--primary">
            Apply filters
          </button>
          <button
            type="button"
            className="btn btn--ghost btn--compact"
            onClick={() => {
              setDraft(EMPTY);
              setApplied(EMPTY);
            }}
          >
            Clear
          </button>
          <span className="topbar__spacer" />
          <a className="caption" href={csvHref} download>
            Download CSV
          </a>
        </div>
      </form>

      <section className="card card--pad stack-12">
        <h3 className="card__title">Audit log</h3>
        {/* Remount on filter change so pagination resets to the first page. */}
        <AuditList key={appliedKey} filters={applied} />
      </section>
    </main>
  );
}

// AuditList renders one AuditPage per fetched page token, appending pages as
// the operator loads more. Each page owns its own query and is stable under
// its token key (strict-mode safe, no accumulation effect).
function AuditList({ filters }: { filters: Filters }) {
  const [tokens, setTokens] = useState<string[]>([""]);
  return (
    <div>
      {tokens.map((tok, i) => (
        <AuditPage
          key={tok === "" ? "first" : tok}
          filters={filters}
          token={tok}
          isLast={i === tokens.length - 1}
          onMore={(next) =>
            setTokens((ts) => (ts.includes(next) ? ts : [...ts, next]))
          }
        />
      ))}
    </div>
  );
}

function AuditPage({
  filters,
  token,
  isLast,
  onMore,
}: {
  filters: Filters;
  token: string;
  isLast: boolean;
  onMore: (next: string) => void;
}) {
  const q = useQuery(AuditService.method.listAuditLog, buildRequest(filters, token));

  if (q.isPending) return token === "" ? <Loading /> : null;
  if (q.isError) return <ErrorNote message={errorMessage(q.error)} />;

  const entries = q.data.entries;
  const next = q.data.nextPageToken;

  if (token === "" && entries.length === 0) {
    return (
      <div className="empty">
        <p className="body-strong">No audit entries</p>
        <p className="caption">
          Nothing matches these filters yet. Admin actions appear here as they
          happen.
        </p>
      </div>
    );
  }

  return (
    <>
      {entries.map((e) => (
        <AuditRow key={e.id} entry={e} />
      ))}
      {isLast && next !== "" && (
        <div className="row-12" style={{ marginTop: 8 }}>
          <button
            type="button"
            className="btn btn--secondary btn--compact"
            onClick={() => onMore(next)}
          >
            Load more
          </button>
        </div>
      )}
    </>
  );
}

// Humanized action names; unknown actions fall back to the raw machine name.
const ACTION_LABELS: Record<string, string> = {
  "project.create": "Created project",
  "project.update": "Updated project",
  "project.delete": "Deleted project",
  "provider.update": "Updated provider config",
  "signing_key.rotate": "Rotated signing key",
  "signing_key.reset": "Reset signing key",
  "secret_key.regenerate": "Regenerated secret key",
  "user.create": "Created user",
  "user.disable": "Disabled user",
  "user.enable": "Enabled user",
  "user.delete": "Deleted user",
  "family.revoke": "Revoked token family (reuse detected)",
  "admin.invite": "Invited admin",
  "admin.remove": "Removed admin",
  "pat.create": "Created access token",
  "pat.revoke": "Revoked access token",
  "settings.update": "Updated instance settings",
};

function AuditRow({ entry }: { entry: AuditEntry }) {
  const label = ACTION_LABELS[entry.action] ?? entry.action;
  const actor = entry.actorLabel || entry.actorId || entry.actorType || "system";
  const target =
    entry.targetType !== "" ? `${entry.targetType}${entry.targetId ? ` ${entry.targetId}` : ""}` : "";
  return (
    <div className="feed__row">
      <span>
        <span className="feed__what">{entry.summary || label}</span>
        <span className="feed__meta"> · {actor}</span>
        {target ? <span className="feed__meta"> · {target}</span> : null}
      </span>
      <span className="feed__time" title={formatDateTime(entry.createTime)}>
        {formatRelative(entry.createTime)}
      </span>
    </div>
  );
}

// buildRequest maps the committed filters to a ListAuditLog request, leaving
// unset fields empty (the server treats them as "no filter").
function buildRequest(f: Filters, token: string): AuditRequestInit {
  const req: AuditRequestInit = {
    projectId: f.projectId,
    actorId: f.actorId.trim(),
    action: f.action.trim(),
    pageSize: PAGE_SIZE,
    pageToken: token,
  };
  if (f.from !== "") req.startTime = timestampFromDate(dayStart(f.from));
  // end_time is an exclusive upper bound; add a day so the whole `to` day is
  // included.
  if (f.to !== "") req.endTime = timestampFromDate(dayStart(f.to, 1));
  return req;
}

function dayStart(day: string, plusDays = 0): Date {
  const d = new Date(`${day}T00:00:00Z`);
  if (plusDays) d.setUTCDate(d.getUTCDate() + plusDays);
  return d;
}

function auditCsvHref(f: Filters): string {
  const params = new URLSearchParams();
  if (f.projectId) params.set("project_id", f.projectId);
  if (f.actorId.trim()) params.set("actor_id", f.actorId.trim());
  if (f.action.trim()) params.set("action", f.action.trim());
  if (f.from) params.set("from", f.from);
  if (f.to) params.set("to", f.to);
  const q = params.toString();
  return `/admin/export/audit.csv${q ? `?${q}` : ""}`;
}
