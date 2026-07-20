import { createClient } from "@connectrpc/connect";
import { useState, type CSSProperties, type ReactNode } from "react";
import { Link, useNavigate } from "react-router";

import { errorMessage, invalidate, transport } from "../api";
import {
  AppleCredentialFields,
  GoogleCredentialFields,
  emptyAppleDraft,
  emptyGoogleDraft,
  type AppleDraft,
  type GoogleDraft,
} from "../components/providerFields";
import { CodeBlock, ErrorNote, Field, KeyWell, StringListField } from "../components/ui";
import { CopyScreen, CopyService } from "../gen/moth/admin/v1/copy_pb";
import { EntitlementService } from "../gen/moth/admin/v1/entitlement_pb";
import { ProfilePlatform, ProfileService } from "../gen/moth/admin/v1/profile_pb";
import { ProductService } from "../gen/moth/admin/v1/product_pb";
import type { Project } from "../gen/moth/admin/v1/project_pb";
import { ProjectService } from "../gen/moth/admin/v1/project_pb";
import { PushService } from "../gen/moth/admin/v1/push_pb";
import { LogoVariant, ThemeService } from "../gen/moth/admin/v1/theme_pb";
import { vapidKeyError } from "../lib/push";
import {
  MAX_LOGO_BYTES,
  bestOn,
  deriveDark,
  editorFromProto,
  ensurePreviewFonts,
  fontStack,
  isHexColor,
  normalizeHex,
  type Palette,
} from "../lib/theme";

// ProjectCreate is the milestone-22 guided creation wizard: a full-screen
// stepped flow that asks what the app needs (platforms, sign-in,
// monetization, push, branding) and composes the same per-domain admin RPCs
// the tabs call. All state is client-side until "Review & create" — nothing
// exists server-side before then, so abandoning the wizard creates nothing.
// After CreateProject the follow-up writes run sequentially; a failure never
// rolls the project back, it is surfaced honestly on the keys screen and the
// remaining gap lands on the overview checklist.

