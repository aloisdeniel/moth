import { spawn, spawnSync, type ChildProcess } from "node:child_process";
import { mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));

// Port map (web/admin's e2e owns 8990):
//   8992 moth   8993 stripe double   8991 example app A   8994 example app B
export const MOTH_URL = "http://127.0.0.1:8992";
export const STRIPE_URL = "http://127.0.0.1:8993";
export const APP_URL = "http://127.0.0.1:8991"; // Stripe-enabled project
export const APP_NOSTRIPE_URL = "http://127.0.0.1:8994"; // no billing config

export const STATE_FILE = join(tmpdir(), "moth-react-e2e-state.json");

const ADMIN_EMAIL = "admin@example.com";
const ADMIN_PASSWORD = "admin-password-1";
const STRIPE_SECRET_KEY = "sk_test_react_e2e";
const STRIPE_WEBHOOK_SECRET = "whsec_react_e2e";

interface Child {
  name: string;
  proc: ChildProcess;
  log: () => string;
}

const children: Child[] = [];

function launch(name: string, command: string, args: string[], opts: { cwd?: string; env?: Record<string, string | undefined> }): Child {
  const proc = spawn(command, args, {
    cwd: opts.cwd,
    env: { ...process.env, ...opts.env },
    stdio: ["ignore", "pipe", "pipe"],
    detached: false,
  });
  let output = "";
  proc.stdout?.on("data", (chunk: Buffer) => (output += chunk.toString()));
  proc.stderr?.on("data", (chunk: Buffer) => (output += chunk.toString()));
  const child = { name, proc, log: () => output };
  children.push(child);
  return child;
}

async function waitFor(url: string, child: Child, timeoutMs = 30_000) {
  const deadline = Date.now() + timeoutMs;
  for (;;) {
    if (child.proc.exitCode !== null) {
      throw new Error(`${child.name} exited early (${child.proc.exitCode}):\n${child.log()}`);
    }
    try {
      const resp = await fetch(url);
      if (resp.ok) return;
    } catch {
      // not up yet
    }
    if (Date.now() > deadline) {
      throw new Error(`${child.name} did not answer at ${url}:\n${child.log()}`);
    }
    await new Promise((r) => setTimeout(r, 100));
  }
}

/** Calls a connect unary RPC with a JSON body. */
async function rpc(procedure: string, body: unknown, cookie?: string): Promise<any> {
  const headers: Record<string, string> = { "content-type": "application/json" };
  if (cookie !== undefined) headers.cookie = cookie;
  const resp = await fetch(`${MOTH_URL}${procedure}`, {
    method: "POST",
    headers,
    body: JSON.stringify(body),
  });
  const text = await resp.text();
  if (!resp.ok) throw new Error(`${procedure}: ${resp.status} ${text}`);
  return JSON.parse(text);
}

async function adminLogin(): Promise<string> {
  const resp = await fetch(`${MOTH_URL}/moth.admin.v1.SessionService/Login`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ email: ADMIN_EMAIL, password: ADMIN_PASSWORD }),
  });
  if (!resp.ok) throw new Error(`admin login: ${resp.status} ${await resp.text()}`);
  const setCookie = resp.headers.get("set-cookie");
  if (setCookie === null) throw new Error("admin login: no session cookie");
  return setCookie.split(";")[0]!;
}

async function createProject(cookie: string, name: string, settings: Record<string, unknown>) {
  const created = await rpc("/moth.admin.v1.ProjectService/CreateProject", { name }, cookie);
  const project = created.project as { id: string; slug: string; publishableKey: string };
  await rpc(
    "/moth.admin.v1.ProjectService/UpdateProject",
    { id: project.id, settings, updateMask: "settings" },
    cookie,
  );
  return project;
}

function launchExampleApp(name: string, port: number, publishableKey: string, projectSlug: string): Child {
  const exampleDir = resolve(here, "../example");
  const viteBin = join(exampleDir, "node_modules", "vite", "bin", "vite.js");
  return launch(name, process.execPath, [viteBin, "--port", String(port), "--strictPort", "--host", "127.0.0.1"], {
    cwd: exampleDir,
    env: {
      VITE_MOTH_ENDPOINT: MOTH_URL,
      VITE_MOTH_PUBLISHABLE_KEY: publishableKey,
      VITE_MOTH_PROJECT_SLUG: projectSlug,
    },
  });
}

