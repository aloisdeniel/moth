import { useMutation, useQuery } from "@connectrpc/connect-query";
import { useRef, useState, type CSSProperties } from "react";

import { errorMessage, invalidate } from "../api";
import { Badge, ConfirmDialog, ErrorNote, Field, Loading } from "../components/ui";
import { CopyScreen } from "../gen/moth/admin/v1/copy_pb";
import type { Project } from "../gen/moth/admin/v1/project_pb";
import type { GetThemeResponse, ThemeRevision } from "../gen/moth/admin/v1/theme_pb";
import { LogoVariant, ThemeService } from "../gen/moth/admin/v1/theme_pb";
import { formatDateTime } from "../lib/format";
import {
  COLOR_PAIRS,
  FONT_FAMILIES,
  MAX_CORNER_RADIUS,
  MAX_LOGO_BYTES,
  MAX_SCALE,
  MAX_SPACING_UNIT,
  MIN_CONTRAST,
  MIN_SCALE,
  MIN_SPACING_UNIT,
  contrastIssues,
  contrastRatio,
  deriveDark,
  editorFromProto,
  editorToProto,
  effectiveDark,
  ensurePreviewFonts,
  fontStack,
  isHexColor,
  normalizeHex,
  type ColorKey,
  type EditorTheme,
  type Palette,
} from "../lib/theme";
import { SignInUpTab } from "./ProjectCopy";
import { PaywallEditor } from "./ProjectPaywall";

// The Design tab's sub-tabs (milestone 15): the theme-token editor first,
// then one screen per SDK surface an operator can localize/preview.
const SUBTABS = [
  { id: "theme", label: "Theme" },
  { id: "sign_in", label: "Sign in" },
  { id: "sign_up", label: "Sign up" },
  { id: "paywall", label: "Paywall" },
] as const;

type SubTab = (typeof SUBTABS)[number]["id"];

// ProjectDesign is the design tab: a sub-tab bar over the shared project
// theme. "Theme" is the milestone-06 token editor; "Sign in"/"Sign up" are the
// per-language copy editors with a live login-screen preview; "Paywall" keeps
// the milestone-13 layout/offering config and adds a copy editor. Each sub-tab
// renders a single live preview so only one is on screen at a time.
export function ProjectDesign({ project }: { project: Project }) {
  // The preview renders with the real embedded fonts, not local lookalikes.
  ensurePreviewFonts();
  const [tab, setTab] = useState<SubTab>("theme");
  const theme = useQuery(ThemeService.method.getTheme, { projectId: project.id });

  if (theme.isPending) return <Loading />;
  if (theme.isError) return <ErrorNote message={errorMessage(theme.error)} />;
  return (
    <div className="stack-24">
      <div className="seg" role="group" aria-label="Design screen">
        {SUBTABS.map((s) => (
          <button
            key={s.id}
            type="button"
            className="seg__btn"
            aria-pressed={tab === s.id}
            onClick={() => setTab(s.id)}
          >
            {s.label}
          </button>
        ))}
      </div>
      {tab === "theme" && <DesignEditor project={project} current={theme.data} />}
      {tab === "sign_in" && (
        <SignInUpTab project={project} theme={theme.data} screen={CopyScreen.SIGN_IN} />
      )}
      {tab === "sign_up" && (
        <SignInUpTab project={project} theme={theme.data} screen={CopyScreen.SIGN_UP} />
      )}
      {tab === "paywall" && <PaywallEditor project={project} theme={theme.data} />}
    </div>
  );
}

