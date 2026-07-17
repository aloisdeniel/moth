import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { useEffect, useRef, useState } from 'react';
import { useMothContext, useMothCopy, useMothTheme } from './context.js';
import { friendlyMothErrorMessage } from './friendlyErrors.js';
const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
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
export function MothLoginScreen(props) {
    const { client, configController } = useMothContext();
    const theme = useMothTheme();
    const copy = useMothCopy();
    const [config, setConfig] = useState(configController.projectConfig);
    const [configFailed, setConfigFailed] = useState(false);
    const [mode, setMode] = useState('signIn');
    const [busy, setBusy] = useState(false);
    const [error, setError] = useState(null);
    const [info, setInfo] = useState(null);
    const [email, setEmail] = useState('');
    const [password, setPassword] = useState('');
    const [emailError, setEmailError] = useState(null);
    const [passwordError, setPasswordError] = useState(null);
    const exchangedCode = useRef(false);
    const t = (key, vars) => copy.value(key, { app: client.config.appName ?? '', ...vars });
    const fetchConfig = () => {
        setConfigFailed(false);
        configController
            .ensureProjectConfig()
            .then(setConfig)
            .catch(() => setConfigFailed(true));
    };
    useEffect(fetchConfig, []); // eslint-disable-line react-hooks/exhaustive-deps
    // Web-redirect OAuth return: the server redirected back with a one-time
    // ?code= parameter. Exchange it for a session (once) and strip it.
    useEffect(() => {
        if (exchangedCode.current || typeof window === 'undefined')
            return;
        const url = new URL(window.location.href);
        const code = url.searchParams.get('code');
        if (code === null || code === '')
            return;
        exchangedCode.current = true;
        url.searchParams.delete('code');
        window.history.replaceState(window.history.state, '', url.toString());
        setBusy(true);
        client
            .exchangeOAuthCode(code)
            .catch((err) => setError(friendlyMothErrorMessage(err, copy)))
            .finally(() => setBusy(false));
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);
    const validateEmail = (value) => {
        const trimmed = value.trim();
        if (trimmed === '')
            return t('sign_in.email_required');
        if (!emailPattern.test(trimmed))
            return t('sign_in.email_invalid');
        return null;
    };
    const validatePassword = (value) => {
        if (value === '')
            return t('sign_in.password_required');
        const min = config?.passwordMinLength ?? 0;
        if (mode === 'signUp' && min > 0 && value.length < min) {
            return t('sign_up.password_too_short', { count: String(min) });
        }
        return null;
    };
    const switchMode = (next) => {
        setMode(next);
        setError(null);
        setInfo(null);
        setEmailError(null);
        setPasswordError(null);
    };
    const submit = async (event) => {
        event.preventDefault();
        const emailErr = validateEmail(email);
        const passwordErr = validatePassword(password);
        setEmailError(emailErr);
        setPasswordError(passwordErr);
        if (emailErr !== null || passwordErr !== null)
            return;
        setBusy(true);
        setError(null);
        setInfo(null);
        try {
            if (mode === 'signUp') {
                const result = await client.signUp({
                    email: email.trim(),
                    password,
                });
                if (!result.signedIn) {
                    // Verification (or approval) required before the first sign-in.
                    setMode('signIn');
                    setInfo(t('sign_up.verify_sent'));
                    setPassword('');
                }
            }
            else {
                await client.signIn({ email: email.trim(), password });
                // On success MothProvider swaps this screen out; nothing more to do.
            }
        }
        catch (err) {
            setError(friendlyMothErrorMessage(err, copy));
        }
        finally {
            setBusy(false);
        }
    };
    const sendReset = async (event) => {
        event.preventDefault();
        const emailErr = validateEmail(email);
        setEmailError(emailErr);
        if (emailErr !== null)
            return;
        setBusy(true);
        setError(null);
        setInfo(null);
        try {
            await client.requestPasswordReset(email.trim());
            setMode('resetSent');
        }
        catch (err) {
            setError(friendlyMothErrorMessage(err, copy));
        }
        finally {
            setBusy(false);
        }
    };
    const signInWithProvider = (provider) => {
        setError(null);
        setInfo(null);
        try {
            // Round-trip back to this page: the server appends ?code=... to the
            // redirect URI, consumed by the effect above. signInWithRedirect
            // strips the URL fragment first — the server refuses redirect URIs
            // containing '#', so hash-routed apps come back to the fragment-less
            // URL and restore their route themselves.
            client.signInWithRedirect(provider);
        }
        catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        }
    };
    const title = props.title ??
        (mode === 'signIn'
            ? t('sign_in.title')
            : mode === 'signUp'
                ? t('sign_up.title')
                : t('password_reset.title'));
    const subtitle = mode === 'signIn'
        ? t('sign_in.subtitle')
        : mode === 'signUp'
            ? t('sign_up.subtitle')
            : mode === 'resetRequest'
                ? t('password_reset.subtitle')
                : null;
    let content;
    if (configFailed) {
        content = (_jsxs("div", { className: "moth-error-state", "data-moth": "config-error", children: [_jsx("h2", { className: "moth-title", children: t('sign_in.config_error_title') }), _jsx("p", { className: "moth-subtitle", children: t('sign_in.config_error_body') }), _jsx("button", { type: "button", className: "moth-btn", "data-moth": "retry-config", onClick: fetchConfig, children: t('sign_in.retry') })] }));
    }
    else if (config === null) {
        content = _jsx("div", { className: "moth-spinner", role: "progressbar", "aria-label": "Loading" });
    }
    else {
        const hasLogo = theme.logoLightUrl !== undefined || theme.logoDarkUrl !== undefined;
        content = (_jsxs(_Fragment, { children: [hasLogo && _jsx(MothLogo, { light: theme.logoLightUrl, dark: theme.logoDarkUrl }), _jsx("h1", { className: "moth-title", children: title }), subtitle !== null && _jsx("p", { className: "moth-subtitle", children: subtitle }), error !== null && (_jsx("div", { className: "moth-banner moth-banner--error", role: "alert", "data-moth": "error", children: error })), info !== null && (_jsx("div", { className: "moth-banner moth-banner--info", role: "status", "data-moth": "info", children: info })), (mode === 'signIn' || mode === 'signUp') && (_jsx(EmailPasswordForm, { mode: mode, busy: busy, config: config, email: email, password: password, emailError: emailError, passwordError: passwordError, onEmail: setEmail, onPassword: setPassword, onSubmit: submit, onForgot: () => switchMode('resetRequest'), onToggle: () => switchMode(mode === 'signUp' ? 'signIn' : 'signUp'), onProvider: signInWithProvider, t: t })), mode === 'resetRequest' && (_jsxs("form", { className: "moth-content", onSubmit: sendReset, noValidate: true, children: [_jsx(Field, { id: "moth-reset-email", label: t('password_reset.email_label'), type: "email", value: email, error: emailError, disabled: busy, autoComplete: "email", onChange: setEmail }), _jsx("button", { type: "submit", className: "moth-btn", disabled: busy, "data-moth": "send-reset", children: t('password_reset.submit') }), _jsx("button", { type: "button", className: "moth-btn-text", disabled: busy, "data-moth": "back-to-sign-in", onClick: () => switchMode('signIn'), children: t('password_reset.back_to_sign_in') })] })), mode === 'resetSent' && (_jsxs("div", { className: "moth-center moth-content", "data-moth": "reset-sent", children: [_jsx("h2", { className: "moth-title", children: t('password_reset.sent_title') }), _jsx("p", { className: "moth-subtitle", children: copy.value('password_reset.sent', { email: email.trim() }) }), _jsx("button", { type: "button", className: "moth-btn-text", "data-moth": "back-to-sign-in", onClick: () => switchMode('signIn'), children: t('password_reset.back_to_sign_in') })] })), _jsx(LegalFooter, { termsUrl: theme.termsUrl, privacyUrl: theme.privacyUrl, termsLabel: t('sign_in.terms_link'), privacyLabel: t('sign_in.privacy_link') })] }));
    }
    return (_jsx("div", { className: "moth-screen", children: _jsx("div", { className: "moth-content", children: content }) }));
}
function EmailPasswordForm(props) {
    const { client } = useMothContext();
    const { mode, busy, config, t } = props;
    const signUp = mode === 'signUp';
    const prefix = signUp ? 'sign_up' : 'sign_in';
    // The web-redirect flow needs the project slug in the start URL; without
    // it the buttons would only dead-end, so they are hidden.
    const showProviders = (config.google.enabled || config.apple.enabled) &&
        Boolean(client.config.projectSlug);
    return (_jsxs("form", { className: "moth-content", onSubmit: props.onSubmit, noValidate: true, children: [_jsx(Field, { id: "moth-email", label: t(`${prefix}.email_label`), type: "email", value: props.email, error: props.emailError, disabled: busy, autoComplete: "email", onChange: props.onEmail }), _jsx(Field, { id: "moth-password", label: t(`${prefix}.password_label`), type: "password", value: props.password, error: props.passwordError, disabled: busy, autoComplete: signUp ? 'new-password' : 'current-password', onChange: props.onPassword }), signUp && _jsx("p", { className: "moth-subtitle", children: t('sign_up.legal') }), _jsx("button", { type: "submit", className: "moth-btn", disabled: busy, "data-moth": "submit", children: t(`${prefix}.submit`) }), !signUp && (_jsx("div", { className: "moth-row moth-row--end", children: _jsx("button", { type: "button", className: "moth-btn-text", disabled: busy, "data-moth": "forgot-password", onClick: props.onForgot, children: t('sign_in.forgot_password') }) })), config.signUpOpen && (_jsxs("div", { className: "moth-row", children: [_jsx("span", { children: signUp ? t('sign_up.have_account') : t('sign_in.no_account') }), _jsx("button", { type: "button", className: "moth-btn-text", disabled: busy, "data-moth": "toggle-mode", onClick: props.onToggle, children: signUp ? t('sign_up.switch_to_sign_in') : t('sign_in.switch_to_sign_up') })] })), showProviders && (_jsxs(_Fragment, { children: [_jsx("div", { className: "moth-divider", children: t('sign_in.divider_or') }), config.google.enabled && (_jsx("button", { type: "button", className: "moth-btn moth-btn-outline", disabled: busy, "data-moth": "google", onClick: () => props.onProvider('google'), children: t('sign_in.continue_with_google') })), config.apple.enabled && (_jsx("button", { type: "button", className: "moth-btn moth-btn-outline", disabled: busy, "data-moth": "apple", onClick: () => props.onProvider('apple'), children: t('sign_in.continue_with_apple') }))] }))] }));
}
function Field(props) {
    return (_jsxs("div", { className: "moth-field", children: [_jsx("label", { className: "moth-label", htmlFor: props.id, children: props.label }), _jsx("input", { id: props.id, className: "moth-input", type: props.type, value: props.value, disabled: props.disabled, autoComplete: props.autoComplete, "aria-invalid": props.error !== null, onChange: (e) => props.onChange(e.target.value) }), props.error !== null && (_jsx("span", { className: "moth-field-error", role: "alert", children: props.error }))] }));
}
function MothLogo(props) {
    const light = props.light ?? props.dark;
    const dark = props.dark ?? props.light;
    return (_jsxs(_Fragment, { children: [light !== undefined && (_jsx("img", { className: "moth-logo moth-logo--light", src: light, alt: "" })), dark !== undefined && (_jsx("img", { className: "moth-logo moth-logo--dark", src: dark, alt: "" }))] }));
}
function LegalFooter(props) {
    const links = [];
    if (props.termsUrl !== undefined) {
        links.push({ label: props.termsLabel, url: props.termsUrl });
    }
    if (props.privacyUrl !== undefined) {
        links.push({ label: props.privacyLabel, url: props.privacyUrl });
    }
    if (links.length === 0)
        return null;
    return (_jsx("div", { className: "moth-footer", children: _jsx("div", { className: "moth-row", children: links.map((link, index) => (_jsxs("span", { className: "moth-row", children: [index > 0 && _jsx("span", { children: "\u00B7" }), _jsx("a", { className: "moth-link-muted", href: link.url, target: "_blank", rel: "noreferrer", children: link.label })] }, link.url))) }) }));
}
