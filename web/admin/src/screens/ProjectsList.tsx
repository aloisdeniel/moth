import { useMutation, useQuery } from "@connectrpc/connect-query";
import { useState } from "react";
import { Link, useNavigate } from "react-router";

import { errorMessage, invalidate } from "../api";
import { Dialog, ErrorNote, Field, KeyWell, Loading } from "../components/ui";
import type { Project } from "../gen/moth/admin/v1/project_pb";
import { ProjectService } from "../gen/moth/admin/v1/project_pb";
import { formatDate } from "../lib/format";

// slugPreview mirrors the server's Slugify (lowercase, [a-z0-9-]).
function slugPreview(name: string): string {
  const slug = name
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
  return slug || "project";
}

export function ProjectsList() {
  const projects = useQuery(ProjectService.method.listProjects);
  const [creating, setCreating] = useState(false);

  return (
    <main className="page">
      <div className="page__header">
        <h1>Projects</h1>
        <button type="button" className="btn btn--primary" onClick={() => setCreating(true)}>
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

      <CreateProjectDialog open={creating} onClose={() => setCreating(false)} />
    </main>
  );
}

function ProjectCard({ project }: { project: Project }) {
  return (
    <Link to={`/projects/${project.id}`} style={{ textDecoration: "none", color: "inherit" }}>
      <div className="card card--pad card--hover stack-12">
        <div className="card__title">{project.name}</div>
        <div className="mono text-secondary">{project.slug}</div>
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

function CreateProjectDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const navigate = useNavigate();
  const [name, setName] = useState("");
  const [created, setCreated] = useState<{ project: Project; secretKey: string }>();

  const create = useMutation(ProjectService.method.createProject, {
    onSuccess: (resp) => {
      invalidate(ProjectService.method.listProjects);
      if (resp.project) {
        setCreated({ project: resp.project, secretKey: resp.secretKey });
      }
    },
  });

  function close() {
    setName("");
    setCreated(undefined);
    create.reset();
    onClose();
  }

  if (created) {
    return (
      <Dialog title={`${created.project.name} is ready`} open={open} onClose={close}>
        <div className="stack-16">
          <div className="stack-8">
            <span className="field__label">Publishable key (for the app)</span>
            <KeyWell value={created.project.publishableKey} />
          </div>
          <div className="stack-8">
            <span className="field__label">Secret key (for your backend)</span>
            <KeyWell value={created.secretKey} secret />
            <p className="caption">
              You won't see this key again. Store it in your backend's secret
              manager.
            </p>
          </div>
          <div className="dialog__actions">
            <button
              type="button"
              className="btn btn--primary"
              onClick={() => {
                void navigate(`/projects/${created.project.id}`);
                close();
              }}
            >
              Open project
            </button>
          </div>
        </div>
      </Dialog>
    );
  }

  return (
    <Dialog title="Create project" open={open} onClose={close}>
      <form
        className="stack-16"
        onSubmit={(e) => {
          e.preventDefault();
          create.mutate({ name });
        }}
      >
        <Field
          label="Name"
          help={name ? `Slug: ${slugPreview(name)}` : "One project per app, e.g. \"Birdwatch\"."}
          error={create.isError ? errorMessage(create.error) : undefined}
        >
          <input
            className="input"
            value={name}
            onChange={(e) => setName(e.target.value)}
            autoFocus
            maxLength={100}
          />
        </Field>
        <div className="dialog__actions">
          <button type="button" className="btn btn--secondary" onClick={close}>
            Cancel
          </button>
          <button
            type="submit"
            className="btn btn--primary"
            disabled={create.isPending || name.trim() === ""}
          >
            {create.isPending ? "Creating…" : "Create project"}
          </button>
        </div>
      </form>
    </Dialog>
  );
}