// Spawns the real moth binary (fresh temp data dir) with its Stripe calls
// pointed at a local test double, provisions two projects over the admin
// RPCs — one fully Stripe-configured with a 5-second access-token TTL, one
// with no billing config at all — and serves the example app once per
// project via Vite.
export default async function globalSetup() {
  const bin = process.env.MOTH_BIN ?? resolve(here, "../../../bin/moth");
  const dataDir = mkdtempSync(join(tmpdir(), "moth-react-e2e-"));

  // The admin account is created directly in the data dir, before serving.
  const create = spawnSync(bin, [
    "admin", "create",
    "--email", ADMIN_EMAIL,
    "--password", ADMIN_PASSWORD,
    "--data-dir", dataDir,
  ]);
  if (create.status !== 0) {
    throw new Error(`moth admin create failed:\n${create.stdout}\n${create.stderr}`);
  }

  try {
    const moth = launch("moth", bin, ["serve"], {
      env: {
        MOTH_ADDR: "127.0.0.1:8992",
        MOTH_BASE_URL: MOTH_URL,
        MOTH_DATA_DIR: dataDir,
        // The env-only testing hook: billing's outbound Stripe calls go to
        // the local double instead of api.stripe.com.
        MOTH_STRIPE_API_URL: STRIPE_URL,
        // The suite signs in/up many times in quick succession from one IP.
        MOTH_RATELIMIT_IP_PER_MINUTE: "600",
      },
    });
    await waitFor(`${MOTH_URL}/healthz`, moth);

    const cookie = await adminLogin();

    // Project A: Stripe-configured, 5-second access tokens (mid-session
    // expiry happens inside a test), no email verification so sign-up opens
    // a session immediately (like the milestone-05 Flutter e2e).
    const projectA = await createProject(cookie, "React SDK e2e", {
      passwordMinLength: 8,
      requireEmailVerification: false,
      allowPublicSignup: true,
      accessTokenTtlSeconds: 5,
      refreshTokenTtlDays: 30,
    });
    const ent = await rpc(
      "/moth.admin.v1.EntitlementService/CreateEntitlement",
      { projectId: projectA.id, identifier: "pro", displayName: "Pro" },
      cookie,
    );
    await rpc(
      "/moth.admin.v1.ProductService/CreateProduct",
      {
        projectId: projectA.id,
        product: {
          identifier: "monthly",
          displayName: "Monthly",
          billingPeriod: "P1M",
          priceAmountMicros: "9990000",
          currency: "USD",
          stripePriceId: "price_e2e_monthly",
          stripeProductId: "prod_e2e_1",
          entitlementIds: [ent.entitlement.id],
        },
      },
      cookie,
    );
    await rpc(
      "/moth.admin.v1.BillingCredentialsService/UpdateBillingCredentials",
      {
        projectId: projectA.id,
        stripe: { secretKey: STRIPE_SECRET_KEY, webhookSecret: STRIPE_WEBHOOK_SECRET },
      },
      cookie,
    );

    // Project B: the same auth settings, no billing configuration at all —
    // entitlement gates must never block here.
    const projectB = await createProject(cookie, "React SDK e2e no stripe", {
      passwordMinLength: 8,
      requireEmailVerification: false,
      allowPublicSignup: true,
    });

    const stripe = launch("stripe-double", process.execPath, [join(here, "stripe-double.mjs")], {
      env: {
        PORT: "8993",
        STRIPE_SECRET_KEY,
        STRIPE_WEBHOOK_SECRET,
        MOTH_WEBHOOK_URL: `${MOTH_URL}/billing/stripe/webhook/${projectA.slug}`,
      },
    });
    await waitFor(`${STRIPE_URL}/healthz`, stripe);

    const appA = launchExampleApp("example-app", 8991, projectA.publishableKey, projectA.slug);
    const appB = launchExampleApp("example-app-nostripe", 8994, projectB.publishableKey, projectB.slug);
    await waitFor(APP_URL, appA);
    await waitFor(APP_NOSTRIPE_URL, appB);

    writeFileSync(
      STATE_FILE,
      JSON.stringify({ pids: children.map((c) => c.proc.pid), dataDir }),
    );
    for (const child of children) child.proc.unref();
  } catch (err) {
    for (const child of children) child.proc.kill();
    throw err;
  }
}
