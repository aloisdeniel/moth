import { useMutation, useQuery } from "@connectrpc/connect-query";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import { useEffect, useState } from "react";
import { Link } from "react-router";

import { errorMessage, invalidate } from "../api";
import {
  Badge,
  ConfirmDialog,
  Dialog,
  ErrorNote,
  Field,
  Loading,
  PasswordInput,
} from "../components/ui";
import { EntitlementService } from "../gen/moth/admin/v1/entitlement_pb";
import { ProductService } from "../gen/moth/admin/v1/product_pb";
import type { Project } from "../gen/moth/admin/v1/project_pb";
import { PushService, type PushDevice } from "../gen/moth/admin/v1/push_pb";
import { SubscriptionService, type Grant } from "../gen/moth/admin/v1/subscription_pb";
import type { User } from "../gen/moth/admin/v1/user_pb";
import { UserService } from "../gen/moth/admin/v1/user_pb";
import {
  formatPrice,
  statusGrantsAccess,
  storeLabel,
  subscriptionStatusMeta,
} from "../lib/billing";
import { formatDate, formatDateTime, formatRelative } from "../lib/format";
import { pushPermissionMeta, pushRevokeReasonLabel, pushTargetLabel } from "../lib/push";

const PAGE_SIZE = 25;

function invalidateUsers() {
  invalidate(UserService.method.listUsers, UserService.method.getUser);
}

