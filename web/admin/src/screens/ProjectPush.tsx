import { useMutation, useQuery } from "@connectrpc/connect-query";
import { useEffect, useState } from "react";
import { Link } from "react-router";

import { errorMessage, invalidate } from "../api";
import { Badge, ConfirmDialog, ErrorNote, Field, Loading } from "../components/ui";
import type { Project } from "../gen/moth/admin/v1/project_pb";
import type { ProjectPushDevice } from "../gen/moth/admin/v1/push_pb";
import { PushService, PushTarget } from "../gen/moth/admin/v1/push_pb";
import { formatRelative } from "../lib/format";
import { pushPermissionMeta, pushTargetLabel, vapidKeyError } from "../lib/push";

// ProjectPush is the dedicated Push tab: the project-wide device registry
// (every signed-in device's registration, with its owning user) above the
// push settings. Tokens never reach the admin surface — senders read them
// over moth.server.v1.
export function ProjectPush({ project }: { project: Project }) {
  return (
    <div className="stack-24">
      <DevicesCard project={project} />
      <PushSection project={project} />
    </div>
  );
}

const TARGET_FILTERS = [
  { id: PushTarget.UNSPECIFIED, label: "All targets" },
  { id: PushTarget.APNS, label: "APNs" },
  { id: PushTarget.FCM, label: "FCM" },
  { id: PushTarget.WEBPUSH, label: "Web Push" },
] as const;

const PAGE_SIZE = 50;

// DevicesCard lists the project's active registrations, newest first, with a
// target filter and keyset load-more. Each row names the owning user (the
// per-user history, including revoked rows, lives in the Users tab drawer).
function DevicesCard({ project }: { project: Project }) {
  const [target, setTarget] = useState<PushTarget>(PushTarget.UNSPECIFIED);
  const [pageToken, setPageToken] = useState("");
  const [rows, setRows] = useState<ProjectPushDevice[]>([]);
  const [revoking, setRevoking] = useState<ProjectPushDevice>();

  const page = useQuery(PushService.method.listPushDevices, {
    projectId: project.id,
    pageSize: PAGE_SIZE,
    pageToken,
    target,
  });

  // Pages accumulate; a filter change resets the accumulation (pageToken
  // goes back to "" so the query refetches from the top).
  useEffect(() => {
    if (!page.data) return;
    setRows((prev) => (pageToken === "" ? page.data.devices : [...prev, ...page.data.devices]));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page.data]);

  const revoke = useMutation(PushService.method.revokePushDevice, {
    onSuccess: () => {
      invalidate(PushService.method.listPushDevices);
      invalidate(PushService.method.listUserPushDevices);
      setPageToken("");
      setRevoking(undefined);
    },
  });

  function metadataLine(d: ProjectPushDevice): string {
    const m = d.device?.metadata;
    const parts = [
      [m?.platform, m?.model].filter(Boolean).join(" "),
      m?.osVersion ? `OS ${m.osVersion}` : "",
      m?.appVersion ? `app ${m.appVersion}` : "",
      m?.locale ?? "",
    ].filter(Boolean);
    return parts.length === 0 ? "no device metadata" : parts.join(" · ");
  }

  const counts = page.data;
  const total = counts
    ? Number(counts.apnsCount) + Number(counts.fcmCount) + Number(counts.webpushCount)
    : 0;

  return (
    <section className="card card--pad stack-16">
      <div className="row-12" style={{ justifyContent: "space-between", flexWrap: "wrap" }}>
        <h3 className="card__title">Registered devices</h3>
        {counts && total > 0 && (
          <span className="row-8">
            <Badge>{`APNs ${counts.apnsCount}`}</Badge>
            <Badge>{`FCM ${counts.fcmCount}`}</Badge>
            <Badge>{`Web Push ${counts.webpushCount}`}</Badge>
          </span>
        )}
      </div>
      <p className="caption">
        Every signed-in device that registered a push credential, newest
        first. Your backend reads this registry (tokens included) over{" "}
        <span className="inline-code">moth.server.v1.PushService</span> and
        sends the notifications itself. A user's full device history —
        including revoked registrations — is in their drawer on the{" "}
        <Link to={`/projects/${project.id}/users`}>Users tab</Link>.
      </p>

      <div className="seg" role="group" aria-label="Push target">
        {TARGET_FILTERS.map((t) => (
          <button
            key={t.id}
            type="button"
            className="seg__btn"
            aria-pressed={target === t.id}
            onClick={() => {
              setTarget(t.id);
              setPageToken("");
            }}
          >
            {t.label}
          </button>
        ))}
      </div>

      {page.isPending && rows.length === 0 && <Loading />}
      {page.isError && <ErrorNote message={errorMessage(page.error)} />}
      {page.data && rows.length === 0 && (
        <p className="caption">
          {target === PushTarget.UNSPECIFIED
            ? "No devices registered yet. Devices appear here as soon as a signed-in app with a push adapter (moth_push, or useMothPush on the web) registers its credential."
            : `No active ${pushTargetLabel(target)} registrations.`}
        </p>
      )}

      {rows.length > 0 && (
        <div className="stack-8">
          {rows.map((r) => {
            const d = r.device;
            if (!d) return null;
            const perm = pushPermissionMeta(d.permission);
            return (
              <div key={d.id} className="keywell" style={{ alignItems: "flex-start" }}>
                <div className="keywell__value stack-8" style={{ gap: 2 }}>
                  <span className="row-8" style={{ flexWrap: "wrap" }}>
                    <span className="mono">{r.userEmail || r.userId}</span>
                    <Badge>{pushTargetLabel(d.target)}</Badge>
                    <Badge tone={perm.tone}>{perm.label}</Badge>
                  </span>
                  <span className="caption">{metadataLine(r)}</span>
                  <span className="caption">last seen {formatRelative(d.lastSeenTime)}</span>
                </div>
                <button
                  type="button"
                  className="btn btn--ghost btn--compact"
                  onClick={() => setRevoking(r)}
                >
                  Revoke
                </button>
              </div>
            );
          })}
          {page.data?.nextPageToken && (
            <button
              type="button"
              className="btn btn--ghost"
              disabled={page.isPending}
              onClick={() => setPageToken(page.data.nextPageToken)}
            >
              {page.isPending ? "Loading…" : "Load more"}
            </button>
          )}
        </div>
      )}

      {revoking?.device && (
        <ConfirmDialog
          title="Revoke push registration"
          open
          onClose={() => {
            setRevoking(undefined);
            revoke.reset();
          }}
          onConfirm={() =>
            revoke.mutate({ projectId: project.id, pushDeviceId: revoking.device!.id })
          }
          confirmLabel="Revoke registration"
          busy={revoke.isPending}
          error={revoke.isError ? errorMessage(revoke.error) : undefined}
        >
          <p>
            The {pushTargetLabel(revoking.device.target)} registration of{" "}
            {revoking.userEmail || "this user"} (
            {revoking.device.metadata?.model || revoking.device.metadata?.platform || "device"})
            stops being served to your backend immediately — it will no longer
            receive pushes. The device re-registers on its next app launch.
          </p>
        </ConfirmDialog>
      )}
    </section>
  );
}

