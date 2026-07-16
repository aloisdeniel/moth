import {
  useEffect,
  useId,
  useRef,
  useState,
  type ReactNode,
} from "react";

// ---------- CopyButton ----------

export function CopyButton({ value, label }: { value: string; label?: string }) {
  const [copied, setCopied] = useState(false);
  const timer = useRef<ReturnType<typeof setTimeout>>(undefined);
  useEffect(() => () => clearTimeout(timer.current), []);
  return (
    <button
      type="button"
      className="btn btn--ghost btn--compact"
      onClick={() => {
        void navigator.clipboard.writeText(value);
        setCopied(true);
        clearTimeout(timer.current);
        timer.current = setTimeout(() => setCopied(false), 1500);
      }}
    >
      {copied ? "Copied" : (label ?? "Copy")}
    </button>
  );
}

// ---------- KeyWell ----------

export function KeyWell({ value, secret }: { value: string; secret?: boolean }) {
  return (
    <div className={secret ? "keywell keywell--secret" : "keywell"}>
      <span className="keywell__value">{value}</span>
      <CopyButton value={value} />
    </div>
  );
}

// ---------- CodeBlock ----------

export function CodeBlock({ code }: { code: string }) {
  return (
    <div className="codeblock">
      <pre>{code}</pre>
      <div className="codeblock__copy">
        <CopyButton value={code} />
      </div>
    </div>
  );
}

// ---------- Badge ----------

type BadgeTone = "neutral" | "success" | "warning" | "danger" | "accent";

export function Badge({ tone = "neutral", children }: { tone?: BadgeTone; children: ReactNode }) {
  const cls = tone === "neutral" ? "badge" : `badge badge--${tone}`;
  return <span className={cls}>{children}</span>;
}

// ---------- Status dot ----------

export function Status({
  tone,
  children,
}: {
  tone: "success" | "warning" | "danger" | "info" | "accent";
  children: ReactNode;
}) {
  return (
    <span className={`status status--${tone}`}>
      <span className="status__dot" />
      {children}
    </span>
  );
}

// ---------- Field ----------

export function Field({
  label,
  error,
  help,
  children,
}: {
  label: string;
  error?: string;
  help?: string;
  children: ReactNode;
}) {
  return (
    <label className="field">
      <span className="field__label">{label}</span>
      {children}
      {error ? <span className="field__error">{error}</span> : null}
      {help && !error ? <span className="field__help">{help}</span> : null}
    </label>
  );
}

// ---------- Dialog ----------

export function Dialog({
  title,
  open,
  onClose,
  wide,
  children,
}: {
  title: string;
  open: boolean;
  onClose: () => void;
  wide?: boolean;
  children: ReactNode;
}) {
  const titleId = useId();
  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, onClose]);
  if (!open) return null;
  return (
    <div
      className="dialog-overlay"
      onMouseDown={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div
        className={wide ? "dialog dialog--wide" : "dialog"}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
      >
        <div className="dialog__title" id={titleId}>
          {title}
        </div>
        {children}
      </div>
    </div>
  );
}

// ConfirmDialog is the destructive-action modal: it names the target and,
// when `confirmText` is set, requires typing it verbatim.
export function ConfirmDialog({
  title,
  open,
  onClose,
  onConfirm,
  confirmLabel,
  confirmText,
  busy,
  error,
  children,
}: {
  title: string;
  open: boolean;
  onClose: () => void;
  onConfirm: () => void;
  confirmLabel: string;
  confirmText?: string;
  busy?: boolean;
  error?: string;
  children: ReactNode;
}) {
  const [typed, setTyped] = useState("");
  useEffect(() => {
    if (open) setTyped("");
  }, [open]);
  const blocked = confirmText !== undefined && typed !== confirmText;
  return (
    <Dialog title={title} open={open} onClose={onClose}>
      <div className="stack-16">
        <div className="stack-8">{children}</div>
        {confirmText !== undefined && (
          <Field label={`Type "${confirmText}" to confirm`}>
            <input
              className="input input--mono"
              value={typed}
              onChange={(e) => setTyped(e.target.value)}
              autoFocus
              spellCheck={false}
              autoComplete="off"
            />
          </Field>
        )}
        {error ? <p className="field__error">{error}</p> : null}
        <div className="dialog__actions">
          <button type="button" className="btn btn--secondary" onClick={onClose}>
            Cancel
          </button>
          <button
            type="button"
            className="btn btn--danger"
            disabled={blocked || busy}
            onClick={onConfirm}
          >
            {busy ? "Working…" : confirmLabel}
          </button>
        </div>
      </div>
    </Dialog>
  );
}

// ---------- Password input with show/hide toggle ----------

export function PasswordInput({
  value,
  onChange,
  placeholder,
  autoComplete,
}: {
  value: string;
  onChange: (v: string) => void;
  placeholder?: string;
  autoComplete?: string;
}) {
  const [show, setShow] = useState(false);
  return (
    <div className="row-8">
      <input
        className="input"
        style={{ flex: 1 }}
        type={show ? "text" : "password"}
        value={value}
        placeholder={placeholder}
        autoComplete={autoComplete}
        onChange={(e) => onChange(e.target.value)}
      />
      <button
        type="button"
        className="btn btn--ghost btn--compact"
        onClick={() => setShow((s) => !s)}
      >
        {show ? "Hide" : "Show"}
      </button>
    </div>
  );
}

// ---------- StringListField ----------

// StringListField is a small multi-value editor: current values render as
// removable rows, plus an input that adds on Enter (intercepted so the
// surrounding form does not submit). Shared by the Providers redirect-scheme
// editor and the Settings email allow/block lists.
export function StringListField({
  label,
  values,
  onChange,
  placeholder,
  help,
}: {
  label: string;
  values: string[];
  onChange: (values: string[]) => void;
  placeholder?: string;
  help?: string;
}) {
  const [draft, setDraft] = useState("");

  function add() {
    const v = draft.trim();
    if (v === "" || values.includes(v)) return;
    onChange([...values, v]);
    setDraft("");
  }

  return (
    <div className="field">
      <span className="field__label">{label}</span>
      {values.map((v) => (
        <div key={v} className="keywell">
          <span className="keywell__value">{v}</span>
          <button
            type="button"
            className="btn btn--ghost btn--compact"
            onClick={() => onChange(values.filter((x) => x !== v))}
          >
            Remove
          </button>
        </div>
      ))}
      <div className="row-8">
        <input
          className="input input--mono"
          style={{ flex: 1 }}
          aria-label={label}
          value={draft}
          placeholder={placeholder}
          spellCheck={false}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") {
              e.preventDefault();
              add();
            }
          }}
        />
        <button type="button" className="btn btn--secondary btn--compact" onClick={add}>
          Add
        </button>
      </div>
      {help && <span className="field__help">{help}</span>}
    </div>
  );
}

// ---------- Loading / error placeholders ----------

export function Loading() {
  return <p className="caption">Loading…</p>;
}

export function ErrorNote({ message }: { message: string }) {
  return <p className="field__error">{message}</p>;
}
