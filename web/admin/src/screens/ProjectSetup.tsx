import { useQuery } from "@connectrpc/connect-query";
import { useEffect, useState } from "react";

import { errorMessage } from "../api";
import { CodeBlock, ErrorNote, KeyWell, Loading } from "../components/ui";
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

// ProjectSetup renders copy-paste instructions with this project's real
// values — the setup page is the product.
export function ProjectSetup({ project }: { project: Project }) {
  const instance = useQuery(InstanceSettingsService.method.getInstanceSettings);
  const signing = useQuery(ProjectService.method.getSigningKey, { projectId: project.id });
  const sdkVersion = useSdkVersion();

  if (instance.isPending || signing.isPending || sdkVersion === undefined) return <Loading />;
  if (instance.isError) return <ErrorNote message={errorMessage(instance.error)} />;
  if (signing.isError) return <ErrorNote message={errorMessage(signing.error)} />;
  if (sdkVersion === null) return <ErrorNote message="could not load the served SDK version from /pub" />;

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

  return (
    <div className="stack-32" style={{ maxWidth: 720 }}>
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
    </div>
  );
}
