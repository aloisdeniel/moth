// Package server assembles the moth HTTP handler: connect (gRPC /
// gRPC-Web) services and the plain-HTTP surfaces, multiplexed on one port.
package server

import (
	"log/slog"
	"net/http"
	"sync/atomic"

	"connectrpc.com/connect"
	connectcors "connectrpc.com/cors"
	"connectrpc.com/grpchealth"
	"connectrpc.com/grpcreflect"
	"github.com/rs/cors"

	"github.com/aloisdeniel/moth/gen/moth/admin/v1/adminv1connect"
	"github.com/aloisdeniel/moth/internal/config"
	"github.com/aloisdeniel/moth/internal/keys"
	adminrpc "github.com/aloisdeniel/moth/internal/server/rpc/admin"
	"github.com/aloisdeniel/moth/internal/version"
)

// Options configures a Server.
type Options struct {
	Config config.Config
	Store  adminrpc.Store
	Master keys.MasterKey
	Logger *slog.Logger
	// SetupToken guards the first-run admin setup screen. Empty when an
	// admin account already exists.
	SetupToken string
}

// Server is the assembled moth server.
type Server struct {
	cfg        config.Config
	store      adminrpc.Store
	master     keys.MasterKey
	log        *slog.Logger
	setupToken atomic.Value // string; "" once setup is complete
	handler    http.Handler
}

// New assembles the full handler.
func New(o Options) *Server {
	if o.Logger == nil {
		o.Logger = slog.Default()
	}
	s := &Server{
		cfg:    o.Config,
		store:  o.Store,
		master: o.Master,
		log:    o.Logger,
	}
	s.setupToken.Store(o.SetupToken)

	interceptors := connect.WithInterceptors(
		newRequestIDInterceptor(),
		newRecoveryInterceptor(o.Logger),
		newLoggingInterceptor(o.Logger),
		adminrpc.NewAuthInterceptor(o.Store),
	)

	mux := http.NewServeMux()

	sessionPath, sessionHandler := adminv1connect.NewSessionServiceHandler(
		adminrpc.NewSessionHandler(o.Store, o.Config.Secure()), interceptors)
	mux.Handle(sessionPath, sessionHandler)

	projectPath, projectHandler := adminv1connect.NewProjectServiceHandler(
		adminrpc.NewProjectHandler(o.Store, o.Master), interceptors)
	mux.Handle(projectPath, projectHandler)

	serviceNames := []string{
		adminv1connect.SessionServiceName,
		adminv1connect.ProjectServiceName,
	}
	mux.Handle(grpchealth.NewHandler(grpchealth.NewStaticChecker(serviceNames...)))
	if version.IsDev() {
		reflector := grpcreflect.NewStaticReflector(serviceNames...)
		mux.Handle(grpcreflect.NewHandlerV1(reflector))
		mux.Handle(grpcreflect.NewHandlerV1Alpha(reflector))
	}

	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("GET /p/{slug}/.well-known/jwks.json", s.handleJWKS)
	mux.HandleFunc("GET /admin", s.handleAdminPage)
	mux.HandleFunc("GET /admin/", s.handleAdminPage)
	mux.HandleFunc("GET /admin/status", s.handleAdminStatus)
	mux.HandleFunc("POST /admin/setup", s.handleAdminSetup)
	mux.HandleFunc("/", s.handleRoot)

	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins:   []string{o.Config.BaseOrigin()},
		AllowedMethods:   connectcors.AllowedMethods(),
		AllowedHeaders:   connectcors.AllowedHeaders(),
		ExposedHeaders:   append(connectcors.ExposedHeaders(), requestIDHeader),
		AllowCredentials: true,
		MaxAge:           7200,
	})

	s.handler = corsMiddleware.Handler(mux)
	return s
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
