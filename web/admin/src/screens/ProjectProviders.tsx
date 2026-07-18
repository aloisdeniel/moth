import { useMutation, useQuery } from "@connectrpc/connect-query";
import { useState } from "react";

import { errorMessage, invalidate } from "../api";
import {
  AppleCredentialFields,
  GoogleCredentialFields,
  type AppleDraft,
  type GoogleDraft,
} from "../components/providerFields";
import { Badge, KeyWell, StringListField } from "../components/ui";
import type { Project } from "../gen/moth/admin/v1/project_pb";
import { ProjectService } from "../gen/moth/admin/v1/project_pb";
import { InstanceSettingsService } from "../gen/moth/admin/v1/settings_pb";

// ProjectProviders configures social sign-in (Google & Apple) with the
// step-by-step console walkthroughs inline — the exact redirect URIs and
// where each pasted value comes from. The credential fields themselves are
// shared with the creation wizard (components/providerFields).
export function ProjectProviders({ project }: { project: Project }) {
  const s = project.settings;
  const g = s?.google;
  const a = s?.apple;

  // Google
  const [googleEnabled, setGoogleEnabled] = useState(g?.enabled ?? false);
  const [google, setGoogle] = useState<GoogleDraft>({
    webClientId: g?.webClientId ?? "",
    webClientSecret: "",
    iosClientId: g?.iosClientId ?? "",
    androidClientId: g?.androidClientId ?? "",
  });

  // Apple
  const [appleEnabled, setAppleEnabled] = useState(a?.enabled ?? false);
  const [apple, setApple] = useState<AppleDraft>({
    servicesId: a?.servicesId ?? "",
    teamId: a?.teamId ?? "",
    keyId: a?.keyId ?? "",
    privateKeyP8: "",
    bundleIds: a?.bundleIds ?? [],
  });

  // Shared
  const [autoLink, setAutoLink] = useState(s?.autoLinkVerifiedEmail ?? true);
  const [redirectSchemes, setRedirectSchemes] = useState<string[]>(s?.redirectSchemes ?? []);
  const [redirectOrigins, setRedirectOrigins] = useState<string[]>(s?.redirectOrigins ?? []);
  const [saved, setSaved] = useState(false);

  const instance = useQuery(InstanceSettingsService.method.getInstanceSettings);
  const base = instance.data?.baseUrl ?? "";
  const host = base.replace(/^https?:\/\//, "");

  const update = useMutation(ProjectService.method.updateProject, {
    onSuccess: () => {
      invalidate(ProjectService.method.getProject, ProjectService.method.listProjects);
      // The secrets were consumed server-side; blank the fields so the
      // "stored" placeholders take over.
      setGoogle((d) => ({ ...d, webClientSecret: "" }));
      setApple((d) => ({ ...d, privateKeyP8: "" }));
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
          webClientId: google.webClientId.trim(),
          iosClientId: google.iosClientId.trim(),
          androidClientId: google.androidClientId.trim(),
          // Write-only: empty keeps the stored secret.
          webClientSecret: google.webClientSecret,
        },
        apple: {
          enabled: appleEnabled,
          servicesId: apple.servicesId.trim(),
          teamId: apple.teamId.trim(),
          keyId: apple.keyId.trim(),
          privateKeyP8: apple.privateKeyP8.trim(),
          bundleIds: apple.bundleIds,
        },
        autoLinkVerifiedEmail: autoLink,
        redirectSchemes,
        redirectOrigins,
        // Owned by the Settings tab; carried through so a providers save
        // does not wipe the abuse-control lists.
        signupEmailAllowlist: s?.signupEmailAllowlist ?? [],
        signupEmailBlocklist: s?.signupEmailBlocklist ?? [],
        captchaVerifyUrl: s?.captchaVerifyUrl ?? "",
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

        <GoogleCredentialFields
          draft={google}
          onChange={setGoogle}
          hasStoredSecret={g?.hasWebClientSecret ?? false}
          beforeWeb={
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
          }
          beforeIos={
            <div className="stack-8">
              <p className="body-strong">3 · Create an iOS client</p>
              <p className="caption">
                <strong>Create credentials → OAuth client ID → iOS</strong>, enter
                your app's bundle ID. Paste the resulting client ID here — native
                Google sign-in on iOS mints tokens with this audience.
              </p>
            </div>
          }
          beforeAndroid={
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
          }
        />
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

        <AppleCredentialFields
          draft={apple}
          onChange={setApple}
          hasStoredKey={a?.hasPrivateKey ?? false}
          beforeBundleIds={
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
            </div>
          }
          beforeServicesId={
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
          }
          beforeKey={
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
          }
        />
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
        <StringListField
          label="Redirect origins (web)"
          values={redirectOrigins}
          onChange={setRedirectOrigins}
          placeholder="https://app.example.com"
          help="https://app.example.com — where the browser SDK may receive the sign-in code; exact origin match (scheme, host and port), any path. Bare origins only; http:// is accepted for localhost during development."
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
