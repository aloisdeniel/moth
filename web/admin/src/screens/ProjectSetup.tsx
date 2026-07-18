import { useQuery } from "@connectrpc/connect-query";
import { useEffect, useState } from "react";

import { errorMessage } from "../api";
import { CodeBlock, ErrorNote, KeyWell, Loading } from "../components/ui";
import { ProfilePlatform, ProfileService } from "../gen/moth/admin/v1/profile_pb";
import type { Project } from "../gen/moth/admin/v1/project_pb";
import { ProjectService } from "../gen/moth/admin/v1/project_pb";
import { InstanceSettingsService } from "../gen/moth/admin/v1/settings_pb";

// useSdkVersion fetches the moth_auth version this instance actually serves
// from its own pub listing, so the pubspec snippet always resolves.
// undefined = loading, null = failed.
function useSdkVersion(): string | null | undefined {
  const [version, setVersion] = useState<string | null | undefined>(undefined);
  useEffect(() => {
    let cancelled = false;
    fetch("/pub/api/packages/moth_auth")
      .then((resp) => (resp.ok ? resp.json() : Promise.reject(resp.status)))
      .then((listing: { latest: { version: string } }) => {
        if (!cancelled) setVersion(listing.latest.version);
      })
      .catch(() => {
        if (!cancelled) setVersion(null);
      });
    return () => {
      cancelled = true;
    };
  }, []);
  return version;
}

// useNpmSdkVersion fetches the @moth/react version served by this instance's
// embedded npm registry (the packument's dist-tags.latest). null on failure —
// e.g. a server built before /npm existed — so the React section degrades to
// an unpinned install instead of breaking the page.
function useNpmSdkVersion(): string | null | undefined {
  const [version, setVersion] = useState<string | null | undefined>(undefined);
  useEffect(() => {
    let cancelled = false;
    fetch("/npm/@moth/react")
      .then((resp) => (resp.ok ? resp.json() : Promise.reject(resp.status)))
      .then((packument: { "dist-tags"?: { latest?: string } }) => {
        if (!cancelled) setVersion(packument["dist-tags"]?.latest ?? null);
      })
      .catch(() => {
        if (!cancelled) setVersion(null);
      });
    return () => {
      cancelled = true;
    };
  }, []);
  return version;
}

