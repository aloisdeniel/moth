import { useState } from "react";
import { useNavigate, useSearchParams } from "react-router";

import { Field, PasswordInput } from "../components/ui";

// Setup is the first-run screen: it redeems the one-time token printed on
// the server console and creates the initial admin account.
export function Setup() {
  const navigate = useNavigate();
  const [params] = useSearchParams();
  const [token, setToken] = useState(params.get("token") ?? "");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function submit() {
    setBusy(true);
    setError("");
    try {
      const resp = await fetch("/admin/setup", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ token, email, password }),
      });
      if (!resp.ok) {
        const body = (await resp.json().catch(() => ({}))) as { error?: string };
        setError(body.error ?? "Setup failed.");
        return;
      }
      void navigate("/");
    } catch {
      setError("The server is unreachable.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="auth-screen">
      <form
        className="auth-card"
        onSubmit={(e) => {
          e.preventDefault();
          void submit();
        }}
      >
        <div className="stack-8">
          <h3>Create the first admin</h3>
          <p className="caption">
            This account operates every project on this moth instance.
          </p>
        </div>
        <Field
          label="Setup token"
          help={params.get("token") ? undefined : "Printed on the server console at startup."}
        >
          <input
            className="input input--mono"
            value={token}
            onChange={(e) => setToken(e.target.value)}
            spellCheck={false}
            autoComplete="off"
          />
        </Field>
        <Field label="Email">
          <input
            className="input"
            type="email"
            autoCapitalize="none"
            autoComplete="username"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
          />
        </Field>
        <Field label="Password" help="At least 8 characters." error={error || undefined}>
          <PasswordInput value={password} onChange={setPassword} autoComplete="new-password" />
        </Field>
        <button className="btn btn--primary" type="submit" disabled={busy}>
          {busy ? "Creating…" : "Create admin account"}
        </button>
      </form>
    </div>
  );
}
