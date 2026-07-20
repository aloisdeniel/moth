import { useEffect, useRef, useState, type FormEvent } from 'react'
import type { MothOAuthProvider } from '../core/client.js'
import type { MothProjectConfig } from '../core/projectConfig.js'
import { useMothContext, useMothCopy, useMothTheme } from './context.js'
import { friendlyMothErrorMessage } from './friendlyErrors.js'

const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

type Mode = 'signIn' | 'signUp' | 'resetRequest' | 'resetSent'

export interface MothLoginScreenProps {
  /** Headline override; defaults to the localized mode title. */
  title?: string
}

/**
 * Batteries-included sign-in / sign-up / forgot-password flow.
 *
 * `MothProvider` shows it by default while signed out. On mount it fetches
 * the project's public config and adapts: the sign-up toggle only appears
 * when public sign-up is open, password validation uses the project's
 * minimum length, and Google/Apple buttons appear per the enabled providers
 * (via the milestone-04 web-redirect flow — requires
 * `MothConfig.projectSlug` and the app's origin registered in the admin
 * under Providers → "Redirect origins (web)"). The OAuth round-trip
 * returns to the current URL with its fragment stripped (the server
 * refuses redirect URIs containing `#`), so hash-routed apps land back on
 * the fragment-less URL after Google/Apple sign-in and should restore
 * their route themselves. Every visual token
 * comes from the project's theme; every string from the negotiated
 * localized copy with the bundled floor as offline fallback.
 */
