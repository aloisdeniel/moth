package server

import (
	"crypto/subtle"
	"embed"
	"encoding/json"
	"errors"
	"mime"
	"net/http"
	"net/mail"
	"path"
	"strings"
	"time"

	"github.com/aloisdeniel/moth/internal/keys"
	"github.com/aloisdeniel/moth/internal/password"
	adminrpc "github.com/aloisdeniel/moth/internal/server/rpc/admin"
	"github.com/aloisdeniel/moth/internal/store"
	protosrc "github.com/aloisdeniel/moth/proto"
)

//go:embed all:web/dist web/page.html.tmpl
var webFS embed.FS

const minPasswordLen = 8

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleProtoFile serves the embedded .proto sources (e.g.
// /protos/moth/auth/v1/auth.proto) so developers can generate their own
// backend clients against this instance.
func (s *Server) handleProtoFile(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Path
	if !strings.HasSuffix(name, ".proto") {
		http.NotFound(w, r)
		return
	}
	raw, err := protosrc.FS.ReadFile(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename="+path.Base(name))
	w.Write(raw)
}

// handleJWKS serves a project's active public keys so any standard JWT
// library can verify that project's tokens offline.
func (s *Server) handleJWKS(w http.ResponseWriter, r *http.Request) {
	project, err := s.store.GetProjectBySlug(r.Context(), r.PathValue("slug"))
	if errors.Is(err, store.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	projectKeys, err := s.store.ListActiveProjectKeys(r.Context(), project.ID)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	pems := make(map[string]string, len(projectKeys))
	for _, k := range projectKeys {
		pems[k.Kid] = k.PublicKeyPEM
	}
	doc, err := keys.BuildJWKS(pems)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.Write(doc)
}

// handleAdminPage serves the embedded admin SPA: real files from the Vite
// build output, and index.html for every client-routed path (SPA
// fallback).
func (s *Server) handleAdminPage(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(strings.TrimPrefix(r.URL.Path, "/admin"), "/")
	if name != "" && name != "index.html" {
		if raw, err := webFS.ReadFile("web/dist/" + name); err == nil {
			if ct := mime.TypeByExtension(path.Ext(name)); ct != "" {
				w.Header().Set("Content-Type", ct)
			}
			// Vite fingerprints everything under assets/; cache those hard.
			if strings.HasPrefix(name, "assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			w.Write(raw)
			return
		}
	}
	page, err := webFS.ReadFile("web/dist/index.html")
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write(page)
}

// handleAdminStatus tells the admin page whether first-run setup is needed.
func (s *Server) handleAdminStatus(w http.ResponseWriter, r *http.Request) {
	count, err := s.store.CountAdmins(r.Context())
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"needsSetup": count == 0})
}

// handleAdminSetup creates the very first admin account. It is guarded by
// the one-time token printed to the server console and refuses to run once
// any admin exists.
func (s *Server) handleAdminSetup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token    string `json:"token"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	expected, _ := s.setupToken.Load().(string)
	if expected == "" ||
		subtle.ConstantTimeCompare([]byte(req.Token), []byte(expected)) != 1 {
		writeJSONError(w, http.StatusForbidden, "invalid setup token")
		return
	}
	count, err := s.store.CountAdmins(r.Context())
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	if count > 0 {
		writeJSONError(w, http.StatusConflict, "an admin account already exists")
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	if _, err := mail.ParseAddress(email); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid email address")
		return
	}
	if len(req.Password) < minPasswordLen {
		writeJSONError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}
	hash, err := password.Hash(req.Password)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	now := time.Now()
	admin := store.Admin{
		ID:           adminrpc.NewID(),
		Email:        email,
		PasswordHash: hash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.store.CreateAdmin(r.Context(), admin); err != nil {
		s.internalError(w, r, err)
		return
	}
	s.setupToken.Store("")
	s.log.Info("first admin account created", "email", email)

	cookie, err := adminrpc.IssueSession(r.Context(), s.store, admin.ID, s.cfg.Secure())
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	http.SetCookie(w, cookie)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) internalError(w http.ResponseWriter, r *http.Request, err error) {
	s.log.Error("http", "path", r.URL.Path, "error", err.Error())
	writeJSONError(w, http.StatusInternalServerError, "internal error")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
