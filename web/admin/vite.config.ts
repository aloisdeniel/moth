import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

// The SPA is served by the moth binary at /admin (embedded via go:embed).
// `vite build` writes straight into the Go package that embeds it.
//
// `vite build --mode demo` builds the website's in-browser demo instead
// (.env.demo sets VITE_DEMO=1, which swaps the transport for the local fake
// backend): output goes into website/public/demo with a relative base so it
// works from whatever path the website is mounted at (see pages.yml).
export default defineConfig(({ mode }) => {
  const demo = mode === "demo";
  return {
    plugins: [react()],
    base: demo ? "./" : "/admin/",
    build: {
      outDir: demo ? "../../website/public/demo" : "../../internal/server/web/dist",
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
        "/oauth": "http://localhost:8080",
        "/pub": "http://localhost:8080",
        "/npm": "http://localhost:8080",
        "/billing": "http://localhost:8080",
        "/assets": "http://localhost:8080",
        "/p": "http://localhost:8080",
        "/healthz": "http://localhost:8080",
      },
    },
  };
});
