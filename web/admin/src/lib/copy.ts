import type { CopyKey } from "../gen/moth/admin/v1/copy_pb";

// Client-side helpers for the localization copy editors (milestone 15).
// Mirrors internal/i18n's literal-placeholder interpolation so the admin
// preview renders the same strings the SDK and hosted pages will; the server
// re-validates every override on save (key exists, required placeholders,
// length bound). Keep in sync with internal/i18n.

// interpolate performs the catalog's literal {name} -> value substitution.
// Unknown placeholders are left untouched, exactly like the server.
export function interpolate(s: string, vars: Record<string, string>): string {
  return s.replace(/\{([a-z0-9_]+)\}/gi, (m, name: string) =>
    Object.prototype.hasOwnProperty.call(vars, name) ? vars[name] : m,
  );
}

// keyLabel turns a dotted catalog key into a human field label, e.g.
// "sign_up.email_label" -> "Email label".
export function keyLabel(key: string): string {
  const name = key.includes(".") ? key.slice(key.indexOf(".") + 1) : key;
  const words = name.replace(/_/g, " ");
  return words.charAt(0).toUpperCase() + words.slice(1);
}

// CopyDraft is the editor's working copy for one locale: key -> override
// string (empty means "fall back to the bundled default").
export type CopyDraft = Record<string, string>;

// draftFromKeys seeds the draft from a GetProjectCopy response.
export function draftFromKeys(keys: CopyKey[]): CopyDraft {
  const d: CopyDraft = {};
  for (const k of keys) d[k.key] = k.overrideValue;
  return d;
}

// draftValues is the UpdateProjectCopy payload for the screen being edited:
// every key the editor holds, trimmed, INCLUDING emptied ones. The server
// merges key-by-key into the locale document — a non-empty value upserts the
// override, an empty value clears it — so saving one screen's draft never
// clobbers another screen's overrides for the same locale (they aren't in the
// payload) yet a field the operator cleared still reverts to the bundled default.
export function draftValues(draft: CopyDraft): Record<string, string> {
  const out: Record<string, string> = {};
  for (const [k, v] of Object.entries(draft)) out[k] = v.trim();
  return out;
}

// copyResolver builds a key -> effective, interpolated string lookup for the
// live preview: the operator's unsaved override when set, else the bundled
// default for the locale, with placeholders filled from vars.
export function copyResolver(
  keys: CopyKey[],
  draft: CopyDraft,
  vars: Record<string, string>,
): (key: string, fallback?: string) => string {
  const defaults = new Map<string, string>();
  for (const k of keys) defaults.set(k.key, k.defaultValue);
  return (key, fallback = "") => {
    const override = (draft[key] ?? "").trim();
    const base = override !== "" ? override : (defaults.get(key) ?? fallback);
    return interpolate(base, vars);
  };
}
