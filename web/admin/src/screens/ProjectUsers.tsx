import { useMutation, useQuery } from "@connectrpc/connect-query";
import { useEffect, useState } from "react";

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
import type { Project } from "../gen/moth/admin/v1/project_pb";
import type { User } from "../gen/moth/admin/v1/user_pb";
import { UserService } from "../gen/moth/admin/v1/user_pb";
import { formatDate, formatDateTime } from "../lib/format";

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
