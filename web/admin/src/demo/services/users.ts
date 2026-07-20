// Demo implementations of moth.admin.v1.UserService and PushService: the
// end users of the seeded projects, their sessions and identities, the push
// device registry and the per-project push settings.

import { UserService } from "../../gen/moth/admin/v1/user_pb";
import {
  PushPermission,
  PushRevokeReason,
  PushService,
  PushTarget,
} from "../../gen/moth/admin/v1/push_pb";
import { demoId, PEOPLE, PROJECT_MAIN, PROJECT_SIDE, PUSH_DEVICE_OWNER_IDS } from "../ids";
import type { DemoPerson } from "../ids";
import { handle } from "../transport";
import { daysAgo, invalidArgument, minutesAgo, notFound, now, randomId, ts } from "../util";
import type { Millis } from "../util";

// ---------------------------------------------------------------------------
// State slice

export interface DemoUserSession {
  id: string;
  deviceInfo: string;
  createTime: Millis;
  expireTime: Millis;
}

export interface DemoIdentity {
  provider: string;
  email: string;
  createTime: Millis;
}

export interface DemoEndUser {
  id: string;
  projectId: string;
  email: string;
  emailVerified: boolean;
  displayName: string;
  disabled: boolean;
  createTime: Millis;
  updateTime: Millis;
  // Unset when the user never signed in.
  lastLoginTime?: Millis;
  providers: string[];
  customClaims: string;
  identities: DemoIdentity[];
  sessions: DemoUserSession[];
}

export interface DemoPushDeviceRecord {
  id: string;
  projectId: string;
  userId: string;
  target: PushTarget;
  deviceId: string;
  permission: PushPermission;
  platform: string;
  model: string;
  osVersion: string;
  appVersion: string;
  locale: string;
  createTime: Millis;
  updateTime: Millis;
  lastSeenTime: Millis;
  // Unset while the registration is active.
  revokeTime?: Millis;
  revokeReason: PushRevokeReason;
}

export interface DemoPushSettings {
  enabled: boolean;
  webpushVapidPublicKey: string;
}

export interface UsersSlice {
  endUsers: DemoEndUser[];
  pushDevices: DemoPushDeviceRecord[];
  // Keyed by project id; missing entries mean "never configured" (defaults).
  pushSettings: Record<string, DemoPushSettings>;
}

// ---------------------------------------------------------------------------
// Seed

const DAY = 24 * 60 * 60 * 1000;
const SESSION_LIFETIME = 30 * DAY;

// Diego Álvarez — not a subscriber, no push device — is the disabled account.
const DISABLED_INDEX = 8;
// Emma Vasquez signed up yesterday and has not verified her email yet.
const UNVERIFIED_INDEXES = [21];

// Last successful sign-in, minutes ago, per PEOPLE index. Mostly recent, a
// few dormant (Jonas, Zofia), the churned subscriber (Claire) two weeks out
// and the disabled account (Diego) forty days out.
const LAST_LOGIN_MINUTES: number[] = [
  12, // Maya
  30 * 24 * 60, // Jonas — dormant
  45, // Amara
  180, // Tomás
  25, // Yuki
  14 * 24 * 60, // Claire — churned
  26 * 60, // Mikkel
  8, // Priya
  40 * 24 * 60, // Diego — disabled
  300, // Hannah
  90, // Liam
  25 * 24 * 60, // Zofia — dormant
  2 * 24 * 60, // Arthur
  3 * 24 * 60, // Nadia
  12 * 60, // Felix
  360, // Ingrid
  24 * 60, // Marco
  30, // Aisha
  240, // Sven
  10 * 60, // Lucia
  55, // Omar
  20 * 60, // Emma
];

// Custom JWT claims for a couple of accounts so the claims editor has
// something to show.
const CUSTOM_CLAIMS: Record<number, string> = {
  0: '{"beta":true}',
  2: '{"role":"moderator"}',
};

