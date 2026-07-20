// Demo implementations of the instance-level admin services: session,
// operator accounts, instance settings (SMTP) and the audit log.

import { handle } from "../transport";
import type { Millis } from "../util";
import {
  daysAgo,
  invalidArgument,
  minutesAgo,
  notFound,
  now,
  randomId,
  ts,
  unauthenticated,
} from "../util";
import {
  ADMIN,
  demoId,
  ENTITLEMENT_PRO_ID,
  PRODUCT_ANNUAL_ID,
  PRODUCT_LIFETIME_ID,
  PRODUCT_MONTHLY_ID,
  PROJECT_MAIN,
  PROJECT_SIDE,
} from "../ids";
import { SessionService } from "../../gen/moth/admin/v1/session_pb";
import { AdminAccountService } from "../../gen/moth/admin/v1/account_pb";
import { InstanceSettingsService, SmtpSource } from "../../gen/moth/admin/v1/settings_pb";
import { AuditService } from "../../gen/moth/admin/v1/audit_pb";

// ---------- State slice ----------

interface AdminState {
  id: string;
  email: string;
  createdAt: Millis;
}

interface AdminInviteState {
  id: string;
  email: string;
  token: string;
  createdAt: Millis;
  expiresAt: Millis;
}

interface PersonalAccessTokenState {
  id: string;
  name: string;
  createdAt: Millis;
  // null when never used / never expires / not revoked (JSON-friendly).
  lastUsedAt: Millis | null;
  expiresAt: Millis | null;
  revokedAt: Millis | null;
}

interface SmtpState {
  host: string;
  port: number;
  username: string;
  from: string;
  source: "none" | "config" | "database";
  hasPassword: boolean;
}

interface AuditEntryState {
  id: string;
  actorType: string;
  actorId: string;
  actorLabel: string;
  action: string;
  targetType: string;
  targetId: string;
  projectId: string;
  summary: string;
  beforeAfter: string;
  ip: string;
  createdAt: Millis;
}

export interface InstanceSlice {
  session: { signedIn: boolean; adminId: string };
  admins: AdminState[];
  adminInvites: AdminInviteState[];
  personalAccessTokens: PersonalAccessTokenState[];
  smtp: SmtpState;
  instanceBaseUrl: string;
  auditEntries: AuditEntryState[];
}

// ---------- Seed ----------

const OPS_ADMIN_ID = demoId("admn", 2);
const OPS_EMAIL = "ops@moth.local";
const PENDING_INVITE_ID = demoId("invt", 1);
const PENDING_INVITE_EMAIL = "jordan@moth.local";
const SEED_PAT_ID = demoId("pato", 1);

const DEMO_IP = "203.0.113.10";

