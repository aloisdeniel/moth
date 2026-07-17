// The SDK's bundled fallback copy: the localization FLOOR for the SDK screens
// (sign_in.*, sign_up.*, password_reset.*, paywall.* and the shared error.*
// toasts). It renders localized copy before GetProjectConfig arrives and
// offline. Server-delivered project copy overrides it when present; a locale
// neither side bundles falls back to English per key.
//
// The sign_in/sign_up/password_reset/paywall keys the moth server also ships
// mirror internal/i18n (catalog.go English defaults + locales/*.json). The
// remaining keys (provider buttons, form validators, footer link labels,
// config/store error chrome, purchase toasts and the error.* mapping) are
// SDK-only: the server catalog has no key for them, so they always resolve
// from this floor. Keep this file's shared keys in sync with internal/i18n
// when the catalog changes.
import 'dart:ui';

/// The BCP-47 language codes the SDK bundles fallback copy for, English first.
const List<String> mothBundledLocales = <String>[
  'en',
  'fr',
  'de',
  'es',
  'pt',
  'it',
  'ja',
];

// Message key -> localized string, per bundled language code. English is
// the complete key set; each other locale mirrors it.
const Map<String, Map<String, String>> _bundled = <String, Map<String, String>>{
  'en': <String, String>{
    'sign_in.title': 'Sign in',
    'sign_in.subtitle': 'Welcome back to {app}.',
    'sign_in.email_label': 'Email',
    'sign_in.password_label': 'Password',
    'sign_in.submit': 'Sign in',
    'sign_in.forgot_password': 'Forgot password?',
    'sign_in.no_account': 'Don\'t have an account?',
    'sign_in.switch_to_sign_up': 'Sign up',
    'sign_in.error_invalid': 'Incorrect email or password.',
    'sign_in.continue_with_google': 'Continue with Google',
    'sign_in.continue_with_apple': 'Continue with Apple',
    'sign_in.divider_or': 'or',
    'sign_in.email_required': 'Enter your email address',
    'sign_in.email_invalid': 'Enter a valid email address',
    'sign_in.password_required': 'Enter your password',
    'sign_in.terms_link': 'Terms of Service',
    'sign_in.privacy_link': 'Privacy Policy',
    'sign_in.config_error_title': 'Cannot reach the server',
    'sign_in.config_error_body':
        'Sign-in options could not be loaded. Check your connection and try again.',
    'sign_in.retry': 'Try again',
    'sign_up.title': 'Create account',
    'sign_up.subtitle': 'Create your {app} account.',
    'sign_up.email_label': 'Email',
    'sign_up.password_label': 'Password',
    'sign_up.submit': 'Create account',
    'sign_up.have_account': 'Already have an account?',
    'sign_up.switch_to_sign_in': 'Sign in',
    'sign_up.legal': 'By continuing you agree to our Terms and Privacy Policy.',
    'sign_up.error_email_taken': 'An account with this email already exists.',
    'sign_up.password_too_short': 'Use at least {count} characters',
    'sign_up.verify_sent':
        'Account created — check your inbox to verify your email address, then sign in.',
    'password_reset.title': 'Reset password',
    'password_reset.subtitle':
        'Enter your email and we\'ll send you a reset link.',
    'password_reset.email_label': 'Email',
    'password_reset.submit': 'Send reset link',
    'password_reset.back_to_sign_in': 'Back to sign in',
    'password_reset.sent':
        'If an account exists for {email}, a reset link is on its way.',
    'password_reset.sent_title': 'Check your email',
    'paywall.title': 'Unlock {app} Premium',
    'paywall.subtitle': 'Get unlimited access to every feature.',
    'paywall.cta': 'Continue',
    'paywall.restore': 'Restore purchases',
    'paywall.terms': 'Payment is charged to your account. Cancel anytime.',
    'paywall.most_popular': 'Most popular',
    'paywall.trial_3_day': '3-day free trial',
    'paywall.trial_1_week': '1-week free trial',
    'paywall.trial_2_week': '2-week free trial',
    'paywall.trial_1_month': '1-month free trial',
    'paywall.trial_generic': 'Free trial',
    'paywall.period_week': 'week',
    'paywall.period_month': 'month',
    'paywall.period_quarter': 'quarter',
    'paywall.period_6_month': '6 months',
    'paywall.period_year': 'year',
    'paywall.manage_subscription': 'Manage subscription',
    'paywall.terms_link': 'Terms of Service',
    'paywall.privacy_link': 'Privacy Policy',
    'paywall.empty_title': 'Nothing to purchase yet',
    'paywall.empty_body':
        'There are no subscriptions available right now. Check back later.',
    'paywall.error_title': 'Cannot reach the store',
    'paywall.error_body':
        'Subscription options could not be loaded. Check your connection and try again.',
    'paywall.retry': 'Try again',
    'paywall.purchase_pending':
        'Your purchase is pending approval. It will unlock once confirmed.',
    'paywall.already_owned':
        'You already own this subscription — tap Restore to re-link it.',
    'paywall.restore_none': 'No previous purchases were found to restore.',
    'paywall.restore_done': 'Your purchases were restored.',
    'paywall.restore_failed': 'Could not restore purchases: {error}',
    'error.email_not_verified':
        'Please verify your email address first — check your inbox.',
    'error.signup_closed': 'Sign-up is currently closed for this app.',
    'error.invalid_email': 'That email address does not look right.',
    'error.session_expired': 'Your session has expired — sign in again.',
    'error.user_disabled': 'This account has been disabled.',
    'error.rate_limited': 'Too many attempts — wait a moment and try again.',
    'error.provider_disabled':
        'This sign-in method is not enabled for this app.',
    'error.provider_failed':
        'Sign-in with the provider failed. Please try again.',
    'error.last_login_method':
        'This is your only way to sign in, so it cannot be removed.',
    'error.network':
        'Cannot reach the server. Check your connection and try again.',
    'error.generic': 'Something went wrong. Please try again.',
  },
  'fr': <String, String>{
    'sign_in.title': 'Connexion',
    'sign_in.subtitle': 'Bon retour sur {app}.',
    'sign_in.email_label': 'E-mail',
    'sign_in.password_label': 'Mot de passe',
    'sign_in.submit': 'Se connecter',
    'sign_in.forgot_password': 'Mot de passe oublié ?',
    'sign_in.no_account': 'Vous n\'avez pas de compte ?',
    'sign_in.switch_to_sign_up': 'S\'inscrire',
    'sign_in.error_invalid': 'E-mail ou mot de passe incorrect.',
    'sign_in.continue_with_google': 'Continuer avec Google',
    'sign_in.continue_with_apple': 'Continuer avec Apple',
    'sign_in.divider_or': 'ou',
    'sign_in.email_required': 'Saisissez votre adresse e-mail',
    'sign_in.email_invalid': 'Saisissez une adresse e-mail valide',
    'sign_in.password_required': 'Saisissez votre mot de passe',
    'sign_in.terms_link': 'Conditions d\'utilisation',
    'sign_in.privacy_link': 'Politique de confidentialité',
    'sign_in.config_error_title': 'Impossible de joindre le serveur',
    'sign_in.config_error_body':
        'Les options de connexion n\'ont pas pu être chargées. Vérifiez votre connexion et réessayez.',
    'sign_in.retry': 'Réessayer',
    'sign_up.title': 'Créer un compte',
    'sign_up.subtitle': 'Créez votre compte {app}.',
    'sign_up.email_label': 'E-mail',
    'sign_up.password_label': 'Mot de passe',
    'sign_up.submit': 'Créer un compte',
    'sign_up.have_account': 'Vous avez déjà un compte ?',
    'sign_up.switch_to_sign_in': 'Se connecter',
    'sign_up.legal':
        'En continuant, vous acceptez nos conditions d\'utilisation et notre politique de confidentialité.',
    'sign_up.error_email_taken': 'Un compte avec cet e-mail existe déjà.',
    'sign_up.password_too_short': 'Utilisez au moins {count} caractères',
    'sign_up.verify_sent':
        'Compte créé — consultez votre boîte de réception pour vérifier votre adresse e-mail, puis connectez-vous.',
    'password_reset.title': 'Réinitialiser le mot de passe',
    'password_reset.subtitle':
        'Saisissez votre e-mail et nous vous enverrons un lien de réinitialisation.',
    'password_reset.email_label': 'E-mail',
    'password_reset.submit': 'Envoyer le lien',
    'password_reset.back_to_sign_in': 'Retour à la connexion',
    'password_reset.sent':
        'Si un compte existe pour {email}, un lien de réinitialisation est en route.',
    'password_reset.sent_title': 'Consultez vos e-mails',
    'paywall.title': 'Débloquez {app} Premium',
    'paywall.subtitle': 'Accédez sans limite à toutes les fonctionnalités.',
    'paywall.cta': 'Continuer',
    'paywall.restore': 'Restaurer les achats',
    'paywall.terms':
        'Le paiement est débité de votre compte. Annulez à tout moment.',
    'paywall.most_popular': 'Le plus populaire',
    'paywall.trial_3_day': 'Essai gratuit de 3 jours',
    'paywall.trial_1_week': 'Essai gratuit d\'une semaine',
    'paywall.trial_2_week': 'Essai gratuit de 2 semaines',
    'paywall.trial_1_month': 'Essai gratuit d\'un mois',
    'paywall.trial_generic': 'Essai gratuit',
    'paywall.period_week': 'semaine',
    'paywall.period_month': 'mois',
    'paywall.period_quarter': 'trimestre',
    'paywall.period_6_month': '6 mois',
    'paywall.period_year': 'an',
    'paywall.manage_subscription': 'Gérer l\'abonnement',
    'paywall.terms_link': 'Conditions d\'utilisation',
    'paywall.privacy_link': 'Politique de confidentialité',
    'paywall.empty_title': 'Rien à acheter pour l\'instant',
    'paywall.empty_body':
        'Aucun abonnement n\'est disponible pour le moment. Revenez plus tard.',
    'paywall.error_title': 'Impossible de joindre la boutique',
    'paywall.error_body':
        'Les options d\'abonnement n\'ont pas pu être chargées. Vérifiez votre connexion et réessayez.',
    'paywall.retry': 'Réessayer',
    'paywall.purchase_pending':
        'Votre achat est en attente d\'approbation. Il sera débloqué une fois confirmé.',
    'paywall.already_owned':
        'Vous possédez déjà cet abonnement — appuyez sur Restaurer pour le réassocier.',
    'paywall.restore_none': 'Aucun achat précédent à restaurer.',
    'paywall.restore_done': 'Vos achats ont été restaurés.',
    'paywall.restore_failed': 'Impossible de restaurer les achats : {error}',
    'error.email_not_verified':
        'Veuillez d\'abord vérifier votre adresse e-mail — consultez votre boîte de réception.',
    'error.signup_closed':
        'Les inscriptions sont actuellement fermées pour cette application.',
    'error.invalid_email': 'Cette adresse e-mail ne semble pas valide.',
    'error.session_expired': 'Votre session a expiré — reconnectez-vous.',
    'error.user_disabled': 'Ce compte a été désactivé.',
    'error.rate_limited':
        'Trop de tentatives — patientez un instant et réessayez.',
    'error.provider_disabled':
        'Cette méthode de connexion n\'est pas activée pour cette application.',
    'error.provider_failed':
        'La connexion avec le fournisseur a échoué. Veuillez réessayer.',
    'error.last_login_method':
        'C\'est votre seul moyen de connexion, il ne peut donc pas être supprimé.',
    'error.network':
        'Impossible de joindre le serveur. Vérifiez votre connexion et réessayez.',
    'error.generic': 'Une erreur s\'est produite. Veuillez réessayer.',
  },
  'de': <String, String>{
    'sign_in.title': 'Anmelden',
    'sign_in.subtitle': 'Willkommen zurück bei {app}.',
    'sign_in.email_label': 'E-Mail',
    'sign_in.password_label': 'Passwort',
    'sign_in.submit': 'Anmelden',
    'sign_in.forgot_password': 'Passwort vergessen?',
    'sign_in.no_account': 'Noch kein Konto?',
    'sign_in.switch_to_sign_up': 'Registrieren',
    'sign_in.error_invalid': 'E-Mail oder Passwort ist falsch.',
    'sign_in.continue_with_google': 'Mit Google fortfahren',
    'sign_in.continue_with_apple': 'Mit Apple fortfahren',
    'sign_in.divider_or': 'oder',
    'sign_in.email_required': 'Gib deine E-Mail-Adresse ein',
    'sign_in.email_invalid': 'Gib eine gültige E-Mail-Adresse ein',
    'sign_in.password_required': 'Gib dein Passwort ein',
    'sign_in.terms_link': 'Nutzungsbedingungen',
    'sign_in.privacy_link': 'Datenschutzerklärung',
    'sign_in.config_error_title': 'Server nicht erreichbar',
    'sign_in.config_error_body':
        'Die Anmeldeoptionen konnten nicht geladen werden. Überprüfe deine Verbindung und versuche es erneut.',
    'sign_in.retry': 'Erneut versuchen',
    'sign_up.title': 'Konto erstellen',
    'sign_up.subtitle': 'Erstelle dein {app}-Konto.',
    'sign_up.email_label': 'E-Mail',
    'sign_up.password_label': 'Passwort',
    'sign_up.submit': 'Konto erstellen',
    'sign_up.have_account': 'Du hast bereits ein Konto?',
    'sign_up.switch_to_sign_in': 'Anmelden',
    'sign_up.legal':
        'Wenn du fortfährst, stimmst du unseren Nutzungsbedingungen und der Datenschutzerklärung zu.',
    'sign_up.error_email_taken':
        'Ein Konto mit dieser E-Mail existiert bereits.',
    'sign_up.password_too_short': 'Verwende mindestens {count} Zeichen',
    'sign_up.verify_sent':
        'Konto erstellt — überprüfe dein Postfach, um deine E-Mail-Adresse zu bestätigen, und melde dich dann an.',
    'password_reset.title': 'Passwort zurücksetzen',
    'password_reset.subtitle':
        'Gib deine E-Mail-Adresse ein und wir senden dir einen Link zum Zurücksetzen.',
    'password_reset.email_label': 'E-Mail',
    'password_reset.submit': 'Link senden',
    'password_reset.back_to_sign_in': 'Zurück zur Anmeldung',
    'password_reset.sent':
        'Wenn ein Konto für {email} existiert, ist ein Link zum Zurücksetzen unterwegs.',
    'password_reset.sent_title': 'Überprüfe deine E-Mails',
    'paywall.title': 'Schalte {app} Premium frei',
    'paywall.subtitle': 'Erhalte unbegrenzten Zugriff auf alle Funktionen.',
    'paywall.cta': 'Weiter',
    'paywall.restore': 'Käufe wiederherstellen',
    'paywall.terms':
        'Die Zahlung wird deinem Konto belastet. Jederzeit kündbar.',
    'paywall.most_popular': 'Am beliebtesten',
    'paywall.trial_3_day': '3 Tage kostenlos testen',
    'paywall.trial_1_week': '1 Woche kostenlos testen',
    'paywall.trial_2_week': '2 Wochen kostenlos testen',
    'paywall.trial_1_month': '1 Monat kostenlos testen',
    'paywall.trial_generic': 'Kostenlos testen',
    'paywall.period_week': 'Woche',
    'paywall.period_month': 'Monat',
    'paywall.period_quarter': 'Quartal',
    'paywall.period_6_month': '6 Monate',
    'paywall.period_year': 'Jahr',
    'paywall.manage_subscription': 'Abo verwalten',
    'paywall.terms_link': 'Nutzungsbedingungen',
    'paywall.privacy_link': 'Datenschutzerklärung',
    'paywall.empty_title': 'Noch nichts zu kaufen',
    'paywall.empty_body':
        'Derzeit sind keine Abos verfügbar. Schau später noch einmal vorbei.',
    'paywall.error_title': 'Store nicht erreichbar',
    'paywall.error_body':
        'Die Abo-Optionen konnten nicht geladen werden. Überprüfe deine Verbindung und versuche es erneut.',
    'paywall.retry': 'Erneut versuchen',
    'paywall.purchase_pending':
        'Dein Kauf wartet auf Bestätigung. Er wird freigeschaltet, sobald er bestätigt ist.',
    'paywall.already_owned':
        'Du besitzt dieses Abo bereits — tippe auf Wiederherstellen, um es erneut zu verknüpfen.',
    'paywall.restore_none':
        'Es wurden keine früheren Käufe zum Wiederherstellen gefunden.',
    'paywall.restore_done': 'Deine Käufe wurden wiederhergestellt.',
    'paywall.restore_failed':
        'Käufe konnten nicht wiederhergestellt werden: {error}',
    'error.email_not_verified':
        'Bitte bestätige zuerst deine E-Mail-Adresse — überprüfe dein Postfach.',
    'error.signup_closed':
        'Die Registrierung ist für diese App derzeit geschlossen.',
    'error.invalid_email': 'Diese E-Mail-Adresse sieht nicht richtig aus.',
    'error.session_expired':
        'Deine Sitzung ist abgelaufen — melde dich erneut an.',
    'error.user_disabled': 'Dieses Konto wurde deaktiviert.',
    'error.rate_limited':
        'Zu viele Versuche — warte einen Moment und versuche es erneut.',
    'error.provider_disabled':
        'Diese Anmeldemethode ist für diese App nicht aktiviert.',
    'error.provider_failed':
        'Die Anmeldung beim Anbieter ist fehlgeschlagen. Bitte versuche es erneut.',
    'error.last_login_method':
        'Dies ist deine einzige Anmeldemethode und kann daher nicht entfernt werden.',
    'error.network':
        'Server nicht erreichbar. Überprüfe deine Verbindung und versuche es erneut.',
    'error.generic': 'Etwas ist schiefgelaufen. Bitte versuche es erneut.',
  },
  'es': <String, String>{
    'sign_in.title': 'Iniciar sesión',
    'sign_in.subtitle': 'Bienvenido de nuevo a {app}.',
    'sign_in.email_label': 'Correo electrónico',
    'sign_in.password_label': 'Contraseña',
    'sign_in.submit': 'Iniciar sesión',
    'sign_in.forgot_password': '¿Olvidaste tu contraseña?',
    'sign_in.no_account': '¿No tienes una cuenta?',
    'sign_in.switch_to_sign_up': 'Regístrate',
    'sign_in.error_invalid': 'Correo o contraseña incorrectos.',
    'sign_in.continue_with_google': 'Continuar con Google',
    'sign_in.continue_with_apple': 'Continuar con Apple',
    'sign_in.divider_or': 'o',
    'sign_in.email_required': 'Introduce tu correo electrónico',
    'sign_in.email_invalid': 'Introduce un correo electrónico válido',
    'sign_in.password_required': 'Introduce tu contraseña',
    'sign_in.terms_link': 'Términos del servicio',
    'sign_in.privacy_link': 'Política de privacidad',
    'sign_in.config_error_title': 'No se puede conectar con el servidor',
    'sign_in.config_error_body':
        'No se pudieron cargar las opciones de inicio de sesión. Comprueba tu conexión e inténtalo de nuevo.',
    'sign_in.retry': 'Reintentar',
    'sign_up.title': 'Crear cuenta',
    'sign_up.subtitle': 'Crea tu cuenta de {app}.',
    'sign_up.email_label': 'Correo electrónico',
    'sign_up.password_label': 'Contraseña',
    'sign_up.submit': 'Crear cuenta',
    'sign_up.have_account': '¿Ya tienes una cuenta?',
    'sign_up.switch_to_sign_in': 'Iniciar sesión',
    'sign_up.legal':
        'Al continuar, aceptas nuestros términos y nuestra política de privacidad.',
    'sign_up.error_email_taken': 'Ya existe una cuenta con este correo.',
    'sign_up.password_too_short': 'Usa al menos {count} caracteres',
    'sign_up.verify_sent':
        'Cuenta creada: revisa tu bandeja de entrada para verificar tu correo electrónico y luego inicia sesión.',
    'password_reset.title': 'Restablecer contraseña',
    'password_reset.subtitle':
        'Introduce tu correo y te enviaremos un enlace para restablecerla.',
    'password_reset.email_label': 'Correo electrónico',
    'password_reset.submit': 'Enviar enlace',
    'password_reset.back_to_sign_in': 'Volver al inicio de sesión',
    'password_reset.sent':
        'Si existe una cuenta para {email}, un enlace de restablecimiento está en camino.',
    'password_reset.sent_title': 'Revisa tu correo',
    'paywall.title': 'Desbloquea {app} Premium',
    'paywall.subtitle': 'Obtén acceso ilimitado a todas las funciones.',
    'paywall.cta': 'Continuar',
    'paywall.restore': 'Restaurar compras',
    'paywall.terms': 'El pago se cargará a tu cuenta. Cancela cuando quieras.',
    'paywall.most_popular': 'Más popular',
    'paywall.trial_3_day': 'Prueba gratis de 3 días',
    'paywall.trial_1_week': 'Prueba gratis de 1 semana',
    'paywall.trial_2_week': 'Prueba gratis de 2 semanas',
    'paywall.trial_1_month': 'Prueba gratis de 1 mes',
    'paywall.trial_generic': 'Prueba gratis',
    'paywall.period_week': 'semana',
    'paywall.period_month': 'mes',
    'paywall.period_quarter': 'trimestre',
    'paywall.period_6_month': '6 meses',
    'paywall.period_year': 'año',
    'paywall.manage_subscription': 'Gestionar suscripción',
    'paywall.terms_link': 'Términos del servicio',
    'paywall.privacy_link': 'Política de privacidad',
    'paywall.empty_title': 'Nada que comprar todavía',
    'paywall.empty_body':
        'No hay suscripciones disponibles en este momento. Vuelve más tarde.',
    'paywall.error_title': 'No se puede conectar con la tienda',
    'paywall.error_body':
        'No se pudieron cargar las opciones de suscripción. Comprueba tu conexión e inténtalo de nuevo.',
    'paywall.retry': 'Reintentar',
    'paywall.purchase_pending':
        'Tu compra está pendiente de aprobación. Se desbloqueará una vez confirmada.',
    'paywall.already_owned':
        'Ya tienes esta suscripción: toca Restaurar para volver a vincularla.',
    'paywall.restore_none':
        'No se encontraron compras anteriores para restaurar.',
    'paywall.restore_done': 'Tus compras se han restaurado.',
    'paywall.restore_failed': 'No se pudieron restaurar las compras: {error}',
    'error.email_not_verified':
        'Verifica primero tu correo electrónico: revisa tu bandeja de entrada.',
    'error.signup_closed':
        'El registro está cerrado actualmente para esta aplicación.',
    'error.invalid_email': 'Esa dirección de correo no parece correcta.',
    'error.session_expired': 'Tu sesión ha expirado: inicia sesión de nuevo.',
    'error.user_disabled': 'Esta cuenta ha sido deshabilitada.',
    'error.rate_limited':
        'Demasiados intentos: espera un momento e inténtalo de nuevo.',
    'error.provider_disabled':
        'Este método de inicio de sesión no está habilitado para esta aplicación.',
    'error.provider_failed':
        'El inicio de sesión con el proveedor falló. Inténtalo de nuevo.',
    'error.last_login_method':
        'Es tu única forma de iniciar sesión, así que no se puede eliminar.',
    'error.network':
        'No se puede conectar con el servidor. Comprueba tu conexión e inténtalo de nuevo.',
    'error.generic': 'Algo salió mal. Inténtalo de nuevo.',
  },
  'pt': <String, String>{
    'sign_in.title': 'Entrar',
    'sign_in.subtitle': 'Bem-vindo de volta ao {app}.',
    'sign_in.email_label': 'E-mail',
    'sign_in.password_label': 'Senha',
    'sign_in.submit': 'Entrar',
    'sign_in.forgot_password': 'Esqueceu a senha?',
    'sign_in.no_account': 'Não tem uma conta?',
    'sign_in.switch_to_sign_up': 'Cadastre-se',
    'sign_in.error_invalid': 'E-mail ou senha incorretos.',
    'sign_in.continue_with_google': 'Continuar com o Google',
    'sign_in.continue_with_apple': 'Continuar com a Apple',
    'sign_in.divider_or': 'ou',
    'sign_in.email_required': 'Digite seu e-mail',
    'sign_in.email_invalid': 'Digite um e-mail válido',
    'sign_in.password_required': 'Digite sua senha',
    'sign_in.terms_link': 'Termos de Serviço',
    'sign_in.privacy_link': 'Política de Privacidade',
    'sign_in.config_error_title': 'Não foi possível conectar ao servidor',
    'sign_in.config_error_body':
        'Não foi possível carregar as opções de login. Verifique sua conexão e tente novamente.',
    'sign_in.retry': 'Tentar novamente',
    'sign_up.title': 'Criar conta',
    'sign_up.subtitle': 'Crie a sua conta {app}.',
    'sign_up.email_label': 'E-mail',
    'sign_up.password_label': 'Senha',
    'sign_up.submit': 'Criar conta',
    'sign_up.have_account': 'Já tem uma conta?',
    'sign_up.switch_to_sign_in': 'Entrar',
    'sign_up.legal':
        'Ao continuar, você concorda com os nossos termos e a política de privacidade.',
    'sign_up.error_email_taken': 'Já existe uma conta com este e-mail.',
    'sign_up.password_too_short': 'Use pelo menos {count} caracteres',
    'sign_up.verify_sent':
        'Conta criada — verifique sua caixa de entrada para confirmar seu e-mail e depois entre.',
    'password_reset.title': 'Redefinir senha',
    'password_reset.subtitle':
        'Digite seu e-mail e enviaremos um link de redefinição.',
    'password_reset.email_label': 'E-mail',
    'password_reset.submit': 'Enviar link',
    'password_reset.back_to_sign_in': 'Voltar ao login',
    'password_reset.sent':
        'Se existir uma conta para {email}, um link de redefinição está a caminho.',
    'password_reset.sent_title': 'Verifique seu e-mail',
    'paywall.title': 'Desbloqueie o {app} Premium',
    'paywall.subtitle': 'Tenha acesso ilimitado a todos os recursos.',
    'paywall.cta': 'Continuar',
    'paywall.restore': 'Restaurar compras',
    'paywall.terms':
        'O pagamento será cobrado na sua conta. Cancele quando quiser.',
    'paywall.most_popular': 'Mais popular',
    'paywall.trial_3_day': 'Teste gratuito de 3 dias',
    'paywall.trial_1_week': 'Teste gratuito de 1 semana',
    'paywall.trial_2_week': 'Teste gratuito de 2 semanas',
    'paywall.trial_1_month': 'Teste gratuito de 1 mês',
    'paywall.trial_generic': 'Teste gratuito',
    'paywall.period_week': 'semana',
    'paywall.period_month': 'mês',
    'paywall.period_quarter': 'trimestre',
    'paywall.period_6_month': '6 meses',
    'paywall.period_year': 'ano',
    'paywall.manage_subscription': 'Gerenciar assinatura',
    'paywall.terms_link': 'Termos de Serviço',
    'paywall.privacy_link': 'Política de Privacidade',
    'paywall.empty_title': 'Nada para comprar ainda',
    'paywall.empty_body':
        'Não há assinaturas disponíveis no momento. Volte mais tarde.',
    'paywall.error_title': 'Não foi possível conectar à loja',
    'paywall.error_body':
        'Não foi possível carregar as opções de assinatura. Verifique sua conexão e tente novamente.',
    'paywall.retry': 'Tentar novamente',
    'paywall.purchase_pending':
        'Sua compra está aguardando aprovação. Ela será desbloqueada assim que confirmada.',
    'paywall.already_owned':
        'Você já tem esta assinatura — toque em Restaurar para revinculá-la.',
    'paywall.restore_none':
        'Nenhuma compra anterior foi encontrada para restaurar.',
    'paywall.restore_done': 'Suas compras foram restauradas.',
    'paywall.restore_failed': 'Não foi possível restaurar as compras: {error}',
    'error.email_not_verified':
        'Primeiro verifique seu e-mail — confira sua caixa de entrada.',
    'error.signup_closed': 'O cadastro está fechado no momento para este app.',
    'error.invalid_email': 'Esse endereço de e-mail não parece certo.',
    'error.session_expired': 'Sua sessão expirou — entre novamente.',
    'error.user_disabled': 'Esta conta foi desativada.',
    'error.rate_limited':
        'Tentativas demais — aguarde um momento e tente novamente.',
    'error.provider_disabled':
        'Este método de login não está ativado para este app.',
    'error.provider_failed': 'O login com o provedor falhou. Tente novamente.',
    'error.last_login_method':
        'Esta é sua única forma de entrar, por isso não pode ser removida.',
    'error.network':
        'Não foi possível conectar ao servidor. Verifique sua conexão e tente novamente.',
    'error.generic': 'Algo deu errado. Tente novamente.',
  },
  'it': <String, String>{
    'sign_in.title': 'Accedi',
    'sign_in.subtitle': 'Bentornato su {app}.',
    'sign_in.email_label': 'Email',
    'sign_in.password_label': 'Password',
    'sign_in.submit': 'Accedi',
    'sign_in.forgot_password': 'Password dimenticata?',
    'sign_in.no_account': 'Non hai un account?',
    'sign_in.switch_to_sign_up': 'Registrati',
    'sign_in.error_invalid': 'Email o password non corretti.',
    'sign_in.continue_with_google': 'Continua con Google',
    'sign_in.continue_with_apple': 'Continua con Apple',
    'sign_in.divider_or': 'oppure',
    'sign_in.email_required': 'Inserisci il tuo indirizzo email',
    'sign_in.email_invalid': 'Inserisci un indirizzo email valido',
    'sign_in.password_required': 'Inserisci la tua password',
    'sign_in.terms_link': 'Termini di servizio',
    'sign_in.privacy_link': 'Informativa sulla privacy',
    'sign_in.config_error_title': 'Impossibile raggiungere il server',
    'sign_in.config_error_body':
        'Impossibile caricare le opzioni di accesso. Controlla la connessione e riprova.',
    'sign_in.retry': 'Riprova',
    'sign_up.title': 'Crea account',
    'sign_up.subtitle': 'Crea il tuo account {app}.',
    'sign_up.email_label': 'Email',
    'sign_up.password_label': 'Password',
    'sign_up.submit': 'Crea account',
    'sign_up.have_account': 'Hai già un account?',
    'sign_up.switch_to_sign_in': 'Accedi',
    'sign_up.legal':
        'Continuando accetti i nostri termini e la nostra informativa sulla privacy.',
    'sign_up.error_email_taken': 'Esiste già un account con questa email.',
    'sign_up.password_too_short': 'Usa almeno {count} caratteri',
    'sign_up.verify_sent':
        'Account creato — controlla la posta in arrivo per verificare il tuo indirizzo email, poi accedi.',
    'password_reset.title': 'Reimposta password',
    'password_reset.subtitle':
        'Inserisci la tua email e ti invieremo un link per reimpostarla.',
    'password_reset.email_label': 'Email',
    'password_reset.submit': 'Invia link',
    'password_reset.back_to_sign_in': 'Torna all\'accesso',
    'password_reset.sent':
        'Se esiste un account per {email}, un link per reimpostare la password è in arrivo.',
    'password_reset.sent_title': 'Controlla la tua email',
    'paywall.title': 'Sblocca {app} Premium',
    'paywall.subtitle': 'Ottieni accesso illimitato a tutte le funzionalità.',
    'paywall.cta': 'Continua',
    'paywall.restore': 'Ripristina acquisti',
    'paywall.terms':
        'Il pagamento verrà addebitato sul tuo account. Annulla quando vuoi.',
    'paywall.most_popular': 'Più popolare',
    'paywall.trial_3_day': 'Prova gratuita di 3 giorni',
    'paywall.trial_1_week': 'Prova gratuita di 1 settimana',
    'paywall.trial_2_week': 'Prova gratuita di 2 settimane',
    'paywall.trial_1_month': 'Prova gratuita di 1 mese',
    'paywall.trial_generic': 'Prova gratuita',
    'paywall.period_week': 'settimana',
    'paywall.period_month': 'mese',
    'paywall.period_quarter': 'trimestre',
    'paywall.period_6_month': '6 mesi',
    'paywall.period_year': 'anno',
    'paywall.manage_subscription': 'Gestisci abbonamento',
    'paywall.terms_link': 'Termini di servizio',
    'paywall.privacy_link': 'Informativa sulla privacy',
    'paywall.empty_title': 'Ancora niente da acquistare',
    'paywall.empty_body':
        'Al momento non ci sono abbonamenti disponibili. Torna più tardi.',
    'paywall.error_title': 'Impossibile raggiungere lo store',
    'paywall.error_body':
        'Impossibile caricare le opzioni di abbonamento. Controlla la connessione e riprova.',
    'paywall.retry': 'Riprova',
    'paywall.purchase_pending':
        'Il tuo acquisto è in attesa di approvazione. Verrà sbloccato una volta confermato.',
    'paywall.already_owned':
        'Possiedi già questo abbonamento — tocca Ripristina per ricollegarlo.',
    'paywall.restore_none': 'Nessun acquisto precedente da ripristinare.',
    'paywall.restore_done': 'I tuoi acquisti sono stati ripristinati.',
    'paywall.restore_failed': 'Impossibile ripristinare gli acquisti: {error}',
    'error.email_not_verified':
        'Verifica prima il tuo indirizzo email — controlla la posta in arrivo.',
    'error.signup_closed':
        'Le registrazioni sono attualmente chiuse per questa app.',
    'error.invalid_email': 'Questo indirizzo email non sembra corretto.',
    'error.session_expired': 'La tua sessione è scaduta — accedi di nuovo.',
    'error.user_disabled': 'Questo account è stato disattivato.',
    'error.rate_limited': 'Troppi tentativi — attendi un momento e riprova.',
    'error.provider_disabled':
        'Questo metodo di accesso non è abilitato per questa app.',
    'error.provider_failed': 'Accesso con il provider non riuscito. Riprova.',
    'error.last_login_method':
        'È il tuo unico modo per accedere, quindi non può essere rimosso.',
    'error.network':
        'Impossibile raggiungere il server. Controlla la connessione e riprova.',
    'error.generic': 'Qualcosa è andato storto. Riprova.',
  },
  'ja': <String, String>{
    'sign_in.title': 'ログイン',
    'sign_in.subtitle': '{app} へおかえりなさい。',
    'sign_in.email_label': 'メールアドレス',
    'sign_in.password_label': 'パスワード',
    'sign_in.submit': 'ログイン',
    'sign_in.forgot_password': 'パスワードをお忘れですか？',
    'sign_in.no_account': 'アカウントをお持ちでないですか？',
    'sign_in.switch_to_sign_up': '新規登録',
    'sign_in.error_invalid': 'メールアドレスまたはパスワードが正しくありません。',
    'sign_in.continue_with_google': 'Google で続ける',
    'sign_in.continue_with_apple': 'Apple で続ける',
    'sign_in.divider_or': 'または',
    'sign_in.email_required': 'メールアドレスを入力してください',
    'sign_in.email_invalid': '有効なメールアドレスを入力してください',
    'sign_in.password_required': 'パスワードを入力してください',
    'sign_in.terms_link': '利用規約',
    'sign_in.privacy_link': 'プライバシーポリシー',
    'sign_in.config_error_title': 'サーバーに接続できません',
    'sign_in.config_error_body': 'サインインオプションを読み込めませんでした。接続を確認してもう一度お試しください。',
    'sign_in.retry': '再試行',
    'sign_up.title': 'アカウント作成',
    'sign_up.subtitle': '{app} のアカウントを作成します。',
    'sign_up.email_label': 'メールアドレス',
    'sign_up.password_label': 'パスワード',
    'sign_up.submit': 'アカウントを作成',
    'sign_up.have_account': 'すでにアカウントをお持ちですか？',
    'sign_up.switch_to_sign_in': 'ログイン',
    'sign_up.legal': '続行すると、利用規約とプライバシーポリシーに同意したものとみなされます。',
    'sign_up.error_email_taken': 'このメールアドレスのアカウントはすでに存在します。',
    'sign_up.password_too_short': '{count} 文字以上で入力してください',
    'sign_up.verify_sent': 'アカウントを作成しました。受信トレイを確認してメールアドレスを認証し、サインインしてください。',
    'password_reset.title': 'パスワードをリセット',
    'password_reset.subtitle': 'メールアドレスを入力すると、リセット用のリンクを送信します。',
    'password_reset.email_label': 'メールアドレス',
    'password_reset.submit': 'リンクを送信',
    'password_reset.back_to_sign_in': 'サインインに戻る',
    'password_reset.sent': '{email} のアカウントが存在する場合、リセット用のリンクを送信します。',
    'password_reset.sent_title': 'メールを確認してください',
    'paywall.title': '{app} プレミアムを解除',
    'paywall.subtitle': 'すべての機能を無制限に利用できます。',
    'paywall.cta': '続ける',
    'paywall.restore': '購入を復元',
    'paywall.terms': '料金はアカウントに請求されます。いつでもキャンセルできます。',
    'paywall.most_popular': '一番人気',
    'paywall.trial_3_day': '3日間の無料トライアル',
    'paywall.trial_1_week': '1週間の無料トライアル',
    'paywall.trial_2_week': '2週間の無料トライアル',
    'paywall.trial_1_month': '1か月の無料トライアル',
    'paywall.trial_generic': '無料トライアル',
    'paywall.period_week': '週',
    'paywall.period_month': '月',
    'paywall.period_quarter': '四半期',
    'paywall.period_6_month': '6か月',
    'paywall.period_year': '年',
    'paywall.manage_subscription': 'サブスクリプションを管理',
    'paywall.terms_link': '利用規約',
    'paywall.privacy_link': 'プライバシーポリシー',
    'paywall.empty_title': 'まだ購入できる商品がありません',
    'paywall.empty_body': '現在利用できるサブスクリプションはありません。後ほどご確認ください。',
    'paywall.error_title': 'ストアに接続できません',
    'paywall.error_body': 'サブスクリプションのオプションを読み込めませんでした。接続を確認してもう一度お試しください。',
    'paywall.retry': '再試行',
    'paywall.purchase_pending': '購入は承認待ちです。確認され次第、利用できるようになります。',
    'paywall.already_owned': 'このサブスクリプションはすでに購入済みです。「復元」をタップして再度リンクしてください。',
    'paywall.restore_none': '復元できる過去の購入が見つかりませんでした。',
    'paywall.restore_done': '購入を復元しました。',
    'paywall.restore_failed': '購入を復元できませんでした: {error}',
    'error.email_not_verified': 'まずメールアドレスを認証してください。受信トレイをご確認ください。',
    'error.signup_closed': 'このアプリの新規登録は現在受け付けていません。',
    'error.invalid_email': 'そのメールアドレスは正しくないようです。',
    'error.session_expired': 'セッションの有効期限が切れました。もう一度サインインしてください。',
    'error.user_disabled': 'このアカウントは無効化されています。',
    'error.rate_limited': '試行回数が多すぎます。しばらくしてからもう一度お試しください。',
    'error.provider_disabled': 'このサインイン方法はこのアプリでは有効になっていません。',
    'error.provider_failed': 'プロバイダーでのサインインに失敗しました。もう一度お試しください。',
    'error.last_login_method': 'これは唯一のサインイン方法のため、削除できません。',
    'error.network': 'サーバーに接続できません。接続を確認してもう一度お試しください。',
    'error.generic': '問題が発生しました。もう一度お試しください。',
  },
};

/// The bundled fallback copy for [locale]'s language, as a message key ->
/// string map, with English filled in for any key the language lacks (so every
/// bundled-screen key always resolves non-empty). A language the SDK does not
/// bundle returns the full English map.
Map<String, String> bundledCopy(Locale locale) {
  final en = _bundled['en']!;
  final lang = _bundled[locale.languageCode];
  if (lang == null) return Map<String, String>.of(en);
  return <String, String>{
    for (final entry in en.entries)
      entry.key: (lang[entry.key]?.isNotEmpty ?? false)
          ? lang[entry.key]!
          : entry.value,
  };
}