function DesignEditor({ project, current }: { project: Project; current: GetThemeResponse }) {
  const [t, setT] = useState<EditorTheme>(() => editorFromProto(current.theme));
  const [scheme, setScheme] = useState<"light" | "dark">("light");
  const [saved, setSaved] = useState(false);
  const [resetOpen, setResetOpen] = useState(false);
  const [restoreTarget, setRestoreTarget] = useState<ThemeRevision | null>(null);

  const revisions = useQuery(ThemeService.method.listThemeRevisions, {
    projectId: project.id,
  });

  const refresh = () =>
    invalidate(ThemeService.method.getTheme, ThemeService.method.listThemeRevisions);

  const update = useMutation(ThemeService.method.updateTheme, {
    onSuccess: () => {
      refresh();
      setSaved(true);
      setTimeout(() => setSaved(false), 2000);
    },
  });
  const restore = useMutation(ThemeService.method.restoreThemeRevision, {
    onSuccess: (res) => {
      setT(editorFromProto(res.theme));
      setRestoreTarget(null);
      refresh();
    },
  });
  const reset = useMutation(ThemeService.method.resetTheme, {
    onSuccess: (res) => {
      setT(editorFromProto(res.theme));
      setResetOpen(false);
      refresh();
    },
  });

  const issues = contrastIssues(t);

  function setColor(group: "colors" | "dark", key: ColorKey, value: string) {
    setT((prev) => ({ ...prev, [group]: { ...prev[group], [key]: value } }));
  }

  function toggleDark(enabled: boolean) {
    // Entering override mode seeds the group with the currently derived
    // palette, so enabling it changes nothing until a field is edited.
    setT((prev) => ({ ...prev, darkEnabled: enabled, dark: deriveDark(prev.colors) }));
  }

  function save() {
    update.mutate({ projectId: project.id, theme: editorToProto(t) });
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
          <h3 className="card__title">Colors</h3>
          <p className="caption">
            Each color pairs with an “on” color used for text and icons drawn on top of it. Every
            pair must reach a WCAG AA contrast ratio of {MIN_CONTRAST}:1 — the editor warns live
            and the server rejects anything below it.
          </p>
          {COLOR_PAIRS.map((pair) => (
            <ColorPairRow
              key={pair.role}
              labels={[pair.label, pair.onLabel]}
              palette={t.colors}
              pair={pair}
              onChange={(key, value) => setColor("colors", key, value)}
            />
          ))}

          <label className="check">
            <input
              type="checkbox"
              checked={t.darkEnabled}
              onChange={(e) => toggleDark(e.target.checked)}
            />
            <span>
              Customize the dark palette
              <span className="caption" style={{ display: "block" }}>
                Off, the dark palette is derived automatically: backgrounds shift toward black,
                accents toward white, and text picks black or white for maximum contrast.
              </span>
            </span>
          </label>
          {t.darkEnabled &&
            COLOR_PAIRS.map((pair) => (
              <ColorPairRow
                key={`dark-${pair.role}`}
                labels={[`Dark ${pair.label.toLowerCase()}`, `Dark ${pair.onLabel.toLowerCase()}`]}
                palette={t.dark}
                pair={pair}
                onChange={(key, value) => setColor("dark", key, value)}
              />
            ))}
        </section>

        <section className="card card--pad stack-16">
          <h3 className="card__title">Typography</h3>
          <Field label="Font family" help="One of the open-license fonts embedded in moth.">
            <select
              className="select"
              value={t.fontFamily}
              onChange={(e) => setT((prev) => ({ ...prev, fontFamily: e.target.value }))}
            >
              {FONT_FAMILIES.map((f) => (
                <option key={f.name} value={f.name}>
                  {f.name}
                </option>
              ))}
            </select>
          </Field>
          <SliderField
            label="Text scale"
            value={t.scale}
            min={MIN_SCALE}
            max={MAX_SCALE}
            step={0.05}
            format={(v) => `${v.toFixed(2)}×`}
            onChange={(v) => setT((prev) => ({ ...prev, scale: v }))}
          />
        </section>

        <section className="card card--pad stack-16">
          <h3 className="card__title">Shape &amp; spacing</h3>
          <SliderField
            label="Corner radius"
            value={t.cornerRadius}
            min={0}
            max={MAX_CORNER_RADIUS}
            step={1}
            format={(v) => `${v} px`}
            onChange={(v) => setT((prev) => ({ ...prev, cornerRadius: v }))}
          />
          <SliderField
            label="Spacing unit"
            value={t.spacingUnit}
            min={MIN_SPACING_UNIT}
            max={MAX_SPACING_UNIT}
            step={1}
            format={(v) => `${v} px`}
            onChange={(v) => setT((prev) => ({ ...prev, spacingUnit: v }))}
          />
        </section>

        <section className="card card--pad stack-16">
          <h3 className="card__title">Logo</h3>
          <p className="caption">
            PNG or SVG, at most 512 KiB. Images are re-encoded server-side; uploading replaces the
            previous file immediately (no separate save).
          </p>
          <LogoField
            project={project}
            variant={LogoVariant.LIGHT}
            label="Light logo"
            path={current.theme?.logo?.lightPath ?? ""}
            surface={t.colors.background}
          />
          <LogoField
            project={project}
            variant={LogoVariant.DARK}
            label="Dark logo"
            path={current.theme?.logo?.darkPath ?? ""}
            surface={effectiveDark(t).background}
          />
        </section>

        <section className="card card--pad stack-16">
          <h3 className="card__title">Legal links</h3>
          <Field
            label="Terms of service URL"
            help="Optional; rendered in the login screen footer. Absolute http(s) URL."
          >
            <input
              className="input"
              type="url"
              value={t.termsUrl}
              onChange={(e) => setT((prev) => ({ ...prev, termsUrl: e.target.value }))}
              placeholder="https://example.com/terms"
              spellCheck={false}
            />
          </Field>
          <Field label="Privacy policy URL" help="Optional; rendered next to the terms link.">
            <input
              className="input"
              type="url"
              value={t.privacyUrl}
              onChange={(e) => setT((prev) => ({ ...prev, privacyUrl: e.target.value }))}
              placeholder="https://example.com/privacy"
              spellCheck={false}
            />
          </Field>
        </section>

        <div className="stack-8">
          {issues.length > 0 && (
            <p className="field__error">
              Fix the failing contrast pairs before saving — every color must reach{" "}
              {MIN_CONTRAST}:1 against its “on” color (
              {issues.map((i) => `${i.scheme} ${i.label}`).join(", ")}).
            </p>
          )}
          <div className="row-12">
            <button
              type="submit"
              className="btn btn--primary"
              disabled={update.isPending || issues.length > 0}
            >
              {update.isPending ? "Saving…" : "Save theme"}
            </button>
            {saved && <span className="caption text-success">Saved.</span>}
            {update.isError && <span className="field__error">{errorMessage(update.error)}</span>}
          </div>
        </div>

        <section className="card card--pad stack-16">
          <h3 className="card__title">Revisions</h3>
          <p className="caption">
            Every save keeps a revision (the last 10). Restoring re-installs an old token set as a
            new revision.
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
                    <th>Primary</th>
                    <th>Revision</th>
                    <th />
                  </tr>
                </thead>
                <tbody>
                  {revisions.data.revisions.map((rev) => (
                    <tr key={rev.revisionId}>
                      <td className="mono nowrap">{formatDateTime(rev.createTime)}</td>
                      <td>
                        <span className="row-8">
                          <span
                            className="design__revswatch"
                            style={{ background: rev.theme?.colors?.primary }}
                          />
                          <span className="mono">{rev.theme?.colors?.primary}</span>
                        </span>
                      </td>
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
                Revert to the built-in theme. Saved revisions are kept, so the current theme stays
                restorable.
              </span>
            </div>
            <button
              type="button"
              className="btn btn--danger"
              onClick={() => setResetOpen(true)}
            >
              Reset theme
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
        <LoginPreview project={project} t={t} scheme={scheme} theme={current} />
        <p className="caption" style={{ textAlign: "center" }}>
          Live preview of the SDK login screen, rendered from the unsaved editor state.
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
          Restore the theme saved {formatDateTime(restoreTarget?.createTime)} (revision{" "}
          <span className="mono">{restoreTarget?.revisionId.slice(0, 8)}</span>)? Unsaved edits in
          the editor are discarded.
        </p>
      </ConfirmDialog>

      <ConfirmDialog
        title="Reset theme"
        open={resetOpen}
        onClose={() => setResetOpen(false)}
        onConfirm={() => reset.mutate({ projectId: project.id })}
        confirmLabel="Reset theme"
        busy={reset.isPending}
        error={reset.isError ? errorMessage(reset.error) : undefined}
      >
        <p>
          Reset <strong>{project.name}</strong> to the built-in default theme? The revision
          history is kept, so the current theme can be restored.
        </p>
      </ConfirmDialog>
    </div>
  );
}

// ---------- Color editing ----------

// ColorPairRow renders a color and its "on" counterpart side by side with a
// live WCAG contrast badge for the pair.
function ColorPairRow({
  labels,
  palette,
  pair,
  onChange,
}: {
  labels: [string, string];
  palette: Palette;
  pair: (typeof COLOR_PAIRS)[number];
  onChange: (key: ColorKey, value: string) => void;
}) {
  const ratio = contrastRatio(palette[pair.role], palette[pair.on]);
  const passes = ratio >= MIN_CONTRAST;
  return (
    <div className="design__pair">
      <ColorInput
        label={labels[0]}
        value={palette[pair.role]}
        onChange={(v) => onChange(pair.role, v)}
      />
      <ColorInput
        label={labels[1]}
        value={palette[pair.on]}
        onChange={(v) => onChange(pair.on, v)}
      />
      <span className="design__ratio">
        <Badge tone={passes ? "success" : "danger"}>
          {ratio.toFixed(1)}:1 {passes ? "AA" : "fails AA"}
        </Badge>
      </span>
    </div>
  );
}

// ColorInput keeps a native color picker and a hex text field in sync. The
// text field holds a local draft so partial input doesn't destroy the
// value; only a valid #RRGGBB commits upstream.
function ColorInput({
  label,
  value,
  onChange,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
}) {
  const [draft, setDraft] = useState(value);
  const [lastValue, setLastValue] = useState(value);
  if (value !== lastValue) {
    // External change (picker, restore, reset): resync the draft.
    setLastValue(value);
    setDraft(value);
  }
  const valid = isHexColor(draft);
  return (
    <div className="field">
      <span className="field__label">{label}</span>
      <div className="row-8">
        <input
          type="color"
          className="design__swatch"
          aria-label={`${label} picker`}
          value={value}
          onChange={(e) => onChange(normalizeHex(e.target.value))}
        />
        <input
          className={valid ? "input input--mono" : "input input--mono input--error"}
          style={{ flex: 1, minWidth: 0 }}
          aria-label={label}
          value={draft}
          spellCheck={false}
          onChange={(e) => {
            const v = e.target.value;
            setDraft(v);
            if (isHexColor(v)) {
              const normalized = normalizeHex(v);
              setLastValue(normalized);
              onChange(normalized);
            }
          }}
        />
      </div>
    </div>
  );
}

// ---------- Sliders ----------

function SliderField({
  label,
  value,
  min,
  max,
  step,
  format,
  onChange,
}: {
  label: string;
  value: number;
  min: number;
  max: number;
  step: number;
  format: (v: number) => string;
  onChange: (v: number) => void;
}) {
  return (
    <div className="field">
      <span className="field__label">{label}</span>
      <div className="row-12">
        <input
          type="range"
          className="slider"
          aria-label={label}
          min={min}
          max={max}
          step={step}
          value={value}
          onChange={(e) => onChange(Number(e.target.value))}
        />
        <span className="mono tabular design__slider-value">{format(value)}</span>
      </div>
    </div>
  );
}

// ---------- Logo upload ----------

function LogoField({
  project,
  variant,
  label,
  path,
  surface,
}: {
  project: Project;
  variant: LogoVariant;
  label: string;
  path: string;
  surface: string;
}) {
  const fileInput = useRef<HTMLInputElement>(null);
  const [fileError, setFileError] = useState("");

  const refresh = () =>
    invalidate(ThemeService.method.getTheme, ThemeService.method.listThemeRevisions);
  const upload = useMutation(ThemeService.method.uploadLogo, { onSuccess: refresh });
  const remove = useMutation(ThemeService.method.deleteLogo, { onSuccess: refresh });

  async function onFile(file: File) {
    setFileError("");
    if (file.type !== "image/png" && file.type !== "image/svg+xml") {
      setFileError("PNG or SVG only.");
      return;
    }
    if (file.size > MAX_LOGO_BYTES) {
      setFileError("File is too large — the limit is 512 KiB.");
      return;
    }
    const data = new Uint8Array(await file.arrayBuffer());
    upload.mutate({ projectId: project.id, variant, data, contentType: file.type });
  }

  const rpcError = upload.isError
    ? errorMessage(upload.error)
    : remove.isError
      ? errorMessage(remove.error)
      : "";

  return (
    <div className="field">
      <span className="field__label">{label}</span>
      <div className="design__logo">
        <span className="design__logo-thumb" style={{ background: surface }}>
          {path ? (
            <img src={path} alt={`${project.name} logo`} />
          ) : (
            <span className="caption text-tertiary">None</span>
          )}
        </span>
        <input
          ref={fileInput}
          type="file"
          accept="image/png,image/svg+xml"
          hidden
          onChange={(e) => {
            const f = e.target.files?.[0];
            if (f) void onFile(f);
            e.target.value = "";
          }}
        />
        <button
          type="button"
          className="btn btn--secondary btn--compact"
          disabled={upload.isPending}
          onClick={() => fileInput.current?.click()}
        >
          {upload.isPending ? "Uploading…" : path ? "Replace" : "Upload"}
        </button>
        {path && (
          <button
            type="button"
            className="btn btn--ghost btn--compact"
            disabled={remove.isPending}
            onClick={() => remove.mutate({ projectId: project.id, variant })}
          >
            {remove.isPending ? "Removing…" : "Remove"}
          </button>
        )}
      </div>
      {(fileError || rpcError) && <span className="field__error">{fileError || rpcError}</span>}
    </div>
  );
}

// ---------- Live preview ----------

// LoginPreview is a phone-framed HTML/CSS replica of MothLoginScreen. The
// inner screen is styled exclusively through --p-* custom properties set
// from the editor state, so it shares the theme's token semantics (colors,
// font stack, radius, spacing unit) rather than the admin's own tokens.
function LoginPreview({
  project,
  t,
  scheme,
  theme,
}: {
  project: Project;
  t: EditorTheme;
  scheme: "light" | "dark";
  theme: GetThemeResponse;
}) {
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
  const legal = [
    ...(t.termsUrl.trim() ? ["Terms"] : []),
    ...(t.privacyUrl.trim() ? ["Privacy"] : []),
  ];

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
            <div className="mothpv__title">Sign in to {project.name}</div>
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
            {legal.length > 0 && (
              <span>
                {legal.map((l, i) => (
                  <span key={l}>
                    {i > 0 && " · "}
                    <span className="mothpv__footer-link">{l}</span>
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
