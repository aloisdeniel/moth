/// <reference types="vite/client" />

interface ImportMetaEnv {
  // Set by .env.demo (`--mode demo` builds): swaps the connect transport for
  // the in-browser demo backend. Absent in production builds, so every
  // `import.meta.env.VITE_DEMO` branch is dead code there.
  readonly VITE_DEMO?: string;
}
