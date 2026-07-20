import { useMutation, useQuery } from "@connectrpc/connect-query";
import { useState, type CSSProperties, type ReactNode } from "react";

import { errorMessage, invalidate } from "../api";
import { Badge, ConfirmDialog, ErrorNote, Field, Loading } from "../components/ui";
import type { CopyKey, CopyRevision, Locale } from "../gen/moth/admin/v1/copy_pb";
import { CopyScreen, CopyService } from "../gen/moth/admin/v1/copy_pb";
import type { Project } from "../gen/moth/admin/v1/project_pb";
import type { GetThemeResponse } from "../gen/moth/admin/v1/theme_pb";
import {
  copyResolver,
  draftFromKeys,
  draftValues,
  keyLabel,
  type CopyDraft,
} from "../lib/copy";
import { formatDateTime } from "../lib/format";
import { editorFromProto, effectiveDark, fontStack } from "../lib/theme";

// ProjectCopy hosts the per-screen localization editors of the Design tab
// (milestone 15): a language selector over the project's available + bundled
// locales, a per-catalog-key copy editor for the selected screen and locale,
// and a live phone-frame preview of that screen rendered from the (unsaved)
// copy merged over the bundled defaults. Mirrors the theme editor's shape.

// A locale option in the selector: either a server-listed locale or one the
// operator added client-side for a tag moth does not bundle.
export type LocaleOption = { tag: string; displayName: string; isDefault: boolean };

// ---------- Sign in / Sign up sub-tab ----------

export function SignInUpTab({
  project,
  theme,
  screen,
}: {
  project: Project;
  theme: GetThemeResponse;
  screen: CopyScreen.SIGN_IN | CopyScreen.SIGN_UP;
}) {
  const locales = useQuery(CopyService.method.listLocales, { projectId: project.id });
  if (locales.isPending) return <Loading />;
  if (locales.isError) return <ErrorNote message={errorMessage(locales.error)} />;
  return (
    <CopyScreenBody
      project={project}
      screen={screen}
      defaultLocale={locales.data.defaultLocale || "en"}
      serverLocales={locales.data.locales}
      preview={(p) => (
        <SignPreview
          project={project}
          theme={theme}
          scheme={p.scheme}
          mode={screen === CopyScreen.SIGN_UP ? "sign_up" : "sign_in"}
          get={p.get}
        />
      )}
      caption={
        screen === CopyScreen.SIGN_UP
          ? "Live preview of the SDK sign-up screen for the selected language, from the unsaved editor state."
          : "Live preview of the SDK sign-in screen for the selected language, from the unsaved editor state."
      }
    />
  );
}

// ---------- Shared editor body (selector + fields + preview) ----------

