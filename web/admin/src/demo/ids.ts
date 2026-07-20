// Shared identities of the seeded demo world. Every service slice seeds its
// own data, but cross-references (project ids, end-user ids, product ids…)
// come from here so the world stays consistent: the same person owns the
// device, the subscription and the audit trail.

// demoId mints a deterministic, UUID-shaped id. `group` tags the object
// family so seeded ids stay readable in the UI ("0197use0-…" is a user).
export function demoId(group: string, n: number): string {
  const g = (group + "0000").slice(0, 4).toLowerCase();
  return `0197${g}0-de30-7000-8000-${n.toString(16).padStart(12, "0")}`;
}

export const ADMIN = {
  id: demoId("admn", 1),
  email: "demo@moth.local",
};

// The fully configured flagship project: a journaling app with email +
// Google + Apple sign-in, a Pro subscription and push notifications.
export const PROJECT_MAIN = {
  id: demoId("proj", 1),
  name: "Aurora Journal",
  slug: "aurora",
  publishableKey: "pk_demo_aurora_k3v9q2m8w1x4",
};

// A young, barely configured second project so the projects list and the
// empty states are represented too.
export const PROJECT_SIDE = {
  id: demoId("proj", 2),
  name: "Skylark",
  slug: "skylark",
  publishableKey: "pk_demo_skylark_p7d2r5t8n0c3",
};

export interface DemoPerson {
  id: string;
  name: string;
  email: string;
  // Sign-in method, so the users slice and the analytics/audit slices tell
  // the same story.
  provider: "password" | "google" | "apple";
  // How many days ago the person signed up (spread over ~6 months).
  signupDaysAgo: number;
}

function person(
  n: number,
  name: string,
  email: string,
  provider: DemoPerson["provider"],
  signupDaysAgo: number,
): DemoPerson {
  return { id: demoId("user", n), name, email, provider, signupDaysAgo };
}

// End users of PROJECT_MAIN, newest last.
export const PEOPLE: DemoPerson[] = [
  person(1, "Maya Lindqvist", "maya.lindqvist@example.com", "google", 176),
  person(2, "Jonas Berg", "jonas.berg@example.com", "password", 171),
  person(3, "Amara Diallo", "amara.diallo@example.com", "apple", 163),
  person(4, "Tomás Ferreira", "tomas.ferreira@example.com", "password", 150),
  person(5, "Yuki Tanaka", "yuki.tanaka@example.com", "google", 141),
  person(6, "Claire Dubois", "claire.dubois@example.com", "password", 128),
  person(7, "Mikkel Sørensen", "mikkel.sorensen@example.com", "apple", 117),
  person(8, "Priya Raman", "priya.raman@example.com", "google", 104),
  person(9, "Diego Álvarez", "diego.alvarez@example.com", "password", 96),
  person(10, "Hannah Fischer", "hannah.fischer@example.com", "google", 88),
  person(11, "Liam O'Connell", "liam.oconnell@example.com", "apple", 74),
  person(12, "Zofia Nowak", "zofia.nowak@example.com", "password", 63),
  person(13, "Arthur Petit", "arthur.petit@example.com", "google", 55),
  person(14, "Nadia El-Sayed", "nadia.elsayed@example.com", "password", 47),
  person(15, "Felix Bauer", "felix.bauer@example.com", "apple", 38),
  person(16, "Ingrid Halvorsen", "ingrid.halvorsen@example.com", "google", 29),
  person(17, "Marco Ricci", "marco.ricci@example.com", "password", 21),
  person(18, "Aisha Khan", "aisha.khan@example.com", "google", 14),
  person(19, "Sven Eriksson", "sven.eriksson@example.com", "apple", 9),
  person(20, "Lucia Moreau", "lucia.moreau@example.com", "password", 6),
  person(21, "Omar Haddad", "omar.haddad@example.com", "google", 3),
  person(22, "Emma Vasquez", "emma.vasquez@example.com", "password", 1),
];

// Aurora Pro products and the entitlement they unlock.
export const PRODUCT_MONTHLY_ID = demoId("prod", 1);
export const PRODUCT_ANNUAL_ID = demoId("prod", 2);
export const PRODUCT_LIFETIME_ID = demoId("prod", 3);
export const ENTITLEMENT_PRO_ID = demoId("entl", 1);

// Who pays: active subscribers, one trialing, one churned, one lifetime
// grant. Slices that mention subscription state agree on these.
export const SUBSCRIBER_IDS = [
  PEOPLE[0].id, // Maya — annual
  PEOPLE[4].id, // Yuki — annual
  PEOPLE[7].id, // Priya — monthly
  PEOPLE[10].id, // Liam — monthly
  PEOPLE[15].id, // Ingrid — monthly
];
export const TRIAL_ID = PEOPLE[20].id; // Omar — in trial
export const CHURNED_ID = PEOPLE[5].id; // Claire — canceled
export const LIFETIME_ID = PEOPLE[2].id; // Amara — lifetime purchase

// People with a registered push device (a realistic ~60% opt-in).
export const PUSH_DEVICE_OWNER_IDS = [
  PEOPLE[0].id,
  PEOPLE[2].id,
  PEOPLE[3].id,
  PEOPLE[4].id,
  PEOPLE[6].id,
  PEOPLE[7].id,
  PEOPLE[9].id,
  PEOPLE[10].id,
  PEOPLE[13].id,
  PEOPLE[15].id,
  PEOPLE[17].id,
  PEOPLE[18].id,
  PEOPLE[20].id,
  PEOPLE[21].id,
];
