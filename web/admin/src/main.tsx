import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter, HashRouter } from "react-router";
import { TransportProvider } from "@connectrpc/connect-query";
import { QueryClientProvider } from "@tanstack/react-query";

import { App } from "./App";
import { queryClient, transport } from "./api";
import { DemoBadge } from "./demo";
import "./styles/app.css";

// Demo builds are static files on the website (no server to rewrite deep
// links) and are mounted under an arbitrary base path, so they route in the
// URL hash. Production keeps clean /admin URLs, rewritten by the moth server.
const Router = import.meta.env.VITE_DEMO ? HashRouter : BrowserRouter;

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <TransportProvider transport={transport}>
      <QueryClientProvider client={queryClient}>
        <Router basename={import.meta.env.VITE_DEMO ? undefined : "/admin"}>
          <App />
        </Router>
        {import.meta.env.VITE_DEMO ? <DemoBadge /> : null}
      </QueryClientProvider>
    </TransportProvider>
  </StrictMode>,
);