export function seedInstance(): InstanceSlice {
  // Seed audit times: `at(days, minutes)` orders same-day events.
  const at = (days: number, minutes = 0): Millis => daysAgo(days) + minutes * 60_000;

  let auditSeq = 0;
  const entry = (
    createdAt: Millis,
    action: string,
    summary: string,
    over: Partial<AuditEntryState> = {},
  ): AuditEntryState => ({
    id: demoId("audt", ++auditSeq),
    actorType: "cookie",
    actorId: ADMIN.id,
    actorLabel: ADMIN.email,
    action,
    targetType: "",
    targetId: "",
    projectId: "",
    summary,
    beforeAfter: "",
    ip: DEMO_IP,
    createdAt,
    ...over,
  });
  const system = { actorType: "system", actorId: "", actorLabel: "moth server", ip: "" };
  const main = { projectId: PROJECT_MAIN.id };

  const auditEntries: AuditEntryState[] = [
    entry(at(180), "instance.setup", "Instance initialized", {
      ...system,
      targetType: "instance",
    }),
    entry(at(180, 10), "admin.create", `Created the first admin account ${ADMIN.email}`, {
      ...system,
      targetType: "admin",
      targetId: ADMIN.id,
    }),
    entry(at(179), "project.create", "Created project Aurora Journal", {
      ...main,
      targetType: "project",
      targetId: PROJECT_MAIN.id,
    }),
    entry(at(179, 30), "provider.update", "Enabled email & password sign-in", {
      ...main,
      targetType: "provider",
      targetId: "password",
    }),
    entry(at(178), "provider.update", "Configured Google sign-in", {
      ...main,
      targetType: "provider",
      targetId: "google",
    }),
    entry(at(178, 45), "provider.update", "Configured Sign in with Apple", {
      ...main,
      targetType: "provider",
      targetId: "apple",
    }),
    entry(at(177), "settings.update", "Configured outgoing email (smtp.example.com)", {
      targetType: "settings",
      targetId: "smtp",
    }),
    entry(at(152), "product.create", "Created product Aurora Pro Monthly", {
      ...main,
      targetType: "product",
      targetId: PRODUCT_MONTHLY_ID,
    }),
    entry(at(152, 20), "product.create", "Created product Aurora Pro Annual", {
      ...main,
      targetType: "product",
      targetId: PRODUCT_ANNUAL_ID,
    }),
    entry(at(150), "product.create", "Created product Aurora Pro Lifetime", {
      ...main,
      targetType: "product",
      targetId: PRODUCT_LIFETIME_ID,
    }),
    entry(at(150, 25), "entitlement.create", "Created entitlement Pro", {
      ...main,
      targetType: "entitlement",
      targetId: ENTITLEMENT_PRO_ID,
    }),
    entry(at(121), "push.update", "Enabled push notifications (APNs and FCM)", {
      ...main,
      targetType: "project",
      targetId: PROJECT_MAIN.id,
    }),
    entry(at(95), "admin.invite", `Invited admin ${OPS_EMAIL}`, {
      targetType: "admin_invite",
      targetId: OPS_EMAIL,
    }),
    entry(at(93), "admin.create", `${OPS_EMAIL} accepted the admin invite`, {
      ...system,
      targetType: "admin",
      targetId: OPS_ADMIN_ID,
    }),
    entry(at(75), "signing_key.rotate", "Rotated signing key", {
      ...main,
      targetType: "signing_key",
      targetId: PROJECT_MAIN.id,
    }),
    entry(at(47), "secret_key.regenerate", "Regenerated secret key", {
      ...main,
      targetType: "secret_key",
      targetId: PROJECT_MAIN.id,
    }),
    entry(at(24), "pat.create", "Created access token “CI deploy”", {
      targetType: "personal_access_token",
      targetId: SEED_PAT_ID,
    }),
    entry(at(18), "provider.update", "Rotated Google OAuth client secret", {
      ...main,
      targetType: "provider",
      targetId: "google",
    }),
    entry(at(9), "project.update", "Updated paywall copy", {
      ...main,
      targetType: "project",
      targetId: PROJECT_MAIN.id,
    }),
    entry(at(3), "project.create", "Created project Skylark", {
      projectId: PROJECT_SIDE.id,
      targetType: "project",
      targetId: PROJECT_SIDE.id,
    }),
    entry(at(3, 20), "provider.update", "Enabled email & password sign-in", {
      projectId: PROJECT_SIDE.id,
      targetType: "provider",
      targetId: "password",
    }),
    entry(at(2), "admin.login", "Signed in", {
      actorId: OPS_ADMIN_ID,
      actorLabel: OPS_EMAIL,
      targetType: "admin",
      targetId: OPS_ADMIN_ID,
      ip: "198.51.100.23",
    }),
    entry(at(1), "admin.invite", `Invited admin ${PENDING_INVITE_EMAIL}`, {
      targetType: "admin_invite",
      targetId: PENDING_INVITE_ID,
    }),
    entry(at(1, 5), "settings.update", "Updated instance settings", {
      targetType: "settings",
      targetId: "smtp",
    }),
    entry(minutesAgo(75), "admin.login", "Signed in", {
      targetType: "admin",
      targetId: ADMIN.id,
    }),
  ];

  return {
    session: { signedIn: true, adminId: ADMIN.id },
    admins: [
      { id: ADMIN.id, email: ADMIN.email, createdAt: daysAgo(180) },
      { id: OPS_ADMIN_ID, email: OPS_EMAIL, createdAt: daysAgo(93) },
    ],
    adminInvites: [
      {
        id: PENDING_INVITE_ID,
        email: PENDING_INVITE_EMAIL,
        token: "invite_demo_jordan_x1y2z3",
        createdAt: daysAgo(1),
        expiresAt: daysAgo(1) + 72 * 60 * 60 * 1000,
      },
    ],
    personalAccessTokens: [
      {
        id: SEED_PAT_ID,
        name: "CI deploy",
        createdAt: daysAgo(24),
        lastUsedAt: minutesAgo(6 * 60),
        expiresAt: daysAgo(24) + 90 * 24 * 60 * 60 * 1000,
        revokedAt: null,
      },
    ],
    smtp: {
      host: "smtp.example.com",
      port: 587,
      username: "postmaster@demo.moth.local",
      from: "auth@demo.moth.local",
      source: "database",
      hasPassword: true,
    },
    instanceBaseUrl: "https://demo.moth.local",
    auditEntries,
  };
}

