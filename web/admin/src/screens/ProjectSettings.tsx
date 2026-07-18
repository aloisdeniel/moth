import { useMutation, useQuery } from "@connectrpc/connect-query";
import { useEffect, useState } from "react";

import { errorMessage, invalidate } from "../api";
import { ErrorNote, Field, Loading, Status, StringListField } from "../components/ui";
import { ProfilePlatform, ProfileService } from "../gen/moth/admin/v1/profile_pb";
import type { Project } from "../gen/moth/admin/v1/project_pb";
import { ProjectService } from "../gen/moth/admin/v1/project_pb";
import { InstanceSettingsService, SmtpSource } from "../gen/moth/admin/v1/settings_pb";

// ProjectSettings edits the per-project auth policy (the milestone-02
// settings JSON, as a form).
export function ProjectSettings({ project }: { project: Project }) {
  const s = project.settings;
  const [name, setName] = useState(project.name);
  const [minLen, setMinLen] = useState(String(s?.passwordMinLength ?? 8));
  const [requireVerification, setRequireVerification] = useState(
    s?.requireEmailVerification ?? false,
  );
  const [publicSignup, setPublicSignup] = useState(s?.allowPublicSignup ?? true);
  const [enumSafe, setEnumSafe] = useState(s?.enumerationSafeSignup ?? false);
  const [accessTTL, setAccessTTL] = useState(String(s?.accessTokenTtlSeconds ?? 900));
  const [refreshTTL, setRefreshTTL] = useState(String(s?.refreshTokenTtlDays ?? 30));
  const [retention, setRetention] = useState(String(s?.analyticsRetentionDays || 90));
  const [rollupTz, setRollupTz] = useState(s?.rollupTimezone || "UTC");
  const [allowlist, setAllowlist] = useState<string[]>(s?.signupEmailAllowlist ?? []);
  const [blocklist, setBlocklist] = useState<string[]>(s?.signupEmailBlocklist ?? []);
  const [captchaUrl, setCaptchaUrl] = useState(s?.captchaVerifyUrl ?? "");
  const [saved, setSaved] = useState(false);

  const update = useMutation(ProjectService.method.updateProject, {
    onSuccess: () => {
      invalidate(ProjectService.method.getProject, ProjectService.method.listProjects);
      setSaved(true);
      setTimeout(() => setSaved(false), 2000);
    },
  });

  const instance = useQuery(InstanceSettingsService.method.getInstanceSettings);
  const smtpOn =
    instance.data !== undefined && instance.data.smtpSource !== SmtpSource.NONE;

  function save() {
    update.mutate({
      id: project.id,
      name,
      settings: {
        passwordMinLength: parseInt(minLen, 10) || 8,
        requireEmailVerification: requireVerification,
        allowPublicSignup: publicSignup,
        enumerationSafeSignup: enumSafe,
        accessTokenTtlSeconds: parseInt(accessTTL, 10) || 900,
        refreshTokenTtlDays: parseInt(refreshTTL, 10) || 30,
        analyticsRetentionDays: parseInt(retention, 10) || 90,
        rollupTimezone: rollupTz.trim(),
        // `settings` replaces wholesale under the update mask — carry the
        // provider config owned by the Providers tab through unchanged
        // (stored write-only secrets survive: empty means "keep").
        google: s?.google,
        apple: s?.apple,
        autoLinkVerifiedEmail: s?.autoLinkVerifiedEmail,
        redirectSchemes: s?.redirectSchemes ?? [],
        redirectOrigins: s?.redirectOrigins ?? [],
        signupEmailAllowlist: allowlist,
        signupEmailBlocklist: blocklist,
        captchaVerifyUrl: captchaUrl.trim(),
      },
      updateMask: { paths: ["name", "settings"] },
    });
  }

  return (
    <>
    <form
      className="stack-24"
      style={{ maxWidth: 640 }}
      onSubmit={(e) => {
        e.preventDefault();
        save();
      }}
    >
      <section className="card card--pad stack-16">
        <h3 className="card__title">Project</h3>
        <Field label="Name">
          <input
            className="input"
            value={name}
            onChange={(e) => setName(e.target.value)}
            maxLength={100}
          />
        </Field>
      </section>

      <section className="card card--pad stack-16">
        <h3 className="card__title">Sign-up</h3>
        <label className="check">
          <input
            type="checkbox"
            checked={publicSignup}
            onChange={(e) => setPublicSignup(e.target.checked)}
          />
          <span>
            Open sign-up
            <span className="caption" style={{ display: "block" }}>
              Anyone can create an account from the app. Off = invite-only:
              accounts are created here or through the server API.
            </span>
          </span>
        </label>
        <label className="check">
          <input
            type="checkbox"
            checked={enumSafe}
            onChange={(e) => setEnumSafe(e.target.checked)}
          />
          <span>
            Enumeration-safe sign-up
            <span className="caption" style={{ display: "block" }}>
              Signing up with an already-registered email returns OK and mails
              the owner, so responses never reveal whether an account exists.
            </span>
          </span>
        </label>
        <label className="check">
          <input
            type="checkbox"
            checked={requireVerification}
            onChange={(e) => setRequireVerification(e.target.checked)}
          />
          <span>
            Require email verification
            <span className="caption" style={{ display: "block" }}>
              Blocks sign-in until the address is verified via the emailed
              link.
            </span>
          </span>
        </label>
      </section>

      <section className="card card--pad stack-16">
        <h3 className="card__title">Abuse controls</h3>
        <StringListField
          label="Allowed email domains"
          values={allowlist}
          onChange={setAllowlist}
          placeholder="example.com"
          help={
            "When non-empty, sign-up is restricted to these email domains — " +
            'every other domain is rejected. Glob patterns allowed (e.g. "*.acme.io").'
          }
        />
        <StringListField
          label="Blocked email domains"
          values={blocklist}
          onChange={setBlocklist}
          placeholder="mailinator.com"
          help="Email domains rejected at sign-up, evaluated after the allowlist. Glob patterns allowed."
        />
        <Field
          label="CAPTCHA verification URL"
          help="Optional, off by default in v1. Documented hook: a verification endpoint moth POSTs the CAPTCHA token to. Stored but not yet enforced."
        >
          <input
            className="input input--mono"
            value={captchaUrl}
            onChange={(e) => setCaptchaUrl(e.target.value)}
            placeholder="https://example.com/captcha/verify"
            spellCheck={false}
          />
        </Field>
      </section>

      <section className="card card--pad stack-16">
        <h3 className="card__title">Passwords & tokens</h3>
        <Field label="Minimum password length">
          <input
            className="input"
            type="number"
            min={8}
            max={128}
            value={minLen}
            onChange={(e) => setMinLen(e.target.value)}
          />
        </Field>
        <Field label="Access token lifetime (seconds)" help="JWT expiry; default 900 (15 minutes).">
          <input
            className="input"
            type="number"
            min={60}
            max={86400}
            value={accessTTL}
            onChange={(e) => setAccessTTL(e.target.value)}
          />
        </Field>
        <Field
          label="Refresh token window (days)"
          help="Sliding window extended on each use; default 30."
        >
          <input
            className="input"
            type="number"
            min={1}
            max={365}
            value={refreshTTL}
            onChange={(e) => setRefreshTTL(e.target.value)}
          />
        </Field>
      </section>

      <section className="card card--pad stack-16">
        <h3 className="card__title">Analytics</h3>
        <Field
          label="Raw event retention (days)"
          help="Events older than this are pruned by the daily rollup; default 90."
        >
          <input
            className="input"
            type="number"
            min={1}
            max={366}
            value={retention}
            onChange={(e) => setRetention(e.target.value)}
          />
        </Field>
        <Field
          label="Rollup timezone"
          help='IANA name (e.g. "Europe/Paris") the daily stats are bucketed in; default UTC.'
        >
          <input
            className="input"
            value={rollupTz}
            onChange={(e) => setRollupTz(e.target.value)}
          />
        </Field>
      </section>

      <section className="card card--pad stack-12">
        <h3 className="card__title">Email sender</h3>
        {smtpOn ? (
          <Status tone="success">SMTP configured — verification and reset emails are delivered.</Status>
        ) : (
          <Status tone="warning">
            No SMTP configured — emails are logged to the server console. Set
            it up in instance settings.
          </Status>
        )}
      </section>

      <div className="row-12">
        <button type="submit" className="btn btn--primary" disabled={update.isPending}>
          {update.isPending ? "Saving…" : "Save settings"}
        </button>
        {saved && <span className="caption text-success">Saved.</span>}
        {update.isError && <span className="field__error">{errorMessage(update.error)}</span>}
      </div>
    </form>
    <ProfileSection project={project} />
    </>
  );
}

