import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter } from "react-router";
import { TransportProvider } from "@connectrpc/connect-query";
import { QueryClientProvider } from "@tanstack/react-query";

import { App } from "./App";
import { queryClient, transport } from "./api";
import "./styles/app.css";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <TransportProvider transport={transport}>
      <QueryClientProvider client={queryClient}>
        <BrowserRouter basename="/admin">
          <App />
        </BrowserRouter>
      </QueryClientProvider>
    </TransportProvider>
  </StrictMode>,
);
