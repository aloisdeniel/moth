import { useMutation } from "@connectrpc/connect-query";
import { useState } from "react";
import { useNavigate, useSearchParams } from "react-router";

import { errorMessage, queryClient } from "../api";
import { Field, PasswordInput } from "../components/ui";
import { AdminAccountService } from "../gen/moth/admin/v1/account_pb";

// InviteAccept redeems an operator invite link (/admin/invite?token=…).
export function InviteAccept() {
  const navigate = useNavigate();
  const [params] = useSearchParams();
  const token = params.get("token") ?? "";
  const [password, setPassword] = useState("");

  const accept = useMutation(AdminAccountService.method.acceptAdminInvite, {
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
          accept.mutate({ token, password });
        }}
      >
        <div className="stack-8">
          <h3>Activate your admin account</h3>
          <p className="caption">
            You were invited to administer this moth instance. Choose a
            password to finish.
          </p>
        </div>
        {token === "" && (
          <p className="field__error">
            This page needs the invite link from your email or from the admin
            who invited you.
          </p>
        )}
        <Field
          label="Password"
          help="At least 8 characters."
          error={accept.isError ? errorMessage(accept.error) : undefined}
        >
          <PasswordInput value={password} onChange={setPassword} autoComplete="new-password" />
        </Field>
        <button
          className="btn btn--primary"
          type="submit"
          disabled={accept.isPending || token === ""}
        >
          {accept.isPending ? "Activating…" : "Activate account"}
        </button>
      </form>
    </div>
  );
}
