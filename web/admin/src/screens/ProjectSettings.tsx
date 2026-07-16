import { useMutation, useQuery } from "@connectrpc/connect-query";
import { useState } from "react";

import { errorMessage, invalidate } from "../api";
import { Field, Status, StringListField } from "../components/ui";
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
        signupEmailAllowlist: allowlist,
        signupEmailBlocklist: blocklist,
        captchaVerifyUrl: captchaUrl.trim(),
      },
      updateMask: { paths: ["name", "settings"] },
    });
  }

  return (
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
  );
}
