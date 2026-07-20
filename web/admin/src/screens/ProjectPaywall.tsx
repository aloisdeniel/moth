import { useMutation, useQuery } from "@connectrpc/connect-query";
import { useState, type CSSProperties } from "react";

import { errorMessage, invalidate } from "../api";
import { Badge, ConfirmDialog, ErrorNote, Field, Loading, StringListField } from "../components/ui";
import type { ListLocalesResponse } from "../gen/moth/admin/v1/copy_pb";
import { CopyScreen, CopyService } from "../gen/moth/admin/v1/copy_pb";
import { MonetizationService } from "../gen/moth/admin/v1/monetization_pb";
import type { GetPaywallConfigResponse, PaywallRevision } from "../gen/moth/admin/v1/paywall_pb";
import { PaywallLayout, PaywallService } from "../gen/moth/admin/v1/paywall_pb";
import type { Product } from "../gen/moth/admin/v1/product_pb";
import { ProductService } from "../gen/moth/admin/v1/product_pb";
import type { Project } from "../gen/moth/admin/v1/project_pb";
import type { GetThemeResponse } from "../gen/moth/admin/v1/theme_pb";
import { formatPrice } from "../lib/billing";
import { formatDateTime } from "../lib/format";
import {
  LAYOUT_OPTIONS,
  MAX_BENEFITS,
  MAX_HEADLINE,
  MAX_SUBTITLE,
  paywallFromProto,
  paywallToProto,
  type EditorPaywall,
} from "../lib/paywall";
import { copyResolver, draftFromKeys, draftValues, type CopyDraft } from "../lib/copy";
import { editorFromProto, effectiveDark, fontStack } from "../lib/theme";
import { CopyFields, LocaleSelector, mergeLocaleOptions } from "./ProjectCopy";

// PaywallEditor is the paywall half of the Design tab: the paywall copy and
// layout on the left, a live phone-framed replica of MothPaywallScreen on
// the right, rendered from the current (unsaved) editor state and the
// project's saved theme tokens (colors/typography inherit from the theme —
// this editor never introduces its own token space).
export function PaywallEditor({ project, theme }: { project: Project; theme: GetThemeResponse }) {
  const config = useQuery(PaywallService.method.getPaywallConfig, { projectId: project.id });
  const locales = useQuery(CopyService.method.listLocales, { projectId: project.id });
  if (config.isPending || locales.isPending) return <Loading />;
  if (config.isError) return <ErrorNote message={errorMessage(config.error)} />;
  if (locales.isError) return <ErrorNote message={errorMessage(locales.error)} />;
  return <PaywallForm project={project} theme={theme} current={config.data} locales={locales.data} />;
}