// CopyScreenBody is the reusable left-editor / right-preview layout the copy
// sub-tabs share. The paywall sub-tab embeds the language selector and copy
// fields directly (it also owns the milestone-13 config), so this component
// serves the sign-in/sign-up screens; `preview` renders the phone frame.
export function CopyScreenBody({
  project,
  screen,
  defaultLocale,
  serverLocales,
  preview,
  caption,
  extraCards,
}: {
  project: Project;
  screen: CopyScreen;
  defaultLocale: string;
  serverLocales: Locale[];
  preview: (p: {
    scheme: "light" | "dark";
    get: (key: string, fallback?: string) => string;
  }) => ReactNode;
  caption: string;
  extraCards?: ReactNode;
}) {
  const [added, setAdded] = useState<LocaleOption[]>([]);
  const [locale, setLocale] = useState(defaultLocale);
  const [scheme, setScheme] = useState<"light" | "dark">("light");
  const [saved, setSaved] = useState(false);

  const copy = useQuery(CopyService.method.getProjectCopy, {
    projectId: project.id,
    locale,
    screen,
  });

  // Derived-state resync (same pattern as the theme ColorInput): when the
  // fetched document changes — locale switch, save, restore — reseed the draft.
  const [draft, setDraft] = useState<CopyDraft>({});
  const [syncKey, setSyncKey] = useState<string | null>(null);
  const dataKey = copy.data ? `${copy.data.locale}:${copy.data.revisionId}` : null;
  if (copy.data && dataKey !== syncKey) {
    setSyncKey(dataKey);
    setDraft(draftFromKeys(copy.data.keys));
  }

  const update = useMutation(CopyService.method.updateProjectCopy, {
    onSuccess: () => {
      invalidate(
        CopyService.method.getProjectCopy,
        CopyService.method.listCopyRevisions,
        CopyService.method.listLocales,
      );
      setSaved(true);
      setTimeout(() => setSaved(false), 2000);
    },
  });

  const keys = copy.data?.keys ?? [];
  const get = copyResolver(keys, draft, { app: project.name, email: "you@example.com" });

  const options = mergeLocaleOptions(serverLocales, added, defaultLocale);
  const currentName = options.find((o) => o.tag === locale)?.displayName ?? locale;

  function save() {
    update.mutate({ projectId: project.id, locale, values: draftValues(draft) });
  }

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
          <h3 className="card__title">Language</h3>
          <LocaleSelector
            options={options}
            value={locale}
            onChange={setLocale}
            onAdd={(o) => {
              setAdded((a) => (a.some((x) => x.tag === o.tag) ? a : [...a, o]));
              setLocale(o.tag);
            }}
          />
        </section>

        <section className="card card--pad stack-16">
          <h3 className="card__title">Copy</h3>
          <p className="caption">
            Every field's placeholder is the bundled default for {currentName}. Leave a field
            empty to keep that default; a filled field overrides it for this language.
          </p>
          {copy.isPending && <Loading />}
          {copy.isError && <ErrorNote message={errorMessage(copy.error)} />}
          {copy.data && (
            <CopyFields
              keys={keys}
              draft={draft}
              onChange={(k, v) => setDraft((d) => ({ ...d, [k]: v }))}
              onReset={(k) => setDraft((d) => ({ ...d, [k]: "" }))}
            />
          )}
        </section>

        {extraCards}

        <div className="row-12">
          <button type="submit" className="btn btn--primary" disabled={update.isPending}>
            {update.isPending ? "Saving…" : "Save copy"}
          </button>
          {saved && <span className="caption text-success">Saved.</span>}
          {update.isError && <span className="field__error">{errorMessage(update.error)}</span>}
        </div>

        <CopyRevisions project={project} currentRevisionId={copy.data?.revisionId ?? ""} />
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
        {preview({ scheme, get })}
        <p className="caption" style={{ textAlign: "center" }}>
          {caption}
        </p>
      </div>
    </div>
  );
}

// mergeLocaleOptions unions the server's locales with any client-added tags,
// deduped, with the project default flagged.
export function mergeLocaleOptions(
  serverLocales: Locale[],
  added: LocaleOption[],
  defaultLocale: string,
): LocaleOption[] {
  const seen = new Set<string>();
  const out: LocaleOption[] = [];
  for (const l of serverLocales) {
    seen.add(l.tag);
    out.push({ tag: l.tag, displayName: l.displayName || l.tag, isDefault: l.isDefault });
  }
  for (const a of added) {
    if (seen.has(a.tag)) continue;
    seen.add(a.tag);
    out.push({ tag: a.tag, displayName: a.displayName, isDefault: a.tag === defaultLocale });
  }
  return out;
}

// ---------- Language selector ----------

