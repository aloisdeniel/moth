package server

import (
	"errors"
	"html/template"
	"net/http"

	"connectrpc.com/connect"

	authv1 "github.com/aloisdeniel/moth/gen/moth/auth/v1"
	"github.com/aloisdeniel/moth/internal/httpsec"
	"github.com/aloisdeniel/moth/internal/i18n"
	authrpc "github.com/aloisdeniel/moth/internal/server/rpc/auth"
	"github.com/aloisdeniel/moth/internal/store"
)

// Hosted confirmation pages: the links in auth emails must open in a
// browser, so these plain-HTML pages invoke the confirm RPCs in-process.
// They work before the app has any deep links. Every user-facing string is
// resolved from the i18n catalog for the locale negotiated from the request's
// Accept-Language (the project's bundled ∪ override locales); the negotiated
// locale is stamped as the page's lang/dir and echoed as x-moth-locale.

var pageTemplate = template.Must(template.ParseFS(webFS, "web/page.html.tmpl"))

type pageData struct {
	Project string
	Title   string
	Message string
	Error   string
	// IsError styles the heading in the error color (a failed confirmation).
	IsError       bool
	ShowResetForm bool
	Token         string
	// PasswordLabel and SubmitLabel localize the reset form controls.
	PasswordLabel string
	SubmitLabel   string
	// Lang and Dir are the negotiated locale's html attributes.
	Lang string
	Dir  string
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

// localize negotiates the hosted-page locale for the request against the
// project's copy, degrading to bundled defaults when the copy store cannot be
// read (a hosted page must still render).
func (s *Server) localize(r *http.Request, p store.Project) authrpc.Localizer {
	loc, err := authrpc.NewLocalizer(r.Context(), s.store, p.ID, r.Header)
	if err != nil {
		s.log.ErrorContext(r.Context(), "hosted page localize", "error", err.Error())
		return authrpc.NewFallbackLocalizer()
	}
	return loc
}

// handleVerifyPage consumes an email-verification link.
func (s *Server) handleVerifyPage(w http.ResponseWriter, r *http.Request) {
	project, ok := s.pageProject(w, r)
	if !ok {
		return
	}
	loc := s.localize(r, project)
	vars := map[string]string{"app": project.Name}
	ctx := authrpc.WithProject(r.Context(), project)
	data := pageData{Title: loc.Value(i18n.HostedVerifySuccess, vars)}
	_, err := s.auth.ConfirmEmailVerification(ctx, connect.NewRequest(
		&authv1.ConfirmEmailVerificationRequest{Token: r.URL.Query().Get("token")}))
	if err != nil {
		data.Title = loc.Value(i18n.HostedVerifyFailed, vars)
		data.IsError = true
	}
	s.renderPage(w, r, project, data, loc)
}

// handleResetPage shows the new-password form of a reset link.
func (s *Server) handleResetPage(w http.ResponseWriter, r *http.Request) {
	project, ok := s.pageProject(w, r)
	if !ok {
		return
	}
	loc := s.localize(r, project)
	s.renderPage(w, r, project, s.resetFormData(loc, project, r.URL.Query().Get("token")), loc)
}

// resetFormData builds the localized new-password form.
func (s *Server) resetFormData(loc authrpc.Localizer, project store.Project, token string) pageData {
	vars := map[string]string{"app": project.Name}
	return pageData{
		Title:         loc.Value(i18n.HostedResetTitle, vars),
		ShowResetForm: true,
		Token:         token,
		PasswordLabel: loc.Value(i18n.HostedResetPasswordLabel, vars),
		SubmitLabel:   loc.Value(i18n.HostedResetSubmit, vars),
	}
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
	loc := s.localize(r, project)
	vars := map[string]string{"app": project.Name}
	ctx := authrpc.WithProject(r.Context(), project)
	token := r.PostFormValue("token")
	_, err := s.auth.ConfirmPasswordReset(ctx, connect.NewRequest(&authv1.ConfirmPasswordResetRequest{
		Token:       token,
		NewPassword: r.PostFormValue("password"),
	}))
	data := pageData{Title: loc.Value(i18n.HostedResetSuccess, vars)}
	switch {
	case err != nil && authrpc.ErrorReason(err) == authrpc.ReasonWeakPassword:
		// Keep the form up so the user can try a longer password; the inline
		// hint is localized from the catalog alongside the form controls.
		data = s.resetFormData(loc, project, token)
		data.Error = loc.Value(i18n.HostedResetTooShort, vars)
	case err != nil:
		data.Title = loc.Value(i18n.HostedVerifyFailed, vars)
		data.IsError = true
	}
	s.renderPage(w, r, project, data, loc)
}

// handleConfirmEmailPage consumes email-change confirmation and revert
// links.
func (s *Server) handleConfirmEmailPage(w http.ResponseWriter, r *http.Request) {
	project, ok := s.pageProject(w, r)
	if !ok {
		return
	}
	loc := s.localize(r, project)
	vars := map[string]string{"app": project.Name}
	ctx := authrpc.WithProject(r.Context(), project)
	data := pageData{Title: loc.Value(i18n.HostedEmailChangeSuccess, vars)}
	_, err := s.auth.ConfirmEmailChange(ctx, connect.NewRequest(
		&authv1.ConfirmEmailChangeRequest{Token: r.URL.Query().Get("token")}))
	if err != nil {
		data.Title = loc.Value(i18n.HostedVerifyFailed, vars)
		data.IsError = true
	}
	s.renderPage(w, r, project, data, loc)
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

func (s *Server) renderPage(w http.ResponseWriter, r *http.Request, p store.Project, data pageData, loc authrpc.Localizer) {
	s.renderPageStatus(w, r, http.StatusOK, p, data, loc)
}

func (s *Server) renderPageStatus(w http.ResponseWriter, r *http.Request, status int, p store.Project, data pageData, loc authrpc.Localizer) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Referrer-Policy", "no-referrer")
	// Echo the negotiated locale so a caching layer keys on it.
	w.Header().Set(authrpc.LocaleHeader, string(loc.Locale))
	data.Lang = string(loc.Locale)
	data.Dir = loc.Dir()
	// The security middleware minted a CSP nonce for this request; the inline
	// <style> block carries it so the strict policy admits it.
	data.Nonce = httpsec.NonceFromContext(r.Context())
	w.WriteHeader(status)
	if err := pageTemplate.Execute(w, themedData(p, data)); err != nil {
		s.log.Error("render page", "path", r.URL.Path, "error", err.Error())
	}
}
