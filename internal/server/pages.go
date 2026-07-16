package server

import (
	"errors"
	"html/template"
	"net/http"

	"connectrpc.com/connect"

	authv1 "github.com/aloisdeniel/moth/gen/moth/auth/v1"
	"github.com/aloisdeniel/moth/internal/httpsec"
	authrpc "github.com/aloisdeniel/moth/internal/server/rpc/auth"
	"github.com/aloisdeniel/moth/internal/store"
)

// Hosted confirmation pages: the links in auth emails must open in a
// browser, so these plain-HTML pages invoke the confirm RPCs in-process.
// They work before the app has any deep links.

var pageTemplate = template.Must(template.ParseFS(webFS, "web/page.html.tmpl"))

type pageData struct {
	Project       string
	Title         string
	Message       string
	Error         string
	ShowResetForm bool
	Token         string
	// Nonce is the per-request CSP nonce stamped onto the inline <style>
	// element so the strict hosted-page policy admits it without
	// 'unsafe-inline'.
	Nonce string
	// Theme fields, filled by themedData from the project's design system.
	ThemeCSS   template.CSS
	LogoLight  string
	LogoDark   string
	TermsURL   string
	PrivacyURL string
}

// handleVerifyPage consumes an email-verification link.
func (s *Server) handleVerifyPage(w http.ResponseWriter, r *http.Request) {
	project, ok := s.pageProject(w, r)
	if !ok {
		return
	}
	ctx := authrpc.WithProject(r.Context(), project)
	data := pageData{Title: "Email verified",
		Message: "Your email address is verified. You can return to the app."}
	_, err := s.auth.ConfirmEmailVerification(ctx, connect.NewRequest(
		&authv1.ConfirmEmailVerificationRequest{Token: r.URL.Query().Get("token")}))
	if err != nil {
		data.Title = "Verification failed"
		data.Error = "This link is invalid or has expired. Request a new verification email from the app."
	}
	s.renderPage(w, r, project, data)
}

// handleResetPage shows the new-password form of a reset link.
func (s *Server) handleResetPage(w http.ResponseWriter, r *http.Request) {
	project, ok := s.pageProject(w, r)
	if !ok {
		return
	}
	s.renderPage(w, r, project, pageData{
		Title:         "Choose a new password",
		Message:       "Enter a new password for your account.",
		ShowResetForm: true,
		Token:         r.URL.Query().Get("token"),
	})
}

// handleResetSubmit completes a password reset.
func (s *Server) handleResetSubmit(w http.ResponseWriter, r *http.Request) {
	project, ok := s.pageProject(w, r)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	ctx := authrpc.WithProject(r.Context(), project)
	token := r.PostFormValue("token")
	_, err := s.auth.ConfirmPasswordReset(ctx, connect.NewRequest(&authv1.ConfirmPasswordResetRequest{
		Token:       token,
		NewPassword: r.PostFormValue("password"),
	}))
	data := pageData{Title: "Password updated",
		Message: "Your password was changed. You can now sign in to the app with it. All other sessions were signed out."}
	switch {
	case err != nil && authrpc.ErrorReason(err) == authrpc.ReasonWeakPassword:
		// Keep the form up so the user can try a longer password.
		data.Title = "Choose a new password"
		data.Error = "That password is too short for this app. Try a longer one."
		data.ShowResetForm = true
		data.Token = token
	case err != nil:
		data.Title = "Reset failed"
		data.Error = "This link is invalid or has expired. Request a new password reset from the app."
	}
	s.renderPage(w, r, project, data)
}

// handleConfirmEmailPage consumes email-change confirmation and revert
// links.
func (s *Server) handleConfirmEmailPage(w http.ResponseWriter, r *http.Request) {
	project, ok := s.pageProject(w, r)
	if !ok {
		return
	}
	ctx := authrpc.WithProject(r.Context(), project)
	data := pageData{Title: "Email address updated",
		Message: "The email address on your account was updated. Sign in with it from now on."}
	_, err := s.auth.ConfirmEmailChange(ctx, connect.NewRequest(
		&authv1.ConfirmEmailChangeRequest{Token: r.URL.Query().Get("token")}))
	if err != nil {
		data.Title = "Update failed"
		data.Error = "This link is invalid or has expired."
	}
	s.renderPage(w, r, project, data)
}

func (s *Server) pageProject(w http.ResponseWriter, r *http.Request) (store.Project, bool) {
	project, err := s.store.GetProjectBySlug(r.Context(), r.PathValue("slug"))
	if errors.Is(err, store.ErrNotFound) {
		http.NotFound(w, r)
		return store.Project{}, false
	}
	if err != nil {
		s.internalError(w, r, err)
		return store.Project{}, false
	}
	return project, true
}

func (s *Server) renderPage(w http.ResponseWriter, r *http.Request, p store.Project, data pageData) {
	s.renderPageStatus(w, r, http.StatusOK, p, data)
}

func (s *Server) renderPageStatus(w http.ResponseWriter, r *http.Request, status int, p store.Project, data pageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Referrer-Policy", "no-referrer")
	// The security middleware minted a CSP nonce for this request; the inline
	// <style> block carries it so the strict policy admits it.
	data.Nonce = httpsec.NonceFromContext(r.Context())
	w.WriteHeader(status)
	if err := pageTemplate.Execute(w, themedData(p, data)); err != nil {
		s.log.Error("render page", "path", r.URL.Path, "error", err.Error())
	}
}
