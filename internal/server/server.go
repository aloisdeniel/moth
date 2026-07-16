// Package server assembles the moth HTTP handler: connect (gRPC /
// gRPC-Web) services and the plain-HTTP surfaces, multiplexed on one port.
package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"connectrpc.com/connect"
	connectcors "connectrpc.com/cors"
	"connectrpc.com/grpchealth"
	"connectrpc.com/grpcreflect"
	"github.com/rs/cors"

	"github.com/aloisdeniel/moth/gen/moth/admin/v1/adminv1connect"
	"github.com/aloisdeniel/moth/gen/moth/auth/v1/authv1connect"
	"github.com/aloisdeniel/moth/gen/moth/server/v1/serverv1connect"
	"github.com/aloisdeniel/moth/internal/analytics"
	"github.com/aloisdeniel/moth/internal/config"
	"github.com/aloisdeniel/moth/internal/events"
	"github.com/aloisdeniel/moth/internal/fonts"
	"github.com/aloisdeniel/moth/internal/keys"
	"github.com/aloisdeniel/moth/internal/mail"
	adminrpc "github.com/aloisdeniel/moth/internal/server/rpc/admin"
	authrpc "github.com/aloisdeniel/moth/internal/server/rpc/auth"
	"github.com/aloisdeniel/moth/internal/server/rpc/serverapi"
	"github.com/aloisdeniel/moth/internal/store"
	"github.com/aloisdeniel/moth/internal/version"
)

// rpcReadMaxBytes caps the size of a single decoded RPC message on every
// service. The largest legitimate message is an UploadLogo carrying its
// 512 KiB image payload (see adminrpc.maxLogoBytes); 1 MiB leaves headroom
// for encoding overhead while rejecting oversized bodies during the read
// instead of after they are fully buffered.
const rpcReadMaxBytes = 1 << 20

// Store is everything the assembled server needs from persistence.
// adminrpc.Store already covers the event and daily-stats surfaces the
// analytics pipeline needs.
type Store interface {
	adminrpc.Store
	store.EmailTokenStore
	store.OAuthTokenStore
}

// Options configures a Server.
type Options struct {
	Config config.Config
	Store  Store
	Master keys.MasterKey
	Logger *slog.Logger
	// Mailer delivers auth emails; nil falls back to the console
	// transport (dev default).
	Mailer mail.Mailer
	// RateLimits override the auth-service throttles; zero value means
	// defaults.
	RateLimits authrpc.RateLimits
	// SetupToken guards the first-run admin setup screen. Empty when an
	// admin account already exists.
	SetupToken string
	// AuthEndpoints override the Google/Apple OAuth endpoint locations;
	// zero value means the real providers (tests point them at doubles).
	AuthEndpoints authrpc.ProviderEndpoints
	// Now is injectable for tests; defaults to time.Now.
	Now func() time.Time
}

// Server is the assembled moth server.
type Server struct {
	cfg        config.Config
	store      Store
	master     keys.MasterKey
	log        *slog.Logger
	auth       *authrpc.Handler // shared with the hosted confirmation pages
	setupToken atomic.Value     // string; "" once setup is complete
	handler    http.Handler
	// pub is the embedded moth_auth Flutter package, built once so the
	// sha256 in the /pub version listing always matches the served bytes.
	pub *pubArchive
	// rateLimits also throttles the plain-HTTP OAuth endpoints, which sit
	// outside the connect interceptor chain.
	rateLimits authrpc.RateLimits
	// uploads is where theme assets (project logos) live on disk; served
	// back at /assets/{projectID}/....
	uploads string
	// fonts serves the embedded font catalogue under /assets/fonts/.
	fonts http.Handler
	// events is the async analytics writer the handlers emit through;
	// Close drains it on shutdown.
	events *events.Writer
	// rollup is the analytics aggregate-and-prune job, shared by the
	// AnalyticsService handler and the serve-loop scheduler.
	rollup *analytics.Rollup
}

