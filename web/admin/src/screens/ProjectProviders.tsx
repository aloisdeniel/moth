import { useMutation, useQuery } from "@connectrpc/connect-query";
import { useState } from "react";

import { errorMessage, invalidate } from "../api";
import { Badge, Field, KeyWell, PasswordInput } from "../components/ui";
import type { Project } from "../gen/moth/admin/v1/project_pb";
import { ProjectService } from "../gen/moth/admin/v1/project_pb";
import { InstanceSettingsService } from "../gen/moth/admin/v1/settings_pb";

// ProjectProviders configures social sign-in (Google & Apple) with the
// step-by-step console walkthroughs inline — the exact redirect URIs and
// where each pasted value comes from.
export function ProjectProviders({ project }: { project: Project }) {
  const s = project.settings;
  const g = s?.google;
  const a = s?.apple;

  // Google
  const [googleEnabled, setGoogleEnabled] = useState(g?.enabled ?? false);
  const [webClientId, setWebClientId] = useState(g?.webClientId ?? "");
  const [webClientSecret, setWebClientSecret] = useState("");
  const [iosClientId, setIosClientId] = useState(g?.iosClientId ?? "");
  const [androidClientId, setAndroidClientId] = useState(g?.androidClientId ?? "");

  // Apple
  const [appleEnabled, setAppleEnabled] = useState(a?.enabled ?? false);
  const [servicesId, setServicesId] = useState(a?.servicesId ?? "");
  const [teamId, setTeamId] = useState(a?.teamId ?? "");
  const [keyId, setKeyId] = useState(a?.keyId ?? "");
  const [privateKeyP8, setPrivateKeyP8] = useState("");
  const [bundleIds, setBundleIds] = useState<string[]>(a?.bundleIds ?? []);

  // Shared
  const [autoLink, setAutoLink] = useState(s?.autoLinkVerifiedEmail ?? true);
  const [redirectSchemes, setRedirectSchemes] = useState<string[]>(s?.redirectSchemes ?? []);
  const [saved, setSaved] = useState(false);

  const instance = useQuery(InstanceSettingsService.method.getInstanceSettings);
  const base = instance.data?.baseUrl ?? "";
  const host = base.replace(/^https?:\/\//, "");

  const update = useMutation(ProjectService.method.updateProject, {
    onSuccess: () => {
      invalidate(ProjectService.method.getProject, ProjectService.method.listProjects);
      // The secrets were consumed server-side; blank the fields so the
      // "stored" placeholders take over.
      setWebClientSecret("");
      setPrivateKeyP8("");
      setSaved(true);
      setTimeout(() => setSaved(false), 2000);
    },
  });

  function save() {
    update.mutate({
      id: project.id,
      settings: {
        // `settings` replaces wholesale under the update mask — carry the
        // auth-policy fields owned by the Settings tab through unchanged.
        passwordMinLength: s?.passwordMinLength ?? 8,
        requireEmailVerification: s?.requireEmailVerification ?? false,
        allowPublicSignup: s?.allowPublicSignup ?? true,
        enumerationSafeSignup: s?.enumerationSafeSignup ?? false,
        accessTokenTtlSeconds: s?.accessTokenTtlSeconds ?? 900,
        refreshTokenTtlDays: s?.refreshTokenTtlDays ?? 30,
        google: {
          enabled: googleEnabled,
          webClientId: webClientId.trim(),
          iosClientId: iosClientId.trim(),
          androidClientId: androidClientId.trim(),
          // Write-only: empty keeps the stored secret.
          webClientSecret,
        },
        apple: {
          enabled: appleEnabled,
          servicesId: servicesId.trim(),
          teamId: teamId.trim(),
          keyId: keyId.trim(),
          privateKeyP8: privateKeyP8.trim(),
          bundleIds,
        },
        autoLinkVerifiedEmail: autoLink,
        redirectSchemes,
      },
      updateMask: { paths: ["settings"] },
    });
  }

  return (
    <form
      className="stack-24"
      style={{ maxWidth: 720 }}
      onSubmit={(e) => {
        e.preventDefault();
        save();
      }}
    >
      <section className="card card--pad stack-16">
        <div className="page__header">
          <h3 className="card__title">Sign in with Google</h3>
          {g?.enabled ? <Badge tone="success">Enabled</Badge> : <Badge>Off</Badge>}
        </div>

        <label className="check">
          <input
            type="checkbox"
            checked={googleEnabled}
            onChange={(e) => setGoogleEnabled(e.target.checked)}
          />
          <span>
            Enable Sign in with Google
            <span className="caption" style={{ display: "block" }}>
              The client IDs below become the accepted audiences when moth
              verifies Google ID tokens.
            </span>
          </span>
        </label>

        <div className="stack-8">
          <p className="body-strong">1 · Configure the consent screen</p>
          <p className="caption">
            In{" "}
            <a href="https://console.cloud.google.com/apis/credentials" target="_blank" rel="noreferrer">
              Google Cloud Console → APIs &amp; Services → Credentials
            </a>
            , pick (or create) the Google Cloud project for this app. If you
            have not yet, open <strong>OAuth consent screen</strong> first and
            fill in the app name, support email and (for production) your
            domain — Google shows these on the sign-in sheet.
          </p>
        </div>

        <div className="stack-8">
          <p className="body-strong">2 · Create a Web application client</p>
          <p className="caption">
            <strong>Create credentials → OAuth client ID → Web application</strong>.
            Under <strong>Authorized redirect URIs</strong>, add exactly:
          </p>
          {base && <KeyWell value={`${base}/oauth/google/callback`} />}
          <p className="caption">
            Google shows a <strong>Client ID</strong> (ends in{" "}
            <span className="inline-code">.apps.googleusercontent.com</span>)
            and a <strong>Client secret</strong> — paste both here. The web
            client also serves as the server-side audience for the redirect
            fallback flow.
          </p>
        </div>
        <Field label="Web client ID">
          <input
            className="input input--mono"
            value={webClientId}
            onChange={(e) => setWebClientId(e.target.value)}
            placeholder="1234567890-abc.apps.googleusercontent.com"
            spellCheck={false}
          />
        </Field>
        <Field
          label="Web client secret"
          help={
            g?.hasWebClientSecret
              ? "A secret is stored (encrypted). Leave blank to keep it; paste a new value to replace it."
              : "Needed only for the web-redirect fallback flow. Stored encrypted, never shown again."
          }
        >
          <PasswordInput
            value={webClientSecret}
            onChange={setWebClientSecret}
            placeholder={g?.hasWebClientSecret ? "•••••••• (stored)" : "GOCSPX-…"}
            autoComplete="off"
          />
        </Field>

        <div className="stack-8">
          <p className="body-strong">3 · Create an iOS client</p>
          <p className="caption">
            <strong>Create credentials → OAuth client ID → iOS</strong>, enter
            your app's bundle ID. Paste the resulting client ID here — native
            Google sign-in on iOS mints tokens with this audience.
          </p>
        </div>
        <Field label="iOS client ID">
          <input
            className="input input--mono"
            value={iosClientId}
            onChange={(e) => setIosClientId(e.target.value)}
            placeholder="1234567890-ios.apps.googleusercontent.com"
            spellCheck={false}
          />
        </Field>

        <div className="stack-8">
          <p className="body-strong">4 · Create an Android client</p>
          <p className="caption">
            <strong>Create credentials → OAuth client ID → Android</strong>,
            enter the package name and the SHA-1 of your signing certificate
            (debug and release each need their own client;{" "}
            <span className="inline-code">./gradlew signingReport</span> prints
            the fingerprints). Note: Google's Android sign-in issues ID tokens
            with the <em>web</em> client ID as audience, so fill in step 2 even
            for Android-only apps.
          </p>
        </div>
        <Field label="Android client ID">
          <input
            className="input input--mono"
            value={androidClientId}
            onChange={(e) => setAndroidClientId(e.target.value)}
            placeholder="1234567890-android.apps.googleusercontent.com"
            spellCheck={false}
          />
        </Field>
      </section>

      <section className="card card--pad stack-16">
        <div className="page__header">
          <h3 className="card__title">Sign in with Apple</h3>
          {a?.enabled ? <Badge tone="success">Enabled</Badge> : <Badge>Off</Badge>}
        </div>

        <label className="check">
          <input
            type="checkbox"
            checked={appleEnabled}
            onChange={(e) => setAppleEnabled(e.target.checked)}
          />
          <span>
            Enable Sign in with Apple
            <span className="caption" style={{ display: "block" }}>
              Native flow on iOS; Android and web go through the redirect
              fallback, which needs the Services ID and key below.
            </span>
          </span>
        </label>

        <div className="stack-8">
          <p className="body-strong">1 · Add the capability to your App ID</p>
          <p className="caption">
            In{" "}
            <a
              href="https://developer.apple.com/account/resources/identifiers/list"
              target="_blank"
              rel="noreferrer"
            >
              Apple Developer → Certificates, Identifiers &amp; Profiles →
              Identifiers
            </a>
            , open your app's <strong>App ID</strong> and tick{" "}
            <strong>Sign in with Apple</strong> under Capabilities (then
            regenerate provisioning profiles). List every bundle ID that will
            sign in natively — tokens from the iOS app carry the bundle ID as
            audience:
          </p>
          <StringListField
            label="Bundle IDs"
            values={bundleIds}
            onChange={setBundleIds}
            placeholder="com.example.birdwatch"
          />
        </div>

        <div className="stack-8">
          <p className="body-strong">2 · Create a Services ID (web &amp; Android)</p>
          <p className="caption">
            Still under Identifiers, create a new identifier of type{" "}
            <strong>Services IDs</strong> — the convention is your bundle ID
            plus a suffix, e.g.{" "}
            <span className="inline-code">com.example.birdwatch.signin</span>.
            Enable <strong>Sign in with Apple</strong> on it, click{" "}
            <strong>Configure</strong>, pick your App ID as the primary, then
            register the domain and return URL:
          </p>
          {base && (
            <>
              <div className="stack-8">
                <span className="field__label">Domain</span>
                <KeyWell value={host} />
              </div>
              <div className="stack-8">
                <span className="field__label">Return URL</span>
                <KeyWell value={`${base}/oauth/apple/callback`} />
              </div>
            </>
          )}
          <p className="caption">
            Apple requires the domain to be publicly reachable over HTTPS for
            production use.
          </p>
        </div>
        <Field label="Services ID">
          <input
            className="input input--mono"
            value={servicesId}
            onChange={(e) => setServicesId(e.target.value)}
            placeholder="com.example.birdwatch.signin"
            spellCheck={false}
          />
        </Field>

        <div className="stack-8">
          <p className="body-strong">3 · Create a private key</p>
          <p className="caption">
            Go to{" "}
            <a
              href="https://developer.apple.com/account/resources/authkeys/list"
              target="_blank"
              rel="noreferrer"
            >
              Keys
            </a>
            , create a key with <strong>Sign in with Apple</strong> enabled and
            your App ID as its primary. Download the{" "}
            <span className="inline-code">.p8</span> file —{" "}
            <strong>Apple lets you download it exactly once</strong> — and note
            the <strong>Key ID</strong> shown next to it. Your{" "}
            <strong>Team ID</strong> is in the top-right of the developer
            account (or under Membership). moth signs Apple's required client
            secret from this key and rotates it automatically.
          </p>
        </div>
        <div className="row-16" style={{ alignItems: "flex-start" }}>
          <div style={{ flex: 1 }}>
            <Field label="Team ID">
              <input
                className="input input--mono"
                value={teamId}
                onChange={(e) => setTeamId(e.target.value)}
                placeholder="AB12CD34EF"
                spellCheck={false}
              />
            </Field>
          </div>
          <div style={{ flex: 1 }}>
            <Field label="Key ID">
              <input
                className="input input--mono"
                value={keyId}
                onChange={(e) => setKeyId(e.target.value)}
                placeholder="XYZ987WV65"
                spellCheck={false}
              />
            </Field>
          </div>
        </div>
        <Field
          label="Private key (.p8)"
          help={
            a?.hasPrivateKey
              ? "Leave blank to keep the stored key; paste a new one to replace it."
              : "Paste the full contents of the downloaded .p8 file. Stored encrypted, never shown again."
          }
        >
          <textarea
            className="input input--mono"
            rows={6}
            value={privateKeyP8}
            onChange={(e) => setPrivateKeyP8(e.target.value)}
            placeholder={
              a?.hasPrivateKey
                ? "Private key stored (encrypted)"
                : "-----BEGIN PRIVATE KEY-----\n…\n-----END PRIVATE KEY-----"
            }
            spellCheck={false}
          />
        </Field>
        {a?.hasPrivateKey && <Badge tone="success">Private key stored</Badge>}
      </section>

      <section className="card card--pad stack-16">
        <h3 className="card__title">Linking &amp; redirects</h3>
        <label className="check">
          <input
            type="checkbox"
            checked={autoLink}
            onChange={(e) => setAutoLink(e.target.checked)}
          />
          <span>
            Auto-link on verified email
            <span className="caption" style={{ display: "block" }}>
              When a social sign-in arrives whose email the provider asserts
              as verified and it matches an existing account, the identity is
              linked to that account (one user, several sign-in methods).
              Unverified emails never link — they always create a separate
              account, so nobody can take over an account by claiming its
              email at a provider.
            </span>
          </span>
        </label>
        <StringListField
          label="Redirect schemes"
          values={redirectSchemes}
          onChange={setRedirectSchemes}
          placeholder={project.slug}
          help={`Custom URL schemes the web-redirect fallback may send users back to, e.g. "${project.slug}" for ${project.slug}://auth?code=…. Open-redirect protection: callbacks only redirect to schemes on this list.`}
        />
      </section>

      <div className="row-12">
        <button type="submit" className="btn btn--primary" disabled={update.isPending}>
          {update.isPending ? "Saving…" : "Save providers"}
        </button>
        {saved && <span className="caption text-success">Saved.</span>}
        {update.isError && <span className="field__error">{errorMessage(update.error)}</span>}
      </div>
    </form>
  );
}

// StringListField is a small multi-value editor: current values as removable
// rows plus an input that adds on Enter (intercepted so the surrounding form
// does not submit).
function StringListField({
  label,
  values,
  onChange,
  placeholder,
  help,
}: {
  label: string;
  values: string[];
  onChange: (values: string[]) => void;
  placeholder?: string;
  help?: string;
}) {
  const [draft, setDraft] = useState("");

  function add() {
    const v = draft.trim();
    if (v === "" || values.includes(v)) return;
    onChange([...values, v]);
    setDraft("");
  }

  return (
    <div className="field">
      <span className="field__label">{label}</span>
      {values.map((v) => (
        <div key={v} className="keywell">
          <span className="keywell__value">{v}</span>
          <button
            type="button"
            className="btn btn--ghost btn--compact"
            onClick={() => onChange(values.filter((x) => x !== v))}
          >
            Remove
          </button>
        </div>
      ))}
      <div className="row-8">
        <input
          className="input input--mono"
          style={{ flex: 1 }}
          aria-label={label}
          value={draft}
          placeholder={placeholder}
          spellCheck={false}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") {
              e.preventDefault();
              add();
            }
          }}
        />
        <button type="button" className="btn btn--secondary btn--compact" onClick={add}>
          Add
        </button>
      </div>
      {help && <span className="field__help">{help}</span>}
    </div>
  );
}