// Session device names for people whose story names a device elsewhere.
const SESSION_DEVICE: Record<number, string> = {
  0: "iPhone 15 Pro · iOS 18.5",
  2: "iPhone 14 · iOS 18.4",
  3: "Pixel 8 · Android 15",
  4: "iPhone 16 · iOS 18.5",
  6: "iPhone 13 mini · iOS 18.3",
  7: "Samsung Galaxy S24 · Android 14",
  9: "Pixel 7a · Android 14",
  10: "iPhone 15 · iOS 18.5",
  13: "OnePlus 12 · Android 14",
  15: "iPhone 12 · iOS 18.2",
  17: "Pixel 9 · Android 15",
  18: "iPhone 16 Pro · iOS 18.5",
  20: "Firefox on Windows",
  21: "iPhone 15 Plus · iOS 18.5",
};

function seedPerson(index: number, person: DemoPerson): DemoEndUser {
  const signup = daysAgo(person.signupDaysAgo);
  const disabled = index === DISABLED_INDEX;
  const lastLogin = minutesAgo(LAST_LOGIN_MINUTES[index] ?? 60);
  const sessions: DemoUserSession[] = [];
  if (!disabled) {
    sessions.push({
      id: demoId("sess", index * 2 + 1),
      deviceInfo: SESSION_DEVICE[index] ?? "Chrome on macOS",
      createTime: lastLogin,
      expireTime: lastLogin + SESSION_LIFETIME,
    });
    // The two-device owners also carry a second live session.
    if (index === 0 || index === 7) {
      sessions.push({
        id: demoId("sess", index * 2 + 2),
        deviceInfo: index === 0 ? "Chrome on macOS" : "iPad Air · iPadOS 18.5",
        createTime: lastLogin - 2 * DAY,
        expireTime: lastLogin - 2 * DAY + SESSION_LIFETIME,
      });
    }
  }
  return {
    id: person.id,
    projectId: PROJECT_MAIN.id,
    email: person.email,
    emailVerified: !UNVERIFIED_INDEXES.includes(index),
    displayName: person.name,
    disabled,
    createTime: signup,
    updateTime: disabled ? daysAgo(38) : signup,
    lastLoginTime: lastLogin,
    providers: [person.provider],
    customClaims: CUSTOM_CLAIMS[index] ?? "",
    identities: [
      {
        provider: person.provider,
        email: person.provider === "password" ? "" : person.email,
        createTime: signup,
      },
    ],
    sessions,
  };
}

interface DeviceSeed {
  owner: number; // PEOPLE index
  target: PushTarget;
  platform: string;
  model: string;
  os: string;
  app: string;
  locale: string;
  permission: PushPermission;
  registeredDaysAgo: number;
  lastSeenMinutes: number;
  revoked?: PushRevokeReason;
}

