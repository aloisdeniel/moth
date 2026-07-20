import { useMutation, useQuery } from "@connectrpc/connect-query";
import { useEffect } from "react";
import { Link, Navigate, Outlet, useLocation, useNavigate } from "react-router";

import { invalidate, isUnauthenticated, queryClient } from "../api";
import { SessionService } from "../gen/moth/admin/v1/session_pb";
import { Loading } from "../components/ui";
import { Logo } from "../components/Logo";

// Shell wraps every authenticated screen: top bar, session guard, logout.
export function Shell() {
  const location = useLocation();
  const navigate = useNavigate();
  const me = useQuery(SessionService.method.getCurrentAdmin);
  const logout = useMutation(SessionService.method.logout, {
    onSettled: () => {
      queryClient.clear();
      void navigate("/login");
    },
  });

  // A fresh instance (?setup=TOKEN printed on the server console) goes
  // straight to the first-run screen.
  const params = new URLSearchParams(location.search);
  const setupToken = params.get("setup");
  useEffect(() => {
    if (setupToken) {
      void navigate(`/setup?token=${encodeURIComponent(setupToken)}`, { replace: true });
    }
  }, [setupToken, navigate]);

  if (me.isPending) {
    return (
      <div className="auth-screen">
        <Loading />
      </div>
    );
  }
  if (me.isError) {
    if (isUnauthenticated(me.error)) {
      return <Navigate to="/login" replace />;
    }
    return (
      <div className="auth-screen">
        <p className="field__error">The server is unreachable. Retry in a moment.</p>
      </div>
    );
  }

  return (
    <>
      <header className="topbar">
        <Link to="/" className="topbar__brand">
          <Logo size={15} />
          moth
        </Link>
        <span className="topbar__sep">/</span>
        <Link to="/" className="caption" style={{ color: "var(--text-secondary)" }}>
          Projects
        </Link>
        <span className="topbar__spacer" />
        <Link to="/audit" className="caption" style={{ color: "var(--text-secondary)" }}>
          Audit
        </Link>
        <Link to="/settings" className="caption" style={{ color: "var(--text-secondary)" }}>
          Instance settings
        </Link>
        <span className="caption mono">{me.data.admin?.email}</span>
        <button
          type="button"
          className="btn btn--ghost btn--compact"
          onClick={() => {
            logout.mutate({});
            invalidate();
          }}
        >
          Log out
        </button>
      </header>
      <Outlet />
    </>
  );
}
