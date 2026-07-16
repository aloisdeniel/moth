import { useQuery } from "@connectrpc/connect-query";
import { useNavigate, useParams } from "react-router";

import { errorMessage } from "../api";
import { ErrorNote, Loading } from "../components/ui";
import { ProjectService } from "../gen/moth/admin/v1/project_pb";
import { ProjectDesign } from "./ProjectDesign";
import { ProjectOverview } from "./ProjectOverview";
import { ProjectProviders } from "./ProjectProviders";
import { ProjectSettings } from "./ProjectSettings";
import { ProjectSetup } from "./ProjectSetup";
import { ProjectUsers } from "./ProjectUsers";

const TABS = [
  { id: "overview", label: "Overview" },
  { id: "users", label: "Users" },
  { id: "providers", label: "Providers" },
  { id: "design", label: "Design" },
  { id: "settings", label: "Settings" },
  { id: "setup", label: "Setup" },
] as const;

type TabID = (typeof TABS)[number]["id"];

export function Project() {
  const { projectId = "", tab = "overview" } = useParams();
  const navigate = useNavigate();
  const project = useQuery(ProjectService.method.getProject, { id: projectId });

  const active: TabID = (TABS.find((t) => t.id === tab)?.id ?? "overview") as TabID;

  if (project.isPending) {
    return (
      <main className="page">
        <Loading />
      </main>
    );
  }
  if (project.isError) {
    return (
      <main className="page">
        <ErrorNote message={errorMessage(project.error)} />
      </main>
    );
  }
  const p = project.data.project;
  if (!p) return null;

  return (
    <main className="page">
      <div className="stack-8">
        <h1>{p.name}</h1>
        <span className="mono text-secondary">{p.slug}</span>
      </div>

      <div className="tabs" role="tablist">
        {TABS.map((t) => (
          <button
            key={t.id}
            role="tab"
            aria-selected={active === t.id}
            className="tabs__tab"
            onClick={() =>
              void navigate(`/projects/${projectId}${t.id === "overview" ? "" : `/${t.id}`}`)
            }
          >
            {t.label}
          </button>
        ))}
      </div>

      {active === "overview" && <ProjectOverview project={p} />}
      {active === "users" && <ProjectUsers project={p} />}
      {active === "providers" && <ProjectProviders project={p} />}
      {active === "design" && <ProjectDesign project={p} />}
      {active === "settings" && <ProjectSettings project={p} />}
      {active === "setup" && <ProjectSetup project={p} />}
    </main>
  );
}