// One or two registrations per PUSH_DEVICE_OWNER_IDS entry: a realistic mix
// of APNs / FCM / Web Push, one iOS provisional grant, one denied, and one
// revoked row (Priya's old iPad, unregistered on sign-out) so the user
// drawer shows the audit trail.
const DEVICE_SEEDS: DeviceSeed[] = [
  { owner: 0, target: PushTarget.APNS, platform: "ios", model: "iPhone 15 Pro", os: "18.5", app: "2.4.1+87", locale: "sv-SE", permission: PushPermission.GRANTED, registeredDaysAgo: 175, lastSeenMinutes: 12 },
  { owner: 0, target: PushTarget.WEBPUSH, platform: "web", model: "Chrome", os: "126", app: "2.4.0", locale: "sv-SE", permission: PushPermission.GRANTED, registeredDaysAgo: 88, lastSeenMinutes: 2 * 24 * 60 },
  { owner: 2, target: PushTarget.APNS, platform: "ios", model: "iPhone 14", os: "18.4", app: "2.4.1+87", locale: "fr-FR", permission: PushPermission.GRANTED, registeredDaysAgo: 162, lastSeenMinutes: 45 },
  { owner: 3, target: PushTarget.FCM, platform: "android", model: "Pixel 8", os: "15", app: "2.4.1+87", locale: "pt-PT", permission: PushPermission.GRANTED, registeredDaysAgo: 149, lastSeenMinutes: 180 },
  { owner: 4, target: PushTarget.APNS, platform: "ios", model: "iPhone 16", os: "18.5", app: "2.4.1+87", locale: "ja-JP", permission: PushPermission.PROVISIONAL, registeredDaysAgo: 140, lastSeenMinutes: 25 },
  { owner: 6, target: PushTarget.APNS, platform: "ios", model: "iPhone 13 mini", os: "18.3", app: "2.3.9+81", locale: "da-DK", permission: PushPermission.GRANTED, registeredDaysAgo: 116, lastSeenMinutes: 26 * 60 },
  { owner: 7, target: PushTarget.FCM, platform: "android", model: "Samsung Galaxy S24", os: "14", app: "2.4.1+87", locale: "en-IN", permission: PushPermission.GRANTED, registeredDaysAgo: 103, lastSeenMinutes: 8 },
  { owner: 7, target: PushTarget.APNS, platform: "ios", model: "iPad Air", os: "18.5", app: "2.4.0+85", locale: "en-IN", permission: PushPermission.GRANTED, registeredDaysAgo: 95, lastSeenMinutes: 12 * 24 * 60, revoked: PushRevokeReason.SIGNED_OUT },
  { owner: 9, target: PushTarget.FCM, platform: "android", model: "Pixel 7a", os: "14", app: "2.4.0+85", locale: "de-DE", permission: PushPermission.GRANTED, registeredDaysAgo: 87, lastSeenMinutes: 5 * 60 },
  { owner: 10, target: PushTarget.APNS, platform: "ios", model: "iPhone 15", os: "18.5", app: "2.4.1+87", locale: "en-IE", permission: PushPermission.GRANTED, registeredDaysAgo: 73, lastSeenMinutes: 90 },
  { owner: 13, target: PushTarget.FCM, platform: "android", model: "OnePlus 12", os: "14", app: "2.4.1+87", locale: "ar-EG", permission: PushPermission.DENIED, registeredDaysAgo: 46, lastSeenMinutes: 3 * 24 * 60 },
  { owner: 15, target: PushTarget.APNS, platform: "ios", model: "iPhone 12", os: "18.2", app: "2.4.1+87", locale: "nb-NO", permission: PushPermission.GRANTED, registeredDaysAgo: 28, lastSeenMinutes: 6 * 60 },
  { owner: 17, target: PushTarget.FCM, platform: "android", model: "Pixel 9", os: "15", app: "2.4.1+87", locale: "en-GB", permission: PushPermission.GRANTED, registeredDaysAgo: 13, lastSeenMinutes: 30 },
  { owner: 18, target: PushTarget.APNS, platform: "ios", model: "iPhone 16 Pro", os: "18.5", app: "2.4.1+87", locale: "sv-SE", permission: PushPermission.GRANTED, registeredDaysAgo: 8, lastSeenMinutes: 4 * 60 },
  { owner: 20, target: PushTarget.WEBPUSH, platform: "web", model: "Firefox", os: "128", app: "2.4.1", locale: "fr-FR", permission: PushPermission.GRANTED, registeredDaysAgo: 2, lastSeenMinutes: 55 },
  { owner: 21, target: PushTarget.APNS, platform: "ios", model: "iPhone 15 Plus", os: "18.5", app: "2.4.1+87", locale: "es-US", permission: PushPermission.GRANTED, registeredDaysAgo: 1, lastSeenMinutes: 20 * 60 },
];

// A deterministic, obviously-fake Web Push VAPID public key that still
// passes the SPA's shape check (base64url, 65 bytes, leading 0x04).
const DEMO_VAPID_PUBLIC_KEY =
  "BBQbIikwNz5FTFNaYWhvdn2Ei5KZoKeutbzDytHY3-bt9PsCCRAXHiUsMzpBSE9WXWRrcnmAh46VnKOqsbi_xs0";