// New assembles the full handler.
func New(o Options) (*Server, error) {
	if o.Logger == nil {
		o.Logger = slog.Default()
	}
	if o.Mailer == nil {
		o.Mailer = mail.Console{Log: o.Logger}
	}
	if o.RateLimits.PerIP == nil || o.RateLimits.PerAccount == nil {
		o.RateLimits = authrpc.DefaultRateLimits()
	}
	s := &Server{
		cfg:        o.Config,
		store:      o.Store,
		master:     o.Master,
		log:        o.Logger,
		rateLimits: o.RateLimits,
		uploads:    filepath.Join(o.Config.DataDir, "uploads"),
		fonts:      http.StripPrefix("/assets/fonts/", fonts.Handler()),
		// Analytics events flow through a bounded async writer so emission
		// never adds latency to auth; Server.Close drains it on shutdown.
		events: events.NewWriter(eventSink{o.Store}, events.Config{Logger: o.Logger}),
		rollup: analytics.NewRollup(o.Store, o.Logger, o.Now),
	}
	s.setupToken.Store(o.SetupToken)

	// The moth_auth Flutter SDK served at /pub; its version tracks the
	// binary's own build version.
	pub, err := buildPubArchive(version.Version)
	if err != nil {
		return nil, err
	}
	s.pub = pub

	// Every email goes through one swappable transport so the admin
	// console can reconfigure SMTP at runtime. Options.Mailer is the
	// transport of last resort (console logger, or a recording mailer in
	// tests); the settings handler points dyn at the effective SMTP.
	dynMailer := mail.NewDynamic(o.Mailer)
	settingsHandler, err := adminrpc.NewSettingsHandler(
		context.Background(), o.Store, o.Config, dynMailer, o.Mailer)
	if err != nil {
		return nil, fmt.Errorf("resolve smtp settings: %w", err)
	}

	// One shared, timeout-bounded client for every outbound provider call
	// (JWKS fetches, code exchanges, Apple revocations).
	httpc := &http.Client{Timeout: 10 * time.Second}
	s.auth = authrpc.New(authrpc.Options{
		Store:      o.Store,
		Master:     o.Master,
		Mailer:     dynMailer,
		BaseURL:    o.Config.BaseURL,
		Logger:     o.Logger,
		Now:        o.Now,
		HTTPClient: httpc,
		Endpoints:  o.AuthEndpoints,
		Events:     s.events,
	})

	// chain prepends the shared observability interceptors to a service's
	// own auth interceptors, and caps how much of a request body connect
	// will buffer for a single message: interceptors (auth included) only
	// run on fully decoded messages, so without a read cap any caller could
	// make the server buffer an arbitrarily large body before the first
	// check runs.
	chain := func(extra ...connect.Interceptor) connect.Option {
		all := []connect.Interceptor{
			newRequestIDInterceptor(),
			newVersionInterceptor(),
			newRecoveryInterceptor(o.Logger),
			newLoggingInterceptor(o.Logger),
		}
		return connect.WithOptions(
			connect.WithReadMaxBytes(rpcReadMaxBytes),
			connect.WithInterceptors(append(all, extra...)...),
		)
	}
	adminInterceptors := chain(adminrpc.NewAuthInterceptor(o.Store))
	authInterceptors := chain(
		authrpc.NewProjectInterceptor(o.Store),
		authrpc.NewRateLimitInterceptor(o.RateLimits))
	serverInterceptors := chain(serverapi.NewSecretKeyInterceptor(o.Store))

	mux := http.NewServeMux()

	// moth.admin.v1 — the admin console (session-cookie auth).
	sessionPath, sessionHandler := adminv1connect.NewSessionServiceHandler(
		adminrpc.NewSessionHandler(o.Store, o.Config.Secure()), adminInterceptors)
	mux.Handle(sessionPath, sessionHandler)
	projectPath, projectHandler := adminv1connect.NewProjectServiceHandler(
		adminrpc.NewProjectHandler(o.Store, o.Master, o.Config.BaseURL), adminInterceptors)
	mux.Handle(projectPath, projectHandler)
	adminUserPath, adminUserHandler := adminv1connect.NewUserServiceHandler(
		adminrpc.NewUserHandler(o.Store, s.auth, dynMailer, s.events), adminInterceptors)
	mux.Handle(adminUserPath, adminUserHandler)
	accountPath, accountHandler := adminv1connect.NewAdminAccountServiceHandler(
		adminrpc.NewAccountHandler(o.Store, dynMailer, o.Config.BaseURL,
			o.Config.Secure(), settingsHandler.SMTPConfigured), adminInterceptors)
	mux.Handle(accountPath, accountHandler)
	settingsPath, settingsSvcHandler := adminv1connect.NewInstanceSettingsServiceHandler(
		settingsHandler, adminInterceptors)
	mux.Handle(settingsPath, settingsSvcHandler)
	themePath, themeHandler := adminv1connect.NewThemeServiceHandler(
		adminrpc.NewThemeHandler(o.Store, s.uploads), adminInterceptors)
	mux.Handle(themePath, themeHandler)
	analyticsPath, analyticsHandler := adminv1connect.NewAnalyticsServiceHandler(
		adminrpc.NewAnalyticsHandler(o.Store, s.rollup, o.Now), adminInterceptors)
	mux.Handle(analyticsPath, analyticsHandler)

	// moth.auth.v1 — the public end-user API (publishable-key auth).
	authPath, authHandler := authv1connect.NewAuthServiceHandler(s.auth, authInterceptors)
	mux.Handle(authPath, authHandler)
	configPath, configHandler := authv1connect.NewConfigServiceHandler(s.auth, authInterceptors)
	mux.Handle(configPath, configHandler)

	// moth.server.v1 — the developer-backend API (secret-key auth).
	tokenPath, tokenHandler := serverv1connect.NewTokenServiceHandler(
		serverapi.NewTokenHandler(o.Store, nil), serverInterceptors)
	mux.Handle(tokenPath, tokenHandler)
	serverUserPath, serverUserHandler := serverv1connect.NewUserServiceHandler(
		serverapi.NewUserHandler(o.Store, nil), serverInterceptors)
	mux.Handle(serverUserPath, serverUserHandler)

	serviceNames := []string{
		adminv1connect.SessionServiceName,
		adminv1connect.ProjectServiceName,
		adminv1connect.UserServiceName,
		adminv1connect.AdminAccountServiceName,
		adminv1connect.InstanceSettingsServiceName,
		adminv1connect.ThemeServiceName,
		adminv1connect.AnalyticsServiceName,
		authv1connect.AuthServiceName,
		authv1connect.ConfigServiceName,
		serverv1connect.TokenServiceName,
		serverv1connect.UserServiceName,
	}
	mux.Handle(grpchealth.NewHandler(grpchealth.NewStaticChecker(serviceNames...)))
	if version.IsDev() {
		reflector := grpcreflect.NewStaticReflector(serviceNames...)
		mux.Handle(grpcreflect.NewHandlerV1(reflector))
		mux.Handle(grpcreflect.NewHandlerV1Alpha(reflector))
	}

	// The .proto sources, offered for download from the setup page.
	mux.Handle("GET /protos/", http.StripPrefix("/protos/", http.HandlerFunc(s.handleProtoFile)))

	// The pub hosted repository serving the moth_auth Flutter SDK
	// (`dart pub` speaks plain HTTP; see plan/05).
	mux.HandleFunc("GET /pub/api/packages/{package}", s.handlePubVersions)
	mux.HandleFunc("GET /pub/packages/{package}/versions/{file}", s.handlePubArchive)

	// Theme assets: embedded fonts and uploaded project logos. One wildcard
	// route because "/assets/fonts/" and "/assets/{project}/{file}" would
	// overlap ambiguously as separate mux patterns.
	mux.HandleFunc("GET /assets/{path...}", s.handleAsset)

	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("GET /p/{slug}/.well-known/jwks.json", s.handleJWKS)
	mux.HandleFunc("GET /p/{slug}/verify", s.handleVerifyPage)
	mux.HandleFunc("GET /p/{slug}/reset", s.handleResetPage)
	mux.HandleFunc("POST /p/{slug}/reset", s.handleResetSubmit)
	mux.HandleFunc("GET /p/{slug}/confirm-email", s.handleConfirmEmailPage)
	// Web-redirect OAuth fallback. The callback URL pasted into the
	// provider consoles is project-agnostic; the state carries the project.
	mux.HandleFunc("GET /oauth/{provider}/start", s.handleOAuthStart)
	mux.HandleFunc("GET /oauth/{provider}/callback", s.handleOAuthCallback)
	// Apple posts the callback (response_mode=form_post).
	mux.HandleFunc("POST /oauth/{provider}/callback", s.handleOAuthCallback)
	mux.HandleFunc("GET /admin", s.handleAdminPage)
	mux.HandleFunc("GET /admin/", s.handleAdminPage)
	mux.HandleFunc("GET /admin/status", s.handleAdminStatus)
	mux.HandleFunc("GET /admin/export/stats.csv", s.handleExportStats)
	mux.HandleFunc("POST /admin/setup", s.handleAdminSetup)
	mux.HandleFunc("/", s.handleRoot)

	// Two CORS policies. The end-user surface (moth.auth.v1 via gRPC-Web,
	// the pub repository, JWKS and hosted pages) must be callable from any
	// origin: a Flutter Web app is essentially never served from the moth
	// origin. It authenticates with publishable keys and Bearer tokens,
	// never cookies, so a wildcard origin without credentials is safe. The
	// admin surface rides the session cookie and stays locked to the
	// instance's own origin.
	publicCORS := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: connectcors.AllowedMethods(),
		// The SDK attaches x-moth-key, x-moth-platform, x-moth-sdk-version
		// and authorization metadata to every call; with no credentials in
		// play the wildcard allows them all (and whatever later SDKs add).
		AllowedHeaders: []string{"*"},
		ExposedHeaders: append(connectcors.ExposedHeaders(), requestIDHeader, versionHeader),
		MaxAge:         7200,
	})
	adminCORS := cors.New(cors.Options{
		AllowedOrigins:   []string{o.Config.BaseOrigin()},
		AllowedMethods:   connectcors.AllowedMethods(),
		AllowedHeaders:   append(connectcors.AllowedHeaders(), "X-Moth-Key"),
		ExposedHeaders:   append(connectcors.ExposedHeaders(), requestIDHeader, versionHeader),
		AllowCredentials: true,
		MaxAge:           7200,
	})

	publicHandler := publicCORS.Handler(mux)
	adminHandler := adminCORS.Handler(mux)
	s.handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPublicSurface(r.URL.Path) {
			publicHandler.ServeHTTP(w, r)
			return
		}
		adminHandler.ServeHTTP(w, r)
	})
	return s, nil
}

