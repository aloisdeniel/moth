/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_MOTH_ENDPOINT?: string
  readonly VITE_MOTH_PUBLISHABLE_KEY?: string
  readonly VITE_MOTH_PROJECT_SLUG?: string
  readonly VITE_BACKEND_URL?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