export function LocaleSelector({
  options,
  value,
  onChange,
  onAdd,
}: {
  options: LocaleOption[];
  value: string;
  onChange: (tag: string) => void;
  onAdd: (o: LocaleOption) => void;
}) {
  const [adding, setAdding] = useState(false);
  const [tag, setTag] = useState("");

  function commit() {
    const t = tag.trim();
    if (t === "") return;
    onAdd({ tag: t, displayName: t, isDefault: false });
    setTag("");
    setAdding(false);
  }

  return (
    <div className="stack-8">
      <Field label="Language" help="The project's available and bundled languages.">
        <select className="select" value={value} onChange={(e) => onChange(e.target.value)}>
          {options.map((o) => (
            <option key={o.tag} value={o.tag}>
              {o.displayName}
              {o.isDefault ? " (default)" : ""}
            </option>
          ))}
        </select>
      </Field>
      {adding ? (
        <div className="row-8">
          <input
            className="input input--mono"
            style={{ flex: 1, minWidth: 0 }}
            aria-label="New language tag"
            placeholder="BCP-47 tag, e.g. nl or pt-BR"
            value={tag}
            spellCheck={false}
            onChange={(e) => setTag(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                e.preventDefault();
                commit();
              }
            }}
          />
          <button type="button" className="btn btn--secondary btn--compact" onClick={commit}>
            Add
          </button>
          <button
            type="button"
            className="btn btn--ghost btn--compact"
            onClick={() => setAdding(false)}
          >
            Cancel
          </button>
        </div>
      ) : (
        <button
          type="button"
          className="btn btn--ghost btn--compact"
          onClick={() => setAdding(true)}
        >
          Add language
        </button>
      )}
    </div>
  );
}

// ---------- Per-key copy fields ----------

export function CopyFields({
  keys,
  draft,
  onChange,
  onReset,
}: {
  keys: CopyKey[];
  draft: CopyDraft;
  onChange: (key: string, value: string) => void;
  onReset: (key: string) => void;
}) {
  return (
    <>
      {keys.map((k) => {
        const value = draft[k.key] ?? "";
        // Longer strings (helper/legal/error copy) get a textarea.
        const multiline = k.maxLength > 90 || k.defaultValue.length > 60;
        return (
          <div className="field" key={k.key}>
            <div className="copyfield__head">
              <span className="field__label">{keyLabel(k.key)}</span>
              {value.trim() !== "" && (
                <button
                  type="button"
                  className="btn btn--ghost btn--compact"
                  onClick={() => onReset(k.key)}
                >
                  Reset
                </button>
              )}
            </div>
            {multiline ? (
              <textarea
                className="input"
                rows={2}
                aria-label={k.key}
                value={value}
                maxLength={k.maxLength || undefined}
                placeholder={k.defaultValue}
                onChange={(e) => onChange(k.key, e.target.value)}
              />
            ) : (
              <input
                className="input"
                aria-label={k.key}
                value={value}
                maxLength={k.maxLength || undefined}
                placeholder={k.defaultValue}
                onChange={(e) => onChange(k.key, e.target.value)}
              />
            )}
            {k.placeholders.length > 0 && (
              <span className="field__help">
                Must include {k.placeholders.map((p) => `{${p}}`).join(", ")}
              </span>
            )}
          </div>
        );
      })}
    </>
  );
}

// ---------- Copy revisions ----------

