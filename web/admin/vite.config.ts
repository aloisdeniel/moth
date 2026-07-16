import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

// The SPA is served by the moth binary at /admin (embedded via go:embed).
// `vite build` writes straight into the Go package that embeds it.
export default defineConfig({
  plugins: [react()],
  base: "/admin/",
  build: {
    outDir: "../../internal/server/web/dist",
    emptyOutDir: true,
  },
  server: {
    // `make dev`: the Go server runs on :8080; everything that is not a
    // static SPA asset is proxied to it (connect RPCs, setup endpoints,
    // hosted pages, JWKS, proto downloads).
    proxy: {
      "^/moth\\..*": "http://localhost:8080",
      "/admin/status": "http://localhost:8080",
      "/admin/setup": "http://localhost:8080",
      "/protos": "http://localhost:8080",
      "/p": "http://localhost:8080",
      "/healthz": "http://localhost:8080",
    },
  },
});