function seedDevice(n: number, seedRow: DeviceSeed): DemoPushDeviceRecord {
  const person = PEOPLE[seedRow.owner];
  const created = daysAgo(seedRow.registeredDaysAgo);
  const lastSeen = minutesAgo(seedRow.lastSeenMinutes);
  const revoked = seedRow.revoked !== undefined;
  return {
    id: demoId("push", n),
    projectId: PROJECT_MAIN.id,
    userId: person.id,
    target: seedRow.target,
    deviceId: `ins_demo_${seedRow.platform}_${n.toString(16).padStart(4, "0")}`,
    permission: seedRow.permission,
    platform: seedRow.platform,
    model: seedRow.model,
    osVersion: seedRow.os,
    appVersion: seedRow.app,
    locale: seedRow.locale,
    createTime: created,
    updateTime: lastSeen,
    lastSeenTime: lastSeen,
    // Revoked rows were revoked when last seen (sign-out unregisters).
    revokeTime: revoked ? lastSeen : undefined,
    revokeReason: seedRow.revoked ?? PushRevokeReason.UNSPECIFIED,
  };
}

export function seedUsers(): UsersSlice {
  // PUSH_DEVICE_OWNER_IDS is the authority on who owns a device: the seed
  // table covers exactly that set, and the filter keeps it that way should
  // the two ever drift.
  const owners = new Set(PUSH_DEVICE_OWNER_IDS);
  return {
    // PROJECT_SIDE deliberately has zero users: its empty states show.
    endUsers: PEOPLE.map((person, index) => seedPerson(index, person)),
    pushDevices: DEVICE_SEEDS.filter((row) => owners.has(PEOPLE[row.owner].id)).map(
      (row, i) => seedDevice(i + 1, row),
    ),
    pushSettings: {
      [PROJECT_MAIN.id]: { enabled: true, webpushVapidPublicKey: DEMO_VAPID_PUBLIC_KEY },
      [PROJECT_SIDE.id]: { enabled: false, webpushVapidPublicKey: "" },
    },
  };
}

// ---------------------------------------------------------------------------
// Response shaping

function optTs(millis: Millis | undefined) {
  return millis === undefined ? undefined : ts(millis);
}

function toUser(u: DemoEndUser) {
  return {
    id: u.id,
    email: u.email,
    emailVerified: u.emailVerified,
    displayName: u.displayName,
    disabled: u.disabled,
    createTime: ts(u.createTime),
    providers: [...u.providers],
    lastLoginTime: optTs(u.lastLoginTime),
    updateTime: ts(u.updateTime),
    customClaims: u.customClaims,
  };
}

function toDevice(d: DemoPushDeviceRecord) {
  return {
    id: d.id,
    target: d.target,
    deviceId: d.deviceId,
    permission: d.permission,
    metadata: {
      platform: d.platform,
      model: d.model,
      osVersion: d.osVersion,
      appVersion: d.appVersion,
      locale: d.locale,
    },
    createTime: ts(d.createTime),
    updateTime: ts(d.updateTime),
    lastSeenTime: ts(d.lastSeenTime),
    revokeTime: optTs(d.revokeTime),
    revokeReason: d.revokeReason,
  };
}

function findUser(state: UsersSlice, projectId: string, userId: string): DemoEndUser {
  const user = state.endUsers.find((u) => u.projectId === projectId && u.id === userId);
  if (!user) throw notFound("user");
  return user;
}

function pageSizeOf(requested: number): number {
  if (requested <= 0) return 50;
  return Math.min(requested, 200);
}

function offsetOf(pageToken: string): number {
  if (pageToken === "") return 0;
  const offset = Number(pageToken);
  if (!Number.isInteger(offset) || offset < 0) throw invalidArgument("invalid page_token");
  return offset;
}

// vapidKeyValid mirrors the server-side shape check: base64url without
// padding decoding to a 65-byte uncompressed P-256 point. Empty is valid.
function vapidKeyValid(key: string): boolean {
  if (key === "") return true;
  if (!/^[A-Za-z0-9_-]+$/.test(key)) return false;
  let raw: string;
  try {
    raw = atob(key.replace(/-/g, "+").replace(/_/g, "/"));
  } catch {
    return false;
  }
  return raw.length === 65 && raw.charCodeAt(0) === 0x04;
}