// slugify mirrors the server's Slugify (lowercase, [a-z0-9-]).
function slugify(name: string): string {
  return name
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

const SLUG_RE = /^[a-z0-9]+(-[a-z0-9]+)*$/;

const STEPS = [
  { id: "basics", label: "Basics" },
  { id: "signin", label: "Sign-in" },
  { id: "monetization", label: "Monetization" },
  { id: "push", label: "Push notifications" },
  { id: "branding", label: "Branding" },
  { id: "review", label: "Review & create" },
] as const;

type StepID = (typeof STEPS)[number]["id"];

const PLATFORMS: { value: ProfilePlatform; label: string }[] = [
  { value: ProfilePlatform.IOS, label: "iOS" },
  { value: ProfilePlatform.ANDROID, label: "Android" },
  { value: ProfilePlatform.WEB, label: "Web" },
];

// The bundled locales beyond English (internal/i18n.BundledLocales), offered
// as the branding step's language checklist.
const EXTRA_LOCALES: { tag: string; name: string }[] = [
  { tag: "fr", name: "French" },
  { tag: "de", name: "German" },
  { tag: "es", name: "Spanish" },
  { tag: "pt", name: "Portuguese" },
  { tag: "it", name: "Italian" },
  { tag: "ja", name: "Japanese" },
];

type TierDraft = {
  identifier: string;
  displayName: string;
  price: string;
  currency: string;
  billingPeriod: string;
  appleId: string;
  googleId: string;
};

const emptyTier: TierDraft = {
  identifier: "",
  displayName: "",
  price: "",
  currency: "",
  billingPeriod: "monthly",
  appleId: "",
  googleId: "",
};

type WriteFailure = { label: string; tab: string; message: string };

type Done = { project: Project; secretKey: string; failures: WriteFailure[] };

export function ProjectCreate() {
  ensurePreviewFonts();
  const navigate = useNavigate();

  // ---------- Client-side wizard state (nothing is written until Review) ----------

  // Basics
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [slugEdited, setSlugEdited] = useState(false);
  const [platforms, setPlatforms] = useState<ProfilePlatform[]>([]);

  // Sign-in
  const [publicSignup, setPublicSignup] = useState(true);
  const [requireVerification, setRequireVerification] = useState(false);
  const [minLen, setMinLen] = useState("8");
  const [googleEnabled, setGoogleEnabled] = useState(false);
  const [googleDefer, setGoogleDefer] = useState(false);
  const [google, setGoogle] = useState<GoogleDraft>(emptyGoogleDraft);
  const [appleEnabled, setAppleEnabled] = useState(false);
  const [appleDefer, setAppleDefer] = useState(false);
  const [apple, setApple] = useState<AppleDraft>(emptyAppleDraft);
  const [redirectOrigins, setRedirectOrigins] = useState<string[]>([]);

  // Monetization
  const [sells, setSells] = useState(false);
  const [entIdentifier, setEntIdentifier] = useState("");
  const [entDisplayName, setEntDisplayName] = useState("");
  const [tiers, setTiers] = useState<TierDraft[]>([]);

  // Push
  const [sendsPushes, setSendsPushes] = useState(false);
  const [vapidNow, setVapidNow] = useState(false);
  const [vapidKey, setVapidKey] = useState("");

  // Branding
  const defaults = editorFromProto(undefined);
  const [brandColor, setBrandColor] = useState(defaults.colors.primary);
  const [brandDraft, setBrandDraft] = useState(defaults.colors.primary);
  const [pref, setPref] = useState<"light" | "dark">("light");
  const [logo, setLogo] = useState<{ file: File; url: string } | null>(null);
  const [logoError, setLogoError] = useState("");
  const [locales, setLocales] = useState<string[]>([]);

  // Flow
  const [step, setStep] = useState(0);
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState("");
  const [done, setDone] = useState<Done>();

  // ---------- Derived ----------

  const hasIos = platforms.includes(ProfilePlatform.IOS);
  const hasAndroid = platforms.includes(ProfilePlatform.ANDROID);
  const hasWeb = platforms.includes(ProfilePlatform.WEB);

  const effectiveSlug = slugEdited ? slug : slugify(name);
  const nameOk = name.trim() !== "";
  const slugOk = effectiveSlug !== "" && SLUG_RE.test(effectiveSlug);
  const basicsOk = nameOk && slugOk && platforms.length > 0;

  const minLenNum = parseInt(minLen, 10) || 8;
  const vapidErr = vapidNow ? vapidKeyError(vapidKey.trim()) : "";
  const tiersOk =
    !sells ||
    tiers.length === 0 ||
    (entIdentifier.trim() !== "" && tiers.every((t) => t.identifier.trim() !== ""));
  const brandOk = isHexColor(brandColor);

  const current: StepID = STEPS[step].id;
  const stepOk =
    current === "basics"
      ? basicsOk
      : current === "push"
        ? vapidErr === ""
        : current === "monetization"
          ? tiersOk
          : current === "branding"
            ? brandOk && logoError === ""
            : true;

  // Whether the sign-in step changed anything worth writing.
  const settingsTouched =
    googleEnabled ||
    appleEnabled ||
    !publicSignup ||
    requireVerification ||
    minLenNum !== 8 ||
    redirectOrigins.length > 0;
  const brandTouched = brandColor !== defaults.colors.primary || pref === "dark";

  function togglePlatform(p: ProfilePlatform) {
    setPlatforms((cur) => (cur.includes(p) ? cur.filter((x) => x !== p) : [...cur, p]));
  }

  function toggleLocale(tag: string) {
    setLocales((cur) => (cur.includes(tag) ? cur.filter((x) => x !== tag) : [...cur, tag]));
  }

  function onLogoFile(file: File) {
    setLogoError("");
    if (file.type !== "image/png" && file.type !== "image/svg+xml") {
      setLogoError("PNG or SVG only.");
      return;
    }
    if (file.size > MAX_LOGO_BYTES) {
      setLogoError("File is too large — the limit is 512 KiB.");
      return;
    }
    if (logo) URL.revokeObjectURL(logo.url);
    setLogo({ file, url: URL.createObjectURL(file) });
  }

  // The seeded light palette: defaults with the brand color as primary and
  // its AA-safe "on" counterpart. A dark preference makes the seeded palette
  // itself the derived dark one (and pins darkColors to it), so the app
  // reads dark in both schemes.
  const lightSeed: Palette = {
    ...defaults.colors,
    primary: brandColor,
    onPrimary: bestOn(brandColor),
  };
  const seededPalette: Palette = pref === "dark" ? deriveDark(lightSeed) : lightSeed;

  // ---------- The create sequence (Review & create) ----------

  async function createAll() {
    setCreating(true);
    setCreateError("");

    const projects = createClient(ProjectService, transport);
    let project: Project;
    let secretKey: string;
    try {
      const resp = await projects.createProject({ name: name.trim(), slug: effectiveSlug });
      if (!resp.project) throw new Error("the server returned no project");
      project = resp.project;
      secretKey = resp.secretKey;
    } catch (err) {
      setCreateError(errorMessage(err));
      setCreating(false);
      return;
    }
    invalidate(ProjectService.method.listProjects);

    // The project exists from here on; follow-up writes reuse the existing
    // per-domain RPCs sequentially and never roll it back. Failures are
    // collected and surfaced on the keys screen (and the gap stays visible
    // on the overview checklist).
    const failures: WriteFailure[] = [];
    const attempt = async (label: string, tab: string, fn: () => Promise<unknown>) => {
      try {
        await fn();
      } catch (err) {
        failures.push({ label, tab, message: errorMessage(err) });
      }
    };
    const id = project.id;

    if (settingsTouched) {
      await attempt("Sign-in settings & providers", "providers", () =>
        projects.updateProject({
          id,
          settings: {
            passwordMinLength: minLenNum,
            requireEmailVerification: requireVerification,
            allowPublicSignup: publicSignup,
            enumerationSafeSignup: false,
            accessTokenTtlSeconds: 900,
            refreshTokenTtlDays: 30,
            // Deferred providers are written disabled with no credentials:
            // the server rejects enabling a provider without them (which
            // would take down this whole settings write), and the checklist
            // derives the "finish it" item from the profile's intent, not
            // from the provider flag. `moth setup google|apple` (or the
            // Providers tab) enables it later.
            google: {
              enabled: googleEnabled && !googleDefer,
              webClientId: googleDefer ? "" : google.webClientId.trim(),
              iosClientId: googleDefer ? "" : google.iosClientId.trim(),
              androidClientId: googleDefer ? "" : google.androidClientId.trim(),
              webClientSecret: googleDefer ? "" : google.webClientSecret,
            },
            apple: {
              enabled: appleEnabled && !appleDefer,
              servicesId: appleDefer ? "" : apple.servicesId.trim(),
              teamId: appleDefer ? "" : apple.teamId.trim(),
              keyId: appleDefer ? "" : apple.keyId.trim(),
              privateKeyP8: appleDefer ? "" : apple.privateKeyP8.trim(),
              bundleIds: appleDefer ? [] : apple.bundleIds,
            },
            autoLinkVerifiedEmail: true,
            redirectSchemes: [],
            redirectOrigins,
            signupEmailAllowlist: [],
            signupEmailBlocklist: [],
            captchaVerifyUrl: "",
          },
          updateMask: { paths: ["settings"] },
        }),
      );
    }

    if (sells && entIdentifier.trim() !== "") {
      const entitlements = createClient(EntitlementService, transport);
      const products = createClient(ProductService, transport);
      let entId = "";
      try {
        const r = await entitlements.createEntitlement({
          projectId: id,
          identifier: entIdentifier.trim(),
          displayName: entDisplayName.trim(),
        });
        entId = r.entitlement?.id ?? "";
      } catch (err) {
        failures.push({ label: "Entitlement", tab: "monetization", message: errorMessage(err) });
      }
      if (entId !== "") {
        for (const [i, t] of tiers.entries()) {
          await attempt(`Tier "${t.identifier.trim()}"`, "monetization", () =>
            products.createProduct({
              projectId: id,
              product: {
                identifier: t.identifier.trim(),
                displayName: t.displayName.trim(),
                appleProductId: hasIos ? t.appleId.trim() : "",
                googleProductId: hasAndroid ? t.googleId.trim() : "",
                stripePriceId: "",
                stripeProductId: "",
                billingPeriod: t.billingPeriod.trim(),
                priceAmountMicros:
                  t.price.trim() === ""
                    ? 0n
                    : BigInt(Math.round(parseFloat(t.price) * 1_000_000)),
                currency: t.currency.trim().toUpperCase(),
                trialPeriod: "",
                offering: "default",
                sortOrder: i,
                entitlementIds: [entId],
              },
            }),
          );
        }
      }
    }

    if (sendsPushes) {
      const push = createClient(PushService, transport);
      await attempt("Push settings", "settings", () =>
        push.updatePushSettings({
          projectId: id,
          settings: {
            enabled: true,
            webpushVapidPublicKey: vapidNow && vapidErr === "" ? vapidKey.trim() : "",
          },
        }),
      );
    }

    const themes = createClient(ThemeService, transport);
    if (brandTouched) {
      await attempt("Theme", "design", () =>
        themes.updateTheme({
          projectId: id,
          theme: {
            colors: { ...seededPalette },
            darkColors: pref === "dark" ? { ...seededPalette } : undefined,
            typography: { fontFamily: defaults.fontFamily, scale: defaults.scale },
            spacing: { unit: defaults.spacingUnit },
            shape: { cornerRadius: defaults.cornerRadius },
            legal: { termsUrl: "", privacyUrl: "" },
          },
        }),
      );
    }
    if (logo) {
      await attempt("Logo", "design", async () => {
        const data = new Uint8Array(await logo.file.arrayBuffer());
        return themes.uploadLogo({
          projectId: id,
          variant: LogoVariant.LIGHT,
          data,
          contentType: logo.file.type,
        });
      });
    }

    if (locales.length > 0) {
      // Seeding a language marks it customized in the copy editor by pinning
      // one key (the sign-in title) to its own bundled default — nothing
      // user-visible changes, the language just becomes part of the
      // project's copy document.
      const copy = createClient(CopyService, transport);
      for (const tag of locales) {
        await attempt(`Language "${tag}"`, "design", async () => {
          const doc = await copy.getProjectCopy({
            projectId: id,
            locale: tag,
            screen: CopyScreen.SIGN_IN,
          });
          const title = doc.keys.find((k) => k.key === "sign_in.title");
          if (!title || title.defaultValue === "") return;
          return copy.updateProjectCopy({
            projectId: id,
            locale: tag,
            values: { "sign_in.title": title.defaultValue },
          });
        });
      }
    }

    // The profile is the wizard's answers — written last so the checklist
    // derives from whatever configuration actually landed above.
    const profiles = createClient(ProfileService, transport);
    await attempt("Setup profile", "settings", () =>
      profiles.updateProfile({
        projectId: id,
        profile: {
          platforms,
          googleSignIn: googleEnabled,
          appleSignIn: appleEnabled,
          sellsSubscriptions: sells,
          sendsPushes,
          checklistDismissed: false,
        },
      }),
    );

    setDone({ project, secretKey, failures });
    setCreating(false);
  }

  // ---------- The keys finale (shown once, then the setup tab) ----------

  if (done) {
    return (
      <main className="page" style={{ maxWidth: 720 }}>
        <div className="stack-8">
          <h1>{done.project.name} is ready</h1>
          <span className="mono text-secondary">{done.project.slug}</span>
        </div>
        <div className="stack-24">
          <section className="card card--pad stack-16">
            <div className="stack-8">
              <span className="field__label">Publishable key (for the app)</span>
              <KeyWell value={done.project.publishableKey} />
            </div>
            <div className="stack-8">
              <span className="field__label">Secret key (for your backend)</span>
              <KeyWell value={done.secretKey} secret />
              <p className="caption">
                You won't see this key again. Store it in your backend's secret
                manager.
              </p>
            </div>
          </section>

          {done.failures.length > 0 && (
            <section className="card card--pad stack-12">
              <h3 className="card__title">Some setup did not land</h3>
              <p className="caption">
                The project was created, but these follow-up writes failed.
                Finish each one in its tab — the overview checklist keeps track.
              </p>
              {done.failures.map((f) => (
                <div key={f.label} className="stack-8" style={{ gap: 2 }}>
                  <span className="body-strong">
                    {f.label} —{" "}
                    <Link to={`/projects/${done.project.id}/${f.tab}`}>open the {f.tab} tab</Link>
                  </span>
                  <span className="field__error">{f.message}</span>
                </div>
              ))}
            </section>
          )}

          <div className="row-12">
            <button
              type="button"
              className="btn btn--primary"
              onClick={() => void navigate(`/projects/${done.project.id}/setup`)}
            >
              Continue to setup
            </button>
          </div>
        </div>
      </main>
    );
  }

  // ---------- The stepped flow ----------

  const back = () => setStep((s) => Math.max(0, s - 1));
  const next = () => setStep((s) => Math.min(STEPS.length - 1, s + 1));
  const toReview = () => setStep(STEPS.length - 1);

  return (
    <main className="page">
      <div className="stack-8">
        <h1>Create project</h1>
        <span className="caption">
          Step {step + 1} of {STEPS.length} · {STEPS[step].label}
          {step > 0 && step < STEPS.length - 1 ? " — every step from here is skippable" : ""}
        </span>
      </div>

      <div className="stack-24">
        {current === "basics" && (
          <BasicsStep
            name={name}
            onName={(v) => setName(v)}
            slug={effectiveSlug}
            slugOk={slugOk}
            onSlug={(v) => {
              setSlugEdited(true);
              setSlug(v);
            }}
            platforms={platforms}
            onTogglePlatform={togglePlatform}
          />
        )}

        {current === "signin" && (
          <div className="stack-24" style={{ maxWidth: 720 }}>
            <section className="card card--pad stack-16">
              <h3 className="card__title">Email &amp; password</h3>
              <p className="caption">
                Always on. These are the project defaults — change them any time
                under Settings.
              </p>
              <label className="check">
                <input
                  type="checkbox"
                  checked={publicSignup}
                  onChange={(e) => setPublicSignup(e.target.checked)}
                />
                <span>
                  Open sign-up
                  <span className="caption" style={{ display: "block" }}>
                    Anyone can create an account from the app. Off = invite-only.
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
            </section>

            <ProviderStepCard
              title="Sign in with Google"
              enabled={googleEnabled}
              onEnabled={setGoogleEnabled}
              defer={googleDefer}
              onDefer={setGoogleDefer}
              deferHint={`moth setup google --project ${effectiveSlug}`}
            >
              <GoogleCredentialFields
                draft={google}
                onChange={setGoogle}
                showIos={hasIos}
                showAndroid={hasAndroid}
              />
              {hasAndroid && !hasWeb && (
                <p className="caption">
                  Google's Android sign-in issues ID tokens with the{" "}
                  <em>web</em> client ID as audience, so the web client is
                  needed even for Android-only apps.
                </p>
              )}
            </ProviderStepCard>

            <ProviderStepCard
              title="Sign in with Apple"
              enabled={appleEnabled}
              onEnabled={setAppleEnabled}
              defer={appleDefer}
              onDefer={setAppleDefer}
              deferHint={`moth setup apple --project ${effectiveSlug}`}
            >
              <AppleCredentialFields
                draft={apple}
                onChange={setApple}
                showNative={hasIos}
                showWeb={hasWeb || hasAndroid}
              />
            </ProviderStepCard>

            {hasWeb && (googleEnabled || appleEnabled) && (
              <section className="card card--pad stack-16">
                <h3 className="card__title">Web redirects</h3>
                <StringListField
                  label="Redirect origins (web)"
                  values={redirectOrigins}
                  onChange={setRedirectOrigins}
                  placeholder="https://app.example.com"
                  help="Where the browser SDK may receive the sign-in code; exact origin match. Editable later under Providers."
                />
              </section>
            )}
          </div>
        )}

        {current === "monetization" && (
          <div className="stack-24" style={{ maxWidth: 720 }}>
            <section className="card card--pad stack-16">
              <h3 className="card__title">Does this app sell subscriptions?</h3>
              <YesNo value={sells} onChange={setSells} labelledBy="Sells subscriptions" />
              {!sells && (
                <p className="caption">
                  No — nothing to configure. Every user always has a valid free
                  state; you can add tiers later under Monetization.
                </p>
              )}
            </section>

            {sells && (
              <>
                <section className="card card--pad stack-16">
                  <h3 className="card__title">First entitlement</h3>
                  <p className="caption">
                    The capability your app gates on (e.g.{" "}
                    <span className="inline-code">pro</span>). Leave blank to
                    define the catalog later under Monetization.
                  </p>
                  <div className="row-16" style={{ alignItems: "flex-start" }}>
                    <div style={{ flex: 1 }}>
                      <Field label="Identifier" help='Lowercase, stable, e.g. "pro".'>
                        <input
                          className="input input--mono"
                          value={entIdentifier}
                          onChange={(e) => setEntIdentifier(e.target.value)}
                          placeholder="pro"
                          spellCheck={false}
                        />
                      </Field>
                    </div>
                    <div style={{ flex: 1 }}>
                      <Field label="Display name">
                        <input
                          className="input"
                          value={entDisplayName}
                          onChange={(e) => setEntDisplayName(e.target.value)}
                          placeholder="Pro"
                        />
                      </Field>
                    </div>
                  </div>
                </section>

                <section className="card card--pad stack-16">
                  <div className="page__header">
                    <h3 className="card__title">Tiers</h3>
                    <button
                      type="button"
                      className="btn btn--secondary btn--compact"
                      onClick={() => setTiers((t) => [...t, { ...emptyTier }])}
                    >
                      Add tier
                    </button>
                  </div>
                  <p className="caption">
                    Your platforms imply the stores:{" "}
                    {[
                      hasIos ? "App Store" : "",
                      hasAndroid ? "Google Play" : "",
                      hasWeb ? "Stripe" : "",
                    ]
                      .filter(Boolean)
                      .join(", ")}
                    . Price and period are display metadata — the store read
                    stays authoritative.
                  </p>
                  {tiers.length === 0 && (
                    <p className="caption">No tiers yet — add one, or define them later.</p>
                  )}
                  {tiers.map((t, i) => (
                    <TierRow
                      key={i}
                      tier={t}
                      index={i}
                      showApple={hasIos}
                      showGoogle={hasAndroid}
                      onChange={(nt) =>
                        setTiers((cur) => cur.map((x, j) => (j === i ? nt : x)))
                      }
                      onRemove={() => setTiers((cur) => cur.filter((_, j) => j !== i))}
                    />
                  ))}
                  {sells && tiers.length > 0 && entIdentifier.trim() === "" && (
                    <p className="field__error">
                      Tiers need the entitlement above — give it an identifier.
                    </p>
                  )}
                  <p className="caption">
                    Store credentials and catalog sync are deferred on purpose —
                    they take a console round-trip. The overview checklist will
                    point at the Monetization tab (or{" "}
                    <span className="inline-code">moth setup billing --project {effectiveSlug}</span>
                    ) to finish them.
                  </p>
                </section>
              </>
            )}
          </div>
        )}

        {current === "push" && (
          <div className="stack-24" style={{ maxWidth: 720 }}>
            <section className="card card--pad stack-16">
              <h3 className="card__title">Will your backend send pushes?</h3>
              <p className="caption">
                moth registers devices; your backend sends. Yes enables push
                registration for signed-in devices.
              </p>
              <YesNo value={sendsPushes} onChange={setSendsPushes} labelledBy="Sends pushes" />
            </section>

            {sendsPushes && hasWeb && (
              <section className="card card--pad stack-16">
                <h3 className="card__title">Web Push</h3>
                <label className="check">
                  <input
                    type="checkbox"
                    checked={vapidNow}
                    onChange={(e) => setVapidNow(e.target.checked)}
                  />
                  <span>
                    I have a VAPID keypair — paste the public key now
                    <span className="caption" style={{ display: "block" }}>
                      Off = decide later; the checklist will point at the Push tab.
                    </span>
                  </span>
                </label>
                {vapidNow ? (
                  <Field
                    label="Web Push VAPID public key"
                    error={vapidErr}
                    help="The public half of your VAPID keypair (base64url). Keep the private key in your sender — it never touches moth."
                  >
                    <input
                      className={vapidErr ? "input input--mono input--error" : "input input--mono"}
                      value={vapidKey}
                      onChange={(e) => setVapidKey(e.target.value)}
                      placeholder="BPz3…"
                      spellCheck={false}
                      autoComplete="off"
                    />
                  </Field>
                ) : (
                  <>
                    <p className="caption">Generate a keypair when you're ready:</p>
                    <CodeBlock code="npx web-push generate-vapid-keys" />
                  </>
                )}
              </section>
            )}

            {sendsPushes && hasAndroid && (
              <section className="card card--pad stack-12">
                <h3 className="card__title">Android</h3>
                <p className="caption">
                  FCM needs your app's own Firebase config
                  (google-services.json) — the one piece of setup moth cannot
                  absorb. The setup tab walks through it.
                </p>
              </section>
            )}

            {sendsPushes && hasIos && (
              <section className="card card--pad stack-12">
                <h3 className="card__title">iOS</h3>
                <p className="caption">
                  Nothing to paste — add the Push Notifications capability in
                  Xcode and the SDK handles registration.
                </p>
              </section>
            )}
          </div>
        )}

        {current === "branding" && (
          <div className="design">
            <div className="stack-24">
              <section className="card card--pad stack-16">
                <h3 className="card__title">Brand</h3>
                <p className="caption">
                  Seeds the project theme — a few tokens, not the whole editor.
                  Refine everything later under Design.
                </p>
                <div className="field">
                  <span className="field__label">Brand color</span>
                  <div className="row-8">
                    <input
                      type="color"
                      className="design__swatch"
                      aria-label="Brand color picker"
                      value={isHexColor(brandColor) ? brandColor : defaults.colors.primary}
                      onChange={(e) => {
                        const v = normalizeHex(e.target.value);
                        setBrandColor(v);
                        setBrandDraft(v);
                      }}
                    />
                    <input
                      className={
                        isHexColor(brandDraft)
                          ? "input input--mono"
                          : "input input--mono input--error"
                      }
                      style={{ flex: 1, minWidth: 0 }}
                      aria-label="Brand color"
                      value={brandDraft}
                      spellCheck={false}
                      onChange={(e) => {
                        setBrandDraft(e.target.value);
                        if (isHexColor(e.target.value)) {
                          setBrandColor(normalizeHex(e.target.value));
                        }
                      }}
                    />
                  </div>
                </div>
                <div className="field">
                  <span className="field__label">Appearance</span>
                  <div className="seg" role="group" aria-label="Appearance">
                    <button
                      type="button"
                      className="seg__btn"
                      aria-pressed={pref === "light"}
                      onClick={() => setPref("light")}
                    >
                      Light
                    </button>
                    <button
                      type="button"
                      className="seg__btn"
                      aria-pressed={pref === "dark"}
                      onClick={() => setPref("dark")}
                    >
                      Dark
                    </button>
                  </div>
                  <span className="field__help">
                    Dark seeds a dark palette as the app's look; either way the
                    theme stays fully editable.
                  </span>
                </div>
                <LogoPicker logo={logo} error={logoError} onFile={onLogoFile} />
              </section>

              <section className="card card--pad stack-16">
                <h3 className="card__title">Languages</h3>
                <p className="caption">
                  English is always available. Ticking a language adds it to the
                  project's copy document so its translations are ready to edit
                  under Design.
                </p>
                {EXTRA_LOCALES.map((l) => (
                  <label className="check" key={l.tag}>
                    <input
                      type="checkbox"
                      checked={locales.includes(l.tag)}
                      onChange={() => toggleLocale(l.tag)}
                    />
                    <span>
                      {l.name} <span className="mono text-tertiary">{l.tag}</span>
                    </span>
                  </label>
                ))}
              </section>
            </div>

            <div className="design__preview">
              <WizardPreview
                appName={name.trim() || "Your app"}
                palette={seededPalette}
                fontFamily={defaults.fontFamily}
                logoUrl={logo?.url}
                providers={[
                  ...(googleEnabled ? ["Continue with Google"] : []),
                  ...(appleEnabled ? ["Continue with Apple"] : []),
                ]}
              />
              <p className="caption" style={{ textAlign: "center" }}>
                Preview of the SDK login screen with the seeded theme.
              </p>
            </div>
          </div>
        )}

        {current === "review" && (
          <ReviewStep
            name={name.trim()}
            slug={effectiveSlug}
            platforms={platforms}
            publicSignup={publicSignup}
            requireVerification={requireVerification}
            minLen={minLenNum}
            googleEnabled={googleEnabled}
            googleDefer={googleDefer}
            appleEnabled={appleEnabled}
            appleDefer={appleDefer}
            sells={sells}
            entIdentifier={entIdentifier.trim()}
            tiers={tiers}
            sendsPushes={sendsPushes}
            vapidSet={vapidNow && vapidKey.trim() !== "" && vapidErr === ""}
            hasWeb={hasWeb}
            brandTouched={brandTouched}
            brandColor={brandColor}
            pref={pref}
            hasLogo={logo !== null}
            locales={locales}
          />
        )}

        {createError !== "" && <ErrorNote message={createError} />}

        <div className="row-12" style={{ maxWidth: 720 }}>
          {step > 0 ? (
            <button type="button" className="btn btn--secondary" onClick={back}>
              Back
            </button>
          ) : (
            <button
              type="button"
              className="btn btn--secondary"
              onClick={() => void navigate("/")}
            >
              Cancel
            </button>
          )}
          <span className="topbar__spacer" />
          {step > 0 && step < STEPS.length - 1 && (
            <button
              type="button"
              className="btn btn--ghost"
              disabled={!stepOk}
              onClick={toReview}
            >
              Skip to review
            </button>
          )}
          {current === "review" ? (
            <button
              type="button"
              className="btn btn--primary"
              disabled={creating}
              onClick={() => void createAll()}
            >
              {creating ? "Creating…" : "Create project"}
            </button>
          ) : (
            <button
              type="button"
              className="btn btn--primary"
              disabled={!stepOk}
              onClick={next}
            >
              Continue
            </button>
          )}
        </div>
        <p className="caption">
          Nothing is created until the review step — leaving the wizard now
          leaves no project behind.
        </p>
      </div>
    </main>
  );
}

// ---------- Step pieces ----------

function BasicsStep({
  name,
  onName,
  slug,
  slugOk,
  onSlug,
  platforms,
  onTogglePlatform,
}: {
  name: string;
  onName: (v: string) => void;
  slug: string;
  slugOk: boolean;
  onSlug: (v: string) => void;
  platforms: ProfilePlatform[];
  onTogglePlatform: (p: ProfilePlatform) => void;
}) {
  return (
    <div className="stack-24" style={{ maxWidth: 720 }}>
      <section className="card card--pad stack-16">
        <h3 className="card__title">Project</h3>
        <Field label="Name" help='One project per app, e.g. "Birdwatch".'>
          <input
            className="input"
            value={name}
            onChange={(e) => onName(e.target.value)}
            autoFocus
            maxLength={100}
          />
        </Field>
        <Field
          label="Slug"
          error={slug !== "" && !slugOk ? "Lowercase letters, digits and single dashes." : undefined}
          help="Derived from the name; used in URLs (JWKS, hosted pages). Editable."
        >
          <input
            className="input input--mono"
            value={slug}
            onChange={(e) => onSlug(e.target.value)}
            spellCheck={false}
          />
        </Field>
      </section>

      <section className="card card--pad stack-16">
        <h3 className="card__title">Platforms</h3>
        <p className="caption">
          Pick every platform the app ships on — at least one. This drives
          which questions the wizard asks and what the setup tab shows,
          forever after (editable under Settings).
        </p>
        {PLATFORMS.map((p) => (
          <label className="check" key={p.value}>
            <input
              type="checkbox"
              checked={platforms.includes(p.value)}
              onChange={() => onTogglePlatform(p.value)}
            />
            <span>{p.label}</span>
          </label>
        ))}
        {platforms.length === 0 && (
          <p className="caption">Select at least one platform to continue.</p>
        )}
      </section>
    </div>
  );
}

// ProviderStepCard wraps one social provider: the enable toggle, then the
// explicit configure-now / configure-later choice. "Later" is first-class —
// it shows the CLI hint and the gap lands on the overview checklist.
function ProviderStepCard({
  title,
  enabled,
  onEnabled,
  defer,
  onDefer,
  deferHint,
  children,
}: {
  title: string;
  enabled: boolean;
  onEnabled: (v: boolean) => void;
  defer: boolean;
  onDefer: (v: boolean) => void;
  deferHint: string;
  children: ReactNode;
}) {
  return (
    <section className="card card--pad stack-16">
      <h3 className="card__title">{title}</h3>
      <label className="check">
        <input type="checkbox" checked={enabled} onChange={(e) => onEnabled(e.target.checked)} />
        <span>Enable {title}</span>
      </label>
      {enabled && (
        <>
          <div className="seg" role="group" aria-label={`${title} credentials`}>
            <button
              type="button"
              className="seg__btn"
              aria-pressed={!defer}
              onClick={() => onDefer(false)}
            >
              Configure now
            </button>
            <button
              type="button"
              className="seg__btn"
              aria-pressed={defer}
              onClick={() => onDefer(true)}
            >
              Configure later
            </button>
          </div>
          {defer ? (
            <>
              <p className="caption">
                Deferred — it lands on the overview checklist. The Providers tab
                has the full walkthrough, or let the CLI drive the console:
              </p>
              <CodeBlock code={deferHint} />
            </>
          ) : (
            children
          )}
        </>
      )}
    </section>
  );
}

// YesNo is the wizard's explicit two-way ask; the answer is stored in the
// profile, so downstream surfaces branch on intent, never on inference.
function YesNo({
  value,
  onChange,
  labelledBy,
}: {
  value: boolean;
  onChange: (v: boolean) => void;
  labelledBy: string;
}) {
  return (
    <div className="seg" role="group" aria-label={labelledBy}>
      <button
        type="button"
        className="seg__btn"
        aria-pressed={!value}
        onClick={() => onChange(false)}
      >
        No
      </button>
      <button
        type="button"
        className="seg__btn"
        aria-pressed={value}
        onClick={() => onChange(true)}
      >
        Yes
      </button>
    </div>
  );
}

function TierRow({
  tier,
  index,
  showApple,
  showGoogle,
  onChange,
  onRemove,
}: {
  tier: TierDraft;
  index: number;
  showApple: boolean;
  showGoogle: boolean;
  onChange: (t: TierDraft) => void;
  onRemove: () => void;
}) {
  const set = (patch: Partial<TierDraft>) => onChange({ ...tier, ...patch });
  return (
    <div className="stack-12" style={{ borderTop: "1px solid var(--border)", paddingTop: 12 }}>
      <div className="row-8">
        <span className="body-strong" style={{ flex: 1 }}>
          Tier {index + 1}
        </span>
        <button type="button" className="btn btn--ghost btn--compact" onClick={onRemove}>
          Remove
        </button>
      </div>
      <div className="row-16" style={{ alignItems: "flex-start" }}>
        <div style={{ flex: 1 }}>
          <Field label="Identifier" help='e.g. "monthly"'>
            <input
              className="input input--mono"
              value={tier.identifier}
              onChange={(e) => set({ identifier: e.target.value })}
              placeholder="monthly"
              spellCheck={false}
            />
          </Field>
        </div>
        <div style={{ flex: 1 }}>
          <Field label="Display name">
            <input
              className="input"
              value={tier.displayName}
              onChange={(e) => set({ displayName: e.target.value })}
              placeholder="Monthly Pro"
            />
          </Field>
        </div>
      </div>
      <div className="row-16" style={{ alignItems: "flex-start" }}>
        <div style={{ flex: 1 }}>
          <Field label="Price">
            <input
              className="input input--mono"
              value={tier.price}
              onChange={(e) => set({ price: e.target.value })}
              placeholder="9.99"
              inputMode="decimal"
              spellCheck={false}
            />
          </Field>
        </div>
        <div style={{ flex: 1 }}>
          <Field label="Currency">
            <input
              className="input input--mono"
              value={tier.currency}
              onChange={(e) => set({ currency: e.target.value })}
              placeholder="USD"
              spellCheck={false}
              maxLength={3}
            />
          </Field>
        </div>
        <div style={{ flex: 1 }}>
          <Field label="Billing period">
            <input
              className="input"
              value={tier.billingPeriod}
              onChange={(e) => set({ billingPeriod: e.target.value })}
              placeholder="monthly"
            />
          </Field>
        </div>
      </div>
      {(showApple || showGoogle) && (
        <div className="row-16" style={{ alignItems: "flex-start" }}>
          {showApple && (
            <div style={{ flex: 1 }}>
              <Field label="Apple product id" help="App Store SKU; blank to map later.">
                <input
                  className="input input--mono"
                  value={tier.appleId}
                  onChange={(e) => set({ appleId: e.target.value })}
                  placeholder="com.example.pro.monthly"
                  spellCheck={false}
                />
              </Field>
            </div>
          )}
          {showGoogle && (
            <div style={{ flex: 1 }}>
              <Field label="Google product id" help="Play SKU; blank to map later.">
                <input
                  className="input input--mono"
                  value={tier.googleId}
                  onChange={(e) => set({ googleId: e.target.value })}
                  placeholder="pro_monthly"
                  spellCheck={false}
                />
              </Field>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function LogoPicker({
  logo,
  error,
  onFile,
}: {
  logo: { file: File; url: string } | null;
  error: string;
  onFile: (f: File) => void;
}) {
  return (
    <div className="field">
      <span className="field__label">Logo</span>
      <div className="design__logo">
        <span className="design__logo-thumb">
          {logo ? (
            <img src={logo.url} alt="Logo preview" />
          ) : (
            <span className="caption text-tertiary">None</span>
          )}
        </span>
        <label className="btn btn--secondary btn--compact" style={{ cursor: "pointer" }}>
          {logo ? "Replace" : "Upload"}
          <input
            type="file"
            accept="image/png,image/svg+xml"
            hidden
            onChange={(e) => {
              const f = e.target.files?.[0];
              if (f) onFile(f);
              e.target.value = "";
            }}
          />
        </label>
      </div>
      {error !== "" ? (
        <span className="field__error">{error}</span>
      ) : (
        <span className="field__help">PNG or SVG, at most 512 KiB. Uploaded on create.</span>
      )}
    </div>
  );
}

// WizardPreview is the Design tab's phone-frame login preview, self-contained
// for the wizard: same .mothpv token contract, driven by the seeded palette
// and the not-yet-uploaded logo file.
function WizardPreview({
  appName,
  palette,
  fontFamily,
  logoUrl,
  providers,
}: {
  appName: string;
  palette: Palette;
  fontFamily: string;
  logoUrl?: string;
  providers: string[];
}) {
  const vars = {
    "--p-primary": palette.primary,
    "--p-on-primary": palette.onPrimary,
    "--p-background": palette.background,
    "--p-on-background": palette.onBackground,
    "--p-surface": palette.surface,
    "--p-on-surface": palette.onSurface,
    "--p-font": fontStack(fontFamily),
    "--p-body": "15px",
    "--p-unit": "8px",
    "--p-radius": "12px",
  } as CSSProperties;

  return (
    <div className="phone">
      <div className="phone__screen">
        <div className="mothpv" style={vars}>
          <div className="mothpv__card">
            {logoUrl ? (
              <img className="mothpv__logo" src={logoUrl} alt="" />
            ) : (
              <span className="mothpv__logo-fallback">
                {(appName[0] ?? "A").toUpperCase()}
              </span>
            )}
            <div className="mothpv__title">Sign in to {appName}</div>
            <div className="mothpv__field">
              <span className="mothpv__label">Email</span>
              <span className="mothpv__input">you@example.com</span>
            </div>
            <div className="mothpv__field">
              <span className="mothpv__label">Password</span>
              <span className="mothpv__input">••••••••</span>
            </div>
            <div className="mothpv__btn">Sign in</div>
            <div className="mothpv__link">Forgot password?</div>
            {providers.length > 0 && (
              <>
                <div className="mothpv__divider">or</div>
                {providers.map((p) => (
                  <div key={p} className="mothpv__provider">
                    {p}
                  </div>
                ))}
              </>
            )}
          </div>
          <div className="mothpv__footer">
            <span>{appName}</span>
          </div>
        </div>
      </div>
    </div>
  );
}

// ---------- Review ----------

function ReviewStep(props: {
  name: string;
  slug: string;
  platforms: ProfilePlatform[];
  publicSignup: boolean;
  requireVerification: boolean;
  minLen: number;
  googleEnabled: boolean;
  googleDefer: boolean;
  appleEnabled: boolean;
  appleDefer: boolean;
  sells: boolean;
  entIdentifier: string;
  tiers: TierDraft[];
  sendsPushes: boolean;
  vapidSet: boolean;
  hasWeb: boolean;
  brandTouched: boolean;
  brandColor: string;
  pref: "light" | "dark";
  hasLogo: boolean;
  locales: string[];
}) {
  const platformNames = PLATFORMS.filter((p) => props.platforms.includes(p.value)).map(
    (p) => p.label,
  );
  const provider = (enabled: boolean, defer: boolean) =>
    !enabled ? "off" : defer ? "on — credentials deferred to the checklist" : "on — configured";

  return (
    <div className="stack-24" style={{ maxWidth: 720 }}>
      <p className="caption">
        Nothing has been created yet. Confirm below and moth creates the
        project, then applies everything in one pass — anything deferred shows
        up on the overview checklist.
      </p>
      <section className="card card--pad stack-12">
        <h3 className="card__title">Basics</h3>
        <ReviewRow label="Name" value={props.name} />
        <ReviewRow label="Slug" value={props.slug} mono />
        <ReviewRow label="Platforms" value={platformNames.join(", ")} />
      </section>
      <section className="card card--pad stack-12">
        <h3 className="card__title">Sign-in</h3>
        <ReviewRow
          label="Email & password"
          value={`sign-up ${props.publicSignup ? "open" : "invite-only"} · verification ${
            props.requireVerification ? "required" : "optional"
          } · min length ${props.minLen}`}
        />
        <ReviewRow label="Google" value={provider(props.googleEnabled, props.googleDefer)} />
        <ReviewRow label="Apple" value={provider(props.appleEnabled, props.appleDefer)} />
      </section>
      <section className="card card--pad stack-12">
        <h3 className="card__title">Monetization</h3>
        {props.sells ? (
          <>
            <ReviewRow
              label="Entitlement"
              value={props.entIdentifier || "none yet — define it under Monetization"}
              mono={props.entIdentifier !== ""}
            />
            <ReviewRow
              label="Tiers"
              value={
                props.tiers.length === 0
                  ? "none yet"
                  : props.tiers.map((t) => t.identifier.trim() || "?").join(", ")
              }
            />
            <ReviewRow label="Store credentials & sync" value="deferred to the checklist" />
          </>
        ) : (
          <ReviewRow label="Subscriptions" value="no — free app" />
        )}
      </section>
      <section className="card card--pad stack-12">
        <h3 className="card__title">Push notifications</h3>
        {props.sendsPushes ? (
          <>
            <ReviewRow label="Push registration" value="enabled" />
            {props.hasWeb && (
              <ReviewRow
                label="VAPID public key"
                value={props.vapidSet ? "set" : "deferred to the checklist"}
              />
            )}
          </>
        ) : (
          <ReviewRow label="Pushes" value="no" />
        )}
      </section>
      <section className="card card--pad stack-12">
        <h3 className="card__title">Branding</h3>
        {props.brandTouched || props.hasLogo || props.locales.length > 0 ? (
          <>
            {props.brandTouched && (
              <ReviewRow
                label="Theme"
                value={`${props.brandColor} · ${props.pref} appearance`}
                mono
              />
            )}
            {props.hasLogo && <ReviewRow label="Logo" value="uploaded on create" />}
            {props.locales.length > 0 && (
              <ReviewRow label="Languages" value={`en, ${props.locales.join(", ")}`} mono />
            )}
          </>
        ) : (
          <ReviewRow label="Theme" value="defaults — brand it later under Design" />
        )}
      </section>
    </div>
  );
}

function ReviewRow({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="row-12" style={{ alignItems: "baseline" }}>
      <span className="caption" style={{ width: 180, flexShrink: 0 }}>
        {label}
      </span>
      <span className={mono ? "mono" : undefined}>{value}</span>
    </div>
  );
}