// ProjectSetup renders copy-paste instructions with this project's real
// values — the setup page is the product. With a setup profile (milestone
// 22) it adapts to the wizard's answers: the SDK picker is limited to the
// chosen platforms and the monetization/push sections only render when the
// profile intends them. Without a profile (pre-wizard project) everything
// shows, exactly as before.
export function ProjectSetup({ project }: { project: Project }) {
  const instance = useQuery(InstanceSettingsService.method.getInstanceSettings);
  const signing = useQuery(ProjectService.method.getSigningKey, { projectId: project.id });
  const profileQ = useQuery(ProfileService.method.getProfile, { projectId: project.id });
  const sdkVersion = useSdkVersion();
  const npmVersion = useNpmSdkVersion();
  const [platformChoice, setPlatformChoice] = useState<"flutter" | "react" | null>(null);

  if (
    instance.isPending ||
    signing.isPending ||
    profileQ.isPending ||
    sdkVersion === undefined ||
    npmVersion === undefined
  )
    return <Loading />;
  if (instance.isError) return <ErrorNote message={errorMessage(instance.error)} />;
  if (signing.isError) return <ErrorNote message={errorMessage(signing.error)} />;
  if (profileQ.isError) return <ErrorNote message={errorMessage(profileQ.error)} />;
  if (sdkVersion === null) return <ErrorNote message="could not load the served SDK version from /pub" />;

  // The profile filters, never invents: no profile → all platforms and
  // sections, today's behavior.
  const profile = profileQ.data.hasProfile ? profileQ.data.profile : undefined;
  const profilePlatforms = profile?.platforms ?? [];
  const hasNative =
    profile === undefined ||
    profilePlatforms.includes(ProfilePlatform.IOS) ||
    profilePlatforms.includes(ProfilePlatform.ANDROID);
  const hasWeb = profile === undefined || profilePlatforms.includes(ProfilePlatform.WEB);
  const showMonetization = profile === undefined || profile.sellsSubscriptions;
  const showPush = profile === undefined || profile.sendsPushes;
  const platform: "flutter" | "react" =
    platformChoice !== null && (platformChoice === "flutter" ? hasNative : hasWeb)
      ? platformChoice
      : hasNative
        ? "flutter"
        : "react";
  const monetizeSection = 6;
  const pushSection = showMonetization ? 7 : 6;

  const base = instance.data.baseUrl;
  const host = base.replace(/^https?:\/\//, "");
  const grpcHost = host.includes(":") ? host : `${host}:443`;
  const jwks = signing.data.jwksUrl;
  const issuer = signing.data.issuer;
  const audience = signing.data.audience;

  // Pre-release versions (dev builds serve 0.0.0-dev.*) must be pinned
  // exactly — Dart version ranges never match pre-releases; releases get a
  // caret so patch updates of the same major resolve.
  const versionConstraint = sdkVersion.includes("-") ? sdkVersion : `^${sdkVersion}`;

  const pubspec = `dependencies:
  moth_auth:
    hosted: ${base}/pub
    version: ${versionConstraint}`;

  // npm ranges never match pre-releases either — same pin-or-caret rule as
  // the pubspec above. npmVersion === null (no /npm on this server yet)
  // degrades to an unpinned install.
  const npmConstraint =
    npmVersion === null ? null : npmVersion.includes("-") ? npmVersion : `^${npmVersion}`;

  const npmrc = `@moth:registry=${base}/npm`;

  const npmInstall =
    npmConstraint === null
      ? "npm install @moth/react"
      : `npm install @moth/react@"${npmConstraint}"`;

  const mainTsx = `import { createRoot } from 'react-dom/client'
import { MothProvider, MothLoginScreen } from '@moth/react'

import App from './App'

createRoot(document.getElementById('root')!).render(
  <MothProvider
    config={{
      endpoint: '${base}',
      publishableKey: '${project.publishableKey}',
      projectSlug: '${project.slug}',
    }}
    signedOut={<MothLoginScreen />}
  >
    <App />
  </MothProvider>,
)`;

  const mainDart = `import 'package:flutter/material.dart';
import 'package:moth_auth/moth_auth.dart';

void main() {
  runApp(
    MothApp(
      config: MothConfig(
        endpoint: Uri.parse('${base}'),
        publishableKey: '${project.publishableKey}',
      ),
      // Signed out -> the SDK's built-in MothLoginScreen; signed in -> child.
      child: const MyApp(),
    ),
  );
}

class MyApp extends StatelessWidget {
  const MyApp({super.key});

  @override
  Widget build(BuildContext context) {
    final user = MothScope.of(context).user;
    return MaterialApp(
      home: Scaffold(
        body: Center(child: Text('Signed in as \${user?.email}')),
      ),
    );
  }
}`;

  const nodeJose = `import { createRemoteJWKSet, jwtVerify } from "jose";

const jwks = createRemoteJWKSet(
  new URL("${jwks}"),
);

export async function verifyMothToken(token) {
  const { payload } = await jwtVerify(token, jwks, {
    issuer: "${issuer}",
    audience: "${audience}",
  });
  return payload; // payload.sub is the moth user id
}`;

  const goJwx = `import (
	"context"

	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"
)

var cache, _ = jwk.NewCache(context.Background(), nil)

func init() {
	cache.Register(context.Background(), "${jwks}")
}

func verifyMothToken(ctx context.Context, raw string) (jwt.Token, error) {
	keys, err := cache.Lookup(ctx, "${jwks}")
	if err != nil {
		return nil, err
	}
	return jwt.Parse([]byte(raw),
		jwt.WithKeySet(keys),
		jwt.WithIssuer("${issuer}"),
		jwt.WithAudience("${audience}"),
	)
}`;

  const dartVerify = `// On a Dart backend (e.g. shelf / serverpod), with package:jose
import 'package:jose/jose.dart';

Future<Map<String, dynamic>> verifyMothToken(String token) async {
  final jwt = await JsonWebToken.decodeAndVerify(
    token,
    JsonWebKeyStore()
      ..addKeySetUrl(Uri.parse('${jwks}')),
  );
  final claims = jwt.claims;
  if (claims.issuer != Uri.parse('${issuer}') ||
      !(claims.audience?.contains('${audience}') ?? false)) {
    throw Exception('token was not minted for this app');
  }
  return claims.toJson();
}`;

  const grpcurl = `grpcurl \\
  -H 'x-moth-key: sk_YOUR_SECRET_KEY' \\
  -d '{"access_token": "eyJ..."}' \\
  ${grpcHost} \\
  moth.server.v1.TokenService/IntrospectToken`;

  const cliSetup = `moth login ${base}
moth setup google --project ${project.slug}
moth setup apple --project ${project.slug}
moth doctor --project ${project.slug}`;

  const pubspecBilling = `dependencies:
  moth_billing:
    hosted: ${base}/pub
    version: ${versionConstraint}`;

  const paywallDart = `import 'package:moth_billing/moth_billing.dart';

MothApp(
  config: MothConfig(
    endpoint: Uri.parse('${base}'),
    publishableKey: '${project.publishableKey}',
  ),
  // moth's native billing (StoreKit 2 / Play Billing) runs the store
  // purchase; moth validates the receipt server-side and derives
  // entitlements. Custom stores implement MothBillingAdapter instead.
  billingAdapter: MothStoreBilling(),
  requiresEntitlement: 'pro', // free users see the paywall
  paywall: const MothPaywallScreen(),
  child: const MyApp(),
);`;

  const paywallReact = `<MothProvider
  config={{
    endpoint: '${base}',
    publishableKey: '${project.publishableKey}',
    projectSlug: '${project.slug}',
  }}
  signedOut={<MothLoginScreen />}
>
  {/* Free users see the themed paywall; purchase redirects to Stripe
      Checkout and the gate unlocks when the entitlement lands. */}
  <MothGate entitlement="pro" fallback={<MothPaywallScreen />}>
    <ProFeatures />
  </MothGate>
</MothProvider>`;

  const cliBilling = `moth setup billing --project ${project.slug}`;

  const pubspecPush = `dependencies:
  moth_push:
    hosted: ${base}/pub
    version: ${versionConstraint}`;

  const pushDart = `import 'package:moth_push/moth_push.dart';

MothApp(
  config: MothConfig(
    endpoint: Uri.parse('${base}'),
    publishableKey: '${project.publishableKey}',
  ),
  // moth's native push registration (APNs on iOS, FCM on Android): while a
  // user is signed in the SDK keeps this project's device registry current.
  // The OS permission prompt only appears when your app calls
  // MothScope.of(context).requestPushPermission().
  pushAdapter: MothNativePush(),
  child: const MyApp(),
);`;

  const pushReact = `import { useMothPush } from '@moth/react'

// Once at startup — the app owns its service worker (display and click
// handling); the SDK only manages the subscription. See the SDK README
// for a minimal sw.js.
navigator.serviceWorker.register('/sw.js')

function PushToggle() {
  const { status, subscribe, unsubscribe } = useMothPush()
  if (status === 'unavailable' || status === 'unsupported') return null
  if (status === 'denied') return <p>Notifications are blocked in the browser.</p>
  return status === 'subscribed' ? (
    <button onClick={() => void unsubscribe()}>Disable notifications</button>
  ) : (
    <button onClick={() => void subscribe()}>Enable notifications</button>
  )
}`;

  const vapidGen = `npx web-push generate-vapid-keys`;

  return (
    <div className="stack-32" style={{ maxWidth: 720 }}>
      {hasNative && hasWeb && (
        <div className="seg" role="group" aria-label="Client SDK">
          <button
            type="button"
            className="seg__btn"
            aria-pressed={platform === "flutter"}
            onClick={() => setPlatformChoice("flutter")}
          >
            Flutter
          </button>
          <button
            type="button"
            className="seg__btn"
            aria-pressed={platform === "react"}
            onClick={() => setPlatformChoice("react")}
          >
            React
          </button>
        </div>
      )}

      {platform === "flutter" ? (
        <>
          <section className="stack-12">
            <h2>1 · Add the SDK</h2>
            <p className="caption">
              This instance serves the{" "}
              <span className="inline-code">moth_auth</span> Flutter SDK from its
              own pub repository at{" "}
              <span className="inline-code">{base}/pub</span>; the SDK version
              tracks the server version.
            </p>
            <p className="caption body-strong">pubspec.yaml</p>
            <CodeBlock code={pubspec} />
          </section>

          <section className="stack-12">
            <h2>2 · Wrap your app</h2>
            <p className="caption">
              Your publishable key is safe to embed in the app:
            </p>
            <KeyWell value={project.publishableKey} />
            <p className="caption body-strong">lib/main.dart</p>
            <CodeBlock code={mainDart} />
          </section>
        </>
      ) : (
        <>
          <section className="stack-12">
            <h2>1 · Add the SDK</h2>
            <p className="caption">
              This instance serves the{" "}
              <span className="inline-code">@moth/react</span> package from its
              own npm registry at{" "}
              <span className="inline-code">{base}/npm</span>; the SDK version
              tracks the server version. The scope line routes only{" "}
              <span className="inline-code">@moth</span> packages here —
              everything else stays on npmjs.
            </p>
            <p className="caption body-strong">.npmrc</p>
            <CodeBlock code={npmrc} />
            <CodeBlock code={npmInstall} />
            {npmConstraint === null && (
              <p className="caption">
                This server does not expose{" "}
                <span className="inline-code">/npm</span> yet, so the served
                version could not be read — the install resolves once the
                instance is upgraded.
              </p>
            )}
          </section>

          <section className="stack-12">
            <h2>2 · Wrap your app</h2>
            <p className="caption">
              Your publishable key is safe to embed in the app:
            </p>
            <KeyWell value={project.publishableKey} />
            <p className="caption body-strong">src/main.tsx</p>
            <CodeBlock code={mainTsx} />
            <p className="caption">
              <span className="inline-code">projectSlug</span> enables the
              Google/Apple buttons (the web-redirect OAuth flow); also
              register your app's origin under Providers →{" "}
              <span className="body-strong">"Redirect origins (web)"</span>.
            </p>
            <p className="caption">
              To call your own backend, attach{" "}
              <span className="inline-code">
                Authorization: Bearer &lt;accessToken&gt;
              </span>{" "}
              (the SDK's fetch wrapper does it for you) and verify the token
              exactly as in step 3 below — same JWKS, issuer and audience.
            </p>
          </section>
        </>
      )}

      <section className="stack-12">
        <h2>3 · Verify tokens on your backend</h2>
        <p className="caption">
          moth signs each access token with this project's ES256 key. Verify
          offline against the JWKS — no call to moth needed:
        </p>
        <div className="stack-8">
          <span className="field__label">JWKS URL</span>
          <KeyWell value={jwks} />
        </div>
        <div className="stack-8">
          <span className="field__label">Expected claims</span>
          <div className="keywell">
            <span className="keywell__value">
              iss = {issuer} · aud = {audience} · alg = ES256
            </span>
          </div>
        </div>

        <p className="caption body-strong">Node — jose</p>
        <CodeBlock code={nodeJose} />
        <p className="caption body-strong">Go — lestrrat-go/jwx</p>
        <CodeBlock code={goJwx} />
        <p className="caption body-strong">Dart — jose</p>
        <CodeBlock code={dartVerify} />
      </section>

      <section className="stack-12">
        <h2>4 · Or introspect online</h2>
        <p className="caption">
          When you'd rather ask moth (instant revocation checks), call{" "}
          <span className="inline-code">IntrospectToken</span> with your secret
          key:
        </p>
        <CodeBlock code={grpcurl} />
        <p className="caption">
          Generate a typed client for any language from the proto files:{" "}
          <a href={`${base}/protos/moth/auth/v1/auth.proto`} download>
            moth.auth.v1
          </a>
          {" · "}
          <a href={`${base}/protos/moth/server/v1/token.proto`} download>
            moth.server.v1 tokens
          </a>
          {" · "}
          <a href={`${base}/protos/moth/server/v1/user.proto`} download>
            moth.server.v1 users
          </a>
        </p>
      </section>

      <section className="stack-12">
        <h2>5 · Prefer the terminal?</h2>
        <p className="caption">
          The <span className="inline-code">moth</span> CLI configures the
          Google and Apple sign-in consoles for this project in one command
          each — automated where the providers expose APIs, guided (with the
          exact values to paste) where they don't — then verifies the result.
          Create a personal access token under Settings first;{" "}
          <span className="inline-code">moth login</span> asks for it.
        </p>
        <CodeBlock code={cliSetup} />
      </section>

      {showMonetization && (
      <section className="stack-12">
        <h2>{monetizeSection} · Monetize (optional)</h2>
        <p className="caption">
          Sell subscriptions without a billing SaaS: define an entitlement
          (e.g. <span className="inline-code">pro</span>) and your tiers under
          the <span className="body-strong">Monetization</span> tab, connect
          the store credentials there (or run the one-command setup below),
          and push the catalog to App Store Connect, Google Play and Stripe. Paywall
          copy and layout live under{" "}
          <span className="body-strong">Design → Paywall</span>, per language.
          A free tier is always built in — apps without paid tiers keep
          working unchanged.
        </p>
        <CodeBlock code={cliBilling} />
        <p className="caption">
          In the app, gate content behind the entitlement — free users see the
          themed paywall; receipts are validated server-side and your backend
          can double-check with{" "}
          <span className="inline-code">
            moth.server.v1.EntitlementService/GetUserEntitlements
          </span>
          .
        </p>
        {platform === "flutter" ? (
          <>
            <p className="caption">
              Add <span className="inline-code">moth_billing</span> — moth's
              first-party StoreKit 2 / Play Billing plugin, served from this
              instance's <span className="inline-code">/pub</span> at the same
              version as <span className="inline-code">moth_auth</span> — and
              pass its adapter. No billing plugin to wire, no adapter code to
              write.
            </p>
            <p className="caption body-strong">pubspec.yaml</p>
            <CodeBlock code={pubspecBilling} />
            <p className="caption body-strong">lib/main.dart</p>
            <CodeBlock code={paywallDart} />
          </>
        ) : (
          <>
            <p className="caption body-strong">src/main.tsx</p>
            <CodeBlock code={paywallReact} />
          </>
        )}
      </section>
      )}

      {showPush && (
      <section className="stack-12">
        <h2>{pushSection} · Push notifications (optional)</h2>
        <p className="caption">
          moth registers devices; your backend sends. Enable push registration
          under the <span className="body-strong">Settings</span> tab, and
          every signed-in device registers its push credential (APNs, FCM or
          Web Push) with an honest permission state. Your server reads the
          registry through{" "}
          <span className="inline-code">
            moth.server.v1.PushService/ListUserPushDevices
          </span>{" "}
          and delivers with the push services' own APIs — moth never sends,
          and sender credentials never touch it.
        </p>
        {platform === "flutter" ? (
          <>
            <p className="caption">
              Add <span className="inline-code">moth_push</span> — moth's
              first-party APNs / FCM plugin, served from this instance's{" "}
              <span className="inline-code">/pub</span> at the same version as{" "}
              <span className="inline-code">moth_auth</span> — and pass its
              adapter. Registration is automatic while signed in; the OS
              permission prompt stays an explicit app call.
            </p>
            <p className="caption body-strong">pubspec.yaml</p>
            <CodeBlock code={pubspecPush} />
            <p className="caption body-strong">lib/main.dart</p>
            <CodeBlock code={pushDart} />
            <p className="caption">
              On Android, FCM needs your app's own Firebase config
              (google-services.json) — the one piece of setup moth cannot
              absorb. iOS needs the Push Notifications capability in Xcode.
            </p>
          </>
        ) : (
          <>
            <p className="caption">
              Web Push additionally needs a VAPID keypair — generate one,
              paste the <span className="body-strong">public</span> key under
              the <span className="body-strong">Settings</span> tab and keep
              the private key in your sender (it never touches moth):
            </p>
            <CodeBlock code={vapidGen} />
            <p className="caption body-strong">src/App.tsx</p>
            <CodeBlock code={pushReact} />
          </>
        )}
      </section>
      )}
    </div>
  );
}
