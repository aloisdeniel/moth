import { useQuery } from "@connectrpc/connect-query";
import { Link, useNavigate } from "react-router";

import { errorMessage } from "../api";
import { Sparkline } from "../components/charts";
import { Badge, ErrorNote, Loading } from "../components/ui";
import { AnalyticsService, Granularity } from "../gen/moth/admin/v1/analytics_pb";
import type { Project } from "../gen/moth/admin/v1/project_pb";
import { ProjectService } from "../gen/moth/admin/v1/project_pb";
import { failuresElevated } from "../lib/failures";
import { dayAgo, formatDate } from "../lib/format";

export function ProjectsList() {
  const projects = useQuery(ProjectService.method.listProjects);
  const navigate = useNavigate();

  return (
    <main className="page">
      <div className="page__header">
        <h1>Projects</h1>
        <button
          type="button"
          className="btn btn--primary"
          onClick={() => void navigate("/projects/new")}
        >
          Create project
        </button>
      </div>

      {projects.isPending && <Loading />}
      {projects.isError && <ErrorNote message={errorMessage(projects.error)} />}

      {projects.data &&
        (projects.data.projects.length === 0 ? (
          <div className="card empty">
            <p className="body-strong">No projects yet</p>
            <p className="caption">
              Each mobile app you ship is one project: its own users, keys and
              configuration.
            </p>
          </div>
        ) : (
          <div className="grid-cards">
            {projects.data.projects.map((p) => (
              <ProjectCard key={p.id} project={p} />
            ))}
          </div>
        ))}
    </main>
  );
}

function ProjectCard({ project }: { project: Project }) {
  // 30 completed days of logins as a sparkline (today is never rolled up,
  // so including it would fake a final dip); failures elevated → warning
  // badge. Analytics is decoration on this screen: while loading (or on
  // error) the card simply renders without it, keeping the list usable.
  const stats = useQuery(AnalyticsService.method.getStats, {
    projectId: project.id,
    fromDate: dayAgo(30),
    toDate: dayAgo(1),
    granularity: Granularity.DAY,
  });
  const tiles = stats.data?.tiles;
  const elevated = failuresElevated(tiles);
  const logins = stats.data?.series.map((d) => Number(d.logins)) ?? [];

  return (
    <Link to={`/projects/${project.id}`} style={{ textDecoration: "none", color: "inherit" }}>
      <div className="card card--pad card--hover stack-12">
        <div className="row-8">
          <div className="card__title" style={{ flex: 1 }}>
            {project.name}
          </div>
          {elevated && <Badge tone="warning">Failures elevated</Badge>}
        </div>
        <div className="mono text-secondary">{project.slug}</div>
        {logins.some((v) => v > 0) && (
          <div className="stack-8" style={{ gap: 4 }} title="Logins, last 30 days">
            <Sparkline values={logins} width={160} height={28} />
            <span className="caption text-tertiary">logins · 30d</span>
          </div>
        )}
        <div className="row-12 caption">
          <span className="tabular mono">
            {project.userCount.toString()} {project.userCount === 1n ? "user" : "users"}
          </span>
          <span>·</span>
          <span className="mono">{formatDate(project.createTime)}</span>
        </div>
      </div>
    </Link>
  );
}

