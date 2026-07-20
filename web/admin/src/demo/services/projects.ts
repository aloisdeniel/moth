// Demo implementations of ProjectService and ProfileService: the projects
// list, per-project settings/providers, secret & signing keys, and the
// milestone-22 setup profile + derived checklist. State is one JSON slice;
// every secret returned is an obviously-fake demo value.

import { handle } from "../transport";
import {
  daysAgo,
  invalidArgument,
  notFound,
  now,
  randomId,
  ts,
  type Millis,
} from "../util";
import { PEOPLE, PROJECT_MAIN, PROJECT_SIDE } from "../ids";
import { ProjectService } from "../../gen/moth/admin/v1/project_pb";
import type { ProjectSettings } from "../../gen/moth/admin/v1/project_pb";
import { ProfilePlatform, ProfileService } from "../../gen/moth/admin/v1/profile_pb";
import type { SetupItemSchema } from "../../gen/moth/admin/v1/profile_pb";
import type { MessageInitShape } from "@bufbuild/protobuf";
import type {
  ProjectSchema,
  ProjectSettingsSchema,
  SigningKeySchema,
} from "../../gen/moth/admin/v1/project_pb";

// ---------------------------------------------------------------------------
// State slice
// ---------------------------------------------------------------------------

export interface DemoGoogleConfig {
  enabled: boolean;
  webClientId: string;
  iosClientId: string;
  androidClientId: string;
  // The secret itself is write-only and never stored in demo state — only
  // its presence.
  hasWebClientSecret: boolean;
}

export interface DemoAppleConfig {
  enabled: boolean;
  servicesId: string;
  teamId: string;
  keyId: string;
  hasPrivateKey: boolean;
  bundleIds: string[];
}

export interface DemoProjectSettings {
  passwordMinLength: number;
  requireEmailVerification: boolean;
  allowPublicSignup: boolean;
  enumerationSafeSignup: boolean;
  accessTokenTtlSeconds: number;
  refreshTokenTtlDays: number;
  google: DemoGoogleConfig;
  apple: DemoAppleConfig;
  autoLinkVerifiedEmail: boolean;
  redirectSchemes: string[];
  redirectOrigins: string[];
  analyticsRetentionDays: number;
  rollupTimezone: string;
  signupEmailAllowlist: string[];
  signupEmailBlocklist: string[];
  captchaVerifyUrl: string;
}

export interface DemoSigningKey {
  kid: string;
  algorithm: string;
  publicKeyPem: string;
  createdAt: Millis;
}

export interface DemoProject {
  id: string;
  name: string;
  slug: string;
  publishableKey: string;
  createdAt: Millis;
  updatedAt: Millis;
  userCount: number;
  settings: DemoProjectSettings;
  signingKey: DemoSigningKey;
  // Stored stand-ins for config owned by other domains (billing credentials
  // + catalog sync, push registry). The real server derives the checklist
  // from live config; the demo stores the two bits it cannot see.
  monetizationConfigured: boolean;
  pushConfigured: boolean;
}

export interface DemoProfile {
  platforms: ProfilePlatform[];
  googleSignIn: boolean;
  appleSignIn: boolean;
  sellsSubscriptions: boolean;
  sendsPushes: boolean;
  checklistDismissed: boolean;
}

export interface ProjectsSlice {
  projects: DemoProject[];
  // Keyed by project id; a missing entry is a pre-wizard project.
  profiles: Record<string, DemoProfile>;
}

// ---------------------------------------------------------------------------
// Fake key material
// ---------------------------------------------------------------------------

// The demo instance's public base URL (mirrors the instance slice's seed).
const BASE_URL = "https://demo.moth.local";

const SLUG_RE = /^[a-z0-9]+(-[a-z0-9]+)*$/;