export function MothLoginScreen(props: MothLoginScreenProps) {
  const { client, configController } = useMothContext()
  const theme = useMothTheme()
  const copy = useMothCopy()

  const [config, setConfig] = useState<MothProjectConfig | null>(
    configController.projectConfig,
  )
  const [configFailed, setConfigFailed] = useState(false)
  const [mode, setMode] = useState<Mode>('signIn')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [info, setInfo] = useState<string | null>(null)
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [emailError, setEmailError] = useState<string | null>(null)
  const [passwordError, setPasswordError] = useState<string | null>(null)
  const exchangedCode = useRef(false)

  const t = (key: string, vars?: Record<string, string>) =>
    copy.value(key, { app: client.config.appName ?? '', ...vars })

  const fetchConfig = () => {
    setConfigFailed(false)
    configController
      .ensureProjectConfig()
      .then(setConfig)
      .catch(() => setConfigFailed(true))
  }
  useEffect(fetchConfig, []) // eslint-disable-line react-hooks/exhaustive-deps

  // Web-redirect OAuth return: the server redirected back with a one-time
  // ?code= parameter. Exchange it for a session (once) and strip it.
  useEffect(() => {
    if (exchangedCode.current || typeof window === 'undefined') return
    const url = new URL(window.location.href)
    const code = url.searchParams.get('code')
    if (code === null || code === '') return
    exchangedCode.current = true
    url.searchParams.delete('code')
    window.history.replaceState(window.history.state, '', url.toString())
    setBusy(true)
    client
      .exchangeOAuthCode(code)
      .catch((err: unknown) => setError(friendlyMothErrorMessage(err, copy)))
      .finally(() => setBusy(false))
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const validateEmail = (value: string): string | null => {
    const trimmed = value.trim()
    if (trimmed === '') return t('sign_in.email_required')
    if (!emailPattern.test(trimmed)) return t('sign_in.email_invalid')
    return null
  }

  const validatePassword = (value: string): string | null => {
    if (value === '') return t('sign_in.password_required')
    const min = config?.passwordMinLength ?? 0
    if (mode === 'signUp' && min > 0 && value.length < min) {
      return t('sign_up.password_too_short', { count: String(min) })
    }
    return null
  }

  const switchMode = (next: Mode) => {
    setMode(next)
    setError(null)
    setInfo(null)
    setEmailError(null)
    setPasswordError(null)
  }

  const submit = async (event: FormEvent) => {
    event.preventDefault()
    const emailErr = validateEmail(email)
    const passwordErr = validatePassword(password)
    setEmailError(emailErr)
    setPasswordError(passwordErr)
    if (emailErr !== null || passwordErr !== null) return
    setBusy(true)
    setError(null)
    setInfo(null)
    try {
      if (mode === 'signUp') {
        const result = await client.signUp({
          email: email.trim(),
          password,
        })
        if (!result.signedIn) {
          // Verification (or approval) required before the first sign-in.
          setMode('signIn')
          setInfo(t('sign_up.verify_sent'))
          setPassword('')
        }
      } else {
        await client.signIn({ email: email.trim(), password })
        // On success MothProvider swaps this screen out; nothing more to do.
      }
    } catch (err) {
      setError(friendlyMothErrorMessage(err, copy))
    } finally {
      setBusy(false)
    }
  }

  const sendReset = async (event: FormEvent) => {
    event.preventDefault()
    const emailErr = validateEmail(email)
    setEmailError(emailErr)
    if (emailErr !== null) return
    setBusy(true)
    setError(null)
    setInfo(null)
    try {
      await client.requestPasswordReset(email.trim())
      setMode('resetSent')
    } catch (err) {
      setError(friendlyMothErrorMessage(err, copy))
    } finally {
      setBusy(false)
    }
  }

  const signInWithProvider = (provider: MothOAuthProvider) => {
    setError(null)
    setInfo(null)
    try {
      // Round-trip back to this page: the server appends ?code=... to the
      // redirect URI, consumed by the effect above. signInWithRedirect
      // strips the URL fragment first — the server refuses redirect URIs
      // containing '#', so hash-routed apps come back to the fragment-less
      // URL and restore their route themselves.
      client.signInWithRedirect(provider)
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    }
  }

  const title =
    props.title ??
    (mode === 'signIn'
      ? t('sign_in.title')
      : mode === 'signUp'
        ? t('sign_up.title')
        : t('password_reset.title'))
  const subtitle =
    mode === 'signIn'
      ? t('sign_in.subtitle')
      : mode === 'signUp'
        ? t('sign_up.subtitle')
        : mode === 'resetRequest'
          ? t('password_reset.subtitle')
          : null

  let content
  if (configFailed) {
    content = (
      <div className="moth-error-state" data-moth="config-error">
        <h2 className="moth-title">{t('sign_in.config_error_title')}</h2>
        <p className="moth-subtitle">{t('sign_in.config_error_body')}</p>
        <button
          type="button"
          className="moth-btn"
          data-moth="retry-config"
          onClick={fetchConfig}
        >
          {t('sign_in.retry')}
        </button>
      </div>
    )
  } else if (config === null) {
    content = <div className="moth-spinner" role="progressbar" aria-label="Loading" />
  } else {
    const hasLogo =
      theme.logoLightUrl !== undefined || theme.logoDarkUrl !== undefined
    content = (
      <>
        {hasLogo && <MothLogo light={theme.logoLightUrl} dark={theme.logoDarkUrl} />}
        <h1 className="moth-title">{title}</h1>
        {subtitle !== null && <p className="moth-subtitle">{subtitle}</p>}
        {error !== null && (
          <div className="moth-banner moth-banner--error" role="alert" data-moth="error">
            {error}
          </div>
        )}
        {info !== null && (
          <div className="moth-banner moth-banner--info" role="status" data-moth="info">
            {info}
          </div>
        )}
        {(mode === 'signIn' || mode === 'signUp') && (
          <EmailPasswordForm
            mode={mode}
            busy={busy}
            config={config}
            email={email}
            password={password}
            emailError={emailError}
            passwordError={passwordError}
            onEmail={setEmail}
            onPassword={setPassword}
            onSubmit={submit}
            onForgot={() => switchMode('resetRequest')}
            onToggle={() => switchMode(mode === 'signUp' ? 'signIn' : 'signUp')}
            onProvider={signInWithProvider}
            t={t}
          />
        )}
        {mode === 'resetRequest' && (
          <form className="moth-content" onSubmit={sendReset} noValidate>
            <Field
              id="moth-reset-email"
              label={t('password_reset.email_label')}
              type="email"
              value={email}
              error={emailError}
              disabled={busy}
              autoComplete="email"
              onChange={setEmail}
            />
            <button
              type="submit"
              className="moth-btn"
              disabled={busy}
              data-moth="send-reset"
            >
              {t('password_reset.submit')}
            </button>
            <button
              type="button"
              className="moth-btn-text"
              disabled={busy}
              data-moth="back-to-sign-in"
              onClick={() => switchMode('signIn')}
            >
              {t('password_reset.back_to_sign_in')}
            </button>
          </form>
        )}
        {mode === 'resetSent' && (
          <div className="moth-center moth-content" data-moth="reset-sent">
            <h2 className="moth-title">{t('password_reset.sent_title')}</h2>
            <p className="moth-subtitle">
              {copy.value('password_reset.sent', { email: email.trim() })}
            </p>
            <button
              type="button"
              className="moth-btn-text"
              data-moth="back-to-sign-in"
              onClick={() => switchMode('signIn')}
            >
              {t('password_reset.back_to_sign_in')}
            </button>
          </div>
        )}
        <LegalFooter
          termsUrl={theme.termsUrl}
          privacyUrl={theme.privacyUrl}
          termsLabel={t('sign_in.terms_link')}
          privacyLabel={t('sign_in.privacy_link')}
        />
      </>
    )
  }

  return (
    <div className="moth-screen">
      <div className="moth-content">{content}</div>
    </div>
  )
}

function EmailPasswordForm(props: {
  mode: 'signIn' | 'signUp'
  busy: boolean
  config: MothProjectConfig
  email: string
  password: string
  emailError: string | null
  passwordError: string | null
  onEmail: (v: string) => void
  onPassword: (v: string) => void
  onSubmit: (event: FormEvent) => void
  onForgot: () => void
  onToggle: () => void
  onProvider: (provider: MothOAuthProvider) => void
  t: (key: string, vars?: Record<string, string>) => string
}) {
  const { client } = useMothContext()
  const { mode, busy, config, t } = props
  const signUp = mode === 'signUp'
  const prefix = signUp ? 'sign_up' : 'sign_in'
  // The web-redirect flow needs the project slug in the start URL; without
  // it the buttons would only dead-end, so they are hidden.
  const showProviders =
    (config.google.enabled || config.apple.enabled) &&
    Boolean(client.config.projectSlug)
  return (
    <form className="moth-content" onSubmit={props.onSubmit} noValidate>
      <Field
        id="moth-email"
        label={t(`${prefix}.email_label`)}
        type="email"
        value={props.email}
        error={props.emailError}
        disabled={busy}
        autoComplete="email"
        onChange={props.onEmail}
      />
      <Field
        id="moth-password"
        label={t(`${prefix}.password_label`)}
        type="password"
        value={props.password}
        error={props.passwordError}
        disabled={busy}
        autoComplete={signUp ? 'new-password' : 'current-password'}
        onChange={props.onPassword}
      />
      {signUp && <p className="moth-subtitle">{t('sign_up.legal')}</p>}
      <button type="submit" className="moth-btn" disabled={busy} data-moth="submit">
        {t(`${prefix}.submit`)}
      </button>
      {!signUp && (
        <div className="moth-row moth-row--end">
          <button
            type="button"
            className="moth-btn-text"
            disabled={busy}
            data-moth="forgot-password"
            onClick={props.onForgot}
          >
            {t('sign_in.forgot_password')}
          </button>
        </div>
      )}
      {config.signUpOpen && (
        <div className="moth-row">
          <span>{signUp ? t('sign_up.have_account') : t('sign_in.no_account')}</span>
          <button
            type="button"
            className="moth-btn-text"
            disabled={busy}
            data-moth="toggle-mode"
            onClick={props.onToggle}
          >
            {signUp ? t('sign_up.switch_to_sign_in') : t('sign_in.switch_to_sign_up')}
          </button>
        </div>
      )}
      {showProviders && (
        <>
          <div className="moth-divider">{t('sign_in.divider_or')}</div>
          {config.google.enabled && (
            <button
              type="button"
              className="moth-btn moth-btn-outline"
              disabled={busy}
              data-moth="google"
              onClick={() => props.onProvider('google')}
            >
              {t('sign_in.continue_with_google')}
            </button>
          )}
          {config.apple.enabled && (
            <button
              type="button"
              className="moth-btn moth-btn-outline"
              disabled={busy}
              data-moth="apple"
              onClick={() => props.onProvider('apple')}
            >
              {t('sign_in.continue_with_apple')}
            </button>
          )}
        </>
      )}
    </form>
  )
}

function Field(props: {
  id: string
  label: string
  type: string
  value: string
  error: string | null
  disabled: boolean
  autoComplete: string
  onChange: (value: string) => void
}) {
  return (
    <div className="moth-field">
      <label className="moth-label" htmlFor={props.id}>
        {props.label}
      </label>
      <input
        id={props.id}
        className="moth-input"
        type={props.type}
        value={props.value}
        disabled={props.disabled}
        autoComplete={props.autoComplete}
        aria-invalid={props.error !== null}
        onChange={(e) => props.onChange(e.target.value)}
      />
      {props.error !== null && (
        <span className="moth-field-error" role="alert">
          {props.error}
        </span>
      )}
    </div>
  )
}

function MothLogo(props: { light?: string; dark?: string }) {
  const light = props.light ?? props.dark
  const dark = props.dark ?? props.light
  return (
    <>
      {light !== undefined && (
        <img className="moth-logo moth-logo--light" src={light} alt="" />
      )}
      {dark !== undefined && (
        <img className="moth-logo moth-logo--dark" src={dark} alt="" />
      )}
    </>
  )
}

function LegalFooter(props: {
  termsUrl?: string
  privacyUrl?: string
  termsLabel: string
  privacyLabel: string
}) {
  const links: { label: string; url: string }[] = []
  if (props.termsUrl !== undefined) {
    links.push({ label: props.termsLabel, url: props.termsUrl })
  }
  if (props.privacyUrl !== undefined) {
    links.push({ label: props.privacyLabel, url: props.privacyUrl })
  }
  if (links.length === 0) return null
  return (
    <div className="moth-footer">
      <div className="moth-row">
        {links.map((link, index) => (
          <span key={link.url} className="moth-row">
            {index > 0 && <span>·</span>}
            <a
              className="moth-link-muted"
              href={link.url}
              target="_blank"
              rel="noreferrer"
            >
              {link.label}
            </a>
          </span>
        ))}
      </div>
    </div>
  )
}
