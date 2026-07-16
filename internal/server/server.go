// Package server assembles the moth HTTP handler: connect (gRPC /
// gRPC-Web) services and the plain-HTTP surfaces, multiplexed on one port.
package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync/atomic"

	"connectrpc.com/connect"
	connectcors "connectrpc.com/cors"
	"connectrpc.com/grpchealth"
	"connectrpc.com/grpcreflect"
	"github.com/rs/cors"

	"github.com/aloisdeniel/moth/gen/moth/admin/v1/adminv1connect"
	"github.com/aloisdeniel/moth/gen/moth/auth/v1/authv1connect"
	"github.com/aloisdeniel/moth/gen/moth/server/v1/serverv1connect"
	"github.com/aloisdeniel/moth/internal/config"
	"github.com/aloisdeniel/moth/internal/keys"
	"github.com/aloisdeniel/moth/internal/mail"
	adminrpc "github.com/aloisdeniel/moth/internal/server/rpc/admin"
	authrpc "github.com/aloisdeniel/moth/internal/server/rpc/auth"
	"github.com/aloisdeniel/moth/internal/server/rpc/serverapi"
	"github.com/aloisdeniel/moth/internal/store"
	"github.com/aloisdeniel/moth/internal/version"
)

// Store is everything the assembled server needs from persistence.
type Store interface {
	adminrpc.Store
	store.EmailTokenStore
	store.EventStore
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
		cfg:    o.Config,
		store:  o.Store,
		master: o.Master,
		log:    o.Logger,
	}
	s.setupToken.Store(o.SetupToken)

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

	s.auth = authrpc.New(authrpc.Options{
		Store:   o.Store,
		Master:  o.Master,
		Mailer:  dynMailer,
		BaseURL: o.Config.BaseURL,
		Logger:  o.Logger,
	})

	// chain prepends the shared observability interceptors to a service's
	// own auth interceptors.
	chain := func(extra ...connect.Interceptor) connect.Option {
		all := []connect.Interceptor{
			newRequestIDInterceptor(),
			newRecoveryInterceptor(o.Logger),
			newLoggingInterceptor(o.Logger),
		}
		return connect.WithInterceptors(append(all, extra...)...)
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
		adminrpc.NewUserHandler(o.Store, s.auth, dynMailer), adminInterceptors)
	mux.Handle(adminUserPath, adminUserHandler)
	accountPath, accountHandler := adminv1connect.NewAdminAccountServiceHandler(
		adminrpc.NewAccountHandler(o.Store, dynMailer, o.Config.BaseURL,
			o.Config.Secure(), settingsHandler.SMTPConfigured), adminInterceptors)
	mux.Handle(accountPath, accountHandler)
	settingsPath, settingsSvcHandler := adminv1connect.NewInstanceSettingsServiceHandler(
		settingsHandler, adminInterceptors)
	mux.Handle(settingsPath, settingsSvcHandler)

	// moth.auth.v1 — the public end-user API (publishable-key auth).
	authPath, authHandler := authv1connect.NewAuthServiceHandler(s.auth, authInterceptors)
	mux.Handle(authPath, authHandler)

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
		authv1connect.AuthServiceName,
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

	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("GET /p/{slug}/.well-known/jwks.json", s.handleJWKS)
	mux.HandleFunc("GET /p/{slug}/verify", s.handleVerifyPage)
	mux.HandleFunc("GET /p/{slug}/reset", s.handleResetPage)
	mux.HandleFunc("POST /p/{slug}/reset", s.handleResetSubmit)
	mux.HandleFunc("GET /p/{slug}/confirm-email", s.handleConfirmEmailPage)
	mux.HandleFunc("GET /admin", s.handleAdminPage)
	mux.HandleFunc("GET /admin/", s.handleAdminPage)
	mux.HandleFunc("GET /admin/status", s.handleAdminStatus)
	mux.HandleFunc("POST /admin/setup", s.handleAdminSetup)
	mux.HandleFunc("/", s.handleRoot)

	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins:   []string{o.Config.BaseOrigin()},
		AllowedMethods:   connectcors.AllowedMethods(),
		AllowedHeaders:   append(connectcors.AllowedHeaders(), "X-Moth-Key"),
		ExposedHeaders:   append(connectcors.ExposedHeaders(), requestIDHeader),
		AllowCredentials: true,
		MaxAge:           7200,
	})

	s.handler = corsMiddleware.Handler(mux)
	return s, nil
}

// Handler returns the root HTTP handler. Serve it with Protocols() so
// native gRPC clients can speak HTTP/2 without TLS (h2c) on the same port
// as plain HTTP/1.1 (browsers, curl, pub client).
func (s *Server) Handler() http.Handler { return s.handler }

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
