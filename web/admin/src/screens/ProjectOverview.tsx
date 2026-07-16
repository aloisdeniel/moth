import { useMutation, useQuery } from "@connectrpc/connect-query";
import { useState } from "react";
import { useNavigate } from "react-router";

import { errorMessage, invalidate } from "../api";
import {
  ConfirmDialog,
  CopyButton,
  Dialog,
  ErrorNote,
  KeyWell,
  Loading,
} from "../components/ui";
import type { Project } from "../gen/moth/admin/v1/project_pb";
import { ProjectService } from "../gen/moth/admin/v1/project_pb";
import { formatDate } from "../lib/format";

export function ProjectOverview({ project }: { project: Project }) {
  return (
    <div className="stack-24">
      <PublishableKeyCard project={project} />
      <SecretKeyCard project={project} />
      <SigningKeyCard project={project} />
      <DangerZone project={project} />
    </div>
  );
}

function PublishableKeyCard({ project }: { project: Project }) {
  return (
    <section className="card card--pad stack-12">
      <h3 className="card__title">Publishable key</h3>
      <p className="caption">
        Identifies this project to the mobile SDK. Safe to embed in your app.
      </p>
      <KeyWell value={project.publishableKey} />
    </section>
  );
}

function SecretKeyCard({ project }: { project: Project }) {
  const [confirming, setConfirming] = useState(false);
  const [fresh, setFresh] = useState("");
  const regen = useMutation(ProjectService.method.regenerateSecretKey, {
    onSuccess: (resp) => {
      setConfirming(false);
      setFresh(resp.secretKey);
    },
  });

  return (
    <section className="card card--pad stack-12">
      <div className="page__header">
        <h3 className="card__title">Secret key</h3>
        <button
          type="button"
          className="btn btn--secondary btn--compact"
          onClick={() => setConfirming(true)}
        >
          Regenerate
        </button>
      </div>
      <p className="caption">
        Authenticates your backend to the <span className="inline-code">moth.server.v1</span> API.
        Stored hashed; shown only at creation.
      </p>
      <div className="keywell">
        <span className="keywell__value">sk_••••••••••••••••</span>
      </div>

      <ConfirmDialog
        title="Regenerate secret key"
        open={confirming}
        onClose={() => setConfirming(false)}
        onConfirm={() => regen.mutate({ projectId: project.id })}
        confirmLabel="Regenerate key"
        busy={regen.isPending}
        error={regen.isError ? errorMessage(regen.error) : undefined}
      >
        <p>
          The current secret key of <strong>{project.name}</strong> stops
          working immediately. Any backend using it will get authentication
          errors until you deploy the new key.
        </p>
      </ConfirmDialog>

      <Dialog title="New secret key" open={fresh !== ""} onClose={() => setFresh("")}>
        <div className="stack-16">
          <KeyWell value={fresh} secret />
          <p className="caption">
            You won't see this key again. Store it in your backend's secret
            manager.
          </p>
          <div className="dialog__actions">
            <button type="button" className="btn btn--primary" onClick={() => setFresh("")}>
              Done
            </button>
          </div>
        </div>
      </Dialog>
    </section>
  );
}

function SigningKeyCard({ project }: { project: Project }) {
  const key = useQuery(ProjectService.method.getSigningKey, { projectId: project.id });

  return (
    <section className="card card--pad stack-12">
      <h3 className="card__title">Token signing key</h3>
      <p className="caption">
        ES256 keypair minting this project's access tokens. Your backend
        verifies them offline against the JWKS.
      </p>
      {key.isPending && <Loading />}
      {key.isError && <ErrorNote message={errorMessage(key.error)} />}
      {key.data && key.data.key && (
        <div className="stack-12">
          <div className="stack-8">
            <span className="field__label">Key ID (kid)</span>
            <KeyWell value={key.data.key.kid} />
          </div>
          <div className="stack-8">
            <span className="field__label">JWKS URL</span>
            <KeyWell value={key.data.jwksUrl} />
          </div>
          <div className="row-8">
            <span className="caption">
              {key.data.key.algorithm} · created{" "}
              <span className="mono">{formatDate(key.data.key.createTime)}</span>
            </span>
            <span className="topbar__spacer" />
            <a
              className="btn btn--secondary btn--compact"
              href={`data:application/x-pem-file;charset=utf-8,${encodeURIComponent(key.data.key.publicKeyPem)}`}
              download={`${project.slug}-public-key.pem`}
              style={{ textDecoration: "none" }}
            >
              Download public key (PEM)
            </a>
            <CopyButton value={key.data.key.publicKeyPem} label="Copy PEM" />
          </div>
        </div>
      )}
    </section>
  );
}

function DangerZone({ project }: { project: Project }) {
  const navigate = useNavigate();
  const [resetting, setResetting] = useState(false);
  const [deleting, setDeleting] = useState(false);

  const reset = useMutation(ProjectService.method.resetSigningKey, {
    onSuccess: () => {
      setResetting(false);
      invalidate(ProjectService.method.getSigningKey);
    },
  });
  const del = useMutation(ProjectService.method.deleteProject, {
    onSuccess: () => {
      invalidate(ProjectService.method.listProjects);
      void navigate("/");
    },
  });

  return (
    <section className="card card--pad danger-zone">
      <h3 className="card__title">Danger zone</h3>
      <div className="danger-zone__row">
        <div className="stack-8" style={{ maxWidth: 560 }}>
          <span className="body-strong">Reset signing key</span>
          <span className="caption">
            Generates a new keypair, removes the old key from the JWKS
            immediately and revokes all refresh tokens. Every issued token
            becomes invalid and all users must sign in again.
          </span>
        </div>
        <button type="button" className="btn btn--danger" onClick={() => setResetting(true)}>
          Reset signing key
        </button>
      </div>
      <div className="danger-zone__row">
        <div className="stack-8" style={{ maxWidth: 560 }}>
          <span className="body-strong">Delete project</span>
          <span className="caption">
            Permanently deletes {project.name}, its users, keys and sessions.
            This cannot be undone.
          </span>
        </div>
        <button type="button" className="btn btn--danger" onClick={() => setDeleting(true)}>
          Delete project
        </button>
      </div>

      <ConfirmDialog
        title="Reset signing key"
        open={resetting}
        onClose={() => setResetting(false)}
        onConfirm={() => reset.mutate({ projectId: project.id })}
        confirmLabel="Reset signing key"
        confirmText={`reset ${project.slug}`}
        busy={reset.isPending}
        error={reset.isError ? errorMessage(reset.error) : undefined}
      >
        <p>
          This removes the current key from the JWKS <strong>immediately</strong>{" "}
          and revokes every refresh token of <strong>{project.name}</strong>:
        </p>
        <p className="caption">
          · Every access token ever issued stops validating.
          <br />· Every signed-in user of the app is logged out and must sign
          in again.
          <br />· Backends caching the old JWKS reject tokens until they
          refetch it.
        </p>
      </ConfirmDialog>

      <ConfirmDialog
        title="Delete project"
        open={deleting}
        onClose={() => setDeleting(false)}
        onConfirm={() => del.mutate({ id: project.id })}
        confirmLabel="Delete project"
        confirmText={project.slug}
        busy={del.isPending}
        error={del.isError ? errorMessage(del.error) : undefined}
      >
        <p>
          Permanently deletes <strong>{project.name}</strong> with all{" "}
          <span className="tabular">{project.userCount.toString()}</span> user
          accounts, keys and sessions. This cannot be undone.
        </p>
      </ConfirmDialog>
    </section>
  );
}