export function ProjectUsers({ project }: { project: Project }) {
  const [query, setQuery] = useState("");
  const [search, setSearch] = useState("");
  const [pageToken, setPageToken] = useState("");
  // Tokens of the pages we came from, so "Previous" works.
  const [trail, setTrail] = useState<string[]>([]);
  const [adding, setAdding] = useState(false);
  const [selected, setSelected] = useState<string>();

  // Debounce the search box into the RPC query.
  useEffect(() => {
    const t = setTimeout(() => {
      setSearch(query.trim());
      setPageToken("");
      setTrail([]);
    }, 250);
    return () => clearTimeout(t);
  }, [query]);

  const users = useQuery(UserService.method.listUsers, {
    projectId: project.id,
    pageSize: PAGE_SIZE,
    pageToken,
    query: search,
  });

  return (
    <div className="stack-16">
      <div className="page__header">
        <input
          className="input"
          style={{ maxWidth: 320 }}
          placeholder="Search email or name"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        />
        <button type="button" className="btn btn--primary" onClick={() => setAdding(true)}>
          Add user
        </button>
      </div>

      {users.isPending && <Loading />}
      {users.isError && <ErrorNote message={errorMessage(users.error)} />}

      {users.data &&
        (users.data.users.length === 0 ? (
          <div className="card empty">
            <p className="body-strong">{search ? "No users match" : "No users yet"}</p>
            <p className="caption">
              {search
                ? "Try another search."
                : "Users appear here when they sign up in the app — or add one yourself."}
            </p>
          </div>
        ) : (
          <div className="card" style={{ overflowX: "auto" }}>
            <table className="table">
              <thead>
                <tr>
                  <th>Email</th>
                  <th>Name</th>
                  <th>Providers</th>
                  <th>Status</th>
                  <th>Last login</th>
                  <th>Created</th>
                </tr>
              </thead>
              <tbody>
                {users.data.users.map((u) => (
                  <tr key={u.id} onClick={() => setSelected(u.id)} style={{ cursor: "pointer" }}>
                    <td className="mono">{u.email}</td>
                    <td>{u.displayName || <span className="text-tertiary">—</span>}</td>
                    <td>
                      <span className="row-8">
                        {u.providers.map((p) => (
                          <Badge key={p}>{p}</Badge>
                        ))}
                      </span>
                    </td>
                    <td>
                      <span className="row-8">
                        {u.disabled ? (
                          <Badge tone="danger">Disabled</Badge>
                        ) : u.emailVerified ? (
                          <Badge tone="success">Verified</Badge>
                        ) : (
                          <Badge tone="warning">Pending</Badge>
                        )}
                      </span>
                    </td>
                    <td className="mono">{formatDate(u.lastLoginTime)}</td>
                    <td className="mono">{formatDate(u.createTime)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ))}

      {users.data && (
        <div className="row-12">
          <span className="caption tabular">
            {users.data.totalSize.toString()} {users.data.totalSize === 1n ? "user" : "users"}
          </span>
          <span className="topbar__spacer" />
          <button
            type="button"
            className="btn btn--secondary btn--compact"
            disabled={trail.length === 0}
            onClick={() => {
              const prev = [...trail];
              const back = prev.pop() ?? "";
              setTrail(prev);
              setPageToken(back);
            }}
          >
            Previous
          </button>
          <button
            type="button"
            className="btn btn--secondary btn--compact"
            disabled={users.data.nextPageToken === ""}
            onClick={() => {
              setTrail([...trail, pageToken]);
              setPageToken(users.data.nextPageToken);
            }}
          >
            Next
          </button>
        </div>
      )}

      <AddUserDialog project={project} open={adding} onClose={() => setAdding(false)} />
      {selected && (
        <UserDrawer project={project} userId={selected} onClose={() => setSelected(undefined)} />
      )}
    </div>
  );
}

function AddUserDialog({
  project,
  open,
  onClose,
}: {
  project: Project;
  open: boolean;
  onClose: () => void;
}) {
  const [email, setEmail] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [mode, setMode] = useState<"password" | "invite">("invite");
  const [password, setPassword] = useState("");
  const [verified, setVerified] = useState(false);

  const create = useMutation(UserService.method.createUser, {
    onSuccess: () => {
      invalidateUsers();
      invalidate(); // user counts on project cards
      close();
    },
  });

  function close() {
    setEmail("");
    setDisplayName("");
    setPassword("");
    setMode("invite");
    setVerified(false);
    create.reset();
    onClose();
  }

  return (
    <Dialog title="Add user" open={open} onClose={close}>
      <form
        className="stack-16"
        onSubmit={(e) => {
          e.preventDefault();
          create.mutate({
            projectId: project.id,
            email,
            displayName,
            password: mode === "password" ? password : "",
            sendInvite: mode === "invite",
            emailVerified: verified,
          });
        }}
      >
        <Field label="Email">
          <input
            className="input"
            type="email"
            autoCapitalize="none"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            autoFocus
          />
        </Field>
        <Field label="Display name (optional)">
          <input
            className="input"
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
          />
        </Field>
        <div className="stack-8">
          <span className="field__label">Credentials</span>
          <label className="check">
            <input
              type="radio"
              name="mode"
              checked={mode === "invite"}
              onChange={() => setMode("invite")}
            />
            <span>
              Send a set-password invite email
              <span className="caption" style={{ display: "block" }}>
                The user chooses their own password through an emailed link
                (valid 72 hours).
              </span>
            </span>
          </label>
          <label className="check">
            <input
              type="radio"
              name="mode"
              checked={mode === "password"}
              onChange={() => setMode("password")}
            />
            <span>Set a password now</span>
          </label>
          {mode === "password" && (
            <PasswordInput value={password} onChange={setPassword} autoComplete="new-password" />
          )}
        </div>
        <label className="check">
          <input
            type="checkbox"
            checked={verified}
            onChange={(e) => setVerified(e.target.checked)}
          />
          <span>Mark email as verified</span>
        </label>
        {create.isError && <p className="field__error">{errorMessage(create.error)}</p>}
        <div className="dialog__actions">
          <button type="button" className="btn btn--secondary" onClick={close}>
            Cancel
          </button>
          <button
            type="submit"
            className="btn btn--primary"
            disabled={create.isPending || email === "" || (mode === "password" && password === "")}
          >
            {create.isPending ? "Adding…" : mode === "invite" ? "Add & send invite" : "Add user"}
          </button>
        </div>
      </form>
    </Dialog>
  );
}

// UserDrawer is the user detail: identities, sessions, custom claims
// editor and the destructive row actions.
function UserDrawer({
  project,
  userId,
  onClose,
}: {
  project: Project;
  userId: string;
  onClose: () => void;
}) {
  const detail = useQuery(UserService.method.getUser, { projectId: project.id, userId });
  const [confirm, setConfirm] = useState<"delete" | "revoke" | "reset">();
  const [notice, setNotice] = useState("");

  const onDone = (message: string) => ({
    onSuccess: () => {
      invalidateUsers();
      setConfirm(undefined);
      setNotice(message);
    },
  });
  const disable = useMutation(UserService.method.disableUser, onDone("User disabled."));
  const enable = useMutation(UserService.method.enableUser, onDone("User enabled."));
  const revoke = useMutation(UserService.method.revokeUserSessions, onDone("All sessions revoked."));
  const sendReset = useMutation(UserService.method.sendPasswordReset, onDone("Reset email sent."));
  const del = useMutation(UserService.method.deleteUser, {
    onSuccess: () => {
      invalidateUsers();
      invalidate();
      onClose();
    },
  });

  const u: User | undefined = detail.data?.user;

  return (
    <>
      <div className="drawer-overlay" onMouseDown={onClose} />
      <aside className="drawer">
        {detail.isPending && <Loading />}
        {detail.isError && <ErrorNote message={errorMessage(detail.error)} />}
        {u && (
          <>
            <div className="stack-8">
              <div className="page__header">
                <h3>{u.email}</h3>
                <button type="button" className="btn btn--ghost btn--compact" onClick={onClose}>
                  Close
                </button>
              </div>
              <div className="row-8">
                {u.disabled ? (
                  <Badge tone="danger">Disabled</Badge>
                ) : u.emailVerified ? (
                  <Badge tone="success">Verified</Badge>
                ) : (
                  <Badge tone="warning">Pending</Badge>
                )}
                {u.providers.map((p) => (
                  <Badge key={p}>{p}</Badge>
                ))}
              </div>
              <p className="caption">
                <span className="mono">{u.id}</span>
              </p>
              <p className="caption">
                Created <span className="mono">{formatDateTime(u.createTime)}</span> · last login{" "}
                <span className="mono">{formatDateTime(u.lastLoginTime)}</span>
              </p>
              {notice && <p className="caption text-success">{notice}</p>}
            </div>

            <div className="stack-8">
              <span className="field__label">Identities</span>
              {(detail.data?.identities.length ?? 0) === 0 ? (
                <p className="caption">No linked identities.</p>
              ) : (
                <div className="stack-8">
                  {detail.data?.identities.map((i) => (
                    <div key={i.provider} className="keywell">
                      <Badge>{i.provider}</Badge>
                      <span className="keywell__value">
                        {i.email || u.email} · linked {formatDate(i.createTime)}
                      </span>
                    </div>
                  ))}
                </div>
              )}
            </div>

            <ClaimsEditor project={project} user={u} />

            <SubscriptionSection project={project} userId={userId} />

            <div className="stack-8">
              <span className="field__label">Active sessions</span>
              {(detail.data?.sessions.length ?? 0) === 0 ? (
                <p className="caption">No active sessions.</p>
              ) : (
                <div className="stack-8">
                  {detail.data?.sessions.map((s) => (
                    <div key={s.id} className="keywell">
                      <span className="keywell__value">
                        {s.deviceInfo || "unknown device"} · since {formatDate(s.createTime)}
                      </span>
                    </div>
                  ))}
                </div>
              )}
            </div>

            <PushDevicesSection project={project} userId={userId} />

            <div className="stack-8">
              <span className="field__label">Actions</span>
              <div className="row-8" style={{ flexWrap: "wrap" }}>
                {u.disabled ? (
                  <button
                    type="button"
                    className="btn btn--secondary btn--compact"
                    disabled={enable.isPending}
                    onClick={() => enable.mutate({ projectId: project.id, userId })}
                  >
                    Enable
                  </button>
                ) : (
                  <button
                    type="button"
                    className="btn btn--secondary btn--compact"
                    disabled={disable.isPending}
                    onClick={() => disable.mutate({ projectId: project.id, userId })}
                  >
                    Disable
                  </button>
                )}
                {u.providers.includes("password") && (
                  <button
                    type="button"
                    className="btn btn--secondary btn--compact"
                    onClick={() => setConfirm("reset")}
                  >
                    Send password reset
                  </button>
                )}
                <button
                  type="button"
                  className="btn btn--secondary btn--compact"
                  onClick={() => setConfirm("revoke")}
                >
                  Revoke sessions
                </button>
                <button
                  type="button"
                  className="btn btn--danger btn--compact"
                  onClick={() => setConfirm("delete")}
                >
                  Delete
                </button>
              </div>
              {(disable.isError || enable.isError) && (
                <p className="field__error">
                  {errorMessage(disable.isError ? disable.error : enable.error)}
                </p>
              )}
            </div>

            <ConfirmDialog
              title="Send password reset"
              open={confirm === "reset"}
              onClose={() => setConfirm(undefined)}
              onConfirm={() => sendReset.mutate({ projectId: project.id, userId })}
              confirmLabel="Send email"
              busy={sendReset.isPending}
              error={sendReset.isError ? errorMessage(sendReset.error) : undefined}
            >
              <p>
                Emails <span className="mono">{u.email}</span> a password-reset
                link, as if they had used "forgot password" themselves.
              </p>
            </ConfirmDialog>

            <ConfirmDialog
              title="Revoke all sessions"
              open={confirm === "revoke"}
              onClose={() => setConfirm(undefined)}
              onConfirm={() => revoke.mutate({ projectId: project.id, userId })}
              confirmLabel="Revoke sessions"
              busy={revoke.isPending}
              error={revoke.isError ? errorMessage(revoke.error) : undefined}
            >
              <p>
                Signs <span className="mono">{u.email}</span> out of every
                device. Outstanding access tokens die at their expiry (at most
                the token lifetime).
              </p>
            </ConfirmDialog>

            <ConfirmDialog
              title="Delete user"
              open={confirm === "delete"}
              onClose={() => setConfirm(undefined)}
              onConfirm={() => del.mutate({ projectId: project.id, userId })}
              confirmLabel="Delete user"
              confirmText={u.email}
              busy={del.isPending}
              error={del.isError ? errorMessage(del.error) : undefined}
            >
              <p>
                Permanently deletes <span className="mono">{u.email}</span>{" "}
                with their identities, sessions and pending email tokens. This
                cannot be undone.
              </p>
            </ConfirmDialog>
          </>
        )}
      </aside>
    </>
  );
}

// ClaimsEditor validates the JSON locally before allowing a save.
function ClaimsEditor({ project, user }: { project: Project; user: User }) {
  const [text, setText] = useState(user.customClaims || "{}");
  const [saved, setSaved] = useState(false);
  useEffect(() => {
    setText(user.customClaims || "{}");
  }, [user.id, user.customClaims]);

  let parseError = "";
  try {
    const v: unknown = JSON.parse(text);
    if (typeof v !== "object" || v === null || Array.isArray(v)) {
      parseError = "Claims must be a JSON object.";
    }
  } catch {
    parseError = "Not valid JSON.";
  }

  const update = useMutation(UserService.method.updateUser, {
    onSuccess: () => {
      invalidateUsers();
      setSaved(true);
      setTimeout(() => setSaved(false), 2000);
    },
  });

  return (
    <div className="stack-8">
      <span className="field__label">Custom claims (embedded in the JWT)</span>
      <textarea
        className={parseError ? "input input--mono input--error" : "input input--mono"}
        rows={4}
        value={text}
        spellCheck={false}
        onChange={(e) => setText(e.target.value)}
      />
      {parseError && text !== "" && <span className="field__error">{parseError}</span>}
      {update.isError && <span className="field__error">{errorMessage(update.error)}</span>}
      <div className="row-8">
        <button
          type="button"
          className="btn btn--secondary btn--compact"
          disabled={parseError !== "" || update.isPending}
          onClick={() =>
            update.mutate({
              projectId: project.id,
              userId: user.id,
              user: { customClaims: text },
              updateMask: { paths: ["custom_claims"] },
            })
          }
        >
          {update.isPending ? "Saving…" : "Save claims"}
        </button>
        {saved && <span className="caption text-success">Saved — applies to new tokens.</span>}
      </div>
    </div>
  );
}

// PushDevicesSection lists the user's push registrations (active and
// revoked — revocation is auditable, not a delete) with the admin revoke
// action. Metadata only: the admin proto never carries the push token.
function PushDevicesSection({ project, userId }: { project: Project; userId: string }) {
  const devices = useQuery(PushService.method.listUserPushDevices, {
    projectId: project.id,
    userId,
  });
  const [revoking, setRevoking] = useState<PushDevice>();

  const revoke = useMutation(PushService.method.revokePushDevice, {
    onSuccess: () => {
      invalidate(PushService.method.listUserPushDevices);
      setRevoking(undefined);
    },
  });

  function metadataLine(d: PushDevice): string {
    const m = d.metadata;
    const parts = [
      [m?.platform, m?.model].filter(Boolean).join(" "),
      m?.osVersion ? `OS ${m.osVersion}` : "",
      m?.appVersion ? `app ${m.appVersion}` : "",
      m?.locale ?? "",
    ].filter(Boolean);
    return parts.length === 0 ? "no device metadata" : parts.join(" · ");
  }

  return (
    <div className="stack-8">
      <span className="field__label">Push devices</span>
      <p className="caption">
        This user's registrations, including revoked ones. The project-wide
        list lives on the <Link to={`/projects/${project.id}/push`}>Push tab</Link>.
      </p>
      {devices.isPending && <Loading />}
      {devices.isError && <ErrorNote message={errorMessage(devices.error)} />}
      {devices.data &&
        (devices.data.devices.length === 0 ? (
          <p className="caption">No push devices registered.</p>
        ) : (
          <div className="stack-8">
            {devices.data.devices.map((d) => {
              const perm = pushPermissionMeta(d.permission);
              const revoked = d.revokeTime !== undefined;
              const reason = pushRevokeReasonLabel(d.revokeReason);
              return (
                <div key={d.id} className="keywell" style={{ alignItems: "flex-start" }}>
                  <div className="keywell__value stack-8" style={{ gap: 2 }}>
                    <span className="row-8" style={{ flexWrap: "wrap" }}>
                      <Badge>{pushTargetLabel(d.target)}</Badge>
                      <Badge tone={revoked ? "neutral" : perm.tone}>{perm.label}</Badge>
                      {revoked && (
                        <Badge tone="danger">Revoked{reason && ` · ${reason}`}</Badge>
                      )}
                    </span>
                    <span className="caption">{metadataLine(d)}</span>
                    <span className="caption">
                      last seen {formatRelative(d.lastSeenTime)}
                      {revoked && ` · revoked ${formatDate(d.revokeTime)}`}
                    </span>
                  </div>
                  {!revoked && (
                    <button
                      type="button"
                      className="btn btn--ghost btn--compact"
                      onClick={() => setRevoking(d)}
                    >
                      Revoke
                    </button>
                  )}
                </div>
              );
            })}
          </div>
        ))}

      {revoking && (
        <ConfirmDialog
          title="Revoke push registration"
          open
          onClose={() => {
            setRevoking(undefined);
            revoke.reset();
          }}
          onConfirm={() => revoke.mutate({ projectId: project.id, pushDeviceId: revoking.id })}
          confirmLabel="Revoke registration"
          busy={revoke.isPending}
          error={revoke.isError ? errorMessage(revoke.error) : undefined}
        >
          <p>
            The {pushTargetLabel(revoking.target)} registration of this{" "}
            {revoking.metadata?.model || revoking.metadata?.platform || "device"} stops
            being served to your backend immediately — it will no longer receive
            pushes. The device re-registers on its next app launch.
          </p>
        </ConfirmDialog>
      )}
    </div>
  );
}

function grantIsActive(g: Grant, now: number): boolean {
  if (g.revokeTime) return false;
  if (g.expireTime && Number(g.expireTime.seconds) * 1000 <= now) return false;
  return true;
}

// SubscriptionSection shows the user's store subscriptions, derived active
// entitlements and operator grants, plus the comp/revoke actions.
function SubscriptionSection({ project, userId }: { project: Project; userId: string }) {
  const subs = useQuery(SubscriptionService.method.listUserSubscriptions, {
    projectId: project.id,
    userId,
  });
  const ents = useQuery(EntitlementService.method.listEntitlements, { projectId: project.id });
  const prods = useQuery(ProductService.method.listProducts, { projectId: project.id });
  const [granting, setGranting] = useState(false);

  const entName = (id: string) => ents.data?.entitlements.find((e) => e.id === id)?.identifier ?? id;
  const prodName = (id: string) => {
    const p = prods.data?.products.find((x) => x.id === id);
    return p ? p.displayName || p.identifier : "";
  };

  const revoke = useMutation(SubscriptionService.method.revokeGrant, {
    onSuccess: () => invalidate(SubscriptionService.method.listUserSubscriptions),
  });

  // Derive the active entitlement set client-side (plan/11 matrix): granting
  // store statuses contribute their product's entitlements, plus every active
  // grant.
  const now = Date.now();
  const active = new Set<string>();
  for (const s of subs.data?.subscriptions ?? []) {
    if (!statusGrantsAccess(s.status) || !s.productId) continue;
    const p = prods.data?.products.find((x) => x.id === s.productId);
    for (const id of p?.entitlementIds ?? []) active.add(id);
  }
  for (const g of subs.data?.grants ?? []) {
    if (grantIsActive(g, now)) active.add(g.entitlementId);
  }

  return (
    <div className="stack-8">
      <div className="page__header">
        <span className="field__label">Subscriptions &amp; entitlements</span>
        <button
          type="button"
          className="btn btn--secondary btn--compact"
          onClick={() => setGranting(true)}
          disabled={(ents.data?.entitlements.length ?? 0) === 0}
        >
          Grant entitlement
        </button>
      </div>

      {(subs.isPending || ents.isPending) && <Loading />}
      {subs.isError && <ErrorNote message={errorMessage(subs.error)} />}

      {subs.data && (
        <>
          <div className="row-8" style={{ flexWrap: "wrap" }}>
            {active.size === 0 ? (
              <Badge>none (free)</Badge>
            ) : (
              [...active].map((id) => (
                <Badge key={id} tone="success">
                  {entName(id)}
                </Badge>
              ))
            )}
          </div>

          {subs.data.subscriptions.length === 0 ? (
            <p className="caption">No store subscriptions.</p>
          ) : (
            <div className="stack-8">
              {subs.data.subscriptions.map((s) => {
                const meta = subscriptionStatusMeta(s.status);
                const p = prods.data?.products.find((x) => x.id === s.productId);
                return (
                  <div key={s.id} className="keywell" style={{ alignItems: "flex-start" }}>
                    <div className="keywell__value stack-8" style={{ gap: 2 }}>
                      <span className="row-8" style={{ flexWrap: "wrap" }}>
                        <strong>{prodName(s.productId) || "Unmapped product"}</strong>
                        <Badge>{storeLabel(s.store)}</Badge>
                        <Badge tone={meta.tone}>{meta.label}</Badge>
                        {s.environment === "sandbox" && <Badge tone="warning">sandbox</Badge>}
                      </span>
                      <span className="caption">
                        {p && `${formatPrice(p.priceAmountMicros, p.currency)} · `}
                        renews {formatDate(s.currentPeriodEnd)}
                        {s.autoRenew ? "" : " · auto-renew off"}
                      </span>
                    </div>
                  </div>
                );
              })}
            </div>
          )}

          <span className="field__label">Operator grants</span>
          {subs.data.grants.length === 0 ? (
            <p className="caption">No promo or comp grants.</p>
          ) : (
            <div className="stack-8">
              {subs.data.grants.map((g) => {
                const activeGrant = grantIsActive(g, now);
                return (
                  <div key={g.id} className="keywell" style={{ alignItems: "flex-start" }}>
                    <div className="keywell__value stack-8" style={{ gap: 2 }}>
                      <span className="row-8" style={{ flexWrap: "wrap" }}>
                        <Badge tone={activeGrant ? "success" : "neutral"}>
                          {entName(g.entitlementId)}
                        </Badge>
                        {g.revokeTime ? (
                          <span className="caption">revoked {formatDate(g.revokeTime)}</span>
                        ) : g.expireTime ? (
                          <span className="caption">
                            {activeGrant ? "expires" : "expired"} {formatDate(g.expireTime)}
                          </span>
                        ) : (
                          <span className="caption">no expiry</span>
                        )}
                      </span>
                      <span className="caption">
                        {g.reason || "no reason"}
                        {g.grantedBy && ` · by ${g.grantedBy}`}
                      </span>
                    </div>
                    {activeGrant && (
                      <button
                        type="button"
                        className="btn btn--ghost btn--compact"
                        disabled={revoke.isPending}
                        onClick={() => revoke.mutate({ projectId: project.id, grantId: g.id })}
                      >
                        Revoke
                      </button>
                    )}
                  </div>
                );
              })}
            </div>
          )}
          {revoke.isError && <p className="field__error">{errorMessage(revoke.error)}</p>}
        </>
      )}

      {granting && (
        <GrantDialog
          project={project}
          userId={userId}
          entitlements={(ents.data?.entitlements ?? []).map((e) => ({
            id: e.id,
            label: `${e.displayName || e.identifier} (${e.identifier})`,
          }))}
          onClose={() => setGranting(false)}
        />
      )}
    </div>
  );
}

function GrantDialog({
  project,
  userId,
  entitlements,
  onClose,
}: {
  project: Project;
  userId: string;
  entitlements: { id: string; label: string }[];
  onClose: () => void;
}) {
  const [entitlementId, setEntitlementId] = useState(entitlements[0]?.id ?? "");
  const [expiry, setExpiry] = useState("");
  const [reason, setReason] = useState("");

  const grant = useMutation(SubscriptionService.method.grantEntitlement, {
    onSuccess: () => {
      invalidate(SubscriptionService.method.listUserSubscriptions);
      onClose();
    },
  });

  return (
    <Dialog title="Grant entitlement" open onClose={onClose}>
      <form
        className="stack-16"
        onSubmit={(e) => {
          e.preventDefault();
          grant.mutate({
            projectId: project.id,
            userId,
            entitlementId,
            reason: reason.trim(),
            expireTime:
              expiry.trim() === "" ? undefined : timestampFromDate(new Date(expiry)),
          });
        }}
      >
        <p className="caption">
          A comp/promo grant unlocks an entitlement independent of any store
          purchase (grant it to a reviewer, extend a grace period). It is
          audit-logged.
        </p>
        <Field label="Entitlement">
          <select
            className="select"
            value={entitlementId}
            onChange={(e) => setEntitlementId(e.target.value)}
          >
            {entitlements.map((e) => (
              <option key={e.id} value={e.id}>
                {e.label}
              </option>
            ))}
          </select>
        </Field>
        <Field label="Expiry (optional)" help="Leave blank for an indefinite grant.">
          <input
            className="input"
            type="date"
            value={expiry}
            onChange={(e) => setExpiry(e.target.value)}
          />
        </Field>
        <Field label="Reason">
          <input
            className="input"
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            placeholder="App Store reviewer comp"
          />
        </Field>
        {grant.isError && <p className="field__error">{errorMessage(grant.error)}</p>}
        <div className="dialog__actions">
          <button type="button" className="btn btn--secondary" onClick={onClose}>
            Cancel
          </button>
          <button
            type="submit"
            className="btn btn--primary"
            disabled={grant.isPending || entitlementId === ""}
          >
            {grant.isPending ? "Granting…" : "Grant"}
          </button>
        </div>
      </form>
    </Dialog>
  );
}