// ---------- Helpers ----------

function requireAdmin(s: InstanceSlice): AdminState {
  if (!s.session.signedIn) throw unauthenticated();
  return s.admins.find((a) => a.id === s.session.adminId) ?? s.admins[0];
}

function adminMsg(a: AdminState) {
  return { id: a.id, email: a.email, createTime: ts(a.createdAt) };
}

function inviteMsg(inv: AdminInviteState) {
  return {
    id: inv.id,
    email: inv.email,
    createTime: ts(inv.createdAt),
    expireTime: ts(inv.expiresAt),
  };
}

function patMsg(t: PersonalAccessTokenState) {
  return {
    id: t.id,
    name: t.name,
    createTime: ts(t.createdAt),
    lastUsedTime: t.lastUsedAt === null ? undefined : ts(t.lastUsedAt),
    expireTime: t.expiresAt === null ? undefined : ts(t.expiresAt),
    revokeTime: t.revokedAt === null ? undefined : ts(t.revokedAt),
  };
}

function smtpSourceEnum(source: SmtpState["source"]): SmtpSource {
  switch (source) {
    case "none":
      return SmtpSource.NONE;
    case "config":
      return SmtpSource.CONFIG;
    case "database":
      return SmtpSource.DATABASE;
  }
}

// The effective SMTP settings as read responses return them: password
// blanked (write-only field).
function smtpMsg(s: SmtpState) {
  return { host: s.host, port: s.port, username: s.username, password: "", from: s.from };
}

function audit(
  s: InstanceSlice,
  actor: AdminState,
  action: string,
  summary: string,
  over: Partial<AuditEntryState> = {},
): void {
  s.auditEntries.push({
    id: randomId(),
    actorType: "cookie",
    actorId: actor.id,
    actorLabel: actor.email,
    action,
    targetType: "",
    targetId: "",
    projectId: "",
    summary,
    beforeAfter: "",
    ip: DEMO_IP,
    createdAt: now(),
    ...over,
  });
}

function millisOf(t: { seconds: bigint; nanos: number }): Millis {
  return Number(t.seconds) * 1000 + Math.floor(t.nanos / 1e6);
}

// ---------- Handlers ----------