function PaywallForm({
  project,
  theme,
  current,
  locales,
}: {
  project: Project;
  theme: GetThemeResponse;
  current: GetPaywallConfigResponse;
  locales: ListLocalesResponse;
}) {
  const [p, setP] = useState<EditorPaywall>(() => paywallFromProto(current.config));
  const [scheme, setScheme] = useState<"light" | "dark">("light");
  const [saved, setSaved] = useState(false);
  const [resetOpen, setResetOpen] = useState(false);
  const [restoreTarget, setRestoreTarget] = useState<PaywallRevision | null>(null);

  // The offering the paywall presents drives both the tier previews and the
  // "highlight" dropdown; empty selects the project's default offering.
  const offering = useQuery(MonetizationService.method.getOffering, {
    projectId: project.id,
    offering: p.offering,
  });
  const products = useQuery(ProductService.method.listProducts, { projectId: project.id });
  const revisions = useQuery(PaywallService.method.listPaywallRevisions, { projectId: project.id });

  const tiers = offering.data?.offering?.products ?? [];
  // Distinct non-empty offering tags across the catalog, for the selector.
  const offeringTags = [
    ...new Set((products.data?.products ?? []).map((x) => x.offering).filter((x) => x !== "")),
  ].sort();

  const refresh = () =>
    invalidate(
      PaywallService.method.getPaywallConfig,
      PaywallService.method.listPaywallRevisions,
    );

  const update = useMutation(PaywallService.method.updatePaywallConfig, {
    onSuccess: () => {
      refresh();
      setSaved(true);
      setTimeout(() => setSaved(false), 2000);
    },
  });
  const restore = useMutation(PaywallService.method.restorePaywallRevision, {
    onSuccess: (res) => {
      setP(paywallFromProto(res.config));
      setRestoreTarget(null);
      refresh();
    },
  });
  const reset = useMutation(PaywallService.method.resetPaywall, {
    onSuccess: (res) => {
      setP(paywallFromProto(res.config));
      setResetOpen(false);
      refresh();
    },
  });

  // ---- Localized copy (milestone 15): the paywall.* catalog keys for the
  // selected language, previewed alongside the layout/offering config above.
  const [copyLocale, setCopyLocale] = useState(locales.defaultLocale || "en");
  const [added, setAdded] = useState<{ tag: string; displayName: string; isDefault: boolean }[]>(
    [],
  );
  const [copySaved, setCopySaved] = useState(false);
  const copy = useQuery(CopyService.method.getProjectCopy, {
    projectId: project.id,
    locale: copyLocale,
    screen: CopyScreen.PAYWALL,
  });
  const [copyDraft, setCopyDraft] = useState<CopyDraft>({});
  const [copySyncKey, setCopySyncKey] = useState<string | null>(null);
  const copyDataKey = copy.data ? `${copy.data.locale}:${copy.data.revisionId}` : null;
  if (copy.data && copyDataKey !== copySyncKey) {
    setCopySyncKey(copyDataKey);
    setCopyDraft(draftFromKeys(copy.data.keys));
  }
  const updateCopy = useMutation(CopyService.method.updateProjectCopy, {
    onSuccess: () => {
      invalidate(CopyService.method.getProjectCopy, CopyService.method.listLocales);
      setCopySaved(true);
      setTimeout(() => setCopySaved(false), 2000);
    },
  });
  const copyKeys = copy.data?.keys ?? [];
  const copyGet = copyResolver(copyKeys, copyDraft, { app: project.name });
  const localeOptions = mergeLocaleOptions(locales.locales, added, locales.defaultLocale);

  // The paywall's title/subtitle come from the localized copy catalog
  // (paywall.title / paywall.subtitle) so editing them reflects live in the
  // preview (plan/15). The milestone-13 structural headline/subtitle remain the
  // non-localized fallback when this language has no override of that key.
  const titleOverridden = (copyDraft["paywall.title"] ?? "").trim() !== "";
  const previewHeadline = titleOverridden ? copyGet("paywall.title") : p.headline.trim() || copyGet("paywall.title");
  const subtitleOverridden = (copyDraft["paywall.subtitle"] ?? "").trim() !== "";
  const previewSubtitle = subtitleOverridden
    ? copyGet("paywall.subtitle")
    : p.subtitle.trim() || copyGet("paywall.subtitle");

  function saveCopy() {
    updateCopy.mutate({
      projectId: project.id,
      locale: copyLocale,
      values: draftValues(copyDraft),
    });
  }

  function set<K extends keyof EditorPaywall>(key: K, value: EditorPaywall[K]) {
    setP((prev) => ({ ...prev, [key]: value }));
  }

  function save() {
    update.mutate({ projectId: project.id, config: paywallToProto(p) });
  }

  const headlineInvalid = p.headline.trim() === "" || p.headline.length > MAX_HEADLINE;

  return (
    <div className="design">
      <form
        className="stack-24"
        onSubmit={(e) => {
          e.preventDefault();
          save();
        }}
      >
        <section className="card card--pad stack-16">
          <h3 className="card__title">Copy</h3>
          <p className="caption">
            The paywall's wording. Colors, typography, spacing and the logo all
            inherit from the project theme — edit those in the Login screen tab.
          </p>
          <Field label="Headline" help={`Required, at most ${MAX_HEADLINE} characters.`}>
            <input
              className={headlineInvalid ? "input input--error" : "input"}
              value={p.headline}
              maxLength={MAX_HEADLINE}
              onChange={(e) => set("headline", e.target.value)}
              placeholder="Unlock everything"
            />
          </Field>
          <Field label="Subtitle" help={`Optional, at most ${MAX_SUBTITLE} characters.`}>
            <input
              className="input"
              value={p.subtitle}
              maxLength={MAX_SUBTITLE}
              onChange={(e) => set("subtitle", e.target.value)}
              placeholder="Go further with a subscription."
            />
          </Field>
        </section>

        <section className="card card--pad stack-16">
          <h3 className="card__title">Benefits</h3>
          <p className="caption">
            The feature bullets listed above the tiers, in display order (at most{" "}
            {MAX_BENEFITS}).
          </p>
          <StringListField
            label="Benefit"
            values={p.benefits}
            onChange={(v) => set("benefits", v.slice(0, MAX_BENEFITS))}
            placeholder="Unlimited projects"
          />
        </section>

        <section className="card card--pad stack-16">
          <h3 className="card__title">Offering &amp; tiers</h3>
          <p className="caption">
            Which offering the paywall lists and which tier it highlights as “most
            popular”. Manage the tiers and their order under Monetization.
          </p>
          <Field label="Offering" help="The group of tiers to present; the default offering when blank.">
            <select
              className="select"
              value={p.offering}
              onChange={(e) => {
                set("offering", e.target.value);
                set("highlightedProductIdentifier", "");
              }}
            >
              <option value="">Default offering</option>
              {offeringTags.map((tag) => (
                <option key={tag} value={tag}>
                  {tag}
                </option>
              ))}
            </select>
          </Field>
          <Field label="Highlighted tier" help="Rendered as “most popular”; none when blank.">
            <select
              className="select"
              value={p.highlightedProductIdentifier}
              onChange={(e) => set("highlightedProductIdentifier", e.target.value)}
            >
              <option value="">No highlight</option>
              {tiers.map((tier) => (
                <option key={tier.id} value={tier.identifier}>
                  {tier.displayName || tier.identifier}
                </option>
              ))}
            </select>
          </Field>
          {offering.data && tiers.length === 0 && (
            <p className="caption">
              This offering has no tiers yet — add products under Monetization to
              populate the paywall.
            </p>
          )}
        </section>

        <section className="card card--pad stack-16">
          <h3 className="card__title">Layout</h3>
          <div className="seg" role="group" aria-label="Paywall layout">
            {LAYOUT_OPTIONS.map((opt) => (
              <button
                key={opt.value}
                type="button"
                className="seg__btn"
                aria-pressed={p.layout === opt.value}
                onClick={() => set("layout", opt.value)}
              >
                {opt.label}
              </button>
            ))}
          </div>
          <p className="caption">
            {LAYOUT_OPTIONS.find((o) => o.value === p.layout)?.help}
          </p>
        </section>

        <section className="card card--pad stack-16">
          <h3 className="card__title">Legal links</h3>
          <Field
            label="Terms of service URL"
            help="Optional; rendered in the paywall footer. Absolute http(s) URL."
          >
            <input
              className="input"
              type="url"
              value={p.termsUrl}
              onChange={(e) => set("termsUrl", e.target.value)}
              placeholder="https://example.com/terms"
              spellCheck={false}
            />
          </Field>
          <Field label="Privacy policy URL" help="Optional; rendered next to the terms link.">
            <input
              className="input"
              type="url"
              value={p.privacyUrl}
              onChange={(e) => set("privacyUrl", e.target.value)}
              placeholder="https://example.com/privacy"
              spellCheck={false}
            />
          </Field>
        </section>

        <section className="card card--pad stack-16">
          <h3 className="card__title">Localized copy</h3>
          <p className="caption">
            The paywall's catalog strings (title, subtitle, buttons, terms) per language, layered
            over moth's bundled defaults. The headline, subtitle and benefits above are the
            structural config; these localize the surrounding chrome the SDK renders.
          </p>
          <LocaleSelector
            options={localeOptions}
            value={copyLocale}
            onChange={setCopyLocale}
            onAdd={(o) => {
              setAdded((a) => (a.some((x) => x.tag === o.tag) ? a : [...a, o]));
              setCopyLocale(o.tag);
            }}
          />
          {copy.isPending && <Loading />}
          {copy.isError && <ErrorNote message={errorMessage(copy.error)} />}
          {copy.data && (
            <CopyFields
              keys={copyKeys}
              draft={copyDraft}
              onChange={(k, v) => setCopyDraft((d) => ({ ...d, [k]: v }))}
              onReset={(k) => setCopyDraft((d) => ({ ...d, [k]: "" }))}
            />
          )}
          <div className="row-12">
            <button
              type="button"
              className="btn btn--primary"
              disabled={updateCopy.isPending}
              onClick={saveCopy}
            >
              {updateCopy.isPending ? "Saving…" : "Save copy"}
            </button>
            {copySaved && <span className="caption text-success">Saved.</span>}
            {updateCopy.isError && (
              <span className="field__error">{errorMessage(updateCopy.error)}</span>
            )}
          </div>
        </section>

        <div className="stack-8">
          {headlineInvalid && (
            <p className="field__error">A headline is required before saving.</p>
          )}
          <div className="row-12">
            <button
              type="submit"
              className="btn btn--primary"
              disabled={update.isPending || headlineInvalid}
            >
              {update.isPending ? "Saving…" : "Save paywall"}
            </button>
            {saved && <span className="caption text-success">Saved.</span>}
            {update.isError && <span className="field__error">{errorMessage(update.error)}</span>}
          </div>
        </div>

        <section className="card card--pad stack-16">
          <h3 className="card__title">Revisions</h3>
          <p className="caption">
            Every save keeps a revision (the last 10). Restoring re-installs an old
            paywall config as a new revision.
          </p>
          {revisions.isPending && <Loading />}
          {revisions.isError && <ErrorNote message={errorMessage(revisions.error)} />}
          {revisions.data &&
            (revisions.data.revisions.length === 0 ? (
              <p className="caption">No revisions yet — the project renders the defaults.</p>
            ) : (
              <table className="table">
                <thead>
                  <tr>
                    <th>Saved</th>
                    <th>Headline</th>
                    <th>Revision</th>
                    <th />
                  </tr>
                </thead>
                <tbody>
                  {revisions.data.revisions.map((rev) => (
                    <tr key={rev.revisionId}>
                      <td className="mono nowrap">{formatDateTime(rev.createTime)}</td>
                      <td>{rev.config?.headline || <span className="text-tertiary">—</span>}</td>
                      <td className="mono">{rev.revisionId.slice(0, 8)}</td>
                      <td style={{ textAlign: "right" }}>
                        {rev.revisionId === current.revisionId ? (
                          <Badge tone="accent">Current</Badge>
                        ) : (
                          <button
                            type="button"
                            className="btn btn--secondary btn--compact"
                            onClick={() => setRestoreTarget(rev)}
                          >
                            Restore
                          </button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            ))}
        </section>

        <section className="card card--pad danger-zone">
          <div className="danger-zone__row">
            <div className="stack-8">
              <span className="body-strong">Reset to defaults</span>
              <span className="caption">
                Revert to the built-in paywall config. Saved revisions are kept, so
                the current config stays restorable.
              </span>
            </div>
            <button type="button" className="btn btn--danger" onClick={() => setResetOpen(true)}>
              Reset paywall
            </button>
          </div>
        </section>
      </form>

      <div className="design__preview">
        <div className="seg" role="group" aria-label="Preview color scheme">
          <button
            type="button"
            className="seg__btn"
            aria-pressed={scheme === "light"}
            onClick={() => setScheme("light")}
          >
            Light
          </button>
          <button
            type="button"
            className="seg__btn"
            aria-pressed={scheme === "dark"}
            onClick={() => setScheme("dark")}
          >
            Dark
          </button>
        </div>
        <PaywallPreview
          project={project}
          p={p}
          tiers={tiers}
          scheme={scheme}
          theme={theme}
          copyGet={copyGet}
          headline={previewHeadline}
          subtitle={previewSubtitle}
        />
        <p className="caption" style={{ textAlign: "center" }}>
          Live preview of the SDK paywall screen, rendered from the unsaved editor
          state, the selected language's copy and the project theme.
        </p>
      </div>

      <ConfirmDialog
        title="Restore revision"
        open={restoreTarget !== null}
        onClose={() => setRestoreTarget(null)}
        onConfirm={() =>
          restoreTarget &&
          restore.mutate({ projectId: project.id, revisionId: restoreTarget.revisionId })
        }
        confirmLabel="Restore"
        busy={restore.isPending}
        error={restore.isError ? errorMessage(restore.error) : undefined}
      >
        <p>
          Restore the paywall saved {formatDateTime(restoreTarget?.createTime)} (revision{" "}
          <span className="mono">{restoreTarget?.revisionId.slice(0, 8)}</span>)? Unsaved edits in
          the editor are discarded.
        </p>
      </ConfirmDialog>

      <ConfirmDialog
        title="Reset paywall"
        open={resetOpen}
        onClose={() => setResetOpen(false)}
        onConfirm={() => reset.mutate({ projectId: project.id })}
        confirmLabel="Reset paywall"
        busy={reset.isPending}
        error={reset.isError ? errorMessage(reset.error) : undefined}
      >
        <p>
          Reset <strong>{project.name}</strong> to the built-in default paywall config? The
          revision history is kept, so the current config can be restored.
        </p>
      </ConfirmDialog>
    </div>
  );
}

// ---------- Live preview ----------

// PaywallPreview is a phone-framed HTML/CSS replica of MothPaywallScreen. Like
// the login preview, the inner screen is styled exclusively through --p-*
// custom properties set from the project's saved theme tokens, so it shares
// the theme's token semantics rather than the admin's own tokens.
function PaywallPreview({
  project,
  p,
  tiers,
  scheme,
  theme,
  copyGet,
  headline,
  subtitle,
}: {
  project: Project;
  p: EditorPaywall;
  tiers: Product[];
  scheme: "light" | "dark";
  theme: GetThemeResponse;
  copyGet: (key: string, fallback?: string) => string;
  headline: string;
  subtitle: string;
}) {
  const t = editorFromProto(theme.theme);
  const palette = scheme === "light" ? t.colors : effectiveDark(t);
  const logo =
    scheme === "light"
      ? theme.theme?.logo?.lightPath
      : theme.theme?.logo?.darkPath || theme.theme?.logo?.lightPath;

  const vars = {
    "--p-primary": palette.primary,
    "--p-on-primary": palette.onPrimary,
    "--p-background": palette.background,
    "--p-on-background": palette.onBackground,
    "--p-surface": palette.surface,
    "--p-on-surface": palette.onSurface,
    "--p-font": fontStack(t.fontFamily),
    "--p-body": `${(15 * t.scale).toFixed(2)}px`,
    "--p-unit": `${t.spacingUnit}px`,
    "--p-radius": `${t.cornerRadius}px`,
  } as CSSProperties;

  const layoutName =
    p.layout === PaywallLayout.LIST ? "list" : p.layout === PaywallLayout.COMPACT ? "compact" : "tiles";
  // Compact renders only the highlighted tier (or the first when none is set).
  const shown =
    p.layout === PaywallLayout.COMPACT
      ? tiers.filter((x) => x.identifier === p.highlightedProductIdentifier).concat(tiers).slice(0, 1)
      : tiers;

  const legal = [
    ...(p.termsUrl.trim() ? ["Terms"] : []),
    ...(p.privacyUrl.trim() ? ["Privacy"] : []),
  ];

  return (
    <div className="phone">
      <div className="phone__screen">
        <div className="mothpw" style={vars} data-scheme={scheme}>
          <div className="mothpw__scroll">
            {logo ? (
              <img className="mothpw__logo" src={logo} alt="" />
            ) : (
              <span className="mothpw__logo-fallback">
                {(project.name[0] ?? "A").toUpperCase()}
              </span>
            )}
            <div className="mothpw__headline">{headline || "Unlock everything"}</div>
            {subtitle.trim() && <div className="mothpw__subtitle">{subtitle}</div>}

            {p.benefits.length > 0 && (
              <ul className="mothpw__benefits">
                {p.benefits.map((b, i) => (
                  <li key={i} className="mothpw__benefit">
                    <span className="mothpw__check" aria-hidden>
                      ✓
                    </span>
                    <span>{b}</span>
                  </li>
                ))}
              </ul>
            )}

            {shown.length > 0 ? (
              <div className="mothpw__tiers" data-layout={layoutName}>
                {shown.map((tier) => {
                  const highlighted = tier.identifier === p.highlightedProductIdentifier;
                  return (
                    <div
                      key={tier.id}
                      className="mothpw__tier"
                      data-highlighted={highlighted || undefined}
                    >
                      {highlighted && <span className="mothpw__ribbon">Most popular</span>}
                      <span className="mothpw__tier-name">{tier.displayName || tier.identifier}</span>
                      <span className="mothpw__tier-price">
                        {formatPrice(tier.priceAmountMicros, tier.currency)}
                        {tier.billingPeriod && (
                          <span className="mothpw__tier-period"> / {tier.billingPeriod}</span>
                        )}
                      </span>
                      {tier.trialPeriod && (
                        <span className="mothpw__tier-trial">{tier.trialPeriod}</span>
                      )}
                    </div>
                  );
                })}
              </div>
            ) : (
              <div className="mothpw__empty">Nothing to purchase yet</div>
            )}

            <div className="mothpw__btn">{copyGet("paywall.cta", "Continue")}</div>
            <div className="mothpw__link">{copyGet("paywall.restore", "Restore purchases")}</div>
            {copyGet("paywall.terms").trim() !== "" && (
              <div className="mothpw__terms">{copyGet("paywall.terms")}</div>
            )}
          </div>
          <div className="mothpw__footer">
            {legal.length > 0 && (
              <span>
                {legal.map((l, i) => (
                  <span key={l}>
                    {i > 0 && " · "}
                    <span className="mothpw__footer-link">{l}</span>
                  </span>
                ))}
              </span>
            )}
            <span>{project.name}</span>
          </div>
        </div>
      </div>
    </div>
  );
}