// ---------------------------------------------------------------------------
// Handlers

export function registerUsers(): void {
  // ----- UserService -----

  handle(UserService.method.listUsers, (state: UsersSlice, req) => {
    if (req.projectId === "") throw invalidArgument("project_id is required");
    const q = req.query.trim().toLowerCase();
    const matches = state.endUsers
      .filter((u) => u.projectId === req.projectId)
      .filter(
        (u) =>
          q === "" ||
          u.email.toLowerCase().includes(q) ||
          u.displayName.toLowerCase().includes(q),
      )
      .sort((a, b) => b.createTime - a.createTime);
    const size = pageSizeOf(req.pageSize);
    const offset = offsetOf(req.pageToken);
    const page = matches.slice(offset, offset + size);
    const end = offset + page.length;
    return {
      users: page.map(toUser),
      nextPageToken: end < matches.length ? String(end) : "",
      totalSize: BigInt(matches.length),
    };
  });

  handle(UserService.method.getUser, (state: UsersSlice, req) => {
    const user = findUser(state, req.projectId, req.userId);
    return {
      user: toUser(user),
      sessions: user.sessions.map((s) => ({
        id: s.id,
        deviceInfo: s.deviceInfo,
        createTime: ts(s.createTime),
        expireTime: ts(s.expireTime),
      })),
      identities: user.identities.map((i) => ({
        provider: i.provider,
        email: i.email,
        createTime: ts(i.createTime),
      })),
    };
  });

  handle(UserService.method.createUser, (state: UsersSlice, req) => {
    if (req.projectId === "") throw invalidArgument("project_id is required");
    const email = req.email.trim().toLowerCase();
    if (email === "" || !email.includes("@")) throw invalidArgument("a valid email is required");
    if (req.password === "" && !req.sendInvite) {
      throw invalidArgument("set a password or send an invite");
    }
    const duplicate = state.endUsers.some(
      (u) => u.projectId === req.projectId && u.email === email,
    );
    if (duplicate) throw invalidArgument("a user with this email already exists");
    const created = now();
    const user: DemoEndUser = {
      id: randomId(),
      projectId: req.projectId,
      email,
      emailVerified: req.emailVerified,
      displayName: req.displayName.trim(),
      disabled: false,
      createTime: created,
      updateTime: created,
      lastLoginTime: undefined,
      providers: ["password"],
      customClaims: "",
      identities: [{ provider: "password", email: "", createTime: created }],
      sessions: [],
    };
    state.endUsers.push(user);
    return { user: toUser(user) };
  });

  handle(UserService.method.updateUser, (state: UsersSlice, req) => {
    const user = findUser(state, req.projectId, req.userId);
    const paths = req.updateMask?.paths ?? [];
    if (paths.length === 0) throw invalidArgument("update_mask is required");
    for (const path of paths) {
      switch (path) {
        case "display_name":
          user.displayName = req.user?.displayName ?? "";
          break;
        case "custom_claims":
          user.customClaims = req.user?.customClaims ?? "";
          break;
        default:
          throw invalidArgument(`unsupported update_mask path "${path}"`);
      }
    }
    user.updateTime = now();
    return { user: toUser(user) };
  });

  handle(UserService.method.disableUser, (state: UsersSlice, req) => {
    const user = findUser(state, req.projectId, req.userId);
    user.disabled = true;
    // Disabling revokes the user's refresh tokens.
    user.sessions = [];
    user.updateTime = now();
    return { user: toUser(user) };
  });

  handle(UserService.method.enableUser, (state: UsersSlice, req) => {
    const user = findUser(state, req.projectId, req.userId);
    user.disabled = false;
    user.updateTime = now();
    return { user: toUser(user) };
  });

  handle(UserService.method.deleteUser, (state: UsersSlice, req) => {
    const index = state.endUsers.findIndex(
      (u) => u.projectId === req.projectId && u.id === req.userId,
    );
    if (index === -1) throw notFound("user");
    const [removed] = state.endUsers.splice(index, 1);
    // The user's push registrations go with the account.
    state.pushDevices = state.pushDevices.filter(
      (d) => !(d.projectId === req.projectId && d.userId === removed.id),
    );
    return {};
  });

  handle(UserService.method.revokeUserSessions, (state: UsersSlice, req) => {
    const user = findUser(state, req.projectId, req.userId);
    const revoked = user.sessions.length;
    user.sessions = [];
    user.updateTime = now();
    return { revokedCount: BigInt(revoked) };
  });

  handle(UserService.method.sendPasswordReset, (state: UsersSlice, req) => {
    findUser(state, req.projectId, req.userId);
    // The demo has no mailer; the send simply succeeds.
    return {};
  });

  // ----- PushService -----

  handle(PushService.method.getPushSettings, (state: UsersSlice, req) => {
    if (req.projectId === "") throw invalidArgument("project_id is required");
    const settings = state.pushSettings[req.projectId];
    return {
      settings: settings ?? { enabled: false, webpushVapidPublicKey: "" },
    };
  });

  handle(PushService.method.updatePushSettings, (state: UsersSlice, req) => {
    if (req.projectId === "") throw invalidArgument("project_id is required");
    const next: DemoPushSettings = {
      enabled: req.settings?.enabled ?? false,
      webpushVapidPublicKey: (req.settings?.webpushVapidPublicKey ?? "").trim(),
    };
    if (!vapidKeyValid(next.webpushVapidPublicKey)) {
      throw invalidArgument(
        "webpush_vapid_public_key must be a base64url uncompressed P-256 public key",
      );
    }
    state.pushSettings[req.projectId] = next;
    return { settings: next };
  });

  handle(PushService.method.listUserPushDevices, (state: UsersSlice, req) => {
    findUser(state, req.projectId, req.userId);
    const devices = state.pushDevices
      .filter((d) => d.projectId === req.projectId && d.userId === req.userId)
      .sort((a, b) => b.lastSeenTime - a.lastSeenTime);
    return { devices: devices.map(toDevice) };
  });

  handle(PushService.method.listPushDevices, (state: UsersSlice, req) => {
    if (req.projectId === "") throw invalidArgument("project_id is required");
    const active = state.pushDevices.filter(
      (d) => d.projectId === req.projectId && d.revokeTime === undefined,
    );
    const matches = active
      .filter((d) => req.target === PushTarget.UNSPECIFIED || d.target === req.target)
      .sort((a, b) => b.createTime - a.createTime);
    const size = pageSizeOf(req.pageSize);
    const offset = offsetOf(req.pageToken);
    const page = matches.slice(offset, offset + size);
    const end = offset + page.length;
    const byUser = new Map(state.endUsers.map((u) => [u.id, u.email]));
    const count = (target: PushTarget) =>
      BigInt(active.filter((d) => d.target === target).length);
    return {
      devices: page.map((d) => ({
        device: toDevice(d),
        userId: d.userId,
        userEmail: byUser.get(d.userId) ?? "",
      })),
      nextPageToken: end < matches.length ? String(end) : "",
      apnsCount: count(PushTarget.APNS),
      fcmCount: count(PushTarget.FCM),
      webpushCount: count(PushTarget.WEBPUSH),
    };
  });

  handle(PushService.method.revokePushDevice, (state: UsersSlice, req) => {
    const device = state.pushDevices.find(
      (d) => d.projectId === req.projectId && d.id === req.pushDeviceId,
    );
    if (!device) throw notFound("push device");
    // Idempotent: an already-revoked registration keeps its original reason.
    if (device.revokeTime === undefined) {
      device.revokeTime = now();
      device.revokeReason = PushRevokeReason.ADMIN;
      device.updateTime = device.revokeTime;
    }
    return { device: toDevice(device) };
  });
}