// ProfileSection edits the project's setup profile (milestone 22): the
// platforms and feature intents the wizard recorded. The setup tab and the
// overview checklist adapt to it; a pre-wizard project has none until one is
// saved here. UpdateProfile is a full replacement — platforms must stay
// non-empty, mirrored client-side.
function ProfileSection({ project }: { project: Project }) {
  const current = useQuery(ProfileService.method.getProfile, { projectId: project.id });
  const [platforms, setPlatforms] = useState<ProfilePlatform[]>([]);
  const [google, setGoogle] = useState(false);
  const [apple, setApple] = useState(false);
  const [sells, setSells] = useState(false);
  const [pushes, setPushes] = useState(false);
  const [dismissed, setDismissed] = useState(false);
  const [saved, setSaved] = useState(false);

  useEffect(() => {
    const p = current.data?.profile;
    setPlatforms(p?.platforms ?? []);
    setGoogle(p?.googleSignIn ?? false);
    setApple(p?.appleSignIn ?? false);
    setSells(p?.sellsSubscriptions ?? false);
    setPushes(p?.sendsPushes ?? false);
    setDismissed(p?.checklistDismissed ?? false);
  }, [current.data]);

  const update = useMutation(ProfileService.method.updateProfile, {
    onSuccess: () => {
      invalidate(
        ProfileService.method.getProfile,
        ProfileService.method.getProjectSetupStatus,
      );
      setSaved(true);
      setTimeout(() => setSaved(false), 2000);
    },
  });

  function toggle(p: ProfilePlatform) {
    setPlatforms((cur) => (cur.includes(p) ? cur.filter((x) => x !== p) : [...cur, p]));
  }

  const options: { value: ProfilePlatform; label: string }[] = [
    { value: ProfilePlatform.IOS, label: "iOS" },
    { value: ProfilePlatform.ANDROID, label: "Android" },
    { value: ProfilePlatform.WEB, label: "Web" },
  ];

  return (
    <form
      className="stack-24"
      style={{ maxWidth: 640 }}
      onSubmit={(e) => {
        e.preventDefault();
        update.mutate({
          projectId: project.id,
          profile: {
            platforms,
            googleSignIn: google,
            appleSignIn: apple,
            sellsSubscriptions: sells,
            sendsPushes: pushes,
            checklistDismissed: dismissed,
          },
        });
      }}
    >
      <section className="card card--pad stack-16">
        <h3 className="card__title">Project profile</h3>
        <p className="caption">
          What this app intends: platforms and features. The setup tab and the
          overview checklist adapt to it — it records intent only, never
          configuration.
        </p>
        {current.isPending && <Loading />}
        {current.isError && <ErrorNote message={errorMessage(current.error)} />}
        {current.data && (
          <>
            {!current.data.hasProfile && (
              <p className="caption">
                This project has no profile yet (it predates the creation
                wizard) — its setup tab shows everything. Save one to tailor
                it.
              </p>
            )}
            <div className="stack-8">
              <span className="field__label">Platforms</span>
              {options.map((o) => (
                <label className="check" key={o.value}>
                  <input
                    type="checkbox"
                    checked={platforms.includes(o.value)}
                    onChange={() => toggle(o.value)}
                  />
                  <span>{o.label}</span>
                </label>
              ))}
              {platforms.length === 0 && (
                <span className="field__error">Pick at least one platform.</span>
              )}
            </div>
            <div className="stack-8">
              <span className="field__label">Features</span>
              <label className="check">
                <input
                  type="checkbox"
                  checked={google}
                  onChange={(e) => setGoogle(e.target.checked)}
                />
                <span>Google sign-in</span>
              </label>
              <label className="check">
                <input
                  type="checkbox"
                  checked={apple}
                  onChange={(e) => setApple(e.target.checked)}
                />
                <span>Apple sign-in</span>
              </label>
              <label className="check">
                <input
                  type="checkbox"
                  checked={sells}
                  onChange={(e) => setSells(e.target.checked)}
                />
                <span>Sells subscriptions</span>
              </label>
              <label className="check">
                <input
                  type="checkbox"
                  checked={pushes}
                  onChange={(e) => setPushes(e.target.checked)}
                />
                <span>Sends push notifications</span>
              </label>
            </div>
            {current.data.hasProfile && dismissed && (
              <label className="check">
                <input
                  type="checkbox"
                  checked={dismissed}
                  onChange={(e) => setDismissed(e.target.checked)}
                />
                <span>
                  Checklist dismissed
                  <span className="caption" style={{ display: "block" }}>
                    Untick to bring the overview checklist card back.
                  </span>
                </span>
              </label>
            )}
            <div className="row-12">
              <button
                type="submit"
                className="btn btn--primary"
                disabled={update.isPending || platforms.length === 0}
              >
                {update.isPending
                  ? "Saving…"
                  : current.data.hasProfile
                    ? "Save profile"
                    : "Create profile"}
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
