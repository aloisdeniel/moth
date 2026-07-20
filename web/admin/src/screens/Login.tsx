import { useMutation } from "@connectrpc/connect-query";
import { useEffect, useState } from "react";
import { useNavigate } from "react-router";

import { errorMessage, queryClient } from "../api";
import { Field, PasswordInput } from "../components/ui";
import { SessionService } from "../gen/moth/admin/v1/session_pb";

export function Login() {
  const navigate = useNavigate();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [needsSetup, setNeedsSetup] = useState(false);

  useEffect(() => {
    fetch("/admin/status")
      .then((r) => r.json())
      .then((s: { needsSetup?: boolean }) => setNeedsSetup(Boolean(s.needsSetup)))
      .catch(() => {});
  }, []);

  const login = useMutation(SessionService.method.login, {
    onSuccess: () => {
      queryClient.clear();
      void navigate("/");
    },
  });

  return (
    <div className="auth-screen">
      <form
        className="auth-card"
        onSubmit={(e) => {
          e.preventDefault();
          login.mutate({ email, password });
        }}
      >
        <div className="stack-8">
          <h3>Sign in to moth</h3>
          <p className="caption">The admin console of this instance.</p>
        </div>
        {needsSetup && (
          <p className="caption">
            No admin account exists yet. Open the setup link printed on the
            server console to create one.
          </p>
        )}
        <Field label="Email">
          <input
            className="input"
            type="email"
            autoCapitalize="none"
            autoComplete="username"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            autoFocus
          />
        </Field>
        <Field label="Password" error={login.isError ? errorMessage(login.error) : undefined}>
          <PasswordInput value={password} onChange={setPassword} autoComplete="current-password" />
        </Field>
        <button className="btn btn--primary" type="submit" disabled={login.isPending}>
          {login.isPending ? "Signing in…" : "Sign in"}
        </button>
      </form>
    </div>
  );
}