export function registerInstance(): void {
  // --- SessionService ---

  handle(SessionService.method.login, (s: InstanceSlice, req) => {
    if (req.email === "" || req.password === "") {
      throw invalidArgument("email and password are required");
    }
    // The demo accepts any credentials; a known email signs in as that admin.
    const admin =
      s.admins.find((a) => a.email.toLowerCase() === req.email.toLowerCase()) ?? s.admins[0];
    s.session.signedIn = true;
    s.session.adminId = admin.id;
    audit(s, admin, "admin.login", "Signed in", {
      targetType: "admin",
      targetId: admin.id,
    });
    return { admin: adminMsg(admin) };
  });

  handle(SessionService.method.logout, (s: InstanceSlice, _req) => {
    s.session.signedIn = false;
    return {};
  });

  handle(SessionService.method.getCurrentAdmin, (s: InstanceSlice, _req) => {
    const admin = requireAdmin(s);
    return { admin: adminMsg(admin), serverVersion: "demo" };
  });

  // --- AdminAccountService ---

  handle(AdminAccountService.method.listAdmins, (s: InstanceSlice, _req) => {
    requireAdmin(s);
    return { admins: s.admins.map(adminMsg) };
  });

  handle(AdminAccountService.method.inviteAdmin, (s: InstanceSlice, req) => {
    const actor = requireAdmin(s);
    const email = req.email.trim();
    if (email === "") throw invalidArgument("email is required");
    if (s.admins.some((a) => a.email.toLowerCase() === email.toLowerCase())) {
      throw invalidArgument("an admin with this email already exists");
    }
    const invite: AdminInviteState = {
      id: randomId(),
      email,
      token: `invite_demo_${randomId().slice(0, 8)}`,
      createdAt: now(),
      expiresAt: now() + 72 * 60 * 60 * 1000,
    };
    s.adminInvites.push(invite);
    audit(s, actor, "admin.invite", `Invited admin ${email}`, {
      targetType: "admin_invite",
      targetId: invite.id,
    });
    return {
      invite: inviteMsg(invite),
      inviteUrl: `${s.instanceBaseUrl}/admin/invite?token=${invite.token}`,
      emailed: s.smtp.source !== "none",
    };
  });

  handle(AdminAccountService.method.listAdminInvites, (s: InstanceSlice, _req) => {
    requireAdmin(s);
    return { invites: s.adminInvites.map(inviteMsg) };
  });

  handle(AdminAccountService.method.revokeAdminInvite, (s: InstanceSlice, req) => {
    const actor = requireAdmin(s);
    const invite = s.adminInvites.find((i) => i.id === req.id);
    if (!invite) throw notFound("invite");
    s.adminInvites = s.adminInvites.filter((i) => i.id !== req.id);
    audit(s, actor, "admin.invite.revoke", `Revoked admin invite for ${invite.email}`, {
      targetType: "admin_invite",
      targetId: invite.id,
    });
    return {};
  });

  handle(AdminAccountService.method.acceptAdminInvite, (s: InstanceSlice, req) => {
    // Unauthenticated by design: the invite token is the credential.
    const invite = s.adminInvites.find((i) => i.token === req.token || i.id === req.token);
    if (!invite) throw invalidArgument("this invite link is invalid or was revoked");
    if (invite.expiresAt <= now()) throw invalidArgument("this invite has expired");
    if (req.password === "") throw invalidArgument("password is required");
    if (req.password.length < 8) {
      throw invalidArgument("password must be at least 8 characters");
    }
    const admin: AdminState = { id: randomId(), email: invite.email, createdAt: now() };
    s.admins.push(admin);
    s.adminInvites = s.adminInvites.filter((i) => i.id !== invite.id);
    // Accepting signs the new admin in (the screen navigates straight to /).
    s.session.signedIn = true;
    s.session.adminId = admin.id;
    audit(s, admin, "admin.create", `${admin.email} accepted the admin invite`, {
      actorType: "system",
      actorId: "",
      actorLabel: "moth server",
      ip: "",
      targetType: "admin",
      targetId: admin.id,
    });
    return { admin: adminMsg(admin) };
  });

  handle(AdminAccountService.method.changePassword, (s: InstanceSlice, req) => {
    requireAdmin(s);
    if (req.currentPassword === "") throw invalidArgument("current password is required");
    if (req.newPassword === "") throw invalidArgument("new password is required");
    if (req.newPassword.length < 8) {
      throw invalidArgument("password must be at least 8 characters");
    }
    // The demo stores no password hashes; any current password is accepted.
    return {};
  });

  handle(AdminAccountService.method.createPersonalAccessToken, (s: InstanceSlice, req) => {
    const actor = requireAdmin(s);
    const name = req.name.trim();
    if (name === "") throw invalidArgument("name is required");
    if (req.expiresInDays < 0) throw invalidArgument("expires_in_days must be >= 0");
    const pat: PersonalAccessTokenState = {
      id: randomId(),
      name,
      createdAt: now(),
      lastUsedAt: null,
      expiresAt:
        req.expiresInDays === 0 ? null : now() + req.expiresInDays * 24 * 60 * 60 * 1000,
      revokedAt: null,
    };
    s.personalAccessTokens.push(pat);
    audit(s, actor, "pat.create", `Created access token “${name}”`, {
      targetType: "personal_access_token",
      targetId: pat.id,
    });
    return {
      // Obviously fake plaintext, returned exactly once; only metadata stays.
      token: `moth_pat_demo_${randomId().replace(/-/g, "").slice(0, 24)}`,
      metadata: patMsg(pat),
    };
  });

  handle(AdminAccountService.method.listPersonalAccessTokens, (s: InstanceSlice, _req) => {
    requireAdmin(s);
    const tokens = [...s.personalAccessTokens].sort((a, b) => b.createdAt - a.createdAt);
    return { tokens: tokens.map(patMsg) };
  });

  handle(AdminAccountService.method.revokePersonalAccessToken, (s: InstanceSlice, req) => {
    const actor = requireAdmin(s);
    const pat = s.personalAccessTokens.find((t) => t.id === req.id);
    if (!pat) throw notFound("personal access token");
    if (pat.revokedAt === null) {
      pat.revokedAt = now();
      audit(s, actor, "pat.revoke", `Revoked access token “${pat.name}”`, {
        targetType: "personal_access_token",
        targetId: pat.id,
      });
    }
    return {};
  });

  // --- InstanceSettingsService ---

  handle(InstanceSettingsService.method.getInstanceSettings, (s: InstanceSlice, _req) => {
    requireAdmin(s);
    return {
      baseUrl: s.instanceBaseUrl,
      version: "demo",
      smtp: smtpMsg(s.smtp),
      smtpSource: smtpSourceEnum(s.smtp.source),
      smtpHasPassword: s.smtp.hasPassword,
    };
  });

  handle(InstanceSettingsService.method.updateSmtpSettings, (s: InstanceSlice, req) => {
    const actor = requireAdmin(s);
    const smtp = req.smtp;
    if (!smtp) throw invalidArgument("smtp settings are required");
    if (smtp.host.trim() === "") {
      // An empty host clears the stored override; the demo has no config
      // file underneath, so email falls back to the console transport.
      s.smtp = { host: "", port: 587, username: "", from: "", source: "none", hasPassword: false };
      audit(s, actor, "settings.update", "Cleared the SMTP override", {
        targetType: "settings",
        targetId: "smtp",
      });
      return {
        smtp: smtpMsg(s.smtp),
        smtpSource: smtpSourceEnum(s.smtp.source),
        smtpHasPassword: s.smtp.hasPassword,
      };
    }
    if (smtp.port < 0 || smtp.port > 65535) throw invalidArgument("port must be 1-65535");
    s.smtp = {
      host: smtp.host.trim(),
      port: smtp.port || 587,
      username: smtp.username,
      from: smtp.from,
      source: "database",
      // An empty password keeps the stored one (write-only semantics).
      hasPassword: smtp.password !== "" ? true : s.smtp.hasPassword,
    };
    audit(s, actor, "settings.update", `Updated SMTP settings (${s.smtp.host})`, {
      targetType: "settings",
      targetId: "smtp",
    });
    return {
      smtp: smtpMsg(s.smtp),
      smtpSource: smtpSourceEnum(s.smtp.source),
      smtpHasPassword: s.smtp.hasPassword,
    };
  });

  handle(InstanceSettingsService.method.sendTestEmail, (s: InstanceSlice, req) => {
    requireAdmin(s);
    if (req.to.trim() === "") throw invalidArgument("recipient is required");
    // The demo has no outbox; sending always succeeds.
    return {};
  });

  // --- AuditService ---

  handle(AuditService.method.listAuditLog, (s: InstanceSlice, req) => {
    requireAdmin(s);
    let entries = [...s.auditEntries].sort((a, b) => b.createdAt - a.createdAt);
    if (req.projectId !== "") entries = entries.filter((e) => e.projectId === req.projectId);
    if (req.actorId !== "") entries = entries.filter((e) => e.actorId === req.actorId);
    if (req.action !== "") entries = entries.filter((e) => e.action === req.action);
    if (req.startTime) {
      const start = millisOf(req.startTime);
      entries = entries.filter((e) => e.createdAt >= start);
    }
    if (req.endTime) {
      const end = millisOf(req.endTime);
      entries = entries.filter((e) => e.createdAt < end);
    }
    const size = req.pageSize === 0 ? 50 : Math.min(Math.max(req.pageSize, 1), 200);
    // Page tokens are plain offsets into the filtered, newest-first list.
    const offset = req.pageToken === "" ? 0 : Number.parseInt(req.pageToken, 10);
    if (!Number.isFinite(offset) || offset < 0) throw invalidArgument("invalid page token");
    const page = entries.slice(offset, offset + size);
    return {
      entries: page.map((e) => ({
        id: e.id,
        actorType: e.actorType,
        actorId: e.actorId,
        actorLabel: e.actorLabel,
        action: e.action,
        targetType: e.targetType,
        targetId: e.targetId,
        projectId: e.projectId,
        summary: e.summary,
        beforeAfter: e.beforeAfter,
        ip: e.ip,
        createTime: ts(e.createdAt),
      })),
      nextPageToken: offset + size < entries.length ? String(offset + size) : "",
    };
  });
}
