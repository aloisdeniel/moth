import { timestampDate } from "@bufbuild/protobuf/wkt";
import { useMutation, useQuery } from "@connectrpc/connect-query";
import { useEffect, useState } from "react";

import { errorMessage, invalidate } from "../api";
import {
  Badge,
  ConfirmDialog,
  Dialog,
  ErrorNote,
  Field,
  KeyWell,
  Loading,
  PasswordInput,
  Status,
} from "../components/ui";
import type { PersonalAccessToken } from "../gen/moth/admin/v1/account_pb";
import { AdminAccountService } from "../gen/moth/admin/v1/account_pb";
import { InstanceSettingsService, SmtpSource } from "../gen/moth/admin/v1/settings_pb";
import { formatDate, formatRelative } from "../lib/format";

export function InstanceSettings() {
  return (
    <main className="page page--narrow">
      <h1>Instance settings</h1>
      <AdminsCard />
      <ChangePasswordCard />
      <PatsCard />
      <SmtpCard />
    </main>
  );
}

// ---------- Admin accounts ----------

function AdminsCard() {
  const admins = useQuery(AdminAccountService.method.listAdmins);
  const invites = useQuery(AdminAccountService.method.listAdminInvites);
  const [inviting, setInviting] = useState(false);
  const [revokeId, setRevokeId] = useState<string>();

  const revoke = useMutation(AdminAccountService.method.revokeAdminInvite, {
    onSuccess: () => {
      invalidate(AdminAccountService.method.listAdminInvites);
      setRevokeId(undefined);
    },
  });

  return (
    <section className="card card--pad stack-16">
      <div className="page__header">
        <h3 className="card__title">Admin accounts</h3>
        <button
          type="button"
          className="btn btn--secondary btn--compact"
          onClick={() => setInviting(true)}
        >
          Invite admin
        </button>
      </div>

      {admins.isPending && <Loading />}
      {admins.isError && <ErrorNote message={errorMessage(admins.error)} />}
      {admins.data && (
        <table className="table">
          <thead>
            <tr>
              <th>Email</th>
              <th>Since</th>
            </tr>
          </thead>
          <tbody>
            {admins.data.admins.map((a) => (
              <tr key={a.id}>
                <td className="mono">{a.email}</td>
                <td className="mono">{formatDate(a.createTime)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      {invites.data && invites.data.invites.length > 0 && (
        <div className="stack-8">
          <span className="field__label">Pending invites</span>
          {invites.data.invites.map((inv) => (
            <div key={inv.id} className="keywell">
              <span className="keywell__value">
                {inv.email} · expires {formatDate(inv.expireTime)}
              </span>
              <button
                type="button"
                className="btn btn--ghost btn--compact"
                onClick={() => setRevokeId(inv.id)}
              >
                Revoke
              </button>
            </div>
          ))}
        </div>
      )}

      <InviteAdminDialog open={inviting} onClose={() => setInviting(false)} />
      <ConfirmDialog
        title="Revoke invite"
        open={revokeId !== undefined}
        onClose={() => setRevokeId(undefined)}
        onConfirm={() => revokeId && revoke.mutate({ id: revokeId })}
        confirmLabel="Revoke invite"
        busy={revoke.isPending}
        error={revoke.isError ? errorMessage(revoke.error) : undefined}
      >
        <p>The invite link stops working immediately.</p>
      </ConfirmDialog>
    </section>
  );
}

function InviteAdminDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const [email, setEmail] = useState("");
  const [result, setResult] = useState<{ url: string; emailed: boolean }>();

  const invite = useMutation(AdminAccountService.method.inviteAdmin, {
    onSuccess: (resp) => {
      invalidate(AdminAccountService.method.listAdminInvites);
      setResult({ url: resp.inviteUrl, emailed: resp.emailed });
    },
  });

  function close() {
    setEmail("");
    setResult(undefined);
    invite.reset();
    onClose();
  }

  if (result) {
    return (
      <Dialog title="Invite created" open={open} onClose={close}>
        <div className="stack-16">
          <p className="caption">
            {result.emailed
              ? "The invite was emailed. You can also share the link directly:"
              : "No SMTP is configured, so share this link with the new admin yourself:"}
          </p>
          <KeyWell value={result.url} />
          <p className="caption">
            Anyone opening this link can claim the account. It expires in 72
            hours.
          </p>
          <div className="dialog__actions">
            <button type="button" className="btn btn--primary" onClick={close}>
              Done
            </button>
          </div>
        </div>
      </Dialog>
    );
  }

  return (
    <Dialog title="Invite admin" open={open} onClose={close}>
      <form
        className="stack-16"
        onSubmit={(e) => {
          e.preventDefault();
          invite.mutate({ email });
        }}
      >
        <p className="caption">
          Admins have full control over every project on this instance.
        </p>
        <Field label="Email" error={invite.isError ? errorMessage(invite.error) : undefined}>
          <input
            className="input"
            type="email"
            autoCapitalize="none"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            autoFocus
          />
        </Field>
        <div className="dialog__actions">
          <button type="button" className="btn btn--secondary" onClick={close}>
            Cancel
          </button>
          <button
            type="submit"
            className="btn btn--primary"
            disabled={invite.isPending || email === ""}
          >
            {invite.isPending ? "Creating…" : "Create invite"}
          </button>
        </div>
      </form>
    </Dialog>
  );
}

// ---------- Change password ----------

function ChangePasswordCard() {
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [done, setDone] = useState(false);

  const change = useMutation(AdminAccountService.method.changePassword, {
    onSuccess: () => {
      setCurrent("");
      setNext("");
      setDone(true);
      setTimeout(() => setDone(false), 3000);
    },
  });

  return (
    <section className="card card--pad stack-16">
      <h3 className="card__title">Change password</h3>
      <form
        className="stack-16"
        onSubmit={(e) => {
          e.preventDefault();
          change.mutate({ currentPassword: current, newPassword: next });
        }}
      >
        <Field label="Current password">
          <PasswordInput value={current} onChange={setCurrent} autoComplete="current-password" />
        </Field>
        <Field
          label="New password"
          help="At least 8 characters. Your other browser sessions are signed out."
          error={change.isError ? errorMessage(change.error) : undefined}
        >
          <PasswordInput value={next} onChange={setNext} autoComplete="new-password" />
        </Field>
        <div className="row-12">
          <button
            type="submit"
            className="btn btn--primary"
            disabled={change.isPending || current === "" || next === ""}
          >
            {change.isPending ? "Changing…" : "Change password"}
          </button>
          {done && <span className="caption text-success">Password changed.</span>}
        </div>
      </form>
    </section>
  );
}

// ---------- Personal access tokens ----------

// PatStateBadge renders the lifecycle state of a token: revoked wins over
// expired wins over active.
function PatStateBadge({ token }: { token: PersonalAccessToken }) {
  if (token.revokeTime) return <Badge tone="danger">Revoked</Badge>;
  if (token.expireTime && timestampDate(token.expireTime).getTime() <= Date.now()) {
    return <Badge tone="warning">Expired</Badge>;
  }
  return <Badge tone="success">Active</Badge>;
}

function PatsCard() {
  const tokens = useQuery(AdminAccountService.method.listPersonalAccessTokens);
  const instance = useQuery(InstanceSettingsService.method.getInstanceSettings);
  const [creating, setCreating] = useState(false);
  const [revokeTarget, setRevokeTarget] = useState<PersonalAccessToken>();

  const revoke = useMutation(AdminAccountService.method.revokePersonalAccessToken, {
    onSuccess: () => {
      invalidate(AdminAccountService.method.listPersonalAccessTokens);
      setRevokeTarget(undefined);
    },
  });

  return (
    <section className="card card--pad stack-16">
      <div className="page__header">
        <h3 className="card__title">Personal access tokens</h3>
        <button
          type="button"
          className="btn btn--secondary btn--compact"
          onClick={() => setCreating(true)}
        >
          Create token
        </button>
      </div>
      <p className="caption">
        Tokens authenticate the <span className="inline-code">moth</span> CLI
        (and scripts) as your admin account. They carry the same permissions
        as your session.
      </p>

      {tokens.isPending && <Loading />}
      {tokens.isError && <ErrorNote message={errorMessage(tokens.error)} />}
      {tokens.data && tokens.data.tokens.length > 0 && (
        <table className="table">
          <thead>
            <tr>
              <th>Name</th>
              <th>Created</th>
              <th>Last used</th>
              <th>Expires</th>
              <th>State</th>
              <th />
            </tr>
          </thead>
          <tbody>
            {tokens.data.tokens.map((t) => (
              <tr key={t.id}>
                <td>{t.name}</td>
                <td className="mono">{formatDate(t.createTime)}</td>
                <td className="mono">{formatRelative(t.lastUsedTime)}</td>
                <td className="mono">{t.expireTime ? formatDate(t.expireTime) : "never"}</td>
                <td>
                  <PatStateBadge token={t} />
                </td>
                <td style={{ textAlign: "right" }}>
                  {!t.revokeTime && (
                    <button
                      type="button"
                      className="btn btn--ghost btn--compact"
                      onClick={() => setRevokeTarget(t)}
                    >
                      Revoke
                    </button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
      {tokens.data && tokens.data.tokens.length === 0 && (
        <p className="caption">No tokens yet.</p>
      )}

      {instance.data && (
        <div className="stack-8" style={{ borderTop: "1px solid var(--border)", paddingTop: 16 }}>
          <span className="field__label">Use the CLI</span>
          <p className="caption">
            Sign in from a terminal — the command asks for a personal access
            token and remembers this instance:
          </p>
          <KeyWell value={`moth login ${instance.data.baseUrl}`} />
        </div>
      )}

      <CreatePatDialog open={creating} onClose={() => setCreating(false)} />
      <ConfirmDialog
        title="Revoke token"
        open={revokeTarget !== undefined}
        onClose={() => setRevokeTarget(undefined)}
        onConfirm={() => revokeTarget && revoke.mutate({ id: revokeTarget.id })}
        confirmLabel="Revoke token"
        busy={revoke.isPending}
        error={revoke.isError ? errorMessage(revoke.error) : undefined}
      >
        <p>
          “{revokeTarget?.name}” stops authenticating immediately. Anything
          still using it (a CLI context, a script) has to log in again.
        </p>
      </ConfirmDialog>
    </section>
  );
}

function CreatePatDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const [name, setName] = useState("");
  const [expiresInDays, setExpiresInDays] = useState("90");
  const [token, setToken] = useState<string>();

  const create = useMutation(AdminAccountService.method.createPersonalAccessToken, {
    onSuccess: (resp) => {
      invalidate(AdminAccountService.method.listPersonalAccessTokens);
      setToken(resp.token);
    },
  });

  function close() {
    setName("");
    setExpiresInDays("90");
    setToken(undefined);
    create.reset();
    onClose();
  }

  if (token) {
    return (
      <Dialog title="Token created" open={open} onClose={close}>
        <div className="stack-16">
          <KeyWell value={token} secret />
          <p className="caption">
            You won't see this token again — copy it now. Paste it when{" "}
            <span className="inline-code">moth login</span> asks for it.
          </p>
          <div className="dialog__actions">
            <button type="button" className="btn btn--primary" onClick={close}>
              Done
            </button>
          </div>
        </div>
      </Dialog>
    );
  }

  return (
    <Dialog title="Create token" open={open} onClose={close}>
      <form
        className="stack-16"
        onSubmit={(e) => {
          e.preventDefault();
          create.mutate({ name, expiresInDays: parseInt(expiresInDays, 10) });
        }}
      >
        <Field
          label="Name"
          help="What will use it — “laptop”, “ci”."
          error={create.isError ? errorMessage(create.error) : undefined}
        >
          <input
            className="input"
            value={name}
            onChange={(e) => setName(e.target.value)}
            autoFocus
          />
        </Field>
        <Field label="Expires">
          <select
            className="input"
            value={expiresInDays}
            onChange={(e) => setExpiresInDays(e.target.value)}
          >
            <option value="30">In 30 days</option>
            <option value="90">In 90 days</option>
            <option value="365">In 1 year</option>
            <option value="0">Never</option>
          </select>
        </Field>
        <div className="dialog__actions">
          <button type="button" className="btn btn--secondary" onClick={close}>
            Cancel
          </button>
          <button
            type="submit"
            className="btn btn--primary"
            disabled={create.isPending || name === ""}
          >
            {create.isPending ? "Creating…" : "Create token"}
          </button>
        </div>
      </form>
    </Dialog>
  );
}

// ---------- SMTP ----------

function SmtpCard() {
  const settings = useQuery(InstanceSettingsService.method.getInstanceSettings);
  const [host, setHost] = useState("");
  const [port, setPort] = useState("587");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [from, setFrom] = useState("");
  const [testTo, setTestTo] = useState("");
  const [saved, setSaved] = useState(false);
  const [testResult, setTestResult] = useState("");

  useEffect(() => {
    const smtp = settings.data?.smtp;
    if (smtp) {
      setHost(smtp.host);
      setPort(String(smtp.port || 587));
      setUsername(smtp.username);
      setFrom(smtp.from);
    }
  }, [settings.data]);

  const update = useMutation(InstanceSettingsService.method.updateSmtpSettings, {
    onSuccess: () => {
      invalidate(InstanceSettingsService.method.getInstanceSettings);
      setPassword("");
      setSaved(true);
      setTimeout(() => setSaved(false), 2000);
    },
  });
  const test = useMutation(InstanceSettingsService.method.sendTestEmail, {
    onSuccess: () => setTestResult("ok"),
    onError: () => setTestResult("error"),
  });

  const source = settings.data?.smtpSource;

  return (
    <section className="card card--pad stack-16">
      <h3 className="card__title">Outgoing email (SMTP)</h3>
      {settings.isPending && <Loading />}
      {settings.data && (
        <>
          {source === SmtpSource.NONE && (
            <Status tone="warning">
              Not configured — emails are logged to the server console.
            </Status>
          )}
          {source === SmtpSource.CONFIG && (
            <Status tone="success">
              Configured from the server config file. Saving here overrides it.
            </Status>
          )}
          {source === SmtpSource.DATABASE && (
            <Status tone="success">Configured from the admin console.</Status>
          )}

          <form
            className="stack-16"
            onSubmit={(e) => {
              e.preventDefault();
              update.mutate({
                smtp: {
                  host,
                  port: parseInt(port, 10) || 587,
                  username,
                  password,
                  from,
                },
              });
            }}
          >
            <div className="row-12" style={{ alignItems: "flex-start" }}>
              <div style={{ flex: 3 }}>
                <Field label="Host">
                  <input
                    className="input"
                    value={host}
                    onChange={(e) => setHost(e.target.value)}
                    placeholder="smtp.example.com"
                  />
                </Field>
              </div>
              <div style={{ flex: 1 }}>
                <Field label="Port">
                  <input
                    className="input"
                    type="number"
                    min={1}
                    max={65535}
                    value={port}
                    onChange={(e) => setPort(e.target.value)}
                  />
                </Field>
              </div>
            </div>
            <Field label="Username (optional)">
              <input
                className="input"
                autoCapitalize="none"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
              />
            </Field>
            <Field
              label="Password"
              help={
                settings.data.smtpHasPassword
                  ? "Leave blank to keep the stored password."
                  : undefined
              }
            >
              <PasswordInput value={password} onChange={setPassword} autoComplete="off" />
            </Field>
            <Field label="Sender address (from)">
              <input
                className="input"
                type="email"
                autoCapitalize="none"
                value={from}
                onChange={(e) => setFrom(e.target.value)}
                placeholder="auth@example.com"
              />
            </Field>
            {update.isError && <p className="field__error">{errorMessage(update.error)}</p>}
            <div className="row-12">
              <button type="submit" className="btn btn--primary" disabled={update.isPending}>
                {update.isPending ? "Saving…" : "Save SMTP settings"}
              </button>
              {source === SmtpSource.DATABASE && (
                <button
                  type="button"
                  className="btn btn--ghost"
                  disabled={update.isPending}
                  onClick={() => update.mutate({ smtp: { host: "" } })}
                >
                  Clear override
                </button>
              )}
              {saved && <span className="caption text-success">Saved.</span>}
            </div>
          </form>

          <div className="stack-8" style={{ borderTop: "1px solid var(--border)", paddingTop: 16 }}>
            <span className="field__label">Send a test email</span>
            <div className="row-8">
              <input
                className="input"
                style={{ flex: 1 }}
                type="email"
                autoCapitalize="none"
                placeholder="you@example.com"
                value={testTo}
                onChange={(e) => setTestTo(e.target.value)}
              />
              <button
                type="button"
                className="btn btn--secondary"
                disabled={test.isPending || testTo === ""}
                onClick={() => {
                  setTestResult("");
                  test.mutate({ to: testTo });
                }}
              >
                {test.isPending ? "Sending…" : "Send test email"}
              </button>
            </div>
            {testResult === "ok" && (
              <span className="caption text-success">
                Sent. {source === SmtpSource.NONE ? "Check the server console." : "Check the inbox."}
              </span>
            )}
            {testResult === "error" && test.error && (
              <span className="field__error">{errorMessage(test.error)}</span>
            )}
          </div>
        </>
      )}
    </section>
  );
}
