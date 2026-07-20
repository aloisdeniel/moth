import { MothProvider } from '@moth/react'
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { App } from './App.js'

// Values from the project's setup-instructions tab in the moth admin:
//
//   VITE_MOTH_ENDPOINT=http://localhost:8080
//   VITE_MOTH_PUBLISHABLE_KEY=pk_...
//   VITE_MOTH_PROJECT_SLUG=my-app        (only for web OAuth buttons)
//   VITE_BACKEND_URL=http://localhost:8081  (scripts/example_backend)
const endpoint = import.meta.env.VITE_MOTH_ENDPOINT ?? 'http://localhost:8080'
const publishableKey = import.meta.env.VITE_MOTH_PUBLISHABLE_KEY ?? ''

// The app owns its service worker (public/sw.js — display and click
// handling); the SDK subscribes its PushManager when the user opts in via
// useMothPush().subscribe(). Registering it once at startup also lets the
// launch sync re-register an existing subscription.
if ('serviceWorker' in navigator) {
  void navigator.serviceWorker.register('/sw.js')
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <MothProvider
      config={{
        endpoint,
        publishableKey,
        appName: 'Moth Example',
        projectSlug: import.meta.env.VITE_MOTH_PROJECT_SLUG,
      }}
    >
      <App />
    </MothProvider>
  </StrictMode>,
)