function slugify(name: string): string {
  return name
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

function randomToken(length: number): string {
  return (randomId() + randomId()).replace(/-/g, "").slice(0, length);
}

// A well-formed-looking (but entirely fake) ES256 SPKI PEM derived from the
// kid, so rotation visibly changes the PEM too. Never real key material.
function fakePem(kid: string): string {
  const filler = (kid.replace(/[^A-Za-z0-9]/g, "") + "demo".repeat(24)).slice(0, 51);
  const body = "MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE" + filler;
  return [
    "-----BEGIN PUBLIC KEY-----",
    body.slice(0, 64),
    body.slice(64) + "==",
    "-----END PUBLIC KEY-----",
  ].join("\n");
}

function newSigningKey(at: Millis): DemoSigningKey {
  const kid = "demo-" + randomToken(24);
  return { kid, algorithm: "ES256", publicKeyPem: fakePem(kid), createdAt: at };
}

function seededSigningKey(kid: string, at: Millis): DemoSigningKey {
  return { kid, algorithm: "ES256", publicKeyPem: fakePem(kid), createdAt: at };
}

// ---------------------------------------------------------------------------
// Seed
// ---------------------------------------------------------------------------

function defaultSettings(): DemoProjectSettings {
  return {
    passwordMinLength: 8,
    requireEmailVerification: false,
    allowPublicSignup: true,
    enumerationSafeSignup: false,
    accessTokenTtlSeconds: 900,
    refreshTokenTtlDays: 30,
    google: {
      enabled: false,
      webClientId: "",
      iosClientId: "",
      androidClientId: "",
      hasWebClientSecret: false,
    },
    apple: {
      enabled: false,
      servicesId: "",
      teamId: "",
      keyId: "",
      hasPrivateKey: false,
      bundleIds: [],
    },
    autoLinkVerifiedEmail: true,
    redirectSchemes: [],
    redirectOrigins: [],
    analyticsRetentionDays: 90,
    rollupTimezone: "UTC",
    signupEmailAllowlist: [],
    signupEmailBlocklist: [],
    captchaVerifyUrl: "",
  };
}

export function seedProjects(): ProjectsSlice {
  const aurora: DemoProject = {
    id: PROJECT_MAIN.id,
    name: PROJECT_MAIN.name,
    slug: PROJECT_MAIN.slug,
    publishableKey: PROJECT_MAIN.publishableKey,
    createdAt: daysAgo(180),
    updatedAt: daysAgo(2),
    userCount: PEOPLE.length,
    settings: {
      ...defaultSettings(),
      requireEmailVerification: true,
      enumerationSafeSignup: true,
      google: {
        enabled: true,
        webClientId: "281764509123-aurorawebk3v9q2m8w1x4.apps.googleusercontent.com",
        iosClientId: "281764509123-auroraios8f4j2n7d5s6q.apps.googleusercontent.com",
        androidClientId: "281764509123-auroradroidq2w9e4r7t1.apps.googleusercontent.com",
        hasWebClientSecret: true,
      },
      apple: {
        enabled: true,
        servicesId: "app.aurorajournal.signin",
        teamId: "9QDEMOAU23",
        keyId: "D3MOAPL42K",
        hasPrivateKey: true,
        bundleIds: ["app.aurorajournal.ios"],
      },
      redirectSchemes: ["aurora"],
      redirectOrigins: [],
      rollupTimezone: "Europe/Paris",
      signupEmailBlocklist: ["mailinator.com"],
    },
    // Rotated once mid-life, so "created" reads as an operated project.
    signingKey: seededSigningKey("demo-aurora-es256-4f2b9c1d7a8e", daysAgo(45)),
    monetizationConfigured: true,
    pushConfigured: true,
  };

  const skylark: DemoProject = {
    id: PROJECT_SIDE.id,
    name: PROJECT_SIDE.name,
    slug: PROJECT_SIDE.slug,
    publishableKey: PROJECT_SIDE.publishableKey,
    createdAt: daysAgo(3),
    updatedAt: daysAgo(3),
    userCount: 0,
    settings: defaultSettings(),
    signingKey: seededSigningKey("demo-skylark-es256-9c0d3e6f2a1b", daysAgo(3)),
    monetizationConfigured: false,
    pushConfigured: false,
  };

  return {
    projects: [aurora, skylark],
    profiles: {
      // Aurora's wizard answers: a mobile journaling app with both social
      // providers, a Pro subscription and pushes — everything configured, so
      // the checklist is clean. Skylark predates the wizard: no profile.
      [PROJECT_MAIN.id]: {
        platforms: [ProfilePlatform.IOS, ProfilePlatform.ANDROID],
        googleSignIn: true,
        appleSignIn: true,
        sellsSubscriptions: true,
        sendsPushes: true,
        checklistDismissed: false,
      },
    },
  };
}

// ---------------------------------------------------------------------------
// Proto conversion
// ---------------------------------------------------------------------------

function settingsToProto(s: DemoProjectSettings): MessageInitShape<typeof ProjectSettingsSchema> {
  return {
    passwordMinLength: s.passwordMinLength,
    requireEmailVerification: s.requireEmailVerification,
    allowPublicSignup: s.allowPublicSignup,
    enumerationSafeSignup: s.enumerationSafeSignup,
    accessTokenTtlSeconds: s.accessTokenTtlSeconds,
    refreshTokenTtlDays: s.refreshTokenTtlDays,
    google: {
      enabled: s.google.enabled,
      webClientId: s.google.webClientId,
      iosClientId: s.google.iosClientId,
      androidClientId: s.google.androidClientId,
      // Write-only: never returned.
      webClientSecret: "",
      hasWebClientSecret: s.google.hasWebClientSecret,
    },
    apple: {
      enabled: s.apple.enabled,
      servicesId: s.apple.servicesId,
      teamId: s.apple.teamId,
      keyId: s.apple.keyId,
      privateKeyP8: "",
      hasPrivateKey: s.apple.hasPrivateKey,
      bundleIds: [...s.apple.bundleIds],
    },
    autoLinkVerifiedEmail: s.autoLinkVerifiedEmail,
    redirectSchemes: [...s.redirectSchemes],
    redirectOrigins: [...s.redirectOrigins],
    analyticsRetentionDays: s.analyticsRetentionDays,
    rollupTimezone: s.rollupTimezone,
    signupEmailAllowlist: [...s.signupEmailAllowlist],
    signupEmailBlocklist: [...s.signupEmailBlocklist],
    captchaVerifyUrl: s.captchaVerifyUrl,
  };
}

// settingsFromProto applies a wholesale settings replacement, honoring the
// write-only secret convention (empty keeps the stored one) and normalizing
// zero values to the server defaults, like the real handler does.
function settingsFromProto(
  s: ProjectSettings,
  prev: DemoProjectSettings,
): DemoProjectSettings {
  return {
    passwordMinLength: s.passwordMinLength || 8,
    requireEmailVerification: s.requireEmailVerification,
    allowPublicSignup: s.allowPublicSignup,
    enumerationSafeSignup: s.enumerationSafeSignup,
    accessTokenTtlSeconds: s.accessTokenTtlSeconds || 900,
    refreshTokenTtlDays: s.refreshTokenTtlDays || 30,
    google: {
      enabled: s.google?.enabled ?? false,
      webClientId: s.google?.webClientId ?? "",
      iosClientId: s.google?.iosClientId ?? "",
      androidClientId: s.google?.androidClientId ?? "",
      hasWebClientSecret:
        (s.google?.webClientSecret ?? "") !== "" ? true : prev.google.hasWebClientSecret,
    },
    apple: {
      enabled: s.apple?.enabled ?? false,
      servicesId: s.apple?.servicesId ?? "",
      teamId: s.apple?.teamId ?? "",
      keyId: s.apple?.keyId ?? "",
      hasPrivateKey:
        (s.apple?.privateKeyP8 ?? "") !== "" ? true : prev.apple.hasPrivateKey,
      bundleIds: [...(s.apple?.bundleIds ?? [])],
    },
    autoLinkVerifiedEmail: s.autoLinkVerifiedEmail ?? true,
    redirectSchemes: [...s.redirectSchemes],
    redirectOrigins: [...s.redirectOrigins],
    analyticsRetentionDays: s.analyticsRetentionDays || 90,
    rollupTimezone: s.rollupTimezone || "UTC",
    signupEmailAllowlist: [...s.signupEmailAllowlist],
    signupEmailBlocklist: [...s.signupEmailBlocklist],
    captchaVerifyUrl: s.captchaVerifyUrl,
  };
}

function projectToProto(p: DemoProject): MessageInitShape<typeof ProjectSchema> {
  return {
    id: p.id,
    name: p.name,
    slug: p.slug,
    publishableKey: p.publishableKey,
    createTime: ts(p.createdAt),
    updateTime: ts(p.updatedAt),
    settings: settingsToProto(p.settings),
    userCount: BigInt(p.userCount),
  };
}

function signingKeyToProto(k: DemoSigningKey): MessageInitShape<typeof SigningKeySchema> {
  return {
    kid: k.kid,
    algorithm: k.algorithm,
    publicKeyPem: k.publicKeyPem,
    createTime: ts(k.createdAt),
  };
}

function mustProject(state: ProjectsSlice, id: string): DemoProject {
  const p = state.projects.find((x) => x.id === id);
  if (!p) throw notFound("project");
  return p;
}

// ---------------------------------------------------------------------------
// Setup checklist derivation
// ---------------------------------------------------------------------------

// Mirrors the real GetProjectSetupStatus as far as this slice can see:
// provider items derive from the stored provider config; billing and push
// come from the stored stand-in flags.
function deriveSetupItems(
  p: DemoProject,
  profile: DemoProfile,
): MessageInitShape<typeof SetupItemSchema>[] {
  const items: MessageInitShape<typeof SetupItemSchema>[] = [];
  if (profile.googleSignIn && !(p.settings.google.enabled && p.settings.google.webClientId !== "")) {
    items.push({
      id: "google_credentials",
      title: "Finish Google sign-in",
      detail:
        "Google sign-in was chosen, but the provider is not enabled with a client ID yet.",
      tab: "providers",
      cliCommand: `moth setup google --project ${p.slug}`,
    });
  }
  if (profile.appleSignIn && !p.settings.apple.enabled) {
    items.push({
      id: "apple_credentials",
      title: "Finish Sign in with Apple",
      detail: "Apple sign-in was chosen, but the provider is not enabled yet.",
      tab: "providers",
      cliCommand: `moth setup apple --project ${p.slug}`,
    });
  }
  if (profile.sellsSubscriptions && !p.monetizationConfigured) {
    items.push({
      id: "billing_credentials",
      title: "Add store billing credentials",
      detail:
        "Subscriptions were chosen, but the store credentials and catalog sync are not set up yet.",
      tab: "monetization",
      cliCommand: `moth setup billing --project ${p.slug}`,
    });
  }
  if (profile.sendsPushes && !p.pushConfigured) {
    items.push({
      id: "push_vapid",
      title: "Finish push setup",
      detail:
        "Push notifications were chosen, but the push registry is not fully configured yet.",
      tab: "push",
      cliCommand: "",
    });
  }
  return items;
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

export function registerProjects(): void {
  // ----- ProjectService -----

  handle(ProjectService.method.listProjects, (state: ProjectsSlice, _req) => ({
    projects: state.projects.map(projectToProto),
  }));

  handle(ProjectService.method.getProject, (state: ProjectsSlice, req) => ({
    project: projectToProto(mustProject(state, req.id)),
  }));

  handle(ProjectService.method.createProject, (state: ProjectsSlice, req) => {
    const name = req.name.trim();
    if (name === "") throw invalidArgument("name is required");
    let slug = req.slug.trim();
    if (slug !== "") {
      if (!SLUG_RE.test(slug)) {
        throw invalidArgument("slug must be lowercase letters, digits and single dashes");
      }
      if (state.projects.some((p) => p.slug === slug)) {
        throw invalidArgument(`slug "${slug}" is already taken`);
      }
    } else {
      slug = slugify(name);
      if (slug === "") throw invalidArgument("name must contain letters or digits");
      const base = slug;
      for (let n = 2; state.projects.some((p) => p.slug === slug); n++) {
        slug = `${base}-${n}`;
      }
    }
    const at = now();
    const project: DemoProject = {
      id: randomId(),
      name,
      slug,
      publishableKey: `pk_demo_${slug.replace(/-/g, "")}_${randomToken(12)}`,
      createdAt: at,
      updatedAt: at,
      userCount: 0,
      settings: defaultSettings(),
      signingKey: newSigningKey(at),
      monetizationConfigured: false,
      pushConfigured: false,
    };
    state.projects.push(project);
    return {
      project: projectToProto(project),
      secretKey: `sk_demo_${randomToken(32)}`,
    };
  });

  handle(ProjectService.method.updateProject, (state: ProjectsSlice, req) => {
    const p = mustProject(state, req.id);
    const paths = req.updateMask?.paths ?? [];
    const hasMask = paths.length > 0;
    const applyName = hasMask ? paths.includes("name") : true;
    const applySettings =
      (hasMask ? paths.includes("settings") : true) && req.settings !== undefined;
    if (applyName) {
      const name = req.name.trim();
      if (name === "") throw invalidArgument("name is required");
      p.name = name;
    }
    if (applySettings && req.settings) {
      p.settings = settingsFromProto(req.settings, p.settings);
    }
    p.updatedAt = now();
    return { project: projectToProto(p) };
  });

  handle(ProjectService.method.deleteProject, (state: ProjectsSlice, req) => {
    const i = state.projects.findIndex((p) => p.id === req.id);
    if (i < 0) throw notFound("project");
    state.projects.splice(i, 1);
    delete state.profiles[req.id];
    return {};
  });

  handle(ProjectService.method.regenerateSecretKey, (state: ProjectsSlice, req) => {
    const p = mustProject(state, req.projectId);
    p.updatedAt = now();
    return {
      project: projectToProto(p),
      secretKey: `sk_demo_${randomToken(32)}`,
    };
  });

  handle(ProjectService.method.getSigningKey, (state: ProjectsSlice, req) => {
    const p = mustProject(state, req.projectId);
    return {
      key: signingKeyToProto(p.signingKey),
      jwksUrl: `${BASE_URL}/p/${p.slug}/.well-known/jwks.json`,
      issuer: `${BASE_URL}/p/${p.slug}`,
      audience: p.slug,
    };
  });

  handle(ProjectService.method.rotateSigningKey, (state: ProjectsSlice, req) => {
    const p = mustProject(state, req.projectId);
    const at = now();
    p.signingKey = newSigningKey(at);
    p.updatedAt = at;
    // Default grace: access-token TTL + one minute of clock skew.
    const grace =
      req.graceSeconds > 0 ? req.graceSeconds : p.settings.accessTokenTtlSeconds + 60;
    return {
      key: signingKeyToProto(p.signingKey),
      graceExpireTime: ts(at + grace * 1000),
    };
  });

  handle(ProjectService.method.resetSigningKey, (state: ProjectsSlice, req) => {
    const p = mustProject(state, req.projectId);
    const at = now();
    p.signingKey = newSigningKey(at);
    p.updatedAt = at;
    return { key: signingKeyToProto(p.signingKey) };
  });

  handle(ProjectService.method.exportProject, (state: ProjectsSlice, req) => {
    const p = mustProject(state, req.projectId);
    if (p.id !== PROJECT_MAIN.id) return { users: [] };
    // The flagship project's people, rebuilt from the shared cast so the
    // export tells the same story as the users screen.
    return {
      users: PEOPLE.map((person) => ({
        id: person.id,
        email: person.email,
        emailVerified: true,
        displayName: person.name,
        avatarUrl: "",
        customClaims: "{}",
        disabled: false,
        createTime: ts(daysAgo(person.signupDaysAgo)),
        lastLoginTime: ts(daysAgo(person.signupDaysAgo % 7)),
        passwordHash:
          person.provider === "password"
            ? "$argon2id$v=19$m=65536,t=3,p=4$ZGVtby1zYWx0$ZmFrZS1kZW1vLWhhc2g"
            : "",
        passwordAlgorithm: person.provider === "password" ? "argon2id" : "",
        identities: [
          {
            provider: person.provider,
            providerSubject:
              person.provider === "password" ? person.id : `${person.provider}-${randomToken(16)}`,
            email: person.email,
          },
        ],
      })),
    };
  });

  handle(ProjectService.method.importProject, (state: ProjectsSlice, req) => {
    const p = mustProject(state, req.projectId);
    const imported = req.users.length;
    p.userCount += imported;
    p.updatedAt = now();
    return { importedCount: imported, skippedCount: 0 };
  });

  // ----- ProfileService -----

  handle(ProfileService.method.getProfile, (state: ProjectsSlice, req) => {
    mustProject(state, req.projectId);
    const profile = state.profiles[req.projectId];
    if (!profile) return { hasProfile: false };
    return { hasProfile: true, profile: { ...profile, platforms: [...profile.platforms] } };
  });

  handle(ProfileService.method.updateProfile, (state: ProjectsSlice, req) => {
    mustProject(state, req.projectId);
    if (!req.profile) throw invalidArgument("profile is required");
    const platforms = req.profile.platforms.filter(
      (p) => p !== ProfilePlatform.UNSPECIFIED,
    );
    if (platforms.length === 0) throw invalidArgument("platforms must be non-empty");
    const profile: DemoProfile = {
      platforms,
      googleSignIn: req.profile.googleSignIn,
      appleSignIn: req.profile.appleSignIn,
      sellsSubscriptions: req.profile.sellsSubscriptions,
      sendsPushes: req.profile.sendsPushes,
      checklistDismissed: req.profile.checklistDismissed,
    };
    state.profiles[req.projectId] = profile;
    return { profile: { ...profile, platforms: [...profile.platforms] } };
  });

  handle(ProfileService.method.getProjectSetupStatus, (state: ProjectsSlice, req) => {
    const p = mustProject(state, req.projectId);
    const profile = state.profiles[req.projectId];
    if (!profile) {
      return { hasProfile: false, items: [], checklistDismissed: false };
    }
    return {
      hasProfile: true,
      items: deriveSetupItems(p, profile),
      checklistDismissed: profile.checklistDismissed,
    };
  });
}