// CopyRevisions lists the saved copy revisions (one document across all
// locales/screens) and restores an old one, mirroring the theme editor.
export function CopyRevisions({
  project,
  currentRevisionId,
}: {
  project: Project;
  currentRevisionId: string;
}) {
  const revisions = useQuery(CopyService.method.listCopyRevisions, { projectId: project.id });
  const [target, setTarget] = useState<CopyRevision | null>(null);
  const restore = useMutation(CopyService.method.restoreCopyRevision, {
    onSuccess: () => {
      setTarget(null);
      invalidate(
        CopyService.method.getProjectCopy,
        CopyService.method.listCopyRevisions,
        CopyService.method.listLocales,
      );
    },
  });

  return (
    <section className="card card--pad stack-16">
      <h3 className="card__title">Revisions</h3>
      <p className="caption">
        Every save keeps a revision (the last 10). Restoring re-installs an old copy document as a
        new revision.
      </p>
      {revisions.isPending && <Loading />}
      {revisions.isError && <ErrorNote message={errorMessage(revisions.error)} />}
      {revisions.data &&
        (revisions.data.revisions.length === 0 ? (
          <p className="caption">No revisions yet — the project renders the bundled copy.</p>
        ) : (
          <table className="table">
            <thead>
              <tr>
                <th>Saved</th>
                <th>Languages</th>
                <th>Revision</th>
                <th />
              </tr>
            </thead>
            <tbody>
              {revisions.data.revisions.map((rev) => (
                <tr key={rev.revisionId}>
                  <td className="mono nowrap">{formatDateTime(rev.createTime)}</td>
                  <td>
                    {rev.locales.length > 0 ? (
                      rev.locales.join(", ")
                    ) : (
                      <span className="text-tertiary">—</span>
                    )}
                  </td>
                  <td className="mono">{rev.revisionId.slice(0, 8)}</td>
                  <td style={{ textAlign: "right" }}>
                    {rev.revisionId === currentRevisionId ? (
                      <Badge tone="accent">Current</Badge>
                    ) : (
                      <button
                        type="button"
                        className="btn btn--secondary btn--compact"
                        onClick={() => setTarget(rev)}
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

      <ConfirmDialog
        title="Restore revision"
        open={target !== null}
        onClose={() => setTarget(null)}
        onConfirm={() =>
          target && restore.mutate({ projectId: project.id, revisionId: target.revisionId })
        }
        confirmLabel="Restore"
        busy={restore.isPending}
        error={restore.isError ? errorMessage(restore.error) : undefined}
      >
        <p>
          Restore the copy saved {formatDateTime(target?.createTime)} (revision{" "}
          <span className="mono">{target?.revisionId.slice(0, 8)}</span>)? Unsaved edits in the
          editor are discarded.
        </p>
      </ConfirmDialog>
    </section>
  );
}

// ---------- Sign in / Sign up live preview ----------

// SignPreview is a phone-framed HTML/CSS replica of MothLoginScreen driven by
// the copy catalog: the same --p-* token contract as the theme editor's
// preview, with every string resolved through `get` (unsaved override →
// bundled default for the locale, interpolated). The sign-in and sign-up
// modes render distinct field sets from the sign_in.* / sign_up.* keys.
export function SignPreview({
  project,
  theme,
  scheme,
  mode,
  get,
}: {
  project: Project;
  theme: GetThemeResponse;
  scheme: "light" | "dark";
  mode: "sign_in" | "sign_up";
  get: (key: string, fallback?: string) => string;
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

  const providers = [
    ...(project.settings?.google?.enabled ? ["Continue with Google"] : []),
    ...(project.settings?.apple?.enabled ? ["Continue with Apple"] : []),
  ];

  const s = mode; // "sign_in" | "sign_up"
  const subtitle = get(`${s}.subtitle`);
  const switchPrompt =
    mode === "sign_in" ? get("sign_in.no_account") : get("sign_up.have_account");
  const switchLink =
    mode === "sign_in" ? get("sign_in.switch_to_sign_up") : get("sign_up.switch_to_sign_in");

  return (
    <div className="phone">
      <div className="phone__screen">
        <div className="mothpv" style={vars} data-scheme={scheme}>
          <div className="mothpv__card">
            {logo ? (
              <img className="mothpv__logo" src={logo} alt="" />
            ) : (
              <span className="mothpv__logo-fallback">
                {(project.name[0] ?? "A").toUpperCase()}
              </span>
            )}
            <div className="mothpv__title">{get(`${s}.title`)}</div>
            {subtitle && <div className="mothpv__subtitle">{subtitle}</div>}
            <div className="mothpv__field">
              <span className="mothpv__label">{get(`${s}.email_label`)}</span>
              <span className="mothpv__input">you@example.com</span>
            </div>
            <div className="mothpv__field">
              <span className="mothpv__label">{get(`${s}.password_label`)}</span>
              <span className="mothpv__input">••••••••</span>
            </div>
            <div className="mothpv__btn">{get(`${s}.submit`)}</div>
            {mode === "sign_in" && (
              <div className="mothpv__link">{get("sign_in.forgot_password")}</div>
            )}
            {mode === "sign_up" && <div className="mothpv__legal">{get("sign_up.legal")}</div>}
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
            <span className="mothpv__switch">
              {switchPrompt} <span className="mothpv__footer-link">{switchLink}</span>
            </span>
            <span>{project.name}</span>
          </div>
        </div>
      </div>
    </div>
  );
}