// isPublicSurface reports whether the path belongs to the end-user API or
// another public resource browsers may fetch cross-origin: the moth.auth.v1
// services, the pub repository, the per-project hosted pages/JWKS and the
// theme assets (logos, fonts) that Flutter Web apps download.
func isPublicSurface(path string) bool {
	return strings.HasPrefix(path, "/moth.auth.v1.") ||
		strings.HasPrefix(path, "/pub/") ||
		strings.HasPrefix(path, "/p/") ||
		strings.HasPrefix(path, "/assets/")
}

// Handler returns the root HTTP handler. Serve it with Protocols() so
// native gRPC clients can speak HTTP/2 without TLS (h2c) on the same port
// as plain HTTP/1.1 (browsers, curl, pub client).
func (s *Server) Handler() http.Handler { return s.handler }

// Close stops the async analytics writer, draining its buffered events
// until ctx expires. Call it after the HTTP server has stopped accepting
// requests so no event is emitted into a closed writer.
func (s *Server) Close(ctx context.Context) error { return s.events.Close(ctx) }

// Rollup exposes the analytics aggregate-and-prune job so the serve loop
// can schedule it next to its other maintenance goroutines.
func (s *Server) Rollup() *analytics.Rollup { return s.rollup }

// Protocols returns the protocol set for the http.Server: HTTP/1.1 plus
// unencrypted HTTP/2.
func Protocols() *http.Protocols {
	p := new(http.Protocols)
	p.SetHTTP1(true)
	p.SetUnencryptedHTTP2(true)
	return p
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/admin", http.StatusFound)
}
