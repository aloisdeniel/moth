// The SDK ships no CSS file to import: one <style> tag is injected
// (idempotently) the first time a moth surface renders. Every rule is
// scoped under `.moth-root` — the wrapper MothProvider only renders around
// moth-owned surfaces, never around the app's children — and consumes the
// theme's CSS custom properties, with light/dark resolved from the
// palette pairs per prefers-color-scheme.
const STYLE_ATTR = 'data-moth-styles';
const palette = (from) => [
    'primary',
    'on-primary',
    'background',
    'on-background',
    'surface',
    'on-surface',
    'error',
    'on-error',
]
    .map((role) => `--moth-${role}: var(--moth-${from}-${role});`)
    .join('\n  ');
const css = `
.moth-root {
  ${palette('l')}
  font-family: var(--moth-font, system-ui, sans-serif);
  font-size: calc(1rem * var(--moth-font-scale, 1));
  color: var(--moth-on-background);
  background: var(--moth-background);
  line-height: 1.5;
  box-sizing: border-box;
}
@media (prefers-color-scheme: dark) {
  .moth-root {
    ${palette('d')}
  }
}
.moth-root *, .moth-root *::before, .moth-root *::after { box-sizing: inherit; }

.moth-screen {
  min-height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: calc(var(--moth-unit) * 3);
}
.moth-content {
  width: 100%;
  max-width: 400px;
  display: flex;
  flex-direction: column;
  gap: calc(var(--moth-unit) * 2);
}
.moth-content--wide { max-width: 480px; }
.moth-center { text-align: center; }

.moth-logo { display: block; margin: 0 auto; max-height: calc(var(--moth-unit) * 8); max-width: 100%; }
.moth-logo--light { display: block; }
.moth-logo--dark { display: none; }
@media (prefers-color-scheme: dark) {
  .moth-root .moth-logo--light { display: none; }
  .moth-root .moth-logo--dark { display: block; }
}

.moth-title { margin: 0; font-size: 1.6em; font-weight: 600; text-align: center; }
.moth-subtitle { margin: 0; text-align: center; opacity: 0.72; }

.moth-field { display: flex; flex-direction: column; gap: calc(var(--moth-unit) * 0.5); }
.moth-label { font-size: 0.875em; font-weight: 500; }
.moth-input {
  font: inherit;
  color: inherit;
  background: var(--moth-surface);
  border: 1px solid color-mix(in srgb, var(--moth-on-surface) 45%, transparent);
  border-radius: var(--moth-radius);
  padding: calc(var(--moth-unit) * 1.25) calc(var(--moth-unit) * 1.5);
  outline: none;
  width: 100%;
}
.moth-input:focus { border-color: var(--moth-primary); box-shadow: 0 0 0 1px var(--moth-primary); }
.moth-input[aria-invalid='true'] { border-color: var(--moth-error); }
.moth-field-error { color: var(--moth-error); font-size: 0.8em; }

.moth-btn {
  font: inherit;
  font-weight: 500;
  border: none;
  border-radius: var(--moth-radius);
  min-height: calc(var(--moth-unit) * 6);
  padding: 0 calc(var(--moth-unit) * 2);
  background: var(--moth-primary);
  color: var(--moth-on-primary);
  cursor: pointer;
  width: 100%;
}
.moth-btn:disabled { opacity: 0.55; cursor: default; }
.moth-btn-outline {
  background: transparent;
  color: var(--moth-on-surface);
  border: 1px solid color-mix(in srgb, var(--moth-on-surface) 45%, transparent);
}
.moth-btn-text {
  font: inherit;
  background: none;
  border: none;
  color: var(--moth-primary);
  cursor: pointer;
  padding: calc(var(--moth-unit) * 0.5);
  border-radius: var(--moth-radius);
}
.moth-btn-text:disabled { opacity: 0.55; cursor: default; }
.moth-link-muted { color: inherit; opacity: 0.72; font-size: 0.85em; text-decoration: none; }
.moth-link-muted:hover { text-decoration: underline; }

.moth-banner {
  border-radius: var(--moth-radius);
  padding: calc(var(--moth-unit) * 1.5);
  font-size: 0.9em;
}
.moth-banner--error { background: var(--moth-error); color: var(--moth-on-error); }
.moth-banner--info {
  background: color-mix(in srgb, var(--moth-primary) 14%, var(--moth-surface));
  color: var(--moth-on-surface);
}

.moth-row { display: flex; align-items: center; justify-content: center; gap: calc(var(--moth-unit) * 0.5); flex-wrap: wrap; }
.moth-row--end { justify-content: flex-end; }
.moth-divider { display: flex; align-items: center; gap: calc(var(--moth-unit) * 1.5); opacity: 0.6; font-size: 0.85em; }
.moth-divider::before, .moth-divider::after {
  content: '';
  flex: 1;
  border-top: 1px solid color-mix(in srgb, var(--moth-on-surface) 22%, transparent);
}

.moth-spinner {
  width: calc(var(--moth-unit) * 4);
  height: calc(var(--moth-unit) * 4);
  margin: calc(var(--moth-unit) * 2) auto;
  border-radius: 50%;
  border: 3px solid color-mix(in srgb, var(--moth-primary) 25%, transparent);
  border-top-color: var(--moth-primary);
  animation: moth-spin 0.8s linear infinite;
}
@keyframes moth-spin { to { transform: rotate(360deg); } }

.moth-benefits { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: var(--moth-unit); }
.moth-benefits li { display: flex; gap: calc(var(--moth-unit) * 1.5); align-items: baseline; }
.moth-benefits li::before { content: '✓'; color: var(--moth-primary); font-weight: 700; }

.moth-tiers { display: flex; flex-direction: column; gap: calc(var(--moth-unit) * 1.5); }
.moth-tiers--tiles { flex-direction: row; align-items: stretch; }
.moth-tiers--tiles .moth-tier { flex: 1 1 0; }
@media (max-width: 480px) { .moth-root .moth-tiers--tiles { flex-direction: column; } }

.moth-tier {
  font: inherit;
  color: inherit;
  text-align: left;
  display: flex;
  gap: calc(var(--moth-unit) * 1.5);
  align-items: flex-start;
  background: var(--moth-surface);
  border: 1px solid color-mix(in srgb, var(--moth-on-surface) 22%, transparent);
  border-radius: var(--moth-radius);
  padding: calc(var(--moth-unit) * 2);
  cursor: pointer;
}
.moth-tier[aria-checked='true'] {
  border: 2px solid var(--moth-primary);
  background: color-mix(in srgb, var(--moth-primary) 8%, var(--moth-surface));
  padding: calc(var(--moth-unit) * 2 - 1px);
}
.moth-tier--highlighted { border-color: color-mix(in srgb, var(--moth-primary) 50%, transparent); }
.moth-tier--unavailable { opacity: 0.55; cursor: default; }
.moth-tier-body { flex: 1; display: flex; flex-direction: column; gap: calc(var(--moth-unit) * 0.75); }
.moth-tier-line { display: flex; justify-content: space-between; gap: var(--moth-unit); align-items: baseline; }
.moth-tier-name { font-weight: 600; }
.moth-tier-price { font-weight: 600; white-space: nowrap; }
.moth-tier-note { font-size: 0.8em; opacity: 0.72; }

.moth-badge {
  display: inline-block;
  font-size: 0.72em;
  font-weight: 600;
  padding: calc(var(--moth-unit) * 0.25) var(--moth-unit);
  border-radius: calc(var(--moth-radius) / 2);
}
.moth-badge--primary { background: var(--moth-primary); color: var(--moth-on-primary); }
.moth-badge--soft {
  background: color-mix(in srgb, var(--moth-primary) 16%, var(--moth-surface));
  color: var(--moth-on-surface);
}

.moth-segments {
  display: flex;
  border: 1px solid color-mix(in srgb, var(--moth-on-surface) 30%, transparent);
  border-radius: var(--moth-radius);
  overflow: hidden;
}
.moth-segments button {
  font: inherit;
  flex: 1;
  border: none;
  background: transparent;
  color: inherit;
  padding: var(--moth-unit);
  cursor: pointer;
}
.moth-segments button[aria-pressed='true'] { background: var(--moth-primary); color: var(--moth-on-primary); }

.moth-footer { display: flex; flex-direction: column; align-items: center; gap: calc(var(--moth-unit) * 0.5); font-size: 0.85em; }
.moth-empty, .moth-error-state { display: flex; flex-direction: column; align-items: center; gap: var(--moth-unit); text-align: center; }
`;
/**
 * Injects the SDK stylesheet once per document. Keyed by a data attribute,
 * so calling it from every surface render stays idempotent.
 */
export function ensureMothStyles() {
    if (typeof document === 'undefined')
        return;
    if (document.head.querySelector(`style[${STYLE_ATTR}]`) !== null)
        return;
    const style = document.createElement('style');
    style.setAttribute(STYLE_ATTR, '');
    style.textContent = css;
    document.head.appendChild(style);
}