// PushSection edits the per-project push settings (milestone 20): the
// registry enable switch and the Web Push VAPID public key. Plain config
// with its own save — a full replacement via UpdatePushSettings, separate
// from the auth-policy form above. The VAPID private key never touches moth.
function PushSection({ project }: { project: Project }) {
  const current = useQuery(PushService.method.getPushSettings, { projectId: project.id });
  const [enabled, setEnabled] = useState(false);
  const [vapidKey, setVapidKey] = useState("");
  const [saved, setSaved] = useState(false);

  useEffect(() => {
    setEnabled(current.data?.settings?.enabled ?? false);
    setVapidKey(current.data?.settings?.webpushVapidPublicKey ?? "");
  }, [current.data]);

  const update = useMutation(PushService.method.updatePushSettings, {
    onSuccess: () => {
      invalidate(PushService.method.getPushSettings);
      setSaved(true);
      setTimeout(() => setSaved(false), 2000);
    },
  });

  const keyError = vapidKeyError(vapidKey.trim());

  return (
    <form
      className="stack-24"
      style={{ maxWidth: 640 }}
      onSubmit={(e) => {
        e.preventDefault();
        update.mutate({
          projectId: project.id,
          settings: { enabled, webpushVapidPublicKey: vapidKey.trim() },
        });
      }}
    >
      <section className="card card--pad stack-16">
        <h3 className="card__title">Push notifications</h3>
        <p className="caption">
          moth registers devices; your backend sends. The SDK registers each
          signed-in device's push credential here, and your server reads the
          registry through <span className="inline-code">moth.server.v1</span>{" "}
          to deliver via APNs, FCM or Web Push itself.
        </p>
        {current.isPending && <Loading />}
        {current.isError && <ErrorNote message={errorMessage(current.error)} />}
        {current.data && (
          <>
            <label className="check">
              <input
                type="checkbox"
                checked={enabled}
                onChange={(e) => setEnabled(e.target.checked)}
              />
              <span>
                Enable push registration
                <span className="caption" style={{ display: "block" }}>
                  Lets signed-in devices register their push credentials. Off =
                  new registrations are refused; existing ones are kept.
                </span>
              </span>
            </label>
            <Field
              label="Web Push VAPID public key"
              error={keyError}
              help={
                "Only needed for Web Push: the public half of your VAPID keypair " +
                "(base64url), delivered to browsers so they can subscribe. Keep the " +
                "private key in your sender — it never touches moth."
              }
            >
              <input
                className={keyError ? "input input--mono input--error" : "input input--mono"}
                value={vapidKey}
                onChange={(e) => setVapidKey(e.target.value)}
                placeholder="BPz3…"
                spellCheck={false}
                autoComplete="off"
              />
            </Field>
            <div className="row-12">
              <button
                type="submit"
                className="btn btn--primary"
                disabled={update.isPending || keyError !== ""}
              >
                {update.isPending ? "Saving…" : "Save push settings"}
              </button>
              {saved && <span className="caption text-success">Saved.</span>}
              {update.isError && (
                <span className="field__error">{errorMessage(update.error)}</span>
              )}
            </div>
          </>
        )}
      </section>
    </form>
  );
}